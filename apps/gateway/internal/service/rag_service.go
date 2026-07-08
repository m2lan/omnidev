// Package service contains business logic for the API Gateway.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"
)

// RAGService provides access to the RAG Service for knowledge retrieval.
type RAGService struct {
	baseURL    string
	httpClient *http.Client
}

// NewRAGService creates a new RAG service client.
func NewRAGService(baseURL string) *RAGService {
	return &RAGService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchResult mirrors the RAG service search result structure.
type RAGSearchResult struct {
	Chunk struct {
		ID              string                 `json:"id"`
		DocumentID      string                 `json:"document_id"`
		KnowledgeBaseID string                 `json:"knowledge_base_id"`
		ChunkIndex      int                    `json:"chunk_index"`
		Content         string                 `json:"content"`
		ContentLength   int                    `json:"content_length"`
		TokenCount      int                    `json:"token_count"`
		Heading         *string                `json:"heading,omitempty"`
		Metadata        map[string]interface{} `json:"metadata"`
	} `json:"chunk"`
	Score  float64 `json:"score"`
	Source string  `json:"source"`
}

// Search performs a hybrid search on a knowledge base via the RAG service.
func (s *RAGService) Search(ctx context.Context, token string, kbID string, query string, topK int) ([]RAGSearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"top_k": topK,
	})

	url := fmt.Sprintf("%s/api/v1/knowledge/%s/search", s.baseURL, kbID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RAG service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Log.Warn("RAG search failed",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("RAG service returned %d", resp.StatusCode)
	}

	var result struct {
		Data []RAGSearchResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Data, nil
}
