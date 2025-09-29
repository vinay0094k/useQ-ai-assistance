// Why this file: ./models/token_model.go
// This implements comprehensive token usage and cost tracking - a key requirement you mentioned.
// It tracks costs per provider, session budgets, optimization suggestions, and detailed metrics for financial control and transparency.
package models

import (
	"time"
)

// TokenUsage represents token consumption for a request
type TokenUsage struct {
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	TotalTokens     int       `json:"total_tokens"`
	CachedTokens    int       `json:"cached_tokens,omitempty"`
	ReasoningTokens int       `json:"reasoning_tokens,omitempty"`
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	Timestamp       time.Time `json:"timestamp"`
}

// Cost represents the financial cost of a request
type Cost struct {
	InputCost  float64   `json:"input_cost"`
	OutputCost float64   `json:"output_cost"`
	TotalCost  float64   `json:"total_cost"`
	Currency   string    `json:"currency"`
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	Timestamp  time.Time `json:"timestamp"`
}

// SessionTokens tracks token usage for an entire session
type SessionTokens struct {
	SessionID         string                   `json:"session_id"`
	StartTime         time.Time                `json:"start_time"`
	EndTime           *time.Time               `json:"end_time,omitempty"`
	TotalQueries      int                      `json:"total_queries"`
	TotalInputTokens  int                      `json:"total_input_tokens"`
	TotalOutputTokens int                      `json:"total_output_tokens"`
	TotalCost         float64                  `json:"total_cost"`
	ProviderUsage     map[string]ProviderUsage `json:"provider_usage"`
	QueryBreakdown    []QueryTokens            `json:"query_breakdown"`
}

// ProviderUsage tracks usage per AI provider
type ProviderUsage struct {
	Provider       string        `json:"provider"`
	RequestCount   int           `json:"request_count"`
	InputTokens    int           `json:"input_tokens"`
	OutputTokens   int           `json:"output_tokens"`
	TotalCost      float64       `json:"total_cost"`
	AverageLatency time.Duration `json:"average_latency"`
	SuccessRate    float64       `json:"success_rate"`
	LastUsed       time.Time     `json:"last_used"`
}

// QueryTokens represents token usage for a specific query
type QueryTokens struct {
	QueryID    string        `json:"query_id"`
	QueryType  QueryType     `json:"query_type"`
	TokenUsage TokenUsage    `json:"token_usage"`
	Cost       Cost          `json:"cost"`
	Duration   time.Duration `json:"duration"`
	Success    bool          `json:"success"`
	Provider   string        `json:"provider"`
	Agent      string        `json:"agent"`
}

// TokenBudget represents spending limits and tracking
type TokenBudget struct {
	DailyLimit       float64         `json:"daily_limit"`
	MonthlyLimit     float64         `json:"monthly_limit"`
	CurrentDaily     float64         `json:"current_daily"`
	CurrentMonthly   float64         `json:"current_monthly"`
	LastResetDaily   time.Time       `json:"last_reset_daily"`
	LastResetMonthly time.Time       `json:"last_reset_monthly"`
	Warnings         []BudgetWarning `json:"warnings,omitempty"`
}

// BudgetWarning represents budget threshold warnings
type BudgetWarning struct {
	Type      WarningType `json:"type"`
	Threshold float64     `json:"threshold"`
	Current   float64     `json:"current"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
}

// WarningType defines types of budget warnings
type WarningType string

const (
	WarningTypeDaily75   WarningType = "daily_75_percent"
	WarningTypeDaily90   WarningType = "daily_90_percent"
	WarningTypeMonthly75 WarningType = "monthly_75_percent"
	WarningTypeMonthly90 WarningType = "monthly_90_percent"
	WarningTypeExceeded  WarningType = "limit_exceeded"
)

// ModelPricing represents pricing information for different models
type ModelPricing struct {
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	InputCostPer1K  float64   `json:"input_cost_per_1k"`
	OutputCostPer1K float64   `json:"output_cost_per_1k"`
	Currency        string    `json:"currency"`
	LastUpdated     time.Time `json:"last_updated"`
	Tier            string    `json:"tier"` // free, paid, premium
	RateLimit       RateLimit `json:"rate_limit"`
}

// RateLimit represents API rate limiting information
type RateLimit struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	RequestsPerDay    int `json:"requests_per_day,omitempty"`
}

// TokenMetrics provides aggregated token usage statistics
type TokenMetrics struct {
	Period                Period    `json:"period"`
	StartDate             time.Time `json:"start_date"`
	EndDate               time.Time `json:"end_date"`
	TotalQueries          int       `json:"total_queries"`
	TotalTokens           int       `json:"total_tokens"`
	TotalCost             float64   `json:"total_cost"`
	AverageTokensPerQuery int       `json:"average_tokens_per_query"`
	AverageCostPerQuery   float64   `json:"average_cost_per_query"`

	// Breakdown by type
	ByProvider  map[string]MetricValue `json:"by_provider"`
	ByQueryType map[string]MetricValue `json:"by_query_type"`
	ByAgent     map[string]MetricValue `json:"by_agent"`

	// Trends
	DailyBreakdown []DailyMetric `json:"daily_breakdown,omitempty"`
	HourlyPattern  []int         `json:"hourly_pattern,omitempty"`
}

// Period represents time periods for metrics
type Period string

const (
	PeriodHourly  Period = "hourly"
	PeriodDaily   Period = "daily"
	PeriodWeekly  Period = "weekly"
	PeriodMonthly Period = "monthly"
)

// MetricValue holds aggregated metric values
type MetricValue struct {
	Count      int     `json:"count"`
	Tokens     int     `json:"tokens"`
	Cost       float64 `json:"cost"`
	Percentage float64 `json:"percentage"`
}

// DailyMetric represents metrics for a specific day
type DailyMetric struct {
	Date    time.Time `json:"date"`
	Queries int       `json:"queries"`
	Tokens  int       `json:"tokens"`
	Cost    float64   `json:"cost"`
}

// CostOptimization provides cost optimization suggestions
type CostOptimization struct {
	Suggestions      []OptimizationSuggestion `json:"suggestions"`
	PotentialSavings float64                  `json:"potential_savings"`
	GeneratedAt      time.Time                `json:"generated_at"`
}

// OptimizationSuggestion represents a cost optimization suggestion
type OptimizationSuggestion struct {
	Type        OptimizationType `json:"type"`
	Description string           `json:"description"`
	Impact      string           `json:"impact"` // high, medium, low
	Savings     float64          `json:"estimated_savings"`
	Effort      string           `json:"effort"` // high, medium, low
}

// OptimizationType defines types of cost optimizations
type OptimizationType string

const (
	OptimizationTypeProvider  OptimizationType = "switch_provider"
	OptimizationTypeModel     OptimizationType = "switch_model"
	OptimizationTypeCaching   OptimizationType = "enable_caching"
	OptimizationTypeBatching  OptimizationType = "batch_requests"
	OptimizationTypeFiltering OptimizationType = "optimize_context"
)
