package darwin

import (
	"bytes"
	"context"
	"encoding/hex"
	"runtime"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/decrypt"
	"github.com/sjzar/chatlog/internal/wechat/key/darwin/glance"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

const (
	MaxWorkers = 8
)

var V4KeyPatterns = []KeyPatternInfo{
	{
		Pattern: []byte{0x20, 0x66, 0x74, 0x73, 0x35, 0x28, 0x25, 0x00},
		Offsets: []int{16, -80, 64},
	},
	{
		Pattern: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Offsets: []int{-32},
	},
}

var V4ImgKeyPatterns = []KeyPatternInfo{
	{
		Pattern: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Offsets: []int{-32},
	},
}

type V4Extractor struct {
	validator         *decrypt.Validator
	dataKeyPatterns   []KeyPatternInfo
	imgKeyPatterns    []KeyPatternInfo
	processedDataKeys sync.Map // Thread-safe map for processed data keys
	processedImgKeys  sync.Map // Thread-safe map for processed image keys
}

func NewV4Extractor() *V4Extractor {
	return &V4Extractor{
		dataKeyPatterns: V4KeyPatterns,
		imgKeyPatterns:  V4ImgKeyPatterns,
	}
}

func (e *V4Extractor) Extract(ctx context.Context, proc *model.Process) (string, string, error) {
	if proc.Status == model.StatusOffline {
		return "", "", errors.ErrWeChatOffline
	}

	// Check if SIP is disabled, as it's required for memory reading on macOS
	if !glance.IsSIPDisabled() {
		return "", "", errors.ErrSIPEnabled
	}

	if e.validator == nil {
		return "", "", errors.ErrValidatorNotSet
	}

	// Create context to control all goroutines
	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create channels for memory data and results
	memoryChannel := make(chan []byte, 200)
	resultChannel := make(chan [2]string, 1)

	// Determine number of worker goroutines
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2
	}
	if workerCount > MaxWorkers {
		workerCount = MaxWorkers
	}
	log.Debug().Msgf("Starting %d workers for V4 key search", workerCount)

	// Start consumer goroutines
	var workerWaitGroup sync.WaitGroup
	workerWaitGroup.Add(workerCount)
	for index := 0; index < workerCount; index++ {
		go func() {
			defer workerWaitGroup.Done()
			e.worker(searchCtx, memoryChannel, resultChannel)
		}()
	}

	// Start producer goroutine
	var producerWaitGroup sync.WaitGroup
	producerWaitGroup.Add(1)
	go func() {
		defer producerWaitGroup.Done()
		defer close(memoryChannel) // Close channel when producer is done
		err := e.findMemory(searchCtx, uint32(proc.PID), memoryChannel)
		if err != nil {
			log.Err(err).Msg("Failed to read memory")
		}
	}()

	// Wait for producer and consumers to complete
	go func() {
		producerWaitGroup.Wait()
		workerWaitGroup.Wait()
		close(resultChannel)
	}()

	// Wait for result
	var finalDataKey, finalImgKey string

	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case result, ok := <-resultChannel:
			if !ok {
				// Channel closed, all workers finished, return whatever keys we found
				if finalDataKey != "" || finalImgKey != "" {
					return finalDataKey, finalImgKey, nil
				}
				return "", "", errors.ErrNoValidKey
			}

			// Update our best found keys
			if result[0] != "" {
				finalDataKey = result[0]
			}
			if result[1] != "" {
				finalImgKey = result[1]
			}

			// If we have both keys, we can return early
			if finalDataKey != "" && finalImgKey != "" {
				cancel() // Cancel remaining work
				return finalDataKey, finalImgKey, nil
			}
		}
	}
}

// findMemory searches for memory regions using Glance
func (e *V4Extractor) findMemory(ctx context.Context, pid uint32, memoryChannel chan<- []byte) error {
	// Initialize a Glance instance to read process memory
	g := glance.NewGlance(pid)

	// Use the Read2Chan method to read and chunk memory
	return g.Read2Chan(ctx, memoryChannel)
}

