package vectordb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// EmbeddingService handles text-to-vector conversion with caching
type EmbeddingService struct {
	config *EmbeddingConfig
	cache  *EmbeddingCache
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(config *EmbeddingConfig) *EmbeddingService {
	// Load API key from environment if not provided
	if config.APIKey == "" {
		config.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	
	// Set defaults
	if config.Endpoint == "" {
		config.Endpoint = "https://api.openai.com/v1/embeddings"
	}
	if config.Model == "" {
		config.Model = "text-embedding-3-small"
	}

	return &EmbeddingService{
		config: config,
		cache:  NewEmbeddingCache(1000), // Cache 1000 embeddings
	}
}

// GenerateEmbedding generates a single embedding
func (es *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := es.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings generated")
	}
	return embeddings[0], nil
}

// GenerateEmbeddings generates multiple embeddings with caching
func (es *EmbeddingService) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Check cache first
	var uncachedTexts []string
	var uncachedIndices []int
	results := make([][]float32, len(texts))

	for i, text := range texts {
		if cached := es.cache.Get(text); cached != nil {
			results[i] = cached
		} else {
			uncachedTexts = append(uncachedTexts, text)
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// Generate embeddings for uncached texts
	if len(uncachedTexts) > 0 {
		if es.config.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key not configured")
		}

		reqBody := EmbeddingRequest{
			Input: uncachedTexts,
			Model: es.config.Model,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", es.config.Endpoint, strings.NewReader(string(jsonData)))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+es.config.APIKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body))
		}

		var embeddingResp EmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
			return nil, err
		}

		// Store results and cache
		for i, idx := range uncachedIndices {
			if i < len(embeddingResp.Data) {
				embedding := embeddingResp.Data[i].Embedding
				results[idx] = embedding
				es.cache.Set(uncachedTexts[i], embedding)
			}
		}
	}

	return results, nil
}

// IsHealthy checks if the embedding service is healthy
func (es *EmbeddingService) IsHealthy(ctx context.Context) bool {
	if es.config.APIKey == "" {
		return false
	}

	// Test with a simple embedding
	_, err := es.GenerateEmbedding(ctx, "test")
	return err == nil
}

// GetCacheStats returns cache statistics
func (es *EmbeddingService) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_size":     es.cache.Size(),
		"cache_max_size": es.cache.maxSize,
	}
}

// NewEmbeddingCache creates a new embedding cache
func NewEmbeddingCache(maxSize int) *EmbeddingCache {
	return &EmbeddingCache{
		cache:   make(map[string][]float32),
		maxSize: maxSize,
	}
}

// Get retrieves an embedding from cache
func (ec *EmbeddingCache) Get(key string) []float32 {
	if embedding, exists := ec.cache[key]; exists {
		return embedding
	}
	return nil
}

// Set stores an embedding in cache
func (ec *EmbeddingCache) Set(key string, embedding []float32) {
	// Simple eviction policy: if cache is full, clear it
	if len(ec.cache) >= ec.maxSize {
		ec.cache = make(map[string][]float32)
	}
	ec.cache[key] = embedding
}

// Size returns the current cache size
func (ec *EmbeddingCache) Size() int {
	return len(ec.cache)
}

// Clear clears the cache
func (ec *EmbeddingCache) Clear() {
	ec.cache = make(map[string][]float32)
}