package indexer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/yourusername/useq-ai-assistant/display"
	"github.com/yourusername/useq-ai-assistant/internal/vectordb"
	"github.com/yourusername/useq-ai-assistant/storage"
)

// CodeIndexer handles indexing of code files for semantic search
type CodeIndexer struct {
	projectRoot   string
	extensions    []string
	excludedDirs  []string
	vectorDB      *vectordb.QdrantClient
	storage       *storage.SQLiteDB
	goParser      *GoParser
	fileWatcher   *FileWatcher
	config        IndexerConfig
	indexingMutex sync.RWMutex
	stats         IndexingStats
	embedder      *vectordb.EmbeddingService // Use from vectordb package
}

// IndexingStats tracks indexing statistics
type IndexingStats struct {
	TotalFiles     int           `json:"total_files"`
	IndexedFiles   int           `json:"indexed_files"`
	FailedFiles    int           `json:"failed_files"`
	SkippedFiles   int           `json:"skipped_files"`
	TotalFunctions int           `json:"total_functions"`
	TotalTypes     int           `json:"total_types"`
	StartTime      time.Time     `json:"start_time"`
	LastUpdate     time.Time     `json:"last_update"`
	IndexingTime   time.Duration `json:"indexing_time"`
	ProcessingRate float64       `json:"processing_rate"` // files per second
	mu             sync.RWMutex  `json:"-"`
}

// NewCodeIndexer creates a new code indexer
func NewCodeIndexer(projectRoot string, extensions, excludedDirs []string,
	vectorDB *vectordb.QdrantClient, storage *storage.SQLiteDB) (*CodeIndexer, error) {

	config := IndexerConfig{
		MaxFileSize:     10 * 1024 * 1024, // 10MB
		BatchSize:       50,
		MaxWorkers:      4,
		ChunkSize:       1000,
		ChunkOverlap:    200,
		IndexTimeout:    30 * time.Second,
		EnableWatching:  true,
		SkipBinaryFiles: true,
		SkipTestFiles:   false,
		SkipVendor:      true,
	}

	// Load environment variables first
	_ = godotenv.Load()

	// Use EmbeddingService from vectordb package
	embeddingConfig := &vectordb.EmbeddingConfig{
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Endpoint: "https://api.openai.com/v1/embeddings",
		Model:    "text-embedding-3-small",
	}

	var embedder *vectordb.EmbeddingService
	if embeddingConfig.APIKey != "" {
		embedder = vectordb.NewEmbeddingService(embeddingConfig)
	} else {
		fmt.Printf("⚠️  Warning: OPENAI_API_KEY not set\n")
		fmt.Println("📁 Files will be indexed without embeddings")
	}

	indexer := &CodeIndexer{
		projectRoot:  projectRoot,
		extensions:   extensions,
		excludedDirs: excludedDirs,
		vectorDB:     vectorDB,
		storage:      storage,
		goParser:     NewGoParser(),
		config:       config,
		embedder:     embedder,
		stats: IndexingStats{
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
		},
	}

	// Initialize file watcher if enabled
	if config.EnableWatching {
		watcher, err := NewFileWatcher(projectRoot, extensions, excludedDirs)
		if err != nil {
			return nil, fmt.Errorf("failed to create file watcher: %w", err)
		}
		indexer.fileWatcher = watcher
	}

	return indexer, nil
}

// StartFullReindexingWithProgress forces reindexing of all files with progress tracking
func (ci *CodeIndexer) StartFullReindexingWithProgress(ctx context.Context, progressCallback func(display.IndexingProgress)) error {
	ci.indexingMutex.Lock()
	defer ci.indexingMutex.Unlock()

	// Initialize stats
	ci.stats = IndexingStats{
		StartTime: time.Now(),
	}

	// Scan files
	files, err := ci.scanFiles()
	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	fmt.Printf("🔍 Found %d files to index\n", len(files))
	if len(files) == 0 {
		fmt.Printf("⚠️ No files found to index in project root: %s\n", ci.projectRoot)
		return nil
	}

	ci.stats.TotalFiles = len(files)

	// Process files in batches with forced reindexing
	return ci.processFilesInBatchesForced(ctx, files, progressCallback)
}

