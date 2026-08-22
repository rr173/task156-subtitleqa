// Package service is the orchestration layer: it validates input, enforces
// business rules (version gates, publish immutability, conflict detection),
// coordinates the store with the analysis/QA packages and exposes a facade for
// the HTTP layer and the smoke test.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"task156-subtitleqa/internal/config"
	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/publish"
	"task156-subtitleqa/internal/qa"
	"task156-subtitleqa/internal/revision"
	"task156-subtitleqa/internal/store"
)

// Service bundles a store with the configured thresholds.
type Service struct {
	store *store.Store
	cfg   config.Thresholds
}

// New builds a service over an open store.
func New(st *store.Store) *Service {
	return &Service{store: st, cfg: config.Default()}
}

// Store exposes the underlying store for packages that need direct queries.
func (s *Service) Store() *store.Store { return s.store }

// Recover recomputes quality findings for every non-archived media. All state
// lives in SQLite, so restart recovery is simply reopen + re-derive findings;
// recomputation is idempotent and never touches frozen publish snapshots.
func (s *Service) Recover(ctx context.Context) error {
	medias, err := s.store.ListMedia()
	if err != nil {
		return err
	}
	for _, m := range medias {
		if m.Status == model.MediaArchived {
			continue
		}
		if err := qa.Recompute(s.store, m.ID, s.cfg); err != nil {
			return fmt.Errorf("recover media %s: %w", m.ID, err)
		}
	}
	return nil
}

