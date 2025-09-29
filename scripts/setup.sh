package vectordb

import (
	"context"
	"fmt"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

// QdrantClient wraps the Qdrant vector database client
type QdrantClient struct {
	client         *qdrant.Client
	collectionName string
	dimension      uint64
	config         QdrantConfig
}

// QdrantConfig holds Qdrant client configuration
type QdrantConfig struct {
	Host           string
	Port           int
	APIKey         string
	UseTLS         bool
	CollectionName string
	Dimension      uint64
	Timeout        time.Duration
	BatchSize      int
	RetryAttempts  int
}

// CodeEmbedding represents a code embedding with metadata
type CodeEmbedding struct {
	ID       string                 `json:"id"`
	Vector   []float32              `json:"vector"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SearchRequest represents a vector search request
type SearchRequest struct {
	Vector      []float32      `json:"vector"`
	TopK        uint64         `json:"top_k"`
	Filter      *qdrant.Filter `json:"filter,omitempty"`
	WithPayload bool           `json:"with_payload"`
	WithVector  bool           `json:"with_vector"`
	Threshold   float32        `json:"threshold,omitempty"`
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID       string                 `json:"id"`
	Score    float32                `json:"score"`
	Vector   []float32              `json:"vector,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

// BatchOperation represents a batch operation
type BatchOperation struct {
	Operation string           `json:"operation"` // upsert, delete
	Points    []*CodeEmbedding `json:"points,omitempty"`
	IDs       []string         `json:"ids,omitempty"`
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(host string, port int, apiKey, collectionName string, dimension uint64) (*QdrantClient, error) {
	config := QdrantConfig{
		Host:           host,
		Port:           port,
		APIKey:         apiKey,
		CollectionName: collectionName,
		Dimension:      dimension,
		Timeout:        30 * time.Second,
		BatchSize:      100,
		RetryAttempts:  3,
		UseTLS:         false,
	}

	return NewQdrantClientWithConfig(config)
}

// NewQdrantClientWithConfig creates a new Qdrant client with full configuration
func NewQdrantClientWithConfig(config QdrantConfig) (*QdrantClient, error) {
	// Create Qdrant client using the high-level API
	clientConfig := &qdrant.Config{
		Host:   config.Host,
		Port:   config.Port,
		APIKey: config.APIKey,
		UseTLS: config.UseTLS,
	}

	client, err := qdrant.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	qc := &QdrantClient{
		client:         client,
		collectionName: config.CollectionName,
		dimension:      config.Dimension,
		config:         config,
	}

	// Initialize collection
	if err := qc.initializeCollection(); err != nil {
		return nil, fmt.Errorf("failed to initialize collection: %w", err)
	}

	return qc, nil
}

// initializeCollection creates the collection if it doesn't exist
func (qc *QdrantClient) initializeCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), qc.config.Timeout)
	defer cancel()

	// Check if collection exists
	exists, err := qc.collectionExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		return nil
	}

	// Create collection using the high-level API
	_, err = qc.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: qc.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     qc.dimension,
			Distance: qdrant.Distance_Cosine,
		}),
		OptimizersConfig: &qdrant.OptimizersConfigDiff{
			DefaultSegmentNumber: uint64Ptr(2),
		},
		ReplicationFactor:      uint32Ptr(1),
		WriteConsistencyFactor: uint32Ptr(1),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// collectionExists checks if the collection exists
func (qc *QdrantClient) collectionExists(ctx context.Context) (bool, error) {
	info, err := qc.client.GetCollectionInfo(ctx, qc.collectionName)
	if err != nil {
		// Collection doesn't exist
		return false, nil
	}
	return info != nil, nil
}

// UpsertEmbedding inserts or updates a single embedding
func (qc *QdrantClient) UpsertEmbedding(ctx context.Context, embedding *CodeEmbedding) error {
	return qc.UpsertEmbeddings(ctx, []*CodeEmbedding{embedding})
}

