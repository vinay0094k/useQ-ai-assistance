package vectordb

import (
	"sort"
	"strings"
)

func NewRankingService() *RankingService {
	return &RankingService{
		weights: RankingWeights{
			Similarity:    0.6,
			TextMatch:     0.2,
			FileRelevance: 0.1,
			Recency:       0.1,
		},
	}
}

func (rs *RankingService) RankResults(results []*SearchResult, query string, contextFiles []string) []*SearchResult {
	for _, result := range results {
		result.Score = rs.calculateCompositeScore(result, query, contextFiles)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

func (rs *RankingService) calculateCompositeScore(result *SearchResult, query string, contextFiles []string) float32 {
	similarityScore := float64(result.Score)
	textMatchScore := rs.calculateTextMatch(result.Chunk.Content, query)
	fileRelevanceScore := rs.calculateFileRelevance(result.Chunk.FilePath, contextFiles)
	
	compositeScore := rs.weights.Similarity*similarityScore +
		rs.weights.TextMatch*textMatchScore +
		rs.weights.FileRelevance*fileRelevanceScore

	return float32(compositeScore)
}

func (rs *RankingService) calculateTextMatch(content, query string) float64 {
	contentLower := strings.ToLower(content)
	queryLower := strings.ToLower(query)
	
	queryWords := strings.Fields(queryLower)
	matches := 0
	
	for _, word := range queryWords {
		if strings.Contains(contentLower, word) {
			matches++
		}
	}
	
	if len(queryWords) == 0 {
		return 0
	}
	return float64(matches) / float64(len(queryWords))
}

func (rs *RankingService) calculateFileRelevance(filePath string, contextFiles []string) float64 {
	for _, contextFile := range contextFiles {
		if strings.Contains(filePath, contextFile) {
			return 1.0
		}
	}
	return 0.0
}
