package indexer

import (
	"context"
)

// UltraFastIndexer combines batch and parallel processing for maximum performance
type UltraFastIndexer struct {
	batchProcessor    *BatchProcessor
	parallelProcessor *ParallelProcessor
	config            IndexerConfig
}

// NewUltraFastIndexer creates a new ultra fast indexer
func NewUltraFastIndexer(config IndexerConfig) *UltraFastIndexer {
	return &UltraFastIndexer{
		batchProcessor:    NewBatchProcessor(config.BatchSize),
		parallelProcessor: NewParallelProcessor(config.MaxWorkers),
		config:            config,
	}
}

// IndexFiles processes files with maximum efficiency
func (ufi *UltraFastIndexer) IndexFiles(ctx context.Context, files []string, processor func(string) *ProcessingResult) ([]*ProcessingResult, error) {
	results := ufi.parallelProcessor.ProcessParallel(ctx, files, processor)
	return results, nil
}

// IndexFilesInBatches processes files in batches
func (ufi *UltraFastIndexer) IndexFilesInBatches(ctx context.Context, files []string, batchProcessor func([]string) ([]*ProcessingResult, error)) ([]*ProcessingResult, error) {
	return ufi.batchProcessor.ProcessBatch(ctx, files, batchProcessor)
}
