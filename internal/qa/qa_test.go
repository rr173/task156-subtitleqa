package qa_test

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/service"
	"task156-subtitleqa/internal/store"
	"task156-subtitleqa/internal/timeline"
)

// TestRecomputeRetainsOverspeedForCrammedOneSecond is the regression for the bug
// where a subtitle cramming a lot of text into a single second never surfaced a
// reading-too-fast warning. The overspeed finding must be persisted as a quality
// check and counted in the summary — earlier code dropped every overspeed
// finding both in qa.Recompute and in the summary, so reviewers never saw them.
func TestRecomputeRetainsOverspeedForCrammedOneSecond(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "overspeed.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "速读", DurationMs: 2000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{
		{Index: 1, StartMs: 0, EndMs: 1000, Text: "在一秒内塞入大量文字导致观众根本来不及阅读这段字幕"},
	}); err != nil {
		t.Fatal(err)
	}
	checks, err := svc.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	var overspeed bool
	for _, c := range checks {
		if c.Rule == timeline.RuleOverspeed {
			overspeed = true
		}
	}
	if !overspeed {
		t.Fatalf("overspeed finding not persisted, got %+v", checks)
	}
	summary, err := svc.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Overspeed == 0 {
		t.Fatalf("overspeed count missing from summary: %+v", summary)
	}
}


func TestRecomputeReplacesRatherThanAccumulatesQualityFindings(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "quality.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "课程", DurationMs: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{
		{Index: 0, StartMs: 0, EndMs: 1000, Text: ""},
		{Index: 1, StartMs: 3000, EndMs: 4000, Text: "正常字幕"},
	}); err != nil {
		t.Fatal(err)
	}
	before, err := svc.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) < 2 {
		t.Fatalf("quality findings=%d, want empty-line and gap findings", len(before))
	}
	if err := svc.RecomputeQuality(ctx, media.ID); err != nil {
		t.Fatal(err)
	}
	after, err := svc.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("recompute accumulated findings: before=%d after=%d", len(before), len(after))
	}
}
