package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/yourusername/useq-ai-assistant/models"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	client  *openai.Client
	config  OpenAIConfig
	info    ProviderInfo
	pricing ProviderPricing
}

// OpenAIConfig holds OpenAI-specific configuration
type OpenAIConfig struct {
	APIKey           string        `json:"api_key"`
	Model            string        `json:"model"`
	MaxTokens        int           `json:"max_tokens"`
	Temperature      float32       `json:"temperature"`
	TopP             float32       `json:"top_p"`
	PresencePenalty  float32       `json:"presence_penalty"`
	FrequencyPenalty float32       `json:"frequency_penalty"`
	Timeout          time.Duration `json:"timeout"`
	BaseURL          string        `json:"base_url,omitempty"`
	OrgID            string        `json:"org_id,omitempty"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config ProviderConfig) (Provider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not provided")
	}

	// Set defaults
	if config.Model == "" {
		config.Model = "gpt-4-turbo-preview"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4000
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	openaiConfig := OpenAIConfig{
		APIKey:           apiKey,
		Model:            config.Model,
		MaxTokens:        config.MaxTokens,
		Temperature:      float32(config.Temperature),
		TopP:             1.0,
		PresencePenalty:  0.0,
		FrequencyPenalty: 0.0,
		Timeout:          config.Timeout,
	}

	// Create OpenAI client configuration
	clientConfig := openai.DefaultConfig(apiKey)

	// Set custom base URL if provided
	if openaiConfig.BaseURL != "" {
		clientConfig.BaseURL = openaiConfig.BaseURL
	}

	// Set organization ID if provided
	if openaiConfig.OrgID != "" {
		clientConfig.OrgID = openaiConfig.OrgID
	}

	client := openai.NewClientWithConfig(clientConfig)

	provider := &OpenAIProvider{
		client: client,
		config: openaiConfig,
		pricing: ProviderPricing{
			InputCostPer1K:  getPricing(config.Model, true),
			OutputCostPer1K: getPricing(config.Model, false),
			Currency:        "USD",
			Model:           config.Model,
			LastUpdated:     time.Now(),
		},
	}

	// Initialize provider info
	provider.initProviderInfo()

	return provider, nil
}

// Generate generates text completion
func (p *OpenAIProvider) Generate(ctx context.Context, request *GenerationRequest) (*GenerationResponse, error) {
	startTime := time.Now()

	// Apply timeout
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}

	// Convert messages
	messages := p.convertMessages(request.Messages)

	// Add system prompt if provided
	if request.SystemPrompt != "" {
		systemMsg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: request.SystemPrompt,
		}
		messages = append([]openai.ChatCompletionMessage{systemMsg}, messages...)
	}

	// Create OpenAI request
	openaiRequest := openai.ChatCompletionRequest{
		Model:            p.getModel(request.Model),
		Messages:         messages,
		MaxTokens:        p.getMaxTokens(request.MaxTokens),
		Temperature:      p.getTemperature(request.Temperature),
		TopP:             p.getTopP(request.TopP),
		Stop:             request.Stop,
		PresencePenalty:  p.getPresencePenalty(request.PresencePenalty),
		FrequencyPenalty: p.getFrequencyPenalty(request.FrequencyPenalty),
		Stream:           false,
	}

	// Call OpenAI API
	response, err := p.client.CreateChatCompletion(ctx, openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	// Extract response
	choice := response.Choices[0]
	content := choice.Message.Content
	finishReason := string(choice.FinishReason)

	// Create token usage
	tokenUsage := models.TokenUsage{
		InputTokens:  response.Usage.PromptTokens,
		OutputTokens: response.Usage.CompletionTokens,
		TotalTokens:  response.Usage.TotalTokens,
		Provider:     "openai",
		Model:        response.Model,
		Timestamp:    time.Now(),
	}

	// Calculate cost
	cost := p.calculateCost(tokenUsage)

	return &GenerationResponse{
		Content:      content,
		FinishReason: finishReason,
		TokenUsage:   tokenUsage,
		Cost:         cost,
		Model:        response.Model,
		Provider:     "openai",
		Latency:      time.Since(startTime),
		Timestamp:    time.Now(),
		Metadata: map[string]interface{}{
			"openai_id":          response.ID,
			"created":            response.Created,
			"system_fingerprint": response.SystemFingerprint,
		},
	}, nil
}

// Stream generates streaming text completion
func (p *OpenAIProvider) Stream(ctx context.Context, request *GenerationRequest) (<-chan *StreamChunk, error) {
	// Apply timeout
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}

	// Convert messages
	messages := p.convertMessages(request.Messages)

	// Add system prompt if provided
	if request.SystemPrompt != "" {
		systemMsg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: request.SystemPrompt,
		}
		messages = append([]openai.ChatCompletionMessage{systemMsg}, messages...)
	}

	// Create OpenAI request with streaming
	openaiRequest := openai.ChatCompletionRequest{
		Model:            p.getModel(request.Model),
		Messages:         messages,
		MaxTokens:        p.getMaxTokens(request.MaxTokens),
		Temperature:      p.getTemperature(request.Temperature),
		TopP:             p.getTopP(request.TopP),
		Stop:             request.Stop,
		PresencePenalty:  p.getPresencePenalty(request.PresencePenalty),
		FrequencyPenalty: p.getFrequencyPenalty(request.FrequencyPenalty),
		Stream:           true,
	}

	// Create stream
	stream, err := p.client.CreateChatCompletionStream(ctx, openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI stream: %w", err)
	}

	// Create output channel
	chunks := make(chan *StreamChunk, 10)

	// Start streaming goroutine
	go p.handleStream(ctx, stream, chunks)

	return chunks, nil
}

// handleStream handles the streaming response
func (p *OpenAIProvider) handleStream(ctx context.Context, stream *openai.ChatCompletionStream, chunks chan<- *StreamChunk) {
	defer close(chunks)
	defer stream.Close()

	var fullContent strings.Builder
	tokenCount := 0

	for {
		select {
		case <-ctx.Done():
			chunks <- &StreamChunk{
				Error:     ctx.Err(),
				Done:      true,
				Timestamp: time.Now(),
			}
			return
		default:
			response, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					// End of stream
					chunks <- &StreamChunk{
						Content:    fullContent.String(),
						Delta:      "",
						TokenCount: tokenCount,
						Done:       true,
						Timestamp:  time.Now(),
					}
					return
				}

				chunks <- &StreamChunk{
					Error:     fmt.Errorf("stream error: %w", err),
					Done:      true,
					Timestamp: time.Now(),
				}
				return
			}

			if len(response.Choices) > 0 {
				choice := response.Choices[0]
				delta := choice.Delta.Content

				if delta != "" {
					fullContent.WriteString(delta)
					tokenCount++ // Rough approximation

					chunks <- &StreamChunk{
						Content:      fullContent.String(),
						Delta:        delta,
						FinishReason: string(choice.FinishReason),
						TokenCount:   tokenCount,
						Done:         false,
						Timestamp:    time.Now(),
					}
				}

				if choice.FinishReason != "" {
					chunks <- &StreamChunk{
						Content:      fullContent.String(),
						Delta:        "",
						FinishReason: string(choice.FinishReason),
						TokenCount:   tokenCount,
						Done:         true,
						Timestamp:    time.Now(),
					}
					return
				}
			}
		}
	}
}

// GetInfo returns provider information
func (p *OpenAIProvider) GetInfo() ProviderInfo {
	return p.info
}

// IsHealthy checks if the provider is healthy
func (p *OpenAIProvider) IsHealthy(ctx context.Context) bool {
	// Simple health check - try to list models
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := p.client.ListModels(ctx)
	return err == nil
}

// GetPricing returns current pricing information
func (p *OpenAIProvider) GetPricing() ProviderPricing {
	return p.pricing
}

// Helper methods

// convertMessages converts generic messages to OpenAI format
func (p *OpenAIProvider) convertMessages(messages []Message) []openai.ChatCompletionMessage {
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))

	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    p.convertRole(msg.Role),
			Content: msg.Content,
		}
	}

	return openaiMessages
}

// convertRole converts generic role to OpenAI role
func (p *OpenAIProvider) convertRole(role string) string {
	switch strings.ToLower(role) {
	case "system":
		return openai.ChatMessageRoleSystem
	case "user":
		return openai.ChatMessageRoleUser
	case "assistant":
		return openai.ChatMessageRoleAssistant
	default:
		return openai.ChatMessageRoleUser
	}
}

// getModel returns the model to use
func (p *OpenAIProvider) getModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return p.config.Model
}

// getMaxTokens returns max tokens to use
func (p *OpenAIProvider) getMaxTokens(requestMaxTokens int) int {
	if requestMaxTokens > 0 {
		return requestMaxTokens
	}
	return p.config.MaxTokens
}

// getTemperature returns temperature to use
func (p *OpenAIProvider) getTemperature(requestTemperature float64) float32 {
	if requestTemperature > 0 {
		return float32(requestTemperature)
	}
	return p.config.Temperature
}

// getTopP returns top_p to use
func (p *OpenAIProvider) getTopP(requestTopP float64) float32 {
	if requestTopP > 0 {
		return float32(requestTopP)
	}
	return p.config.TopP
}

// getPresencePenalty returns presence penalty to use
func (p *OpenAIProvider) getPresencePenalty(requestPenalty float64) float32 {
	if requestPenalty != 0 {
		return float32(requestPenalty)
	}
	return p.config.PresencePenalty
}

// getFrequencyPenalty returns frequency penalty to use
func (p *OpenAIProvider) getFrequencyPenalty(requestPenalty float64) float32 {
	if requestPenalty != 0 {
		return float32(requestPenalty)
	}
	return p.config.FrequencyPenalty
}

// calculateCost calculates the cost of token usage
func (p *OpenAIProvider) calculateCost(usage models.TokenUsage) models.Cost {
	inputCost := float64(usage.InputTokens) / 1000.0 * p.pricing.InputCostPer1K
	outputCost := float64(usage.OutputTokens) / 1000.0 * p.pricing.OutputCostPer1K
	totalCost := inputCost + outputCost

	return models.Cost{
		InputCost:  inputCost,
		OutputCost: outputCost,
		TotalCost:  totalCost,
		Currency:   p.pricing.Currency,
		Provider:   "openai",
		Model:      usage.Model,
		Timestamp:  time.Now(),
	}
}

// initProviderInfo initializes provider information
func (p *OpenAIProvider) initProviderInfo() {
	p.info = ProviderInfo{
		Name:    "OpenAI",
		Version: "1.0.0",
		Models: []string{
			"gpt-4-turbo-preview",
			"gpt-4",
			"gpt-4-32k",
			"gpt-3.5-turbo",
			"gpt-3.5-turbo-16k",
		},
		MaxTokens: getMaxTokensForModel(p.config.Model),
		Capabilities: []string{
			"chat_completion",
			"streaming",
			"function_calling",
			"vision",
		},
		Pricing: p.pricing,
		Status: ProviderStatus{
			Available:    true,
			LastChecked:  time.Now(),
			ResponseTime: 0,
			ErrorRate:    0.0,
			RequestCount: 0,
			SuccessCount: 0,
			Health:       "healthy",
		},
	}
}

// getPricing returns pricing for different models
func getPricing(model string, input bool) float64 {
	pricing := map[string][2]float64{
		"gpt-4-turbo-preview":    {0.01, 0.03},
		"gpt-4":                  {0.03, 0.06},
		"gpt-4-32k":              {0.06, 0.12},
		"gpt-3.5-turbo":          {0.0005, 0.0015},
		"gpt-3.5-turbo-16k":      {0.001, 0.002},
		"gpt-3.5-turbo-instruct": {0.0015, 0.002},
	}

	costs, exists := pricing[model]
	if !exists {
		// Default to GPT-4 pricing
		costs = pricing["gpt-4-turbo-preview"]
	}

	if input {
		return costs[0]
	}
	return costs[1]
}

// getMaxTokensForModel returns max tokens for different models
func getMaxTokensForModel(model string) int {
	maxTokens := map[string]int{
		"gpt-4-turbo-preview": 128000,
		"gpt-4":               8192,
		"gpt-4-32k":           32768,
		"gpt-3.5-turbo":       4096,
		"gpt-3.5-turbo-16k":   16384,
	}

	max, exists := maxTokens[model]
	if !exists {
		return 4096 // Default
	}
	return max
}
