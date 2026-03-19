package corpus

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Recording represents a single voice recording entry from manifest.txt.
type Recording struct {
	UUID       string
	Duration   time.Duration
	Transcript string
}

// Stats holds computed corpus statistics.
type Stats struct {
	TotalRecordings int
	TotalDuration   time.Duration
	AvgDuration     time.Duration
	TotalWords      int
	UniqueWords     int
	TopWords        []WordCount
	DateRange       string
}

// WordCount pairs a word with its frequency.
type WordCount struct {
	Word  string
	Count int
}

// uuidPattern matches UUID-formatted filenames.
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ReadTranscripts reads all .txt transcript files from the given directories.
// For voice-corpus dirs, it reads UUID-named .txt files.
// For older voice dirs, it reads from the transcripts/ subdirectory.
func ReadTranscripts(paths []string) ([]string, error) {
	var transcripts []string

	for _, dir := range paths {
		// Check if this is the older voice directory (has transcripts/ subdir)
		transcriptsDir := filepath.Join(dir, "transcripts")
		if info, err := os.Stat(transcriptsDir); err == nil && info.IsDir() {
			files, err := filepath.Glob(filepath.Join(transcriptsDir, "*.txt"))
			if err != nil {
				return nil, fmt.Errorf("reading transcripts from %s: %w", transcriptsDir, err)
			}
			for _, f := range files {
				data, err := os.ReadFile(f)
				if err != nil {
					continue
				}
				text := strings.TrimSpace(string(data))
				if text != "" {
					transcripts = append(transcripts, text)
				}
			}
		}

		// Read UUID-named .txt files directly in the directory
		files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
		if err != nil {
			continue
		}
		for _, f := range files {
			base := strings.TrimSuffix(filepath.Base(f), ".txt")
			if !uuidPattern.MatchString(base) {
				continue
			}
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			text := strings.TrimSpace(string(data))
			if text != "" {
				transcripts = append(transcripts, text)
			}
		}
	}

	return transcripts, nil
}

// ParseManifest reads a manifest.txt file and returns recordings.
// Format: <uuid>|<duration>|\n<transcript text>
func ParseManifest(path string) ([]Recording, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var recordings []Recording

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}

		uuid := parts[0]
		if !uuidPattern.MatchString(uuid) {
			continue
		}

		dur := parseDuration(parts[1])

		var transcript string
		if i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			// If next line is not a UUID line, it's the transcript
			nextParts := strings.SplitN(nextLine, "|", 3)
			if len(nextParts) < 2 || !uuidPattern.MatchString(nextParts[0]) {
				transcript = nextLine
				i++ // skip transcript line
			}
		}

		recordings = append(recordings, Recording{
			UUID:       uuid,
			Duration:   dur,
			Transcript: transcript,
		})
	}

	return recordings, nil
}

// parseDuration parses durations like "16s", "37s", "1m30s".
func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	d, err := time.ParseDuration(s)
	if err != nil {
		// Try parsing as plain seconds
		if n, err := strconv.Atoi(s); err == nil {
			return time.Duration(n) * time.Second
		}
		return 0
	}
	return d
}

// ComputeStats calculates corpus statistics from recordings and transcript files.
func ComputeStats(recordings []Recording, transcripts []string) Stats {
	stats := Stats{
		TotalRecordings: len(recordings),
	}

	// Duration stats
	for _, r := range recordings {
		stats.TotalDuration += r.Duration
	}
	if stats.TotalRecordings > 0 {
		stats.AvgDuration = stats.TotalDuration / time.Duration(stats.TotalRecordings)
	}

	// Combine all transcript text
	var allText strings.Builder
	for _, t := range transcripts {
		allText.WriteString(t)
		allText.WriteString(" ")
	}
	for _, r := range recordings {
		if r.Transcript != "" {
			allText.WriteString(r.Transcript)
			allText.WriteString(" ")
		}
	}

	// Word counting
	words := tokenize(allText.String())
	stats.TotalWords = len(words)

	wordFreq := make(map[string]int)
	for _, w := range words {
		wordFreq[w]++
	}
	stats.UniqueWords = len(wordFreq)

	// Top words excluding stop words
	type wc struct {
		word  string
		count int
	}
	var sorted []wc
	for w, c := range wordFreq {
		if !isStopWord(w) && len(w) > 1 {
			sorted = append(sorted, wc{w, c})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	limit := 50
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		stats.TopWords = append(stats.TopWords, WordCount{
			Word:  sorted[i].word,
			Count: sorted[i].count,
		})
	}

	return stats
}

// GetFileModTimes returns earliest and latest modification times of .txt files.
func GetFileModTimes(paths []string) (earliest, latest time.Time) {
	first := true
	for _, dir := range paths {
		files, _ := filepath.Glob(filepath.Join(dir, "*.txt"))
		transcriptFiles, _ := filepath.Glob(filepath.Join(dir, "transcripts", "*.txt"))
		files = append(files, transcriptFiles...)
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			t := info.ModTime()
			if first {
				earliest = t
				latest = t
				first = false
			} else {
				if t.Before(earliest) {
					earliest = t
				}
				if t.After(latest) {
					latest = t
				}
			}
		}
	}
	return
}

// tokenize splits text into lowercase words.
func tokenize(text string) []string {
	var words []string
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '\''
	}
	for _, w := range strings.FieldsFunc(text, f) {
		w = strings.ToLower(strings.Trim(w, "'"))
		if w != "" {
			words = append(words, w)
		}
	}
	return words
}

// isStopWord returns true for common English stop words.
func isStopWord(w string) bool {
	stops := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "it": true, "this": true, "that": true, "was": true,
		"are": true, "be": true, "has": true, "have": true, "had": true,
		"not": true, "no": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true, "may": true,
		"might": true, "can": true, "i": true, "you": true, "he": true,
		"she": true, "we": true, "they": true, "me": true, "him": true,
		"her": true, "us": true, "them": true, "my": true, "your": true,
		"his": true, "its": true, "our": true, "their": true, "what": true,
		"which": true, "who": true, "when": true, "where": true, "how": true,
		"all": true, "each": true, "every": true, "both": true, "few": true,
		"more": true, "most": true, "other": true, "some": true, "such": true,
		"than": true, "too": true, "very": true, "just": true, "about": true,
		"been": true, "being": true, "if": true, "so": true, "as": true,
		"into": true, "then": true, "there": true, "these": true, "those": true,
		"am": true, "were": true, "up": true, "out": true, "also": true,
		"don't": true, "i'm": true, "it's": true, "i'll": true, "i've": true,
		"i'd": true, "he's": true, "she's": true, "we're": true, "they're": true,
		"you're": true, "that's": true, "there's": true, "here's": true,
	}
	return stops[w]
}
