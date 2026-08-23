package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
	"task156-subtitleqa/internal/timeline"
)

// TestRecoverFlagsCorruptedZeroDurationSegments ensures restart recovery
// surfaces segments with no valid duration (start == end) as non-positive
// duration findings instead of dropping them. Historical material may hold such
// corrupted segments; recovery must flag them, not mask them.
func TestRecoverFlagsCorruptedZeroDurationSegments(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "recover.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st)
	ctx := context.Background()

	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "历史素材", DurationMs: 4000})
	if err != nil {
		t.Fatal(err)
	}
	// Seed a well-formed segment via the validated import path so the media has
	// real content alongside the corrupted one.
	if _, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{
		{Index: 0, StartMs: 0, EndMs: 1000, Text: "正常片段"},
	}); err != nil {
		t.Fatal(err)
	}
	// Simulate a corrupted historical segment whose start equals its end. It
	// bypasses the import validation the way legacy/migrated data does.
	now := time.Now().UTC()
	broken := model.Segment{
		ID:        model.NewID("seg"),
		MediaID:   media.ID,
		Index:     1,
		StartMs:   2000,
		EndMs:     2000, // zero duration
		Text:      "损坏字幕",
		Language:  media.Language,
		Version:   1,
		Status:    model.SegDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := st.CreateSegments([]model.Segment{broken}); err != nil {
		t.Fatal(err)
	}

	// Restart: close, reopen and re-run recovery.
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	svc2 := New(st2)
	if err := svc2.Recover(ctx); err != nil {
		t.Fatal(err)
	}

	findings, err := svc2.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	var sawNonPositive bool
	for _, f := range findings {
		if f.Rule == timeline.RuleNonPositiveDuration && f.SegmentID == broken.ID {
			sawNonPositive = true
		}
	}
	if !sawNonPositive {
		t.Fatalf("recovery dropped non-positive-duration finding for zero-duration segment; findings=%+v", findings)
	}

	summary, err := svc2.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := summary.ByRule[timeline.RuleNonPositiveDuration]; !ok || n < 1 {
		t.Fatalf("summary missing non_positive_duration count: %+v", summary.ByRule)
	}
}