// processFilesInBatchesForced processes files in batches, forcing reindex of all files
func (ci *CodeIndexer) processFilesInBatchesForced(ctx context.Context, files []string, progressCallback func(display.IndexingProgress)) error {
	// Create channels
	fileChan := make(chan string, ci.config.BatchSize)
	resultChan := make(chan IndexResult, ci.config.BatchSize)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < ci.config.MaxWorkers; i++ {
		wg.Add(1)
		go ci.workerForced(ctx, fileChan, resultChan, &wg)
	}

	// Start result collector
	go ci.collectResults(resultChan)

	// Send files to workers
	go func() {
		defer close(fileChan)
		for _, file := range files {
			select {
			case <-ctx.Done():
				return
			case fileChan <- file:
			}
		}
	}()

	// Progress reporting
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(resultChan)
		close(done)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return nil
		case <-ticker.C:
			if progressCallback != nil {
				progress := ci.getProgress()
				progressCallback(progress)
			}
		}
	}
}

// workerForced processes files with forced reindexing
func (ci *CodeIndexer) workerForced(ctx context.Context, fileChan <-chan string, resultChan chan<- IndexResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range fileChan {
		select {
		case <-ctx.Done():
			return
		default:
			result := ci.indexFileForced(ctx, file)
			resultChan <- result
		}
	}
}

// indexFileForced indexes a file without checking if it needs reindexing
func (ci *CodeIndexer) indexFileForced(ctx context.Context, filePath string) IndexResult {
	result := IndexResult{
		File:    filePath,
		Success: false,
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to read file: %w", err)
		return result
	}

	// Check file size
	if int64(len(content)) > ci.config.MaxFileSize {
		result.Error = fmt.Errorf("file too large: %d bytes", len(content))
		return result
	}

	// Skip binary files
	if ci.config.SkipBinaryFiles && ci.isBinaryFile(content) {
		fmt.Printf("⏭️ Skipping binary file: %s\n", filePath)
		ci.stats.mu.Lock()
		ci.stats.SkippedFiles++
		ci.stats.mu.Unlock()
		result.Success = true
		return result
	}

	// Create file info
	fileInfo := &FileInfo{
		Path:         filePath,
		Hash:         ci.calculateHash(content),
		Size:         int64(len(content)),
		LastModified: ci.getModTime(filePath),
		Language:     ci.detectLanguage(filePath),
		IndexedAt:    time.Now(),
	}

	fmt.Printf("🔍 File: %s, Language: %s, Size: %d\n", filePath, fileInfo.Language, len(content))

	// Parse file based on language
	switch fileInfo.Language {
	case "go":
		fmt.Printf("🔧 Processing Go file: %s\n", filePath)
		result = ci.indexGoFile(ctx, filePath, string(content), fileInfo)
	default:
		fmt.Printf("🔧 Processing generic file: %s (lang: %s)\n", filePath, fileInfo.Language)
		result = ci.indexGenericFile(ctx, filePath, string(content), fileInfo)
	}

	return result
}

// getProgress returns current indexing progress
func (ci *CodeIndexer) getProgress() display.IndexingProgress {
	ci.stats.mu.RLock()
	defer ci.stats.mu.RUnlock()

	processed := ci.stats.IndexedFiles + ci.stats.FailedFiles + ci.stats.SkippedFiles
	elapsed := time.Since(ci.stats.StartTime)
	filesPerSecond := float64(processed) / elapsed.Seconds()

	return display.IndexingProgress{
		ProcessedFiles: processed,
		TotalFiles:     ci.stats.TotalFiles,
		FunctionsFound: ci.stats.TotalFunctions,
		TypesFound:     ci.stats.TotalTypes,
		ElapsedTime:    elapsed,
		FilesPerSecond: filesPerSecond,
	}
}

