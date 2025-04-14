package filecopy

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	// Singleton locks to ensure only one thread processes the same file at a time
	fileOperationLocks = make(map[string]*sync.Mutex)
	locksMutex         = sync.RWMutex{}

	// Mapping from original file paths to temporary file paths
	pathToTempFile = make(map[string]string)
	// Metadata information for original files
	fileMetadata = make(map[string]fileMetaInfo)
	// Track old versions of temporary files for each original file
	oldVersions = make(map[string]string)
	mapMutex    = sync.RWMutex{}

	// Temporary directory
	tempDir string
	// Path to the mapping file
	mappingFilePath string

	// Channel for delayed file deletion
	fileDeletionChan = make(chan FileDeletion, 1000)

	// Default deletion delay time (30 seconds)
	DefaultDeletionDelay = 30 * time.Second
)

type FileDeletion struct {
	Path string
	Time time.Time
}

// File metadata information
type fileMetaInfo struct {
	ModTime time.Time `json:"mod_time"`
	Size    int64     `json:"size"`
}

// Persistent mapping information
type persistentMapping struct {
	OriginalPath string       `json:"original_path"`
	TempPath     string       `json:"temp_path"`
	Metadata     fileMetaInfo `json:"metadata"`
}

// Initialize temporary directory
func initTempDir() {
	// Get process name to create a unique temporary directory
	procName := getProcessName()
	tempDir = filepath.Join(os.TempDir(), "filecopy_"+procName)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		tempDir = filepath.Join(os.TempDir(), "filecopy")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			tempDir = os.TempDir()
		}
	}

	// Set mapping file path
	mappingFilePath = filepath.Join(tempDir, "file_mappings.json")

	// Load existing mappings if available
	loadMappings()

	// Scan and clean existing temporary files
	cleanupExistingTempFiles()
}

// Get process name
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

	// Clean name, keep only letters, numbers, underscores and hyphens
	baseName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, baseName)

	return baseName
}

// Load file mappings from persistent storage
func loadMappings() {
	file, err := os.Open(mappingFilePath)
	if err != nil {
		// It's okay if the file doesn't exist yet
		return
	}
	defer file.Close()

	var mappings []persistentMapping
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&mappings); err != nil {
		// If the file is corrupted, we'll just start fresh
		return
	}

	// Restore mappings
	mapMutex.Lock()
	defer mapMutex.Unlock()

	for _, mapping := range mappings {
		// Verify that both the original file and temp file still exist
		origStat, origErr := os.Stat(mapping.OriginalPath)
		_, tempErr := os.Stat(mapping.TempPath)

		if origErr == nil && tempErr == nil {
			// Check if the original file has changed since the mapping was saved
			if origStat.ModTime() == mapping.Metadata.ModTime && origStat.Size() == mapping.Metadata.Size {
				// The mapping is still valid
				pathToTempFile[mapping.OriginalPath] = mapping.TempPath
				fileMetadata[mapping.OriginalPath] = mapping.Metadata
			}
		}
	}
}

// Save file mappings to persistent storage
func saveMappings() {
	mapMutex.RLock()
	defer mapMutex.RUnlock()

	var mappings []persistentMapping
	for origPath, tempPath := range pathToTempFile {
		if meta, exists := fileMetadata[origPath]; exists {
			mappings = append(mappings, persistentMapping{
				OriginalPath: origPath,
				TempPath:     tempPath,
				Metadata:     meta,
			})
		}
	}

	// Create the file
	file, err := os.Create(mappingFilePath)
	if err != nil {
		return
	}
	defer file.Close()

	// Write the mappings
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(mappings); err != nil {
		// If we can't save, just continue - it's not critical
		return
	}
}

// Clean up existing temporary files
func cleanupExistingTempFiles() {
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return
	}

	// Skip the mapping file
	mappingFileName := filepath.Base(mappingFilePath)

	// First, collect all files that are already in our mapping
	knownFiles := make(map[string]bool)
	mapMutex.RLock()
	for _, tempPath := range pathToTempFile {
		knownFiles[tempPath] = true
	}
	mapMutex.RUnlock()

	// Group files by prefix (baseName_hashPrefix)
	fileGroups := make(map[string][]tempFileInfo)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()

		// Skip the mapping file
		if fileName == mappingFileName {
			continue
		}

		filePath := filepath.Join(tempDir, fileName)
		parts := strings.Split(fileName, "_")

		// Skip files that don't match our naming convention
		if len(parts) < 3 {
			removeFileImmediately(filePath)
			continue
		}

		// Extract base name and hash part as key
		baseName := parts[0]
		hashPart := parts[1]
		groupKey := baseName + "_" + hashPart

		// Extract timestamp
		timeStr := strings.Split(parts[2], ".")[0] // Remove extension part
		var timestamp int64
		if _, err := fmt.Sscanf(timeStr, "%d", &timestamp); err != nil {
			removeFileImmediately(filePath)
			continue
		}

		// Add file info to corresponding group
		fileGroups[groupKey] = append(fileGroups[groupKey], tempFileInfo{
			path:      filePath,
			timestamp: timestamp,
		})
	}

	// Process each group of files, keep only the newest one
	for _, fileInfos := range fileGroups {
		if len(fileInfos) == 0 {
			continue
		}

		// Find the newest file
		var newestFile tempFileInfo
		for _, fileInfo := range fileInfos {
			if fileInfo.timestamp > newestFile.timestamp {
				newestFile = fileInfo
			}
		}

		// Delete all files except the newest one
		for _, fileInfo := range fileInfos {
			if fileInfo.path != newestFile.path {
				// If this file is already in our mapping, keep it
				if knownFiles[fileInfo.path] {
					continue
				}
				removeFileImmediately(fileInfo.path)
			}
		}
	}
}

