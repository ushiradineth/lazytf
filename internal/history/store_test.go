package history

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
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
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	started := time.Now().UTC().Add(-2 * time.Second)
	finished := time.Now().UTC()
	entry := Entry{
		StartedAt:   started,
		FinishedAt:  finished,
		Duration:    2 * time.Second,
		Status:      StatusSuccess,
		Summary:     "+ 1 to create",
		WorkDir:     "/tmp",
		Environment: "dev",
		Output:      "plan output text",
	}

	if err := store.RecordApply(entry); err != nil {
		t.Fatalf("record apply: %v", err)
	}

	entries, err := store.ListRecentForEnvironment("dev", 5)
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
	if loaded.Environment != entry.Environment {
		t.Fatalf("expected environment %q, got %q", entry.Environment, loaded.Environment)
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
	_, err = db.ExecContext(context.Background(), `CREATE TABLE apply_history (
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
	if closeErr := db.Close(); closeErr != nil {
		t.Fatalf("close db: %v", closeErr)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
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

func TestStoreRecordAndQueryOperation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	started := time.Now().Add(-1 * time.Minute).UTC()
	finished := time.Now().UTC()
	entry := OperationEntry{
		StartedAt:   started,
		FinishedAt:  finished,
		Duration:    time.Minute,
		Action:      "plan",
		Command:     "terraform plan -out=tfplan",
		ExitCode:    0,
		Status:      StatusSuccess,
		Summary:     "+ 1 to add",
		User:        "alice",
		Environment: "prod",
		Output:      strings.Repeat("ok\n", 70000),
	}

	if err := store.RecordOperation(entry); err != nil {
		t.Fatalf("record operation: %v", err)
	}

	entries, err := store.QueryOperations(OperationFilter{Action: "plan", Environment: "prod", User: "alice", Limit: 5})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Output == "" {
		t.Fatalf("expected output to be loaded")
	}
	if entries[0].ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", entries[0].ExitCode)
	}

	loaded, err := store.GetOperationByID(entries[0].ID)
	if err != nil {
		t.Fatalf("get operation by id: %v", err)
	}
	if loaded.Command != entry.Command {
		t.Fatalf("expected command %q, got %q", entry.Command, loaded.Command)
	}
	if !loaded.StartedAt.UTC().Equal(started) {
		t.Fatalf("expected started at %v, got %v", started, loaded.StartedAt)
	}
}

func TestStoreGetOperationsForApply(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	// Create apply entry with specific time window
	applyStarted := time.Now().Add(-10 * time.Minute).UTC()
	applyFinished := time.Now().Add(-5 * time.Minute).UTC()
	applyEntry := Entry{
		StartedAt:   applyStarted,
		FinishedAt:  applyFinished,
		Duration:    5 * time.Minute,
		Status:      StatusSuccess,
		Summary:     "Applied",
		Environment: "test-env",
	}
	if err := store.RecordApply(applyEntry); err != nil {
		t.Fatalf("record apply: %v", err)
	}

	// Create plan operation within time window
	planOp := OperationEntry{
		StartedAt:   applyStarted.Add(-30 * time.Minute),
		FinishedAt:  applyStarted.Add(-25 * time.Minute),
		Duration:    5 * time.Minute,
		Action:      "plan",
		Environment: "test-env",
		Output:      "Plan: 1 to add",
	}
	if err := store.RecordOperation(planOp); err != nil {
		t.Fatalf("record plan operation: %v", err)
	}

	// Create apply operation within time window
	applyOp := OperationEntry{
		StartedAt:   applyStarted,
		FinishedAt:  applyFinished,
		Duration:    5 * time.Minute,
		Action:      "apply",
		Environment: "test-env",
		Output:      "Apply complete!",
	}
	if err := store.RecordOperation(applyOp); err != nil {
		t.Fatalf("record apply operation: %v", err)
	}

	// Query operations for the apply entry
	operations, err := store.GetOperationsForApply(applyEntry)
	if err != nil {
		t.Fatalf("get operations for apply: %v", err)
	}

	if len(operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(operations))
	}

	// Verify we got both plan and apply operations
	foundPlan := false
	foundApply := false
	for _, op := range operations {
		switch op.Action {
		case "plan":
			foundPlan = true
		case "apply":
			foundApply = true
		}
	}
	if !foundPlan {
		t.Error("expected to find plan operation")
	}
	if !foundApply {
		t.Error("expected to find apply operation")
	}
}

func TestStoreGetOperationsForApplyNilStore(t *testing.T) {
	var store *Store
	operations, err := store.GetOperationsForApply(Entry{})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if operations != nil {
		t.Errorf("expected nil operations, got %v", operations)
	}
}

func TestStoreListRecentForEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	entryDev := Entry{
		StartedAt:   time.Now(),
		FinishedAt:  time.Now(),
		Duration:    time.Second,
		Status:      StatusSuccess,
		Summary:     "dev apply",
		Environment: "dev",
	}
	entryProd := Entry{
		StartedAt:   time.Now(),
		FinishedAt:  time.Now(),
		Duration:    time.Second,
		Status:      StatusSuccess,
		Summary:     "prod apply",
		Environment: "prod",
	}

	if err := store.RecordApply(entryDev); err != nil {
		t.Fatalf("record dev apply: %v", err)
	}
	if err := store.RecordApply(entryProd); err != nil {
		t.Fatalf("record prod apply: %v", err)
	}

	entries, err := store.ListRecentForEnvironment("dev", 5)
	if err != nil {
		t.Fatalf("list recent dev: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 dev entry, got %d", len(entries))
	}
	if entries[0].Environment != "dev" {
		t.Fatalf("expected dev environment, got %q", entries[0].Environment)
	}
}

func TestDefaultPathAndOpenDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	if path == "" {
		t.Fatalf("expected default path")
	}

	store, err := OpenDefault()
	if err != nil {
		t.Fatalf("open default: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})
}

func TestWithCompressionThreshold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path, WithCompressionThreshold(1234))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})
	if store.compressionThreshold != 1234 {
		t.Fatalf("expected compression threshold to be set")
	}
}

func TestOpenTildePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join("~", "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open tilde path: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})
}

func TestListRecent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	// Record multiple entries
	for i := range 5 {
		entry := Entry{
			StartedAt:   time.Now().Add(time.Duration(-i) * time.Minute),
			FinishedAt:  time.Now().Add(time.Duration(-i) * time.Minute),
			Duration:    time.Second,
			Status:      StatusSuccess,
			Summary:     "apply",
			Environment: "env",
		}
		if err := store.RecordApply(entry); err != nil {
			t.Fatalf("record apply: %v", err)
		}
	}

	// Test with default limit
	entries, err := store.ListRecent(0)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}

	// Test with limit
	entries, err = store.ListRecent(3)
	if err != nil {
		t.Fatalf("list recent with limit: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestListRecentNilStore(t *testing.T) {
	var store *Store
	entries, err := store.ListRecent(10)
	if err != nil {
		t.Errorf("expected no error for nil store, got %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for nil store")
	}
}

func TestListRecentForEnvironmentEmptyEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	entry := Entry{
		StartedAt:   time.Now(),
		FinishedAt:  time.Now(),
		Duration:    time.Second,
		Status:      StatusSuccess,
		Environment: "test",
	}
	if err := store.RecordApply(entry); err != nil {
		t.Fatalf("record apply: %v", err)
	}

	// Empty environment should fall back to ListRecent
	entries, err := store.ListRecentForEnvironment("", 5)
	if err != nil {
		t.Fatalf("list recent for empty env: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	// Whitespace-only environment
	entries, err = store.ListRecentForEnvironment("   ", 5)
	if err != nil {
		t.Fatalf("list recent for whitespace env: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestListRecentForEnvironmentNilStore(t *testing.T) {
	var store *Store
	entries, err := store.ListRecentForEnvironment("test", 10)
	if err != nil {
		t.Errorf("expected no error for nil store, got %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for nil store")
	}
}

func TestStoreCloseNil(t *testing.T) {
	var store *Store
	err := store.Close()
	if err != nil {
		t.Errorf("expected nil error for nil store close, got %v", err)
	}
}

func TestQueryOperationsNoFilters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	entry := OperationEntry{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Duration:   time.Second,
		Action:     "plan",
		Command:    "terraform plan",
		ExitCode:   0,
		Status:     StatusSuccess,
	}
	if err := store.RecordOperation(entry); err != nil {
		t.Fatalf("record operation: %v", err)
	}

	// Query with empty filter
	entries, err := store.QueryOperations(OperationFilter{})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestQueryOperationsNilStore(t *testing.T) {
	var store *Store
	entries, err := store.QueryOperations(OperationFilter{})
	if err != nil {
		t.Errorf("expected no error for nil store, got %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for nil store")
	}
}

func TestGetByIDNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	_, err = store.GetByID(99999)
	// Should return sql.ErrNoRows for non-existent ID
	if err == nil {
		t.Error("expected error for non-existent ID")
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetOperationByIDNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	_, err = store.GetOperationByID(99999)
	// Should return sql.ErrNoRows for non-existent ID
	if err == nil {
		t.Error("expected error for non-existent ID")
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestRecordApplyNilStore(t *testing.T) {
	var store *Store
	err := store.RecordApply(Entry{})
	if err != nil {
		t.Errorf("expected nil error for nil store, got %v", err)
	}
}

func TestRecordOperationNilStore(t *testing.T) {
	var store *Store
	err := store.RecordOperation(OperationEntry{})
	if err != nil {
		t.Errorf("expected nil error for nil store, got %v", err)
	}
}

func TestGetByIDNilStore(t *testing.T) {
	var store *Store
	entry, err := store.GetByID(1)
	if err != nil {
		t.Errorf("expected no error for nil store, got %v", err)
	}
	// Should return empty entry
	if entry.ID != 0 {
		t.Errorf("expected empty entry for nil store")
	}
}

func TestGetOperationByIDNilStore(t *testing.T) {
	var store *Store
	entry, err := store.GetOperationByID(1)
	if err != nil {
		t.Errorf("expected no error for nil store, got %v", err)
	}
	// Should return empty entry
	if entry.ID != 0 {
		t.Errorf("expected empty entry for nil store")
	}
}

func TestRecordApplyWithError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	entry := Entry{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Duration:   time.Second,
		Status:     StatusFailed,
		Error:      "something went wrong",
	}
	if err := store.RecordApply(entry); err != nil {
		t.Fatalf("record apply with error: %v", err)
	}

	entries, err := store.ListRecent(1)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status != StatusFailed {
		t.Errorf("expected status failed, got %s", entries[0].Status)
	}
}

func TestRecordOperationWithFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	entry := OperationEntry{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Duration:   time.Second,
		Action:     "apply",
		Command:    "terraform apply",
		ExitCode:   1,
		Status:     StatusFailed,
	}
	if err := store.RecordOperation(entry); err != nil {
		t.Fatalf("record operation: %v", err)
	}

	entries, err := store.QueryOperations(OperationFilter{Action: "apply", Limit: 1})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", entries[0].ExitCode)
	}
}

func TestWithCompressionThresholdZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path, WithCompressionThreshold(0))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})
	// Zero should not change the default
	if store.compressionThreshold != defaultCompressionThreshold {
		t.Errorf("expected default threshold, got %d", store.compressionThreshold)
	}
}

func TestQueryOperationsWithDateFilter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	now := time.Now()
	// Record entries at different times
	oldEntry := OperationEntry{
		StartedAt:  now.Add(-48 * time.Hour),
		FinishedAt: now.Add(-48 * time.Hour),
		Duration:   time.Second,
		Action:     "plan",
		Status:     StatusSuccess,
	}
	newEntry := OperationEntry{
		StartedAt:  now.Add(-1 * time.Hour),
		FinishedAt: now.Add(-1 * time.Hour),
		Duration:   time.Second,
		Action:     "plan",
		Status:     StatusSuccess,
	}
	if err := store.RecordOperation(oldEntry); err != nil {
		t.Fatalf("record old: %v", err)
	}
	if err := store.RecordOperation(newEntry); err != nil {
		t.Fatalf("record new: %v", err)
	}

	// Query only entries after 24 hours ago
	entries, err := store.QueryOperations(OperationFilter{After: now.Add(-24 * time.Hour)})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 recent entry, got %d", len(entries))
	}

	// Query only entries before 24 hours ago
	entries, err = store.QueryOperations(OperationFilter{Before: now.Add(-24 * time.Hour)})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 old entry, got %d", len(entries))
	}
}

func TestDefaultPathReturnsValidPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !strings.Contains(path, "lazytf") {
		t.Errorf("expected path to contain 'lazytf', got %s", path)
	}
}

func TestParseLocalTimeVariousFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
	}{
		{"RFC3339", "2024-01-02T15:04:05Z", false},
		{"RFC3339Nano", "2024-01-02T15:04:05.123456789Z", false},
		{"empty string", "", true},
		{"invalid", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLocalTime(tt.input)
			if tt.wantZero && !result.IsZero() {
				t.Error("expected zero time")
			}
			if !tt.wantZero && result.IsZero() {
				t.Error("expected non-zero time")
			}
		})
	}
}

func TestListRecentWithLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	defer store.Close()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		entry := Entry{
			StartedAt:   time.Now(),
			FinishedAt:  time.Now(),
			Status:      StatusSuccess,
			Summary:     "test",
			Environment: "dev",
		}
		if err := store.RecordApply(entry); err != nil {
			t.Fatalf("record apply: %v", err)
		}
	}

	// List with limit
	entries, err := store.ListRecent(2)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestListRecentForEnvironmentNoResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	defer store.Close()

	// List for non-existent environment
	entries, err := store.ListRecentForEnvironment("nonexistent", 10)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestOpenWithTildePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Open with tilde path
	store, err := Open("~/test.db")
	if err != nil {
		t.Fatalf("open with tilde path: %v", err)
	}
	store.Close()
	// Store should be opened successfully
}

func TestRecordOperationWithOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	defer store.Close()

	entry := OperationEntry{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Duration:   time.Second,
		Action:     "plan",
		Command:    "terraform plan",
		ExitCode:   0,
		Status:     StatusSuccess,
		Summary:    "1 to add",
		Output:     "Plan output here",
	}

	if err := store.RecordOperation(entry); err != nil {
		t.Fatalf("record operation: %v", err)
	}

	// Verify it was stored
	entries, err := store.QueryOperations(OperationFilter{Action: "plan", Limit: 1})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestOpenWithAbsolutePathSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "test.db")
	store, err := Open(absPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	store.Close()
}

func TestListRecentForEnvironmentMultipleEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	defer store.Close()

	// Record entries for different environments
	envs := []string{"dev", "staging", "prod", "dev", "dev"}
	for i, env := range envs {
		entry := Entry{
			StartedAt:   time.Now().Add(time.Duration(i) * time.Second),
			FinishedAt:  time.Now().Add(time.Duration(i+1) * time.Second),
			Duration:    time.Second,
			Status:      StatusSuccess,
			Environment: env,
		}
		if err := store.RecordApply(entry); err != nil {
			t.Fatalf("record apply: %v", err)
		}
	}

	// Query for dev - should return 3 entries
	entries, err := store.ListRecentForEnvironment("dev", 10)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries for dev, got %d", len(entries))
	}

	// Query for staging - should return 1 entry
	entries, err = store.ListRecentForEnvironment("staging", 10)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry for staging, got %d", len(entries))
	}
}

func TestQueryOperationsWithActionFilter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	defer store.Close()

	actions := []string{"plan", "apply", "plan", "refresh", "plan"}
	for i, action := range actions {
		entry := OperationEntry{
			StartedAt:  time.Now().Add(time.Duration(i) * time.Second),
			FinishedAt: time.Now().Add(time.Duration(i+1) * time.Second),
			Duration:   time.Second,
			Action:     action,
			Status:     StatusSuccess,
		}
		if err := store.RecordOperation(entry); err != nil {
			t.Fatalf("record operation: %v", err)
		}
	}

	// Query for plan actions
	entries, err := store.QueryOperations(OperationFilter{Action: "plan"})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 plan entries, got %d", len(entries))
	}

	// Query for apply actions
	entries, err = store.QueryOperations(OperationFilter{Action: "apply"})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 apply entry, got %d", len(entries))
	}
}
