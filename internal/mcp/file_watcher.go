package mcp

import (
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher monitors filesystem changes for MCP cache invalidation
type FileWatcher struct {
	watcher   *fsnotify.Watcher
	cache     *MCPContextCache
	watchDirs map[string]bool
	mu        sync.RWMutex
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(cache *MCPContextCache) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	
	fw := &FileWatcher{
		watcher:   watcher,
		cache:     cache,
		watchDirs: make(map[string]bool),
	}
	
	go fw.watchLoop()
	
	return fw, nil
}

// AddWatch adds a directory to watch
func (fw *FileWatcher) AddWatch(projectPath string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	
	if fw.watchDirs[projectPath] {
		return nil // Already watching
	}
	
	err := fw.watcher.Add(projectPath)
	if err != nil {
		return err
	}
	
	fw.watchDirs[projectPath] = true
	return nil
}

// RemoveWatch removes a directory from watching
func (fw *FileWatcher) RemoveWatch(projectPath string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	
	if !fw.watchDirs[projectPath] {
		return nil // Not watching
	}
	
	err := fw.watcher.Remove(projectPath)
	if err != nil {
		return err
	}
	
	delete(fw.watchDirs, projectPath)
	return nil
}

// watchLoop processes file system events
func (fw *FileWatcher) watchLoop() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)
			
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

// handleEvent processes a single file system event
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Only care about Go files and important config files
	if !fw.isRelevantFile(event.Name) {
		return
	}
	
	// Find project root for this file
	projectPath := fw.findProjectRoot(event.Name)
	if projectPath == "" {
		return
	}
	
	// Invalidate cache for this project
	fw.cache.Invalidate(projectPath)
}

// isRelevantFile checks if file changes should trigger cache invalidation
func (fw *FileWatcher) isRelevantFile(filename string) bool {
	ext := filepath.Ext(filename)
	base := filepath.Base(filename)
	
	// Go files
	if ext == ".go" {
		return true
	}
	
	// Important config files
	relevantFiles := []string{"go.mod", "go.sum", ".gitignore", "Dockerfile"}
	for _, relevant := range relevantFiles {
		if base == relevant {
			return true
		}
	}
	
	return false
}

// findProjectRoot finds the project root directory for a file
func (fw *FileWatcher) findProjectRoot(filePath string) string {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	
	dir := filepath.Dir(filePath)
	
	// Find the watched directory that contains this file
	for watchedDir := range fw.watchDirs {
		if strings.HasPrefix(dir, watchedDir) {
			return watchedDir
		}
	}
	
	return ""
}

// Close stops the file watcher
func (fw *FileWatcher) Close() error {
	return fw.watcher.Close()
}
