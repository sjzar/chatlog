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
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash"
	"github.com/rs/zerolog/log"
)

// Configuration constants for cache management and behavior tuning.
const (
	// CleanupInterval defines how often to run unified cleanup (1 minute).
	// First run is delayed by CleanupInterval after manager initialization.
	CleanupInterval = 1 * time.Minute

	// OrphanFileCleanupThreshold defines when orphaned files should be cleaned up (10 minutes).
	OrphanFileCleanupThreshold = 10 * time.Minute

	// MaxCacheEntries defines the maximum number of files to keep in the cache to prevent memory leaks.
	MaxCacheEntries = 10000 // Reasonable limit for most use cases

	// RecentFileProtectionWindow prevents deletion of recently modified/accessed files.
	RecentFileProtectionWindow = 2 * CleanupInterval

	// DedupSkipWindow skips version deduplication for very recent files during periodic cleanup.
	DedupSkipWindow = CleanupInterval

	// PathHashHexLen limits path-hash length in filenames (increase to lower collision risk).
	PathHashHexLen = 12

	// DataHashHexLen limits data-hash length in filenames.
	DataHashHexLen = 16

	// MaxBaseNameLen limits base filename length in temp file naming.
	MaxBaseNameLen = 100
)

// Version detection policy for generating data hash in file naming and cache keys.
type VersionDetectionMode int

const (
	// VersionDetectContentHash computes data hash from entire file content (strong consistency).
	VersionDetectContentHash VersionDetectionMode = iota
	// VersionDetectSizeModTime computes data hash from size+modtime only (faster, weaker consistency).
	VersionDetectSizeModTime
)

// VersionDetection controls how data hash is computed. Adjust as needed.
const VersionDetection = VersionDetectContentHash

// Manager instances per instanceID for proper isolation.
var (
	managers   = make(map[string]*FileCopyManager)
	managersMu sync.RWMutex
)

// FileCopyManager manages temporary file copies with persistent caching capabilities.
// It provides thread-safe operations for creating, accessing, and cleaning up temporary files.
type FileCopyManager struct {
	instanceID string             // Instance identifier for this manager
	tempDir    string             // Base directory for storing temporary files
	fileIndex  sync.Map           // File index: key -> *FileIndexEntry (thread-safe)
	lastAccess time.Time          // Last access time for TTL cleanup
	startTime  time.Time          // Manager initialization time
	ctx        context.Context    // Context for goroutine lifecycle management
	cancel     context.CancelFunc // Cancel function for graceful shutdown
	wg         sync.WaitGroup     // WaitGroup for goroutine synchronization
	cacheSize  int64              // Current number of cached entries (atomic)
	pathLocks  sync.Map           // Per-original-path locks to prevent duplicate concurrent copies
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
	Extension    string       // File extension (normalized, without leading dot) (immutable after creation)
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

// declaredExtFromName extracts the declared extension (without dot) from a temp filename
// using the naming convention: instanceID_+baseName_+ext_+pathHash_+dataHash.ext
// Returns ext and true on success; otherwise false.
func declaredExtFromName(fileName string) (string, bool) {
	parts := strings.Split(fileName, "_+")
	if len(parts) < 5 {
		return "", false
	}
	// ext is the third from the end
	return parts[len(parts)-3], true
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
		Extension:    c.ext, // normalized without dot
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

	// Parse filename pattern using "_+" separator, but allow baseName to contain the token.
	// Format: instanceID _+ baseName (can contain _+) _+ ext _+ pathHash _+ dataHash.ext
	parts := strings.Split(fileName, "_+")
	if len(parts) < 5 {
		return nil // Need at least: instanceID, baseName, ext, pathHash, dataHash
	}
	if parts[0] != fm.instanceID {
		return nil
	}
	// Extract from right to left to tolerate _+ in baseName
	// ... [0]=instanceID, [1:len-3]=baseName parts, [len-3]=ext, [len-2]=pathHash, [len-1]=dataHash.ext
	ext := parts[len(parts)-3]
	pathHash := parts[len(parts)-2]
	dataHashPart := parts[len(parts)-1]
	baseName := strings.Join(parts[1:len(parts)-3], "_+")

	dataHash := dataHashPart
	if dotIndex := strings.Index(dataHashPart, "."); dotIndex != -1 {
		dataHash = dataHashPart[:dotIndex]
	}

	// Critical: Verify actual file extension matches declared extension
	// This prevents indexing of auxiliary files like *.db-shm, *.db-wal when we expect *.db
	actualExt := extractFileExtension(fileName)

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
		instanceID: instanceID,
		tempDir:    tempDir,
		lastAccess: time.Now(),
		startTime:  time.Now(),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Build initial file index during initialization
	fm.rebuildIndexAndCleanup()

	// Start managed goroutines with proper lifecycle
	fm.wg.Add(1)
	go fm.periodicCleanupWorker()

	return fm
}

// processDeletionInline deletes a file inline with safety delay and debug logging on failure.
func (fm *FileCopyManager) processDeletionInline(filePath string) {
	// Skip .tmp files to avoid interfering with atomic operations
	if strings.Contains(filePath, ".tmp.") {
		return
	}
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Debug().Err(err).Str("path", filePath).Msg("filecopy: delete failed")
	}
}

