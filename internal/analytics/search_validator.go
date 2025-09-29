package analytics

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SearchValidator compares vector search vs keyword search accuracy
type SearchValidator struct {
	comparisons []SearchComparison
	costTracker *CostTracker
}

// SearchComparison represents a comparison between search methods
type SearchComparison struct {
	Query           string    `json:"query"`
	VectorResults   []string  `json:"vector_results"`
	KeywordResults  []string  `json:"keyword_results"`
	UserPreferred   string    `json:"user_preferred"`
	VectorCost      float64   `json:"vector_cost"`
	KeywordCost     float64   `json:"keyword_cost"`
	VectorTime      time.Duration `json:"vector_time"`
	KeywordTime     time.Duration `json:"keyword_time"`
	Timestamp       time.Time `json:"timestamp"`
}

// NewSearchValidator creates a new search validator
func NewSearchValidator(costTracker *CostTracker) *SearchValidator {
	return &SearchValidator{
		comparisons: make([]SearchComparison, 0),
		costTracker: costTracker,
	}
}

// CompareSearchMethods performs side-by-side comparison
func (sv *SearchValidator) CompareSearchMethods(ctx context.Context, query string) {
	fmt.Printf("\n🔬 SEARCH METHOD COMPARISON\n")
	fmt.Printf("Query: \"%s\"\n", query)
	fmt.Printf(strings.Repeat("-", 50) + "\n")

	// Method 1: Vector Search (with cost tracking)
	fmt.Printf("🧠 Vector Search (with embeddings):\n")
	vectorStart := time.Now()
	vectorResults, vectorCost := sv.performVectorSearch(ctx, query)
	vectorTime := time.Since(vectorStart)

	fmt.Printf("├─ Results: %v\n", vectorResults)
	fmt.Printf("├─ Cost: $%.6f\n", vectorCost)
	fmt.Printf("└─ Time: %v\n", vectorTime)

	// Method 2: Keyword Search (SQLite FTS)
	fmt.Printf("\n🔍 Keyword Search (SQLite FTS):\n")
	keywordStart := time.Now()
	keywordResults := sv.performKeywordSearch(ctx, query)
	keywordTime := time.Since(keywordStart)

	fmt.Printf("├─ Results: %v\n", keywordResults)
	fmt.Printf("├─ Cost: $0.00\n")
	fmt.Printf("└─ Time: %v\n", keywordTime)

	// Ask user for preference
	fmt.Printf("\nWhich results are more relevant?\n")
	fmt.Printf("(v) Vector search\n")
	fmt.Printf("(k) Keyword search\n")
	fmt.Printf("(b) Both equally good\n")
	fmt.Printf("(n) Neither is good\n")
	fmt.Printf("Choice: ")

	// Record comparison (user input would be collected here)
	comparison := SearchComparison{
		Query:          query,
		VectorResults:  vectorResults,
		KeywordResults: keywordResults,
		VectorCost:     vectorCost,
		KeywordCost:    0.0,
		VectorTime:     vectorTime,
		KeywordTime:    keywordTime,
		Timestamp:      time.Now(),
	}

	sv.comparisons = append(sv.comparisons, comparison)
}

// performVectorSearch simulates vector search with cost tracking
func (sv *SearchValidator) performVectorSearch(ctx context.Context, query string) ([]string, float64) {
	// Simulate embedding generation cost
	estimatedTokens := len(query) / 4
	embeddingCost := float64(estimatedTokens) / 1000.0 * 0.0001

	// Track the cost
	sv.costTracker.RecordEmbeddingCost(estimatedTokens, embeddingCost)

	// Simulate vector search results
	results := []string{
		"internal/agents/search_agent.go:45",
		"internal/app/cli.go:123",
		"cmd/main.go:67",
	}

	return results, embeddingCost
}

// performKeywordSearch simulates SQLite FTS keyword search
func (sv *SearchValidator) performKeywordSearch(ctx context.Context, query string) []string {
	// Simulate keyword search results (would use SQLite FTS)
	keywords := strings.Fields(strings.ToLower(query))
	
	// Mock results based on keywords
	results := []string{
		"internal/agents/manager_agent.go:89",
		"internal/app/cli.go:156",
		"models/query_model.go:23",
	}

	return results
}

// GetAccuracyReport returns search method accuracy comparison
func (sv *SearchValidator) GetAccuracyReport() map[string]interface{} {
	if len(sv.comparisons) == 0 {
		return map[string]interface{}{
			"message": "No search comparisons collected yet",
			"action":  "Run search queries and compare methods",
		}
	}

	vectorPreferred := 0
	keywordPreferred := 0
	bothGood := 0
	neitherGood := 0

	for _, comp := range sv.comparisons {
		switch comp.UserPreferred {
		case "vector":
			vectorPreferred++
		case "keyword":
			keywordPreferred++
		case "both":
			bothGood++
		case "neither":
			neitherGood++
		}
	}

	total := len(sv.comparisons)
	return map[string]interface{}{
		"total_comparisons":     total,
		"vector_preferred":      vectorPreferred,
		"keyword_preferred":     keywordPreferred,
		"both_good":            bothGood,
		"neither_good":         neitherGood,
		"vector_preference_rate": float64(vectorPreferred) / float64(total),
		"keyword_preference_rate": float64(keywordPreferred) / float64(total),
		"recommendation": sv.generateSearchRecommendation(vectorPreferred, keywordPreferred, total),
	}
}

// generateSearchRecommendation provides data-driven search method recommendation
func (sv *SearchValidator) generateSearchRecommendation(vectorPreferred, keywordPreferred, total int) string {
	vectorRate := float64(vectorPreferred) / float64(total)
	keywordRate := float64(keywordPreferred) / float64(total)

	if vectorRate > 0.7 {
		return "✅ Vector search significantly better - worth the embedding costs"
	} else if keywordRate > 0.7 {
		return "💡 Keyword search sufficient - consider SQLite FTS instead"
	} else if vectorRate > keywordRate {
		return "⚖️ Vector search slightly better - marginal benefit for cost"
	} else {
		return "🤔 No clear winner - start with keyword search, add vector later"
	}
}