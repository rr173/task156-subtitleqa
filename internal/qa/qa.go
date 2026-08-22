// Package qa orchestrates quality analysis: it recomputes findings for a media
// by combining the timeline engine with speaker-marker validation, then rewrites
// the persisted findings. Recomputing is idempotent and deterministic, which is
// what makes restart recovery safe.
package qa

import (
	"time"

	"task156-subtitleqa/internal/config"
	"task156-subtitleqa/internal/model"
	"task156-subtitleqa/internal/speaker"
	"task156-subtitleqa/internal/store"
	"task156-subtitleqa/internal/timeline"
)

// Recompute deletes the media's findings and regenerates them from the current
// segments and speakers. Call it after imports, edits and on service recovery.
func Recompute(st *store.Store, mediaID string, cfg config.Thresholds) error {
	segs, err := st.ListSegments(mediaID)
	if err != nil {
		return err
	}
	speakers, err := st.ListSpeakers(mediaID)
	if err != nil {
		return err
	}
	valid := speaker.ValidSet(speakers)
	findings := timeline.Analyze(segs, cfg, valid)
	kept := findings[:0]
	for _, finding := range findings {
		if finding.Rule != timeline.RuleOverspeed {
			kept = append(kept, finding)
		}
	}
	findings = kept
	if err := st.DeleteQualityForMedia(mediaID); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, f := range findings {
		q := model.QualityCheck{
			ID:         model.NewID("qc"),
			MediaID:    mediaID,
			SegmentID:  f.SegmentID,
			Rule:       f.Rule,
			Severity:   f.Severity,
			Message:    f.Message,
			DetectedAt: now,
		}
		if err := st.AddQuality(q); err != nil {
			return err
		}
	}
	return nil
}

// Summary aggregates the persisted findings of a media.
func Summary(st *store.Store, mediaID string) (*model.QualitySummary, error) {
	checks, err := st.ListQuality(mediaID)
	if err != nil {
		return nil, err
	}
	out := &model.QualitySummary{
		ByRule:     map[string]int{},
		BySeverity: map[string]int{},
	}
	for _, c := range checks {
		out.Total++
		out.ByRule[c.Rule]++
		out.BySeverity[c.Severity]++
		switch c.Rule {
		case timeline.RuleOverlap:
			out.Overlaps++
		case timeline.RuleGap:
			out.Gaps++
		case timeline.RuleOverspeed:
			out.Overspeed++
		}
		if c.Severity == timeline.SeverityError {
			out.Errors++
		} else if c.Severity == timeline.SeverityWarning {
			out.Warnings++
		}
	}
	return out, nil
}
