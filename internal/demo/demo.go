// Package demo contains the embedded end-to-end smoke test. Seed creates a
// fresh temporary SQLite database, drives the full business loop (media,
// speakers, imports, quality findings, concurrent-edit conflict, publish,
// compare, withdraw), then closes and reopens the database to prove that
// persisted state and restart recovery work. It never needs external services.
package demo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/service"
	"task156-subtitleqa/internal/store"
)

// Result summarises the smoke run; it is exposed by the demo API endpoint too.
type Result struct {
	MediaID     string `json:"media_id"`
	Title       string `json:"title"`
	Segments    int    `json:"segments"`
	Quality     int    `json:"quality"`
	Versions    int    `json:"versions"`
	Conflicts   int    `json:"conflicts"`
	Withdrawals int    `json:"withdrawals"`
	Persisted   bool   `json:"persisted"`
}

// Seed runs the full scenario against a throwaway database and returns the
// summary. Any invariant violation aborts with a non-nil error.
func Seed(ctx context.Context) (*Result, error) {
	tmp, err := os.MkdirTemp("", "subtitleqa-smoke-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	dbPath := filepath.Join(tmp, "smoke.db")

	st, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	svc := service.New(st)
	if err := svc.Recover(ctx); err != nil {
		return nil, err
	}

	// 1) Media + speakers.
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{
		Title: "示例节目：无障碍字幕校对演示", Language: "zh", DurationMs: 30000,
	})
	if err != nil {
		return nil, err
	}
	spA, err := svc.CreateSpeaker(ctx, media.ID, model.CreateSpeakerRequest{Label: "旁白", Color: "#2563eb"})
	if err != nil {
		return nil, err
	}
	spB, err := svc.CreateSpeaker(ctx, media.ID, model.CreateSpeakerRequest{Label: "主持人", Color: "#16a34a"})
	if err != nil {
		return nil, err
	}

	// 2) Import segments that intentionally trip every analysis rule:
	//    #2 overlaps #1, #3 is overspeed, #4 is empty, #5 is a descriptive
	//    segment with no text, #6 references an unknown speaker, and there is a
	//    large silent gap between #3 and #4.
	segs, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{
		{Index: 1, StartMs: 0, EndMs: 3000, Text: "欢迎观看本期无障碍字幕节目", SpeakerID: spA.ID},
		{Index: 2, StartMs: 2500, EndMs: 6000, Text: "今天由主持人带您了解字幕校对", SpeakerID: spB.ID},
		{Index: 3, StartMs: 6000, EndMs: 7600, Text: "这段文字的长度明显超出舒适阅读速度要求因此会被标记为超速行示例", SpeakerID: spA.ID},
		{Index: 4, StartMs: 20000, EndMs: 22000, Text: "", SpeakerID: spA.ID},
		{Index: 5, StartMs: 22000, EndMs: 24000, Text: "", SpeakerID: spB.ID, IsDescriptive: true},
		{Index: 6, StartMs: 24000, EndMs: 26000, Text: "这条片段引用了未注册的说话人", SpeakerID: "ghost-speaker"},
	})
	if err != nil {
		return nil, err
	}
	quality, err := svc.Quality(ctx, media.ID)
	if err != nil {
		return nil, err
	}
	if len(quality) < 5 {
		return nil, fmt.Errorf("expected at least 5 quality findings, got %d", len(quality))
	}

	// 3) Concurrent-safe edit: first edit succeeds at v1 -> v2.
	firstText := "欢迎观看本期无障碍字幕节目（修订）"
	if _, _, err := svc.EditSegment(ctx, segs[0].ID, model.EditSegmentRequest{
		Actor: "editor-a", BaseVersion: 1, Text: &firstText,
	}); err != nil {
		return nil, err
	}

	// 4) A second session editing from the stale base must conflict, not
	//    silently overwrite.
	staleText := "被并发覆盖的文本"
	_, conflict, err := svc.EditSegment(ctx, segs[0].ID, model.EditSegmentRequest{
		Actor: "editor-b", BaseVersion: 1, Text: &staleText,
	})
	if !errors.Is(err, model.ErrConflict) || conflict == nil {
		return nil, fmt.Errorf("expected recorded conflict, got err=%v conflict=%v", err, conflict != nil)
	}

	// 5) Publish v1, edit again, publish v2, compare.
	pv1, err := svc.Publish(ctx, media.ID, model.PublishRequest{Actor: "publisher", Label: "v1", Note: "初次发布"})
	if err != nil {
		return nil, err
	}
	finalText := "欢迎观看本期无障碍字幕节目（最终）"
	if _, _, err := svc.EditSegment(ctx, segs[0].ID, model.EditSegmentRequest{
		Actor: "editor-a", BaseVersion: 2, Text: &finalText,
	}); err != nil {
		return nil, err
	}
	pv2, err := svc.Publish(ctx, media.ID, model.PublishRequest{Actor: "publisher", Label: "v2", Note: "修订发布"})
	if err != nil {
		return nil, err
	}
	diff, err := svc.CompareVersions(ctx, pv1.ID, pv2.ID)
	if err != nil {
		return nil, err
	}
	if len(diff.Changes) == 0 {
		return nil, errors.New("expected a field diff between v1 and v2")
	}

	// 6) Withdraw v1; the snapshot stays but is flagged retracted.
	if err := svc.Withdraw(ctx, pv1.ID, model.WithdrawRequest{Actor: "publisher", Reason: "v1 含重叠问题"}); err != nil {
		return nil, err
	}

	// 7) Persistence + restart recovery: close, reopen, re-run recovery.
	if err := st.Close(); err != nil {
		return nil, err
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer st2.Close()
	svc2 := service.New(st2)
	if err := svc2.Recover(ctx); err != nil {
		return nil, err
	}
	if _, err := svc2.GetMedia(ctx, media.ID); err != nil {
		return nil, err
	}
	segs2, err := svc2.ListSegments(ctx, media.ID)
	if err != nil {
		return nil, err
	}
	vers, err := svc2.ListVersions(ctx, media.ID)
	if err != nil {
		return nil, err
	}
	confs, err := svc2.ListConflicts(ctx, media.ID)
	if err != nil {
		return nil, err
	}
	pv1b, err := svc2.GetVersion(ctx, pv1.ID)
	if err != nil {
		return nil, err
	}
	if len(segs2) != len(segs) || len(vers) != 2 || len(confs) != 1 || !pv1b.Withdrawn {
		return nil, fmt.Errorf("persistence verification failed: segments=%d versions=%d conflicts=%d withdrawn=%v",
			len(segs2), len(vers), len(confs), pv1b.Withdrawn)
	}

	return &Result{
		MediaID:     media.ID,
		Title:       media.Title,
		Segments:    len(segs2),
		Quality:     len(quality),
		Versions:    len(vers),
		Conflicts:   len(confs),
		Withdrawals: 1,
		Persisted:   true,
	}, nil
}
