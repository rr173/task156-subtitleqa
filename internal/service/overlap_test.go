package service

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
	"task156-subtitleqa/internal/timeline"
)

// TestOverlappingSegmentsAreSurfacedInQA guards the full path an editor sees: two
// segments sharing the same time window must produce a persisted overlap
// finding and a non-zero overlap count in the QA summary. Previously three
// independent suppressors hid the problem — the analysis threshold, the
// Recompute filter and the summary deletion — so editors could not spot two
// subtitles fighting for the same time slot.
func TestOverlappingSegmentsAreSurfacedInQA(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "overlap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "课程", DurationMs: 6000})
	if err != nil {
		t.Fatal(err)
	}
	// Segment #2 starts at 2500 ms, 500 ms before segment #1 ends at 3000 ms.
	if _, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{
		{Index: 1, StartMs: 0, EndMs: 3000, Text: "第一行字幕"},
		{Index: 2, StartMs: 2500, EndMs: 6000, Text: "与上一行重叠"},
	}); err != nil {
		t.Fatal(err)
	}

	checks, err := svc.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	var overlapSeen bool
	for _, c := range checks {
		if c.Rule == timeline.RuleOverlap {
			overlapSeen = true
		}
	}
	if !overlapSeen {
		t.Fatalf("no persisted overlap finding; got %d checks", len(checks))
	}

	summary, err := svc.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Overlaps == 0 {
		t.Fatalf("overlap count not surfaced in summary: %+v", summary)
	}
	if summary.ByRule[timeline.RuleOverlap] == 0 {
		t.Fatalf("overlap missing from by_rule summary: %+v", summary)
	}
}
