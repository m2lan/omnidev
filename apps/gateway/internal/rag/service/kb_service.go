// Package service contains the business logic for the RAG Service.
package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appErr "github.com/omnidev/go-common/errors"
	"github.com/omnidev/go-common/logger"
	"github.com/omnidev/go-common/storage"

	"github.com/omnidev/gateway/internal/rag/chunker"
	"github.com/omnidev/gateway/internal/rag/domain"
	"github.com/omnidev/gateway/internal/rag/embedder"
	"github.com/omnidev/gateway/internal/rag/parser"
	"github.com/omnidev/gateway/internal/rag/repository"
)

// KnowledgeBaseService handles knowledge base operations.
type KnowledgeBaseService struct {
	kbRepo     repository.KnowledgeBaseRepository
	docRepo    repository.DocumentRepository
	chunkRepo  repository.ChunkRepository
	minio      *storage.MinIO
	parser     *parser.DocParser
	chunker    *chunker.SemanticChunker
	embedder   embedder.Embedder
}

// NewKnowledgeBaseService creates a new knowledge base service.
func NewKnowledgeBaseService(
	kbRepo repository.KnowledgeBaseRepository,
	docRepo repository.DocumentRepository,
	chunkRepo repository.ChunkRepository,
	minio *storage.MinIO,
	parser *parser.DocParser,
	chunker *chunker.SemanticChunker,
	embedder embedder.Embedder,
) *KnowledgeBaseService {
	return &KnowledgeBaseService{
		kbRepo:    kbRepo,
		docRepo:   docRepo,
		chunkRepo: chunkRepo,
		minio:     minio,
		parser:    parser,
		chunker:   chunker,
		embedder:  embedder,
	}
}

// CreateKBInput defines the input for creating a knowledge base.
type CreateKBInput struct {
	Name           string                 `json:"name" validate:"required,min=1,max=100"`
	Description    string                 `json:"description"`
	EmbeddingModel string                 `json:"embedding_model"`
	ChunkSize      int                    `json:"chunk_size"`
	ChunkOverlap   int                    `json:"chunk_overlap"`
	Settings       map[string]interface{} `json:"settings"`
}

// CreateKnowledgeBase creates a new knowledge base.
func (s *KnowledgeBaseService) CreateKnowledgeBase(ctx context.Context, userID uuid.UUID, input *CreateKBInput) (*domain.KnowledgeBase, error) {
	embeddingModel := input.EmbeddingModel
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small"
	}
	chunkSize := input.ChunkSize
	if chunkSize == 0 {
		chunkSize = 512
	}
	chunkOverlap := input.ChunkOverlap
	if chunkOverlap == 0 {
		chunkOverlap = 50
	}

	kb := &domain.KnowledgeBase{
		ID:             uuid.New(),
		UserID:         userID,
		Name:           input.Name,
		EmbeddingModel: embeddingModel,
		ChunkSize:      chunkSize,
		ChunkOverlap:   chunkOverlap,
		Settings:       input.Settings,
		Status:         "active",
	}

	if input.Description != "" {
		kb.Description = &input.Description
	}
	if kb.Settings == nil {
		kb.Settings = map[string]interface{}{}
	}

	if err := s.kbRepo.Create(ctx, kb); err != nil {
		return nil, appErr.Wrap(err, "failed to create knowledge base")
	}

	return kb, nil
}

// GetKnowledgeBase returns a knowledge base by ID.
func (s *KnowledgeBaseService) GetKnowledgeBase(ctx context.Context, userID, kbID uuid.UUID) (*domain.KnowledgeBase, error) {
	kb, err := s.kbRepo.GetByID(ctx, kbID)
	if err != nil {
		return nil, appErr.NotFound("knowledge base")
	}
	if kb.UserID != userID {
		return nil, appErr.ErrForbidden
	}
	return kb, nil
}

// ListKnowledgeBases returns a paginated list of knowledge bases.
func (s *KnowledgeBaseService) ListKnowledgeBases(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*domain.KnowledgeBase, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.kbRepo.List(ctx, userID, offset, pageSize)
}