// periodicCleanupWorker runs unified cleanup periodically.
// First run is delayed by CleanupInterval, then runs every CleanupInterval.
func (fm *FileCopyManager) periodicCleanupWorker() {
	defer fm.wg.Done()

	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fm.ctx.Done():
			return // Context cancelled, exit
		case <-ticker.C:
			fm.rebuildIndexAndCleanup()
		}
	}
}

// rebuildIndexAndCleanup performs unified cleanup by scanning the temporary directory.
// This function is called:
//  1. Once during manager initialization
//  2. Periodically every CleanupInterval (1 minute)
//
// It performs the following tasks:
//  1. Scans all files belonging to this instanceID
//  2. Groups files by version key (same file, different versions)
//  3. Keeps only the latest version of each file
//  4. Removes orphaned files (not in index, older than threshold)
//  5. Performs LRU cleanup if cache size exceeds limit
//
// IMPORTANT: Uses incremental update strategy to avoid concurrent access issues.
// The index is NOT cleared during cleanup, only updated incrementally.
func (fm *FileCopyManager) rebuildIndexAndCleanup() {
	entries, err := os.ReadDir(fm.tempDir)
	if err != nil {
		return // Directory doesn't exist or is inaccessible, skip indexing
	}

	expectedPrefix := fm.instanceID + "_+"
	now := time.Now()
	recentTimeThreshold := now.Add(-RecentFileProtectionWindow)

	// Only use veryRecentThreshold during periodic cleanup (not during initialization)
	// During init, we want to index all files immediately
	isPeriodicCleanup := now.Sub(fm.startTime) > CleanupInterval
	var veryRecentThreshold time.Time
	if isPeriodicCleanup {
		veryRecentThreshold = now.Add(-DedupSkipWindow)
	}

	// Build set of on-disk files later; avoid snapshotting index here to reduce coupling

	// Track valid files and latest version per versionKey
	// versionKey: instanceID_baseName_ext_pathHash
	latestByKey := make(map[string]*indexCandidate)
	diskFiles := make(map[string]*indexCandidate) // All valid files on disk

	// First pass: collect all matching files and group by version key
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), expectedPrefix) {
			continue
		}

		filePath := filepath.Join(fm.tempDir, entry.Name())

		// Parse filename: instanceID_+baseName_+ext_+pathHash_+dataHash.ext
		if candidate := fm.parseFileCandidate(entry.Name(), filePath); candidate != nil {
			// Always add to diskFiles so third pass knows file exists
			diskFiles[filePath] = candidate

			// Skip very recent files from version deduplication during periodic cleanup only
			// During initialization, process all files normally
			if isPeriodicCleanup && candidate.fileInfo.ModTime().After(veryRecentThreshold) {
				continue // Skip version grouping for very recent files during periodic cleanup
			}

			// Extract components for version grouping
			pathHash, _ := parseHashComponents(candidate.hash)
			versionKey := fm.generateVersionKey(fm.instanceID, candidate.baseName, candidate.ext, pathHash)
			if cur := latestByKey[versionKey]; cur == nil || candidate.timestamp > cur.timestamp {
				latestByKey[versionKey] = candidate
			}
		} else {
			// File doesn't match expected pattern. Consider orphan cleanup.
			if info, err := entry.Info(); err == nil {
				name := entry.Name()
				// Never delete temp intermediates
				if strings.Contains(filePath, ".tmp.") {
					continue
				}
				// If declared ext exists and differs from actual ext, ignore (e.g., .db-wal/.db-shm)
				if declared, ok := declaredExtFromName(name); ok {
					actual := extractFileExtension(name)
					if declared != actual {
						continue
					}
				}
				if now.Sub(info.ModTime()) > OrphanFileCleanupThreshold {
					fm.processDeletionInline(filePath)
				}
			}
		}
	}

	// Second pass: ensure each latest version is present in index
	for _, latest := range latestByKey {
		pathHash, dataHash := parseHashComponents(latest.hash)
		latestCacheKey := fm.generateCacheKey(fm.instanceID, latest.baseName, latest.ext, pathHash, dataHash)
		indexEntry := latest.toIndexEntry()
		if _, loaded := fm.fileIndex.LoadOrStore(latestCacheKey, indexEntry); !loaded {
			atomic.AddInt64(&fm.cacheSize, 1)
		}
	}

	// Third pass: queue old versions for deletion (safe window)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), expectedPrefix) {
			continue
		}
		filePath := filepath.Join(fm.tempDir, entry.Name())
		candidate := fm.parseFileCandidate(entry.Name(), filePath)
		if candidate == nil {
			continue
		}
		// Skip very recent files in periodic cleanup
		if isPeriodicCleanup && candidate.fileInfo.ModTime().After(veryRecentThreshold) {
			continue
		}
		pathHash, _ := parseHashComponents(candidate.hash)
		versionKey := fm.generateVersionKey(fm.instanceID, candidate.baseName, candidate.ext, pathHash)
		latest := latestByKey[versionKey]
		if latest == nil || latest.filePath == candidate.filePath {
			continue
		}
		if !candidate.fileInfo.ModTime().Before(recentTimeThreshold) {
			continue
		}
		// Remove from index if present
		_, dataHash := parseHashComponents(candidate.hash)
		cacheKey := fm.generateCacheKey(fm.instanceID, candidate.baseName, candidate.ext, pathHash, dataHash)
		if _, loaded := fm.fileIndex.LoadAndDelete(cacheKey); loaded {
			atomic.AddInt64(&fm.cacheSize, -1)
		}
		// Delete inline (best-effort)
		fm.processDeletionInline(candidate.filePath)
	}

	// Third pass: Remove stale index entries (files that no longer exist on disk)
	// But ONLY if they're not recently accessed (to avoid race with GetTempCopy)
	fm.fileIndex.Range(func(key, value any) bool {
		entry := value.(*FileIndexEntry)

		// Skip if recently accessed (likely in active use)
		if entry.GetLastAccess().After(recentTimeThreshold) {
			return true // Continue iteration
		}

		// Check if file exists on disk
		if _, exists := diskFiles[entry.TempPath]; !exists {
			// File not on disk and not recently accessed, remove from index
			fm.fileIndex.Delete(key)
			atomic.AddInt64(&fm.cacheSize, -1)
		}

		return true
	})

	// Fourth pass: perform LRU cleanup if cache size exceeds limit
	fm.performCacheCleanup()
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
	removeCount := len(entries) / 4
	if removeCount < 1 {
		removeCount = 1
	}

	for i := 0; i < removeCount && i < len(entries); i++ {
		entry := entries[i]
		fm.fileIndex.Delete(entry.key)
		atomic.AddInt64(&fm.cacheSize, -1)

		// Skip .tmp files (safety check - should not happen but防御性编程)
		if strings.Contains(entry.entry.TempPath, ".tmp.") {
			continue
		}

		// Delete inline (best-effort)
		fm.processDeletionInline(entry.entry.TempPath)
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
// Old file versions are cleaned up by the periodic cleanup worker, not here.
func (fm *FileCopyManager) GetTempCopy(originalPath string) (string, error) {
	// Validate original file and get metadata
	stat, err := os.Stat(originalPath)
	if err != nil {
		return "", fmt.Errorf("original file does not exist: %w", err)
	}

	// Ensure only one goroutine copies the same source path at a time
	lock := fm.getPathLock(originalPath)
	lock.Lock()
	defer lock.Unlock()

	now := time.Now()
	currentModTime := stat.ModTime()
	currentSize := stat.Size()
	currentHash := fm.hashString(originalPath)

	// Update last access time for TTL cleanup (no lock needed for time.Time)
	fm.lastAccess = now

	// Compute data hash once per policy under the per-path lock
	var expectedDataHash string
	if VersionDetection == VersionDetectSizeModTime {
		hex := fmt.Sprintf("%x", currentSize+currentModTime.UnixNano())
		if len(hex) > DataHashHexLen {
			expectedDataHash = hex[:DataHashHexLen]
		} else {
			expectedDataHash = hex
		}
	} else {
		expectedDataHash, err = fm.hashFileContent(originalPath, currentSize)
		if err != nil {
			hex := fmt.Sprintf("%x", currentSize+currentModTime.UnixNano())
			if len(hex) > DataHashHexLen {
				expectedDataHash = hex[:DataHashHexLen]
			} else {
				expectedDataHash = hex
			}
		} else if len(expectedDataHash) > DataHashHexLen {
			expectedDataHash = expectedDataHash[:DataHashHexLen]
		}
	}

	// Strategy 1: Check index for existing file using unified cache key
	baseName := fm.extractBaseName(originalPath)
	ext := extractFileExtension(originalPath)
	cacheKey := fm.generateCacheKey(fm.instanceID, baseName, ext, currentHash, expectedDataHash)

	if value, exists := fm.fileIndex.Load(cacheKey); exists {
		entry := value.(*FileIndexEntry)
		if _, err := os.Stat(entry.TempPath); err == nil && currentSize == entry.Size {
			entry.SetLastAccess(now)
			entry.SetOriginalPath(originalPath)
			return entry.TempPath, nil
		}
		// Remove stale index entry; physical deletion handled elsewhere
		fm.fileIndex.Delete(cacheKey)
		atomic.AddInt64(&fm.cacheSize, -1)
	}

	// Strategy 2: No valid cached file found, create new one (avoid re-hashing)
	tempPath := fm.generateTempPathWithHash(originalPath, expectedDataHash)

	if err := fm.atomicCopyFile(originalPath, tempPath); err != nil {
		return "", err
	}

	// Add to index using LoadOrStore to keep cacheSize accurate under races
	newEntry := &FileIndexEntry{
		TempPath:     tempPath,
		OriginalPath: originalPath,
		Size:         currentSize,
		ModTime:      currentModTime,
		lastAccess:   now.UnixNano(),
		PathHash:     currentHash,
		DataHash:     expectedDataHash,
		BaseName:     baseName,
		Extension:    ext, // normalized without dot
	}
	if _, loaded := fm.fileIndex.LoadOrStore(cacheKey, newEntry); !loaded {
		atomic.AddInt64(&fm.cacheSize, 1)
	}

	return tempPath, nil
}

// getPathLock returns a per-original-path mutex to serialize copy operations for the same source.
func (fm *FileCopyManager) getPathLock(key string) *sync.Mutex {
	if v, ok := fm.pathLocks.Load(key); ok {
		return v.(*sync.Mutex)
	}
	m := &sync.Mutex{}
	actual, _ := fm.pathLocks.LoadOrStore(key, m)
	return actual.(*sync.Mutex)
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
	// Reserve space for metadata in filename
	if len(baseName) > MaxBaseNameLen {
		baseName = baseName[:MaxBaseNameLen]
	}

	// Generate path hash for collision avoidance
	pathHash := fm.hashString(originalPath)
	if len(pathHash) > PathHashHexLen {
		pathHash = pathHash[:PathHashHexLen]
	}

	// Generate content hash here is expensive and duplicates work with GetTempCopy.
	// Keep this method for callers that don't pre-compute the data hash.
	stat, err := os.Stat(originalPath)
	var dataHash string
	if err != nil {
		hex := fmt.Sprintf("%x", time.Now().UnixNano())
		if len(hex) > DataHashHexLen {
			dataHash = hex[:DataHashHexLen]
		} else {
			dataHash = hex
		}
	} else {
		// Respect version detection policy
		if VersionDetection == VersionDetectSizeModTime {
			hex := fmt.Sprintf("%x", stat.Size()+stat.ModTime().UnixNano())
			if len(hex) > DataHashHexLen {
				dataHash = hex[:DataHashHexLen]
			} else {
				dataHash = hex
			}
		} else {
			if h, err2 := fm.hashFileContent(originalPath, stat.Size()); err2 == nil {
				if len(h) > DataHashHexLen {
					dataHash = h[:DataHashHexLen]
				} else {
					dataHash = h
				}
			} else {
				hex := fmt.Sprintf("%x", stat.Size()+stat.ModTime().UnixNano())
				if len(hex) > DataHashHexLen {
					dataHash = hex[:DataHashHexLen]
				} else {
					dataHash = hex
				}
			}
		}
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

// generateTempPathWithHash creates a temp path using a precomputed dataHash to avoid re-hashing.
func (fm *FileCopyManager) generateTempPathWithHash(originalPath, dataHash string) string {
	fileName := filepath.Base(originalPath)
	fileExt := filepath.Ext(fileName)
	baseName := fileName
	if len(fileExt) > 0 && len(fileName) > len(fileExt) {
		baseName = fileName[:len(fileName)-len(fileExt)]
	}
	if baseName == "" || baseName == fileExt {
		baseName = "file"
	}

	if len(baseName) > MaxBaseNameLen {
		baseName = baseName[:MaxBaseNameLen]
	}

	pathHash := fm.hashString(originalPath)
	if len(pathHash) > PathHashHexLen {
		pathHash = pathHash[:PathHashHexLen]
	}

	cleanExt := strings.TrimPrefix(fileExt, ".")
	if cleanExt == "" {
		cleanExt = "bin"
	}

	if len(dataHash) > DataHashHexLen {
		dataHash = dataHash[:DataHashHexLen]
	}

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

	// Verify temp file still exists before rename with retries (safety check against race conditions)
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if _, err = os.Stat(tempDst); err == nil {
			break // File exists, proceed with rename
		}
		if attempt < maxRetries-1 {
			time.Sleep(10 * time.Millisecond) // Wait before retry
		} else {
			return fmt.Errorf("temporary file disappeared before rename after %d attempts: %w", maxRetries, err)
		}
	}

	// Atomic rename to final destination
	if err = os.Rename(tempDst, dst); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// hashString generates a 64-bit hash of the input string.
// Use xxhash (fast, well-distributed) to reduce collision risk for version grouping.
func (fm *FileCopyManager) hashString(s string) string {
	h := xxhash.New()
	// write never errors for xxhash
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum64())
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
	// Stop periodic cleanup first
	fm.cancel()

	// Delete all cached temporary files inline (best-effort)
	fm.fileIndex.Range(func(key, value any) bool {
		entry := value.(*FileIndexEntry)
		fm.processDeletionInline(entry.TempPath)
		return true
	})

	// Clear all entries and reset cacheSize
	fm.fileIndex.Range(func(key, value any) bool {
		fm.fileIndex.Delete(key)
		return true
	})
	atomic.StoreInt64(&fm.cacheSize, 0)

	// Wait for all goroutines to finish properly
	fm.wg.Wait()

	// Note: Do NOT remove the shared temp directory here as other instances may still be using it
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
