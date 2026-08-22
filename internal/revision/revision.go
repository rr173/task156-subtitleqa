// Package revision implements the optimistic-concurrency edit engine. Every
// segment carries a version number; an edit must declare the version it was
// based on. If the declared version no longer matches the stored one, the edit
// is rejected (and a conflict is recorded by the caller) instead of silently
// overwriting the concurrent change made by another session.
package revision

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"task156-subtitleqa/internal/model"
)

// Patch applies a validated edit to a segment. It returns the patched segment
// plus the audit Revision when the base version matches; otherwise ok=false and
// the segment is left untouched. Patch never mutates the input value.
func Patch(seg model.Segment, req model.EditSegmentRequest) (patched model.Segment, rev model.Revision, ok bool) {
	if seg.Version != req.BaseVersion {
		return seg, model.Revision{}, false
	}
	before := snapshot(seg)
	if req.StartMs != nil {
		seg.StartMs = *req.StartMs
	}
	if req.EndMs != nil {
		seg.EndMs = *req.EndMs
	}
	if req.Text != nil {
		seg.Text = *req.Text
	}
	if req.SpeakerID != nil {
		seg.SpeakerID = *req.SpeakerID
	}
	if req.IsDescriptive != nil {
		seg.IsDescriptive = *req.IsDescriptive
	}
	seg.Version++
	seg.UpdatedAt = time.Now().UTC()
	after := snapshot(seg)
	rev = model.Revision{
		ID:          model.NewID("rev"),
		MediaID:     seg.MediaID,
		SegmentID:   seg.ID,
		Actor:       req.Actor,
		OpType:      OpType(seg, before),
		BaseVersion: req.BaseVersion,
		BeforeJSON:  before,
		AfterJSON:   after,
		CreatedAt:   seg.UpdatedAt,
	}
	return seg, rev, true
}

// OpType classifies the edit for the audit log.
func OpType(seg model.Segment, beforeJSON string) string {
	var before model.Segment
	_ = json.Unmarshal([]byte(beforeJSON), &before)
	switch {
	case seg.StartMs != before.StartMs || seg.EndMs != before.EndMs:
		return "timing"
	case seg.SpeakerID != before.SpeakerID:
		return "reassign_speaker"
	case seg.IsDescriptive != before.IsDescriptive:
		return "toggle_descriptive"
	case strings.TrimSpace(seg.Text) != strings.TrimSpace(before.Text):
		return "text"
	default:
		return "noop"
	}
}

// Describe returns a human-readable summary of the change between two versions,
// used in the audit log and the web UI.
func Describe(before, after model.Segment) string {
	var parts []string
	if before.StartMs != after.StartMs || before.EndMs != after.EndMs {
		parts = append(parts, fmt.Sprintf("timing %d..%d -> %d..%d", before.StartMs, before.EndMs, after.StartMs, after.EndMs))
	}
	if before.Text != after.Text {
		parts = append(parts, "text changed")
	}
	if before.SpeakerID != after.SpeakerID {
		parts = append(parts, fmt.Sprintf("speaker %q -> %q", before.SpeakerID, after.SpeakerID))
	}
	if before.IsDescriptive != after.IsDescriptive {
		parts = append(parts, fmt.Sprintf("descriptive %v -> %v", before.IsDescriptive, after.IsDescriptive))
	}
	if len(parts) == 0 {
		return "no-op edit"
	}
	return strings.Join(parts, "; ")
}

func snapshot(seg model.Segment) string {
	raw, _ := json.Marshal(seg)
	return string(raw)
}
