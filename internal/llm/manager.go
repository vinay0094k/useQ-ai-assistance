package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// Manager manages multiple AI providers with fallback and load balancing
type Manager struct {
	providers       map[string]Provider
	primaryProvider string
	fallbackOrder   []string
	config          ManagerConfig
	stats           map[string]*ProviderStats
	circuitBreakers map[string]*CircuitBreaker
	mu              sync.RWMutex
}

// NewManager creates a new LLM manager
func NewManager(config AIProvidersConfig) (*Manager, error) {
	manager := &Manager{
		providers:       make(map[string]Provider),
		primaryProvider: config.Primary,
		fallbackOrder:   config.FallbackOrder,
		stats:           make(map[string]*ProviderStats),
		circuitBreakers: make(map[string]*CircuitBreaker),
		config: ManagerConfig{
			DefaultTimeout:          30 * time.Second,
			RetryAttempts:           3,
			FallbackEnabled:         true,
			HealthCheckInterval:     5 * time.Minute,
			CircuitBreakerThreshold: 5,
		},
	}

	// Initialize OpenAI provider if configured
	if config.OpenAI.APIKey != "" {
		openaiProvider, err := NewOpenAIProvider(config.OpenAI)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize OpenAI provider: %w", err)
		}
		manager.providers["openai"] = openaiProvider
		manager.initProviderStats("openai")
		manager.initCircuitBreaker("openai")
	}

	// TODO: Initialize other providers when implemented
	// if config.Gemini.APIKey != "" {
	//     geminiProvider, err := providers.NewGeminiProvider(config.Gemini)
	//     if err != nil {
	//         return nil, fmt.Errorf("failed to initialize Gemini provider: %w", err)
	//     }
	//     manager.providers["gemini"] = geminiProvider
	// }

	// Validate that primary provider exists
	if _, exists := manager.providers[manager.primaryProvider]; !exists {
		return nil, fmt.Errorf("primary provider '%s' not available", manager.primaryProvider)
	}

	return manager, nil
}

// Generate generates text using the primary provider with fallback
func (m *Manager) Generate(ctx context.Context, request *GenerationRequest) (*GenerationResponse, error) {
	// Enhance prompt with MCP context if available
	enhancedRequest := m.enhanceRequestWithMCP(request)
	
	// Try primary provider first
	response, err := m.generateWithProvider(ctx, m.primaryProvider, enhancedRequest)
	if err == nil {
		return response, nil
	}

	// Log primary provider failure
	m.recordFailure(m.primaryProvider, err)

	// Try fallback providers if enabled
	if m.config.FallbackEnabled {
		for _, providerName := range m.fallbackOrder {
			if providerName == m.primaryProvider {
				continue // Skip primary (already tried)
			}

			if !m.isProviderAvailable(providerName) {
				continue // Skip unavailable providers
			}

			response, fallbackErr := m.generateWithProvider(ctx, providerName, request)
			if fallbackErr == nil {
				// Success with fallback
				m.recordSuccess(providerName, response)
				return response, nil
			}

			// Record fallback failure
			m.recordFailure(providerName, fallbackErr)
		}
	}

	// All providers failed
	return nil, fmt.Errorf("all providers failed, primary error: %w", err)
}

// generateWithProvider generates text using a specific provider
func (m *Manager) generateWithProvider(ctx context.Context, providerName string, request *GenerationRequest) (*GenerationResponse, error) {
	// Check circuit breaker
	if !m.isCircuitBreakerClosed(providerName) {
		return nil, fmt.Errorf("circuit breaker open for provider: %s", providerName)
	}

	provider, exists := m.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	// Apply timeout if not set
	if request.Timeout == 0 {
		request.Timeout = m.config.DefaultTimeout
	}

	startTime := time.Now()

	// Make the request
	response, err := provider.Generate(ctx, request)
	if err != nil {
		m.updateCircuitBreaker(providerName, false)
		return nil, err
	}

	// Record success
	m.updateCircuitBreaker(providerName, true)
	m.recordSuccess(providerName, response)

	// Update response metadata
	response.Latency = time.Since(startTime)
	response.Timestamp = time.Now()

	return response, nil
}

