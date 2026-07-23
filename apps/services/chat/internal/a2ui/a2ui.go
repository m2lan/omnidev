// Package a2ui provides A2UI (Agent-to-UI) message parsing and detection utilities.
package a2ui

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// a2uiBlockRegex matches ```a2ui\n...\n``` code blocks.
var a2uiBlockRegex = regexp.MustCompile("(?s)```a2ui\\s*\\n(.*?)\\n```")

// A2UIMessage represents a parsed A2UI protocol message.
type A2UIMessage struct {
	Version        string                 `json:"version"`
	CreateSurface  map[string]interface{} `json:"createSurface,omitempty"`
	UpdateComponents map[string]interface{} `json:"updateComponents,omitempty"`
	UpdateDataModel  map[string]interface{} `json:"updateDataModel,omitempty"`
	DeleteSurface    map[string]interface{} `json:"deleteSurface,omitempty"`
}

// ExtractResult holds the result of extracting A2UI blocks from text.
type ExtractResult struct {
	// CleanText is the original text with A2UI blocks removed.
	CleanText string
	// Messages contains all parsed A2UI messages from the text.
	Messages []A2UIMessage
	// RawJSONs contains the raw JSON objects for each A2UI block.
	RawJSONs []map[string]interface{}
}

// ExtractA2UIBlocks finds and extracts all ```a2ui code blocks from the text.
// Returns the cleaned text (with blocks removed) and the parsed A2UI messages.
func ExtractA2UIBlocks(text string) ExtractResult {
	result := ExtractResult{
		CleanText: text,
	}

	matches := a2uiBlockRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return result
	}

	var cleanParts []string
	lastEnd := 0

	for _, match := range matches {
		// Add text before this block
		if match[0] > lastEnd {
			cleanParts = append(cleanParts, text[lastEnd:match[0]])
		}

		// Extract the JSON content
		jsonStr := strings.TrimSpace(text[match[2]:match[3]])

		var msg A2UIMessage
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &raw); err == nil {
			if err2 := json.Unmarshal([]byte(jsonStr), &msg); err2 == nil {
				result.Messages = append(result.Messages, msg)
				result.RawJSONs = append(result.RawJSONs, raw)
			}
		}

		lastEnd = match[1]
	}

	// Add remaining text
	if lastEnd < len(text) {
		cleanParts = append(cleanParts, text[lastEnd:])
	}

	result.CleanText = strings.Join(cleanParts, "")
	// Trim leading/trailing newlines from clean text
	result.CleanText = strings.TrimLeft(result.CleanText, "\n")

	return result
}

// IsA2UIBlock checks if the given text contains an A2UI code block.
func IsA2UIBlock(text string) bool {
	return a2uiBlockRegex.MatchString(text)
}

// HasCompleteA2UIBlock checks if there is at least one complete ```a2ui...``` block.
// Used during streaming to detect when a block has been fully received.
func HasCompleteA2UIBlock(text string) bool {
	return a2uiBlockRegex.MatchString(text)
}

// HasOpenA2UIBlock checks if there's an opening ```a2ui without a closing ```.
// Used during streaming to know we're still inside an A2UI block.
func HasOpenA2UIBlock(text string) bool {
	// Count ``` markers after "a2ui"
	idx := strings.LastIndex(text, "```a2ui")
	if idx < 0 {
		return false
	}
	// Check if there's a closing ``` after the opening ```a2ui
	rest := text[idx+7:] // skip "```a2ui" (7 chars)
	closeIdx := strings.Index(rest, "```")
	return closeIdx < 0
}

// EndsLikeA2UIPrefix checks if the text ends with a prefix of the "```a2ui" marker.
// Used during streaming to hold back text that might be the start of an A2UI block.
// For example, "```", "```a", "```a2", "```a2u", "```a2ui" all match.
func EndsLikeA2UIPrefix(text string) bool {
	marker := "```a2ui"
	for i := 1; i <= len(marker) && i <= len(text); i++ {
		if text[len(text)-i:] == marker[:i] {
			return true
		}
	}
	return false
}

// ValidateA2UIMessage checks if a raw JSON object looks like a valid A2UI message.
func ValidateA2UIMessage(raw map[string]interface{}) error {
	if raw == nil {
		return fmt.Errorf("nil message")
	}

	version, ok := raw["version"].(string)
	if !ok || version == "" {
		return fmt.Errorf("missing or invalid 'version' field")
	}

	hasSurface := raw["createSurface"] != nil ||
		raw["updateComponents"] != nil ||
		raw["updateDataModel"] != nil ||
		raw["deleteSurface"] != nil

	if !hasSurface {
		return fmt.Errorf("message must contain one of: createSurface, updateComponents, updateDataModel, deleteSurface")
	}

	return nil
}

// ToInterfaceSlice converts A2UI messages to []interface{} for use in domain.ChatChunk.
func ToInterfaceSlice(messages []A2UIMessage) []interface{} {
	result := make([]interface{}, 0, len(messages))
	for _, msg := range messages {
		result = append(result, msg)
	}
	return result
}

// RawToInterfaceSlice converts raw JSON maps to []interface{} for use in domain.ChatChunk.
func RawToInterfaceSlice(raws []map[string]interface{}) []interface{} {
	result := make([]interface{}, 0, len(raws))
	for _, raw := range raws {
		result = append(result, raw)
	}
	return result
}
