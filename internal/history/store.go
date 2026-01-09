package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Status represents the outcome of an apply.
type Status string

const (
	StatusSuccess  Status = "success"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
)

// Entry represents a stored apply history record.
type Entry struct {
	ID         int64
	StartedAt  time.Time
	FinishedAt time.Time
	Duration   time.Duration
	Status     Status
	Summary    string
	Error      string
	WorkDir    string
	Output     string
}

// Store persists apply history in SQLite.
type Store struct {
	db *sql.DB
}

// DefaultPath returns the default history DB path.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tftui", "history.db"), nil
}

// OpenDefault opens the default history store.
func OpenDefault() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return Open(path)
}

// Open opens or creates the history store.
func Open(path string) (*Store, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create history dir: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open history db: %w", err)
	}

	store := &Store{db: db}
	if err := store.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// RecordApply writes an apply history entry.
func (s *Store) RecordApply(entry Entry) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO apply_history (started_at, finished_at, duration_ms, status, summary, error, workdir, output_text)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.StartedAt.UTC().Format(time.RFC3339Nano),
		entry.FinishedAt.UTC().Format(time.RFC3339Nano),
		entry.Duration.Milliseconds(),
		string(entry.Status),
		entry.Summary,
		entry.Error,
		entry.WorkDir,
		entry.Output,
	)
	return err
}

// ListRecent returns the most recent apply history entries.
func (s *Store) ListRecent(limit int) ([]Entry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(
		`SELECT id, started_at, finished_at, duration_ms, status, summary, error, workdir
		 FROM apply_history
		 ORDER BY finished_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		var started string
		var finished string
		var durationMs int64
		var status string
		if err := rows.Scan(&entry.ID, &started, &finished, &durationMs, &status, &entry.Summary, &entry.Error, &entry.WorkDir); err != nil {
			return nil, err
		}
		entry.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
		entry.FinishedAt, _ = time.Parse(time.RFC3339Nano, finished)
		entry.StartedAt = entry.StartedAt.Local()
		entry.FinishedAt = entry.FinishedAt.Local()
		entry.Duration = time.Duration(durationMs) * time.Millisecond
		entry.Status = Status(status)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// GetByID returns a history entry by ID.
func (s *Store) GetByID(id int64) (Entry, error) {
	var entry Entry
	if s == nil || s.db == nil {
		return entry, nil
	}
	row := s.db.QueryRow(
		`SELECT id, started_at, finished_at, duration_ms, status, summary, error, workdir, output_text
		 FROM apply_history
		 WHERE id = ?`,
		id,
	)
	var started string
	var finished string
	var durationMs int64
	var status string
	if err := row.Scan(&entry.ID, &started, &finished, &durationMs, &status, &entry.Summary, &entry.Error, &entry.WorkDir, &entry.Output); err != nil {
		return entry, err
	}
	entry.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
	entry.FinishedAt, _ = time.Parse(time.RFC3339Nano, finished)
	entry.StartedAt = entry.StartedAt.Local()
	entry.FinishedAt = entry.FinishedAt.Local()
	entry.Duration = time.Duration(durationMs) * time.Millisecond
	entry.Status = Status(status)
	return entry, nil
}

func (s *Store) ensureSchema() error {
	_, err := s.db.Exec(
		`CREATE TABLE IF NOT EXISTS apply_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			status TEXT NOT NULL,
			summary TEXT,
			error TEXT,
			workdir TEXT,
			output_text TEXT
		)`,
	)
	if err != nil {
		return fmt.Errorf("init history schema: %w", err)
	}
	if err := s.ensureColumn("apply_history", "output_text", "TEXT"); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureColumn(table, column, columnType string) error {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dfltValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnType))
	return err
}
