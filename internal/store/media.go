package store

import (
	"time"

	"task156-subtitleqa/internal/model"
)

func scanMedia(row interface{ Scan(...interface{}) error }) (model.Media, error) {
	var m model.Media
	var created, updated string
	err := row.Scan(&m.ID, &m.Title, &m.Language, &m.DurationMs, &m.FrameRate,
		&m.SourceURL, &m.Checksum, &m.Status, &created, &updated)
	if err != nil {
		return m, err
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, created)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return m, nil
}

// CreateMedia inserts a new media row in draft status.
func (s *Store) CreateMedia(m model.Media) error {
	_, err := s.db.Exec(
		`INSERT INTO media (id,title,language,duration_ms,frame_rate,source_url,checksum,status,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		m.ID, m.Title, m.Language, m.DurationMs, m.FrameRate, m.SourceURL, m.Checksum,
		string(m.Status), m.CreatedAt.UTC().Format(time.RFC3339), m.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

// GetMedia fetches a single media by id.
func (s *Store) GetMedia(id string) (model.Media, error) {
	row := s.db.QueryRow(
		`SELECT id,title,language,duration_ms,frame_rate,source_url,checksum,status,created_at,updated_at
		 FROM media WHERE id=?`, id)
	return scanMedia(row)
}

// ListMedia returns every media ordered by creation time.
func (s *Store) ListMedia() ([]model.Media, error) {
	rows, err := s.db.Query(
		`SELECT id,title,language,duration_ms,frame_rate,source_url,checksum,status,created_at,updated_at
		 FROM media ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Media
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateMediaStatus changes a media's lifecycle status.
func (s *Store) UpdateMediaStatus(id string, status model.MediaStatus) error {
	_, err := s.db.Exec(`UPDATE media SET status=?, updated_at=? WHERE id=?`,
		string(status), now(), id)
	return err
}

// CountMedia returns the total number of media rows.
func (s *Store) CountMedia() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM media`).Scan(&n)
	return n, err
}

// CountSegments returns the number of segments for a media.
func (s *Store) CountSegments(mediaID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM segments WHERE media_id=?`, mediaID).Scan(&n)
	return n, err
}