// UpdateKnowledgeBase updates a knowledge base.
func (s *KnowledgeBaseService) UpdateKnowledgeBase(ctx context.Context, userID, kbID uuid.UUID, input *CreateKBInput) (*domain.KnowledgeBase, error) {
	kb, err := s.kbRepo.GetByID(ctx, kbID)
	if err != nil {
		return nil, appErr.NotFound("knowledge base")
	}
	if kb.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	update := &repository.KBUpdate{
		Name:         &input.Name,
		ChunkSize:    &input.ChunkSize,
		ChunkOverlap: &input.ChunkOverlap,
	}
	if input.Description != "" {
		update.Description = &input.Description
	}

	if err := s.kbRepo.Update(ctx, kbID, update); err != nil {
		return nil, appErr.Wrap(err, "failed to update knowledge base")
	}

	return s.kbRepo.GetByID(ctx, kbID)
}

// DeleteKnowledgeBase deletes a knowledge base and all its documents.
func (s *KnowledgeBaseService) DeleteKnowledgeBase(ctx context.Context, userID, kbID uuid.UUID) error {
	kb, err := s.kbRepo.GetByID(ctx, kbID)
	if err != nil {
		return appErr.NotFound("knowledge base")
	}
	if kb.UserID != userID {
		return appErr.ErrForbidden
	}

	return s.kbRepo.Delete(ctx, kbID)
}

// UploadDocument uploads and processes a document.
func (s *KnowledgeBaseService) UploadDocument(ctx context.Context, userID, kbID uuid.UUID, filename string, fileSize int64, reader interface{ Read([]byte) (int, error) }) (*domain.Document, error) {
	// Verify ownership
	kb, err := s.kbRepo.GetByID(ctx, kbID)
	if err != nil {
		return nil, appErr.NotFound("knowledge base")
	}
	if kb.UserID != userID {
		return nil, appErr.ErrForbidden
	}

	// Determine file type
	fileType := strings.TrimPrefix(filepath.Ext(filename), ".")

	// Upload to MinIO
	key := fmt.Sprintf("%s/%s/%s", kbID.String(), uuid.New().String(), filename)
	uploadReader, ok := reader.(interface{ Read([]byte) (int, error) })
	if !ok {
		return nil, appErr.Validation("invalid file reader")
	}

	// Upload returns presigned URL, but we store the object key for later download
	_, err = s.minio.Upload(ctx, "rag-documents", key, uploadReader, fileSize, "application/octet-stream")
	if err != nil {
		return nil, appErr.Wrap(err, "failed to upload file")
	}

	// Create document record
	doc := &domain.Document{
		ID:             uuid.New(),
		KnowledgeBaseID: kbID,
		Filename:       filename,
		FileType:       fileType,
		FileSize:       fileSize,
		FileURL:        key, // Store object key, not presigned URL
		Status:         domain.DocumentStatusUploading,
		Metadata:       map[string]interface{}{},
	}

	if err := s.docRepo.Create(ctx, doc); err != nil {
		return nil, appErr.Wrap(err, "failed to create document record")
	}

	// Process document asynchronously with timeout
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		s.processDocument(ctx, kb, doc)
	}()

	return doc, nil
}

