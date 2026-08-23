package service

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
)

func TestBug04_BlankLineRemainsVisible(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "rule.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "纪录片", DurationMs: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{{Index: 0, StartMs: 0, EndMs: 1000, Text: "   "}}); err != nil {
		t.Fatal(err)
	}
	summary, err := svc.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ByRule["empty"] != 1 {
		t.Fatalf("blank subtitle findings=%+v, want one empty finding", summary.ByRule)
	}
}
