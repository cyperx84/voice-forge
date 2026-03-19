package cmd

import (
	"strings"
	"testing"
)

func TestSpeakTextValidation(t *testing.T) {
	tests := []struct {
		input string
		empty bool
	}{
		{"hello world", false},
		{"", true},
		{"   ", true},
		{"\t\n", true},
		{"a", false},
	}
	for _, tt := range tests {
		got := strings.TrimSpace(tt.input) == ""
		if got != tt.empty {
			t.Errorf("TrimSpace(%q)==\"\" = %v, want %v", tt.input, got, tt.empty)
		}
	}
}
