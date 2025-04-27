package model

// Result is JSON-serialisable as-is.
type Result struct {
	Original    string  `json:"original"`
	CharCount   int     `json:"charCount"`   // UTF-8 rune length
	ChunkCount  int     `json:"chunkCount"`  // ≤ 300 어절 chunks
	Corrections []Chunk `json:"corrections"` // nil if no errors
	ErrorCount  int     `json:"errorCount"`  // total number of detected errors
}

// Chunk corresponds to one 300-어절 POST.
type Chunk struct {
	Idx   int          `json:"idx"`
	Input string       `json:"input"`
	Items []Correction `json:"items"`
}

// Correction represents a single error span.
type Correction struct {
	Start   int      `json:"start"`          // rune offsets
	End     int      `json:"end"`            // rune offsets
	Origin  string   `json:"origin"`         // wrong slice
	Suggest []string `json:"suggest"`        // ≥1 candidate
	Help    string   `json:"help,omitempty"` // optional HTML
}

// RawCorrection is the raw format from server before we transform it.
type RawCorrection struct {
	Start    int    `json:"start"`
	End      int    `json:"end"`
	OrgStr   string `json:"orgStr"`
	CandWord string `json:"candWord"`
	Help     string `json:"help"`
}

// RawChunk is the wrapper for ErrInfo array (from server JSON structure).
type RawChunk struct {
	ErrInfo []RawCorrection `json:"errInfo"`
}
