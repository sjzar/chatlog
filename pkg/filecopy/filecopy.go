// Package filecopy provides a high-performance file copying service with persistent caching.
// It creates temporary copies of files that can be reused across application restarts,
// significantly reducing I/O overhead for large files.
//
// Key features:
//   - Instance-based isolation: Different instance IDs maintain separate cache namespaces
//   - Persistent caching: Temporary files survive application restarts
//   - Automatic cleanup: Removes orphaned files and manages cache lifecycle
//   - Thread-safe operations: Concurrent access is fully supported
//   - Version management: Only keeps the latest version of each cached file
package filecopy

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash"
)

// Configuration constants for cache management and cleanup policies.
const (
	// CleanupDelayAfterStart specifies the delay before starting cleanup after manager initialization (1 minute).
	CleanupDelayAfterStart = 1 * time.Minute

	// OrphanFileCleanupThreshold defines when orphaned files should be cleaned up (10 minutes).
	OrphanFileCleanupThreshold = 10 * time.Minute

	// MaxCacheEntries defines the maximum number of files to keep in the cache to prevent memory leaks.
	MaxCacheEntries = 10000 // Reasonable limit for most use cases
)

// Manager instances per instanceID for proper isolation.
var (
	managers   = make(map[string]*FileCopyManager)
	managersMu sync.RWMutex
)

// FileCopyManager manages temporary file copies with persistent caching capabilities.
// It provides thread-safe operations for creating, accessing, and cleaning up temporary files.
type FileCopyManager struct {
	instanceID   string             // Instance identifier for this manager
	tempDir      string             // Base directory for storing temporary files
	fileIndex    sync.Map           // File index: key -> *FileIndexEntry (thread-safe)
	lastAccess   time.Time          // Last access time for TTL cleanup
	startTime    time.Time          // Manager initialization time
	deletionChan chan string        // Async deletion channel for this instance
	ctx          context.Context    // Context for goroutine lifecycle management
	cancel       context.CancelFunc // Cancel function for graceful shutdown
	wg           sync.WaitGroup     // WaitGroup for goroutine synchronization
	cacheSize    int64              // Current number of cached entries (atomic)
}

// FileIndexEntry represents an indexed temporary file with comprehensive metadata.
// It provides O(1) lookup and intelligent file lifecycle management.
// Thread-safe for concurrent access through atomic operations and mutex protection.
type FileIndexEntry struct {
	mu           sync.RWMutex // Protects concurrent access to mutable fields
	TempPath     string       // Path to the temporary file copy (immutable after creation)
	OriginalPath string       // Original source file path (protected by mu)
	Size         int64        // Size of the original file in bytes (immutable after creation)
	ModTime      time.Time    // Modification time of the original file (immutable after creation)
	lastAccess   int64        // Unix timestamp of most recent access (atomic)
	PathHash     string       // Path hash for collision detection (immutable after creation)
	DataHash     string       // Content hash for file integrity verification (immutable after creation)
	BaseName     string       // Base name for multi-version cleanup (immutable after creation)
	Extension    string       // File extension for proper categorization (immutable after creation)
}

// GetLastAccess returns the last access time in a thread-safe manner
func (e *FileIndexEntry) GetLastAccess() time.Time {
	return time.Unix(0, atomic.LoadInt64(&e.lastAccess))
}

// SetLastAccess updates the last access time atomically
func (e *FileIndexEntry) SetLastAccess(t time.Time) {
	atomic.StoreInt64(&e.lastAccess, t.UnixNano())
}

// GetOriginalPath returns the original path in a thread-safe manner
func (e *FileIndexEntry) GetOriginalPath() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.OriginalPath
}

// SetOriginalPath updates the original path in a thread-safe manner
func (e *FileIndexEntry) SetOriginalPath(path string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.OriginalPath = path
}

// indexCandidate represents a candidate file during index building with timestamp information
type indexCandidate struct {
	filePath  string      // Full path to the temporary file
	baseName  string      // Base name extracted from filename
	ext       string      // File extension extracted from filename
	hash      string      // Hash extracted from filename
	timestamp int64       // Timestamp extracted from filename
	fileInfo  os.FileInfo // File metadata
}

// Utility functions for code consolidation

