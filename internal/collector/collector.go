package collector

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ParseResult holds the result of parsing a transcript file.
type ParseResult struct {
	TotalTokens  int64
	MessageCount int
	Model        string
	NewOffset    int64
}

// transcriptLine represents a single line in the JSONL transcript.
type transcriptLine struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

// assistantMessage represents the message field for assistant-type lines.
type assistantMessage struct {
	Model string `json:"model"`
	Usage struct {
		InputTokens               int64 `json:"input_tokens"`
		CacheCreationInputTokens  int64 `json:"cache_creation_input_tokens"`
		CacheReadInputTokens      int64 `json:"cache_read_input_tokens"`
		OutputTokens              int64 `json:"output_tokens"`
	} `json:"usage"`
}

// ParseTranscript reads a JSONL transcript file starting from fromOffset,
// accumulates token usage from assistant messages, and returns a ParseResult.
func ParseTranscript(path string, fromOffset int64) (*ParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("collector: open %q: %w", path, err)
	}
	defer f.Close()

	if fromOffset > 0 {
		if _, err := f.Seek(fromOffset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("collector: seek %q to %d: %w", path, fromOffset, err)
		}
	}

	const maxScanBuf = 1 * 1024 * 1024 // 1 MB
	scanner := bufio.NewScanner(f)
	buf := make([]byte, maxScanBuf)
	scanner.Buffer(buf, maxScanBuf)

	result := &ParseResult{}
	var bytesRead int64

	for scanner.Scan() {
		line := scanner.Bytes()
		bytesRead += int64(len(line)) + 1 // +1 for the newline

		var entry transcriptLine
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip malformed lines
			continue
		}

		if entry.Type != "assistant" {
			continue
		}

		var msg assistantMessage
		if err := json.Unmarshal(entry.Message, &msg); err != nil {
			continue
		}

		tokens := msg.Usage.InputTokens +
			msg.Usage.CacheCreationInputTokens +
			msg.Usage.CacheReadInputTokens +
			msg.Usage.OutputTokens

		result.TotalTokens += tokens
		result.MessageCount++
		if result.Model == "" && msg.Model != "" {
			result.Model = msg.Model
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("collector: scan %q: %w", path, err)
	}

	result.NewOffset = fromOffset + bytesRead
	return result, nil
}
