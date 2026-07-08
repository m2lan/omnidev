// Package retriever provides hybrid search capabilities.
package retriever

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"

	"github.com/omnidev/gateway/internal/rag/domain"
	"github.com/omnidev/gateway/internal/rag/embedder"
)

// Retriever defines the interface for document retrieval.
type Retriever interface {
	// Search performs a hybrid search (vector + BM25).
	Search(ctx context.Context, req *domain.SearchRequest) ([]domain.SearchResult, error)
}

// HybridRetriever combines vector similarity search with BM25 keyword search.
type HybridRetriever struct {
	pool     *pgxpool.Pool
	embedder embedder.Embedder
}

// NewHybridRetriever creates a new hybrid retriever.
func NewHybridRetriever(pool *pgxpool.Pool, emb embedder.Embedder) *HybridRetriever {
	return &HybridRetriever{pool: pool, embedder: emb}
}

// Search performs a hybrid search combining vector and keyword search.
func (r *HybridRetriever) Search(ctx context.Context, req *domain.SearchRequest) ([]domain.SearchResult, error) {
	if req.TopK <= 0 {
		req.TopK = 10
	}
	if req.MinScore <= 0 {
		req.MinScore = 0.3
	}

	// Generate query embedding
	queryVector, err := r.embedder.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Vector search
	vectorResults, err := r.vectorSearch(ctx, req.KnowledgeBaseID, queryVector, req.TopK*2)
	if err != nil {
		logger.Log.Warn("Vector search failed", zap.Error(err))
		vectorResults = []domain.SearchResult{}
	}

	// BM25 keyword search
	bm25Results, err := r.bm25Search(ctx, req.KnowledgeBaseID, req.Query, req.TopK*2)
	if err != nil {
		logger.Log.Warn("BM25 search failed", zap.Error(err))
		bm25Results = []domain.SearchResult{}
	}

	// Merge and deduplicate
	merged := r.mergeResults(vectorResults, bm25Results, req.TopK)

	// Filter by minimum score
	filtered := make([]domain.SearchResult, 0, len(merged))
	for _, result := range merged {
		if result.Score >= req.MinScore {
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}

// vectorSearch performs cosine similarity search using pgvector.
func (r *HybridRetriever) vectorSearch(ctx context.Context, kbID uuid.UUID, queryVector []float32, topK int) ([]domain.SearchResult, error) {
	// Convert float32 to string for pgvector
	vectorStr := vectorToString(queryVector)

	query := `
		SELECT dc.id, dc.document_id, dc.knowledge_base_id, dc.chunk_index,
		       dc.content, dc.content_length, dc.token_count,
		       dc.start_page, dc.end_page, dc.heading, dc.metadata,
		       1 - (dc.embedding <=> $1::vector) AS score
		FROM document_chunks dc
		WHERE dc.knowledge_base_id = $2
		ORDER BY dc.embedding <=> $1::vector
		LIMIT $3`

	rows, err := r.pool.Query(ctx, query, vectorStr, kbID, topK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	defer rows.Close()

	results := make([]domain.SearchResult, 0)
	for rows.Next() {
		chunk := domain.DocumentChunk{}
		var score float64

		if err := rows.Scan(
			&chunk.ID, &chunk.DocumentID, &chunk.KnowledgeBaseID, &chunk.ChunkIndex,
			&chunk.Content, &chunk.ContentLength, &chunk.TokenCount,
			&chunk.StartPage, &chunk.EndPage, &chunk.Heading, &chunk.Metadata,
			&score,
		); err != nil {
			return nil, fmt.Errorf("failed to scan vector result: %w", err)
		}

		results = append(results, domain.SearchResult{
			Chunk:  chunk,
			Score:  score,
			Source: "vector",
		})
	}

	return results, nil
}

// bm25Search performs full-text search using PostgreSQL tsvector.
func (r *HybridRetriever) bm25Search(ctx context.Context, kbID uuid.UUID, query string, topK int) ([]domain.SearchResult, error) {
	// Determine the text search configuration to use
	// First try 'chinese' if zhparser is available, otherwise use 'simple'
	tsConfig := "simple"

	// Check if zhparser extension is available
	var hasZhparser bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'zhparser')").Scan(&hasZhparser)
	if err == nil && hasZhparser {
		tsConfig = "chinese"
	}

	// Use plainto_tsquery with the determined config
	sql := fmt.Sprintf(`
		SELECT dc.id, dc.document_id, dc.knowledge_base_id, dc.chunk_index,
		       dc.content, dc.content_length, dc.token_count,
		       dc.start_page, dc.end_page, dc.heading, dc.metadata,
		       ts_rank(dc.content_tsv, plainto_tsquery('%s', $1)) AS score
		FROM document_chunks dc
		WHERE dc.knowledge_base_id = $2
		  AND dc.content_tsv @@ plainto_tsquery('%s', $1)
		ORDER BY score DESC
		LIMIT $3`, tsConfig, tsConfig)

	rows, err := r.pool.Query(ctx, sql, query, kbID, topK)
	if err != nil {
		// Fallback to LIKE search if tsvector not available
		logger.Log.Warn("BM25 search failed, falling back to LIKE search",
			zap.Error(err),
			zap.String("config", tsConfig),
		)
		return r.likeSearch(ctx, kbID, query, topK)
	}
	defer rows.Close()

	results := make([]domain.SearchResult, 0)
	for rows.Next() {
		chunk := domain.DocumentChunk{}
		var score float64

		if err := rows.Scan(
			&chunk.ID, &chunk.DocumentID, &chunk.KnowledgeBaseID, &chunk.ChunkIndex,
			&chunk.Content, &chunk.ContentLength, &chunk.TokenCount,
			&chunk.StartPage, &chunk.EndPage, &chunk.Heading, &chunk.Metadata,
			&score,
		); err != nil {
			return nil, fmt.Errorf("failed to scan BM25 result: %w", err)
		}

		results = append(results, domain.SearchResult{
			Chunk:  chunk,
			Score:  score,
			Source: "bm25",
		})
	}

	return results, nil
}

// likeSearch is a fallback search using LIKE.
func (r *HybridRetriever) likeSearch(ctx context.Context, kbID uuid.UUID, query string, topK int) ([]domain.SearchResult, error) {
	sql := `
		SELECT id, document_id, knowledge_base_id, chunk_index,
		       content, content_length, token_count,
		       start_page, end_page, heading, metadata
		FROM document_chunks
		WHERE knowledge_base_id = $1 AND content ILIKE $2
		LIMIT $3`

	rows, err := r.pool.Query(ctx, sql, kbID, "%"+query+"%", topK)
	if err != nil {
		return nil, fmt.Errorf("LIKE search failed: %w", err)
	}
	defer rows.Close()

	results := make([]domain.SearchResult, 0)
	for rows.Next() {
		chunk := domain.DocumentChunk{}
		if err := rows.Scan(
			&chunk.ID, &chunk.DocumentID, &chunk.KnowledgeBaseID, &chunk.ChunkIndex,
			&chunk.Content, &chunk.ContentLength, &chunk.TokenCount,
			&chunk.StartPage, &chunk.EndPage, &chunk.Heading, &chunk.Metadata,
		); err != nil {
			return nil, fmt.Errorf("failed to scan LIKE result: %w", err)
		}

		// Simple relevance score based on term frequency
		score := calculateBM25Score(query, chunk.Content)
		results = append(results, domain.SearchResult{
			Chunk:  chunk,
			Score:  score,
			Source: "keyword",
		})
	}

	return results, nil
}

// mergeResults merges and deduplicates results from vector and BM25 search.
// Uses Reciprocal Rank Fusion (RRF) for scoring.
func (r *HybridRetriever) mergeResults(vectorResults, bm25Results []domain.SearchResult, topK int) []domain.SearchResult {
	const rrfK = 60 // RRF constant

	// Build a map of chunk ID to result
 resultMap := make(map[uuid.UUID]*domain.SearchResult)
	rankMap := make(map[uuid.UUID]float64)

	// Add vector results
	for rank, result := range vectorResults {
		id := result.Chunk.ID
		if existing, ok := resultMap[id]; ok {
			existing.Score += result.Score
		} else {
			copy := result
			resultMap[id] = &copy
		}
		rankMap[id] += 1.0 / float64(rrfK+rank+1)
	}

	// Add BM25 results
	for rank, result := range bm25Results {
		id := result.Chunk.ID
		if existing, ok := resultMap[id]; ok {
			existing.Score += result.Score
			existing.Source = "hybrid"
		} else {
			copy := result
			resultMap[id] = &copy
		}
		rankMap[id] += 1.0 / float64(rrfK+rank+1)
	}

	// Apply RRF scores
	for id, result := range resultMap {
		result.Score = rankMap[id]
	}

	// Sort by score
	results := make([]domain.SearchResult, 0, len(resultMap))
	for _, r := range resultMap {
		results = append(results, *r)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit to topK
	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

// calculateBM25Score calculates a simple BM25-like score.
func calculateBM25Score(query, content string) float64 {
	// Simplified BM25 scoring
	queryTerms := tokenize(query)
	contentLower := toLower(content)

	score := 0.0
	k1 := 1.2
	b := 0.75
	avgDocLen := 500.0
	docLen := float64(len(content))

	for _, term := range queryTerms {
		// Count term frequency
		tf := float64(countOccurrences(contentLower, term))
		if tf == 0 {
			continue
		}

		// BM25 formula (simplified, no IDF)
		numerator := tf * (k1 + 1)
		denominator := tf + k1*(1-b+b*docLen/avgDocLen)
		score += numerator / denominator
	}

	return math.Min(score/10.0, 1.0) // Normalize to 0-1
}

func tokenize(s string) []string {
	words := []string{}
	current := []rune{}
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' || r == ',' || r == '.' {
			if len(current) > 0 {
				words = append(words, toLower(string(current)))
				current = []rune{}
			}
		} else {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		words = append(words, toLower(string(current)))
	}
	return words
}

func toLower(s string) string {
	result := []rune(s)
	for i, r := range result {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		}
	}
	return string(result)
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}

func vectorToString(v []float32) string {
	result := "["
	for i, f := range v {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", f)
	}
	result += "]"
	return result
}
