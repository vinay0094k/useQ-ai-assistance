package vectordb

import (
	"time"
)

// =============================================================================
// MINIMAL TYPES - Only what we actually need
// =============================================================================

// QdrantConfig holds basic Qdrant configuration
type QdrantConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Collection string `json:"collection"`
	VectorSize int    `json:"vector_size"`
}

// CodeChunk represents a chunk of code for vector storage
type CodeChunk struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	FilePath  string `json:"file_path"`
	Language  string `json:"language"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// SearchResult represents a vector search result
type SearchResult struct {
	Chunk *CodeChunk `json:"chunk"`
	Score float32    `json:"score"`
}

// EmbeddingConfig holds embedding service configuration
type EmbeddingConfig struct {
	APIKey   string `json:"api_key"`
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
}

// CostTracker tracks embedding costs
type CostTracker struct {
	TotalTokens  int     `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
	RequestCount int     `json:"request_count"`
}

// VectorDBStats represents basic statistics
type VectorDBStats struct {
	TotalVectors   int64     `json:"total_vectors"`
	IndexedFiles   int       `json:"indexed_files"`
	LastIndexed    time.Time `json:"last_indexed"`
	SearchCount    int64     `json:"search_count"`
	EmbeddingCosts float64   `json:"embedding_costs"`
}