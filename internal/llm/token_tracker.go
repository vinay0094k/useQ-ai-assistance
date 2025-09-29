package llm

import (
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// TokenTracker tracks token usage across sessions and providers
type TokenTracker struct {
	sessions   map[string]*models.SessionTokens
	dailyUsage map[string]*models.TokenMetrics // date -> metrics
	budgets    map[string]*models.TokenBudget  // session -> budget
	mu         sync.RWMutex
}

// CostCalculator calculates costs for different providers
type CostCalculator struct {
	pricingCache map[string]models.ModelPricing
	mu           sync.RWMutex
}

// FallbackHandler manages fallback logic between providers
type FallbackHandler struct {
	maxRetries      int
	retryDelay      time.Duration
	circuitBreakers map[string]*CircuitBreaker // Uses CircuitBreaker from llm_types.go
	mu              sync.RWMutex
}

// NewTokenTracker creates a new token tracker
func NewTokenTracker() *TokenTracker {
	return &TokenTracker{
		sessions:   make(map[string]*models.SessionTokens),
		dailyUsage: make(map[string]*models.TokenMetrics),
		budgets:    make(map[string]*models.TokenBudget),
	}
}

// TrackUsage tracks token usage for a session
func (tt *TokenTracker) TrackUsage(sessionID string, usage models.TokenUsage, cost models.Cost) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	// Get or create session tokens
	session, exists := tt.sessions[sessionID]
	if !exists {
		session = &models.SessionTokens{
			SessionID:      sessionID,
			StartTime:      time.Now(),
			ProviderUsage:  make(map[string]models.ProviderUsage),
			QueryBreakdown: make([]models.QueryTokens, 0),
		}
		tt.sessions[sessionID] = session
	}

	// Update session totals
	session.TotalInputTokens += usage.InputTokens
	session.TotalOutputTokens += usage.OutputTokens
	session.TotalCost += cost.TotalCost
	session.TotalQueries++

	// Update provider usage
	providerUsage, exists := session.ProviderUsage[usage.Provider]
	if !exists {
		providerUsage = models.ProviderUsage{
			Provider: usage.Provider,
		}
	}

	providerUsage.RequestCount++
	providerUsage.InputTokens += usage.InputTokens
	providerUsage.OutputTokens += usage.OutputTokens
	providerUsage.TotalCost += cost.TotalCost
	providerUsage.LastUsed = time.Now()

	// Calculate average latency (simplified)
	if providerUsage.RequestCount > 1 {
		// This would need actual latency tracking
		providerUsage.AverageLatency = time.Duration(providerUsage.RequestCount) * time.Millisecond * 100
	}

	session.ProviderUsage[usage.Provider] = providerUsage

	// Update daily usage
	today := time.Now().Format("2006-01-02")
	dailyMetrics, exists := tt.dailyUsage[today]
	if !exists {
		dailyMetrics = &models.TokenMetrics{
			Period:         models.PeriodDaily,
			StartDate:      time.Now(),
			EndDate:        time.Now(),
			ByProvider:     make(map[string]models.MetricValue),
			ByQueryType:    make(map[string]models.MetricValue),
			ByAgent:        make(map[string]models.MetricValue),
			DailyBreakdown: make([]models.DailyMetric, 0),
		}
		tt.dailyUsage[today] = dailyMetrics
	}

	dailyMetrics.TotalQueries++
	dailyMetrics.TotalTokens += usage.TotalTokens
	dailyMetrics.TotalCost += cost.TotalCost
	dailyMetrics.AverageTokensPerQuery = dailyMetrics.TotalTokens / dailyMetrics.TotalQueries
	dailyMetrics.AverageCostPerQuery = dailyMetrics.TotalCost / float64(dailyMetrics.TotalQueries)

	// Update provider breakdown
	providerMetric := dailyMetrics.ByProvider[usage.Provider]
	providerMetric.Count++
	providerMetric.Tokens += usage.TotalTokens
	providerMetric.Cost += cost.TotalCost
	dailyMetrics.ByProvider[usage.Provider] = providerMetric
}

// GetSessionUsage returns token usage for a session
func (tt *TokenTracker) GetSessionUsage(sessionID string) (*models.SessionTokens, bool) {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	session, exists := tt.sessions[sessionID]
	return session, exists
}

