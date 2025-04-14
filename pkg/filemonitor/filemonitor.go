package filemonitor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// FileMonitor manages multiple file groups
type FileMonitor struct {
	groups     map[string]*FileGroup // Map of file groups
	watcher    *fsnotify.Watcher     // File system watcher
	watchDirs  map[string]bool       // Monitored directories
	blacklist  []string              // Global blacklist patterns
	mutex      sync.RWMutex          // Concurrency control for groups and watchDirs
	stopCh     chan struct{}         // Stop signal
	wg         sync.WaitGroup        // Wait group
	isRunning  bool                  // Running state flag
	stateMutex sync.RWMutex          // State mutex
}

func (fm *FileMonitor) Watcher() *fsnotify.Watcher {
	return fm.watcher
}

// NewFileMonitor creates a new file monitor
func NewFileMonitor() *FileMonitor {
	return &FileMonitor{
		groups:    make(map[string]*FileGroup),
		watchDirs: make(map[string]bool),
		blacklist: []string{},
		isRunning: false,
	}
}

// SetBlacklist sets the global directory blacklist
func (fm *FileMonitor) SetBlacklist(blacklist []string) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	fm.blacklist = make([]string, len(blacklist))
	copy(fm.blacklist, blacklist)
}

// AddGroup adds a new file group
func (fm *FileMonitor) AddGroup(group *FileGroup) error {
	if group == nil {
		return errors.New("group cannot be nil")
	}

	// First check if monitor is running
	isRunning := fm.IsRunning()

	// Add group to monitor
	fm.mutex.Lock()
	// Check if ID already exists
	if _, exists := fm.groups[group.ID]; exists {
		fm.mutex.Unlock()
		return fmt.Errorf("group with ID '%s' already exists", group.ID)
	}
	// Add to monitor
	fm.groups[group.ID] = group
	fm.mutex.Unlock()

	// If monitor is running, set up watching
	if isRunning {
		if err := fm.setupWatchForGroup(group); err != nil {
			// Remove group on failure
			fm.mutex.Lock()
			delete(fm.groups, group.ID)
			fm.mutex.Unlock()
			return err
		}
	}

	return nil
}

// CreateGroup creates and adds a new file group (convenience method)
func (fm *FileMonitor) CreateGroup(id, rootDir, pattern string, blacklist []string) (*FileGroup, error) {
	// Create file group
	group, err := NewFileGroup(id, rootDir, pattern, blacklist)
	if err != nil {
		return nil, err
	}

	// Add to monitor
	if err := fm.AddGroup(group); err != nil {
		return nil, err
	}

	return group, nil
}

// RemoveGroup removes a file group
func (fm *FileMonitor) RemoveGroup(id string) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	// Check if group exists
	_, exists := fm.groups[id]
	if !exists {
		return fmt.Errorf("group with ID '%s' does not exist", id)
	}

	// Remove group
	delete(fm.groups, id)
	// log.Info().Str("groupID", id).Msg("Removed file group")

	return nil
}

// GetGroups returns a list of all file group IDs
func (fm *FileMonitor) GetGroups() []*FileGroup {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	groups := make([]*FileGroup, 0, len(fm.groups))
	for _, group := range fm.groups {
		groups = append(groups, group)
	}

	return groups
}

// GetGroup returns the specified file group
func (fm *FileMonitor) GetGroup(id string) (*FileGroup, bool) {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	group, exists := fm.groups[id]
	return group, exists
}

// Start starts the file monitor
func (fm *FileMonitor) Start() error {
	// Check if already running
	fm.stateMutex.Lock()
	if fm.isRunning {
		fm.stateMutex.Unlock()
		return errors.New("file monitor is already running")
	}

	// Create new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fm.stateMutex.Unlock()
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	fm.watcher = watcher

	// Reset stop channel
	fm.stopCh = make(chan struct{})

	// Get groups to monitor (without holding the state lock)
	fm.mutex.RLock()
	groups := make([]*FileGroup, 0, len(fm.groups))
	for _, group := range fm.groups {
		groups = append(groups, group)
	}
	fm.mutex.RUnlock()

	// Reset monitored directories
	fm.mutex.Lock()
	fm.watchDirs = make(map[string]bool)
	fm.mutex.Unlock()

	// Mark as running before setting up watches
	fm.isRunning = true
	fm.stateMutex.Unlock()

	// Set up monitoring for all groups (without holding any locks)
	for _, group := range groups {
		if err := fm.setupWatchForGroup(group); err != nil {
			// Clean up resources on failure
			_ = fm.watcher.Close()

			// Reset running state
			fm.stateMutex.Lock()
			fm.watcher = nil
			fm.isRunning = false
			fm.stateMutex.Unlock()

			return fmt.Errorf("failed to setup watch for group '%s': %w", group.ID, err)
		}
	}

	// Start watch loop
	fm.wg.Add(1)
	go fm.watchLoop()

	// log.Info().Msg("File monitor started")
	return nil
}

// Stop stops the file monitor
func (fm *FileMonitor) Stop() error {
	// Check if already stopped
	fm.stateMutex.Lock()
	if !fm.isRunning {
		fm.stateMutex.Unlock()
		return errors.New("file monitor is not running")
	}

	// Get watcher reference before changing state
	watcher := fm.watcher

	// Send stop signal
	close(fm.stopCh)

	// Mark as not running
	fm.isRunning = false
	fm.stateMutex.Unlock()

	// Wait for all goroutines to exit
	fm.wg.Wait()

	// Close watcher
	if watcher != nil {
		if err := watcher.Close(); err != nil {
			return fmt.Errorf("failed to close watcher: %w", err)
		}

		fm.stateMutex.Lock()
		fm.watcher = nil
		fm.stateMutex.Unlock()
	}

	// log.Info().Msg("File monitor stopped")
	return nil
}

