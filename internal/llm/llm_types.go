package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// Provider interface that all AI providers must implement
type Provider interface {
	// Generate generates a text completion
	Generate(ctx context.Context, request *GenerationRequest) (*GenerationResponse, error)

	// Stream generates a streaming text completion
	Stream(ctx context.Context, request *GenerationRequest) (<-chan *StreamChunk, error)

	// GetInfo returns provider information
	GetInfo() ProviderInfo

	// IsHealthy checks if the provider is healthy
	IsHealthy(ctx context.Context) bool

	// GetPricing returns current pricing information
	GetPricing() ProviderPricing
}

// GenerationRequest represents a request for text generation
type GenerationRequest struct {
	Messages         []Message         `json:"messages"`
	SystemPrompt     string            `json:"system_prompt,omitempty"`
	Model            string            `json:"model,omitempty"`
	MaxTokens        int               `json:"max_tokens,omitempty"`
	Temperature      float64           `json:"temperature,omitempty"`
	TopP             float64           `json:"top_p,omitempty"`
	Stop             []string          `json:"stop,omitempty"`
	PresencePenalty  float64           `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64           `json:"frequency_penalty,omitempty"`
	Stream           bool              `json:stream,omitempty"`
	Timeout          time.Duration     `json:"timeout,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	Prompt           string            `json:"prompt,omitempty"`
	MCPContext       *models.MCPContext `json:"mcp_context,omitempty"`
}

// GenerationResponse represents a response from text generation
type GenerationResponse struct {
	Content      string                 `json:"content"`
	FinishReason string                 `json:"finish_reason"`
	TokenUsage   models.TokenUsage      `json:"token_usage"`
	Cost         models.Cost            `json:"cost"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	Latency      time.Duration          `json:"latency"`
	Metadata     map[string]interface{} `json:"metadata"`
	Timestamp    time.Time              `json:"timestamp"`
}

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Content      string    `json:"content"`
	Delta        string    `json:"delta"`
	FinishReason string    `json:"finish_reason,omitempty"`
	TokenCount   int       `json:"token_count"`
	Done         bool      `json:"done"`
	Error        error     `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	APIKey      string        `json:"api_key" yaml:"api_key"`
	Model       string        `json:"model" yaml:"model"`
	MaxTokens   int           `json:"max_tokens" yaml:"max_tokens"`
	Temperature float64       `json:"temperature" yaml:"temperature"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
	CostPer1K   CostConfig    `json:"cost_per_1k" yaml:"cost_per_1k"`
}

// CostConfig holds cost information per 1K tokens
type CostConfig struct {
	Input  float64 `json:"input" yaml:"input"`
	Output float64 `json:"output" yaml:"output"`
}

// ProviderInfo holds information about a provider
type ProviderInfo struct {
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Models       []string        `json:"models"`
	MaxTokens    int             `json:"max_tokens"`
	Capabilities []string        `json:"capabilities"`
	Pricing      ProviderPricing `json:"pricing"`
	Status       ProviderStatus  `json:"status"`
}

// ProviderPricing holds pricing information
type ProviderPricing struct {
	InputCostPer1K  float64   `json:"input_cost_per_1k"`
	OutputCostPer1K float64   `json:"output_cost_per_1k"`
	Currency        string    `json:"currency"`
	Model           string    `json:"model"`
	LastUpdated     time.Time `json:"last_updated"`
}

// ProviderStatus holds status information
type ProviderStatus struct {
	Available    bool          `json:"available"`
	LastChecked  time.Time     `json:"last_checked"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorRate    float64       `json:"error_rate"`
	RequestCount int64         `json:"request_count"`
	SuccessCount int64         `json:"success_count"`
	LastError    string        `json:"last_error,omitempty"`
	Health       string        `json:"health"` // "healthy", "degraded", "unhealthy"
}

// AIProvidersConfig holds configuration for all AI providers
type AIProvidersConfig struct {
	Primary       string         `json:"primary" yaml:"primary"`
	FallbackOrder []string       `json:"fallback_order" yaml:"fallback_order"`
	OpenAI        ProviderConfig `json:"openai" yaml:"openai"`
	Gemini        ProviderConfig `json:"gemini" yaml:"gemini"`
	Cohere        ProviderConfig `json:"cohere" yaml:"cohere"`
	Claude        ProviderConfig `json:"claude" yaml:"claude"`
}

// ManagerConfig holds configuration for the LLM manager
type ManagerConfig struct {
	DefaultTimeout          time.Duration `json:"default_timeout" yaml:"default_timeout"`
	RetryAttempts           int           `json:"retry_attempts" yaml:"retry_attempts"`
	FallbackEnabled         bool          `json:"fallback_enabled" yaml:"fallback_enabled"`
	HealthCheckInterval     time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
	CircuitBreakerThreshold int           `json:"circuit_breaker_threshold" yaml:"circuit_breaker_threshold"`
}

// ProviderStats holds statistics for a provider
type ProviderStats struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	TotalCost          float64       `json:"total_cost"`
	TotalTokens        int64         `json:"total_tokens"`
	LastUsed           time.Time     `json:"last_used"`
	ErrorRate          float64       `json:"error_rate"`
}

// RequestContext holds context for a request
type RequestContext struct {
	RequestID  string            `json:"request_id"`
	SessionID  string            `json:"session_id"`
	UserID     string            `json:"user_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Priority   Priority          `json:"priority"`
	MaxRetries int               `json:"max_retries"`
	StartTime  time.Time         `json:"start_time"`
}

