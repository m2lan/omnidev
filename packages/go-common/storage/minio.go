// Package storage provides MinIO object storage management.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"

	"github.com/omnidev/go-common/config"
	"github.com/omnidev/go-common/logger"
)

// MinIO wraps the MinIO client with convenience methods.
type MinIO struct {
	client       *minio.Client
	bucketPrefix string
}

// NewMinIO creates a new MinIO client.
func NewMinIO(cfg config.MinIOConfig) (*MinIO, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	logger.Log.Info("MinIO connected",
		zap.String("endpoint", cfg.Endpoint),
		zap.Bool("ssl", cfg.UseSSL),
	)

	return &MinIO{
		client:       client,
		bucketPrefix: cfg.BucketPrefix,
	}, nil
}

// bucketName returns the full bucket name with prefix.
func (m *MinIO) bucketName(name string) string {
	if m.bucketPrefix != "" {
		return fmt.Sprintf("%s-%s", m.bucketPrefix, name)
	}
	return name
}

// EnsureBucket creates a bucket if it doesn't exist.
func (m *MinIO) EnsureBucket(ctx context.Context, name string) error {
	bucket := m.bucketName(name)
	exists, err := m.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		if err := m.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
		}
		logger.Log.Info("Bucket created", zap.String("bucket", bucket))
	}
	return nil
}

// Upload uploads an object to MinIO.
func (m *MinIO) Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) (string, error) {
	bucketName := m.bucketName(bucket)

	// Ensure bucket exists
	if err := m.EnsureBucket(ctx, bucket); err != nil {
		return "", err
	}

	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	info, err := m.client.PutObject(ctx, bucketName, key, reader, size, opts)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	logger.Log.Debug("Object uploaded",
		zap.String("bucket", info.Bucket),
		zap.String("key", info.Key),
		zap.Int64("size", info.Size),
	)

	return fmt.Sprintf("%s/%s", bucketName, key), nil
}

// UploadBytes uploads byte data to MinIO.
func (m *MinIO) UploadBytes(ctx context.Context, bucket, key string, data []byte, contentType string) (string, error) {
	return m.Upload(ctx, bucket, key, bytes.NewReader(data), int64(len(data)), contentType)
}

// Download downloads an object from MinIO.
func (m *MinIO) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	bucketName := m.bucketName(bucket)
	object, err := m.client.GetObject(ctx, bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}
	return object, nil
}

// DownloadBytes downloads an object as bytes.
func (m *MinIO) DownloadBytes(ctx context.Context, bucket, key string) ([]byte, error) {
	reader, err := m.Download(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// Delete deletes an object from MinIO.
func (m *MinIO) Delete(ctx context.Context, bucket, key string) error {
	bucketName := m.bucketName(bucket)
	if err := m.client.RemoveObject(ctx, bucketName, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GetPresignedURL generates a presigned URL for an object.
func (m *MinIO) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	bucketName := m.bucketName(bucket)
	url, err := m.client.PresignedGetObject(ctx, bucketName, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

// GetPresignedUploadURL generates a presigned URL for uploading.
func (m *MinIO) GetPresignedUploadURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	bucketName := m.bucketName(bucket)
	url, err := m.client.PresignedPutObject(ctx, bucketName, key, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}
	return url.String(), nil
}

// ListObjects lists objects in a bucket with a prefix.
func (m *MinIO) ListObjects(ctx context.Context, bucket, prefix string) ([]minio.ObjectInfo, error) {
	bucketName := m.bucketName(bucket)
	var objects []minio.ObjectInfo

	for obj := range m.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", obj.Err)
		}
		objects = append(objects, obj)
	}

	return objects, nil
}

// GetClient returns the underlying MinIO client.
func (m *MinIO) GetClient() *minio.Client {
	return m.client
}
