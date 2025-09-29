package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// CodingAgent handles code generation queries
type CodingAgent struct {
	deps    *Dependencies
	metrics *Metrics
}

// NewCodingAgent creates a new coding agent
func NewCodingAgent(deps *Dependencies) *CodingAgent {
	return &CodingAgent{
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
func (ca *CodingAgent) CanHandle(ctx context.Context, query *models.Query) (bool, float64) {
	input := strings.ToLower(query.UserInput)
	
	// High confidence for code generation terms
	codeTerms := []string{"create", "generate", "write", "build", "implement"}
	confidence := 0.2 // Base confidence
	
	for _, term := range codeTerms {
		if strings.Contains(input, term) {
			confidence += 0.5
			break
		}
	}
	
	// Boost for specific code elements
	elements := []string{"function", "method", "struct", "handler", "service"}
	for _, element := range elements {
		if strings.Contains(input, element) {
			confidence += 0.3
			break
		}
	}
	
	return confidence >= 0.6, confidence
}

// Process handles the code generation query
func (ca *CodingAgent) Process(ctx context.Context, query *models.Query) (*models.Response, error) {
	startTime := time.Now()
	ca.updateMetrics(startTime)

	if ca.deps.LLMManager == nil {
		return ca.createFallbackResponse(query, "LLM Manager not available"), nil
	}

	// Create LLM request
	request := map[string]interface{}{
		"prompt":      fmt.Sprintf("Generate %s code for: %s", query.Language, query.UserInput),
		"max_tokens":  1000,
		"temperature": 0.3,
	}

	// Generate code using LLM
	llmResponse, err := ca.deps.LLMManager.Generate(ctx, request)
	if err != nil {
		ca.metrics.ErrorCount++
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	// Create response
	response := &models.Response{
		ID:      fmt.Sprintf("coding_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeCode,
		Content: models.ResponseContent{
			Text: "Generated code based on your request:",
			Code: &models.CodeResponse{
				Language:    query.Language,
				Code:        fmt.Sprintf("%v", llmResponse), // Simplified conversion
				Explanation: "Generated code using AI",
			},
		},
		AgentUsed:  "coding_agent",
		Provider:   "llm",
		TokenUsage: models.TokenUsage{TotalTokens: 100}, // Placeholder
		Cost:       models.Cost{TotalCost: 0.01, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(startTime),
			Confidence:     0.8,
		},
		Timestamp: time.Now(),
	}

	ca.updateSuccessMetrics(startTime, 0.8, response)
	return response, nil
}

// GetCapabilities returns coding agent capabilities
func (ca *CodingAgent) GetCapabilities() Capabilities {
	return Capabilities{
		CanGenerateCode:    true,
		CanSearchCode:      false,
		CanAnalyzeCode:     true,
		SupportedLanguages: []string{"go", "javascript", "python"},
		MaxComplexity:      8,
		RequiresContext:    true,
	}
}

// GetMetrics returns current metrics
func (ca *CodingAgent) GetMetrics() Metrics {
	return *ca.metrics
}

// Helper methods
func (ca *CodingAgent) updateMetrics(startTime time.Time) {
	ca.metrics.QueriesHandled++
	ca.metrics.LastUsed = startTime
}

func (ca *CodingAgent) updateSuccessMetrics(startTime time.Time, confidence float64, response *models.Response) {
	duration := time.Since(startTime)
	ca.metrics.AverageResponseTime = (ca.metrics.AverageResponseTime + duration) / 2
	ca.metrics.AverageConfidence = (ca.metrics.AverageConfidence + confidence) / 2
	ca.metrics.SuccessRate = float64(ca.metrics.QueriesHandled-ca.metrics.ErrorCount) / float64(ca.metrics.QueriesHandled)
	ca.metrics.TokensUsed += int64(response.TokenUsage.TotalTokens)
	ca.metrics.TotalCost += response.Cost.TotalCost
}

func (ca *CodingAgent) createFallbackResponse(query *models.Query, reason string) *models.Response {
	return &models.Response{
		ID:      fmt.Sprintf("coding_fallback_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeCode,
		Content: models.ResponseContent{
			Text: fmt.Sprintf("Code generation request: '%s'\n\nStatus: %s\n\nTo enable code generation:\n1. ✅ Intent parsing (Ready)\n2. ❌ LLM integration (Connect required)\n3. ✅ Code templates (Ready)", query.UserInput, reason),
		},
		AgentUsed:  "coding_agent",
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