package corpus

// Item represents a universal corpus item that can be voice, text, video, photo, code, or social.
type Item struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`    // voice, text, video, photo, code, social
	Source          string            `json:"source"`  // discord, twitter, blog, github, local, etc.
	CreatedAt       string            `json:"created_at"`
	IngestedAt      string            `json:"ingested_at"`
	Path            string            `json:"path"`       // relative to corpus root
	Transcript      string            `json:"transcript"` // text content or transcript
	Tags            []string          `json:"tags"`
	Metadata        map[string]string `json:"metadata"` // type-specific metadata
	WordCount       int               `json:"word_count"`
	DurationSeconds float64           `json:"duration_seconds"`
	FileSize        int64             `json:"file_size"`
}

// Valid corpus item types.
const (
	TypeVoice  = "voice"
	TypeText   = "text"
	TypeVideo  = "video"
	TypePhoto  = "photo"
	TypeCode   = "code"
	TypeSocial = "social"
)