// GetDailyUsage returns daily usage metrics
func (tt *TokenTracker) GetDailyUsage(date string) (*models.TokenMetrics, bool) {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	metrics, exists := tt.dailyUsage[date]
	return metrics, exists
}

// SetBudget sets a budget limit for a session
func (tt *TokenTracker) SetBudget(sessionID string, budget *models.TokenBudget) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	tt.budgets[sessionID] = budget
}

// CheckBudget checks if usage is within budget limits
func (tt *TokenTracker) CheckBudget(sessionID string) (*models.TokenBudget, []models.BudgetWarning) {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	budget, exists := tt.budgets[sessionID]
	if !exists {
		return nil, nil
	}

	session, sessionExists := tt.sessions[sessionID]
	if !sessionExists {
		return budget, nil
	}

	var warnings []models.BudgetWarning

	// Check daily limit
	if budget.DailyLimit > 0 && session.TotalCost >= budget.DailyLimit*0.75 {
		var warningType models.WarningType
		if session.TotalCost >= budget.DailyLimit {
			warningType = models.WarningTypeExceeded
		} else if session.TotalCost >= budget.DailyLimit*0.9 {
			warningType = models.WarningTypeDaily90
		} else {
			warningType = models.WarningTypeDaily75
		}

		warnings = append(warnings, models.BudgetWarning{
			Type:      warningType,
			Threshold: budget.DailyLimit,
			Current:   session.TotalCost,
			Message:   "Daily budget threshold reached",
			Timestamp: time.Now(),
		})
	}

	// Check monthly limit
	if budget.MonthlyLimit > 0 && budget.CurrentMonthly >= budget.MonthlyLimit*0.75 {
		var warningType models.WarningType
		if budget.CurrentMonthly >= budget.MonthlyLimit {
			warningType = models.WarningTypeExceeded
		} else if budget.CurrentMonthly >= budget.MonthlyLimit*0.9 {
			warningType = models.WarningTypeMonthly90
		} else {
			warningType = models.WarningTypeMonthly75
		}

		warnings = append(warnings, models.BudgetWarning{
			Type:      warningType,
			Threshold: budget.MonthlyLimit,
			Current:   budget.CurrentMonthly,
			Message:   "Monthly budget threshold reached",
			Timestamp: time.Now(),
		})
	}

	return budget, warnings
}

// GetTotalUsage returns total usage across all sessions
func (tt *TokenTracker) GetTotalUsage() models.TokenMetrics {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	total := models.TokenMetrics{
		Period:      models.PeriodDaily,
		StartDate:   time.Now(),
		EndDate:     time.Now(),
		ByProvider:  make(map[string]models.MetricValue),
		ByQueryType: make(map[string]models.MetricValue),
		ByAgent:     make(map[string]models.MetricValue),
	}

	for _, session := range tt.sessions {
		total.TotalQueries += session.TotalQueries
		total.TotalTokens += session.TotalInputTokens + session.TotalOutputTokens
		total.TotalCost += session.TotalCost

		// Aggregate provider usage
		for providerName, usage := range session.ProviderUsage {
			metric := total.ByProvider[providerName]
			metric.Count += usage.RequestCount
			metric.Tokens += usage.InputTokens + usage.OutputTokens
			metric.Cost += usage.TotalCost
			total.ByProvider[providerName] = metric
		}
	}

	if total.TotalQueries > 0 {
		total.AverageTokensPerQuery = total.TotalTokens / total.TotalQueries
		total.AverageCostPerQuery = total.TotalCost / float64(total.TotalQueries)
	}

	return total
}

// NewCostCalculator creates a new cost calculator
func NewCostCalculator() *CostCalculator {
	calculator := &CostCalculator{
		pricingCache: make(map[string]models.ModelPricing),
	}

	// Initialize with default pricing
	calculator.initializeDefaultPricing()

	return calculator
}

