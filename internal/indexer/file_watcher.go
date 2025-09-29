package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher monitors file system changes for real-time indexing updates
type FileWatcher struct {
	watcher         *fsnotify.Watcher
	projectRoot     string
	extensions      []string
	excludedDirs    []string
	config          WatcherConfig
	isRunning       bool
	mu              sync.RWMutex
	eventBuffer     chan FileChangeEvent
	bufferProcessor *EventProcessor
	stats           WatcherStats
}

// WatcherConfig holds file watcher configuration
type WatcherConfig struct {
	BufferSize      int           `json:"buffer_size"`
	ProcessingDelay time.Duration `json:"processing_delay"`
	BatchSize       int           `json:"batch_size"`
	IgnoreHidden    bool          `json:"ignore_hidden"`
	IgnoreTemp      bool          `json:"ignore_temp"`
	MaxEventRate    int           `json:"max_event_rate"` // events per second
	EventTimeout    time.Duration `json:"event_timeout"`
}

// FileChangeEvent represents a file system event
type FileChangeEvent struct {
	Type      FileChangeEventType `json:"type"`
	Path      string              `json:"path"`
	Timestamp time.Time           `json:"timestamp"`
	Size      int64               `json:"size,omitempty"`
	IsDir     bool                `json:"is_dir"`
	OldPath   string              `json:"old_path,omitempty"` // for rename events
}

// FileChangeEventType represents different types of file change events
type FileChangeEventType string

const (
	FileChangeEventCreated  FileChangeEventType = "created"
	FileChangeEventModified FileChangeEventType = "modified"
	FileChangeEventDeleted  FileChangeEventType = "deleted"
	FileChangeEventRenamed  FileChangeEventType = "renamed"
	FileChangeEventChmod    FileChangeEventType = "chmod"
)

// FileChangeHandler is a function that handles file change events
type FileChangeHandler func(event FileChangeEvent)

// EventProcessor processes file change events in batches
type EventProcessor struct {
	eventChan chan FileChangeEvent
	batchSize int
	delay     time.Duration
	handler   FileChangeHandler
	isRunning bool
	mu        sync.RWMutex
}

// WatcherStats tracks file watching statistics
type WatcherStats struct {
	EventsProcessed int64        `json:"events_processed"`
	EventsIgnored   int64        `json:"events_ignored"`
	EventsBuffered  int          `json:"events_buffered"`
	StartTime       time.Time    `json:"start_time"`
	LastEvent       time.Time    `json:"last_event"`
	WatchedPaths    int          `json:"watched_paths"`
	ProcessingRate  float64      `json:"processing_rate"` // events per second
	mu              sync.RWMutex `json:"-"`
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(projectRoot string, extensions, excludedDirs []string) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	config := WatcherConfig{
		BufferSize:      1000,
		ProcessingDelay: 500 * time.Millisecond,
		BatchSize:       10,
		IgnoreHidden:    true,
		IgnoreTemp:      true,
		MaxEventRate:    100, // 100 events per second max
		EventTimeout:    30 * time.Second,
	}

	fw := &FileWatcher{
		watcher:      watcher,
		projectRoot:  projectRoot,
		extensions:   extensions,
		excludedDirs: excludedDirs,
		config:       config,
		eventBuffer:  make(chan FileChangeEvent, config.BufferSize),
		stats: WatcherStats{
			StartTime: time.Now(),
		},
	}

	fw.bufferProcessor = &EventProcessor{
		eventChan: fw.eventBuffer,
		batchSize: config.BatchSize,
		delay:     config.ProcessingDelay,
	}

	return fw, nil
}

// Start starts the file watcher
func (fw *FileWatcher) Start(ctx context.Context, handler FileChangeHandler) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.isRunning {
		return fmt.Errorf("file watcher is already running")
	}

	// Set up event processor
	fw.bufferProcessor.handler = handler

	// Add directories to watch
	if err := fw.addWatchPaths(); err != nil {
		return fmt.Errorf("failed to add watch paths: %w", err)
	}

	// Start event processing goroutines
	go fw.eventLoop(ctx)
	go fw.bufferProcessor.start(ctx)

	fw.isRunning = true

	fmt.Printf("🔍 File watcher started, monitoring %d paths\n", fw.stats.WatchedPaths)

	return nil
}

// addWatchPaths recursively adds directories to watch
func (fw *FileWatcher) addWatchPaths() error {
	pathsAdded := 0

	err := filepath.Walk(fw.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		// Check if directory should be excluded
		if fw.shouldExcludeDir(path) {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if err := fw.watcher.Add(path); err != nil {
			fmt.Printf("⚠️  Failed to watch directory %s: %v\n", path, err)
			return nil // Continue with other directories
		}

		pathsAdded++
		return nil
	})

	fw.stats.mu.Lock()
	fw.stats.WatchedPaths = pathsAdded
	fw.stats.mu.Unlock()

	return err
}

