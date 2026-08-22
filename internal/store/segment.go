package store

import (
	"database/sql"
	"time"

	"task156-subtitleqa/internal/model"
)

func scanSegment(row interface{ Scan(...interface{}) error }) (model.Segment, error) {
	var seg model.Segment
	var desc int
	var speaker sql.NullString
	var created, updated string
	err := row.Scan(&seg.ID, &seg.MediaID, &seg.Index, &seg.StartMs, &seg.EndMs, &seg.Text,
		&speaker, &desc, &seg.Language, &seg.Version, &seg.Status, &created, &updated)
	if err != nil {
		return seg, err
	}
	seg.SpeakerID = speaker.String
	seg.IsDescriptive = desc != 0
	seg.CreatedAt, _ = time.Parse(time.RFC3339, created)
	seg.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return seg, nil
}

// CreateSegments inserts many segments in a single transaction.
func (s *Store) CreateSegments(segs []model.Segment) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO segments (id,media_id,idx,start_ms,end_ms,text,speaker_id,is_descriptive,language,version,status,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, seg := range segs {
		desc := 0
		if seg.IsDescriptive {
			desc = 1
		}
		speaker := sqlNullString(seg.SpeakerID)
		_, err := stmt.Exec(seg.ID, seg.MediaID, seg.Index, seg.StartMs, seg.EndMs, seg.Text,
			speaker, desc, seg.Language, seg.Version, string(seg.Status),
			seg.CreatedAt.UTC().Format(time.RFC3339), seg.UpdatedAt.UTC().Format(time.RFC3339))
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// GetSegment fetches a single segment by id.
func (s *Store) GetSegment(id string) (model.Segment, error) {
	row := s.db.QueryRow(
		`SELECT id,media_id,idx,start_ms,end_ms,text,speaker_id,is_descriptive,language,version,status,created_at,updated_at
		 FROM segments WHERE id=?`, id)
	return scanSegment(row)
}

// ListSegments returns the segments of a media ordered by index then start time.
func (s *Store) ListSegments(mediaID string) ([]model.Segment, error) {
	rows, err := s.db.Query(
		`SELECT id,media_id,idx,start_ms,end_ms,text,speaker_id,is_descriptive,language,version,status,created_at,updated_at
		 FROM segments WHERE media_id=? ORDER BY idx ASC, start_ms ASC`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Segment
	for rows.Next() {
		seg, err := scanSegment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, seg)
	}
	return out, rows.Err()
}

// UpdateSegmentStatus changes a segment's lifecycle status (e.g. mark it as
// conflicting after a rejected concurrent edit).
func (s *Store) UpdateSegmentStatus(id string, status model.SegmentStatus) error {
	_, err := s.db.Exec(`UPDATE segments SET status=?, updated_at=? WHERE id=?`,
		string(status), now(), id)
	return err
}

// UpdateSegment replaces a segment's mutable fields and bumps its version. It is
// the single write path for editor edits, which is what makes the optimistic
// concurrency check reliable.
func (s *Store) UpdateSegment(seg model.Segment) error {
	desc := 0
	if seg.IsDescriptive {
		desc = 1
	}
	speaker := sqlNullString(seg.SpeakerID)
	_, err := s.db.Exec(
		`UPDATE segments SET idx=?, start_ms=?, end_ms=?, text=?, speaker_id=?, is_descriptive=?, language=?, version=?, status=?, updated_at=? WHERE id=?`,
		seg.Index, seg.StartMs, seg.EndMs, seg.Text, speaker, desc, seg.Language,
		seg.Version, string(seg.Status), seg.UpdatedAt.UTC().Format(time.RFC3339), seg.ID)
	return err
}