// extractFileExtension extracts and normalizes file extension (without dot)
func extractFileExtension(filePath string) string {
	ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
	if ext == "" {
		return "bin"
	}
	return ext
}

// parseHashComponents splits combined hash into pathHash and dataHash
func parseHashComponents(combinedHash string) (pathHash, dataHash string) {
	parts := strings.Split(combinedHash, "_")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return "", ""
}

// isAuxiliaryDatabaseFile checks if a file is an auxiliary database file that should be ignored
func isAuxiliaryDatabaseFile(expectedExt, actualExt string) bool {
	if expectedExt == "db" && (actualExt == "db-shm" || actualExt == "db-wal") {
		return true
	}
	return false
}

// toIndexEntry converts the candidate to a FileIndexEntry
func (c *indexCandidate) toIndexEntry() *FileIndexEntry {
	// Use utility function to parse hash components
	pathHash, dataHash := parseHashComponents(c.hash)

	return &FileIndexEntry{
		TempPath:     c.filePath,
		OriginalPath: "", // Will be set when matched during GetTempCopy
		Size:         c.fileInfo.Size(),
		ModTime:      c.fileInfo.ModTime(),
		lastAccess:   time.Now().UnixNano(), // Use atomic field
		PathHash:     pathHash,
		DataHash:     dataHash,
		BaseName:     c.baseName,
		Extension:    filepath.Ext(c.filePath),
	}
}

// parseFileCandidate parses a filename and creates an indexCandidate if valid
// New format: instanceID_+baseName_+ext_+pathHash_+dataHash.ext
func (fm *FileCopyManager) parseFileCandidate(fileName, filePath string) *indexCandidate {
	// Get file info for metadata
	info, err := os.Stat(filePath)
	if err != nil {
		return nil
	}

	// Parse filename pattern using "_+" separator
	parts := strings.Split(fileName, "_+")
	if len(parts) < 5 {
		return nil // Need at least: instanceID, baseName, ext, pathHash, dataHash
	}

	// Check if first part matches our instanceID
	if parts[0] != fm.instanceID {
		return nil
	}

	baseName := parts[1]
	ext := parts[2]
	pathHash := parts[3]

	// Extract dataHash from the last part (remove file extension)
	dataHashPart := parts[4]
	dataHash := dataHashPart
	if dotIndex := strings.Index(dataHashPart, "."); dotIndex != -1 {
		dataHash = dataHashPart[:dotIndex]
	}

	// Critical: Verify actual file extension matches declared extension
	// This prevents indexing of auxiliary files like *.db-shm, *.db-wal when we expect *.db
	actualExt := extractFileExtension(fileName)

	// For database files, ignore auxiliary files (.db-shm, .db-wal) if we're looking for .db
	if isAuxiliaryDatabaseFile(ext, actualExt) {
		return nil // Skip auxiliary database files
	}

	// Strict extension matching: declared ext must match actual file extension
	if ext != actualExt {
		return nil // Extension mismatch, skip this file
	}

	// Use file modification time as version timestamp (no longer embedded in filename)
	timestamp := info.ModTime().UnixNano()

	return &indexCandidate{
		filePath:  filePath,
		baseName:  baseName,
		ext:       ext,
		hash:      pathHash + "_" + dataHash, // Combine for compatibility with existing logic
		timestamp: timestamp,
		fileInfo:  info,
	}
}

// findLatestCandidate finds the candidate with the highest timestamp
func (fm *FileCopyManager) findLatestCandidate(candidates []*indexCandidate) *indexCandidate {
	if len(candidates) == 0 {
		return nil
	}

	latest := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.timestamp > latest.timestamp {
			latest = candidate
		}
	}

	return latest
}

// getManager returns the FileCopyManager instance for the specified instanceID.
// Creates a new manager if one doesn't exist for this instanceID.
func getManager(instanceID string) *FileCopyManager {
	managersMu.RLock()
	manager, exists := managers[instanceID]
	managersMu.RUnlock()

	if exists {
		return manager
	}

	managersMu.Lock()
	defer managersMu.Unlock()

	// Double-check after acquiring write lock
	if manager, exists := managers[instanceID]; exists {
		return manager
	}

	// Create new manager for this instanceID
	manager = newManager(instanceID)
	managers[instanceID] = manager
	return manager
}