// Temporary file information
type tempFileInfo struct {
	path      string
	timestamp int64
}

// Get file lock
func getFileLock(path string) *sync.Mutex {
	locksMutex.RLock()
	lock, exists := fileOperationLocks[path]
	locksMutex.RUnlock()

	if exists {
		return lock
	}

	locksMutex.Lock()
	defer locksMutex.Unlock()

	// Check again, might have been created while we were acquiring the write lock
	lock, exists = fileOperationLocks[path]
	if !exists {
		lock = &sync.Mutex{}
		fileOperationLocks[path] = lock
	}

	return lock
}

// GetTempCopy returns a temporary copy path of the original file
// If the file hasn't changed since the last copy, returns the existing copy
func GetTempCopy(originalPath string) (string, error) {
	// Get the operation lock for this file to ensure thread safety
	fileLock := getFileLock(originalPath)
	fileLock.Lock()
	defer fileLock.Unlock()

	// Check if original file exists
	stat, err := os.Stat(originalPath)
	if err != nil {
		return "", fmt.Errorf("original file does not exist: %w", err)
	}

	// Current file info
	currentInfo := fileMetaInfo{
		ModTime: stat.ModTime(),
		Size:    stat.Size(),
	}

	// Check existing mapping
	mapMutex.RLock()
	tempPath, pathExists := pathToTempFile[originalPath]
	cachedInfo, infoExists := fileMetadata[originalPath]
	mapMutex.RUnlock()

	// If we have an existing temp file and original file hasn't changed, return it
	if pathExists && infoExists {
		fileChanged := currentInfo.ModTime.After(cachedInfo.ModTime) ||
			currentInfo.Size != cachedInfo.Size

		if !fileChanged {
			// Verify temp file still exists
			if _, err := os.Stat(tempPath); err == nil {
				// Try to open file to verify accessibility
				if file, err := os.Open(tempPath); err == nil {
					file.Close()
					return tempPath, nil
				}
			}
		}
	}

	// Generate new temp file path
	fileName := filepath.Base(originalPath)
	fileExt := filepath.Ext(fileName)
	baseName := fileName[:len(fileName)-len(fileExt)]
	if baseName == "" {
		baseName = "file" // Use default name if empty
	}

	// Generate hash for original path
	pathHash := hashString(originalPath)
	hashPrefix := getHashPrefix(pathHash, 8)

	// Format: basename_pathhash_timestamp.ext
	timestamp := time.Now().UnixNano()
	tempPath = filepath.Join(tempDir,
		fmt.Sprintf("%s_%s_%d%s",
			baseName,
			hashPrefix,
			timestamp,
			fileExt))

	// Copy file (with retry mechanism)
	if err := copyFileWithRetry(originalPath, tempPath, 3); err != nil {
		return "", err
	}

	// Update mappings
	mapMutex.Lock()
	oldPath := pathToTempFile[originalPath]

	// If there's an old path and it's different, move it to old versions and schedule for deletion
	if oldPath != "" && oldPath != tempPath {
		// First clean up previous old version (if any)
		if oldVersionPath, hasOldVersion := oldVersions[originalPath]; hasOldVersion && oldVersionPath != oldPath {
			removeFileImmediately(oldVersionPath)
		}

		// Set current version as old version
		oldVersions[originalPath] = oldPath
		scheduleForDeletion(oldPath)
	}

	// Update to new temp file
	pathToTempFile[originalPath] = tempPath
	fileMetadata[originalPath] = currentInfo
	mapMutex.Unlock()

	// Save mappings to persistent storage
	go saveMappings()

	// Immediately clean up any other related temp files
	go cleanupRelatedTempFiles(originalPath, tempPath, oldPath)

	return tempPath, nil
}

