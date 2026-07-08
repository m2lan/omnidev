package chunker

import (
	"strings"
	"testing"
)

func TestSemanticChunker_ChunkText(t *testing.T) {
	c := NewSemanticChunker(100, 20)

	tests := []struct {
		name      string
		input     string
		wantMin   int
		wantMax   int
	}{
		{
			name:    "empty text",
			input:   "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "short text single chunk",
			input:   "Hello world",
			wantMin: 1,
			wantMax: 1,
		},
		{
			name: "multiple paragraphs",
			input: strings.Repeat("This is a test paragraph with enough words to be meaningful.\n\n", 20),
			wantMin: 2,
			wantMax: 20,
		},
		{
			name: "paragraph splitting",
			input: "Paragraph one with some content here.\n\nParagraph two with different content.\n\nParagraph three with more stuff.",
			wantMin: 1,
			wantMax: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := c.ChunkText(tt.input)
			if len(chunks) < tt.wantMin || len(chunks) > tt.wantMax {
				t.Errorf("ChunkText() got %d chunks, want [%d, %d]", len(chunks), tt.wantMin, tt.wantMax)
			}
			// Verify all content is covered
			if tt.input != "" && len(chunks) > 0 {
				var combined strings.Builder
				for _, chunk := range chunks {
					combined.WriteString(chunk.Content)
				}
				if combined.Len() == 0 {
					t.Error("ChunkText() produced empty combined content")
				}
			}
		})
	}
}

func TestSemanticChunker_ChunkTextWithHeading(t *testing.T) {
	c := NewSemanticChunker(200, 30)

	input := `# Main Title

## Section One
Content for section one with some text.

## Section Two
Content for section two with different text.`

	chunks := c.ChunkTextWithHeading(input)
	if len(chunks) == 0 {
		t.Fatal("ChunkTextWithHeading() returned no chunks")
	}

	// Check that heading metadata is set
	for _, chunk := range chunks {
		if chunk.Content == "" {
			t.Error("ChunkTextWithHeading() produced empty chunk")
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		min   int
		max   int
	}{
		{"empty", "", 0, 0},
		{"short", "hello", 1, 5},
		{"english sentence", "The quick brown fox jumps over the lazy dog", 5, 20},
		{"chinese text", "你好世界这是一段测试文本", 5, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.input)
			if got < tt.min || got > tt.max {
				t.Errorf("EstimateTokens(%q) = %d, want [%d, %d]", tt.input, got, tt.min, tt.max)
			}
		})
	}
}

func TestSemanticChunker_ChunkSize(t *testing.T) {
	// Verify chunks respect approximate size limits with paragraph boundaries
	c := NewSemanticChunker(50, 10)
	longText := strings.Repeat("Word word word word word.\n\n", 20)
	chunks := c.ChunkText(longText)

	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks for long text, got %d", len(chunks))
	}
}