// initializeDefaultPricing sets up default pricing for known models
func (cc *CostCalculator) initializeDefaultPricing() {
	defaultPricing := []models.ModelPricing{
		{
			Provider:        "openai",
			Model:           "gpt-4-turbo-preview",
			InputCostPer1K:  0.01,
			OutputCostPer1K: 0.03,
			Currency:        "USD",
			LastUpdated:     time.Now(),
			Tier:            "paid",
			RateLimit: models.RateLimit{
				RequestsPerMinute: 500,
				TokensPerMinute:   10000,
			},
		},
		{
			Provider:        "openai",
			Model:           "gpt-3.5-turbo",
			InputCostPer1K:  0.0005,
			OutputCostPer1K: 0.0015,
			Currency:        "USD",
			LastUpdated:     time.Now(),
			Tier:            "paid",
		},
		{
			Provider:        "gemini",
			Model:           "gemini-1.5-pro",
			InputCostPer1K:  0.0035,
			OutputCostPer1K: 0.0105,
			Currency:        "USD",
			LastUpdated:     time.Now(),
			Tier:            "paid",
		},
		{
			Provider:        "cohere",
			Model:           "command-r-plus",
			InputCostPer1K:  0.003,
			OutputCostPer1K: 0.015,
			Currency:        "USD",
			LastUpdated:     time.Now(),
			Tier:            "paid",
		},
		{
			Provider:        "claude",
			Model:           "claude-3-sonnet-20240229",
			InputCostPer1K:  0.003,
			OutputCostPer1K: 0.015,
			Currency:        "USD",
			LastUpdated:     time.Now(),
			Tier:            "paid",
		},
	}

	for _, pricing := range defaultPricing {
		key := pricing.Provider + ":" + pricing.Model
		cc.pricingCache[key] = pricing
	}
}

// CalculateCost calculates the cost of token usage
func (cc *CostCalculator) CalculateCost(usage models.TokenUsage, pricing ProviderPricing) models.Cost {
	inputCost := float64(usage.InputTokens) / 1000.0 * pricing.InputCostPer1K
	outputCost := float64(usage.OutputTokens) / 1000.0 * pricing.OutputCostPer1K
	totalCost := inputCost + outputCost

	return models.Cost{
		InputCost:  inputCost,
		OutputCost: outputCost,
		TotalCost:  totalCost,
		Currency:   "USD",
		Provider:   usage.Provider,
		Model:      usage.Model,
		Timestamp:  time.Now(),
	}
}

// GetPricing returns pricing for a provider/model combination
func (cc *CostCalculator) GetPricing(provider, model string) (models.ModelPricing, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	key := provider + ":" + model
	pricing, exists := cc.pricingCache[key]
	return pricing, exists
}

// UpdatePricing updates pricing information
func (cc *CostCalculator) UpdatePricing(pricing models.ModelPricing) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	key := pricing.Provider + ":" + pricing.Model
	pricing.LastUpdated = time.Now()
	cc.pricingCache[key] = pricing
}

// EstimateCost estimates cost for a given input
func (cc *CostCalculator) EstimateCost(provider, model string, inputTokens, estimatedOutputTokens int) (float64, error) {
	pricing, exists := cc.GetPricing(provider, model)
	if !exists {
		return 0, fmt.Errorf("pricing not found for %s:%s", provider, model)
	}

	inputCost := float64(inputTokens) / 1000.0 * pricing.InputCostPer1K
	outputCost := float64(estimatedOutputTokens) / 1000.0 * pricing.OutputCostPer1K

	return inputCost + outputCost, nil
}

