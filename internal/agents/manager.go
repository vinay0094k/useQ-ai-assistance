package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// Manager routes queries to appropriate agents
type Manager struct {
	searchAgent *SearchAgent
	codingAgent *CodingAgent
	deps        *Dependencies
	metrics     *Metrics
}

// NewManager creates a new agent manager
func NewManager(deps *Dependencies) *Manager {
	return &Manager{
		searchAgent: NewSearchAgent(deps),
		codingAgent: NewCodingAgent(deps),
		deps:        deps,
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

// RouteQuery intelligently routes queries to the best agent
func (m *Manager) RouteQuery(ctx context.Context, query *models.Query) (*models.Response, error) {
	startTime := time.Now()
	m.updateMetrics(startTime)

	// Determine best agent
	agent, confidence := m.selectBestAgent(ctx, query)
	
	if m.deps.Logger != nil {
		m.deps.Logger.Info("Routing query", 
			"query", query.UserInput,
			"selected_agent", agent,
			"confidence", confidence)
	}

	// Route to selected agent
	var response *models.Response
	var err error

	switch agent {
	case "search":
		response, err = m.searchAgent.Process(ctx, query)
	case "coding":
		response, err = m.codingAgent.Process(ctx, query)
	default:
		response, err = m.searchAgent.Process(ctx, query) // Default fallback
	}

	if err != nil {
		m.metrics.ErrorCount++
		return nil, fmt.Errorf("agent processing failed: %w", err)
	}

	m.updateSuccessMetrics(startTime, confidence, response)
	return response, nil
}

// selectBestAgent chooses the most appropriate agent
func (m *Manager) selectBestAgent(ctx context.Context, query *models.Query) (string, float64) {
	// Evaluate each agent
	searchCanHandle, searchConf := m.searchAgent.CanHandle(ctx, query)
	codingCanHandle, codingConf := m.codingAgent.CanHandle(ctx, query)

	// Simple selection logic
	if codingCanHandle && codingConf > searchConf {
		return "coding", codingConf
	}
	
	if searchCanHandle {
		return "search", searchConf
	}

	// Default to search
	return "search", 0.5
}

// GetMetrics returns manager metrics
func (m *Manager) GetMetrics() Metrics {
	return *m.metrics
}

// Helper methods
func (m *Manager) updateMetrics(startTime time.Time) {
	m.metrics.QueriesHandled++
	m.metrics.LastUsed = startTime
}

func (m *Manager) updateSuccessMetrics(startTime time.Time, confidence float64, response *models.Response) {
	duration := time.Since(startTime)
	m.metrics.AverageResponseTime = (m.metrics.AverageResponseTime + duration) / 2
	m.metrics.AverageConfidence = (m.metrics.AverageConfidence + confidence) / 2
	m.metrics.SuccessRate = float64(m.metrics.QueriesHandled-m.metrics.ErrorCount) / float64(m.metrics.QueriesHandled)
	m.metrics.TokensUsed += int64(response.TokenUsage.TotalTokens)
	m.metrics.TotalCost += response.Cost.TotalCost
}