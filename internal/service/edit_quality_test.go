package service

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
	"task156-subtitleqa/internal/timeline"
)

// TestEditRefreshesQualityAndDropsResolvedFindings guards the scenario the bug
// report describes: a proofreader fills in an empty subtitle line, and the
// quality view must no longer show the stale empty-line warning. Before the fix
// EditSegment never recomputed, and even when it did Recompute accumulated
// instead of replacing findings — both left the resolved warning visible.
func TestEditRefreshesQualityAndDropsResolvedFindings(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "edit_quality.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "访谈", DurationMs: 3000})
	if err != nil {
		t.Fatal(err)
	}
	segs, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{
		{Index: 0, StartMs: 0, EndMs: 1000, Text: ""},
	})
	if err != nil {
		t.Fatal(err)
	}

	// The empty segment must be flagged before the edit.
	hasEmpty := func(checks []model.QualityCheck) bool {
		for _, c := range checks {
			if c.Rule == timeline.RuleEmpty && c.SegmentID == segs[0].ID {
				return true
			}
		}
		return false
	}
	before, err := svc.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEmpty(before) {
		t.Fatalf("expected empty-line finding before edit, got %+v", before)
	}

	// Proofreader fills in the blank line. The edit must immediately refresh
	// the quality conclusion so the resolved warning disappears.
	filled := "现在有正文了"
	if _, _, err := svc.EditSegment(ctx, segs[0].ID, model.EditSegmentRequest{
		Actor: "proofreader", BaseVersion: 1, Text: &filled,
	}); err != nil {
		t.Fatal(err)
	}

	after, err := svc.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if hasEmpty(after) {
		t.Fatalf("stale empty-line finding survived the edit that resolved it: %+v", after)
	}
}
