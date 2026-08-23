package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/store"
)

func TestBug09_RecoveryFlagsZeroLengthSegment(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "recover.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	ctx := context.Background()
	media, err := svc.CreateMedia(ctx, model.CreateMediaRequest{Title: "迁移素材", DurationMs: 5000})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB().Exec(`INSERT INTO segments (id,media_id,idx,start_ms,end_ms,text,speaker_id,is_descriptive,language,version,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, "bad", media.ID, 0, 1000, 1000, "损坏行", nil, 0, "zh", 1, "draft", now, now); err != nil {
		t.Fatal(err)
	}
	if err := svc.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	summary, err := svc.QualitySummary(ctx, media.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ByRule["non_positive_duration"] != 1 {
		t.Fatalf("recovery quality=%+v, want non_positive_duration", summary.ByRule)
	}
}