// shouldExcludeDir checks if a directory should be excluded from watching
func (fw *FileWatcher) shouldExcludeDir(path string) bool {
	relPath, _ := filepath.Rel(fw.projectRoot, path)

	// Check excluded directories
	for _, excluded := range fw.excludedDirs {
		if strings.HasPrefix(relPath, excluded) || strings.Contains(relPath, "/"+excluded) {
			return true
		}
	}

	// Ignore hidden directories if configured
	if fw.config.IgnoreHidden {
		parts := strings.Split(relPath, string(filepath.Separator))
		for _, part := range parts {
			if strings.HasPrefix(part, ".") && part != "." {
				return true
			}
		}
	}

	return false
}

// eventLoop processes file system events from fsnotify
func (fw *FileWatcher) eventLoop(ctx context.Context) {
	rateLimiter := time.NewTicker(time.Second / time.Duration(fw.config.MaxEventRate))
	defer rateLimiter.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Rate limiting
			select {
			case <-rateLimiter.C:
				fw.handleFSEvent(event)
			case <-ctx.Done():
				return
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("❌ File watcher error: %v\n", err)
		}
	}
}

// handleFSEvent processes a single file system event
func (fw *FileWatcher) handleFSEvent(event fsnotify.Event) {
	// Skip if not a supported file type
	if !fw.isSupportedFile(event.Name) {
		fw.stats.mu.Lock()
		fw.stats.EventsIgnored++
		fw.stats.mu.Unlock()
		return
	}

	// Skip temporary files if configured
	if fw.config.IgnoreTemp && fw.isTempFile(event.Name) {
		fw.stats.mu.Lock()
		fw.stats.EventsIgnored++
		fw.stats.mu.Unlock()
		return
	}

	// Convert fsnotify event to our event type
	changeEvent := fw.convertEvent(event)

	// Buffer the event
	select {
	case fw.eventBuffer <- changeEvent:
		fw.stats.mu.Lock()
		fw.stats.EventsProcessed++
		fw.stats.LastEvent = time.Now()
		fw.stats.EventsBuffered = len(fw.eventBuffer)

		// Update processing rate
		if elapsed := time.Since(fw.stats.StartTime); elapsed > 0 {
			fw.stats.ProcessingRate = float64(fw.stats.EventsProcessed) / elapsed.Seconds()
		}
		fw.stats.mu.Unlock()

	default:
		// Buffer full, drop event
		fmt.Printf("⚠️  Event buffer full, dropping event for: %s\n", event.Name)
		fw.stats.mu.Lock()
		fw.stats.EventsIgnored++
		fw.stats.mu.Unlock()
	}
}

// convertEvent converts fsnotify.Event to FileChangeEvent
func (fw *FileWatcher) convertEvent(event fsnotify.Event) FileChangeEvent {
	changeEvent := FileChangeEvent{
		Path:      event.Name,
		Timestamp: time.Now(),
	}

	// Get file info
	if info, err := os.Stat(event.Name); err == nil {
		changeEvent.Size = info.Size()
		changeEvent.IsDir = info.IsDir()
	}

	// Determine event type
	if event.Op&fsnotify.Create == fsnotify.Create {
		changeEvent.Type = FileChangeEventCreated
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		changeEvent.Type = FileChangeEventModified
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		changeEvent.Type = FileChangeEventDeleted
	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		changeEvent.Type = FileChangeEventRenamed
	} else if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		changeEvent.Type = FileChangeEventChmod
	}

	return changeEvent
}

// isSupportedFile checks if file has a supported extension
func (fw *FileWatcher) isSupportedFile(path string) bool {
	ext := filepath.Ext(path)
	for _, supportedExt := range fw.extensions {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

// isTempFile checks if file appears to be temporary
func (fw *FileWatcher) isTempFile(path string) bool {
	basename := filepath.Base(path)

	// Common temp file patterns
	tempPatterns := []string{
		"~",           // Vim temp files
		".tmp",        // Generic temp files
		".swp",        // Vim swap files
		".bak",        // Backup files
		".DS_Store",   // macOS
		"Thumbs.db",   // Windows
		".gitignore~", // Git temp files
	}

	for _, pattern := range tempPatterns {
		if strings.HasSuffix(basename, pattern) || strings.HasPrefix(basename, ".#") {
			return true
		}
	}

	return false
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.isRunning {
		return nil
	}

	// Stop the event processor
	if fw.bufferProcessor != nil {
		fw.bufferProcessor.stop()
	}

	// Close the watcher
	if err := fw.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close file watcher: %w", err)
	}

	// Close event buffer
	close(fw.eventBuffer)

	fw.isRunning = false

	fmt.Println("🔍 File watcher stopped")

	return nil
}

// GetStats returns current watcher statistics
func (fw *FileWatcher) GetStats() WatcherStats {
	fw.stats.mu.RLock()
	defer fw.stats.mu.RUnlock()

	stats := fw.stats
	stats.EventsBuffered = len(fw.eventBuffer)

	return stats
}

// IsRunning returns whether the watcher is currently running
func (fw *FileWatcher) IsRunning() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.isRunning
}

