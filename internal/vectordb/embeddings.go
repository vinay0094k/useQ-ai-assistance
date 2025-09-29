package vectordb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// EmbeddingService - MINIMAL implementation with accurate cost tracking
type EmbeddingService struct {
	apiKey     string
	httpClient *http.Client
	cache      map[string][]float32
	costTracker *CostTracker
}

// CostTracker tracks actual embedding costs
type CostTracker struct {
	TotalTokens int     `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost"`
	RequestCount int    `json:"request_count"`
}

// EmbeddingConfig holds minimal configuration
type EmbeddingConfig struct {
	APIKey   string `json:"api_key"`
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
}

// NewEmbeddingService creates a minimal embedding service
func NewEmbeddingService(config *EmbeddingConfig) *EmbeddingService {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	return &EmbeddingService{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      make(map[string][]float32),
		costTracker: &CostTracker{},
	}
}

// GenerateEmbedding generates a single embedding with cost tracking
func (es *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Check cache first
	if cached, exists := es.cache[text]; exists {
		fmt.Printf("💾 Cache hit for embedding\n")
		return cached, nil
	}

	if es.apiKey == "" {
		fmt.Printf("⚠️ No OpenAI API key, using fallback embedding\n")
		return es.generateFallbackEmbedding(text), nil
	}

	// Estimate cost BEFORE API call
	estimatedTokens := len(text) / 4
	estimatedCost := float64(estimatedTokens) / 1000.0 * 0.0001
	fmt.Printf("💰 Estimated embedding cost: $%.6f (%d tokens)\n", estimatedCost, estimatedTokens)

	reqBody := map[string]interface{}{
		"input": text,
		"model": "text-embedding-3-small",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+es.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body))
	}

	var embeddingResp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, err
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	embedding := embeddingResp.Data[0].Embedding

	// Track actual cost
	actualCost := float64(embeddingResp.Usage.TotalTokens) / 1000.0 * 0.0001
	es.costTracker.TotalTokens += embeddingResp.Usage.TotalTokens
	es.costTracker.TotalCost += actualCost
	es.costTracker.RequestCount++

	fmt.Printf("💰 Actual cost: $%.6f | Total so far: $%.4f (%d requests)\n", 
		actualCost, es.costTracker.TotalCost, es.costTracker.RequestCount)

	// Cache the result
	es.cache[text] = embedding

	return embedding, nil
}

// GetCostStats returns actual cost statistics
func (es *EmbeddingService) GetCostStats() *CostTracker {
	return es.costTracker
}

// generateFallbackEmbedding creates simple hash-based embedding for testing
func (es *EmbeddingService) generateFallbackEmbedding(text string) []float32 {
	words := strings.Fields(strings.ToLower(text))
	embedding := make([]float32, 1536) // Standard size

	for i, word := range words {
		if i >= len(embedding) {
			break
		}
		// Simple hash-based embedding
		hash := 0
		for _, char := range word {
			hash = hash*31 + int(char)
		}
		embedding[i] = float32(hash%1000) / 1000.0
	}

	return embedding
}