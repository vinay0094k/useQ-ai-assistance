package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatchProcessor handles batch processing of files
type BatchProcessor struct {
	batchSize int
	jobs      map[string]*BatchJob
	mu        sync.RWMutex
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int) *BatchProcessor {
	return &BatchProcessor{
		batchSize: batchSize,
		jobs:      make(map[string]*BatchJob),
	}
}

// ProcessBatch processes files in batches
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, files []string, processor func([]string) ([]*ProcessingResult, error)) ([]*ProcessingResult, error) {
	job := &BatchJob{
		ID:      fmt.Sprintf("batch_%d", time.Now().UnixNano()),
		Files:   files,
		Status:  JobStatusProcessing,
		Started: time.Now(),
	}
	
	bp.mu.Lock()
	bp.jobs[job.ID] = job
	bp.mu.Unlock()
	
	var allResults []*ProcessingResult
	totalFiles := len(files)
	
	for i := 0; i < totalFiles; i += bp.batchSize {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		end := i + bp.batchSize
		if end > totalFiles {
			end = totalFiles
		}
		
		batch := files[i:end]
		results, err := processor(batch)
		if err != nil {
			return allResults, err
		}
		
		allResults = append(allResults, results...)
	}
	
	return allResults, nil
}
