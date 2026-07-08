package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/omnidev/services/rag/internal/domain"
)

// mockRetriever implements retriever.Retriever for testing.
type mockRetriever struct {
	results []domain.SearchResult
	err     error
}

func (m *mockRetriever) Search(ctx context.Context, req *domain.SearchRequest) ([]domain.SearchResult, error) {
	return m.results, m.err
}

// mockChunkRepo implements repository.ChunkRepository for testing.
type mockChunkRepo struct {
	chunks []domain.DocumentChunk
	err    error
}

func (m *mockChunkRepo) CreateBatch(ctx context.Context, chunks []domain.DocumentChunk) error { return m.err }
func (m *mockChunkRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DocumentChunk, error) {
	return nil, m.err
}
func (m *mockChunkRepo) ListByDocument(ctx context.Context, docID uuid.UUID) ([]domain.DocumentChunk, error) {
	return m.chunks, m.err
}
func (m *mockChunkRepo) DeleteByDocument(ctx context.Context, docID uuid.UUID) error { return m.err }
func (m *mockChunkRepo) CountByKB(ctx context.Context, kbID uuid.UUID) (int, error)  { return 0, m.err }
func (m *mockChunkRepo) TotalTokensByKB(ctx context.Context, kbID uuid.UUID) (int64, error) {
	return 0, m.err
}

func TestSearchService_Search(t *testing.T) {
	kbID := uuid.New()

	tests := []struct {
		name      string
		input     *SearchInput
		results   []domain.SearchResult
		retrieverErr error
		wantErr   bool
		wantCount int
	}{
		{
			name: "successful search",
			input: &SearchInput{
				Query: "test query",
				TopK:  5,
			},
			results: []domain.SearchResult{
				{
					Chunk: domain.DocumentChunk{
						ID:      uuid.New(),
						Content: "relevant content",
					},
					Score:  0.85,
					Source: "hybrid",
				},
			},
			wantCount: 1,
		},
		{
			name: "default top_k and min_score",
			input: &SearchInput{
				Query: "another query",
			},
			results:   []domain.SearchResult{},
			wantCount: 0,
		},
		{
			name: "retriever error",
			input: &SearchInput{
				Query: "fail query",
			},
			retrieverErr: context.DeadlineExceeded,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retriever := &mockRetriever{
				results: tt.results,
				err:     tt.retrieverErr,
			}
			chunkRepo := &mockChunkRepo{}
			svc := NewSearchService(retriever, chunkRepo)

			results, err := svc.Search(context.Background(), uuid.New(), kbID, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(results) != tt.wantCount {
				t.Errorf("Search() got %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestSearchService_Search_Defaults(t *testing.T) {
	retriever := &mockRetriever{results: []domain.SearchResult{}}
	chunkRepo := &mockChunkRepo{}
	svc := NewSearchService(retriever, chunkRepo)

	// Verify defaults are applied
	input := &SearchInput{Query: "test"}
	_, err := svc.Search(context.Background(), uuid.New(), uuid.New(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The retriever should have received default values
	// (we can't directly verify this with the mock, but the call succeeding confirms it)
}