// GenerateOptimizationSuggestions analyzes usage and suggests optimizations
func (cc *CostCalculator) GenerateOptimizationSuggestions(metrics models.TokenMetrics) models.CostOptimization {
	suggestions := make([]models.OptimizationSuggestion, 0)
	potentialSavings := 0.0

	// Analyze provider usage
	if len(metrics.ByProvider) > 1 {
		// Find most expensive provider
		var mostExpensive string
		var highestCost float64
		var cheaperAlternative string
		var cheaperCost float64 = 999999.0

		for provider, metric := range metrics.ByProvider {
			avgCost := metric.Cost / float64(metric.Count)
			if avgCost > highestCost {
				mostExpensive = provider
				highestCost = avgCost
			}
			if avgCost < cheaperCost {
				cheaperAlternative = provider
				cheaperCost = avgCost
			}
		}

		if mostExpensive != cheaperAlternative && highestCost > cheaperCost*1.2 {
			savings := (highestCost - cheaperCost) * float64(metrics.ByProvider[mostExpensive].Count)
			suggestions = append(suggestions, models.OptimizationSuggestion{
				Type:        models.OptimizationTypeProvider,
				Description: fmt.Sprintf("Switch from %s to %s for similar requests", mostExpensive, cheaperAlternative),
				Impact:      "high",
				Savings:     savings,
				Effort:      "low",
			})
			potentialSavings += savings
		}
	}

	// Suggest caching for repeated queries
	if metrics.TotalQueries > 100 {
		cachingSavings := metrics.TotalCost * 0.15 // Assume 15% savings
		suggestions = append(suggestions, models.OptimizationSuggestion{
			Type:        models.OptimizationTypeCaching,
			Description: "Enable response caching for repeated queries",
			Impact:      "medium",
			Savings:     cachingSavings,
			Effort:      "medium",
		})
		potentialSavings += cachingSavings
	}

	// Suggest batching if many small requests
	avgTokens := metrics.TotalTokens / metrics.TotalQueries
	if avgTokens < 100 && metrics.TotalQueries > 50 {
		batchingSavings := metrics.TotalCost * 0.10 // Assume 10% savings
		suggestions = append(suggestions, models.OptimizationSuggestion{
			Type:        models.OptimizationTypeBatching,
			Description: "Batch small requests together to reduce API overhead",
			Impact:      "medium",
			Savings:     batchingSavings,
			Effort:      "high",
		})
		potentialSavings += batchingSavings
	}

	return models.CostOptimization{
		Suggestions:      suggestions,
		PotentialSavings: potentialSavings,
		GeneratedAt:      time.Now(),
	}
}

// NewFallbackHandler creates a new fallback handler
func NewFallbackHandler(maxRetries int) *FallbackHandler {
	return &FallbackHandler{
		maxRetries:      maxRetries,
		retryDelay:      time.Second,
		circuitBreakers: make(map[string]*CircuitBreaker), // Uses CircuitBreaker from llm_types.go
	}
}

// GetCircuitBreaker returns or creates a circuit breaker for a provider
func (fh *FallbackHandler) GetCircuitBreaker(providerName string) *CircuitBreaker {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	cb, exists := fh.circuitBreakers[providerName]
	if !exists {
		// Uses CircuitBreaker struct from llm_types.go
		cb = &CircuitBreaker{
			State:           CircuitBreakerClosed, // Uses CircuitBreakerState from llm_types.go
			FailureCount:    0,
			LastFailureTime: time.Time{},
			NextRetryTime:   time.Time{},
			Threshold:       5,
		}
		fh.circuitBreakers[providerName] = cb
	}

	return cb
}

// ShouldAttempt checks if we should attempt to use a provider
func (fh *FallbackHandler) ShouldAttempt(providerName string) bool {
	cb := fh.GetCircuitBreaker(providerName)
	return cb.ShouldAttempt()
}

// RecordSuccess records a successful request
func (fh *FallbackHandler) RecordSuccess(providerName string) {
	cb := fh.GetCircuitBreaker(providerName)
	cb.RecordSuccess()
}

// RecordFailure records a failed request
func (fh *FallbackHandler) RecordFailure(providerName string) {
	cb := fh.GetCircuitBreaker(providerName)
	cb.RecordFailure()
}

// Circuit Breaker helper methods for the CircuitBreaker type from llm_types.go

// ShouldAttempt checks if the circuit breaker allows attempts
func (cb *CircuitBreaker) ShouldAttempt() bool {
	switch cb.State {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		return time.Since(cb.LastFailureTime) > 60*time.Second // resetTimeout
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.FailureCount = 0
	cb.State = CircuitBreakerClosed
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.FailureCount++
	cb.LastFailureTime = time.Now()

	if cb.FailureCount >= cb.Threshold {
		cb.State = CircuitBreakerOpen
		cb.NextRetryTime = time.Now().Add(60 * time.Second)
	}
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.FailureCount = 0
	cb.State = CircuitBreakerClosed
}
