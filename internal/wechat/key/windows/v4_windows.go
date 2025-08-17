package windows

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"runtime"
	"sync"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/model"
)

const (
	MEM_PRIVATE = 0x20000
)

func (e *V4Extractor) Extract(ctx context.Context, proc *model.Process) (string, string, error) {
	if proc.Status == model.StatusOffline {
		return "", "", errors.ErrWeChatOffline
	}

	// Open process handle
	handle, err := windows.OpenProcess(windows.PROCESS_VM_READ|windows.PROCESS_QUERY_INFORMATION, false, proc.PID)
	if err != nil {
		return "", "", errors.OpenProcessFailed(err)
	}
	defer windows.CloseHandle(handle)

	// Create context to control all goroutines
	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create channels for memory data and results
	memoryChannel := make(chan []byte, 100)
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
			e.worker(searchCtx, handle, memoryChannel, resultChannel)
		}()
	}

	// Start producer goroutine
	var producerWaitGroup sync.WaitGroup
	producerWaitGroup.Add(1)
	go func() {
		defer producerWaitGroup.Done()
		defer close(memoryChannel) // Close channel when producer is done
		err := e.findMemory(searchCtx, handle, memoryChannel)
		if err != nil {
			log.Err(err).Msg("Failed to find memory regions")
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

// findMemoryV4 searches for writable memory regions for V4 version
func (e *V4Extractor) findMemory(ctx context.Context, handle windows.Handle, memoryChannel chan<- []byte) error {
	// Define search range
	minAddr := uintptr(0x10000)    // Process space usually starts from 0x10000
	maxAddr := uintptr(0x7FFFFFFF) // 32-bit process space limit

	if runtime.GOARCH == "amd64" {
		maxAddr = uintptr(0x7FFFFFFFFFFF) // 64-bit process space limit
	}
	log.Debug().Msgf("Scanning memory regions from 0x%X to 0x%X", minAddr, maxAddr)

	currentAddr := minAddr

	for currentAddr < maxAddr {
		var memInfo windows.MemoryBasicInformation
		err := windows.VirtualQueryEx(handle, currentAddr, &memInfo, unsafe.Sizeof(memInfo))
		if err != nil {
			break
		}

		// Skip small memory regions
		if memInfo.RegionSize < 1024*1024 {
			currentAddr += uintptr(memInfo.RegionSize)
			continue
		}

		// Check if memory region is readable and private
		if memInfo.State == windows.MEM_COMMIT && (memInfo.Protect&windows.PAGE_READWRITE) != 0 && memInfo.Type == MEM_PRIVATE {
			// Calculate region size, ensure it doesn't exceed limit
			regionSize := uintptr(memInfo.RegionSize)
			if currentAddr+regionSize > maxAddr {
				regionSize = maxAddr - currentAddr
			}

			// Read memory region
			memory := make([]byte, regionSize)
			if err = windows.ReadProcessMemory(handle, currentAddr, &memory[0], regionSize, nil); err == nil {
				select {
				case memoryChannel <- memory:
					log.Debug().Msgf("Memory region for analysis: 0x%X - 0x%X, size: %d bytes", currentAddr, currentAddr+regionSize, regionSize)
				case <-ctx.Done():
					return nil
				}
			}
		}

		// Move to next memory region
		currentAddr = uintptr(memInfo.BaseAddress) + uintptr(memInfo.RegionSize)
	}

	return nil
}

// workerV4 processes memory regions to find V4 version key
func (e *V4Extractor) worker(ctx context.Context, handle windows.Handle, memoryChannel <-chan []byte, resultChannel chan<- [2]string) {
	// Define search pattern for V4
	keyPattern := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x2F, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	ptrSize := 8
	littleEndianFunc := binary.LittleEndian.Uint64

	// Track found keys
	var dataKey, imgKey string
	keysFound := make(map[uint64]bool) // Track processed addresses to avoid duplicates

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

			index := len(memory)
			for {
				select {
				case <-ctx.Done():
					return // Exit if context cancelled
				default:
				}

				// Find pattern from end to beginning
				index = bytes.LastIndex(memory[:index], keyPattern)
				if index == -1 || index-ptrSize < 0 {
					break
				}

				// Extract and validate pointer value
				ptrValue := littleEndianFunc(memory[index-ptrSize : index])
				if ptrValue > 0x10000 && ptrValue < 0x7FFFFFFFFFFF {
					// Skip if we've already processed this address
					if keysFound[ptrValue] {
						index -= 1
						continue
					}
					keysFound[ptrValue] = true

					// Validate key and determine type
					if key, isImgKey := e.validateKey(handle, ptrValue); key != "" {
						if isImgKey {
							if imgKey == "" {
								imgKey = key
								log.Debug().Msg("Image key found: " + key)
								// Report immediately when found
								select {
								case resultChannel <- [2]string{dataKey, imgKey}:
								case <-ctx.Done():
									return
								}
							}
						} else {
							if dataKey == "" {
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

						// If we have both keys, exit worker
						if dataKey != "" && imgKey != "" {
							log.Debug().Msg("Both keys found, worker exiting")
							return
						}
					}
				}
				index -= 1 // Continue searching from previous position
			}
		}
	}
}

// validateKey validates a single key candidate and returns the key and whether it's an image key
func (e *V4Extractor) validateKey(handle windows.Handle, addr uint64) (string, bool) {
	keyData := make([]byte, 0x20) // 32-byte key
	if err := windows.ReadProcessMemory(handle, uintptr(addr), &keyData[0], uintptr(len(keyData)), nil); err != nil {
		return "", false
	}

	// First check if it's a valid database key
	if e.validator.Validate(keyData) {
		return hex.EncodeToString(keyData), false // Data key
	}

	// Then check if it's a valid image key
	if e.validator.ValidateImgKey(keyData) {
		return hex.EncodeToString(keyData[:16]), true // Image key
	}

	return "", false
}
