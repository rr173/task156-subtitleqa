package store

import (
	"time"

	"task156-subtitleqa/internal/model"
)

func scanQuality(row interface{ Scan(...interface{}) error }) (model.QualityCheck, error) {
	var q model.QualityCheck
	var resolved int
	var detected string
	err := row.Scan(&q.ID, &q.MediaID, &q.SegmentID, &q.Rule, &q.Severity, &q.Message, &detected, &resolved)
	if err != nil {
		return q, err
	}
	q.Resolved = resolved != 0
	q.DetectedAt, _ = time.Parse(time.RFC3339, detected)
	return q, nil
}

// DeleteQualityForMedia removes all findings for a media so they can be
// recomputed deterministically (idempotent re-analysis). Recomputation rewrites
// the full set of findings from the current segments, so every row is cleared —
// not just the resolved ones — otherwise a finding whose underlying problem was
// fixed by an edit (e.g. an empty line that got text) would linger as a stale
// warning.
func (s *Store) DeleteQualityForMedia(mediaID string) error {
	_, err := s.db.Exec(`DELETE FROM quality_checks WHERE media_id=?`, mediaID)
	return err
}

// AddQuality inserts a single finding.
func (s *Store) AddQuality(q model.QualityCheck) error {
	resolved := 0
	if q.Resolved {
		resolved = 1
	}
	seg := sqlNullString(q.SegmentID)
	_, err := s.db.Exec(
		`INSERT INTO quality_checks (id,media_id,segment_id,rule,severity,message,detected_at,resolved)
		 VALUES (?,?,?,?,?,?,?,?)`,
		q.ID, q.MediaID, seg, q.Rule, q.Severity, q.Message,
		q.DetectedAt.UTC().Format(time.RFC3339), resolved)
	return err
}

// ListQuality returns the findings of a media ordered by severity.
func (s *Store) ListQuality(mediaID string) ([]model.QualityCheck, error) {
	rows, err := s.db.Query(
		`SELECT id,media_id,segment_id,rule,severity,message,detected_at,resolved
		 FROM quality_checks WHERE media_id=? ORDER BY severity DESC, rule ASC`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.QualityCheck
	for rows.Next() {
		q, err := scanQuality(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// CountQuality returns the total number of findings.
func (s *Store) CountQuality() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM quality_checks`).Scan(&n)
	return n, err
}
