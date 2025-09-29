package indexer

import (
	"context"
	"sync"
)

// ParallelProcessor handles parallel processing of files
type ParallelProcessor struct {
	maxWorkers int
}

// NewParallelProcessor creates a new parallel processor
func NewParallelProcessor(maxWorkers int) *ParallelProcessor {
	return &ParallelProcessor{
		maxWorkers: maxWorkers,
	}
}

// ProcessParallel processes files in parallel
func (pp *ParallelProcessor) ProcessParallel(ctx context.Context, files []string, processor func(string) *ProcessingResult) []*ProcessingResult {
	jobs := make(chan string, len(files))
	results := make(chan *ProcessingResult, len(files))
	
	var wg sync.WaitGroup
	for i := 0; i < pp.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					result := processor(file)
					results <- result
				}
			}
		}()
	}
	
	go func() {
		defer close(jobs)
		for _, file := range files {
			select {
			case <-ctx.Done():
				return
			case jobs <- file:
			}
		}
	}()
	
	go func() {
		wg.Wait()
		close(results)
	}()
	
	var allResults []*ProcessingResult
	for result := range results {
		allResults = append(allResults, result)
	}
	
	return allResults
}
