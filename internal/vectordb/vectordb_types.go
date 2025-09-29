package vectordb

import (
	"context"
	"net/http"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
)

// =============================================================================
// CORE CLIENT TYPES
// =============================================================================

// QdrantClient handles both gRPC and HTTP connections to Qdrant
type QdrantClient struct {
	// gRPC clients
	pointsClient      qdrant.PointsClient
	collectionsClient qdrant.CollectionsClient
	conn              *grpc.ClientConn

	// Configuration and state
	config  *QdrantConfig
	useGRPC bool

	// HTTP client for fallback
	httpClient *http.Client
}

// QdrantConfig holds comprehensive Qdrant configuration
type QdrantConfig struct {
	Host              string        `json:"host"`
	Port              int           `json:"port"`
	Collection        string        `json:"collection"`
	VectorSize        int           `json:"vector_size"`
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	BatchSize         int           `json:"batch_size"`
	APIKey            string        `json:"api_key,omitempty"`
	UseTLS            bool          `json:"use_tls"`
}

// =============================================================================
// CODE CHUNK AND SEARCH TYPES
// =============================================================================

// CodeChunk represents a chunk of code for vector storage
type CodeChunk struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	FilePath  string            `json:"file_path"`
	Language  string            `json:"language"`
	StartLine int               `json:"start_line"`
	EndLine   int               `json:"end_line"`
	ChunkType string            `json:"chunk_type"` // function, method, type, file
	Package   string            `json:"package,omitempty"`
	Function  string            `json:"function,omitempty"`
	Metadata  map[string]string `json:"metadata"`
}

// SearchResult represents a vector search result
type SearchResult struct {
	Chunk *CodeChunk `json:"chunk"`
	Score float32    `json:"score"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query           string            `json:"query"`
	Embedding       []float32         `json:"embedding,omitempty"`
	Limit           int               `json:"limit"`
	Threshold       float32           `json:"threshold"`
	Filters         map[string]string `json:"filters"`
	IncludeContent  bool              `json:"include_content"`
	BoostFactors    map[string]float32 `json:"boost_factors"`
}

// =============================================================================
// EMBEDDING SERVICE TYPES
// =============================================================================

// EmbeddingService handles text-to-vector conversion
type EmbeddingService struct {
	config *EmbeddingConfig
	cache  *EmbeddingCache
}

// EmbeddingConfig holds embedding service configuration
type EmbeddingConfig struct {
	APIKey   string `json:"api_key"`
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
}

// EmbeddingCache provides caching for embeddings
type EmbeddingCache struct {
	cache   map[string][]float32
	maxSize int
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// =============================================================================
// SERVICE TYPES
// =============================================================================

// SearchService provides high-level search functionality
type SearchService struct {
	client   *QdrantClient
	embedder *EmbeddingService
}

// SemanticIndex manages semantic indexing operations
type SemanticIndex struct {
	client    *QdrantClient
	embedder  *EmbeddingService
	optimizer *VectorOptimizer
}

// VectorOptimizer optimizes vector operations
type VectorOptimizer struct {
	client *QdrantClient
}

// ContextRetrieval handles context-aware retrieval
type ContextRetrieval struct {
	searchService  *SearchService
	rankingService *RankingService
}

// RankingService handles result ranking
type RankingService struct {
	weights RankingWeights
}

// =============================================================================
// RANKING AND CONTEXT TYPES
// =============================================================================

// RankingWeights defines weights for ranking factors
type RankingWeights struct {
	Similarity    float64 `json:"similarity"`
	TextMatch     float64 `json:"text_match"`
	FileRelevance float64 `json:"file_relevance"`
	Recency       float64 `json:"recency"`
	Frequency     float64 `json:"frequency"`
}

// ContextResult represents a search result with context
type ContextResult struct {
	MainChunk     *CodeChunk   `json:"main_chunk"`
	ContextChunks []*CodeChunk `json:"context_chunks"`
	Score         float32      `json:"score"`
	Summary       string       `json:"summary"`
}

// =============================================================================
// INTERFACE DEFINITIONS
// =============================================================================

// VectorDB interface for dependency injection
type VectorDB interface {
	Search(ctx context.Context, query string, limit int) ([]*SearchResult, error)
	Store(ctx context.Context, chunk *CodeChunk, embedding []float32) error
	Delete(ctx context.Context, id string) error
	Health(ctx context.Context) error
	Close() error
}

// Embedder interface for embedding generation
type Embedder interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// Searcher interface for search operations
type Searcher interface {
	Search(ctx context.Context, query string, limit int, filters map[string]string) ([]*SearchResult, error)
	SearchSimilar(ctx context.Context, chunk *CodeChunk, limit int) ([]*SearchResult, error)
}

// =============================================================================
// STATISTICS AND MONITORING
// =============================================================================

// VectorDBStats represents vector database statistics
type VectorDBStats struct {
	TotalVectors    int64     `json:"total_vectors"`
	CollectionSize  int64     `json:"collection_size"`
	IndexedFiles    int       `json:"indexed_files"`
	LastIndexed     time.Time `json:"last_indexed"`
	SearchCount     int64     `json:"search_count"`
	AverageLatency  float64   `json:"average_latency"`
	CacheHitRate    float64   `json:"cache_hit_rate"`
	HealthStatus    string    `json:"health_status"`
}

// SearchMetrics tracks search performance
type SearchMetrics struct {
	TotalSearches     int64         `json:"total_searches"`
	AverageLatency    time.Duration `json:"average_latency"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	TopQueries        []string      `json:"top_queries"`
	SuccessRate       float64       `json:"success_rate"`
	LastSearchTime    time.Time     `json:"last_search_time"`
}

// =============================================================================
// ERROR TYPES
// =============================================================================

// VectorDBError represents vector database errors
type VectorDBError struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
	Retryable bool      `json:"retryable"`
}

func (e *VectorDBError) Error() string {
	return fmt.Sprintf("VectorDB %s error: %s", e.Type, e.Message)
}

// =============================================================================
// BATCH OPERATION TYPES
// =============================================================================

// BatchOperation represents batch operations
type BatchOperation struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"` // upsert, delete, update
	Chunks    []*CodeChunk `json:"chunks,omitempty"`
	IDs       []string    `json:"ids,omitempty"`
	Status    string      `json:"status"`
	Progress  int         `json:"progress"`
	StartTime time.Time   `json:"start_time"`
	EndTime   *time.Time  `json:"end_time,omitempty"`
}

// BatchResult represents batch operation results
type BatchResult struct {
	OperationID   string        `json:"operation_id"`
	Success       bool          `json:"success"`
	ProcessedCount int          `json:"processed_count"`
	FailedCount   int           `json:"failed_count"`
	Duration      time.Duration `json:"duration"`
	Errors        []string      `json:"errors,omitempty"`
}