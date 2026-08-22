package store

import (
	"time"

	"task156-subtitleqa/internal/model"
)

func scanRevision(row interface{ Scan(...interface{}) error }) (model.Revision, error) {
	var r model.Revision
	var created string
	err := row.Scan(&r.ID, &r.MediaID, &r.SegmentID, &r.Actor, &r.OpType,
		&r.BaseVersion, &r.BeforeJSON, &r.AfterJSON, &created)
	if err != nil {
		return r, err
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return r, nil
}

// AddRevision records a single edit operation.
func (s *Store) AddRevision(r model.Revision) error {
	_, err := s.db.Exec(
		`INSERT INTO revisions (id,media_id,segment_id,actor,op_type,base_version,before_json,after_json,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		r.ID, r.MediaID, r.SegmentID, r.Actor, r.OpType, r.BaseVersion,
		r.BeforeJSON, r.AfterJSON, r.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

// ListRevisions returns the edit history of a segment, newest first.
func (s *Store) ListRevisions(segmentID string) ([]model.Revision, error) {
	rows, err := s.db.Query(
		`SELECT id,media_id,segment_id,actor,op_type,base_version,before_json,after_json,created_at
		 FROM revisions WHERE segment_id=? ORDER BY created_at DESC`, segmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Revision
	for rows.Next() {
		r, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
