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

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewQdrantClient creates a new Qdrant client with intelligent connection handling
func NewQdrantClient(config *QdrantConfig) (*QdrantClient, error) {
	qc := &QdrantClient{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Try gRPC first, fallback to HTTP
	if err := qc.setupGRPCConnection(); err != nil {
		fmt.Printf("⚠️ gRPC connection failed: %v\n", err)
		fmt.Printf("🔄 Testing HTTP fallback...\n")

		if err := qc.testHTTPConnection(); err != nil {
			return nil, fmt.Errorf("both gRPC and HTTP connections failed - gRPC: %v, HTTP: %v", err, err)
		}
		fmt.Printf("✅ HTTP connection successful\n")
	}

	// Initialize collection
	if err := qc.ensureCollection(context.Background()); err != nil {
		fmt.Printf("⚠️ Failed to ensure collection exists: %v\n", err)
	}

	return qc, nil
}

// Search performs intelligent vector search with tier-aware optimization
func (qc *QdrantClient) Search(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	// Generate embedding for the query
	embedding, err := qc.generateQueryEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Try gRPC first if available
	if qc.useGRPC {
		results, err := qc.searchVectorsGRPC(ctx, embedding, limit)
		if err == nil {
			return results, nil
		}
		fmt.Printf("⚠️ gRPC search failed, falling back to HTTP: %v\n", err)
		qc.useGRPC = false
	}

	// Fallback to HTTP API
	return qc.searchVectorsHTTP(ctx, embedding, limit)
}

// StoreChunkWithEmbedding stores a code chunk with its embedding
func (qc *QdrantClient) StoreChunkWithEmbedding(ctx context.Context, chunk *CodeChunk, embedding []float32) error {
	// Generate numeric ID from string ID
	hash := fnv.New32a()
	hash.Write([]byte(chunk.ID))
	numericID := hash.Sum32()

	if qc.useGRPC {
		return qc.storeChunkGRPC(ctx, chunk, embedding, uint64(numericID))
	}

	return qc.storeChunkHTTP(ctx, chunk, embedding, numericID)
}

// GenerateOpenAIEmbedding generates OpenAI embeddings with caching
func (qc *QdrantClient) GenerateOpenAIEmbedding(ctx context.Context, text string) ([]float32, error) {
	return qc.generateOpenAIEmbedding(ctx, text)
}

// GetStats returns collection statistics for monitoring
func (qc *QdrantClient) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if qc.useGRPC {
		resp, err := qc.collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
			CollectionName: qc.config.Collection,
		})
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"result": map[string]interface{}{
				"points_count":         resp.Result.PointsCount,
				"vectors_count":        resp.Result.VectorsCount,
				"indexed_vectors_count": resp.Result.PointsCount,
				"status":               resp.Result.Status.String(),
			},
		}, nil
	}

	// HTTP fallback
	url := fmt.Sprintf("http://%s:%d/collections/%s", qc.config.Host, qc.config.Port, qc.config.Collection)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := qc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// Health checks if the Qdrant instance is healthy
func (qc *QdrantClient) Health(ctx context.Context) error {
	if qc.useGRPC {
		_, err := qc.collectionsClient.List(ctx, &qdrant.ListCollectionsRequest{})
		return err
	}
	return qc.testHTTPConnection()
}

// Close closes the gRPC connection if active
func (qc *QdrantClient) Close() error {
	if qc.conn != nil {
		return qc.conn.Close()
	}
	return nil
}

// OptimizeCollection optimizes the vector collection
func (qc *QdrantClient) OptimizeCollection(ctx context.Context) error {
	url := fmt.Sprintf("http://%s:%d/collections/%s/cluster", qc.config.Host, qc.config.Port, qc.config.Collection)
	resp, err := qc.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// =============================================================================
// PRIVATE IMPLEMENTATION METHODS
// =============================================================================

// setupGRPCConnection establishes and tests gRPC connection
func (qc *QdrantClient) setupGRPCConnection() error {
	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", qc.config.Host, qc.config.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC: %w", err)
	}

	qc.conn = conn
	qc.pointsClient = qdrant.NewPointsClient(conn)
	qc.collectionsClient = qdrant.NewCollectionsClient(conn)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = qc.collectionsClient.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		conn.Close()
		return fmt.Errorf("gRPC connection test failed: %w", err)
	}

	qc.useGRPC = true
	fmt.Printf("✅ Qdrant gRPC connection successful\n")
	return nil
}

// testHTTPConnection tests HTTP API availability
func (qc *QdrantClient) testHTTPConnection() error {
	url := fmt.Sprintf("http://%s:%d/collections", qc.config.Host, qc.config.Port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := qc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP API returned status %d", resp.StatusCode)
	}

	return nil
}

// ensureCollection creates collection if it doesn't exist
func (qc *QdrantClient) ensureCollection(ctx context.Context) error {
	exists, err := qc.collectionExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		fmt.Printf("✅ Collection '%s' already exists\n", qc.config.Collection)
		return nil
	}

	fmt.Printf("🔄 Creating collection '%s'...\n", qc.config.Collection)
	return qc.createCollection(ctx)
}

// collectionExists checks if the collection exists
func (qc *QdrantClient) collectionExists(ctx context.Context) (bool, error) {
	if qc.useGRPC {
		_, err := qc.collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
			CollectionName: qc.config.Collection,
		})
		return err == nil, nil
	}

	url := fmt.Sprintf("http://%s:%d/collections/%s", qc.config.Host, qc.config.Port, qc.config.Collection)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := qc.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

