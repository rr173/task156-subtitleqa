package qa_test

import (
	"context"
	"path/filepath"
	"testing"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/service"
	"task156-subtitleqa/internal/store"
	"task156-subtitleqa/internal/timeline"
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

// TestRecomputeReportsOrphanSpeakerReference guards the regression where a
// segment referencing a speaker marker that does not belong to the media was
// silently dropped instead of being reported. The orphan finding must survive
// the persist path and remain visible in the summary.
func TestRecomputeReportsOrphanSpeakerReference(t *testing.T) {
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
	// Register one speaker; the imported segment references a different,
	// unregistered speaker id, so the attribution does not belong to the media.
	if _, err := svc.CreateSpeaker(ctx, media.ID, model.CreateSpeakerRequest{Label: "主讲"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportSegments(ctx, media.ID, []model.SegmentInput{
		{Index: 0, StartMs: 0, EndMs: 2000, Text: "引用了不属于本素材的说话人", SpeakerID: "ghost"},
	}); err != nil {
		t.Fatal(err)
	}

	checks, err := svc.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	var orphan *model.QualityCheck
	for i := range checks {
		if checks[i].Rule == timeline.RuleUnknownSpeaker {
			orphan = &checks[i]
		}
	}
	if orphan == nil {
		t.Fatalf("orphan speaker reference not reported: %+v", checks)
	}
	if orphan.Severity != timeline.SeverityError {
		t.Fatalf("orphan speaker severity=%q, want %q", orphan.Severity, timeline.SeverityError)
	}

	// Recompute must not drop the finding (idempotent re-derivation).
	if err := svc.RecomputeQuality(ctx, media.ID); err != nil {
		t.Fatal(err)
	}
	after, err := svc.Quality(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	stillReported := false
	for _, c := range after {
		if c.Rule == timeline.RuleUnknownSpeaker {
			stillReported = true
		}
	}
	if !stillReported {
		t.Fatalf("recompute dropped orphan speaker finding: %+v", after)
	}

	// And the summary must surface it, not hide it.
	summary, err := svc.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ByRule[timeline.RuleUnknownSpeaker] == 0 {
		t.Fatalf("summary hides orphan speaker finding: %+v", summary)
	}
}