// StartIndexingWithProgress begins indexing with progress callback
func (ci *CodeIndexer) StartIndexingWithProgress(ctx context.Context, progressCallback func(display.IndexingProgress)) error {
	ci.indexingMutex.Lock()
	defer ci.indexingMutex.Unlock()

	startTime := time.Now()
	files, err := ci.scanFiles()
	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	totalFiles := len(files)
	var functionsFound, typesFound int

	for i, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result := ci.indexFile(ctx, file)
		fmt.Printf("📋 File %s: Success=%v, Error=%v\n", file, result.Success, result.Error)
		if !result.Success {
			continue
		}

		elapsed := time.Since(startTime)
		progressCallback(display.IndexingProgress{
			ProcessedFiles: i + 1,
			TotalFiles:     totalFiles,
			FunctionsFound: functionsFound,
			TypesFound:     typesFound,
			ElapsedTime:    elapsed,
		})

		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// GetIndexedFiles returns list of indexed files from storage
func (ci *CodeIndexer) GetIndexedFiles() ([]string, error) {
	return ci.storage.GetIndexedFiles()
}

// StartIndexing begins the initial indexing process
func (ci *CodeIndexer) StartIndexing(ctx context.Context) error {
	ci.indexingMutex.Lock()
	defer ci.indexingMutex.Unlock()

	ci.stats.mu.Lock()
	ci.stats.StartTime = time.Now()
	ci.stats.LastUpdate = time.Now()
	ci.stats.mu.Unlock()

	// Scan for files to index
	files, err := ci.scanFiles()
	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	ci.stats.mu.Lock()
	ci.stats.TotalFiles = len(files)
	ci.stats.mu.Unlock()

	// Process files in batches using worker pool
	return ci.processFilesInBatches(ctx, files)
}

// scanFiles scans the project directory for files to index
func (ci *CodeIndexer) scanFiles() ([]string, error) {
	var files []string
	var mu sync.Mutex
	
	fmt.Printf("🔍 Scanning project root: %s\n", ci.projectRoot)
	fmt.Printf("🔍 Looking for extensions: %v\n", ci.extensions)
	
	// Convert to absolute path for debugging
	absPath, _ := filepath.Abs(ci.projectRoot)
	fmt.Printf("🔍 Absolute path: %s\n", absPath)
	
	// Pre-compile extension map for O(1) lookup
	extMap := make(map[string]bool)
	for _, ext := range ci.extensions {
		extMap[ext] = true
	}

	err := filepath.WalkDir(ci.projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Fast directory exclusion check
		if d.IsDir() {
			name := d.Name()
			// Skip common excluded directories immediately (but not root)
			if path != ci.projectRoot && (name == ".git" || name == "vendor" || name == "node_modules" || 
			   name == ".vscode" || name == ".idea" || (strings.HasPrefix(name, ".") && name != ".")) {
				fmt.Printf("⏭️ Skipping common excluded dir: %s\n", path)
				return filepath.SkipDir
			}
			
			// Check configured exclusions only if needed
			relPath, _ := filepath.Rel(ci.projectRoot, path)
			for _, excluded := range ci.excludedDirs {
				if strings.HasPrefix(relPath, excluded) {
					fmt.Printf("⏭️ Skipping configured excluded dir: %s\n", path)
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Fast extension check using map lookup
		ext := filepath.Ext(path)
		if !extMap[ext] {
			return nil
		}

		fmt.Printf("✅ Found matching file: %s\n", path)

		// Skip test files if configured (single check)
		if ci.config.SkipTestFiles && strings.Contains(path, "_test.go") {
			fmt.Printf("⏭️ Skipping test file: %s\n", path)
			return nil
		}

		mu.Lock()
		files = append(files, path)
		mu.Unlock()
		return nil
	})

	fmt.Printf("📊 Total files found: %d\n", len(files))
	return files, err
}

// processFilesInBatches processes files using a worker pool
func (ci *CodeIndexer) processFilesInBatches(ctx context.Context, files []string) error {
	// Create work channels
	fileChan := make(chan string, ci.config.BatchSize)
	resultChan := make(chan IndexResult, ci.config.BatchSize)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < ci.config.MaxWorkers; i++ {
		wg.Add(1)
		go ci.worker(ctx, fileChan, resultChan, &wg)
	}

	// Start result collector
	go ci.collectResults(resultChan)

	// Send files to workers
	go func() {
		defer close(fileChan)
		for _, file := range files {
			select {
			case fileChan <- file:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for all workers to complete
	wg.Wait()
	close(resultChan)

	return nil
}

// IndexResult represents the result of indexing a file
type IndexResult struct {
	File     string
	Success  bool
	Error    error
	FileInfo *FileInfo
	Chunks   []*CodeChunk
}

// worker processes files from the channel
func (ci *CodeIndexer) worker(ctx context.Context, fileChan <-chan string,
	resultChan chan<- IndexResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range fileChan {
		select {
		case <-ctx.Done():
			return
		default:
			result := ci.indexFile(ctx, file)
			resultChan <- result
		}
	}
}

// collectResults collects and processes indexing results
func (ci *CodeIndexer) collectResults(resultChan <-chan IndexResult) {
	for result := range resultChan {
		ci.stats.mu.Lock()
		if result.Success {
			ci.stats.IndexedFiles++

			// Update function and type counts
			if result.FileInfo != nil && result.FileInfo.ParsedData != nil {
				ci.stats.TotalFunctions += len(result.FileInfo.ParsedData.Functions) + len(result.FileInfo.ParsedData.Methods)
				ci.stats.TotalTypes += len(result.FileInfo.ParsedData.Types) + len(result.FileInfo.ParsedData.Interfaces)
			}
		} else {
			ci.stats.FailedFiles++
			if result.Error != nil {
				fmt.Printf("❌ Failed to index %s: %v\n", result.File, result.Error)
			}
		}

		ci.stats.LastUpdate = time.Now()
		ci.stats.IndexingTime = ci.stats.LastUpdate.Sub(ci.stats.StartTime)

		// Calculate processing rate
		if ci.stats.IndexingTime > 0 {
			totalProcessed := ci.stats.IndexedFiles + ci.stats.FailedFiles
			ci.stats.ProcessingRate = float64(totalProcessed) / ci.stats.IndexingTime.Seconds()
		}

		ci.stats.mu.Unlock()

		// Print progress
		ci.printProgress()
	}
}

// indexFile indexes a single file
func (ci *CodeIndexer) indexFile(ctx context.Context, filePath string) IndexResult {
	result := IndexResult{
		File:    filePath,
		Success: false,
	}

	// Check if file needs to be reindexed
	needsReindex, err := ci.needsReindex(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to check if reindex needed: %w", err)
		return result
	}

	if !needsReindex {
		ci.stats.mu.Lock()
		ci.stats.SkippedFiles++
		ci.stats.mu.Unlock()
		result.Success = true
		return result
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to read file: %w", err)
		return result
	}

	// Check file size
	if int64(len(content)) > ci.config.MaxFileSize {
		result.Error = fmt.Errorf("file too large: %d bytes", len(content))
		return result
	}

	// Skip binary files
	if ci.config.SkipBinaryFiles && ci.isBinaryFile(content) {
		fmt.Printf("⏭️ Skipping binary file: %s\n", filePath)
		ci.stats.mu.Lock()
		ci.stats.SkippedFiles++
		ci.stats.mu.Unlock()
		result.Success = true
		return result
	}

	// Create file info
	fileInfo := &FileInfo{
		Path:         filePath,
		Hash:         ci.calculateHash(content),
		Size:         int64(len(content)),
		LastModified: ci.getModTime(filePath),
		Language:     ci.detectLanguage(filePath),
		IndexedAt:    time.Now(),
	}

	// fmt.Printf("🔍 File: %s, Language: %s\n", filePath, fileInfo.Language)
	fmt.Printf("🔍 File: %s, Language: %s, Size: %d\n", filePath, fileInfo.Language, len(content))

	// Parse file based on language
	switch fileInfo.Language {
	case "go":
		fmt.Printf("🔧 Processing Go file: %s\n", filePath)
		result = ci.indexGoFile(ctx, filePath, string(content), fileInfo)
	default:
		fmt.Printf("🔧 Processing generic file: %s (lang: %s)\n", filePath, fileInfo.Language)
		result = ci.indexGenericFile(ctx, filePath, string(content), fileInfo)
	}

	return result
}

// indexGoFile indexes a Go source file
func (ci *CodeIndexer) indexGoFile(ctx context.Context, filePath, content string, fileInfo *FileInfo) IndexResult {
	result := IndexResult{
		File:     filePath,
		Success:  false,
		FileInfo: fileInfo,
	}

	// Parse Go code
	fmt.Printf("🔧 Parsing Go file: %s\n", filePath)
	parsedCode, err := ci.goParser.ParseFile(filePath, content)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse Go file: %w", err)
		fmt.Printf("❌ Go parse error for %s: %v\n", filePath, err)
		return result
	}

	fmt.Printf("📊 Parsed %s: %d functions, %d types\n", filePath, len(parsedCode.Functions), len(parsedCode.Types))

	fileInfo.ParsedData = parsedCode

	// Create chunks for different code elements
	chunks, err := ci.createGoChunks(filePath, content, parsedCode)
	if err != nil {
		result.Error = fmt.Errorf("failed to create chunks: %w", err)
		return result
	}

	// If no chunks created, create a basic file chunk to ensure storage
	if len(chunks) == 0 {
		chunks = []*CodeChunk{{
			ID:         ci.calculateHash([]byte(filePath)),
			FileID:     ci.calculateHash([]byte(filePath)),
			FilePath:   filePath,
			ChunkIndex: 0,
			StartLine:  1,
			EndLine:    len(strings.Split(content, "\n")),
			Language:   "go",
			Type:       ChunkTypeFile,
			Content:    content,
			Context: ChunkContext{
				PackageName: parsedCode.PackageName,
			},
		}}
	}

	result.Chunks = chunks
	fileInfo.ChunkCount = len(chunks)

	// Always store file, even with empty chunks
	if err := ci.storeFileAndChunks(ctx, fileInfo, chunks); err != nil {
		result.Error = fmt.Errorf("failed to store file and chunks: %w", err)
		fmt.Printf("❌ Go file storage error for %s: %v\n", filePath, err)
		return result
	}

	fmt.Printf("✅ Go file indexed: %s (%d functions, %d types)\n", filePath,
		len(parsedCode.Functions), len(parsedCode.Types))
	result.Success = true
	return result
}

// indexGenericFile indexes a non-Go file
func (ci *CodeIndexer) indexGenericFile(ctx context.Context, filePath, content string, fileInfo *FileInfo) IndexResult {
	result := IndexResult{
		File:     filePath,
		Success:  false,
		FileInfo: fileInfo,
	}

	// Create generic chunks
	chunks := ci.createGenericChunks(filePath, content, fileInfo.Language)
	result.Chunks = chunks
	fileInfo.ChunkCount = len(chunks)

	// Store chunks
	if err := ci.storeFileAndChunks(ctx, fileInfo, chunks); err != nil {
		result.Error = fmt.Errorf("failed to store file and chunks: %w", err)
		fmt.Printf("❌ Generic file storage error for %s: %v\n", filePath, err)
		return result
	}

	fmt.Printf("✅ Generic file indexed: %s\n", filePath)
	result.Success = true
	return result
}

// createGoChunks creates chunks for Go code elements
func (ci *CodeIndexer) createGoChunks(filePath, content string, parsed *ParsedCode) ([]*CodeChunk, error) {
	var chunks []*CodeChunk
	lines := strings.Split(content, "\n")

	chunkID := 0

	// Create chunks for functions
	for _, function := range parsed.Functions {
		chunk := &CodeChunk{
			ID:         fmt.Sprintf("%s-func-%d", filePath[:min(8, len(filePath))], chunkID),
			FileID:     ci.calculateHash([]byte(filePath))[:16],
			FilePath:   filePath,
			ChunkIndex: chunkID,
			StartLine:  function.StartLine,
			EndLine:    function.EndLine,
			Language:   "go",
			Type:       ChunkTypeFunction,
			Context: ChunkContext{
				PackageName:  parsed.PackageName,
				FunctionName: function.Name,
				Dependencies: parsed.Dependencies,
			},
			Metadata: map[string]string{
				"function_name": function.Name,
				"visibility":    function.Visibility,
				"signature":     function.Signature,
				"complexity":    fmt.Sprintf("%d", function.Complexity),
				"is_test":       fmt.Sprintf("%t", function.IsTest),
			},
		}

		// Extract function content
		if function.EndLine <= len(lines) {
			chunk.Content = strings.Join(lines[function.StartLine-1:function.EndLine], "\n")
		}

		chunks = append(chunks, chunk)
		chunkID++
	}

	// Create chunks for methods
	for _, method := range parsed.Methods {
		chunk := &CodeChunk{
			ID:         fmt.Sprintf("%s_method_%d", ci.calculateHash([]byte(filePath)), chunkID),
			FileID:     ci.calculateHash([]byte(filePath)),
			FilePath:   filePath,
			ChunkIndex: chunkID,
			StartLine:  method.StartLine,
			EndLine:    method.EndLine,
			Language:   "go",
			Type:       ChunkTypeMethod,
			Context: ChunkContext{
				PackageName:  parsed.PackageName,
				FunctionName: method.Name,
				TypeName:     method.ReceiverType,
				Dependencies: parsed.Dependencies,
			},
			Metadata: map[string]string{
				"method_name":   method.Name,
				"receiver_type": method.ReceiverType,
				"visibility":    method.Visibility,
				"signature":     method.Signature,
				"complexity":    fmt.Sprintf("%d", method.Complexity),
			},
		}

		if method.EndLine <= len(lines) {
			chunk.Content = strings.Join(lines[method.StartLine-1:method.EndLine], "\n")
		}

		chunks = append(chunks, chunk)
		chunkID++
	}

	// Create chunks for types
	for _, typeDef := range parsed.Types {
		chunk := &CodeChunk{
			ID:         fmt.Sprintf("%s_type_%d", ci.calculateHash([]byte(filePath)), chunkID),
			FileID:     ci.calculateHash([]byte(filePath)),
			FilePath:   filePath,
			ChunkIndex: chunkID,
			StartLine:  typeDef.StartLine,
			EndLine:    typeDef.EndLine,
			Language:   "go",
			Type:       ChunkTypeType,
			Context: ChunkContext{
				PackageName:  parsed.PackageName,
				TypeName:     typeDef.Name,
				Dependencies: parsed.Dependencies,
			},
			Metadata: map[string]string{
				"type_name":   typeDef.Name,
				"type_kind":   typeDef.Kind,
				"field_count": fmt.Sprintf("%d", len(typeDef.Fields)),
			},
		}

		if typeDef.EndLine <= len(lines) {
			chunk.Content = strings.Join(lines[typeDef.StartLine-1:typeDef.EndLine], "\n")
		}

		chunks = append(chunks, chunk)
		chunkID++
	}

	// Create chunks for interfaces
	for _, interfaceDef := range parsed.Interfaces {
		chunk := &CodeChunk{
			ID:         fmt.Sprintf("%s_interface_%d", ci.calculateHash([]byte(filePath)), chunkID),
			FileID:     ci.calculateHash([]byte(filePath)),
			FilePath:   filePath,
			ChunkIndex: chunkID,
			StartLine:  interfaceDef.StartLine,
			EndLine:    interfaceDef.EndLine,
			Language:   "go",
			Type:       ChunkTypeInterface,
			Context: ChunkContext{
				PackageName:   parsed.PackageName,
				InterfaceName: interfaceDef.Name,
				Dependencies:  parsed.Dependencies,
			},
			Metadata: map[string]string{
				"interface_name": interfaceDef.Name,
				"method_count":   fmt.Sprintf("%d", len(interfaceDef.Methods)),
			},
		}

		if interfaceDef.EndLine <= len(lines) {
			chunk.Content = strings.Join(lines[interfaceDef.StartLine-1:interfaceDef.EndLine], "\n")
		}

		chunks = append(chunks, chunk)
		chunkID++
	}

	return chunks, nil
}

// createGenericChunks creates chunks for non-Go files
func (ci *CodeIndexer) createGenericChunks(filePath, content, language string) []*CodeChunk {
	var chunks []*CodeChunk
	lines := strings.Split(content, "\n")

	// Simple chunking by size
	chunkSize := ci.config.ChunkSize
	overlap := ci.config.ChunkOverlap

	for i := 0; i < len(lines); i += chunkSize - overlap {
		endIdx := i + chunkSize
		if endIdx > len(lines) {
			endIdx = len(lines)
		}

		chunk := &CodeChunk{
			ID:         fmt.Sprintf("%s_chunk_%d", ci.calculateHash([]byte(filePath)), len(chunks)),
			FileID:     ci.calculateHash([]byte(filePath)),
			FilePath:   filePath,
			ChunkIndex: len(chunks),
			Content:    strings.Join(lines[i:endIdx], "\n"),
			StartLine:  i + 1,
			EndLine:    endIdx,
			Language:   language,
			Type:       ChunkTypeGeneric,
			Metadata: map[string]string{
				"chunk_size": fmt.Sprintf("%d", chunkSize),
			},
		}

		chunks = append(chunks, chunk)

		if endIdx == len(lines) {
			break
		}
	}

	return chunks
}

// storeFileAndChunks stores file metadata and chunks in both SQLite and vector DB
func (ci *CodeIndexer) storeFileAndChunks(ctx context.Context, fileInfo *FileInfo, chunks []*CodeChunk) error {
	fmt.Printf("📁 Storing file: %s\n", fileInfo.Path)

	// Read file content for storage
	content, err := os.ReadFile(fileInfo.Path)
	if err != nil {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	// Store file in SQLite
	sqliteFile := &storage.CodeFile{
		Path:         fileInfo.Path,
		Name:         filepath.Base(fileInfo.Path),
		Extension:    filepath.Ext(fileInfo.Path),
		Size:         fileInfo.Size,
		Hash:         fileInfo.Hash,
		Language:     fileInfo.Language,
		Content:      string(content),
		LastModified: fileInfo.LastModified,
		LastIndexed:  fileInfo.IndexedAt,
	}

	if err := ci.storage.SaveFile(sqliteFile); err != nil {
		fmt.Printf("❌ Failed to save file %s: %v\n", fileInfo.Path, err)
		return fmt.Errorf("failed to save file to SQLite: %w", err)
	}
	fmt.Printf("✅ Saved file to DB: %s\n", fileInfo.Path)

	// Store functions if parsed data is available
	fmt.Printf("🔍 DEBUG: Checking parsed data for %s\n", fileInfo.Path)
	if fileInfo.ParsedData != nil {
		parsedCode := fileInfo.ParsedData
		fmt.Printf("🔍 DEBUG: Found %d functions to store\n", len(parsedCode.Functions))
		for _, function := range parsedCode.Functions {
			fmt.Printf("🔍 DEBUG: Storing function: %s\n", function.Name)
			sqliteFunction := &storage.CodeFunction{
				FileID:     0, // Will be resolved by SaveFunction using file path
				Name:       function.Name,
				Signature:  function.Signature,
				StartLine:  function.StartLine,
				EndLine:    function.EndLine,
				Visibility: function.Visibility,
				Type:       "function",
			}
			// Pass the file path so SaveFunction can resolve the correct file_id
			if err := ci.storage.SaveFunctionForFile(sqliteFunction, fileInfo.Path); err != nil {
				fmt.Printf("❌ Failed to save function %s: %v\n", function.Name, err)
			} else {
				fmt.Printf("✅ Saved function: %s\n", function.Name)
			}
		}
		fmt.Printf("✅ Saved %d functions for %s\n", len(parsedCode.Functions), fileInfo.Path)
	} else {
		fmt.Printf("🔍 DEBUG: No parsed data for %s\n", fileInfo.Path)
	}

	// Store chunks even without embeddings
	for _, chunk := range chunks {
		chunkFile := &storage.CodeFile{
			Path:      fmt.Sprintf("%s#chunk_%d", fileInfo.Path, chunk.ChunkIndex),
			Name:      fmt.Sprintf("chunk_%d", chunk.ChunkIndex),
			Extension: filepath.Ext(fileInfo.Path),
			Content:   chunk.Content,
			Language:  fileInfo.Language,
			Hash:      ci.calculateHash([]byte(chunk.Content)),
		}
		if err := ci.storage.SaveFile(chunkFile); err != nil {
			fmt.Printf("⚠️ Failed to save chunk %d for %s: %v\n", chunk.ChunkIndex, fileInfo.Path, err)
		}
	}
	if ci.vectorDB != nil {
		fmt.Printf("🔄 Processing %d chunks for vector storage\n", len(chunks))
		for _, chunk := range chunks {
			// Generate OpenAI embedding
			embedding, err := ci.vectorDB.GenerateOpenAIEmbedding(ctx, chunk.Content)
			if err != nil {
				fmt.Printf("⚠️ Failed to generate embedding for chunk %s: %v\n", chunk.ID, err)
				continue
			}

			// Create CodeChunk for vector storage
			codeChunk := &vectordb.CodeChunk{
				ID:        chunk.ID,
				Content:   chunk.Content,
				FilePath:  chunk.FilePath,
				Language:  chunk.Language,
				StartLine: chunk.StartLine,
				EndLine:   chunk.EndLine,
			}

			// Store in Qdrant with embedding
			if err := ci.vectorDB.StoreChunkWithEmbedding(ctx, codeChunk, embedding); err != nil {
				fmt.Printf("⚠️ Failed to store chunk in Qdrant: %v\n", err)
			} else {
				fmt.Printf("✅ Stored chunk %s in vector DB\n", chunk.ID)
			}
		}
	} else {
		fmt.Printf("⚠️ VectorDB is nil, skipping vector storage\n")
	}

	return nil
}

// needsReindex checks if a file needs to be reindexed
func (ci *CodeIndexer) needsReindex(filePath string) (bool, error) {
	existingFile, err := ci.storage.GetFile(filePath)
	fmt.Printf("🔍 GetFile(%s): file=%v, err=%v\n", filePath, existingFile != nil, err)
	if err != nil || existingFile == nil {
		fmt.Printf("📋 File %s: NeedsReindex=true (not in DB)\n", filePath)
		return true, nil // File not indexed yet
	}

	// Check if file has been modified
	currentModTime := ci.getModTime(filePath)
	needsUpdate := currentModTime.After(existingFile.LastModified)
	fmt.Printf("📋 File %s: NeedsReindex=%v (mod time check)\n", filePath, needsUpdate)
	return needsUpdate, nil
}

// calculateHash calculates SHA-256 hash
func (ci *CodeIndexer) calculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// GetProjectRoot returns the project root directory
func (ci *CodeIndexer) GetProjectRoot() string {
	return ci.projectRoot
}
func (ci *CodeIndexer) getModTime(filePath string) time.Time {
	if stat, err := os.Stat(filePath); err == nil {
		return stat.ModTime()
	}
	return time.Now()
}

// detectLanguage detects programming language based on file extension
func (ci *CodeIndexer) detectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".cs":
		return "csharp"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".r":
		return "r"
	case ".sh", ".bash":
		return "bash"
	case ".sql":
		return "sql"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".md":
		return "markdown"
	case ".tex":
		return "latex"
	case ".html":
		return "html"
	case ".css":
		return "css"
	default:
		return "text"
	}
}

// isBinaryFile checks if content appears to be binary
func (ci *CodeIndexer) isBinaryFile(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < len(content) && i < 512; i++ {
		if content[i] == 0 {
			return true
		}
	}

	// For known text extensions, don't check ratio
	// This is a simple fix to avoid false positives
	return false
}

// printProgress prints indexing progress
func (ci *CodeIndexer) printProgress() {
	ci.stats.mu.RLock()
	total := ci.stats.IndexedFiles + ci.stats.FailedFiles + ci.stats.SkippedFiles
	percentage := float64(total) / float64(ci.stats.TotalFiles) * 100
	ci.stats.mu.RUnlock()

	fmt.Printf("\r🔄 Indexing: %.1f%% (%d/%d files, %.1f files/sec, %d functions, %d types)",
		percentage, total, ci.stats.TotalFiles, ci.stats.ProcessingRate,
		ci.stats.TotalFunctions, ci.stats.TotalTypes)

	if total == ci.stats.TotalFiles {
		fmt.Println(" ✅ Complete!")
	}
}

// GetStats returns current indexing statistics
func (ci *CodeIndexer) GetStats() IndexingStats {
	ci.stats.mu.RLock()
	defer ci.stats.mu.RUnlock()
	return ci.stats
}

// StartWatching starts watching for file changes
func (ci *CodeIndexer) StartWatching(ctx context.Context) error {
	if ci.fileWatcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}

	return ci.fileWatcher.Start(ctx, ci.handleFileChange)
}

// handleFileChange handles file change events
func (ci *CodeIndexer) handleFileChange(event FileChangeEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), ci.config.IndexTimeout)
	defer cancel()

	switch event.Type {
	case FileChangeEventModified, FileChangeEventCreated:
		fmt.Printf("🔄 Re-indexing changed file: %s\n", event.Path)
		result := ci.indexFile(ctx, event.Path)
		if result.Success {
			fmt.Printf("✅ Successfully re-indexed: %s\n", event.Path)
		} else {
			fmt.Printf("❌ Failed to re-index %s: %v\n", event.Path, result.Error)
		}
	case FileChangeEventDeleted:
		fmt.Printf("🗑️  Removing deleted file from index: %s\n", event.Path)
		if err := ci.removeFileFromIndex(ctx, event.Path); err != nil {
			fmt.Printf("❌ Failed to remove %s from index: %v\n", event.Path, err)
		}
	}
}

// removeFileFromIndex removes a file from both SQLite and vector DB
func (ci *CodeIndexer) removeFileFromIndex(ctx context.Context, filePath string) error {
	// Remove from SQLite
	if err := ci.storage.DeleteFile(filePath); err != nil {
		return fmt.Errorf("failed to delete from SQLite: %w", err)
	}

	// Remove embeddings from vector DB by file path
	// Note: Basic Qdrant client doesn't have DeleteByFilePath, would need custom implementation
	fmt.Printf("⚠️  File deletion from vector DB not implemented for: %s\n", filePath)

	return nil
}

// Stop stops the indexer and cleans up resources
func (ci *CodeIndexer) Stop() error {
	if ci.fileWatcher != nil {
		if err := ci.fileWatcher.Stop(); err != nil {
			return fmt.Errorf("failed to stop file watcher: %w", err)
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Note: FileWatcher, FileChangeEvent, and FileChangeEventType are already implemented
// in your existing file_watcher.go - using those implementations
