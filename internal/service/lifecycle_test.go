package service

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
)

func TestPublishedSnapshotRemainsFrozenAcrossLaterEdit(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "subtitle.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "访谈", Language: "zh", DurationMs: 5000})
	if err != nil {
		t.Fatal(err)
	}
	segs, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{{Index: 0, StartMs: 0, EndMs: 2000, Text: "原始字幕"}})
	if err != nil {
		t.Fatal(err)
	}
	v1, err := svc.Publish(ctx, media.ID, model.PublishRequest{Actor: "reviewer", Label: "first"})
	if err != nil {
		t.Fatal(err)
	}
	changed := "修订后的字幕"
	if _, _, err := svc.EditSegment(ctx, segs[0].ID, model.EditSegmentRequest{Actor: "editor", BaseVersion: 1, Text: &changed}); err != nil {
		t.Fatal(err)
	}
	v2, err := svc.Publish(ctx, media.ID, model.PublishRequest{Actor: "reviewer", Label: "second"})
	if err != nil {
		t.Fatal(err)
	}
	diff, err := svc.CompareVersions(ctx, v1.ID, v2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) == 0 {
		t.Fatal("later edit did not create a field-level version difference")
	}
	saved, err := svc.GetVersion(ctx, v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.SnapshotJSON == v2.SnapshotJSON {
		t.Fatal("first publish snapshot was rewritten by later edit")
	}
}
