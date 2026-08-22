// Package store provides SQLite-backed persistence for every entity in the
// subtitle proofreading workbench. It uses the pure-Go modernc.org/sqlite
// driver (CGO-free, offline-buildable) and stores all state in a single file so
// the service can be restarted and resume exactly where it left off.
package store

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a *sql.DB and exposes CRUD helpers. The connection pool is capped
// at one writer because SQLite serialises writes; this keeps concurrent edits
// safe without external locking.
type Store struct {
	db   *sql.DB
	path string
}

// Open connects to (and creates if needed) the SQLite database at path and runs
// the schema migration. It is safe to call on an existing database.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, path: path}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

// DB exposes the raw handle for packages that need ad-hoc queries.
func (s *Store) DB() *sql.DB { return s.db }

// Path returns the database file location.
func (s *Store) Path() string { return s.path }

func (s *Store) migrate() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS media (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			language TEXT NOT NULL DEFAULT 'zh',
			duration_ms INTEGER NOT NULL DEFAULT 0,
			frame_rate REAL NOT NULL DEFAULT 0,
			source_url TEXT NOT NULL DEFAULT '',
			checksum TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'draft',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS speakers (
			id TEXT PRIMARY KEY,
			media_id TEXT NOT NULL,
			label TEXT NOT NULL,
			color TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS segments (
			id TEXT PRIMARY KEY,
			media_id TEXT NOT NULL,
			idx INTEGER NOT NULL,
			start_ms INTEGER NOT NULL,
			end_ms INTEGER NOT NULL,
			text TEXT NOT NULL DEFAULT '',
			speaker_id TEXT,
			is_descriptive INTEGER NOT NULL DEFAULT 0,
			language TEXT NOT NULL DEFAULT '',
			version INTEGER NOT NULL DEFAULT 1,
			status TEXT NOT NULL DEFAULT 'draft',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS revisions (
			id TEXT PRIMARY KEY,
			media_id TEXT NOT NULL,
			segment_id TEXT NOT NULL,
			actor TEXT NOT NULL,
			op_type TEXT NOT NULL,
			base_version INTEGER NOT NULL,
			before_json TEXT NOT NULL DEFAULT '{}',
			after_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS quality_checks (
			id TEXT PRIMARY KEY,
			media_id TEXT NOT NULL,
			segment_id TEXT,
			rule TEXT NOT NULL,
			severity TEXT NOT NULL,
			message TEXT NOT NULL,
			detected_at TEXT NOT NULL,
			resolved INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS publish_versions (
			id TEXT PRIMARY KEY,
			media_id TEXT NOT NULL,
			version_no INTEGER NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			actor TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			frozen INTEGER NOT NULL DEFAULT 1,
			withdrawn INTEGER NOT NULL DEFAULT 0,
			snapshot_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS withdrawals (
			id TEXT PRIMARY KEY,
			publish_version_id TEXT NOT NULL,
			media_id TEXT NOT NULL,
			actor TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS conflicts (
			id TEXT PRIMARY KEY,
			media_id TEXT NOT NULL,
			segment_id TEXT NOT NULL,
			actor_a TEXT NOT NULL,
			actor_b TEXT NOT NULL,
			base_version INTEGER NOT NULL,
			attempted_version INTEGER NOT NULL,
			explanation TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_segments_media ON segments(media_id, idx)`,
		`CREATE INDEX IF NOT EXISTS idx_quality_media ON quality_checks(media_id)`,
		`CREATE INDEX IF NOT EXISTS idx_revisions_segment ON revisions(segment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_publish_media ON publish_versions(media_id, version_no)`,
		`CREATE INDEX IF NOT EXISTS idx_conflicts_media ON conflicts(media_id)`,
	}
	for _, stmt := range schema {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// now returns the current UTC time formatted for storage.
func now() string { return time.Now().UTC().Format(time.RFC3339) }
