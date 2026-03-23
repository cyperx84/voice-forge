package tts

import (
	"fmt"
	"os"
	"sync"
)

// ScriptOpts configures parallel multi-line TTS generation.
type ScriptOpts struct {
	Workers    int                              // max concurrent workers (default: 4)
	OnProgress func(done, total int, line string) // called as each line completes
}

// lineResult holds the output of a single line's generation.
type lineResult struct {
	Index int
	Audio []byte
	Err   error
}

// SpeakScript generates audio for multiple lines in parallel, returning
// the ordered audio segments. Use audioout.ConcatAudio to join them.
func SpeakScript(lines []string, backend Backend, opts SpeakOpts, scriptOpts ScriptOpts) ([][]byte, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("no lines to speak")
	}

	workers := scriptOpts.Workers
	if workers <= 0 {
		workers = 4
	}

	results := make([]lineResult, len(lines))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0

	for i, line := range lines {
		wg.Add(1)
		go func(idx int, text string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			lineOpts := opts
			lineOpts.OutputPath = "" // generate to temp

			audio, err := backend.Speak(text, lineOpts)
			results[idx] = lineResult{Index: idx, Audio: audio, Err: err}

			if scriptOpts.OnProgress != nil {
				mu.Lock()
				done++
				scriptOpts.OnProgress(done, len(lines), text)
				mu.Unlock()
			}
		}(i, line)
	}

	wg.Wait()

	// Collect results in order
	segments := make([][]byte, len(lines))
	for i, r := range results {
		if r.Err != nil {
			return nil, fmt.Errorf("line %d failed: %w", i+1, r.Err)
		}
		segments[i] = r.Audio
	}

	return segments, nil
}

// SpeakScriptToFiles generates audio for multiple lines in parallel,
// writing each to a temp file. Returns the ordered file paths.
func SpeakScriptToFiles(lines []string, backend Backend, opts SpeakOpts, scriptOpts ScriptOpts) ([]string, error) {
	segments, err := SpeakScript(lines, backend, opts, scriptOpts)
	if err != nil {
		return nil, err
	}

	var paths []string
	for i, audio := range segments {
		f, err := os.CreateTemp("", fmt.Sprintf("forge-line-%03d-*.wav", i))
		if err != nil {
			// Clean up already-created files
			for _, p := range paths {
				os.Remove(p)
			}
			return nil, fmt.Errorf("creating temp file for line %d: %w", i+1, err)
		}
		if _, err := f.Write(audio); err != nil {
			f.Close()
			for _, p := range paths {
				os.Remove(p)
			}
			return nil, fmt.Errorf("writing line %d audio: %w", i+1, err)
		}
		f.Close()
		paths = append(paths, f.Name())
	}

	return paths, nil
}
