package vectordb

import (
	"context"
	"crypto/md5"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

func NewSemanticIndex(client *QdrantClient, embedder *EmbeddingService) *SemanticIndex {
	return &SemanticIndex{
		client:    client,
		embedder:  embedder,
		optimizer: NewVectorOptimizer(client),
	}
}

func (si *SemanticIndex) IndexCodeChunk(ctx context.Context, chunk *CodeChunk) error {
	embedding, err := si.embedder.GenerateEmbedding(ctx, chunk.Content)
	if err != nil {
		return err
	}

	point := &qdrant.PointStruct{
		Id: &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: chunk.ID}},
		Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: embedding}}},
		Payload: map[string]*qdrant.Value{
			"content":   {Kind: &qdrant.Value_StringValue{StringValue: chunk.Content}},
			"file_path": {Kind: &qdrant.Value_StringValue{StringValue: chunk.FilePath}},
			"language":  {Kind: &qdrant.Value_StringValue{StringValue: chunk.Language}},
			"start_line": {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(chunk.StartLine)}},
			"end_line":   {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(chunk.EndLine)}},
		},
	}

	return si.client.Upsert(ctx, []*qdrant.PointStruct{point})
}

func (si *SemanticIndex) IndexCodeChunks(ctx context.Context, chunks []*CodeChunk) error {
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := si.embedder.GenerateEmbeddings(ctx, texts)
	if err != nil {
		return err
	}

	points := make([]*qdrant.PointStruct, len(chunks))
	for i, chunk := range chunks {
		points[i] = &qdrant.PointStruct{
			Id: &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: chunk.ID}},
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: embeddings[i]}}},
			Payload: map[string]*qdrant.Value{
				"content":   {Kind: &qdrant.Value_StringValue{StringValue: chunk.Content}},
				"file_path": {Kind: &qdrant.Value_StringValue{StringValue: chunk.FilePath}},
				"language":  {Kind: &qdrant.Value_StringValue{StringValue: chunk.Language}},
				"start_line": {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(chunk.StartLine)}},
				"end_line":   {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(chunk.EndLine)}},
			},
		}
	}

	return si.optimizer.BatchUpsert(ctx, points, 100)
}

func (si *SemanticIndex) GenerateChunkID(chunk *CodeChunk) string {
	data := fmt.Sprintf("%s:%d:%d:%s", chunk.FilePath, chunk.StartLine, chunk.EndLine, chunk.Content)
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}