// worker processes memory regions to find V4 version key
func (e *V4Extractor) worker(ctx context.Context, memoryChannel <-chan []byte, resultChannel chan<- [2]string) {
	// Track found keys
	var dataKey, imgKey string

	for {
		select {
		case <-ctx.Done():
			return
		case memory, ok := <-memoryChannel:
			if !ok {
				// Memory scanning complete, return whatever keys we found
				if dataKey != "" || imgKey != "" {
					select {
					case resultChannel <- [2]string{dataKey, imgKey}:
					default:
					}
				}
				return
			}

			// Search for data key
			if dataKey == "" {
				if key, ok := e.SearchKey(ctx, memory); ok {
					dataKey = key
					log.Debug().Msg("Data key found: " + key)
					// Report immediately when found
					select {
					case resultChannel <- [2]string{dataKey, imgKey}:
					case <-ctx.Done():
						return
					}
				}
			}

			// Search for image key
			if imgKey == "" {
				if key, ok := e.SearchImgKey(ctx, memory); ok {
					imgKey = key
					log.Debug().Msg("Image key found: " + key)
					// Report immediately when found
					select {
					case resultChannel <- [2]string{dataKey, imgKey}:
					case <-ctx.Done():
						return
					}
				}
			}

			// If we have both keys, exit worker
			if dataKey != "" && imgKey != "" {
				log.Debug().Msg("Both keys found, worker exiting")
				return
			}
		}
	}
}

func (e *V4Extractor) SearchKey(ctx context.Context, memory []byte) (string, bool) {
	for _, keyPattern := range e.dataKeyPatterns {
		index := len(memory)
		zeroPattern := bytes.Equal(keyPattern.Pattern, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

		for {
			select {
			case <-ctx.Done():
				return "", false
			default:
			}

			// Find pattern from end to beginning
			index = bytes.LastIndex(memory[:index], keyPattern.Pattern)
			if index == -1 {
				break // No more matches found
			}

			// align to 16 bytes
			if zeroPattern {
				index = bytes.LastIndexFunc(memory[:index], func(r rune) bool {
					return r != 0
				})
				if index == -1 {
					break // No more matches found
				}
				index += 1
			}

			// Try each offset for this pattern
			for _, offset := range keyPattern.Offsets {
				// Check if we have enough space for the key
				keyOffset := index + offset
				if keyOffset < 0 || keyOffset+32 > len(memory) {
					continue
				}

				if bytes.Contains(memory[keyOffset:keyOffset+32], []byte{0x00, 0x00}) {
					continue
				}

				// Extract the key data, which is at the offset position and 32 bytes long
				keyData := memory[keyOffset : keyOffset+32]
				keyHex := hex.EncodeToString(keyData)

				// Skip if we've already processed this key (thread-safe check)
				if _, loaded := e.processedDataKeys.LoadOrStore(keyHex, true); loaded {
					continue
				}

				// Validate key against database header
				if e.validator.Validate(keyData) {
					log.Debug().
						Str("pattern", hex.EncodeToString(keyPattern.Pattern)).
						Int("offset", offset).
						Str("key", keyHex).
						Msg("Data key found")
					return keyHex, true
				}
			}

			index -= 1
			if index < 0 {
				break
			}
		}
	}

	return "", false
}

func (e *V4Extractor) SearchImgKey(ctx context.Context, memory []byte) (string, bool) {

	for _, keyPattern := range e.imgKeyPatterns {
		index := len(memory)

		for {
			select {
			case <-ctx.Done():
				return "", false
			default:
			}

			// Find pattern from end to beginning
			index = bytes.LastIndex(memory[:index], keyPattern.Pattern)
			if index == -1 {
				break // No more matches found
			}

			// align to 16 bytes
			index = bytes.LastIndexFunc(memory[:index], func(r rune) bool {
				return r != 0
			})

			if index == -1 {
				break // No more matches found
			}

			index += 1

			// Try each offset for this pattern
			for _, offset := range keyPattern.Offsets {
				// Check if we have enough space for the key (16 bytes for image key)
				keyOffset := index + offset
				if keyOffset < 0 || keyOffset+16 > len(memory) {
					continue
				}

				if bytes.Contains(memory[keyOffset:keyOffset+16], []byte{0x00, 0x00}) {
					continue
				}

				// Extract the key data, which is at the offset position and 16 bytes long
				keyData := memory[keyOffset : keyOffset+16]
				keyHex := hex.EncodeToString(keyData)

				// Skip if we've already processed this key (thread-safe check)
				if _, loaded := e.processedImgKeys.LoadOrStore(keyHex, true); loaded {
					continue
				}

				// Validate key using image key validator
				if e.validator.ValidateImgKey(keyData) {
					log.Debug().
						Str("pattern", hex.EncodeToString(keyPattern.Pattern)).
						Int("offset", offset).
						Str("key", keyHex).
						Msg("Image key found")
					return keyHex, true
				}
			}

			index -= 1
			if index < 0 {
				break
			}
		}
	}

	return "", false
}

func (e *V4Extractor) SetValidate(validator *decrypt.Validator) {
	e.validator = validator
}

type KeyPatternInfo struct {
	Pattern []byte
	Offsets []int
}
