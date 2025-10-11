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
	MaxWorkersV3 = 8
)

var V3KeyPatterns = []KeyPatternInfo{
	{
		Pattern: []byte{0x72, 0x74, 0x72, 0x65, 0x65, 0x5f, 0x69, 0x33, 0x32},
		Offsets: []int{24},
	},
}

type V3Extractor struct {
	validator   *decrypt.Validator
	keyPatterns []KeyPatternInfo
}

func NewV3Extractor() *V3Extractor {
	return &V3Extractor{
		keyPatterns: V3KeyPatterns,
	}
}

func (e *V3Extractor) Extract(ctx context.Context, proc *model.Process) (string, string, error) {
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
	memoryChannel := make(chan []byte, 100)
	resultChannel := make(chan string, 1)

	// Determine number of worker goroutines
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2
	}
	if workerCount > MaxWorkersV3 {
		workerCount = MaxWorkersV3
	}
	log.Debug().Msgf("Starting %d workers for V3 key search", workerCount)

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
	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case result, ok := <-resultChannel:
		if ok && result != "" {
			return result, "", nil
		}
	}

	return "", "", errors.ErrNoValidKey
}

// findMemory searches for memory regions using Glance
func (e *V3Extractor) findMemory(ctx context.Context, pid uint32, memoryChannel chan<- []byte) error {
	// Initialize a Glance instance to read process memory
	g := glance.NewGlance(pid)

	// Use the Read2Chan method to read and chunk memory
	return g.Read2Chan(ctx, memoryChannel)
}

// worker processes memory regions to find V3 version key
func (e *V3Extractor) worker(ctx context.Context, memoryChannel <-chan []byte, resultChannel chan<- string) {
	for {
		select {
		case <-ctx.Done():
			return
		case memory, ok := <-memoryChannel:
			if !ok {
				return
			}

			if key, ok := e.SearchKey(ctx, memory); ok {
				select {
				case resultChannel <- key:
				default:
				}
			}
		}
	}
}

func (e *V3Extractor) SearchKey(ctx context.Context, memory []byte) (string, bool) {
	for _, keyPattern := range e.keyPatterns {
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

			// Try each offset for this pattern
			for _, offset := range keyPattern.Offsets {
				// Check if we have enough space for the key
				keyOffset := index + offset
				if keyOffset < 0 || keyOffset+32 > len(memory) {
					continue
				}

				// Extract the key data, which is at the offset position and 32 bytes long
				keyData := memory[keyOffset : keyOffset+32]

				// Validate key against database header
				if e.validator.Validate(keyData) {
					log.Debug().
						Str("pattern", hex.EncodeToString(keyPattern.Pattern)).
						Int("offset", offset).
						Str("key", hex.EncodeToString(keyData)).
						Msg("Key found")
					return hex.EncodeToString(keyData), true
				}
			}

			index -= 1
		}
	}

	return "", false
}

func (e *V3Extractor) SetValidate(validator *decrypt.Validator) {
	e.validator = validator
}
