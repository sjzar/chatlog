package windows

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/internal/wechat/model"
	"github.com/sjzar/chatlog/pkg/util"
)

const (
	V3ModuleName = "WeChatWin.dll"
	MaxWorkers   = 16
)

func (e *V3Extractor) Extract(ctx context.Context, proc *model.Process) (string, string, error) {
	if proc.Status == model.StatusOffline {
		return "", "", errors.ErrWeChatOffline
	}

	// Open WeChat process
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, proc.PID)
	if err != nil {
		return "", "", errors.OpenProcessFailed(err)
	}
	defer windows.CloseHandle(handle)

	// Check process architecture
	is64Bit, err := util.Is64Bit(handle)
	if err != nil {
		return "", "", err
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
	if workerCount > MaxWorkers {
		workerCount = MaxWorkers
	}
	log.Debug().Msgf("Starting %d workers for V3 key search", workerCount)

	// Start consumer goroutines
	var workerWaitGroup sync.WaitGroup
	workerWaitGroup.Add(workerCount)
	for index := 0; index < workerCount; index++ {
		go func() {
			defer workerWaitGroup.Done()
			e.worker(searchCtx, handle, is64Bit, memoryChannel, resultChannel)
		}()
	}

	// Start producer goroutine
	var producerWaitGroup sync.WaitGroup
	producerWaitGroup.Add(1)
	go func() {
		defer producerWaitGroup.Done()
		defer close(memoryChannel) // Close channel when producer is done
		err := e.findMemory(searchCtx, handle, proc.PID, memoryChannel)
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

// findMemoryV3 searches for writable memory regions in WeChatWin.dll for V3 version
func (e *V3Extractor) findMemory(ctx context.Context, handle windows.Handle, pid uint32, memoryChannel chan<- []byte) error {
	// Find WeChatWin.dll module
	module, isFound := FindModule(pid, V3ModuleName)
	if !isFound {
		return errors.ErrWeChatDLLNotFound
	}
	log.Debug().Msg("Found WeChatWin.dll module at base address: 0x" + fmt.Sprintf("%X", module.ModBaseAddr))

	// Read writable memory regions
	baseAddr := uintptr(module.ModBaseAddr)
	endAddr := baseAddr + uintptr(module.ModBaseSize)
	currentAddr := baseAddr

	for currentAddr < endAddr {
		var mbi windows.MemoryBasicInformation
		err := windows.VirtualQueryEx(handle, currentAddr, &mbi, unsafe.Sizeof(mbi))
		if err != nil {
			break
		}

		// Skip small memory regions
		if mbi.RegionSize < 100*1024 {
			currentAddr += uintptr(mbi.RegionSize)
			continue
		}

		// Check if memory region is writable
		isWritable := (mbi.Protect & (windows.PAGE_READWRITE | windows.PAGE_WRITECOPY | windows.PAGE_EXECUTE_READWRITE | windows.PAGE_EXECUTE_WRITECOPY)) > 0
		if isWritable && uint32(mbi.State) == windows.MEM_COMMIT {
			// Calculate region size, ensure it doesn't exceed DLL bounds
			regionSize := uintptr(mbi.RegionSize)
			if currentAddr+regionSize > endAddr {
				regionSize = endAddr - currentAddr
			}

			// Read writable memory region
			memory := make([]byte, regionSize)
			if err = windows.ReadProcessMemory(handle, currentAddr, &memory[0], regionSize, nil); err == nil {
				select {
				case memoryChannel <- memory:
					log.Debug().Msgf("Memory region: 0x%X - 0x%X, size: %d bytes", currentAddr, currentAddr+regionSize, regionSize)
				case <-ctx.Done():
					return nil
				}
			}
		}

		// Move to next memory region
		currentAddr = uintptr(mbi.BaseAddress) + uintptr(mbi.RegionSize)
	}

	return nil
}

// workerV3 processes memory regions to find V3 version key
func (e *V3Extractor) worker(ctx context.Context, handle windows.Handle, is64Bit bool, memoryChannel <-chan []byte, resultChannel chan<- string) {
	// Define search pattern
	keyPattern := []byte{0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ptrSize := 8
	littleEndianFunc := binary.LittleEndian.Uint64

	// Adjust for 32-bit process
	if !is64Bit {
		keyPattern = keyPattern[:4]
		ptrSize = 4
		littleEndianFunc = func(b []byte) uint64 { return uint64(binary.LittleEndian.Uint32(b)) }
	}

	for {
		select {
		case <-ctx.Done():
			return
		case memory, ok := <-memoryChannel:
			if !ok {
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
					if key := e.validateKey(handle, ptrValue); key != "" {
						select {
						case resultChannel <- key:
							log.Debug().Msg("Valid key found: " + key)
							return
						default:
						}
					}
				}
				index -= 1 // Continue searching from previous position
			}
		}
	}
}

// validateKey validates a single key candidate
func (e *V3Extractor) validateKey(handle windows.Handle, addr uint64) string {
	keyData := make([]byte, 0x20) // 32-byte key
	if err := windows.ReadProcessMemory(handle, uintptr(addr), &keyData[0], uintptr(len(keyData)), nil); err != nil {
		return ""
	}

	// Validate key against database header
	if e.validator.Validate(keyData) {
		return hex.EncodeToString(keyData)
	}

	return ""
}

// FindModule searches for a specified module in the process
func FindModule(pid uint32, name string) (module windows.ModuleEntry32, isFound bool) {
	// Create module snapshot
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE|windows.TH32CS_SNAPMODULE32, pid)
	if err != nil {
		log.Debug().Msgf("Failed to create module snapshot for PID %d: %v", pid, err)
		return module, false
	}
	defer windows.CloseHandle(snapshot)

	// Initialize module entry structure
	module.Size = uint32(windows.SizeofModuleEntry32)

	// Get the first module
	if err := windows.Module32First(snapshot, &module); err != nil {
		log.Debug().Msgf("Module32First failed for PID %d: %v", pid, err)
		return module, false
	}

	// Iterate through all modules to find WeChatWin.dll
	for ; err == nil; err = windows.Module32Next(snapshot, &module) {
		if windows.UTF16ToString(module.Module[:]) == name {
			return module, true
		}
	}
	return module, false
}
