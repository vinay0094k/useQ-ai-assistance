package vectordb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// QdrantClient - MINIMAL implementation focused on core functionality
type QdrantClient struct {
	httpClient     *http.Client
	config         *QdrantConfig
	embeddingCache map[string][]float32 // Simple in-memory cache
}

// QdrantConfig - simplified configuration
type QdrantConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Collection string `json:"collection"`
	VectorSize int    `json:"vector_size"`
}

// CodeChunk - minimal structure for vector storage
type CodeChunk struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	FilePath  string `json:"file_path"`
	Language  string `json:"language"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// SearchResult - minimal search result
type SearchResult struct {
	Chunk *CodeChunk `json:"chunk"`
	Score float32    `json:"score"`
}

// NewQdrantClient creates a minimal Qdrant client
func NewQdrantClient(config *QdrantConfig) (*QdrantClient, error) {
	qc := &QdrantClient{
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		config:         config,
		embeddingCache: make(map[string][]float32),
	}

	// Test connection
	if err := qc.testConnection(); err != nil {
		return nil, fmt.Errorf("Qdrant connection failed: %w", err)
	}

	// Ensure collection exists
	if err := qc.ensureCollection(); err != nil {
		return nil, fmt.Errorf("collection setup failed: %w", err)
	}

	fmt.Printf("✅ Qdrant connected: %s:%d\n", config.Host, config.Port)
	return qc, nil
}

// Search performs semantic search - CORE FUNCTIONALITY
func (qc *QdrantClient) Search(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	// Generate embedding for query
	embedding, err := qc.generateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embedding generation failed: %w", err)
	}

	// Search vectors
	return qc.searchVectors(ctx, embedding, limit)
}

// StoreChunkWithEmbedding stores code chunk with embedding
func (qc *QdrantClient) StoreChunkWithEmbedding(ctx context.Context, chunk *CodeChunk, embedding []float32) error {
	// Generate numeric ID from string ID
	hash := fnv.New32a()
	hash.Write([]byte(chunk.ID))
	numericID := hash.Sum32()

	point := map[string]interface{}{
		"id":     numericID,
		"vector": embedding,
		"payload": map[string]interface{}{
			"original_id": chunk.ID,
			"file":        chunk.FilePath,
			"content":     chunk.Content,
			"language":    chunk.Language,
			"start_line":  chunk.StartLine,
			"end_line":    chunk.EndLine,
		},
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"points": []interface{}{point},
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s:%d/collections/%s/points", qc.config.Host, qc.config.Port, qc.config.Collection)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := qc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("store failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GenerateOpenAIEmbedding generates OpenAI embeddings with cost tracking
func (qc *QdrantClient) GenerateOpenAIEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Check cache first
	if cached, exists := qc.embeddingCache[text]; exists {
		return cached, nil
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return qc.generateFallbackEmbedding(text), nil
	}

	// Calculate cost BEFORE making request
	estimatedTokens := len(text) / 4 // ~4 chars per token
	estimatedCost := float64(estimatedTokens) / 1000.0 * 0.0001 // $0.0001 per 1K tokens
	
	fmt.Printf("💰 Embedding cost: ~$%.6f (%d tokens)\n", estimatedCost, estimatedTokens)

	reqBody := map[string]interface{}{
		"input": text,
		"model": "text-embedding-3-small",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := qc.httpClient.Do(req)
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
	
	// Calculate actual cost
	actualCost := float64(embeddingResp.Usage.TotalTokens) / 1000.0 * 0.0001
	fmt.Printf("💰 Actual embedding cost: $%.6f (%d tokens)\n", actualCost, embeddingResp.Usage.TotalTokens)

	// Cache the result
	qc.embeddingCache[text] = embedding

	return embedding, nil
}

// Health checks if Qdrant is accessible
func (qc *QdrantClient) Health(ctx context.Context) error {
	return qc.testConnection()
}

// Close cleans up resources
func (qc *QdrantClient) Close() error {
	// Clear cache
	qc.embeddingCache = nil
	return nil
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

func (qc *QdrantClient) testConnection() error {
	url := fmt.Sprintf("http://%s:%d/collections", qc.config.Host, qc.config.Port)
	resp, err := qc.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (qc *QdrantClient) ensureCollection() error {
	// Check if collection exists
	url := fmt.Sprintf("http://%s:%d/collections/%s", qc.config.Host, qc.config.Port, qc.config.Collection)
	resp, err := qc.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil // Collection exists
	}

	// Create collection
	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     qc.config.VectorSize,
			"distance": "Cosine",
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = qc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to create collection")
	}

	fmt.Printf("✅ Created collection: %s\n", qc.config.Collection)
	return nil
}

func (qc *QdrantClient) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Try OpenAI first
	if os.Getenv("OPENAI_API_KEY") != "" {
		return qc.GenerateOpenAIEmbedding(ctx, text)
	}

	// Fallback to simple embedding for testing
	return qc.generateFallbackEmbedding(text), nil
}

func (qc *QdrantClient) generateFallbackEmbedding(text string) []float32 {
	words := strings.Fields(strings.ToLower(text))
	embedding := make([]float32, qc.config.VectorSize)

	for i, word := range words {
		if i >= len(embedding) {
			break
		}
		hash := fnv.New32a()
		hash.Write([]byte(word))
		embedding[i] = float32(hash.Sum32()%1000) / 1000.0
	}

	return embedding
}

func (qc *QdrantClient) searchVectors(ctx context.Context, embedding []float32, limit int) ([]*SearchResult, error) {
	searchReq := map[string]interface{}{
		"vector":       embedding,
		"limit":        limit,
		"with_payload": true,
	}

	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://%s:%d/collections/%s/points/search", qc.config.Host, qc.config.Port, qc.config.Collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := qc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search failed with status %d", resp.StatusCode)
	}

	var searchResp struct {
		Result []struct {
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	results := make([]*SearchResult, 0, len(searchResp.Result))
	for _, hit := range searchResp.Result {
		chunk := &CodeChunk{}

		if file, ok := hit.Payload["file"].(string); ok {
			chunk.FilePath = file
		}
		if content, ok := hit.Payload["content"].(string); ok {
			chunk.Content = content
		}
		if language, ok := hit.Payload["language"].(string); ok {
			chunk.Language = language
		}
		if originalID, ok := hit.Payload["original_id"].(string); ok {
			chunk.ID = originalID
		}
		if startLine, ok := hit.Payload["start_line"].(float64); ok {
			chunk.StartLine = int(startLine)
		}
		if endLine, ok := hit.Payload["end_line"].(float64); ok {
			chunk.EndLine = int(endLine)
		}

		results = append(results, &SearchResult{
			Score: float32(hit.Score),
			Chunk: chunk,
		})
	}

	return results, nil
}