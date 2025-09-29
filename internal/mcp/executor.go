package mcp

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourusername/useq-ai-assistant/models"
)

// Executor handles actual MCP operations
type Executor struct{}

// NewExecutor creates a new executor
func NewExecutor() *Executor {
	return &Executor{}
}

// ExecuteOperations executes the required MCP operations
func (e *Executor) ExecuteOperations(ctx context.Context, requirements *models.MCPRequirements) (*models.MCPContext, error) {
	mcpContext := &models.MCPContext{
		RequiresMCP: true,
		Operations:  []string{},
		Data:        make(map[string]interface{}),
	}
	
	// Execute filesystem operations
	if requirements.NeedsFilesystem {
		files, err := e.searchFiles(requirements.FilePatterns)
		if err == nil {
			mcpContext.Operations = append(mcpContext.Operations, "filesystem_search")
			mcpContext.Data["project_files"] = files
			mcpContext.Data["file_count"] = len(files)
		}
	}
	
	// Execute git operations
	if requirements.NeedsGit {
		gitInfo := e.getGitInfo()
		mcpContext.Operations = append(mcpContext.Operations, "git_context")
		mcpContext.Data["git_info"] = gitInfo
	}
	
	return mcpContext, nil
}

// searchFiles searches for files matching patterns with content preview
func (e *Executor) searchFiles(patterns []string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.Contains(path, "/.") {
			return nil
		}
		
		for _, pattern := range patterns {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				fileInfo := map[string]interface{}{
					"path": path,
					"size": e.getFileSize(path),
					"type": filepath.Ext(path),
				}
				
				// Add content preview for small files
				if size := e.getFileSize(path); size < 10000 {
					if content := e.readFilePreview(path, 200); content != "" {
						fileInfo["preview"] = content
					}
				}
				
				results = append(results, fileInfo)
				break
			}
		}
		return nil
	})
	
	return results, err
}

// readFilePreview reads first n characters of a file
func (e *Executor) readFilePreview(path string, maxChars int) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	
	str := string(content)
	if len(str) > maxChars {
		str = str[:maxChars] + "..."
	}
	
	return str
}

// getFileSize returns file size in bytes
func (e *Executor) getFileSize(path string) int64 {
	if info, err := os.Stat(path); err == nil {
		return info.Size()
	}
	return 0
}

// readFile reads entire file content
func (e *Executor) ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return string(content), nil
}

// writeFile writes content to file
func (e *Executor) WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// listDirectory lists directory contents
func (e *Executor) ListDirectory(path string) ([]map[string]interface{}, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	
	var results []map[string]interface{}
	for _, entry := range entries {
		info, _ := entry.Info()
		results = append(results, map[string]interface{}{
			"name":  entry.Name(),
			"type":  e.getEntryType(entry),
			"size":  info.Size(),
			"path":  filepath.Join(path, entry.Name()),
		})
	}
	
	return results, nil
}

// getEntryType returns entry type as string
func (e *Executor) getEntryType(entry fs.DirEntry) string {
	if entry.IsDir() {
		return "directory"
	}
	return "file"
}

// getGitInfo gets basic git information
func (e *Executor) getGitInfo() map[string]string {
	return map[string]string{
		"branch": "main", // Placeholder - would use git commands
		"status": "clean",
	}
}
