package store

import (
	"time"

	"task156-subtitleqa/internal/model"
)

func scanPublish(row interface{ Scan(...interface{}) error }) (model.PublishVersion, error) {
	var pv model.PublishVersion
	var frozen, withdrawn int
	var created string
	err := row.Scan(&pv.ID, &pv.MediaID, &pv.VersionNo, &pv.Label, &pv.Actor,
		&pv.Note, &frozen, &withdrawn, &pv.SnapshotJSON, &created)
	if err != nil {
		return pv, err
	}
	pv.Frozen = frozen != 0
	pv.Withdrawn = withdrawn != 0
	pv.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return pv, nil
}

// AddPublishVersion inserts a frozen snapshot.
func (s *Store) AddPublishVersion(pv model.PublishVersion) error {
	frozen, withdrawn := 0, 0
	if pv.Frozen {
		frozen = 1
	}
	if pv.Withdrawn {
		withdrawn = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO publish_versions (id,media_id,version_no,label,actor,note,frozen,withdrawn,snapshot_json,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		pv.ID, pv.MediaID, pv.VersionNo, pv.Label, pv.Actor, pv.Note, frozen, withdrawn,
		pv.SnapshotJSON, pv.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

// GetPublishVersion fetches a single publish version by id.
func (s *Store) GetPublishVersion(id string) (model.PublishVersion, error) {
	row := s.db.QueryRow(
		`SELECT id,media_id,version_no,label,actor,note,frozen,withdrawn,snapshot_json,created_at
		 FROM publish_versions WHERE id=?`, id)
	return scanPublish(row)
}

// ListPublishVersions returns the version history of a media, newest first.
func (s *Store) ListPublishVersions(mediaID string) ([]model.PublishVersion, error) {
	rows, err := s.db.Query(
		`SELECT id,media_id,version_no,label,actor,note,frozen,withdrawn,snapshot_json,created_at
		 FROM publish_versions WHERE media_id=? ORDER BY version_no DESC`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.PublishVersion
	for rows.Next() {
		pv, err := scanPublish(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, pv)
	}
	return out, rows.Err()
}

// CountPublishVersions returns how many versions a media has (used to assign the next number).
func (s *Store) CountPublishVersions(mediaID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM publish_versions WHERE media_id=?`, mediaID).Scan(&n)
	return n, err
}

// MarkWithdrawn flags a publish version as retracted.
func (s *Store) MarkWithdrawn(id string) error {
	_, err := s.db.Exec(`UPDATE publish_versions SET withdrawn=1 WHERE id=?`, id)
	return err
}

// AddWithdrawal records a retraction.
func (s *Store) AddWithdrawal(w model.Withdrawal) error {
	_, err := s.db.Exec(
		`INSERT INTO withdrawals (id,publish_version_id,media_id,actor,reason,created_at)
		 VALUES (?,?,?,?,?,?)`,
		w.ID, w.PublishVersionID, w.MediaID, w.Actor, w.Reason,
		w.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

// CountWithdrawals returns the total number of retractions.
func (s *Store) CountWithdrawals() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM withdrawals`).Scan(&n)
	return n, err
}
