package service

import (
	"context"

	"github.com/google/uuid"

	appErr "github.com/omnidev/go-common/errors"

	"github.com/omnidev/gateway/internal/rag/domain"
	"github.com/omnidev/gateway/internal/rag/repository"
	"github.com/omnidev/gateway/internal/rag/retriever"
)

// SearchService handles search operations.
type SearchService struct {
	retriever retriever.Retriever
	chunkRepo repository.ChunkRepository
}

// NewSearchService creates a new search service.
func NewSearchService(retriever retriever.Retriever, chunkRepo repository.ChunkRepository) *SearchService {
	return &SearchService{
		retriever: retriever,
		chunkRepo: chunkRepo,
	}
}

// SearchInput defines the input for search.
type SearchInput struct {
	Query    string  `json:"query" validate:"required"`
	TopK     int     `json:"top_k"`
	MinScore float64 `json:"min_score"`
}

// Search performs a hybrid search on a knowledge base.
func (s *SearchService) Search(ctx context.Context, userID, kbID uuid.UUID, input *SearchInput) ([]domain.SearchResult, error) {
	if input.TopK <= 0 {
		input.TopK = 10
	}
	if input.MinScore <= 0 {
		// RRF scores are typically very small (max ~0.016 with k=60)
		// Use a much lower threshold to avoid filtering out valid results
		input.MinScore = 0.01
	}

	req := &domain.SearchRequest{
		Query:           input.Query,
		KnowledgeBaseID: kbID,
		TopK:            input.TopK,
		MinScore:        input.MinScore,
	}

	results, err := s.retriever.Search(ctx, req)
	if err != nil {
		return nil, appErr.Wrap(err, "search failed")
	}

	return results, nil
}