// Stream starts streaming text generation
func (m *Manager) Stream(ctx context.Context, request *GenerationRequest) (<-chan *StreamChunk, error) {
	// For now, only use primary provider for streaming
	provider, exists := m.providers[m.primaryProvider]
	if !exists {
		return nil, fmt.Errorf("primary provider not available: %s", m.primaryProvider)
	}

	if !m.isCircuitBreakerClosed(m.primaryProvider) {
		return nil, fmt.Errorf("circuit breaker open for provider: %s", m.primaryProvider)
	}

	return provider.Stream(ctx, request)
}

// GetProviderInfo returns information about a specific provider
func (m *Manager) GetProviderInfo(providerName string) (ProviderInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[providerName]
	if !exists {
		return ProviderInfo{}, fmt.Errorf("provider not found: %s", providerName)
	}

	return provider.GetInfo(), nil
}

// GetAllProviders returns information about all available providers
func (m *Manager) GetAllProviders() map[string]ProviderInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := make(map[string]ProviderInfo)
	for name, provider := range m.providers {
		info[name] = provider.GetInfo()
	}

	return info
}

// GetStats returns usage statistics
func (m *Manager) GetStats() UsageMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := UsageMetrics{
		ProviderBreakdown: make(map[string]ProviderStats),
	}

	var totalRequests, totalTokens int64
	var totalCost float64
	var successCount int64

	for name, stats := range m.stats {
		metrics.ProviderBreakdown[name] = *stats
		totalRequests += stats.TotalRequests
		totalTokens += stats.TotalTokens
		totalCost += stats.TotalCost
		successCount += stats.SuccessfulRequests
	}

	metrics.TotalRequests = totalRequests
	metrics.TotalTokens = totalTokens
	metrics.TotalCost = totalCost

	if totalRequests > 0 {
		metrics.SuccessRate = float64(successCount) / float64(totalRequests)
	}

	return metrics
}

// IsHealthy checks if the manager and providers are healthy
func (m *Manager) IsHealthy(ctx context.Context) bool {
	// At least the primary provider must be healthy
	if provider, exists := m.providers[m.primaryProvider]; exists {
		return provider.IsHealthy(ctx)
	}
	return false
}

// Helper methods

// initProviderStats initializes statistics for a provider
func (m *Manager) initProviderStats(providerName string) {
	m.stats[providerName] = &ProviderStats{
		ErrorRate: 0.0,
	}
}

// initCircuitBreaker initializes circuit breaker for a provider
func (m *Manager) initCircuitBreaker(providerName string) {
	m.circuitBreakers[providerName] = &CircuitBreaker{
		State:     CircuitBreakerClosed,
		Threshold: m.config.CircuitBreakerThreshold,
	}
}

// recordSuccess records a successful request
func (m *Manager) recordSuccess(providerName string, response *GenerationResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.stats[providerName]
	stats.TotalRequests++
	stats.SuccessfulRequests++
	stats.TotalTokens += int64(response.TokenUsage.TotalTokens)
	stats.TotalCost += response.Cost.TotalCost
	stats.LastUsed = time.Now()

	// Update error rate
	if stats.TotalRequests > 0 {
		stats.ErrorRate = float64(stats.FailedRequests) / float64(stats.TotalRequests)
	}

	// Update average latency (simple moving average)
	if stats.TotalRequests == 1 {
		stats.AverageLatency = response.Latency
	} else {
		stats.AverageLatency = (stats.AverageLatency + response.Latency) / 2
	}
}

// recordFailure records a failed request
func (m *Manager) recordFailure(providerName string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.stats[providerName]
	stats.TotalRequests++
	stats.FailedRequests++

	// Update error rate
	if stats.TotalRequests > 0 {
		stats.ErrorRate = float64(stats.FailedRequests) / float64(stats.TotalRequests)
	}
}

// isProviderAvailable checks if a provider is available
func (m *Manager) isProviderAvailable(providerName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.providers[providerName]
	return exists && m.isCircuitBreakerClosed(providerName)
}