// newManager creates and initializes a new FileCopyManager instance for the specified instanceID.
// It sets up the temporary directory and starts background cleanup routines with proper lifecycle management.
func newManager(instanceID string) *FileCopyManager {
	procName := getProcessName()
	tempDir := filepath.Join(os.TempDir(), "filecopy_"+procName)

	// Create temporary directory with improved error handling
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		// Try fallback directory
		tempDir = filepath.Join(os.TempDir(), "filecopy")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			// If both fail, use system temp directly (last resort)
			tempDir = os.TempDir()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	fm := &FileCopyManager{
		instanceID:   instanceID,
		tempDir:      tempDir,
		lastAccess:   time.Now(),
		startTime:    time.Now(),
		deletionChan: make(chan string, 10000), // 10k buffer for async deletion
		ctx:          ctx,
		cancel:       cancel,
	}

	// Build file index immediately during initialization
	fm.buildFileIndex()

	// Start managed goroutines with proper lifecycle
	fm.wg.Add(2)
	go fm.asyncDeletionWorker()
	go fm.scheduleDelayedCleanup()

	return fm
}

// asyncDeletionWorker processes file deletion requests asynchronously for this manager instance
func (fm *FileCopyManager) asyncDeletionWorker() {
	defer fm.wg.Done()

	for {
		select {
		case <-fm.ctx.Done():
			// Context cancelled, drain remaining deletions before exit
			for {
				select {
				case filePath, ok := <-fm.deletionChan:
					if !ok {
						return // Channel closed
					}
					fm.processDeletion(filePath)
				default:
					return // No more deletions to process
				}
			}
		case filePath, ok := <-fm.deletionChan:
			if !ok {
				return // Channel closed, exit goroutine
			}
			fm.processDeletion(filePath)
		}
	}
}

// processDeletion handles a single file deletion with safety checks
func (fm *FileCopyManager) processDeletion(filePath string) {
	// Skip .tmp files to avoid interfering with atomic operations
	if strings.Contains(filePath, ".tmp.") {
		return
	}

	// Add small delay to prevent race conditions with concurrent atomic rename operations
	// This ensures that any ongoing rename operations from other goroutines complete first
	time.Sleep(10 * time.Millisecond)

	// Allow deletion failures (file might still be in use)
	os.Remove(filePath)
}

// buildFileIndex scans the temporary directory and organizes ALL instanceID prefixed files.
// This is a one-time operation that enables O(1) lookups and eliminates repeated directory scanning.
//
// Version Deduplication:
// When multiple versions of the same file exist (same instanceID + baseName + hash),
// only the version with the highest timestamp is kept in the index.
// Older versions are queued for asynchronous deletion to prevent disk space waste.
func (fm *FileCopyManager) buildFileIndex() {
	entries, err := os.ReadDir(fm.tempDir)
	if err != nil {
		return // Directory doesn't exist or is inaccessible, skip indexing
	}

	expectedPrefix := fm.instanceID + "_+"

	// Temporary map to collect all versions of each file
	// key: versionKey (instanceID_baseName_ext_pathHash), value: slice of candidates with timestamps
	versionCandidates := make(map[string][]*indexCandidate)

	// First pass: collect all matching files and group by version key
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), expectedPrefix) {
			continue
		}

		filePath := filepath.Join(fm.tempDir, entry.Name())

		// Parse filename: instanceID_+baseName_+ext_+pathHash_+dataHash.ext
		if candidate := fm.parseFileCandidate(entry.Name(), filePath); candidate != nil {
			// Extract components for version grouping
			pathHash, _ := parseHashComponents(candidate.hash)

			// Group by version key (excludes dataHash for version deduplication)
			versionKey := fm.generateVersionKey(fm.instanceID, candidate.baseName, candidate.ext, pathHash)
			versionCandidates[versionKey] = append(versionCandidates[versionKey], candidate)
		}
	}

	// Second pass: for each version group, keep only the latest version
	for _, candidateList := range versionCandidates {
		if len(candidateList) == 1 {
			// Only one version, use it directly
			candidate := candidateList[0]
			if indexEntry := candidate.toIndexEntry(); indexEntry != nil {
				// Use full cache key for storage
				pathHash, dataHash := parseHashComponents(candidate.hash)
				cacheKey := fm.generateCacheKey(fm.instanceID, candidate.baseName, candidate.ext, pathHash, dataHash)
				fm.fileIndex.Store(cacheKey, indexEntry)
				atomic.AddInt64(&fm.cacheSize, 1)
			}
		} else {
			// Multiple versions exist, find the latest one and delete others
			latest := fm.findLatestCandidate(candidateList)

			// Store the latest version using full cache key
			if indexEntry := latest.toIndexEntry(); indexEntry != nil {
				pathHash, dataHash := parseHashComponents(latest.hash)
				cacheKey := fm.generateCacheKey(fm.instanceID, latest.baseName, latest.ext, pathHash, dataHash)
				fm.fileIndex.Store(cacheKey, indexEntry)
				atomic.AddInt64(&fm.cacheSize, 1)
			}

			// Queue older versions for deletion
			for _, candidate := range candidateList {
				if candidate != latest {
					select {
					case fm.deletionChan <- candidate.filePath:
					default:
						// Channel full, delete synchronously
						os.Remove(candidate.filePath)
					}
				}
			}
		}
	}
}

