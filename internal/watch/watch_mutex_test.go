package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestProcessExistingConcurrent(t *testing.T) {
	dir := t.TempDir()

	// Create some ogg files (without transcripts, processing will fail due to no ffmpeg,
	// but we're testing that the mutex prevents concurrent access)
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, fmt.Sprintf("test%d.ogg", i))
		os.WriteFile(name, []byte("fake"), 0644)
	}

	w := &Watcher{
		Dir:            dir,
		Interval:       30 * time.Second,
		WhisperCommand: "false",
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.ProcessExisting()
		}()
	}
	wg.Wait()
	// Test passes if no race condition panic occurs
}
