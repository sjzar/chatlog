package glance

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sjzar/chatlog/internal/errors"
)

// FIXME 按照 region 读取效率较低，512MB 内存读取耗时约 18s(darwin 24)

const (
	MaxWorkers        = 8
	MinChunkSize      = 4 * 1024 * 1024 // 4MB
	ChunkOverlapBytes = 1024            // Greater than all offsets
	ChunkMultiplier   = 2               // Number of chunks = MaxWorkers * ChunkMultiplier
)

type Glance struct {
	PID        uint32
	MemRegions []MemRegion
	pipePath   string
	data       []byte
}

func NewGlance(pid uint32) *Glance {
	return &Glance{
		PID:      pid,
		pipePath: filepath.Join(os.TempDir(), fmt.Sprintf("chatlog_pipe_%d", time.Now().UnixNano())),
	}
}

func (g *Glance) Read() ([]byte, error) {
	if g.data != nil {
		return g.data, nil
	}

	regions, err := GetVmmap(g.PID)
	if err != nil {
		return nil, err
	}
	g.MemRegions = MemRegionsFilter(regions)

	if len(g.MemRegions) == 0 {
		return nil, errors.ErrNoMemoryRegionsFound
	}

	region := g.MemRegions[0]

	// 1. Create pipe file
	if err := exec.Command("mkfifo", g.pipePath).Run(); err != nil {
		return nil, errors.CreatePipeFileFailed(err)
	}
	defer os.Remove(g.pipePath)

	// Start a goroutine to read from the pipe
	dataCh := make(chan []byte, 1)
	errCh := make(chan error, 1)
	go func() {
		// Open pipe for reading
		file, err := os.OpenFile(g.pipePath, os.O_RDONLY, 0600)
		if err != nil {
			errCh <- errors.OpenPipeFileFailed(err)
			return
		}
		defer file.Close()

		// Read all data from pipe
		data, err := io.ReadAll(file)
		if err != nil {
			errCh <- errors.ReadPipeFileFailed(err)
			return
		}
		dataCh <- data
	}()

	// 2 & 3. Execute lldb command to read memory directly with all parameters
	size := region.End - region.Start
	lldbCmd := fmt.Sprintf("lldb -p %d -o \"memory read --binary --force --outfile %s --count %d 0x%x\" -o \"quit\"",
		g.PID, g.pipePath, size, region.Start)

	cmd := exec.Command("bash", "-c", lldbCmd)

	// Set up stdout pipe for monitoring (optional)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, errors.RunCmdFailed(err)
	}

	// Monitor lldb output (optional)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// Uncomment for debugging:
			// fmt.Println(scanner.Text())
		}
	}()

	// Wait for data with timeout
	select {
	case data := <-dataCh:
		g.data = data
	case err := <-errCh:
		return nil, errors.ReadMemoryFailed(err)
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return nil, errors.ErrReadMemoryTimeout
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		// We already have the data, so just log the error
		log.Err(err).Msg("lldb process exited with error")
	}

	return g.data, nil
}

// Read2Chan reads memory regions and sends them to a channel in chunks
// If a region is larger than MinChunkSize, it will be split into multiple chunks
// This function processes regions as they are read (streaming), not waiting for all regions to complete
func (g *Glance) Read2Chan(ctx context.Context, memoryChannel chan<- []byte) error {
	regions, err := GetVmmap(g.PID)
	if err != nil {
		return err
	}
	g.MemRegions = MemRegionsFilter(regions)

	if len(g.MemRegions) == 0 {
		return errors.ErrNoMemoryRegionsFound
	}

	// Read all regions using a single lldb instance and process them as they arrive

	if err := g.streamReadRegions(ctx, g.MemRegions, memoryChannel); err != nil {
		return err
	}

	log.Info().Msgf("read memory completed, region length: %d", len(g.MemRegions))
	return nil
}

