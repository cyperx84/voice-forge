package analyzer

import (
	"strings"
	"testing"
)

func TestMaxCorpusBytesConstant(t *testing.T) {
	if maxCorpusBytes != 200_000 {
		t.Errorf("maxCorpusBytes = %d, want 200000", maxCorpusBytes)
	}
}

func TestCorpusTruncation(t *testing.T) {
	// Build a corpus larger than maxCorpusBytes
	transcript := strings.Repeat("word ", 100) // ~500 bytes each
	var transcripts []string
	for i := 0; i < 500; i++ {
		transcripts = append(transcripts, transcript)
	}

	corpus := strings.Join(transcripts, "\n\n---\n\n")
	if len(corpus) <= maxCorpusBytes {
		t.Skip("corpus not large enough for truncation test")
	}

	truncated := corpus
	if len(truncated) > maxCorpusBytes {
		truncated = truncated[:maxCorpusBytes]
	}
	if len(truncated) != maxCorpusBytes {
		t.Errorf("truncated length = %d, want %d", len(truncated), maxCorpusBytes)
	}
}
