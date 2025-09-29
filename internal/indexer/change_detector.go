package indexer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ChangeDetector detects changes in files
type ChangeDetector struct {
	fileHashes map[string]string
	extensions []string
}

// NewChangeDetector creates a new change detector
func NewChangeDetector(extensions []string) *ChangeDetector {
	return &ChangeDetector{
		fileHashes: make(map[string]string),
		extensions: extensions,
	}
}

// DetectChanges detects what files have changed
func (cd *ChangeDetector) DetectChanges(rootPath string) ([]*ChangeEvent, error) {
	var events []*ChangeEvent
	currentFiles := make(map[string]string)
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !cd.isValidExtension(path) {
			return err
		}
		
		hash, err := cd.calculateFileHash(path)
		if err != nil {
			return err
		}
		
		currentFiles[path] = hash
		
		if oldHash, exists := cd.fileHashes[path]; exists {
			if oldHash != hash {
				events = append(events, &ChangeEvent{
					Type:     ChangeTypeModify,
					FilePath: path,
					Time:     time.Now(),
				})
			}
		} else {
			events = append(events, &ChangeEvent{
				Type:     ChangeTypeCreate,
				FilePath: path,
				Time:     time.Now(),
			})
		}
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	for oldPath := range cd.fileHashes {
		if _, exists := currentFiles[oldPath]; !exists {
			events = append(events, &ChangeEvent{
				Type:     ChangeTypeDelete,
				FilePath: oldPath,
				Time:     time.Now(),
			})
		}
	}
	
	cd.fileHashes = currentFiles
	return events, nil
}

func (cd *ChangeDetector) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (cd *ChangeDetector) isValidExtension(filePath string) bool {
	ext := filepath.Ext(filePath)
	for _, validExt := range cd.extensions {
		if strings.EqualFold(ext, validExt) {
			return true
		}
	}
	return false
}
