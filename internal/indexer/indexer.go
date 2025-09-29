package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/internal/vectordb"
	"github.com/yourusername/useq-ai-assistant/storage"
)

// CodeIndexer handles indexing of code files
type CodeIndexer struct {
	projectRoot  string
	extensions   []string
	excludedDirs []string
	vectorDB     *vectordb.QdrantClient
	storage      *storage.SQLiteDB
	goParser     *GoParser
}

// NewCodeIndexer creates a new code indexer
func NewCodeIndexer(projectRoot string, extensions, excludedDirs []string,
	vectorDB *vectordb.QdrantClient, storage *storage.SQLiteDB) (*CodeIndexer, error) {

	return &CodeIndexer{
		projectRoot:  projectRoot,
		extensions:   extensions,
		excludedDirs: excludedDirs,
		vectorDB:     vectorDB,
		storage:      storage,
		goParser:     NewGoParser(),
	}, nil
}

// StartIndexing begins the indexing process
func (ci *CodeIndexer) StartIndexing(ctx context.Context) error {
	fmt.Println("🔄 Starting code indexing...")

	files, err := ci.scanFiles()
	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	fmt.Printf("📁 Found %d files to index\n", len(files))

	for i, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fmt.Printf("📝 Indexing %d/%d: %s\n", i+1, len(files), file)
			if err := ci.indexFile(file); err != nil {
				fmt.Printf("⚠️ Failed to index %s: %v\n", file, err)
			}
		}
	}

	fmt.Println("✅ Indexing completed")
	return nil
}

// GetIndexedFiles returns list of indexed files
func (ci *CodeIndexer) GetIndexedFiles() ([]string, error) {
	return ci.storage.GetIndexedFiles()
}

// GetProjectRoot returns the project root
func (ci *CodeIndexer) GetProjectRoot() string {
	return ci.projectRoot
}

// Private methods
func (ci *CodeIndexer) scanFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(ci.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		// Check extension
		ext := filepath.Ext(path)
		validExt := false
		for _, validExtension := range ci.extensions {
			if ext == validExtension {
				validExt = true
				break
			}
		}
		if !validExt {
			return nil
		}

		// Check excluded directories
		relPath, _ := filepath.Rel(ci.projectRoot, path)
		for _, excluded := range ci.excludedDirs {
			if strings.Contains(relPath, excluded) {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

func (ci *CodeIndexer) indexFile(filePath string) error {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Create file record
	file := &storage.CodeFile{
		Path:         filePath,
		Name:         filepath.Base(filePath),
		Extension:    filepath.Ext(filePath),
		Size:         int64(len(content)),
		Language:     ci.detectLanguage(filePath),
		Content:      string(content),
		LastModified: ci.getModTime(filePath),
		LastIndexed:  time.Now(),
	}

	// Save to storage
	return ci.storage.SaveFile(file)
}

func (ci *CodeIndexer) detectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".py":
		return "python"
	case ".md":
		return "markdown"
	default:
		return "text"
	}
}

func (ci *CodeIndexer) getModTime(filePath string) time.Time {
	if stat, err := os.Stat(filePath); err == nil {
		return stat.ModTime()
	}
	return time.Now()
}