// Priority represents request priority
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

// ProviderType represents different provider types
type ProviderType string

const (
	ProviderTypeOpenAI ProviderType = "openai"
	ProviderTypeGemini ProviderType = "gemini"
	ProviderTypeCohere ProviderType = "cohere"
	ProviderTypeClaude ProviderType = "claude"
)

// ModelCapability represents different model capabilities
type ModelCapability string

const (
	CapabilityChatCompletion  ModelCapability = "chat_completion"
	CapabilityStreaming       ModelCapability = "streaming"
	CapabilityFunctionCalling ModelCapability = "function_calling"
	CapabilityVision          ModelCapability = "vision"
	CapabilityCodeGeneration  ModelCapability = "code_generation"
	CapabilityEmbeddings      ModelCapability = "embeddings"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeAPI            ErrorType = "api_error"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeQuotaExceeded  ErrorType = "quota_exceeded"
	ErrorTypeInvalidRequest ErrorType = "invalid_request"
	ErrorTypeNetworkError   ErrorType = "network_error"
	ErrorTypeProviderError  ErrorType = "provider_error"
)

// ProviderError represents an error from a provider
type ProviderError struct {
	Type       ErrorType              `json:"type"`
	Provider   string                 `json:"provider"`
	Model      string                 `json:"model,omitempty"`
	Message    string                 `json:"message"`
	Code       string                 `json:"code,omitempty"`
	StatusCode int                    `json:"status_code,omitempty"`
	RetryAfter time.Duration          `json:"retry_after,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s error from %s: %s", e.Type, e.Provider, e.Message)
}

// UsageMetrics holds usage metrics
type UsageMetrics struct {
	TotalRequests     int64                    `json:"total_requests"`
	TotalTokens       int64                    `json:"total_tokens"`
	TotalCost         float64                  `json:"total_cost"`
	AverageLatency    time.Duration            `json:"average_latency"`
	SuccessRate       float64                  `json:"success_rate"`
	LastRequest       time.Time                `json:"last_request"`
	ProviderBreakdown map[string]ProviderStats `json:"provider_breakdown"`
}

// FallbackResult represents the result of a fallback attempt
type FallbackResult struct {
	Provider string                 `json:"provider"`
	Success  bool                   `json:"success"`
	Error    error                  `json:"error,omitempty"`
	Attempt  int                    `json:"attempt"`
	Duration time.Duration          `json:"duration"`
	Response *GenerationResponse    `json:"response,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LoadBalancingStrategy represents different load balancing strategies
type LoadBalancingStrategy string

const (
	StrategyRoundRobin     LoadBalancingStrategy = "round_robin"
	StrategyWeightedRandom LoadBalancingStrategy = "weighted_random"
	StrategyLeastLatency   LoadBalancingStrategy = "least_latency"
	StrategyLeastCost      LoadBalancingStrategy = "least_cost"
	StrategyHealthBased    LoadBalancingStrategy = "health_based"
)

// ProviderWeight represents weight for load balancing
type ProviderWeight struct {
	Provider string  `json:"provider"`
	Weight   float64 `json:"weight"`
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"
	CircuitBreakerOpen     CircuitBreakerState = "open"
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open"
)

// CircuitBreaker represents a circuit breaker for a provider
type CircuitBreaker struct {
	State           CircuitBreakerState `json:"state"`
	FailureCount    int                 `json:"failure_count"`
	LastFailureTime time.Time           `json:"last_failure_time"`
	NextRetryTime   time.Time           `json:"next_retry_time"`
	Threshold       int                 `json:"threshold"`
}

// CodeGenerationRequest represents a request for code generation
type CodeGenerationRequest struct {
	Prompt      string `json:"prompt"`
	Language    string `json:"language"`
	Context     string `json:"context"`
	MaxTokens   int    `json:"max_tokens"`
}

// CodeGenerationResponse represents the response from code generation
type CodeGenerationResponse struct {
	Code        string `json:"code"`
	Explanation string `json:"explanation"`
	TokensUsed  int    `json:"tokens_used"`
}
