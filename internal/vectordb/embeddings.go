package vectordb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func NewEmbeddingService(config *EmbeddingConfig) *EmbeddingService {
	return &EmbeddingService{config: config}
}

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

func (es *EmbeddingService) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := EmbeddingRequest{
		Input: texts,
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

	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(embeddingResp.Data))
	for i, data := range embeddingResp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}
