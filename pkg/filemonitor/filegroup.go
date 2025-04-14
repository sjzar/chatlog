package filemonitor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// FileChangeCallback defines the callback function signature for file change events
type FileChangeCallback func(event fsnotify.Event) error

// FileGroup represents a group of files with the same processing logic
type FileGroup struct {
	ID         string               // Unique identifier
	RootDir    string               // Root directory
	Pattern    *regexp.Regexp       // File matching pattern
	PatternStr string               // Original pattern string for rebuilding
	Blacklist  []string             // Blacklist patterns
	Callbacks  []FileChangeCallback // File change callbacks
	mutex      sync.RWMutex         // Concurrency control
}

// NewFileGroup creates a new file group
func NewFileGroup(id, rootDir, pattern string, blacklist []string) (*FileGroup, error) {
	// Compile the regular expression
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
	}

	// Normalize root directory path
	rootDir = filepath.Clean(rootDir)

	return &FileGroup{
		ID:         id,
		RootDir:    rootDir,
		Pattern:    re,
		PatternStr: pattern,
		Blacklist:  blacklist,
		Callbacks:  []FileChangeCallback{},
	}, nil
}

// AddCallback adds a callback function to the file group
func (fg *FileGroup) AddCallback(callback FileChangeCallback) {
	fg.mutex.Lock()
	defer fg.mutex.Unlock()

	fg.Callbacks = append(fg.Callbacks, callback)
}

// RemoveCallback removes a callback function from the file group
func (fg *FileGroup) RemoveCallback(callbackToRemove FileChangeCallback) bool {
	fg.mutex.Lock()
	defer fg.mutex.Unlock()

	for i, callback := range fg.Callbacks {
		// Compare function addresses
		if fmt.Sprintf("%p", callback) == fmt.Sprintf("%p", callbackToRemove) {
			// Remove the callback
			fg.Callbacks = append(fg.Callbacks[:i], fg.Callbacks[i+1:]...)
			return true
		}
	}

	return false
}

// Match checks if a file path matches this group's criteria
func (fg *FileGroup) Match(path string) bool {
	// Normalize paths for comparison
	path = filepath.Clean(path)
	rootDir := filepath.Clean(fg.RootDir)

	// Check if path is under root directory
	// Use filepath.Rel to handle path comparison safely across different OSes
	relPath, err := filepath.Rel(rootDir, path)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return false
	}

	// Check if file matches pattern
	if !fg.Pattern.MatchString(filepath.Base(path)) {
		return false
	}

	// Check blacklist
	for _, blackItem := range fg.Blacklist {
		if strings.Contains(relPath, blackItem) {
			return false
		}
	}

	return true
}

// List returns a list of files in the group (real-time scan)
func (fg *FileGroup) List() ([]string, error) {
	files := []string{}

	// Scan directory for matching files using fs.WalkDir
	err := fs.WalkDir(os.DirFS(fg.RootDir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Convert relative path to absolute
		absPath := filepath.Join(fg.RootDir, path)

		// Use Match function to check if file belongs to this group
		if fg.Match(absPath) {
			files = append(files, absPath)
		}

		return nil
	})

	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("error listing files: %w", err)
	}

	return files, nil
}

// ListMatchingDirectories returns directories containing matching files
func (fg *FileGroup) ListMatchingDirectories() (map[string]bool, error) {
	directories := make(map[string]bool)

	// Get matching files
	files, err := fg.List()
	if err != nil {
		return nil, err
	}

	// Extract directories from matching files
	for _, file := range files {
		dir := filepath.Dir(file)
		directories[dir] = true
	}

	return directories, nil
}

// HandleEvent processes a file event and triggers callbacks if the file matches
func (fg *FileGroup) HandleEvent(event fsnotify.Event) {
	// Check if this event is relevant for this group
	if !fg.Match(event.Name) {
		return
	}

	// Get callbacks under read lock
	fg.mutex.RLock()
	callbacks := make([]FileChangeCallback, len(fg.Callbacks))
	copy(callbacks, fg.Callbacks)
	fg.mutex.RUnlock()

	// Asynchronously call callbacks
	for _, callback := range callbacks {
		go func(cb FileChangeCallback) {
			if err := cb(event); err != nil {
				log.Error().
					Str("file", event.Name).
					Str("op", event.Op.String()).
					Err(err).
					Msg("Callback error")
			}
		}(callback)
	}
}