// processDocument processes a document: parse → chunk → embed → store.
func (s *KnowledgeBaseService) processDocument(ctx context.Context, kb *domain.KnowledgeBase, doc *domain.Document) {
	logger.Log.Info("Processing document", zap.String("doc_id", doc.ID.String()), zap.String("filename", doc.Filename))

	// Update status to processing
	_ = s.docRepo.UpdateStatus(ctx, doc.ID, domain.DocumentStatusProcessing, nil)

	// Download file from MinIO
	reader, err := s.minio.Download(ctx, "rag-documents", doc.FileURL)
	if err != nil {
		errMsg := fmt.Sprintf("failed to download file: %v", err)
		_ = s.docRepo.UpdateStatus(ctx, doc.ID, domain.DocumentStatusFailed, &errMsg)
		return
	}
	defer reader.Close()

	// Parse document
	parseResult, err := s.parser.Parse(doc.Filename, reader)
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse document: %v", err)
		_ = s.docRepo.UpdateStatus(ctx, doc.ID, domain.DocumentStatusFailed, &errMsg)
		return
	}

	// Chunk text
	chunks := s.chunker.ChunkText(parseResult.Content)
	if len(chunks) == 0 {
		errMsg := "document produced no chunks"
		_ = s.docRepo.UpdateStatus(ctx, doc.ID, domain.DocumentStatusFailed, &errMsg)
		return
	}

	// Generate embeddings
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	vectors, err := s.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		errMsg := fmt.Sprintf("failed to generate embeddings: %v", err)
		_ = s.docRepo.UpdateStatus(ctx, doc.ID, domain.DocumentStatusFailed, &errMsg)
		return
	}

	// Build document chunks
	docChunks := make([]domain.DocumentChunk, len(chunks))
	totalTokens := int64(0)
	for i, chunk := range chunks {
		// Ensure chunk content is valid UTF-8 before storing
		content := chunk.Content
		if !utf8.ValidString(content) {
			content = strings.ToValidUTF8(content, "")
		}

		tokenCount := chunker.EstimateTokens(content)
		totalTokens += int64(tokenCount)

		docChunks[i] = domain.DocumentChunk{
			ID:              uuid.New(),
			DocumentID:      doc.ID,
			KnowledgeBaseID: kb.ID,
			ChunkIndex:      i,
			Content:         content,
			ContentLength:   len(content),
			TokenCount:      tokenCount,
			Metadata:        chunk.Metadata,
			Embedding:       vectors[i],
		}
	}

	// Store chunks
	if err := s.chunkRepo.CreateBatch(ctx, docChunks); err != nil {
		errMsg := fmt.Sprintf("failed to store chunks: %v", err)
		_ = s.docRepo.UpdateStatus(ctx, doc.ID, domain.DocumentStatusFailed, &errMsg)
		return
	}

	// Update document stats
	_ = s.docRepo.UpdateStats(ctx, doc.ID, len(chunks), totalTokens)
	_ = s.docRepo.UpdateStatus(ctx, doc.ID, domain.DocumentStatusReady, nil)

	// Update knowledge base stats
	chunkCount, _ := s.chunkRepo.CountByKB(ctx, kb.ID)
	allTokens, _ := s.chunkRepo.TotalTokensByKB(ctx, kb.ID)
	_ = s.kbRepo.UpdateStats(ctx, kb.ID, kb.DocCount+1, chunkCount, allTokens)

	logger.Log.Info("Document processed",
		zap.String("doc_id", doc.ID.String()),
		zap.Int("chunks", len(chunks)),
		zap.Int64("tokens", totalTokens),
	)
}

// ListDocuments returns documents in a knowledge base.
func (s *KnowledgeBaseService) ListDocuments(ctx context.Context, userID, kbID uuid.UUID, page, pageSize int) ([]*domain.Document, int, error) {
	kb, err := s.kbRepo.GetByID(ctx, kbID)
	if err != nil {
		return nil, 0, appErr.NotFound("knowledge base")
	}
	if kb.UserID != userID {
		return nil, 0, appErr.ErrForbidden
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	return s.docRepo.ListByKB(ctx, kbID, offset, pageSize)
}

// DeleteDocument deletes a document and its chunks.
func (s *KnowledgeBaseService) DeleteDocument(ctx context.Context, userID, kbID, docID uuid.UUID) error {
	kb, err := s.kbRepo.GetByID(ctx, kbID)
	if err != nil {
		return appErr.NotFound("knowledge base")
	}
	if kb.UserID != userID {
		return appErr.ErrForbidden
	}

	// Delete chunks
	_ = s.chunkRepo.DeleteByDocument(ctx, docID)

	// Delete document
	return s.docRepo.Delete(ctx, docID)
}
