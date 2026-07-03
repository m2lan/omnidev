// Package service contains the business logic for the Gateway.
package service

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/omnidev/go-common/storage"

	"github.com/omnidev/gateway/internal/domain"
	"github.com/omnidev/gateway/internal/repository"
)

// UploadService handles file upload operations.
type UploadService struct {
	attRepo  repository.AttachmentRepository
	minioCli *storage.MinIO
}

// NewUploadService creates a new upload service.
func NewUploadService(attRepo repository.AttachmentRepository, minioCli *storage.MinIO) *UploadService {
	return &UploadService{
		attRepo:  attRepo,
		minioCli: minioCli,
	}
}

// UploadConfig defines upload constraints.
type UploadConfig struct {
	MaxFileSize  int64
	AllowedTypes map[string]bool
}

// DefaultUploadConfig returns the default upload configuration.
func DefaultUploadConfig() *UploadConfig {
	return &UploadConfig{
		MaxFileSize: 20 * 1024 * 1024, // 20MB
		AllowedTypes: map[string]bool{
			"image/jpeg": true,
			"image/png":  true,
			"image/gif":  true,
			"image/webp": true,
			"application/pdf":    true,
			"application/msword": true,
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
			"text/plain":    true,
			"text/markdown": true,
		},
	}
}

// UploadFile uploads a file to MinIO and creates an attachment record.
func (s *UploadService) UploadFile(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) (*domain.Attachment, error) {
	cfg := DefaultUploadConfig()

	// Validate file size
	if file.Size > cfg.MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d bytes", file.Size, cfg.MaxFileSize)
	}

	// Validate file type
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = detectMimeType(file.Filename)
	}
	if !cfg.AllowedTypes[mimeType] {
		return nil, fmt.Errorf("file type %s is not allowed", mimeType)
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Generate storage key
	dateStr := time.Now().Format("2006-01-02")
	fileID := uuid.New()
	cleanFilename := sanitizeFilename(file.Filename)
	storageKey := fmt.Sprintf("chat/%s/%s/%s/%s", userID.String(), dateStr, fileID.String(), cleanFilename)

	// Upload to MinIO
	if _, err := s.minioCli.Upload(ctx, "chat", storageKey, src, file.Size, mimeType); err != nil {
		return nil, fmt.Errorf("failed to upload file to storage: %w", err)
	}

	// Create attachment record
	att := &domain.Attachment{
		ID:         fileID,
		UserID:     userID,
		Filename:   file.Filename,
		MimeType:   mimeType,
		FileSize:   file.Size,
		StorageKey: storageKey,
		Metadata:   map[string]interface{}{},
	}

	// Try to extract image dimensions
	if att.IsImage() {
		if width, height, err := extractImageDimensions(file); err == nil {
			att.Width = &width
			att.Height = &height
		}
	}

	// Generate presigned URL
	presignedURL, err := s.minioCli.GetPresignedURL(ctx, "chat", storageKey, 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate file URL: %w", err)
	}
	att.StorageURL = presignedURL

	// Save to database
	if err := s.attRepo.Create(ctx, att); err != nil {
		return nil, fmt.Errorf("failed to save attachment record: %w", err)
	}

	return att, nil
}

// GetAttachment returns an attachment by ID.
func (s *UploadService) GetAttachment(ctx context.Context, userID uuid.UUID, attID uuid.UUID) (*domain.Attachment, error) {
	att, err := s.attRepo.GetByID(ctx, attID)
	if err != nil {
		return nil, err
	}
	if att.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Refresh presigned URL
	presignedURL, err := s.minioCli.GetPresignedURL(ctx, "chat", att.StorageKey, 7*24*time.Hour)
	if err == nil {
		att.StorageURL = presignedURL
	}

	return att, nil
}

// DeleteAttachment soft-deletes an attachment.
func (s *UploadService) DeleteAttachment(ctx context.Context, userID uuid.UUID, attID uuid.UUID) error {
	att, err := s.attRepo.GetByID(ctx, attID)
	if err != nil {
		return err
	}
	if att.UserID != userID {
		return fmt.Errorf("access denied")
	}
	return s.attRepo.Delete(ctx, attID)
}

// --- Helper functions ---

func detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeMap := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".txt":  "text/plain",
		".md":   "text/markdown",
	}
	if mime, ok := mimeMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

func extractImageDimensions(file *multipart.FileHeader) (int, int, error) {
	src, err := file.Open()
	if err != nil {
		return 0, 0, err
	}
	defer src.Close()

	cfg, _, err := image.DecodeConfig(src)
	if err != nil {
		return 0, 0, err
	}

	return cfg.Width, cfg.Height, nil
}