// Immediately clean up other temp files related to the specified original file
func cleanupRelatedTempFiles(originalPath, currentTempPath, knownOldPath string) {
	// Extract hash prefix of original file to match related files
	fileName := filepath.Base(originalPath)
	fileExt := filepath.Ext(fileName)
	baseName := fileName[:len(fileName)-len(fileExt)]
	if baseName == "" {
		baseName = "file"
	}

	pathHash := hashString(originalPath)
	hashPrefix := getHashPrefix(pathHash, 8)

	// File name prefix pattern
	filePrefix := baseName + "_" + hashPrefix

	currentTempPathNoExt := strings.TrimSuffix(currentTempPath, filepath.Ext(currentTempPath))
	knownOldPathNoExt := strings.TrimSuffix(knownOldPath, filepath.Ext(knownOldPath))

	files, err := os.ReadDir(tempDir)
	if err != nil {
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()

		// Skip the mapping file
		if fileName == filepath.Base(mappingFilePath) {
			continue
		}

		filePath := filepath.Join(tempDir, fileName)
		filePathNoExt := strings.TrimSuffix(filePath, filepath.Ext(filePath))

		// Skip current file and known old version
		if filePathNoExt == currentTempPathNoExt || filePathNoExt == knownOldPathNoExt {
			continue
		}

		// If file name matches our pattern, delete it immediately
		if strings.HasPrefix(fileName, filePrefix) {
			removeFileImmediately(filePath)
		}
	}
}

// Immediately delete file without waiting for delay
func removeFileImmediately(path string) {
	if path == "" {
		return
	}

	// Try to delete file
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		// Silently fail if we can't delete
	}
}

// Schedule file for delayed deletion
func scheduleForDeletion(path string) {
	if path == "" {
		return
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}

	// Put file in deletion channel
	select {
	case fileDeletionChan <- FileDeletion{Path: path, Time: time.Now().Add(DefaultDeletionDelay)}:
		// Successfully scheduled
	default:
		// If channel is full, delete file immediately
		removeFileImmediately(path)
	}
}

// File deletion handler
func fileDeletionHandler() {
	for {
		// Get file to delete from channel
		file := <-fileDeletionChan

		if !time.Now().After(file.Time) {
			time.Sleep(time.Until(file.Time))
		}

		// Ensure file is not in active mappings
		isActive := false
		mapMutex.RLock()
		for _, activePath := range pathToTempFile {
			if activePath == file.Path {
				isActive = true
				break
			}
		}

		mapMutex.RUnlock()

		if isActive {
			continue
		}

		// Delete file
		removeFileImmediately(file.Path)
	}
}

// CleanupTempFiles cleans up unused temporary files
func CleanupTempFiles() {
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return
	}

	// Skip the mapping file
	mappingFileName := filepath.Base(mappingFilePath)

	// Get current active temp file paths and old version paths
	mapMutex.RLock()
	activeTempFiles := make(map[string]bool)
	for _, tempFilePath := range pathToTempFile {
		tempFilePath = strings.TrimSuffix(tempFilePath, filepath.Ext(tempFilePath))
		activeTempFiles[tempFilePath] = true
	}
	for _, oldVersionPath := range oldVersions {
		oldVersionPath = strings.TrimSuffix(oldVersionPath, filepath.Ext(oldVersionPath))
		activeTempFiles[oldVersionPath] = true
	}
	mapMutex.RUnlock()

	// Schedule deletion of inactive temp files
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()

		// Skip the mapping file
		if fileName == mappingFileName {
			continue
		}

		tempFilePath := filepath.Join(tempDir, fileName)
		tempFilePath = strings.TrimSuffix(tempFilePath, filepath.Ext(tempFilePath))
		if !activeTempFiles[tempFilePath] {
			scheduleForDeletion(tempFilePath)
		}
	}
}

// Copy file with retry mechanism
func copyFileWithRetry(src, dst string, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = copyFile(src, dst)
		if err == nil {
			return nil
		}

		// Wait before retrying
		time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
	}
	return fmt.Errorf("failed to copy file after %d attempts: %w", maxRetries, err)
}

// Copy file
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		cerr := out.Close()
		if err == nil && cerr != nil {
			err = fmt.Errorf("failed to close destination file: %w", cerr)
		}
	}()

	// Use buffered copy for better performance
	buf := make([]byte, 256*1024) // 256KB buffer
	if _, err = io.CopyBuffer(out, in, buf); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return out.Sync()
}

// Generate hash for string
func hashString(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum32())
}

// Safely get hash prefix, avoid index out of bounds
func getHashPrefix(hash string, length int) string {
	if len(hash) <= length {
		return hash
	}
	return hash[:length]
}

// Initialize temp directory and start background cleanup
func init() {
	// Initialize temp directory and scan existing files
	initTempDir()

	// Start multiple file deletion handlers
	for i := 0; i < 2; i++ {
		go fileDeletionHandler()
	}

	// Start periodic cleanup routine
	go func() {
		for {
			time.Sleep(30 * time.Second)
			CleanupTempFiles()

			// Also periodically save mappings
			saveMappings()
		}
	}()
}
