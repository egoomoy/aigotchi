package collector

import (
	"testing"
)

const samplePath = "testdata/sample.jsonl"

func TestParseTranscript_FromZero(t *testing.T) {
	result, err := ParseTranscript(samplePath, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalTokens != 2000 {
		t.Errorf("TotalTokens = %d, want 2000", result.TotalTokens)
	}
	if result.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", result.MessageCount)
	}
	if result.NewOffset <= 0 {
		t.Errorf("NewOffset = %d, want > 0", result.NewOffset)
	}
}

func TestParseTranscript_FromNewOffset_NoMoreTokens(t *testing.T) {
	first, err := ParseTranscript(samplePath, 0)
	if err != nil {
		t.Fatalf("first pass error: %v", err)
	}

	second, err := ParseTranscript(samplePath, first.NewOffset)
	if err != nil {
		t.Fatalf("second pass error: %v", err)
	}

	if second.TotalTokens != 0 {
		t.Errorf("second pass TotalTokens = %d, want 0", second.TotalTokens)
	}
	if second.MessageCount != 0 {
		t.Errorf("second pass MessageCount = %d, want 0", second.MessageCount)
	}
}

func TestParseTranscript_OnlyCountsAssistantMessages(t *testing.T) {
	result, err := ParseTranscript(samplePath, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The sample file has 4 lines: 2 human and 2 assistant.
	// Only the 2 assistant messages should be counted.
	if result.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2 (only assistant messages)", result.MessageCount)
	}
}
