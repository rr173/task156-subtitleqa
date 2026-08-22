// Package model defines the domain entities, value types and errors for the
// subtitle timeline proofreading workbench. A "media" asset (an audio/video
// programme) carries ordered subtitle segments and speaker markers. Editors
// adjust time boundaries and text; the system derives quality findings and
// publishes immutable versions.
package model

import "time"

// MediaStatus enumerates the lifecycle of a media asset.
type MediaStatus string

const (
	MediaDraft     MediaStatus = "draft"
	MediaReady     MediaStatus = "ready"
	MediaPublished MediaStatus = "published"
	MediaArchived  MediaStatus = "archived"
)

// SegmentStatus enumerates the lifecycle of a single subtitle segment.
type SegmentStatus string

const (
	SegDraft     SegmentStatus = "draft"
	SegProofed   SegmentStatus = "proofed"
	SegConflict  SegmentStatus = "conflict"
	SegPublished SegmentStatus = "published"
)

// Media is the top-level asset being proofread.
type Media struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Language   string      `json:"language"`
	DurationMs int64       `json:"duration_ms"`
	FrameRate  float64     `json:"frame_rate"`
	SourceURL  string      `json:"source_url"`
	Checksum   string      `json:"checksum"`
	Status     MediaStatus `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// Speaker is a labelled voice that segments can be attributed to.
type Speaker struct {
	ID        string    `json:"id"`
	MediaID   string    `json:"media_id"`
	Label     string    `json:"label"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// Segment is one subtitle line with an absolute time window.
type Segment struct {
	ID            string        `json:"id"`
	MediaID       string        `json:"media_id"`
	Index         int           `json:"index"`
	StartMs       int64         `json:"start_ms"`
	EndMs         int64         `json:"end_ms"`
	Text          string        `json:"text"`
	SpeakerID     string        `json:"speaker_id"`
	IsDescriptive bool          `json:"is_descriptive"`
	Language      string        `json:"language"`
	Version       int           `json:"version"`
	Status        SegmentStatus `json:"status"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// Revision records a single edit operation applied to a segment.
type Revision struct {
	ID          string    `json:"id"`
	MediaID     string    `json:"media_id"`
	SegmentID   string    `json:"segment_id"`
	Actor       string    `json:"actor"`
	OpType      string    `json:"op_type"`
	BaseVersion int       `json:"base_version"`
	BeforeJSON  string    `json:"before_json"`
	AfterJSON   string    `json:"after_json"`
	CreatedAt   time.Time `json:"created_at"`
}

// QualityCheck is a derived finding about a segment or media.
type QualityCheck struct {
	ID         string    `json:"id"`
	MediaID    string    `json:"media_id"`
	SegmentID  string    `json:"segment_id"`
	Rule       string    `json:"rule"`
	Severity   string    `json:"severity"`
	Message    string    `json:"message"`
	DetectedAt time.Time `json:"detected_at"`
	Resolved   bool      `json:"resolved"`
}

// PublishVersion is an immutable frozen snapshot of a media's segments.
type PublishVersion struct {
	ID           string    `json:"id"`
	MediaID      string    `json:"media_id"`
	VersionNo    int       `json:"version_no"`
	Label        string    `json:"label"`
	Actor        string    `json:"actor"`
	Note         string    `json:"note"`
	Frozen       bool      `json:"frozen"`
	Withdrawn    bool      `json:"withdrawn"`
	SnapshotJSON string    `json:"snapshot_json"`
	CreatedAt    time.Time `json:"created_at"`
}

// Withdrawal records that a published version was retracted (the snapshot stays).
type Withdrawal struct {
	ID              string    `json:"id"`
	PublishVersionID string   `json:"publish_version_id"`
	MediaID         string    `json:"media_id"`
	Actor           string    `json:"actor"`
	Reason          string    `json:"reason"`
	CreatedAt       time.Time `json:"created_at"`
}

// Conflict records a concurrent edit that was rejected to avoid silent overwrite.
type Conflict struct {
	ID               string    `json:"id"`
	MediaID          string    `json:"media_id"`
	SegmentID        string    `json:"segment_id"`
	ActorA           string    `json:"actor_a"`
	ActorB           string    `json:"actor_b"`
	BaseVersion      int       `json:"base_version"`
	AttemptedVersion int       `json:"attempted_version"`
	Explanation      string    `json:"explanation"`
	CreatedAt        time.Time `json:"created_at"`
}