// isCircuitBreakerClosed checks if circuit breaker is closed
func (m *Manager) isCircuitBreakerClosed(providerName string) bool {
	cb, exists := m.circuitBreakers[providerName]
	if !exists {
		return true
	}

	switch cb.State {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerHalfOpen:
		return time.Now().After(cb.NextRetryTime)
	case CircuitBreakerOpen:
		if time.Now().After(cb.NextRetryTime) {
			cb.State = CircuitBreakerHalfOpen
			return true
		}
		return false
	}

	return false
}

// updateCircuitBreaker updates circuit breaker state
func (m *Manager) updateCircuitBreaker(providerName string, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cb, exists := m.circuitBreakers[providerName]
	if !exists {
		return
	}

	if success {
		// Reset on success
		cb.FailureCount = 0
		cb.State = CircuitBreakerClosed
	} else {
		// Increment failure count
		cb.FailureCount++
		cb.LastFailureTime = time.Now()

		// Open circuit breaker if threshold reached
		if cb.FailureCount >= cb.Threshold {
			cb.State = CircuitBreakerOpen
			cb.NextRetryTime = time.Now().Add(30 * time.Second) // 30s timeout
		}
	}
}

// SetPrimaryProvider changes the primary provider
func (m *Manager) SetPrimaryProvider(providerName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.providers[providerName]; !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	m.primaryProvider = providerName
	return nil
}

// GetPrimaryProvider returns the current primary provider name
func (m *Manager) GetPrimaryProvider() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.primaryProvider
}

// EnhanceRequestWithMCP enhances the generation request with MCP context (public for testing)
func (m *Manager) EnhanceRequestWithMCP(request *GenerationRequest) *GenerationRequest {
	return m.enhanceRequestWithMCP(request)
}

// enhanceRequestWithMCP enhances the generation request with MCP context
func (m *Manager) enhanceRequestWithMCP(request *GenerationRequest) *GenerationRequest {
	if request.MCPContext == nil || !request.MCPContext.RequiresMCP {
		return request
	}
	
	// Create enhanced request
	enhanced := *request
	enhanced.Prompt = m.buildMCPEnhancedPrompt(request.Prompt, request.MCPContext)
	
	return &enhanced
}

// buildMCPEnhancedPrompt builds a prompt enhanced with MCP context
func (m *Manager) buildMCPEnhancedPrompt(originalPrompt string, mcpContext *models.MCPContext) string {
	contextInfo := m.extractMCPContextInfo(mcpContext)
	
	if contextInfo == "" {
		return originalPrompt
	}
	
	return fmt.Sprintf(`PROJECT CONTEXT:
%s

USER REQUEST:
%s`, contextInfo, originalPrompt)
}

// extractMCPContextInfo extracts relevant context information from MCP data
func (m *Manager) extractMCPContextInfo(mcpContext *models.MCPContext) string {
	var info []string
	
	// Add file count
	if count, ok := mcpContext.Data["file_count"].(int); ok {
		info = append(info, fmt.Sprintf("Project has %d files", count))
	}
	
	// Add key files
	if files, ok := mcpContext.Data["project_files"].([]map[string]interface{}); ok {
		filePaths := make([]string, 0, min(3, len(files)))
		for _, file := range files[:min(3, len(files))] {
			if path, ok := file["path"].(string); ok {
				filePaths = append(filePaths, path)
			}
		}
		if len(filePaths) > 0 {
			info = append(info, fmt.Sprintf("Key files: %s", strings.Join(filePaths, ", ")))
		}
	}
	
	// Add project structure
	if structure, ok := mcpContext.Data["project_structure"].(map[string]interface{}); ok {
		patterns := m.extractStructurePatterns(structure)
		if len(patterns) > 0 {
			info = append(info, fmt.Sprintf("Architecture: %s", strings.Join(patterns, ", ")))
		}
	}
	
	return strings.Join(info, "\n")
}

// extractStructurePatterns extracts architectural patterns from project structure
func (m *Manager) extractStructurePatterns(structure map[string]interface{}) []string {
	patterns := []string{}
	
	if _, hasInternal := structure["internal"]; hasInternal {
		patterns = append(patterns, "internal modules")
	}
	if _, hasCmd := structure["cmd"]; hasCmd {
		patterns = append(patterns, "CLI commands")
	}
	if _, hasModels := structure["models"]; hasModels {
		patterns = append(patterns, "data models")
	}
	
	return patterns
}
