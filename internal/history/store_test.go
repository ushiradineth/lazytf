package history

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRecordAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	started := time.Now().UTC().Add(-2 * time.Second)
	finished := time.Now().UTC()
	entry := Entry{
		StartedAt:  started,
		FinishedAt: finished,
		Duration:   2 * time.Second,
		Status:     StatusSuccess,
		Summary:    "+ 1 to create",
		WorkDir:    "/tmp",
		Output:     "plan output text",
	}

	if err := store.RecordApply(entry); err != nil {
		t.Fatalf("record apply: %v", err)
	}

	entries, err := store.ListRecent(5)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	loaded, err := store.GetByID(entries[0].ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded.Output != entry.Output {
		t.Fatalf("expected output %q, got %q", entry.Output, loaded.Output)
	}
	if !loaded.StartedAt.UTC().Equal(started) {
		t.Fatalf("expected started at %v, got %v", started, loaded.StartedAt)
	}
	if loaded.StartedAt.Location() != time.Local {
		t.Fatalf("expected local time location, got %v", loaded.StartedAt.Location())
	}
	if !loaded.FinishedAt.UTC().Equal(finished) {
		t.Fatalf("expected finished at %v, got %v", finished, loaded.FinishedAt)
	}
}

func TestStoreMigratesOutputColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE apply_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		started_at TEXT NOT NULL,
		finished_at TEXT NOT NULL,
		duration_ms INTEGER NOT NULL,
		status TEXT NOT NULL,
		summary TEXT,
		error TEXT,
		workdir TEXT
	)`)
	if err != nil {
		t.Fatalf("create old schema: %v", err)
	}
	_ = db.Close()

	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	entry := Entry{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Duration:   time.Second,
		Status:     StatusSuccess,
		Summary:    "ok",
		Output:     "output text",
	}
	if err := store.RecordApply(entry); err != nil {
		t.Fatalf("record apply after migration: %v", err)
	}
}
