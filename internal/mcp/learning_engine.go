package mcp

import (
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// LearningEngine learns from query patterns and optimizes future processing
type LearningEngine struct {
	patterns map[string]*LearnedPattern
	metrics  *LearningMetrics
}

// LearnedPattern represents a learned query pattern
type LearnedPattern struct {
	QueryPattern    string        `json:"query_pattern"`
	Intent          IntentType    `json:"intent"`
	SuccessRate     float64       `json:"success_rate"`
	AverageTime     time.Duration `json:"average_time"`
	OptimalOps      []string      `json:"optimal_operations"`
	ContextNeeds    []ContextType `json:"context_needs"`
	UsageCount      int           `json:"usage_count"`
	LastUsed        time.Time     `json:"last_used"`
	Confidence      float64       `json:"confidence"`
}

// LearningMetrics tracks learning performance
type LearningMetrics struct {
	PatternsLearned   int           `json:"patterns_learned"`
	SuccessRate       float64       `json:"success_rate"`
	AverageImprovement float64      `json:"average_improvement"`
	LastUpdate        time.Time     `json:"last_update"`
}

// NewLearningEngine creates a new learning engine
func NewLearningEngine() *LearningEngine {
	return &LearningEngine{
		patterns: make(map[string]*LearnedPattern),
		metrics: &LearningMetrics{
			LastUpdate: time.Now(),
		},
	}
}

// RecordSuccess records a successful query processing
func (le *LearningEngine) RecordSuccess(query *models.Query, intent *ClassifiedIntent, plan *QueryProcessingPlan, duration time.Duration) {
	patternKey := le.generatePatternKey(query, intent)
	
	pattern, exists := le.patterns[patternKey]
	if !exists {
		pattern = &LearnedPattern{
			QueryPattern: patternKey,
			Intent:       intent.Primary,
			OptimalOps:   plan.RequiredOperations,
			ContextNeeds: intent.RequiredContext,
			UsageCount:   0,
			Confidence:   0.5,
		}
		le.patterns[patternKey] = pattern
	}
	
	// Update pattern metrics
	pattern.UsageCount++
	pattern.LastUsed = time.Now()
	
	// Update average time (simple moving average)
	if pattern.AverageTime == 0 {
		pattern.AverageTime = duration
	} else {
		pattern.AverageTime = (pattern.AverageTime + duration) / 2
	}
	
	// Update success rate
	pattern.SuccessRate = (pattern.SuccessRate*float64(pattern.UsageCount-1) + 1.0) / float64(pattern.UsageCount)
	
	// Update confidence based on usage
	pattern.Confidence = min(0.95, 0.5+float64(pattern.UsageCount)*0.1)
	
	// Update global metrics
	le.updateGlobalMetrics()
}

// GetOptimalPlan gets optimal plan for a query based on learned patterns
func (le *LearningEngine) GetOptimalPlan(query *models.Query, intent *ClassifiedIntent) *QueryProcessingPlan {
	patternKey := le.generatePatternKey(query, intent)
	
	if pattern, exists := le.patterns[patternKey]; exists && pattern.Confidence > 0.7 {
		// Use learned optimal plan
		return &QueryProcessingPlan{
			QueryID:            query.ID,
			Intent:             intent,
			RequiredOperations: pattern.OptimalOps,
			ContextDepth:       le.determineOptimalDepth(pattern),
			TokenBudget:        le.calculateOptimalBudget(pattern),
			EstimatedDuration:  pattern.AverageTime,
		}
	}
	
	// Return default plan
	return nil
}

// generatePatternKey generates a key for pattern matching
func (le *LearningEngine) generatePatternKey(query *models.Query, intent *ClassifiedIntent) string {
	// Create pattern key based on intent and key terms
	keyTerms := []string{}
	for _, keyword := range intent.Keywords {
		if len(keyword) > 3 { // Only meaningful keywords
			keyTerms = append(keyTerms, keyword)
		}
	}
	
	if len(keyTerms) > 3 {
		keyTerms = keyTerms[:3] // Limit to top 3 keywords
	}
	
	return fmt.Sprintf("%s_%s", intent.Primary, strings.Join(keyTerms, "_"))
}

// updateGlobalMetrics updates global learning metrics
func (le *LearningEngine) updateGlobalMetrics() {
	totalPatterns := len(le.patterns)
	totalSuccess := 0.0
	
	for _, pattern := range le.patterns {
		totalSuccess += pattern.SuccessRate
	}
	
	if totalPatterns > 0 {
		le.metrics.SuccessRate = totalSuccess / float64(totalPatterns)
	}
	
	le.metrics.PatternsLearned = totalPatterns
	le.metrics.LastUpdate = time.Now()
}

// Helper methods
func (le *LearningEngine) determineOptimalDepth(pattern *LearnedPattern) ContextDepth {
	if len(pattern.ContextNeeds) >= 4 {
		return ContextComprehensive
	} else if len(pattern.ContextNeeds) >= 2 {
		return ContextModerate
	}
	return ContextMinimal
}

func (le *LearningEngine) calculateOptimalBudget(pattern *LearnedPattern) int {
	baseBudget := 2000
	
	switch pattern.Intent {
	case IntentExplain:
		return baseBudget + 2000
	case IntentGenerate:
		return baseBudget + 1500
	default:
		return baseBudget
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}