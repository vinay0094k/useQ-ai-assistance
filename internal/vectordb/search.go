package vectordb

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// SearchService - MINIMAL search functionality
type SearchService struct {
	client   *QdrantClient
	embedder *EmbeddingService
}

// NewSearchService creates a minimal search service
func NewSearchService(client *QdrantClient, embedder *EmbeddingService) *SearchService {
	return &SearchService{
		client:   client,
		embedder: embedder,
	}
}

// Search performs semantic search with SIMPLE ranking
func (ss *SearchService) Search(ctx context.Context, query string, limit int, filters map[string]string) ([]*SearchResult, error) {
	// For Tier 2: Fast search without LLM
	results, err := ss.client.Search(ctx, query, limit)
	if err != nil {
		// Fallback to empty results rather than failing
		fmt.Printf("⚠️ Vector search failed: %v\n", err)
		return []*SearchResult{}, nil
	}

	// Apply simple filtering
	if len(filters) > 0 {
		results = ss.applyFilters(results, filters)
	}

	// SIMPLE ranking - just by similarity score (no complex weights)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// SearchForTier2 optimized search for Tier 2 (no LLM synthesis)
func (ss *SearchService) SearchForTier2(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	results, err := ss.Search(ctx, query, limit, nil)
	if err != nil {
		return []*SearchResult{}, nil
	}

	// Filter for high-confidence results only (Tier 2 needs confidence)
	var highConfidenceResults []*SearchResult
	for _, result := range results {
		if result.Score > 0.7 { // High confidence threshold
			highConfidenceResults = append(highConfidenceResults, result)
		}
	}

	return highConfidenceResults, nil
}

// SearchForTier3 comprehensive search for Tier 3 (LLM context)
func (ss *SearchService) SearchForTier3(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	// Get more results for LLM context (lower threshold)
	results, err := ss.Search(ctx, query, limit*2, nil)
	if err != nil {
		return []*SearchResult{}, nil
	}

	// Lower threshold for Tier 3 (LLM can synthesize lower-confidence results)
	var contextResults []*SearchResult
	for _, result := range results {
		if result.Score > 0.3 { // Lower threshold for LLM context
			contextResults = append(contextResults, result)
		}
		if len(contextResults) >= limit {
			break
		}
	}

	return contextResults, nil
}

// applyFilters applies simple filters to results
func (ss *SearchService) applyFilters(results []*SearchResult, filters map[string]string) []*SearchResult {
	var filtered []*SearchResult

	for _, result := range results {
		include := true

		// Language filter
		if lang, ok := filters["language"]; ok && lang != "" {
			if result.Chunk.Language != lang {
				include = false
			}
		}

		// File path filter
		if filePath, ok := filters["file_path"]; ok && filePath != "" {
			if !strings.Contains(result.Chunk.FilePath, filePath) {
				include = false
			}
		}

		if include {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// GetStats returns simple search statistics
func (ss *SearchService) GetStats() map[string]interface{} {
	costStats := ss.embedder.GetCostStats()
	return map[string]interface{}{
		"embedding_costs": map[string]interface{}{
			"total_cost":     costStats.TotalCost,
			"total_tokens":   costStats.TotalTokens,
			"request_count":  costStats.RequestCount,
			"avg_cost":       costStats.TotalCost / float64(max(costStats.RequestCount, 1)),
		},
		"cache_size": len(ss.embedder.cache),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}