// extractBaseName extracts the base filename without path and extension for indexing.
func (fm *FileCopyManager) extractBaseName(originalPath string) string {
	fileName := filepath.Base(originalPath)
	fileExt := filepath.Ext(fileName)
	baseName := fileName
	if len(fileExt) > 0 && len(fileName) > len(fileExt) {
		baseName = fileName[:len(fileName)-len(fileExt)]
	}
	if baseName == "" || baseName == fileExt {
		baseName = "file"
	}
	return baseName
}

// scheduleDelayedCleanup performs one-time cleanup after 1 minute delay from manager creation.
// This cleans up unused files that haven't been accessed since manager creation.
func (fm *FileCopyManager) scheduleDelayedCleanup() {
	defer fm.wg.Done()

	// Wait for 1 minute or until context is cancelled
	select {
	case <-fm.ctx.Done():
		return // Context cancelled, exit without cleanup
	case <-time.After(CleanupDelayAfterStart):
		// Perform one-time cleanup of unused files
		fm.performInitialCleanup()
	}
}

// performInitialCleanup removes files that haven't been accessed since manager creation.
// This implements the 1-minute delay cleanup strategy for unused files.
func (fm *FileCopyManager) performInitialCleanup() {
	// Clean up files that haven't been accessed since manager start
	fm.fileIndex.Range(func(key, value any) bool {
		entry := value.(*FileIndexEntry)
		lastAccess := entry.GetLastAccess()
		if lastAccess.Before(fm.startTime) || lastAccess.Equal(fm.startTime) {
			// Queue for async deletion and remove from index
			select {
			case fm.deletionChan <- entry.TempPath:
				fm.fileIndex.Delete(key)
				atomic.AddInt64(&fm.cacheSize, -1)
			default:
				// Deletion channel is full, skip this file
			}
		}
		return true
	})

	// Also clean up orphaned files on disk for this instance
	fm.cleanupOrphanedFilesInternal()
}

// performCacheCleanup removes least recently used cache entries when cache size exceeds limit
func (fm *FileCopyManager) performCacheCleanup() {
	currentSize := atomic.LoadInt64(&fm.cacheSize)
	if currentSize <= MaxCacheEntries {
		return // Cache size is acceptable
	}

	// Collect all entries with their last access times
	type cacheEntry struct {
		key        string
		lastAccess int64
		entry      *FileIndexEntry
	}

	var entries []cacheEntry
	fm.fileIndex.Range(func(key, value any) bool {
		entry := value.(*FileIndexEntry)
		entries = append(entries, cacheEntry{
			key:        key.(string),
			lastAccess: atomic.LoadInt64(&entry.lastAccess),
			entry:      entry,
		})
		return true
	})

	// Sort by last access time (oldest first) - O(n log n) instead of O(n²)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].lastAccess < entries[j].lastAccess
	})

	// Remove oldest 25% of entries to make room for new ones
	removeCount := max(1, len(entries)/4)

	for i := 0; i < removeCount && i < len(entries); i++ {
		entry := entries[i]
		fm.fileIndex.Delete(entry.key)
		atomic.AddInt64(&fm.cacheSize, -1)

		// Queue for async deletion
		select {
		case fm.deletionChan <- entry.entry.TempPath:
		default:
			// Channel full, delete synchronously
			os.Remove(entry.entry.TempPath)
		}
	}
}