// UpsertEmbeddings inserts or updates multiple embeddings
func (qc *QdrantClient) UpsertEmbeddings(ctx context.Context, embeddings []*CodeEmbedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	// Convert to Qdrant points using helper functions
	points := make([]*qdrant.PointStruct, len(embeddings))
	for i, emb := range embeddings {
		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(emb.ID),
			Vectors: qdrant.NewVectors(emb.Vector...),
			Payload: qdrant.NewValueMap(emb.Metadata),
		}
	}

	// Perform upsert
	_, err := qc.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: qc.collectionName,
		Points:         points,
		Wait:           boolPtr(true),
	})
	if err != nil {
		return fmt.Errorf("failed to upsert %d points: %w", len(points), err)
	}

	return nil
}

// Search performs vector similarity search using Query method
func (qc *QdrantClient) Search(ctx context.Context, request *SearchRequest) ([]*SearchResult, error) {
	queryRequest := &qdrant.QueryPoints{
		CollectionName: qc.collectionName,
		Query:          qdrant.NewQuery(request.Vector...),
		Limit:          uintPtr(request.TopK),
		WithPayload:    qdrant.NewWithPayload(request.WithPayload),
		WithVectors:    qdrant.NewWithVectors(request.WithVector),
		Filter:         request.Filter,
	}

	if request.Threshold > 0 {
		queryRequest.ScoreThreshold = &request.Threshold
	}

	response, err := qc.client.Query(ctx, queryRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to query points: %w", err)
	}

	// Convert results
	results := make([]*SearchResult, len(response))
	for i, point := range response {
		result := &SearchResult{
			Score: point.Score,
		}

		// Extract ID
		if point.Id != nil {
			result.ID = qc.extractID(point.Id)
		}

		// Extract vector if requested
		if request.WithVector && point.Vectors != nil {
			result.Vector = qc.extractVector(point.Vectors)
		}

		// Extract payload/metadata
		if request.WithPayload && point.Payload != nil {
			result.Metadata = qc.convertPayloadToMetadata(point.Payload)
		}

		results[i] = result
	}

	return results, nil
}

// SearchByText searches for similar code by text query (requires external embedding)
func (qc *QdrantClient) SearchByText(ctx context.Context, textEmbedding []float32, filters map[string]interface{}, topK uint64) ([]*SearchResult, error) {
	var filter *qdrant.Filter
	if len(filters) > 0 {
		filter = qc.buildFilter(filters)
	}

	request := &SearchRequest{
		Vector:      textEmbedding,
		TopK:        topK,
		Filter:      filter,
		WithPayload: true,
		WithVector:  false,
		Threshold:   0.7,
	}

	return qc.Search(ctx, request)
}

// SearchByFunction searches for similar functions
func (qc *QdrantClient) SearchByFunction(ctx context.Context, functionEmbedding []float32, language string, topK uint64) ([]*SearchResult, error) {
	filters := map[string]interface{}{
		"type":     "function",
		"language": language,
	}

	return qc.SearchByText(ctx, functionEmbedding, filters, topK)
}

// SearchByFile searches for similar files
func (qc *QdrantClient) SearchByFile(ctx context.Context, fileEmbedding []float32, extension string, topK uint64) ([]*SearchResult, error) {
	filters := map[string]interface{}{
		"type":      "file",
		"extension": extension,
	}

	return qc.SearchByText(ctx, fileEmbedding, filters, topK)
}

// DeleteEmbedding deletes a single embedding
func (qc *QdrantClient) DeleteEmbedding(ctx context.Context, id string) error {
	return qc.DeleteEmbeddings(ctx, []string{id})
}

// DeleteEmbeddings deletes multiple embeddings
func (qc *QdrantClient) DeleteEmbeddings(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Convert IDs to PointsSelector
	pointIds := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIds[i] = qdrant.NewIDUUID(id)
	}

	_, err := qc.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: qc.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: pointIds,
				},
			},
		},
		Wait: boolPtr(true),
	})
	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}

	return nil
}

// GetEmbedding retrieves a single embedding by ID
func (qc *QdrantClient) GetEmbedding(ctx context.Context, id string) (*CodeEmbedding, error) {
	embeddings, err := qc.GetEmbeddings(ctx, []string{id})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("embedding not found")
	}

	return embeddings[0], nil
}

