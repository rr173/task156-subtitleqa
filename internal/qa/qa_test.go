package qa_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/service"
	"task156-subtitleqa/internal/store"
	"task156-subtitleqa/internal/timeline"
)

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

func TestRecomputePersistsLongLineFinding(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "longline.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "长行演示", DurationMs: 60000})
	if err != nil {
		t.Fatal(err)
	}
	longText := strings.Repeat("字", 200) // exceeds MaxSegmentChars (84)
	if _, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{
		{Index: 0, StartMs: 0, EndMs: 60000, Text: longText},
	}); err != nil {
		t.Fatal(err)
	}

	checks, err := svc.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for _, c := range checks {
		if c.Rule == timeline.RuleLongLine {
			seen = true
		}
	}
	if !seen {
		t.Fatalf("long_line finding missing from quality list: %+v", checks)
	}

	summary, err := svc.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ByRule[timeline.RuleLongLine] != 1 {
		t.Errorf("summary by_rule[long_line]=%d, want 1; ByRule=%+v", summary.ByRule[timeline.RuleLongLine], summary.ByRule)
	}
	if summary.LongLines != 1 {
		t.Errorf("summary.LongLines=%d, want 1", summary.LongLines)
	}
}
