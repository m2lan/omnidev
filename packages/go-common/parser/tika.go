// Package parser provides document parsing capabilities using Apache Tika.
package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/omnidev/go-common/logger"
)

// TikaConfig holds Tika client configuration.
type TikaConfig struct {
	Endpoint string `mapstructure:"endpoint"` // e.g., http://localhost:9998
	Timeout  int    `mapstructure:"timeout"`  // Request timeout in seconds
}

// TikaClient implements Parser using Apache Tika REST API.
type TikaClient struct {
	endpoint string
	client   *http.Client
}

// NewTikaClient creates a new Tika client.
func NewTikaClient(cfg TikaConfig) *TikaClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60
	}

	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	if endpoint == "" {
		endpoint = "http://localhost:9998"
	}

	logger.Log.Info("Tika client initialized",
		zap.String("endpoint", endpoint),
		zap.Int("timeout", timeout),
	)

	return &TikaClient{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Parse parses a document using Tika and extracts text content.
func (t *TikaClient) Parse(ctx context.Context, filename string, reader io.Reader) (*ParseResult, error) {
	// Detect MIME type from filename
	mimeType := detectMIMEType(filename)

	// Call Tika API to extract text
	content, err := t.extractText(ctx, reader, mimeType)
	if err != nil {
		return nil, fmt.Errorf("tika parse failed: %w", err)
	}

	// Metadata extraction is optional - skip for now to avoid 422 errors
	// TODO: Fix metadata extraction (Tika /meta returns XML, not JSON)
	metadata := map[string]interface{}{}

	return &ParseResult{
		Content:  content,
		Pages:    0, // TODO: Extract page count from text or metadata
		Metadata: metadata,
	}, nil
}

// SupportedFormats returns the list of supported file extensions.
func (t *TikaClient) SupportedFormats() []string {
	return []string{
		".pdf", ".doc", ".docx", ".odt", ".rtf",
		".xls", ".xlsx", ".csv", ".ods",
		".ppt", ".pptx", ".odp",
		".txt", ".md", ".markdown",
		".html", ".htm",
		".json", ".xml",
		".epub",
	}
}

// extractText calls Tika /tika endpoint to extract text.
func (t *TikaClient) extractText(ctx context.Context, reader io.Reader, mimeType string) (string, error) {
	url := t.endpoint + "/tika"

	req, err := http.NewRequestWithContext(ctx, "PUT", url, reader)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("Accept", "text/plain")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tika request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tika returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// extractMetadata calls Tika /meta endpoint to extract metadata.
func (t *TikaClient) extractMetadata(ctx context.Context, reader io.Reader, mimeType string) (map[string]interface{}, error) {
	url := t.endpoint + "/meta"

	req, err := http.NewRequestWithContext(ctx, "PUT", url, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tika request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tika returned status %d", resp.StatusCode)
	}

	var metadata map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return metadata, nil
}

// Health checks if Tika is available.
func (t *TikaClient) Health(ctx context.Context) error {
	url := t.endpoint + "/version"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("tika health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tika returned status %d", resp.StatusCode)
	}

	return nil
}

// detectMIMEType detects MIME type from filename.
func detectMIMEType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeMap := map[string]string{
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".odt":  "application/vnd.oasis.opendocument.text",
		".rtf":  "application/rtf",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".csv":  "text/csv",
		".ods":  "application/vnd.oasis.opendocument.spreadsheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".odp":  "application/vnd.oasis.opendocument.presentation",
		".txt":  "text/plain",
		".md":   "text/markdown",
		".html": "text/html",
		".htm":  "text/html",
		".json": "application/json",
		".xml":  "application/xml",
		".epub": "application/epub+zip",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".tiff": "image/tiff",
		".bmp":  "image/bmp",
	}

	if mime, ok := mimeMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}