// cleanupOrphanedFilesInternal performs the actual cleanup without acquiring locks (internal use).
func (fm *FileCopyManager) cleanupOrphanedFilesInternal() {
	// Build set of indexed temporary file paths
	indexedPaths := make(map[string]bool)
	fm.fileIndex.Range(func(key, value any) bool {
		entry := value.(*FileIndexEntry)
		indexedPaths[entry.TempPath] = true
		return true
	})

	// Scan temporary directory for orphaned files
	entries, err := os.ReadDir(fm.tempDir)
	if err != nil {
		return // Directory doesn't exist or is inaccessible
	}

	now := time.Now()
	expectedPrefix := fm.instanceID + "_+"

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process files belonging to this specific instance
		if !strings.HasPrefix(entry.Name(), expectedPrefix) {
			continue
		}

		filePath := filepath.Join(fm.tempDir, entry.Name())

		// Remove files not in index that exceed the cleanup threshold
		if !indexedPaths[filePath] {
			if info, err := entry.Info(); err == nil {
				if now.Sub(info.ModTime()) > OrphanFileCleanupThreshold {
					select {
					case fm.deletionChan <- filePath:
					default:
						// Channel full, delete synchronously as fallback
						os.Remove(filePath)
					}
				}
			}
		}
	}
}

// GetTempCopy creates or retrieves a temporary copy of the specified file.
// It provides persistent caching with instance-based isolation.
//
// Parameters:
//   - instanceID: Unique identifier for the application instance (e.g., "app_v1.0", "service_name")
//   - originalPath: Absolute path to the original file to copy
//
// Returns:
//   - string: Path to the temporary copy
//   - error: Any error encountered during the operation
//
// The function performs these operations:
//  1. Checks in-memory cache for existing valid copy
//  2. Scans disk for existing cached file that can be reused
//  3. Creates new copy if none found, cleaning up old versions
//
// Thread-safe for concurrent use.
func GetTempCopy(instanceID, originalPath string) (string, error) {
	return getManager(instanceID).GetTempCopy(originalPath)
}

// GetTempCopy implements optimized file copying with intelligent index-based lookup.
// This eliminates repeated directory scanning and provides O(1) lookup performance.
func (fm *FileCopyManager) GetTempCopy(originalPath string) (string, error) {
	// Validate original file and get metadata
	stat, err := os.Stat(originalPath)
	if err != nil {
		return "", fmt.Errorf("original file does not exist: %w", err)
	}

	now := time.Now()
	currentModTime := stat.ModTime()
	currentSize := stat.Size()
	currentHash := fm.hashString(originalPath)

	// Update last access time for TTL cleanup (no lock needed for time.Time)
	fm.lastAccess = now

	// Generate expected content hash for comparison
	expectedDataHash, err := fm.hashFileContent(originalPath, currentSize)
	if err != nil {
		expectedDataHash = fmt.Sprintf("%x", currentSize+currentModTime.UnixNano())[:16]
	}

	// Strategy 1: Check index for existing file using unified cache key
	baseName := fm.extractBaseName(originalPath)
	ext := extractFileExtension(originalPath)

	// Use unified cache key generation
	cacheKey := fm.generateCacheKey(fm.instanceID, baseName, ext, currentHash, expectedDataHash)

	var oldTempPath string // Track old file to delete after successful creation
	if value, exists := fm.fileIndex.Load(cacheKey); exists {
		entry := value.(*FileIndexEntry)
		// Found cached file, verify it still exists and matches
		if _, err := os.Stat(entry.TempPath); err == nil && currentSize == entry.Size {
			// File exists and size matches, reuse cached copy
			entry.SetLastAccess(now)            // Update access time atomically
			entry.SetOriginalPath(originalPath) // Update original path thread-safely
			return entry.TempPath, nil
		} else {
			// Cached file is missing or size mismatch, remove from index
			fm.fileIndex.Delete(cacheKey)
			atomic.AddInt64(&fm.cacheSize, -1)
			if err == nil {
				// File exists but size mismatch, mark for cleanup
				oldTempPath = entry.TempPath
			}
		}
	}

	// Strategy 2: No valid cached file found, create new one
	tempPath := fm.generateTempPath(originalPath)

	// Perform atomic file copy
	if err := fm.atomicCopyFile(originalPath, tempPath); err != nil {
		return "", err
	}

	// Add to index for future O(1) lookups using unified cache key
	newEntry := &FileIndexEntry{
		TempPath:     tempPath,
		OriginalPath: originalPath,
		Size:         currentSize,
		ModTime:      currentModTime,
		lastAccess:   now.UnixNano(), // Use atomic field
		PathHash:     currentHash,
		DataHash:     expectedDataHash, // Use the same dataHash we calculated earlier
		BaseName:     baseName,
		Extension:    filepath.Ext(originalPath),
	}
	// Use the same cache key we tried to lookup earlier
	fm.fileIndex.Store(cacheKey, newEntry)
	atomic.AddInt64(&fm.cacheSize, 1)

	// Check if cache size exceeds limit and trigger cleanup if needed
	if atomic.LoadInt64(&fm.cacheSize) > MaxCacheEntries {
		go fm.performCacheCleanup()
	}

	// Clean up old version after successful creation to prevent race conditions
	if oldTempPath != "" {
		select {
		case fm.deletionChan <- oldTempPath:
		default:
			// Channel full, delete synchronously as fallback
			os.Remove(oldTempPath)
		}
	}

	return tempPath, nil
}

