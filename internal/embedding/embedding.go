package embedding

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Store maps filenames to embedding vectors.
type Store map[string][]float64

// Result holds summary info from an embedding run.
type Result struct {
	Count          int     `json:"count"`
	MeanSimilarity float64 `json:"mean_similarity"`
}

// Generate creates embeddings for all WAV files in inputDir.
func Generate(inputDir, outputDir, model string) (*Result, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	files, err := findWAVFiles(inputDir)
	if err != nil {
		return nil, fmt.Errorf("find wav files: %w", err)
	}

	if len(files) == 0 {
		return &Result{}, nil
	}

	// Detect available Python embedding tool
	tool, err := detectTool(model)
	if err != nil {
		return nil, err
	}

	store := make(Store)
	for _, f := range files {
		emb, err := computeEmbedding(f, tool)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", filepath.Base(f), err)
			continue
		}
		store[filepath.Base(f)] = emb
	}

	// Save embeddings
	storePath := filepath.Join(outputDir, "embeddings.json")
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal embeddings: %w", err)
	}
	if err := os.WriteFile(storePath, data, 0644); err != nil {
		return nil, fmt.Errorf("write embeddings: %w", err)
	}

	// Compute mean self-similarity
	meanSim := MeanSelfSimilarity(store)

	return &Result{
		Count:          len(store),
		MeanSimilarity: meanSim,
	}, nil
}

// CompareToReference computes cosine similarity of each embedding to a reference file.
func CompareToReference(store Store, refPath, outputDir string, tool string) (map[string]float64, error) {
	refEmb, err := computeEmbedding(refPath, tool)
	if err != nil {
		return nil, fmt.Errorf("compute reference embedding: %w", err)
	}

	similarities := make(map[string]float64)
	for name, emb := range store {
		similarities[name] = CosineSimilarity(refEmb, emb)
	}

	simPath := filepath.Join(outputDir, "similarities.json")
	data, err := json.MarshalIndent(similarities, "", "  ")
	if err != nil {
		return nil, err
	}
	os.WriteFile(simPath, data, 0644)

	return similarities, nil
}

// CosineSimilarity computes the cosine similarity between two vectors.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// MeanSelfSimilarity computes the average pairwise cosine similarity.
func MeanSelfSimilarity(store Store) float64 {
	keys := make([]string, 0, len(store))
	for k := range store {
		keys = append(keys, k)
	}
	if len(keys) < 2 {
		return 1.0
	}

	var total float64
	var count int
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			total += CosineSimilarity(store[keys[i]], store[keys[j]])
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// LoadStore reads an embeddings JSON file.
func LoadStore(path string) (Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return store, nil
}

func findWAVFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".wav") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

func detectTool(preferred string) (string, error) {
	if preferred == "resemblyzer" || preferred == "" {
		if checkPython("from resemblyzer import VoiceEncoder") {
			return "resemblyzer", nil
		}
	}
	if preferred == "speechbrain" || preferred == "" {
		if checkPython("import speechbrain") {
			return "speechbrain", nil
		}
	}
	// If a specific model was requested but not found
	if preferred == "resemblyzer" {
		return "", fmt.Errorf("resemblyzer not installed.\nInstall with: pip install resemblyzer")
	}
	if preferred == "speechbrain" {
		return "", fmt.Errorf("speechbrain not installed.\nInstall with: pip install speechbrain")
	}
	return "", fmt.Errorf("no embedding tool available.\nInstall one of:\n  pip install resemblyzer\n  pip install speechbrain")
}

func checkPython(importStmt string) bool {
	cmd := exec.Command("python3", "-c", importStmt)
	return cmd.Run() == nil
}

func computeEmbedding(wavPath, tool string) ([]float64, error) {
	var script string
	switch tool {
	case "resemblyzer":
		script = fmt.Sprintf(`
import json, numpy as np
from resemblyzer import VoiceEncoder, preprocess_wav
encoder = VoiceEncoder()
wav = preprocess_wav("%s")
emb = encoder.embed_utterance(wav)
print(json.dumps(emb.tolist()))
`, wavPath)
	case "speechbrain":
		script = fmt.Sprintf(`
import json, torch
from speechbrain.inference.speaker import EncoderClassifier
classifier = EncoderClassifier.from_hparams(source="speechbrain/spkrec-ecapa-voxceleb")
emb = classifier.encode_batch(classifier.load_audio("%s"))
print(json.dumps(emb.squeeze().tolist()))
`, wavPath)
	default:
		return nil, fmt.Errorf("unknown embedding tool: %s", tool)
	}

	cmd := exec.Command("python3", "-c", script)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("python embedding: %w", err)
	}

	var emb []float64
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &emb); err != nil {
		return nil, fmt.Errorf("parse embedding: %w", err)
	}
	return emb, nil
}
