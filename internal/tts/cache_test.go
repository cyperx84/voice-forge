package tts

import (
	"testing"
)

func TestCache_PutGet(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{Dir: dir}

	audio := []byte("test audio data")
	if err := cache.Put("hello", "cyperx", "chatterbox", audio); err != nil {
		t.Fatalf("Put error: %v", err)
	}

	got, ok := cache.Get("hello", "cyperx", "chatterbox")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != string(audio) {
		t.Fatalf("cache returned %q, want %q", got, audio)
	}
}

func TestCache_Miss(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{Dir: dir}

	_, ok := cache.Get("nonexistent", "voice", "backend")
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestCache_EmptyDir(t *testing.T) {
	cache := &Cache{Dir: ""}

	// Should not error, just no-op
	if err := cache.Put("text", "voice", "backend", []byte("data")); err != nil {
		t.Fatalf("Put with empty dir should not error, got: %v", err)
	}

	_, ok := cache.Get("text", "voice", "backend")
	if ok {
		t.Fatal("expected miss with empty dir")
	}
}

func TestCacheKey_Deterministic(t *testing.T) {
	k1 := cacheKey("hello", "voice", "backend")
	k2 := cacheKey("hello", "voice", "backend")
	if k1 != k2 {
		t.Fatalf("cache key not deterministic: %s != %s", k1, k2)
	}

	k3 := cacheKey("different", "voice", "backend")
	if k1 == k3 {
		t.Fatal("different text should produce different keys")
	}
}
