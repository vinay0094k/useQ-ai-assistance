package vectordb

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// SearchService provides high-level search functionality for the 3-tier system
type SearchService struct {
	client   *QdrantClient
	embedder *EmbeddingService
	ranker   *RankingService
}

// NewSearchService creates a new search service
func NewSearchService(client *QdrantClient, embedder *EmbeddingService) *SearchService {
	return &SearchService{
		client:   client,
		embedder: embedder,
		ranker:   NewRankingService(),
	}
}

// Search performs intelligent search with tier-aware optimization
func (ss *SearchService) Search(ctx context.Context, query string, limit int, filters map[string]string) ([]*SearchResult, error) {
	// Generate embedding for the query
	queryEmbedding, err := ss.embedder.GenerateEmbedding(ctx, query)
	if err != nil {
		// Fallback to simple search if embedding fails
		return ss.performSimpleSearch(ctx, query, limit)
	}

	// Perform vector search
	results, err := ss.client.searchVectors(ctx, queryEmbedding, limit*2) // Get more results for ranking
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Apply filters if provided
	if len(filters) > 0 {
		results = ss.applyFilters(results, filters)
	}

	// Rank and limit results
	rankedResults := ss.ranker.RankResults(results, query, nil)
	if len(rankedResults) > limit {
		rankedResults = rankedResults[:limit]
	}

	return rankedResults, nil
}

// SearchSimilar finds similar code chunks
func (ss *SearchService) SearchSimilar(ctx context.Context, chunk *CodeChunk, limit int) ([]*SearchResult, error) {
	// Use the chunk content as the search query
	return ss.Search(ctx, chunk.Content, limit, map[string]string{
		"language": chunk.Language,
	})
}

// SearchByFunction searches for functions specifically
func (ss *SearchService) SearchByFunction(ctx context.Context, functionName string, language string, limit int) ([]*SearchResult, error) {
	query := fmt.Sprintf("function %s %s", functionName, language)
	return ss.Search(ctx, query, limit, map[string]string{
		"language":   language,
		"chunk_type": "function",
	})
}

// SearchByFile searches within specific files
func (ss *SearchService) SearchByFile(ctx context.Context, query string, filePath string, limit int) ([]*SearchResult, error) {
	return ss.Search(ctx, query, limit, map[string]string{
		"file_path": filePath,
	})
}

// performSimpleSearch performs keyword-based search as fallback
func (ss *SearchService) performSimpleSearch(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	// This would implement keyword-based search
	// For now, return empty results
	fmt.Printf("⚠️ Falling back to simple search for: %s\n", query)
	return []*SearchResult{}, nil
}

// applyFilters applies search filters to results
func (ss *SearchService) applyFilters(results []*SearchResult, filters map[string]string) []*SearchResult {
	var filtered []*SearchResult

	for _, result := range results {
		include := true

		// Apply language filter
		if lang, ok := filters["language"]; ok && lang != "" {
			if result.Chunk.Language != lang {
				include = false
			}
		}

		// Apply file path filter
		if filePath, ok := filters["file_path"]; ok && filePath != "" {
			if !strings.Contains(result.Chunk.FilePath, filePath) {
				include = false
			}
		}

		// Apply chunk type filter
		if chunkType, ok := filters["chunk_type"]; ok && chunkType != "" {
			if result.Chunk.ChunkType != chunkType {
				include = false
			}
		}

		if include {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// GetSearchStats returns search statistics
func (ss *SearchService) GetSearchStats() map[string]interface{} {
	return map[string]interface{}{
		"embedding_cache_size": ss.embedder.GetCacheStats(),
		"client_healthy":       ss.client.Health(context.Background()) == nil,
	}
}

// =============================================================================
// RANKING SERVICE
// =============================================================================

// RankingService handles intelligent result ranking
type RankingService struct {
	weights RankingWeights
}

// RankingWeights defines ranking factors
type RankingWeights struct {
	Similarity    float64 `json:"similarity"`
	TextMatch     float64 `json:"text_match"`
	FileRelevance float64 `json:"file_relevance"`
	Recency       float64 `json:"recency"`
	Frequency     float64 `json:"frequency"`
}

// NewRankingService creates a new ranking service
func NewRankingService() *RankingService {
	return &RankingService{
		weights: RankingWeights{
			Similarity:    0.6,  // Primary factor
			TextMatch:     0.2,  // Keyword matching
			FileRelevance: 0.1,  // File importance
			Recency:       0.05, // How recent the code is
			Frequency:     0.05, // How often accessed
		},
	}
}

// RankResults ranks search results using multiple factors
func (rs *RankingService) RankResults(results []*SearchResult, query string, contextFiles []string) []*SearchResult {
	// Calculate composite scores
	for _, result := range results {
		result.Score = rs.calculateCompositeScore(result, query, contextFiles)
	}

	// Sort by score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// calculateCompositeScore calculates a composite relevance score
func (rs *RankingService) calculateCompositeScore(result *SearchResult, query string, contextFiles []string) float32 {
	// Base similarity score from vector search
	similarityScore := float64(result.Score)

	// Text match score (keyword overlap)
	textMatchScore := rs.calculateTextMatch(result.Chunk.Content, query)

	// File relevance score
	fileRelevanceScore := rs.calculateFileRelevance(result.Chunk.FilePath, contextFiles)

	// Recency score (placeholder - would use actual file modification times)
	recencyScore := 0.5

	// Frequency score (placeholder - would use access patterns)
	frequencyScore := 0.5

	// Calculate weighted composite score
	compositeScore := rs.weights.Similarity*similarityScore +
		rs.weights.TextMatch*textMatchScore +
		rs.weights.FileRelevance*fileRelevanceScore +
		rs.weights.Recency*recencyScore +
		rs.weights.Frequency*frequencyScore

	return float32(compositeScore)
}

// calculateTextMatch calculates keyword matching score
func (rs *RankingService) calculateTextMatch(content, query string) float64 {
	contentLower := strings.ToLower(content)
	queryLower := strings.ToLower(query)

	queryWords := strings.Fields(queryLower)
	if len(queryWords) == 0 {
		return 0
	}

	matches := 0
	for _, word := range queryWords {
		if len(word) > 2 && strings.Contains(contentLower, word) {
			matches++
		}
	}

	return float64(matches) / float64(len(queryWords))
}

// calculateFileRelevance calculates file importance score
func (rs *RankingService) calculateFileRelevance(filePath string, contextFiles []string) float64 {
	// Boost score for important architectural files
	importantFiles := []string{
		"main.go", "cli.go", "manager", "agent", "service", "handler",
	}

	for _, important := range importantFiles {
		if strings.Contains(strings.ToLower(filePath), important) {
			return 1.0
		}
	}

	// Check context files
	for _, contextFile := range contextFiles {
		if strings.Contains(filePath, contextFile) {
			return 0.8
		}
	}

	return 0.3 // Base relevance
}

// BoostMCPDiscoveredFiles boosts scores for files discovered by MCP
func (rs *RankingService) BoostMCPDiscoveredFiles(results []*SearchResult, mcpFiles []string) []*SearchResult {
	mcpFileMap := make(map[string]bool)
	for _, file := range mcpFiles {
		mcpFileMap[file] = true
	}

	for _, result := range results {
		if mcpFileMap[result.Chunk.FilePath] {
			result.Score += 0.1 // Boost MCP-discovered files
		}
	}

	return results
}