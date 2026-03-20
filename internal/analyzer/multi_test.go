package analyzer

import (
	"testing"
)

func TestJoinAndTruncate(t *testing.T) {
	samples := []string{"hello world", "foo bar baz", "testing 123"}

	result := joinAndTruncate(samples, 1000)
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test truncation
	result = joinAndTruncate(samples, 15)
	if len(result) > 15 {
		t.Errorf("result length %d > max 15", len(result))
	}
}

func TestJoinAndTruncate_Empty(t *testing.T) {
	result := joinAndTruncate(nil, 1000)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
