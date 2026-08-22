package store

import (
	"time"

	"task156-subtitleqa/internal/model"
)

func scanSpeaker(row interface{ Scan(...interface{}) error }) (model.Speaker, error) {
	var sp model.Speaker
	var created string
	err := row.Scan(&sp.ID, &sp.MediaID, &sp.Label, &sp.Color, &created)
	if err != nil {
		return sp, err
	}
	sp.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return sp, nil
}

// CreateSpeaker inserts a new speaker for a media.
func (s *Store) CreateSpeaker(sp model.Speaker) error {
	_, err := s.db.Exec(
		`INSERT INTO speakers (id,media_id,label,color,created_at) VALUES (?,?,?,?,?)`,
		sp.ID, sp.MediaID, sp.Label, sp.Color, sp.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

// GetSpeaker fetches a single speaker by id.
func (s *Store) GetSpeaker(id string) (model.Speaker, error) {
	row := s.db.QueryRow(
		`SELECT id,media_id,label,color,created_at FROM speakers WHERE id=?`, id)
	return scanSpeaker(row)
}

// ListSpeakers returns the speakers of a media.
func (s *Store) ListSpeakers(mediaID string) ([]model.Speaker, error) {
	rows, err := s.db.Query(
		`SELECT id,media_id,label,color,created_at FROM speakers WHERE media_id=? ORDER BY label ASC`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Speaker
	for rows.Next() {
		sp, err := scanSpeaker(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

// CountSpeakers returns the total number of speakers.
func (s *Store) CountSpeakers() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM speakers`).Scan(&n)
	return n, err
}
