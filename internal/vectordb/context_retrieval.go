package vectordb

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// ContextRetrieval provides intelligent context-aware retrieval for the 3-tier system
type ContextRetrieval struct {
	searchService  *SearchService
	rankingService *RankingService
}

// NewContextRetrieval creates a new context retrieval service
func NewContextRetrieval(searchService *SearchService, rankingService *RankingService) *ContextRetrieval {
	return &ContextRetrieval{
		searchService:  searchService,
		rankingService: rankingService,
	}
}

// RetrieveContext retrieves relevant context for a query with surrounding code
func (cr *ContextRetrieval) RetrieveContext(ctx context.Context, query string, maxResults int, contextWindow int) ([]*ContextResult, error) {
	// Perform initial search with more results for better context
	searchResults, err := cr.searchService.Search(ctx, query, maxResults*2, nil)
	if err != nil {
		return nil, fmt.Errorf("context search failed: %w", err)
	}

	// Rank results with context awareness
	rankedResults := cr.rankingService.RankResults(searchResults, query, nil)

	// Build context results with surrounding code
	contextResults := make([]*ContextResult, 0, maxResults)
	for i, result := range rankedResults {
		if i >= maxResults {
			break
		}

		// Get surrounding context chunks
		contextChunks, err := cr.getContextChunks(ctx, result.Chunk, contextWindow)
		if err != nil {
			// Continue with just the main chunk if context retrieval fails
			contextChunks = []*CodeChunk{}
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

// RetrieveForTier2 retrieves context optimized for Tier 2 processing (no LLM)
func (cr *ContextRetrieval) RetrieveForTier2(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	// Optimized search for Tier 2 - focus on exact matches and high relevance
	results, err := cr.searchService.Search(ctx, query, limit, nil)
	if err != nil {
		return nil, err
	}

	// Filter for high-confidence results only
	var highConfidenceResults []*SearchResult
	for _, result := range results {
		if result.Score > 0.7 { // High confidence threshold for Tier 2
			highConfidenceResults = append(highConfidenceResults, result)
		}
	}

	return highConfidenceResults, nil
}

// RetrieveForTier3 retrieves comprehensive context for Tier 3 processing (with LLM)
func (cr *ContextRetrieval) RetrieveForTier3(ctx context.Context, query string, maxResults int) ([]*ContextResult, error) {
	// Comprehensive context retrieval for LLM processing
	return cr.RetrieveContext(ctx, query, maxResults, 3) // 3 lines of context per result
}

// getContextChunks retrieves surrounding code chunks for better context
func (cr *ContextRetrieval) getContextChunks(ctx context.Context, mainChunk *CodeChunk, window int) ([]*CodeChunk, error) {
	// Search for chunks in the same file
	filters := map[string]string{"file_path": mainChunk.FilePath}

	results, err := cr.searchService.Search(ctx, mainChunk.Content, window*2, filters)
	if err != nil {
		return nil, err
	}

	// Filter out the main chunk and sort by line number
	var contextChunks []*CodeChunk
	for _, result := range results {
		if result.Chunk.ID != mainChunk.ID {
			contextChunks = append(contextChunks, result.Chunk)
		}
	}

	// Sort by line number for logical ordering
	sort.Slice(contextChunks, func(i, j int) bool {
		return contextChunks[i].StartLine < contextChunks[j].StartLine
	})

	// Limit to window size
	if len(contextChunks) > window {
		contextChunks = contextChunks[:window]
	}

	return contextChunks, nil
}

// generateSummary creates a summary of the context
func (cr *ContextRetrieval) generateSummary(mainChunk *CodeChunk, contextChunks []*CodeChunk) string {
	var summary strings.Builder

	// Main chunk info
	summary.WriteString(fmt.Sprintf("Main: %s:%d-%d", 
		mainChunk.FilePath, mainChunk.StartLine, mainChunk.EndLine))

	// Add function name if available
	if mainChunk.Function != "" {
		summary.WriteString(fmt.Sprintf(" (%s)", mainChunk.Function))
	}

	// Context info
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

// FindRelatedCode finds code related to a specific chunk
func (cr *ContextRetrieval) FindRelatedCode(ctx context.Context, chunk *CodeChunk, maxResults int) ([]*SearchResult, error) {
	// Build a query from the chunk content and metadata
	query := cr.buildRelatedCodeQuery(chunk)
	
	// Search with filters to find related code
	filters := map[string]string{
		"language": chunk.Language,
	}
	
	// Exclude the original chunk from results
	results, err := cr.searchService.Search(ctx, query, maxResults+1, filters)
	if err != nil {
		return nil, err
	}
	
	// Filter out the original chunk
	var relatedResults []*SearchResult
	for _, result := range results {
		if result.Chunk.ID != chunk.ID {
			relatedResults = append(relatedResults, result)
		}
		if len(relatedResults) >= maxResults {
			break
		}
	}
	
	return relatedResults, nil
}

// buildRelatedCodeQuery builds a query to find related code
func (cr *ContextRetrieval) buildRelatedCodeQuery(chunk *CodeChunk) string {
	var queryParts []string
	
	// Add function name if available
	if chunk.Function != "" {
		queryParts = append(queryParts, chunk.Function)
	}
	
	// Add package name if available
	if chunk.Package != "" {
		queryParts = append(queryParts, chunk.Package)
	}
	
	// Add language
	queryParts = append(queryParts, chunk.Language)
	
	// Add key terms from content (simplified)
	content := strings.ToLower(chunk.Content)
	keyTerms := []string{"struct", "interface", "func", "method", "type"}
	for _, term := range keyTerms {
		if strings.Contains(content, term) {
			queryParts = append(queryParts, term)
		}
	}
	
	return strings.Join(queryParts, " ")
}