// IsRunning returns whether the file monitor is running
func (fm *FileMonitor) IsRunning() bool {
	fm.stateMutex.RLock()
	defer fm.stateMutex.RUnlock()
	return fm.isRunning
}

// addWatchDir adds a directory to monitoring
func (fm *FileMonitor) addWatchDir(dirPath string) error {
	// Check global blacklist first
	fm.mutex.RLock()
	for _, pattern := range fm.blacklist {
		if strings.Contains(dirPath, pattern) {
			fm.mutex.RUnlock()
			log.Debug().Str("dir", dirPath).Msg("Skipping blacklisted directory")
			return nil
		}
	}
	fm.mutex.RUnlock()

	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	// Check if directory is already being monitored
	if _, watched := fm.watchDirs[dirPath]; watched {
		return nil // Already monitored, no need to add again
	}

	// Add to monitoring
	if err := fm.watcher.Add(dirPath); err != nil {
		return fmt.Errorf("failed to watch directory '%s': %w", dirPath, err)
	}

	fm.watchDirs[dirPath] = true
	// log.Debug().Str("dir", dirPath).Msg("Added watch for directory")
	return nil
}

// setupWatchForGroup sets up monitoring for a file group
func (fm *FileMonitor) setupWatchForGroup(group *FileGroup) error {
	// Check if file monitor is running
	if !fm.IsRunning() {
		return errors.New("file monitor is not running")
	}

	// Find directories containing matching files
	matchingDirs, err := group.ListMatchingDirectories()
	if err != nil {
		return fmt.Errorf("failed to list matching directories: %w", err)
	}

	// Always watch the root directory to catch new files
	rootDir := filepath.Clean(group.RootDir)
	if err := fm.addWatchDir(rootDir); err != nil {
		return err
	}

	// Watch directories containing matching files
	for dir := range matchingDirs {
		if err := fm.addWatchDir(dir); err != nil {
			return err
		}
	}

	return nil
}

// RefreshWatches updates the watched directories based on current matching files
func (fm *FileMonitor) RefreshWatches() error {
	// Check if file monitor is running
	if !fm.IsRunning() {
		return errors.New("file monitor is not running")
	}

	// Get groups to refresh
	fm.mutex.RLock()
	groups := make([]*FileGroup, 0, len(fm.groups))
	for _, group := range fm.groups {
		groups = append(groups, group)
	}
	fm.mutex.RUnlock()

	// Reset watched directories
	fm.mutex.Lock()
	oldWatchDirs := fm.watchDirs
	fm.watchDirs = make(map[string]bool)
	fm.mutex.Unlock()

	// Setup watches for each group
	for _, group := range groups {
		if err := fm.setupWatchForGroup(group); err != nil {
			return fmt.Errorf("failed to refresh watches for group '%s': %w", group.ID, err)
		}
	}

	// Remove watches for directories no longer needed
	for dir := range oldWatchDirs {
		fm.mutex.RLock()
		_, stillWatched := fm.watchDirs[dir]
		fm.mutex.RUnlock()

		if !stillWatched && fm.watcher != nil {
			_ = fm.watcher.Remove(dir)
			log.Debug().Str("dir", dir).Msg("Removed watch for directory")
		}
	}

	return nil
}

// watchLoop monitors for file system events
func (fm *FileMonitor) watchLoop() {
	defer fm.wg.Done()

	for {
		select {
		case <-fm.stopCh:
			return

		case event, ok := <-fm.watcher.Events:
			if !ok {
				// Channel closed, exit loop
				return
			}

			// Handle directory creation events to add new watches
			info, err := os.Stat(event.Name)
			if err == nil && info.IsDir() && event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
				// Add new directory to monitoring
				if err := fm.addWatchDir(event.Name); err != nil {
					log.Error().
						Str("dir", event.Name).
						Err(err).
						Msg("Error watching new directory")
				}
				continue
			}

			// For file creation/modification, check if we need to watch its directory
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				// Check if this file matches any group
				shouldWatch := false

				fm.mutex.RLock()
				for _, group := range fm.groups {
					if group.Match(event.Name) {
						shouldWatch = true
						break
					}
				}
				fm.mutex.RUnlock()

				// If file matches, ensure its directory is watched
				if shouldWatch {
					dir := filepath.Dir(event.Name)
					if err := fm.addWatchDir(dir); err != nil {
						log.Error().
							Str("dir", dir).
							Err(err).
							Msg("Error watching directory of matching file")
					}
				}
			}

			// Forward event to all groups
			fm.forwardEventToGroups(event)

		case err, ok := <-fm.watcher.Errors:
			if !ok {
				// Channel closed, exit loop
				return
			}
			log.Error().Err(err).Msg("Watcher error")
		}
	}
}

// forwardEventToGroups forwards file events to matching groups
func (fm *FileMonitor) forwardEventToGroups(event fsnotify.Event) {
	// Get a copy of groups to avoid holding lock during processing
	fm.mutex.RLock()
	groupsCopy := make([]*FileGroup, 0, len(fm.groups))
	for _, group := range fm.groups {
		groupsCopy = append(groupsCopy, group)
	}
	fm.mutex.RUnlock()

	// Forward to all groups - each group will check if the event is relevant
	for _, group := range groupsCopy {
		group.HandleEvent(event)
	}
}