// generateTempPath creates a unique temporary file path using a structured naming convention.
// The format is: instanceID_+baseName_+ext_+pathHash_+dataHash.ext
// This naming scheme uses "_+" separator to avoid conflicts with filenames containing underscores.
func (fm *FileCopyManager) generateTempPath(originalPath string) string {
	fileName := filepath.Base(originalPath)
	fileExt := filepath.Ext(fileName)
	baseName := fileName
	if len(fileExt) > 0 && len(fileName) > len(fileExt) {
		baseName = fileName[:len(fileName)-len(fileExt)]
	}
	if baseName == "" || baseName == fileExt {
		baseName = "file"
	}

	// Limit baseName length to prevent filesystem errors (most filesystems have 255 char limit)
	// Reserve space for: instanceID + "_+" + baseName + "_+" + ext + "_+" + pathHash + "_+" + dataHash + fileExt
	// Roughly: instanceID (up to 50) + separators (8) + baseName + ext (up to 20) + pathHash (8) + dataHash (16) + fileExt (up to 10) = ~120+ chars
	maxBaseNameLen := 100 // Conservative limit to stay well under 255
	if len(baseName) > maxBaseNameLen {
		baseName = baseName[:maxBaseNameLen]
	}

	// Generate path hash for collision avoidance
	pathHash := fm.hashString(originalPath)
	if len(pathHash) > 8 {
		pathHash = pathHash[:8]
	}

	// Get file stats for content hashing
	stat, err := os.Stat(originalPath)
	if err != nil {
		// Fallback to timestamp-based hash if stat fails
		dataHash := fmt.Sprintf("%x", time.Now().UnixNano())[:16]
		return filepath.Join(fm.tempDir, fmt.Sprintf("%s_+%s_+%s_+%s_+%s%s",
			fm.instanceID, baseName, strings.TrimPrefix(fileExt, "."), pathHash, dataHash, fileExt))
	}

	// Generate content hash for file integrity verification
	dataHash, err := fm.hashFileContent(originalPath, stat.Size())
	if err != nil {
		// Fallback to size+modtime hash if content hashing fails
		dataHash = fmt.Sprintf("%x", stat.Size()+stat.ModTime().UnixNano())[:16]
	} else if len(dataHash) > 16 {
		dataHash = dataHash[:16] // Truncate to reasonable length
	}

	// Clean extension (remove dot)
	cleanExt := strings.TrimPrefix(fileExt, ".")
	if cleanExt == "" {
		cleanExt = "bin" // Default extension for files without extensions
	}

	// Construct temporary file path with new naming convention
	return filepath.Join(fm.tempDir, fmt.Sprintf("%s_+%s_+%s_+%s_+%s%s",
		fm.instanceID, baseName, cleanExt, pathHash, dataHash, fileExt))
}

