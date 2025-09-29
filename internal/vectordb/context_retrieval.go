package vectordb

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

func NewContextRetrieval(searchService *SearchService, rankingService *RankingService) *ContextRetrieval {
	return &ContextRetrieval{
		searchService:  searchService,
		rankingService: rankingService,
	}
}

func (cr *ContextRetrieval) RetrieveContext(ctx context.Context, query string, maxResults int, contextWindow int) ([]*ContextResult, error) {
	searchResults, err := cr.searchService.Search(ctx, query, maxResults*2, nil)
	if err != nil {
		return nil, err
	}

	rankedResults := cr.rankingService.RankResults(searchResults, query, nil)
	
	contextResults := make([]*ContextResult, 0, maxResults)
	for i, result := range rankedResults {
		if i >= maxResults {
			break
		}

		contextChunks, err := cr.getContextChunks(ctx, result.Chunk, contextWindow)
		if err != nil {
			continue
		}

		contextResult := &ContextResult{
			MainChunk:     result.Chunk,
			ContextChunks: contextChunks,
			Score:         result.Score,
			Summary:       cr.generateSummary(result.Chunk, contextChunks),
		}
		contextResults = append(contextResults, contextResult)
	}

	return contextResults, nil
}

func (cr *ContextRetrieval) getContextChunks(ctx context.Context, mainChunk *CodeChunk, window int) ([]*CodeChunk, error) {
	filters := map[string]string{"file_path": mainChunk.FilePath}
	
	results, err := cr.searchService.Search(ctx, mainChunk.Content, window*2, filters)
	if err != nil {
		return nil, err
	}

	contextChunks := make([]*CodeChunk, 0)
	for _, result := range results {
		if result.Chunk.ID != mainChunk.ID {
			contextChunks = append(contextChunks, result.Chunk)
		}
	}

	sort.Slice(contextChunks, func(i, j int) bool {
		return contextChunks[i].StartLine < contextChunks[j].StartLine
	})

	if len(contextChunks) > window {
		contextChunks = contextChunks[:window]
	}

	return contextChunks, nil
}

func (cr *ContextRetrieval) generateSummary(mainChunk *CodeChunk, contextChunks []*CodeChunk) string {
	var summary strings.Builder
	
	summary.WriteString(fmt.Sprintf("Main: %s:%d-%d", mainChunk.FilePath, mainChunk.StartLine, mainChunk.EndLine))
	
	if len(contextChunks) > 0 {
		summary.WriteString(" | Context: ")
		for i, chunk := range contextChunks {
			if i > 0 {
				summary.WriteString(", ")
			}
			summary.WriteString(fmt.Sprintf("%d-%d", chunk.StartLine, chunk.EndLine))
		}
	}
	
	return summary.String()
}
