package service

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
)

func TestBug01_ImportImmediatelyBuildsQualityFindings(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "quality-refresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	media, err := svc.CreateMedia(context.Background(), model.CreateMediaRequest{Title: "讲座", DurationMs: 5000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportSegments(context.Background(), media.ID, []model.SegmentInput{
		{Index: 0, StartMs: 0, EndMs: 1000, Text: ""},
		{Index: 1, StartMs: 3000, EndMs: 4000, Text: "后续字幕"},
	}); err != nil {
		t.Fatal(err)
	}
	summary, err := svc.QualitySummary(context.Background(), media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ByRule["empty"] != 1 || summary.ByRule["gap"] != 1 {
		t.Fatalf("import quality=%+v, want one empty and one gap finding", summary.ByRule)
	}
}