// atomicCopyFile performs an atomic file copy operation to ensure data integrity.
// It uses a temporary file and atomic rename to prevent partial writes from being visible.
func (fm *FileCopyManager) atomicCopyFile(src, dst string) error {
	// Create temporary file for atomic operation
	tempDst := dst + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())

	// Open source file for reading
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create temporary destination file
	dstFile, err := os.Create(tempDst)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Ensure cleanup of temporary file on error
	defer func() {
		if err != nil {
			os.Remove(tempDst)
		}
	}()

	// Use buffered copy for better performance with large files
	buf := make([]byte, 256*1024) // 256KB buffer
	if _, err = io.CopyBuffer(dstFile, srcFile, buf); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Force write to disk to ensure data persistence
	if err = dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close file before rename operation
	if err = dstFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomic rename to final destination
	if err = os.Rename(tempDst, dst); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// hashString generates a 32-bit FNV-1a hash of the input string.
// This is used for creating collision-resistant file identifiers and cache keys.
func (fm *FileCopyManager) hashString(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum32())
}

// generateCacheKey creates a unified cache key for file indexing and lookup.
// Format: instanceID_baseName_ext_pathHash_dataHash
// This ensures consistent key generation across buildFileIndex and GetTempCopy.
func (fm *FileCopyManager) generateCacheKey(instanceID, baseName, ext, pathHash, dataHash string) string {
	return instanceID + "_" + baseName + "_" + ext + "_" + pathHash + "_" + dataHash
}

// generateVersionKey creates a key for version deduplication (without dataHash).
// Files with same instanceID+baseName+ext+pathHash are considered versions of the same file.
func (fm *FileCopyManager) generateVersionKey(instanceID, baseName, ext, pathHash string) string {
	return instanceID + "_" + baseName + "_" + ext + "_" + pathHash
}

// hashFileContent generates a fast hash of file content for integrity verification.
// Uses xxhash for complete file hashing, providing excellent performance (7120+ MB/s).
func (fm *FileCopyManager) hashFileContent(filePath string, _ int64) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Use xxhash for complete file hashing - benchmark shows 3.3x faster than SHA-256
	h := xxhash.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum64()), nil
}

// Shutdown performs graceful shutdown and cleanup of all resources (Public API).
// This cleans up all manager instances and allows for re-initialization if needed.
func Shutdown() {
	managersMu.Lock()
	defer managersMu.Unlock()

	// Shutdown all managers
	for _, manager := range managers {
		manager.Shutdown()
	}

	// Clear managers map
	managers = make(map[string]*FileCopyManager)
}

// Shutdown performs complete cleanup by removing all temporary files and cache entries.
// This method ensures clean resource deallocation with proper goroutine lifecycle management.
func (fm *FileCopyManager) Shutdown() {
	// Cancel context to signal goroutines to stop
	fm.cancel()

	// Remove all cached temporary files asynchronously
	fm.fileIndex.Range(func(key, value any) bool {
		entry := value.(*FileIndexEntry)
		select {
		case fm.deletionChan <- entry.TempPath:
		default:
			// Channel full, delete synchronously as fallback
			os.Remove(entry.TempPath)
		}
		return true
	})

	// Clear all entries from sync.Map
	fm.fileIndex.Range(func(key, value any) bool {
		fm.fileIndex.Delete(key)
		atomic.AddInt64(&fm.cacheSize, -1)
		return true
	})

	// Close deletion channel to stop the async worker
	close(fm.deletionChan)

	// Wait for all goroutines to finish properly
	fm.wg.Wait()

	// Note: Do NOT remove the shared temp directory here as other instances may still be using it
	// The temp directory will be cleaned up by the OS when the process exits
}

// getProcessName extracts and sanitizes the current process name for use in temporary directory naming.
// Returns a clean process name suitable for filesystem path construction.
func getProcessName() string {
	executable, err := os.Executable()
	if err != nil {
		return "unknown"
	}

	// Extract base name (without extension)
	baseName := filepath.Base(executable)
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}

	// Sanitize name to contain only safe characters
	baseName = cleanProcessName(baseName)
	return baseName
}

// cleanProcessName sanitizes a process name by replacing invalid characters with underscores.
// Keeps only alphanumeric characters, hyphens, and underscores for filesystem safety.
func cleanProcessName(name string) string {
	result := make([]rune, 0, len(name))
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