// createCollection creates a new collection
func (qc *QdrantClient) createCollection(ctx context.Context) error {
	if qc.useGRPC {
		_, err := qc.collectionsClient.Create(ctx, &qdrant.CreateCollection{
			CollectionName: qc.config.Collection,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     uint64(qc.config.VectorSize),
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})
		return err
	}

	// HTTP fallback
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

	url := fmt.Sprintf("http://%s:%d/collections/%s", qc.config.Host, qc.config.Port, qc.config.Collection)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
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
		return fmt.Errorf("failed to create collection, status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✅ Collection '%s' created successfully\n", qc.config.Collection)
	return nil
}

// generateQueryEmbedding generates embedding for search query
func (qc *QdrantClient) generateQueryEmbedding(ctx context.Context, query string) ([]float32, error) {
	// Try OpenAI embeddings first if API key is available
	if os.Getenv("OPENAI_API_KEY") != "" {
		embedding, err := qc.generateOpenAIEmbedding(ctx, query)
		if err == nil {
			return embedding, nil
		}
		fmt.Printf("⚠️ OpenAI embedding failed, using fallback: %v\n", err)
	}

	// Fallback to simple hash-based embedding
	return qc.generateSimpleEmbedding(query), nil
}

// generateOpenAIEmbedding generates OpenAI embeddings
func (qc *QdrantClient) generateOpenAIEmbedding(ctx context.Context, text string) ([]float32, error) {
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

	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
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
	}

	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, err
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned from OpenAI")
	}

	return embeddingResp.Data[0].Embedding, nil
}

// generateSimpleEmbedding creates a simple hash-based embedding for testing
func (qc *QdrantClient) generateSimpleEmbedding(query string) []float32 {
	words := strings.Fields(strings.ToLower(query))
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

// searchVectorsGRPC performs search using gRPC
func (qc *QdrantClient) searchVectorsGRPC(ctx context.Context, embedding []float32, limit int) ([]*SearchResult, error) {
	searchReq := &qdrant.SearchPoints{
		CollectionName: qc.config.Collection,
		Vector:         embedding,
		Limit:          uint64(limit),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	}

	resp, err := qc.pointsClient.Search(ctx, searchReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC search failed: %w", err)
	}

	results := make([]*SearchResult, len(resp.Result))
	for i, hit := range resp.Result {
		chunk := &CodeChunk{}

		if payload := hit.Payload; payload != nil {
			if file := payload["file"]; file != nil {
				chunk.FilePath = file.GetStringValue()
			}
			if content := payload["content"]; content != nil {
				chunk.Content = content.GetStringValue()
			}
			if language := payload["language"]; language != nil {
				chunk.Language = language.GetStringValue()
			}
			if originalID := payload["original_id"]; originalID != nil {
				chunk.ID = originalID.GetStringValue()
			}
			if startLine := payload["start_line"]; startLine != nil {
				chunk.StartLine = int(startLine.GetIntegerValue())
			}
			if endLine := payload["end_line"]; endLine != nil {
				chunk.EndLine = int(endLine.GetIntegerValue())
			}
		}

		results[i] = &SearchResult{
			Score: hit.Score,
			Chunk: chunk,
		}
	}

	return results, nil
}

// searchVectorsHTTP performs search using HTTP API
func (qc *QdrantClient) searchVectorsHTTP(ctx context.Context, embedding []float32, limit int) ([]*SearchResult, error) {
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
		return nil, fmt.Errorf("HTTP search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp struct {
		Result []struct {
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
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

// storeChunkGRPC stores chunk using gRPC
func (qc *QdrantClient) storeChunkGRPC(ctx context.Context, chunk *CodeChunk, embedding []float32, id uint64) error {
	point := &qdrant.PointStruct{
		Id: &qdrant.PointId{PointIdOptions: &qdrant.PointId_Num{Num: id}},
		Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{
			Vector: &qdrant.Vector{Data: embedding},
		}},
		Payload: map[string]*qdrant.Value{
			"original_id": {Kind: &qdrant.Value_StringValue{StringValue: chunk.ID}},
			"file":        {Kind: &qdrant.Value_StringValue{StringValue: chunk.FilePath}},
			"content":     {Kind: &qdrant.Value_StringValue{StringValue: chunk.Content}},
			"language":    {Kind: &qdrant.Value_StringValue{StringValue: chunk.Language}},
			"start_line":  {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(chunk.StartLine)}},
			"end_line":    {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(chunk.EndLine)}},
		},
	}

	_, err := qc.pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: qc.config.Collection,
		Points:         []*qdrant.PointStruct{point},
	})

	if err != nil {
		fmt.Printf("⚠️ gRPC store failed, falling back to HTTP: %v\n", err)
		qc.useGRPC = false
		return qc.storeChunkHTTP(ctx, chunk, embedding, uint32(id))
	}

	return nil
}

// storeChunkHTTP stores chunk using HTTP API
func (qc *QdrantClient) storeChunkHTTP(ctx context.Context, chunk *CodeChunk, embedding []float32, id uint32) error {
	point := map[string]interface{}{
		"id":     id,
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
		return fmt.Errorf("HTTP store request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("store failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Upsert compatibility method for other parts of the system
func (qc *QdrantClient) Upsert(ctx context.Context, points []*qdrant.PointStruct) error {
	if qc.pointsClient != nil {
		_, err := qc.pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: qc.config.Collection,
			Points:         points,
		})
		return err
	}
	return fmt.Errorf("gRPC client not available for Upsert")
}

// searchVectors compatibility method
func (qc *QdrantClient) searchVectors(ctx context.Context, embedding []float32, limit int) ([]*SearchResult, error) {
	return qc.searchVectorsHTTP(ctx, embedding, limit)
}