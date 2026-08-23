package service

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
)

func TestBug05_OverspeedRemainsVisible(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "rule.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "直播", DurationMs: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{{Index: 0, StartMs: 0, EndMs: 1000, Text: "这是一个足够长的字幕文本用于稳定触发每秒阅读速度超限问题"}}); err != nil {
		t.Fatal(err)
	}
	summary, err := svc.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ByRule["overspeed"] != 1 {
		t.Fatalf("overspeed findings=%+v, want one overspeed finding", summary.ByRule)
	}
}
