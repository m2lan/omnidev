// Package parser provides document parsing capabilities using Apache Tika.
package parser

import (
	"context"
	"io"
)

// ParseResult represents the result of parsing a document.
type ParseResult struct {
	Content  string                 `json:"content"`           // Extracted text content
	Pages    int                    `json:"pages"`             // Number of pages (if applicable)
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Document metadata
}

// Parser defines the interface for document parsers.
type Parser interface {
	// Parse parses a document and extracts text content.
	Parse(ctx context.Context, filename string, reader io.Reader) (*ParseResult, error)

	// SupportedFormats returns a list of supported file extensions.
	SupportedFormats() []string
}

// FormatCategory represents the category of a file format.
type FormatCategory string

const (
	FormatCategoryDocument FormatCategory = "document"
	FormatCategorySpreadsheet FormatCategory = "spreadsheet"
	FormatCategoryPresentation FormatCategory = "presentation"
	FormatCategoryImage FormatCategory = "image"
	FormatCategoryCode FormatCategory = "code"
	FormatCategoryText FormatCategory = "text"
	FormatCategoryOther FormatCategory = "other"
)

// GetFormatCategory returns the category of a file format based on extension.
func GetFormatCategory(ext string) FormatCategory {
	switch ext {
	case ".pdf", ".doc", ".docx", ".odt", ".rtf":
		return FormatCategoryDocument
	case ".xls", ".xlsx", ".csv", ".ods":
		return FormatCategorySpreadsheet
	case ".ppt", ".pptx", ".odp":
		return FormatCategoryPresentation
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff":
		return FormatCategoryImage
	case ".go", ".py", ".js", ".ts", ".java", ".rs", ".cpp", ".c", ".rb", ".php", ".swift", ".kt":
		return FormatCategoryCode
	case ".txt", ".md", ".markdown", ".log":
		return FormatCategoryText
	default:
		return FormatCategoryOther
	}
}