// processMemoryRegion processes a single memory region and sends chunks to channel
func (g *Glance) processMemoryRegion(ctx context.Context, memory []byte, regionStart uint64, memoryChannel chan<- []byte) error {
	totalSize := len(memory)

	// If memory is small enough, send it as a single chunk
	if totalSize <= MinChunkSize {
		select {
		case memoryChannel <- memory:
			log.Debug().Msgf("Memory region 0x%x sent as a single chunk for analysis", regionStart)
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}

	// Split large regions into chunks
	chunkCount := MaxWorkers * ChunkMultiplier

	// Calculate chunk size based on fixed chunk count
	chunkSize := totalSize / chunkCount
	if chunkSize < MinChunkSize {
		// Reduce number of chunks if each would be too small
		chunkCount = totalSize / MinChunkSize
		if chunkCount == 0 {
			chunkCount = 1
		}
		chunkSize = totalSize / chunkCount
	}

	// Process memory in chunks from end to beginning
	for i := chunkCount - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Calculate start and end positions for this chunk
			start := i * chunkSize
			end := (i + 1) * chunkSize

			// Ensure the last chunk includes all remaining memory
			if i == chunkCount-1 {
				end = totalSize
			}

			// Add overlap area to catch patterns at chunk boundaries
			if i > 0 {
				start -= ChunkOverlapBytes
				if start < 0 {
					start = 0
				}
			}

			chunk := memory[start:end]

			log.Debug().
				Int("chunk_index", i+1).
				Int("total_chunks", chunkCount).
				Int("chunk_size", len(chunk)).
				Str("start_offset", fmt.Sprintf("%X", start)).
				Str("end_offset", fmt.Sprintf("%X", end)).
				Str("region", fmt.Sprintf("0x%x", regionStart)).
				Msg("Processing memory chunk")

			select {
			case memoryChannel <- chunk:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// streamReadRegions uses a single lldb instance to read all memory regions and processes them as they arrive
func (g *Glance) streamReadRegions(ctx context.Context, regions []MemRegion, memoryChannel chan<- []byte) error {
	if len(regions) == 0 {
		return nil
	}

	// Start lldb process
	cmd := exec.Command("lldb")

	// Set up pipes for stdin/stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.CreatePipeFileFailed(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.CreatePipeFileFailed(err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return errors.RunCmdFailed(err)
	}

	// Create a goroutine to read lldb output
	outputDone := make(chan bool)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			// Uncomment for debugging:
			// log.Debug().Msg(line)
			_ = line
		}
		outputDone <- true
	}()

	// First attach to the process
	if _, err := fmt.Fprintf(stdin, "process attach --pid %d\n", g.PID); err != nil {
		cmd.Process.Kill()
		return errors.RunCmdFailed(err)
	}

	// Wait for the attach to complete
	time.Sleep(2 * time.Second)

	// Channel to signal when we should stop processing
	processingErr := make(chan error, 1)

	// Process each region
	for _, region := range regions {
		select {
		case <-ctx.Done():
			stdin.Close()
			cmd.Process.Kill()
			return ctx.Err()
		case err := <-processingErr:
			stdin.Close()
			cmd.Process.Kill()
			return err
		default:
		}

		readSize := region.End - region.Start

		// Create a unique pipe for each region
		regionPipePath := filepath.Join(os.TempDir(), fmt.Sprintf("chatlog_pipe_%d_%x", time.Now().UnixNano(), region.Start))

		// Create the pipe for this region
		if err := exec.Command("mkfifo", regionPipePath).Run(); err != nil {
			log.Warn().Err(err).Msgf("Failed to create pipe for region 0x%x", region.Start)
			continue
		}

		// WaitGroup for this single region
		var regionWG sync.WaitGroup
		regionWG.Add(1)

		// Start goroutine to read from this region's pipe and process immediately
		go func(pipePath string, regionStart uint64) {
			defer regionWG.Done()
			defer os.Remove(pipePath)

			// Open pipe for reading
			file, err := os.OpenFile(pipePath, os.O_RDONLY, 0600)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed to open pipe for region 0x%x", regionStart)
				return
			}
			defer file.Close()

			// Read all data from pipe
			data, err := io.ReadAll(file)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed to read from pipe for region 0x%x", regionStart)
				return
			}

			// Process and send chunks immediately
			if err := g.processMemoryRegion(ctx, data, regionStart, memoryChannel); err != nil {
				select {
				case processingErr <- err:
				default:
				}
			}
		}(regionPipePath, region.Start)

		// Send memory read command for this region
		memoryReadCmd := fmt.Sprintf("memory read --binary --force --outfile %s --count %d 0x%x\n",
			regionPipePath, readSize, region.Start)

		log.Debug().Msgf("Reading region 0x%x, size: %d bytes", region.Start, readSize)

		if _, err := fmt.Fprint(stdin, memoryReadCmd); err != nil {
			log.Warn().Err(err).Msgf("Failed to send memory read command for region 0x%x", region.Start)
			continue
		}

		// Wait for this region's processing to complete before moving to next region
		regionWG.Wait()
	}

	// Detach and quit
	fmt.Fprint(stdin, "process detach\n")
	time.Sleep(200 * time.Millisecond)
	fmt.Fprint(stdin, "quit\n")

	// Close stdin to signal no more commands
	stdin.Close()

	// Wait for process to complete with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Err(err).Msg("lldb process exited with error")
		}
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		log.Warn().Msg("Timeout waiting for lldb to complete, killed the process")
	}

	// Wait for stdout reader to complete
	select {
	case <-outputDone:
	case <-time.After(10 * time.Second):
		log.Warn().Msg("Timeout waiting for output reader")
	}

	return nil
}
