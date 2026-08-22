package service

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
)

func TestBug10_EditRefreshesResolvedBlankFinding(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "edit-refresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "校对", DurationMs: 3000})
	if err != nil {
		t.Fatal(err)
	}
	segs, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{{Index: 0, StartMs: 0, EndMs: 1000, Text: ""}})
	if err != nil {
		t.Fatal(err)
	}
	text := "补全后的字幕"
	if _, _, err := svc.EditSegment(ctx, segs[0].ID, model.EditSegmentRequest{Actor: "editor", BaseVersion: 1, Text: &text}); err != nil {
		t.Fatal(err)
	}
	summary, err := svc.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ByRule["empty"] != 0 {
		t.Fatalf("resolved empty finding remained: %+v", summary.ByRule)
	}
}
