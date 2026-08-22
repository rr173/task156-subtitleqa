package service

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
)

func TestBug07_MissingDescriptionRemainsVisible(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "rule.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "听障字幕", DurationMs: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{{Index: 0, StartMs: 0, EndMs: 1000, Text: " ", IsDescriptive: true}}); err != nil {
		t.Fatal(err)
	}
	summary, err := svc.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ByRule["descriptive_missing"] != 1 {
		t.Fatalf("missing descriptive text findings=%+v, want one descriptive_missing finding", summary.ByRule)
	}
}
