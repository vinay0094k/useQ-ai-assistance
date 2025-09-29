package vectordb

import (
	"context"
)

func NewSearchService(client *QdrantClient, embedder *EmbeddingService) *SearchService {
	return &SearchService{client: client, embedder: embedder}
}

func (ss *SearchService) Search(ctx context.Context, query string, limit int, filters map[string]string) ([]*SearchResult, error) {
	// Generate OpenAI embedding for query
	queryEmbedding, err := ss.client.GenerateOpenAIEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}

	// Use the existing searchVectors method which handles conversion
	return ss.client.searchVectors(ctx, queryEmbedding, limit)
}

func (ss *SearchService) SearchSimilar(ctx context.Context, chunk *CodeChunk, limit int) ([]*SearchResult, error) {
	return ss.Search(ctx, chunk.Content, limit, nil)
}
