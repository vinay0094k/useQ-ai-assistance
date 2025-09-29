package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// SearchAgent handles code search queries
type SearchAgent struct {
	deps    *Dependencies
	metrics *Metrics
}

// NewSearchAgent creates a new search agent
func NewSearchAgent(deps *Dependencies) *SearchAgent {
	return &SearchAgent{
		deps: deps,
		metrics: &Metrics{
			QueriesHandled:      0,
			SuccessRate:         0.0,
			AverageResponseTime: 0,
			AverageConfidence:   0.0,
			TokensUsed:          0,
			TotalCost:           0.0,
			LastUsed:            time.Now(),
			ErrorCount:          0,
		},
	}
}

// CanHandle determines if this agent can handle the query
func (sa *SearchAgent) CanHandle(ctx context.Context, query *models.Query) (bool, float64) {
	input := strings.ToLower(query.UserInput)
	
	// High confidence for explicit search terms
	searchTerms := []string{"search", "find", "show", "list", "where", "how many"}
	confidence := 0.3 // Base confidence
	
	for _, term := range searchTerms {
		if strings.Contains(input, term) {
			confidence += 0.4
			break
		}
	}
	
	// Boost for code-related terms
	codeTerms := []string{"function", "method", "struct", "interface", "file"}
	for _, term := range codeTerms {
		if strings.Contains(input, term) {
			confidence += 0.2
			break
		}
	}
	
	return confidence >= 0.5, confidence
}

// Process handles the search query
func (sa *SearchAgent) Process(ctx context.Context, query *models.Query) (*models.Response, error) {
	startTime := time.Now()
	sa.updateMetrics(startTime)

	// Simple search implementation
	if sa.deps.VectorDB == nil {
		return sa.createFallbackResponse(query, "Vector database not available"), nil
	}

	// Perform search
	results, err := sa.deps.VectorDB.Search(ctx, query.UserInput, 10)
	if err != nil {
		sa.metrics.ErrorCount++
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Create response
	response := &models.Response{
		ID:      fmt.Sprintf("search_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSearch,
		Content: models.ResponseContent{
			Text: sa.formatSearchResults(results),
		},
		AgentUsed:  "search_agent",
		Provider:   "internal",
		TokenUsage: models.TokenUsage{TotalTokens: 0},
		Cost:       models.Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(startTime),
			Confidence:     0.8,
		},
		Timestamp: time.Now(),
	}

	sa.updateSuccessMetrics(startTime, 0.8, response)
	return response, nil
}

// GetCapabilities returns search agent capabilities
func (sa *SearchAgent) GetCapabilities() Capabilities {
	return Capabilities{
		CanGenerateCode:    false,
		CanSearchCode:      true,
		CanAnalyzeCode:     false,
		SupportedLanguages: []string{"go", "javascript", "python"},
		MaxComplexity:      5,
		RequiresContext:    false,
	}
}

// GetMetrics returns current metrics
func (sa *SearchAgent) GetMetrics() Metrics {
	return *sa.metrics
}

// Helper methods
func (sa *SearchAgent) updateMetrics(startTime time.Time) {
	sa.metrics.QueriesHandled++
	sa.metrics.LastUsed = startTime
}

func (sa *SearchAgent) updateSuccessMetrics(startTime time.Time, confidence float64, response *models.Response) {
	duration := time.Since(startTime)
	sa.metrics.AverageResponseTime = (sa.metrics.AverageResponseTime + duration) / 2
	sa.metrics.AverageConfidence = (sa.metrics.AverageConfidence + confidence) / 2
	sa.metrics.SuccessRate = float64(sa.metrics.QueriesHandled-sa.metrics.ErrorCount) / float64(sa.metrics.QueriesHandled)
}

func (sa *SearchAgent) createFallbackResponse(query *models.Query, reason string) *models.Response {
	return &models.Response{
		ID:      fmt.Sprintf("search_fallback_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSearch,
		Content: models.ResponseContent{
			Text: fmt.Sprintf("Search request: '%s'\n\nStatus: %s\n\nTo enable search:\n1. ✅ Query parsing (Ready)\n2. ❌ Vector database (Connect required)\n3. ✅ Search logic (Ready)", query.UserInput, reason),
		},
		AgentUsed:  "search_agent",
		Provider:   "none",
		TokenUsage: models.TokenUsage{TotalTokens: 0},
		Cost:       models.Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(time.Now()),
			Confidence:     0.6,
		},
		Timestamp: time.Now(),
	}
}

func (sa *SearchAgent) formatSearchResults(results []interface{}) string {
	if len(results) == 0 {
		return "No results found matching your search criteria."
	}
	
	return fmt.Sprintf("Found %d results for your search query.", len(results))
}