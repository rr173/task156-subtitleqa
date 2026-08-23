package model

// CreateMediaRequest is the payload for registering a new media asset.
type CreateMediaRequest struct {
	Title      string  `json:"title"`
	Language   string  `json:"language"`
	DurationMs int64   `json:"duration_ms"`
	FrameRate  float64 `json:"frame_rate"`
	SourceURL  string  `json:"source_url"`
	Checksum   string  `json:"checksum"`
}

// CreateSpeakerRequest is the payload for adding a speaker to a media.
type CreateSpeakerRequest struct {
	Label string `json:"label"`
	Color string `json:"color"`
}

// SegmentInput is a single subtitle line supplied during import.
type SegmentInput struct {
	Index         int    `json:"index"`
	StartMs       int64  `json:"start_ms"`
	EndMs         int64  `json:"end_ms"`
	Text          string `json:"text"`
	SpeakerID     string `json:"speaker_id"`
	IsDescriptive bool   `json:"is_descriptive"`
	Language      string `json:"language"`
}

// EditSegmentRequest carries a concurrent-safe edit. BaseVersion must equal the
// segment's current version; a mismatch produces a recorded conflict instead of
// a silent overwrite. Pointer fields denote "set this value".
type EditSegmentRequest struct {
	Actor         string `json:"actor"`
	BaseVersion   int    `json:"base_version"`
	StartMs       *int64 `json:"start_ms"`
	EndMs         *int64 `json:"end_ms"`
	Text          *string `json:"text"`
	SpeakerID     *string `json:"speaker_id"`
	IsDescriptive *bool  `json:"is_descriptive"`
}

// PublishRequest is the payload for freezing a publishable version.
type PublishRequest struct {
	Actor string `json:"actor"`
	Label string `json:"label"`
	Note  string `json:"note"`
}

// WithdrawRequest is the payload for retracting a published version.
type WithdrawRequest struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

// QualitySummary aggregates quality findings for a media.
type QualitySummary struct {
	Total      int            `json:"total"`
	ByRule     map[string]int `json:"by_rule"`
	BySeverity map[string]int `json:"by_severity"`
	Overlaps   int            `json:"overlaps"`
	Gaps       int            `json:"gaps"`
	Overspeed  int            `json:"overspeed"`
	LongLines  int            `json:"long_lines"`
	Errors     int            `json:"errors"`
	Warnings   int            `json:"warnings"`
}

// Counts is the top-level statistics returned by /api/stats.
type Counts struct {
	Media      int `json:"media"`
	Segments   int `json:"segments"`
	Speakers   int `json:"speakers"`
	Quality    int `json:"quality"`
	Versions   int `json:"versions"`
	Conflicts  int `json:"conflicts"`
	Withdrawals int `json:"withdrawals"`
}
