package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
)

func TestStaleEditIsRecordedWithoutOverwritingCurrentSegment(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "conflict.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "新闻", DurationMs: 3000})
	if err != nil {
		t.Fatal(err)
	}
	segs, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{{Index: 0, StartMs: 0, EndMs: 1000, Text: "初稿"}})
	if err != nil {
		t.Fatal(err)
	}
	first := "甲编辑的版本"
	if _, _, err := svc.EditSegment(ctx, segs[0].ID, model.EditSegmentRequest{Actor: "editor-a", BaseVersion: 1, Text: &first}); err != nil {
		t.Fatal(err)
	}
	stale := "乙编辑的旧版本"
	_, conflict, err := svc.EditSegment(ctx, segs[0].ID, model.EditSegmentRequest{Actor: "editor-b", BaseVersion: 1, Text: &stale})
	if !errors.Is(err, model.ErrConflict) || conflict == nil {
		t.Fatalf("stale edit err=%v conflict=%v, want recorded conflict", err, conflict != nil)
	}
	current, err := svc.GetSegment(ctx, segs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Text != first || current.Version != 2 {
		t.Fatalf("stale edit overwrote current segment: %+v", current)
	}
	conflicts, err := svc.ListConflicts(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 || conflicts[0].ActorB != "editor-b" {
		t.Fatalf("recorded conflicts=%+v, want editor-b conflict", conflicts)
	}
}
