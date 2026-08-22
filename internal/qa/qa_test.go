package qa_test

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/service"
	"task156-subtitleqa/internal/store"
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
