package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FilesystemServer handles filesystem operations
type FilesystemServer struct {
	executor *Executor
}

// NewFilesystemServer creates a new filesystem server
func NewFilesystemServer() *FilesystemServer {
	return &FilesystemServer{
		executor: NewExecutor(),
	}
}

// SearchFiles searches for files with optional content filtering
func (fs *FilesystemServer) SearchFiles(patterns []string, contentFilter string) ([]map[string]interface{}, error) {
	files, err := fs.executor.searchFiles(patterns)
	if err != nil {
		return nil, err
	}
	
	// Filter by content if specified
	if contentFilter != "" {
		var filtered []map[string]interface{}
		for _, file := range files {
			if preview, ok := file["preview"].(string); ok {
				if strings.Contains(strings.ToLower(preview), strings.ToLower(contentFilter)) {
					filtered = append(filtered, file)
				}
			}
		}
		return filtered, nil
	}
	
	return files, nil
}

// ReadFileContent reads complete file content
func (fs *FilesystemServer) ReadFileContent(path string) (map[string]interface{}, error) {
	content, err := fs.executor.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"path":    path,
		"content": content,
		"size":    len(content),
		"lines":   strings.Count(content, "\n") + 1,
	}, nil
}

// ListDirectory lists directory contents with metadata
func (fs *FilesystemServer) ListDirectory(path string) (map[string]interface{}, error) {
	entries, err := fs.executor.ListDirectory(path)
	if err != nil {
		return nil, err
	}
	
	var files, dirs []map[string]interface{}
	for _, entry := range entries {
		if entry["type"] == "directory" {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}
	
	return map[string]interface{}{
		"path":        path,
		"files":       files,
		"directories": dirs,
		"total":       len(entries),
	}, nil
}

// FindInFiles searches for text within files
func (fs *FilesystemServer) FindInFiles(patterns []string, searchText string) ([]map[string]interface{}, error) {
	files, err := fs.SearchFiles(patterns, searchText)
	if err != nil {
		return nil, err
	}
	
	var results []map[string]interface{}
	for _, file := range files {
		path := file["path"].(string)
		content, err := fs.executor.ReadFile(path)
		if err != nil {
			continue
		}
		
		lines := strings.Split(content, "\n")
		var matches []map[string]interface{}
		
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), strings.ToLower(searchText)) {
				matches = append(matches, map[string]interface{}{
					"line_number": i + 1,
					"line":        strings.TrimSpace(line),
				})
			}
		}
		
		if len(matches) > 0 {
			results = append(results, map[string]interface{}{
				"path":    path,
				"matches": matches,
				"count":   len(matches),
			})
		}
	}
	
	return results, nil
}

// GetProjectStructure returns a tree-like structure of the project
func (fs *FilesystemServer) GetProjectStructure(maxDepth int) (map[string]interface{}, error) {
	structure := make(map[string]interface{})
	
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || strings.Contains(path, "/.") {
			return nil
		}
		
		depth := strings.Count(path, string(filepath.Separator))
		if maxDepth > 0 && depth > maxDepth {
			return nil
		}
		
		parts := strings.Split(path, string(filepath.Separator))
		current := structure
		
		for i, part := range parts {
			if part == "." {
				continue
			}
			
			if i == len(parts)-1 {
				// Leaf node
				if info.IsDir() {
					if current[part] == nil {
						current[part] = make(map[string]interface{})
					}
				} else {
					current[part] = map[string]interface{}{
						"type": "file",
						"size": info.Size(),
						"ext":  filepath.Ext(part),
					}
				}
			} else {
				// Directory node
				if current[part] == nil {
					current[part] = make(map[string]interface{})
				}
				current = current[part].(map[string]interface{})
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to build project structure: %w", err)
	}
	
	return structure, nil
}
