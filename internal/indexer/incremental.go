package indexer

import (
	"context"
	"time"
)

// IncrementalIndexer handles incremental indexing operations
type IncrementalIndexer struct {
	changeDetector *ChangeDetector
	ultraFast      *UltraFastIndexer
	state          *IncrementalState
}

// NewIncrementalIndexer creates a new incremental indexer
func NewIncrementalIndexer(extensions []string, config IndexerConfig) *IncrementalIndexer {
	return &IncrementalIndexer{
		changeDetector: NewChangeDetector(extensions),
		ultraFast:      NewUltraFastIndexer(config),
		state: &IncrementalState{
			LastIndexTime: time.Now(),
			FileHashes:    make(map[string]string),
		},
	}
}

// ProcessChanges processes only changed files
func (ii *IncrementalIndexer) ProcessChanges(ctx context.Context, rootPath string, processor func(string) *ProcessingResult) ([]*ProcessingResult, error) {
	events, err := ii.changeDetector.DetectChanges(rootPath)
	if err != nil {
		return nil, err
	}
	
	var changedFiles []string
	for _, event := range events {
		if event.Type == ChangeTypeCreate || event.Type == ChangeTypeModify {
			changedFiles = append(changedFiles, event.FilePath)
		}
	}
	
	if len(changedFiles) == 0 {
		return []*ProcessingResult{}, nil
	}
	
	results, err := ii.ultraFast.IndexFiles(ctx, changedFiles, processor)
	if err != nil {
		return nil, err
	}
	
	ii.state.mu.Lock()
	ii.state.LastIndexTime = time.Now()
	ii.state.mu.Unlock()
	
	return results, nil
}

// GetLastIndexTime returns the last index time
func (ii *IncrementalIndexer) GetLastIndexTime() time.Time {
	ii.state.mu.RLock()
	defer ii.state.mu.RUnlock()
	return ii.state.LastIndexTime
}
