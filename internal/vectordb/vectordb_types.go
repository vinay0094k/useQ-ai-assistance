package vectordb

import (
	"context"
	"net/http"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
)

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

// Data structures
type CodeChunk struct {
	ID        string
	Content   string
	FilePath  string
	Language  string
	StartLine int
	EndLine   int
}

type SearchResult struct {
	Chunk *CodeChunk
	Score float32
}

type ContextResult struct {
	MainChunk     *CodeChunk
	ContextChunks []*CodeChunk
	Score         float32
	Summary       string
}

// Service types
type EmbeddingService struct {
	config *EmbeddingConfig
}

type VectorOptimizer struct {
	client *QdrantClient
}

type SemanticIndex struct {
	client    *QdrantClient
	embedder  *EmbeddingService
	optimizer *VectorOptimizer
}

type SearchService struct {
	client   *QdrantClient
	embedder *EmbeddingService
}

type RankingService struct {
	weights RankingWeights
}

type ContextRetrieval struct {
	searchService  *SearchService
	rankingService *RankingService
}

// Configuration types
type QdrantConfig struct {
	Host              string        `json:"host"`
	Port              int           `json:"port"`
	Collection        string        `json:"collection"`
	VectorSize        int           `json:"vector_size"`
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	BatchSize         int           `json:"batch_size"`
}

type EmbeddingConfig struct {
	APIKey   string `json:"api_key"`
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
}

type RankingWeights struct {
	Similarity    float64
	TextMatch     float64
	FileRelevance float64
	Recency       float64
}

// API request/response types
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Interface definitions
type VectorDB interface {
	Upsert(ctx context.Context, points []*qdrant.PointStruct) error
	Search(ctx context.Context, vector []float32, limit uint64, filter *qdrant.Filter) ([]*qdrant.ScoredPoint, error)
	Close() error
}

type Embedder interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

type Searcher interface {
	Search(ctx context.Context, query string, limit int, filters map[string]string) ([]*SearchResult, error)
	SearchSimilar(ctx context.Context, chunk *CodeChunk, limit int) ([]*SearchResult, error)
}