// GetEmbeddings retrieves multiple embeddings by IDs
func (qc *QdrantClient) GetEmbeddings(ctx context.Context, ids []string) ([]*CodeEmbedding, error) {
	if len(ids) == 0 {
		return []*CodeEmbedding{}, nil
	}

	pointIds := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIds[i] = qdrant.NewIDUUID(id)
	}

	response, err := qc.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: qc.collectionName,
		Ids:            pointIds,
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get points: %w", err)
	}

	embeddings := make([]*CodeEmbedding, len(response))
	for i, point := range response {
		embedding := &CodeEmbedding{
			ID: qc.extractID(point.Id),
		}

		if point.Vectors != nil {
			embedding.Vector = qc.extractVector(point.Vectors)
		}

		if point.Payload != nil {
			embedding.Metadata = qc.convertPayloadToMetadata(point.Payload)
		}

		embeddings[i] = embedding
	}

	return embeddings, nil
}

// GetCollectionInfo returns information about the collection
func (qc *QdrantClient) GetCollectionInfo(ctx context.Context) (*qdrant.CollectionInfo, error) {
	return qc.client.GetCollectionInfo(ctx, qc.collectionName)
}

// Count returns the number of vectors in the collection
func (qc *QdrantClient) Count(ctx context.Context, filter map[string]interface{}) (uint64, error) {
	var qdrantFilter *qdrant.Filter
	if len(filter) > 0 {
		qdrantFilter = qc.buildFilter(filter)
	}

	response, err := qc.client.Count(ctx, &qdrant.CountPoints{
		CollectionName: qc.collectionName,
		Filter:         qdrantFilter,
		Exact:          boolPtr(true),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count points: %w", err)
	}

	return response.Count, nil
}

// Helper methods

// buildFilter builds a Qdrant filter from a map
func (qc *QdrantClient) buildFilter(filters map[string]interface{}) *qdrant.Filter {
	var conditions []*qdrant.Condition

	for key, value := range filters {
		condition := qdrant.NewMatch(key, value)
		conditions = append(conditions, condition)
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}

// extractID extracts string ID from PointId
func (qc *QdrantClient) extractID(pointId *qdrant.PointId) string {
	if pointId == nil {
		return ""
	}
	
	if uuid := pointId.GetUuid(); uuid != "" {
		return uuid
	}
	
	if num := pointId.GetNum(); num != 0 {
		return fmt.Sprintf("%d", num)
	}
	
	return ""
}

// extractVector extracts float32 vector from Vectors
func (qc *QdrantClient) extractVector(vectors *qdrant.Vectors) []float32 {
	if vectors == nil {
		return nil
	}
	
	if vector := vectors.GetVector(); vector != nil {
		return vector.Data
	}
	
	return nil
}

// convertPayloadToMetadata converts Qdrant payload to metadata map
func (qc *QdrantClient) convertPayloadToMetadata(payload map[string]*qdrant.Value) map[string]interface{} {
	metadata := make(map[string]interface{})

	for key, value := range payload {
		if value == nil {
			continue
		}
		
		switch v := value.Kind.(type) {
		case *qdrant.Value_StringValue:
			metadata[key] = v.StringValue
		case *qdrant.Value_IntegerValue:
			metadata[key] = v.IntegerValue
		case *qdrant.Value_DoubleValue:
			metadata[key] = v.DoubleValue
		case *qdrant.Value_BoolValue:
			metadata[key] = v.BoolValue
		}
	}

	return metadata
}

// Close closes the connection to Qdrant
func (qc *QdrantClient) Close() error {
	if qc.client != nil {
		return qc.client.Close()
	}
	return nil
}

// Health checks the health of the Qdrant service
func (qc *QdrantClient) Health(ctx context.Context) error {
	_, err := qc.GetCollectionInfo(ctx)
	return err
}

// Helper functions for pointer creation
func uint64Ptr(v uint64) *uint64 {
	return &v
}

func uint32Ptr(v uint32) *uint32 {
	return &v
}

func uintPtr(v uint64) *uint64 {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}