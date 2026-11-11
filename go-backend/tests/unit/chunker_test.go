package service_test

import (
	"strings"
	"testing"

	"github.com/aiStudySaathi/backend/internal/service/ai"
)

func TestChunkerSplitsLongText(t *testing.T) {
	chunker := &ai.Chunker{
		ChunkSize:    50,
		ChunkOverlap: 10,
		Separators:   []string{"\n\n", "\n", " ", ""},
	}

	text := "First paragraph is here.\n\nSecond paragraph contains more detailed info.\n\nThird paragraph finishes it."
	chunks := chunker.SplitText(text)

	if len(chunks) < 2 {
		t.Fatalf("expected text to be split into multiple chunks, got: %d", len(chunks))
	}

	for i, ch := range chunks {
		if len(ch) > 70 { // Allowing slight leeway for separator boundary
			t.Errorf("chunk %d exceeds expected limit: %d chars (%q)", i, len(ch), ch)
		}
	}
}

func TestChunkerEmptyString(t *testing.T) {
	chunker := ai.NewDefaultChunker()
	chunks := chunker.SplitText("   ")
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty string, got: %d", len(chunks))
	}
}

func TestChunkerShortString(t *testing.T) {
	chunker := ai.NewDefaultChunker()
	text := "Short text under 1000 chars."
	chunks := chunker.SplitText(text)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got: %d", len(chunks))
	}
	if strings.TrimSpace(chunks[0]) != text {
		t.Errorf("expected %q, got %q", text, chunks[0])
	}
}

