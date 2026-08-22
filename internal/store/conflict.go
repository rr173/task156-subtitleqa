package store

import (
	"time"

	"task156-subtitleqa/internal/model"
)

func scanConflict(row interface{ Scan(...interface{}) error }) (model.Conflict, error) {
	var c model.Conflict
	var created string
	err := row.Scan(&c.ID, &c.MediaID, &c.SegmentID, &c.ActorA, &c.ActorB,
		&c.BaseVersion, &c.AttemptedVersion, &c.Explanation, &created)
	if err != nil {
		return c, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return c, nil
}

// AddConflict records a rejected concurrent edit.
func (s *Store) AddConflict(c model.Conflict) error {
	_, err := s.db.Exec(
		`INSERT INTO conflicts (id,media_id,segment_id,actor_a,actor_b,base_version,attempted_version,explanation,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		c.ID, c.MediaID, c.SegmentID, c.ActorA, c.ActorB, c.BaseVersion,
		c.AttemptedVersion, c.Explanation, c.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

// ListConflicts returns the conflicts of a media, newest first.
func (s *Store) ListConflicts(mediaID string) ([]model.Conflict, error) {
	rows, err := s.db.Query(
		`SELECT id,media_id,segment_id,actor_a,actor_b,base_version,attempted_version,explanation,created_at
		 FROM conflicts WHERE media_id=? ORDER BY created_at DESC`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Conflict
	for rows.Next() {
		c, err := scanConflict(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CountConflicts returns the total number of recorded conflicts.
func (s *Store) CountConflicts() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM conflicts`).Scan(&n)
	return n, err
}