// EventProcessor implementation

// start starts the event processor
func (ep *EventProcessor) start(ctx context.Context) {
	ep.mu.Lock()
	ep.isRunning = true
	ep.mu.Unlock()

	ticker := time.NewTicker(ep.delay)
	defer ticker.Stop()

	eventBatch := make([]FileChangeEvent, 0, ep.batchSize)

	for {
		select {
		case <-ctx.Done():
			// Process remaining events before stopping
			if len(eventBatch) > 0 {
				ep.processBatch(eventBatch)
			}
			return

		case event := <-ep.eventChan:
			eventBatch = append(eventBatch, event)

			// Process batch if it's full
			if len(eventBatch) >= ep.batchSize {
				ep.processBatch(eventBatch)
				eventBatch = eventBatch[:0] // Reset slice
			}

		case <-ticker.C:
			// Process batch on timer
			if len(eventBatch) > 0 {
				ep.processBatch(eventBatch)
				eventBatch = eventBatch[:0] // Reset slice
			}
		}
	}
}

// processBatch processes a batch of events
func (ep *EventProcessor) processBatch(events []FileChangeEvent) {
	if ep.handler == nil {
		return
	}

	// Deduplicate events (keep latest for each file)
	eventMap := make(map[string]FileChangeEvent)
	for _, event := range events {
		eventMap[event.Path] = event
	}

	// Process deduplicated events
	for _, event := range eventMap {
		ep.handler(event)
	}
}

// stop stops the event processor
func (ep *EventProcessor) stop() {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.isRunning = false
}

// AddPath dynamically adds a new path to watch
func (fw *FileWatcher) AddPath(path string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.isRunning {
		return fmt.Errorf("file watcher is not running")
	}

	// Check if it's a directory and not excluded
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		if fw.shouldExcludeDir(path) {
			return fmt.Errorf("directory is excluded: %s", path)
		}

		if err := fw.watcher.Add(path); err != nil {
			return fmt.Errorf("failed to add path to watcher: %w", err)
		}

		fw.stats.mu.Lock()
		fw.stats.WatchedPaths++
		fw.stats.mu.Unlock()

		fmt.Printf("🔍 Added new path to watch: %s\n", path)
	}

	return nil
}

// RemovePath removes a path from watching
func (fw *FileWatcher) RemovePath(path string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if !fw.isRunning {
		return fmt.Errorf("file watcher is not running")
	}

	if err := fw.watcher.Remove(path); err != nil {
		return fmt.Errorf("failed to remove path from watcher: %w", err)
	}

	fw.stats.mu.Lock()
	if fw.stats.WatchedPaths > 0 {
		fw.stats.WatchedPaths--
	}
	fw.stats.mu.Unlock()

	fmt.Printf("🔍 Removed path from watch: %s\n", path)

	return nil
}

// GetWatchedPaths returns all currently watched paths
func (fw *FileWatcher) GetWatchedPaths() []string {
	if !fw.isRunning {
		return []string{}
	}

	paths := make([]string, 0)
	for _, path := range fw.watcher.WatchList() {
		paths = append(paths, path)
	}

	return paths
}

// PrintStats prints current statistics to console
func (fw *FileWatcher) PrintStats() {
	stats := fw.GetStats()

	fmt.Printf("\n📊 File Watcher Statistics:\n")
	fmt.Printf("├─ Watched Paths: %d\n", stats.WatchedPaths)
	fmt.Printf("├─ Events Processed: %d\n", stats.EventsProcessed)
	fmt.Printf("├─ Events Ignored: %d\n", stats.EventsIgnored)
	fmt.Printf("├─ Events Buffered: %d\n", stats.EventsBuffered)
	fmt.Printf("├─ Processing Rate: %.2f events/sec\n", stats.ProcessingRate)
	fmt.Printf("├─ Running Since: %s\n", stats.StartTime.Format("15:04:05"))

	if !stats.LastEvent.IsZero() {
		fmt.Printf("└─ Last Event: %s ago\n", time.Since(stats.LastEvent).Truncate(time.Second))
	} else {
		fmt.Printf("└─ Last Event: None\n")
	}
}