// CreateMedia registers a new media asset in draft status.
func (s *Service) CreateMedia(ctx context.Context, req model.CreateMediaRequest) (*model.Media, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, model.FieldError{Field: "title", Message: "must not be empty"}
	}
	lang := strings.TrimSpace(req.Language)
	if lang == "" {
		lang = "zh"
	}
	if req.DurationMs < 0 {
		return nil, model.FieldError{Field: "duration_ms", Message: "must be >= 0"}
	}
	if req.FrameRate < 0 {
		return nil, model.FieldError{Field: "frame_rate", Message: "must be >= 0"}
	}
	now := time.Now().UTC()
	m := model.Media{
		ID:         model.NewID("media"),
		Title:      title,
		Language:   lang,
		DurationMs: req.DurationMs,
		FrameRate:  req.FrameRate,
		SourceURL:  req.SourceURL,
		Checksum:   req.Checksum,
		Status:     model.MediaDraft,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.store.CreateMedia(m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ListMedia returns all media.
func (s *Service) ListMedia(ctx context.Context) ([]model.Media, error) {
	return s.store.ListMedia()
}

// GetMedia returns a single media.
func (s *Service) GetMedia(ctx context.Context, id string) (*model.Media, error) {
	m, err := s.store.GetMedia(id)
	if err != nil {
		return nil, asNotFound(err, "media")
	}
	return &m, nil
}

// CreateSpeaker adds a speaker marker to a media.
func (s *Service) CreateSpeaker(ctx context.Context, mediaID string, req model.CreateSpeakerRequest) (*model.Speaker, error) {
	if _, err := s.store.GetMedia(mediaID); err != nil {
		return nil, asNotFound(err, "media")
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		return nil, model.FieldError{Field: "label", Message: "must not be empty"}
	}
	sp := model.Speaker{
		ID:        model.NewID("spk"),
		MediaID:   mediaID,
		Label:     label,
		Color:     req.Color,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateSpeaker(sp); err != nil {
		return nil, err
	}
	return &sp, nil
}

// ListSpeakers returns the speaker markers of a media.
func (s *Service) ListSpeakers(ctx context.Context, mediaID string) ([]model.Speaker, error) {
	if _, err := s.store.GetMedia(mediaID); err != nil {
		return nil, asNotFound(err, "media")
	}
	return s.store.ListSpeakers(mediaID)
}

// ImportSegments validates and persists a batch of subtitle lines, then
// recomputes quality findings. Duplicate sequence numbers are rejected; speaker
// ids are validated at QA time so orphan references become findings, not lost
// data.
func (s *Service) ImportSegments(ctx context.Context, mediaID string, inputs []model.SegmentInput) ([]model.Segment, error) {
	media, err := s.store.GetMedia(mediaID)
	if err != nil {
		return nil, asNotFound(err, "media")
	}
	if media.Status == model.MediaArchived {
		return nil, model.FieldError{Field: "media", Message: "media is archived"}
	}
	if len(inputs) == 0 {
		return nil, model.FieldError{Field: "segments", Message: "must not be empty"}
	}
	existing, err := s.store.ListSegments(mediaID)
	if err != nil {
		return nil, err
	}
	used := make(map[int]bool, len(existing))
	for _, e := range existing {
		used[e.Index] = true
	}
	now := time.Now().UTC()
	segs := make([]model.Segment, 0, len(inputs))
	for _, in := range inputs {
		if in.Index < 0 {
			return nil, model.FieldError{Field: "index", Message: "must be >= 0"}
		}
		if used[in.Index] {
			return nil, model.FieldError{Field: "index", Message: fmt.Sprintf("index %d already exists", in.Index)}
		}
		used[in.Index] = true
		if in.StartMs < 0 {
			return nil, model.FieldError{Field: "start_ms", Message: fmt.Sprintf("segment #%d start must be >= 0", in.Index)}
		}
		if in.EndMs <= in.StartMs {
			return nil, model.FieldError{Field: "end_ms", Message: fmt.Sprintf("segment #%d end must be after start", in.Index)}
		}
		lang := strings.TrimSpace(in.Language)
		if lang == "" {
			lang = media.Language
		}
		segs = append(segs, model.Segment{
			ID:            model.NewID("seg"),
			MediaID:       mediaID,
			Index:         in.Index,
			StartMs:       in.StartMs,
			EndMs:         in.EndMs,
			Text:          in.Text,
			SpeakerID:     in.SpeakerID,
			IsDescriptive: in.IsDescriptive,
			Language:      lang,
			Version:       1,
			Status:        model.SegDraft,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	if err := s.store.CreateSegments(segs); err != nil {
		return nil, err
	}
	return segs, nil
}

// GetSegment returns a single segment.
func (s *Service) GetSegment(ctx context.Context, id string) (*model.Segment, error) {
	seg, err := s.store.GetSegment(id)
	if err != nil {
		return nil, asNotFound(err, "segment")
	}
	return &seg, nil
}

// ListSegments returns the segments of a media in timeline order.
func (s *Service) ListSegments(ctx context.Context, mediaID string) ([]model.Segment, error) {
	if _, err := s.store.GetMedia(mediaID); err != nil {
		return nil, asNotFound(err, "media")
	}
	return s.store.ListSegments(mediaID)
}

// EditSegment applies a concurrent-safe edit. If the supplied base version does
// not match the stored version, the edit is rejected, a Conflict is recorded
// (with the actor of the last write and an explanation) and ErrConflict is
// returned; the segment is never silently overwritten.
func (s *Service) EditSegment(ctx context.Context, segID string, req model.EditSegmentRequest) (*model.Segment, *model.Conflict, error) {
	seg, err := s.store.GetSegment(segID)
	if err != nil {
		return nil, nil, asNotFound(err, "segment")
	}
	if strings.TrimSpace(req.Actor) == "" {
		return nil, nil, model.FieldError{Field: "actor", Message: "must not be empty"}
	}
	if seg.Version != req.BaseVersion {
		actorA := s.lastActor(ctx, seg.ID)
		c := model.Conflict{
			ID:               model.NewID("conflict"),
			MediaID:          seg.MediaID,
			SegmentID:        seg.ID,
			ActorA:           actorA,
			ActorB:           req.Actor,
			BaseVersion:      seg.Version,
			AttemptedVersion: req.BaseVersion,
			Explanation:      fmt.Sprintf("edit based on version %d but segment is at version %d (last write by %s)", req.BaseVersion, seg.Version, actorA),
			CreatedAt:        time.Now().UTC(),
		}
		if err := s.store.AddConflict(c); err != nil {
			return nil, nil, err
		}
		_ = s.store.UpdateSegmentStatus(seg.ID, model.SegConflict)
		return nil, &c, model.ErrConflict
	}
	patched, rev, ok := revision.Patch(seg, req)
	if !ok {
		return nil, nil, model.ErrConflict
	}
	if patched.EndMs <= patched.StartMs {
		return nil, nil, model.FieldError{Field: "end_ms", Message: "end must be after start"}
	}
	if req.SpeakerID != nil && *req.SpeakerID != "" {
		if _, err := s.store.GetSpeaker(*req.SpeakerID); err != nil {
			return nil, nil, asNotFound(err, "speaker")
		}
	}
	if err := s.store.UpdateSegment(patched); err != nil {
		return nil, nil, err
	}
	if err := s.store.AddRevision(rev); err != nil {
		return nil, nil, err
	}
	if err := qa.Recompute(s.store, patched.MediaID, s.cfg); err != nil {
		return nil, nil, err
	}
	return &patched, nil, nil
}

// lastActor finds the actor of the most recent revision of a segment, if any.
func (s *Service) lastActor(ctx context.Context, segmentID string) string {
	revs, err := s.store.ListRevisions(segmentID)
	if err != nil || len(revs) == 0 {
		return "unknown"
	}
	return revs[0].Actor
}

// Revisions returns the edit history of a segment.
func (s *Service) Revisions(ctx context.Context, segmentID string) ([]model.Revision, error) {
	if _, err := s.store.GetSegment(segmentID); err != nil {
		return nil, asNotFound(err, "segment")
	}
	return s.store.ListRevisions(segmentID)
}

// RecomputeQuality re-derives findings for a media.
func (s *Service) RecomputeQuality(ctx context.Context, mediaID string) error {
	if _, err := s.store.GetMedia(mediaID); err != nil {
		return asNotFound(err, "media")
	}
	return qa.Recompute(s.store, mediaID, s.cfg)
}

// Quality returns the current findings of a media.
func (s *Service) Quality(ctx context.Context, mediaID string) ([]model.QualityCheck, error) {
	if _, err := s.store.GetMedia(mediaID); err != nil {
		return nil, asNotFound(err, "media")
	}
	return s.store.ListQuality(mediaID)
}

// QualitySummary aggregates the findings of a media.
func (s *Service) QualitySummary(ctx context.Context, mediaID string) (*model.QualitySummary, error) {
	if _, err := s.store.GetMedia(mediaID); err != nil {
		return nil, asNotFound(err, "media")
	}
	return qa.Summary(s.store, mediaID)
}

// Publish freezes a version of the media. The snapshot is immutable; later
// changes produce newer versions instead of rewriting history.
func (s *Service) Publish(ctx context.Context, mediaID string, req model.PublishRequest) (*model.PublishVersion, error) {
	media, err := s.store.GetMedia(mediaID)
	if err != nil {
		return nil, asNotFound(err, "media")
	}
	if media.Status == model.MediaArchived {
		return nil, model.FieldError{Field: "media", Message: "media is archived"}
	}
	segs, err := s.store.ListSegments(mediaID)
	if err != nil {
		return nil, err
	}
	speakers, err := s.store.ListSpeakers(mediaID)
	if err != nil {
		return nil, err
	}
	snap, err := publish.BuildSnapshot(media, speakers, segs)
	if err != nil {
		return nil, err
	}
	count, err := s.store.CountPublishVersions(mediaID)
	if err != nil {
		return nil, err
	}
	pv := model.PublishVersion{
		ID:           model.NewID("ver"),
		MediaID:      mediaID,
		VersionNo:    count + 1,
		Label:        req.Label,
		Actor:        req.Actor,
		Note:         req.Note,
		Frozen:       true,
		SnapshotJSON: snap,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.store.AddPublishVersion(pv); err != nil {
		return nil, err
	}
	if err := s.store.UpdateMediaStatus(mediaID, model.MediaPublished); err != nil {
		return nil, err
	}
	return &pv, nil
}

// ListVersions returns the version history of a media.
func (s *Service) ListVersions(ctx context.Context, mediaID string) ([]model.PublishVersion, error) {
	if _, err := s.store.GetMedia(mediaID); err != nil {
		return nil, asNotFound(err, "media")
	}
	return s.store.ListPublishVersions(mediaID)
}

// GetVersion returns a single publish version.
func (s *Service) GetVersion(ctx context.Context, id string) (*model.PublishVersion, error) {
	pv, err := s.store.GetPublishVersion(id)
	if err != nil {
		return nil, asNotFound(err, "version")
	}
	return &pv, nil
}

// Withdraw retracts a published version. The snapshot stays frozen and readable;
// only its withdrawn flag changes and a withdrawal record is appended.
func (s *Service) Withdraw(ctx context.Context, versionID string, req model.WithdrawRequest) error {
	pv, err := s.store.GetPublishVersion(versionID)
	if err != nil {
		return asNotFound(err, "version")
	}
	if pv.Withdrawn {
		return model.ErrWithdrawn
	}
	w := model.Withdrawal{
		ID:               model.NewID("wd"),
		PublishVersionID: versionID,
		MediaID:          pv.MediaID,
		Actor:            req.Actor,
		Reason:           req.Reason,
		CreatedAt:        time.Now().UTC(),
	}
	if err := s.store.AddWithdrawal(w); err != nil {
		return err
	}
	return s.store.MarkWithdrawn(versionID)
}

// CompareVersions computes a field-level diff between two publish versions.
func (s *Service) CompareVersions(ctx context.Context, v1ID, v2ID string) (*publish.Diff, error) {
	p1, err := s.store.GetPublishVersion(v1ID)
	if err != nil {
		return nil, asNotFound(err, "version")
	}
	p2, err := s.store.GetPublishVersion(v2ID)
	if err != nil {
		return nil, asNotFound(err, "version")
	}
	s1, err := publish.ParseSnapshot(p1.SnapshotJSON)
	if err != nil {
		return nil, err
	}
	s2, err := publish.ParseSnapshot(p2.SnapshotJSON)
	if err != nil {
		return nil, err
	}
	return publish.Compare(s1, s2)
}

// ListConflicts returns the recorded concurrent edits of a media.
func (s *Service) ListConflicts(ctx context.Context, mediaID string) ([]model.Conflict, error) {
	if _, err := s.store.GetMedia(mediaID); err != nil {
		return nil, asNotFound(err, "media")
	}
	return s.store.ListConflicts(mediaID)
}

// Counts returns global statistics for the dashboard.
func (s *Service) Counts(ctx context.Context) (*model.Counts, error) {
	medias, err := s.store.ListMedia()
	if err != nil {
		return nil, err
	}
	c := &model.Counts{}
	c.Media = len(medias)
	for _, m := range medias {
		n, err := s.store.CountSegments(m.ID)
		if err != nil {
			return nil, err
		}
		c.Segments += n
		n, err = s.store.CountPublishVersions(m.ID)
		if err != nil {
			return nil, err
		}
		c.Versions += n
	}
	if c.Speakers, err = s.store.CountSpeakers(); err != nil {
		return nil, err
	}
	if c.Quality, err = s.store.CountQuality(); err != nil {
		return nil, err
	}
	if c.Conflicts, err = s.store.CountConflicts(); err != nil {
		return nil, err
	}
	if c.Withdrawals, err = s.store.CountWithdrawals(); err != nil {
		return nil, err
	}
	return c, nil
}

// asNotFound wraps raw driver errors into a typed not-found error.
func asNotFound(err error, kind string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s not found: %w", kind, model.ErrNotFound)
	}
	return err
}
