// Package service provides business logic for the Chat Service.
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

	"github.com/omnidev/services/chat/internal/domain"
	"github.com/omnidev/services/chat/internal/repository"
)

// UploadService handles file upload operations.
type UploadService struct {
	attRepo  repository.AttachmentRepository
	minioCli *storage.MinIOClient
}

// NewUploadService creates a new upload service.
func NewUploadService(attRepo repository.AttachmentRepository, minioCli *storage.MinIOClient) *UploadService {
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
			// Images
			"image/jpeg": true,
			"image/png":  true,
			"image/gif":  true,
			"image/webp": true,
			// Documents
			"application/pdf":    true,
			"application/msword": true,
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
			"text/plain":    true,
			"text/markdown": true,
			// Code
			"text/x-python":     true,
			"text/javascript":   true,
			"application/x-javascript": true,
			"text/typescript":   true,
			"text/x-go":         true,
			"text/x-rust":       true,
			"text/x-java":       true,
			"text/x-c":          true,
			"text/x-c++":        true,
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

	// Generate storage key: chat/{user_id}/{yyyy-mm-dd}/{uuid}/{filename}
	dateStr := time.Now().Format("2006-01-02")
	fileID := uuid.New()
	cleanFilename := sanitizeFilename(file.Filename)
	storageKey := fmt.Sprintf("chat/%s/%s/%s/%s", userID.String(), dateStr, fileID.String(), cleanFilename)

	// Upload to MinIO
	if err := s.minioCli.Upload(ctx, storageKey, src, file.Size, mimeType); err != nil {
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

	// Generate presigned URL (7 days expiry)
	presignedURL, err := s.minioCli.PresignedGetObject(ctx, storageKey, 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate file URL: %w", err)
	}
	att.StorageURL = presignedURL.String()

	// Save to database
	if err := s.attRepo.Create(ctx, att); err != nil {
		return nil, fmt.Errorf("failed to save attachment record: %w", err)
	}

	return att, nil
}

// GetAttachment returns an attachment by ID with a fresh presigned URL.
func (s *UploadService) GetAttachment(ctx context.Context, userID uuid.UUID, attID string) (*domain.Attachment, error) {
	att, err := s.attRepo.GetByID(ctx, attID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if att.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Refresh presigned URL
	presignedURL, err := s.minioCli.PresignedGetObject(ctx, att.StorageKey, 7*24*time.Hour)
	if err == nil {
		att.StorageURL = presignedURL.String()
	}

	return att, nil
}

// GetAttachmentsByMessage returns all attachments for a message.
func (s *UploadService) GetAttachmentsByMessage(ctx context.Context, messageID string) ([]*domain.Attachment, error) {
	return s.attRepo.ListByMessage(ctx, messageID)
}

// AssociateWithMessage associates attachments with a message.
func (s *UploadService) AssociateWithMessage(ctx context.Context, attIDs []string, messageID string) error {
	return s.attRepo.UpdateMessageID(ctx, attIDs, messageID)
}

// DeleteAttachment soft-deletes an attachment.
func (s *UploadService) DeleteAttachment(ctx context.Context, userID uuid.UUID, attID string) error {
	att, err := s.attRepo.GetByID(ctx, attID)
	if err != nil {
		return err
	}
	if att.UserID != userID {
		return fmt.Errorf("access denied")
	}
	return s.attRepo.Delete(ctx, attID)
}

// GetPresignedURL generates a fresh presigned URL for an attachment.
func (s *UploadService) GetPresignedURL(ctx context.Context, userID uuid.UUID, attID string) (string, error) {
	att, err := s.attRepo.GetByID(ctx, attID)
	if err != nil {
		return "", err
	}
	if att.UserID != userID {
		return "", fmt.Errorf("access denied")
	}

	presignedURL, err := s.minioCli.PresignedGetObject(ctx, att.StorageKey, 7*24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

// --- Helper functions ---

// detectMimeType detects MIME type from file extension.
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
		".py":   "text/x-python",
		".js":   "text/javascript",
		".ts":   "text/typescript",
		".go":   "text/x-go",
		".rs":   "text/x-rust",
		".java": "text/x-java",
		".c":    "text/x-c",
		".cpp":  "text/x-c++",
		".h":    "text/x-c",
		".hpp":  "text/x-c++",
	}
	if mime, ok := mimeMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// sanitizeFilename removes unsafe characters from filename.
func sanitizeFilename(name string) string {
	// Replace path separators and unsafe chars
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

// extractImageDimensions extracts width and height from an image file header.
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
