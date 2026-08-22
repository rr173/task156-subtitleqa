// Package publish implements immutable version snapshots. Publishing freezes a
// media's full segment set and metadata into a JSON snapshot; published
// versions are never rewritten. Retraction is handled through withdrawal
// records, and comparison produces a field-level diff between two versions.
package publish

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"task156-subtitleqa/internal/model"
)

// Snapshot is the frozen representation of a media at publish time.
type Snapshot struct {
	Schema      int               `json:"schema"`
	Media       model.Media       `json:"media"`
	Speakers    []model.Speaker   `json:"speakers"`
	Segments    []model.Segment   `json:"segments"`
	GeneratedAt time.Time         `json:"generated_at"`
}

// Change is a field-level difference between two versions.
type Change struct {
	SegmentID string `json:"segment_id"`
	Field     string `json:"field"`
	Before    string `json:"before"`
	After     string `json:"after"`
}

// Diff is the result of comparing two snapshots.
type Diff struct {
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Changes []Change `json:"changes"`
	Summary string   `json:"summary"`
}

// BuildSnapshot serialises the given state into the canonical snapshot JSON.
func BuildSnapshot(media model.Media, speakers []model.Speaker, segs []model.Segment) (string, error) {
	snap := Snapshot{
		Schema:      1,
		Media:       media,
		Speakers:    speakers,
		Segments:    segs,
		GeneratedAt: time.Now().UTC(),
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ParseSnapshot deserialises a stored snapshot.
func ParseSnapshot(raw string) (*Snapshot, error) {
	var snap Snapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// Compare produces a field-level diff between two snapshots. Segments are
// matched by id; only fields that changed are reported.
func Compare(a, b *Snapshot) (*Diff, error) {
	d := &Diff{}
	bMap := make(map[string]model.Segment, len(b.Segments))
	for _, seg := range b.Segments {
		bMap[seg.ID] = seg
	}
	aMap := make(map[string]model.Segment, len(a.Segments))
	for _, seg := range a.Segments {
		aMap[seg.ID] = seg
	}
	for _, seg := range a.Segments {
		if _, ok := bMap[seg.ID]; !ok {
			d.Removed = append(d.Removed, seg.ID)
		}
	}
	for id, segB := range bMap {
		segA, ok := aMap[id]
		if !ok {
			d.Added = append(d.Added, id)
			continue
		}
		d.Changes = append(d.Changes, diffSegment(segA, segB)...)
	}
	d.Summary = fmt.Sprintf("%d added, %d removed, %d changed", len(d.Added), len(d.Removed), len(d.Changes))
	return d, nil
}

func diffSegment(a, b model.Segment) []Change {
	var out []Change
	if a.StartMs != b.StartMs {
		out = append(out, Change{a.ID, "start_ms", fmt.Sprint(a.StartMs), fmt.Sprint(b.StartMs)})
	}
	if a.EndMs != b.EndMs {
		out = append(out, Change{a.ID, "end_ms", fmt.Sprint(a.EndMs), fmt.Sprint(b.EndMs)})
	}
	if a.Text != b.Text {
		out = append(out, Change{a.ID, "text", a.Text, b.Text})
	}
	if a.SpeakerID != b.SpeakerID {
		out = append(out, Change{a.ID, "speaker_id", a.SpeakerID, b.SpeakerID})
	}
	if a.IsDescriptive != b.IsDescriptive {
		out = append(out, Change{a.ID, "is_descriptive", fmt.Sprint(a.IsDescriptive), fmt.Sprint(b.IsDescriptive)})
	}
	return out
}

// SnapshotSummary renders a one-line description of a snapshot for listings.
func SnapshotSummary(raw string) string {
	snap, err := ParseSnapshot(raw)
	if err != nil {
		return "unparsable snapshot"
	}
	names := make([]string, 0, len(snap.Segments))
	for _, seg := range snap.Segments {
		names = append(names, fmt.Sprintf("#%d", seg.Index))
	}
	return strings.Join(names, ",")
}
