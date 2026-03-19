package embedding

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []float64
		b    []float64
		want float64
	}{
		{
			name: "identical vectors",
			a:    []float64{1, 0, 0},
			b:    []float64{1, 0, 0},
			want: 1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float64{1, 0, 0},
			b:    []float64{0, 1, 0},
			want: 0.0,
		},
		{
			name: "opposite vectors",
			a:    []float64{1, 0, 0},
			b:    []float64{-1, 0, 0},
			want: -1.0,
		},
		{
			name: "similar vectors",
			a:    []float64{1, 2, 3},
			b:    []float64{1, 2, 3.1},
			want: 0.9999, // Very close to 1
		},
		{
			name: "empty vectors",
			a:    []float64{},
			b:    []float64{},
			want: 0.0,
		},
		{
			name: "different lengths",
			a:    []float64{1, 2},
			b:    []float64{1, 2, 3},
			want: 0.0,
		},
		{
			name: "zero vector",
			a:    []float64{0, 0, 0},
			b:    []float64{1, 2, 3},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CosineSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("CosineSimilarity() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestMeanSelfSimilarity(t *testing.T) {
	store := Store{
		"a.wav": {1, 0, 0},
		"b.wav": {1, 0, 0},
		"c.wav": {1, 0, 0},
	}

	sim := MeanSelfSimilarity(store)
	if math.Abs(sim-1.0) > 0.001 {
		t.Errorf("identical vectors should have similarity 1.0, got %f", sim)
	}
}

func TestMeanSelfSimilaritySingleFile(t *testing.T) {
	store := Store{
		"a.wav": {1, 0, 0},
	}

	sim := MeanSelfSimilarity(store)
	if sim != 1.0 {
		t.Errorf("single file should have similarity 1.0, got %f", sim)
	}
}

func TestMeanSelfSimilarityOrthogonal(t *testing.T) {
	store := Store{
		"a.wav": {1, 0, 0},
		"b.wav": {0, 1, 0},
		"c.wav": {0, 0, 1},
	}

	sim := MeanSelfSimilarity(store)
	if math.Abs(sim) > 0.001 {
		t.Errorf("orthogonal vectors should have similarity ~0, got %f", sim)
	}
}

func TestStoreLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "embeddings.json")

	original := Store{
		"a.wav": {0.1, 0.2, 0.3, 0.4},
		"b.wav": {0.5, 0.6, 0.7, 0.8},
	}

	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(storePath, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	loaded, err := LoadStore(storePath)
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 entries, got %d", len(loaded))
	}

	aEmb, ok := loaded["a.wav"]
	if !ok {
		t.Fatal("missing a.wav")
	}
	if len(aEmb) != 4 {
		t.Errorf("expected 4-d embedding, got %d", len(aEmb))
	}
	if aEmb[0] != 0.1 {
		t.Errorf("aEmb[0] = %f, want 0.1", aEmb[0])
	}
}

func TestLoadStoreNotFound(t *testing.T) {
	_, err := LoadStore("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
