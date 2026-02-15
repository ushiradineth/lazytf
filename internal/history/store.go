package history

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
	ID          int64
	StartedAt   time.Time
	FinishedAt  time.Time
	Duration    time.Duration
	Status      Status
	Summary     string
	Error       string
	WorkDir     string
	Environment string
	Output      string
}

// OperationEntry represents a stored terraform operation.
type OperationEntry struct {
	ID          int64
	StartedAt   time.Time
	FinishedAt  time.Time
	Duration    time.Duration
	Action      string
	Command     string
	ExitCode    int
	Status      Status
	Summary     string
	User        string
	Environment string
	Output      string
}

// OperationFilter narrows operation history queries.
type OperationFilter struct {
	After       time.Time
	Before      time.Time
	Action      string
	Environment string
	User        string
	Limit       int
}

// Store persists apply history in SQLite.
type Store struct {
	db                   *sql.DB
	compressionThreshold int
	retentionDays        int
	maxEntries           int
	redactSensitive      bool
}

const defaultCompressionThreshold = 64 * 1024
const defaultRetentionDays = 90
const defaultMaxEntries = 10000

// StoreOption configures Store behavior.
type StoreOption func(*Store)

// WithCompressionThreshold sets the output compression threshold in bytes.
func WithCompressionThreshold(bytes int) StoreOption {
	return func(s *Store) {
		if bytes > 0 {
			s.compressionThreshold = bytes
		}
	}
}

// WithRetentionDays sets the number of days to keep history rows.
func WithRetentionDays(days int) StoreOption {
	return func(s *Store) {
		if days >= 0 {
			s.retentionDays = days
		}
	}
}

// WithMaxEntries sets the maximum number of rows kept per history table.
func WithMaxEntries(maxEntries int) StoreOption {
	return func(s *Store) {
		if maxEntries >= 0 {
			s.maxEntries = maxEntries
		}
	}
}

// WithSensitiveRedaction enables or disables secret redaction before persistence.
func WithSensitiveRedaction(enabled bool) StoreOption {
	return func(s *Store) {
		s.redactSensitive = enabled
	}
}

// DefaultPath returns the default history DB path.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lazytf", "history.db"), nil
}

// OpenDefault opens the default history store.
func OpenDefault(opts ...StoreOption) (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return Open(path, opts...)
}

// Open opens or creates the history store.
func Open(path string, opts ...StoreOption) (*Store, error) {
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

	// SQLite is single-writer, configure connection pool appropriately
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{
		db:                   db,
		compressionThreshold: defaultCompressionThreshold,
		retentionDays:        defaultRetentionDays,
		maxEntries:           defaultMaxEntries,
		redactSensitive:      true,
	}
	for _, opt := range opts {
		opt(store)
	}
	if err := store.ensureSchema(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			// Best effort close after schema failure.
			_ = closeErr
		}
		return nil, err
	}
	if err := store.enforceRetention(context.Background()); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			_ = closeErr
		}
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
	ctx := context.Background()
	entry.Summary = s.maybeRedact(entry.Summary)
	entry.Error = s.maybeRedact(entry.Error)
	entry.Output = s.maybeRedact(entry.Output)
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO apply_history (started_at, finished_at, duration_ms, status, summary, error, workdir, environment, output_text)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.StartedAt.UTC().Format(time.RFC3339Nano),
		entry.FinishedAt.UTC().Format(time.RFC3339Nano),
		entry.Duration.Milliseconds(),
		string(entry.Status),
		entry.Summary,
		entry.Error,
		entry.WorkDir,
		entry.Environment,
		entry.Output,
	)
	if err != nil {
		return err
	}
	return s.enforceRetention(ctx)
}

// RecordOperation writes a terraform operation entry.
func (s *Store) RecordOperation(entry OperationEntry) error {
	if s == nil || s.db == nil {
		return nil
	}
	entry.Command = s.maybeRedact(entry.Command)
	entry.Summary = s.maybeRedact(entry.Summary)
	entry.Output = s.maybeRedact(entry.Output)
	outputText, outputGzip, err := prepareOutput(entry.Output, s.compressionThreshold)
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO operations (started_at, finished_at, duration_ms, action, command, exit_code, status, summary, user_name, environment, output_text, output_gzip)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.StartedAt.UTC().Format(time.RFC3339Nano),
		entry.FinishedAt.UTC().Format(time.RFC3339Nano),
		entry.Duration.Milliseconds(),
		entry.Action,
		entry.Command,
		entry.ExitCode,
		string(entry.Status),
		entry.Summary,
		entry.User,
		entry.Environment,
		outputText,
		outputGzip,
	)
	if err != nil {
		return err
	}
	return s.enforceRetention(ctx)
}

func (s *Store) enforceRetention(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}

	if s.retentionDays > 0 {
		cutoff := time.Now().Add(-time.Duration(s.retentionDays) * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
		if _, err := s.db.ExecContext(ctx, `DELETE FROM apply_history WHERE finished_at < ?`, cutoff); err != nil {
			return fmt.Errorf("prune apply_history by retention: %w", err)
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM operations WHERE finished_at < ?`, cutoff); err != nil {
			return fmt.Errorf("prune operations by retention: %w", err)
		}
	}

	if s.maxEntries > 0 {
		if _, err := s.db.ExecContext(ctx,
			`DELETE FROM apply_history
			 WHERE id IN (
			   SELECT id FROM apply_history
			   ORDER BY finished_at DESC
			   LIMIT -1 OFFSET ?
			 )`,
			s.maxEntries,
		); err != nil {
			return fmt.Errorf("prune apply_history by max entries: %w", err)
		}
		if _, err := s.db.ExecContext(ctx,
			`DELETE FROM operations
			 WHERE id IN (
			   SELECT id FROM operations
			   ORDER BY finished_at DESC
			   LIMIT -1 OFFSET ?
			 )`,
			s.maxEntries,
		); err != nil {
			return fmt.Errorf("prune operations by max entries: %w", err)
		}
	}

	return nil
}

func (s *Store) maybeRedact(value string) string {
	if s == nil || !s.redactSensitive {
		return value
	}
	return redactSensitiveData(value)
}

var secretAssignmentPattern = regexp.MustCompile(`(?i)\b(password|passwd|pwd|token|secret|api[_-]?key|access[_-]?key|client[_-]?secret)\b\s*[:=]\s*("[^"]*"|'[^']*'|\S+)`)
var bearerTokenPattern = regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)(\S+)`)
var urlCredentialPattern = regexp.MustCompile(`(?i)(https?://[^\s:@/]+:)([^@\s/]+)(@)`)

func redactSensitiveData(input string) string {
	if strings.TrimSpace(input) == "" {
		return input
	}
	redacted := secretAssignmentPattern.ReplaceAllString(input, `${1}=[REDACTED]`)
	redacted = bearerTokenPattern.ReplaceAllString(redacted, `${1}[REDACTED]`)
	redacted = urlCredentialPattern.ReplaceAllString(redacted, `${1}[REDACTED]${3}`)
	return redacted
}

// scanEntry scans a single row into an Entry struct.
func scanEntry(rows *sql.Rows) (Entry, error) {
	var entry Entry
	var started string
	var finished string
	var durationMs int64
	var status string
	if err := rows.Scan(&entry.ID, &started, &finished, &durationMs, &status, &entry.Summary, &entry.Error, &entry.WorkDir, &entry.Environment); err != nil {
		return Entry{}, err
	}
	startedAt, parseErr := time.Parse(time.RFC3339Nano, started)
	if parseErr == nil {
		entry.StartedAt = startedAt.Local()
	}
	finishedAt, parseErr := time.Parse(time.RFC3339Nano, finished)
	if parseErr == nil {
		entry.FinishedAt = finishedAt.Local()
	}
	entry.Duration = time.Duration(durationMs) * time.Millisecond
	entry.Status = Status(status)
	return entry, nil
}

// ListRecent returns the most recent apply history entries.
func (s *Store) ListRecent(limit int) ([]Entry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, started_at, finished_at, duration_ms, status, summary, error, workdir, environment
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
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// ListRecentForEnvironment returns apply history entries for a specific environment.
func (s *Store) ListRecentForEnvironment(environment string, limit int) ([]Entry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if strings.TrimSpace(environment) == "" {
		return s.ListRecent(limit)
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, started_at, finished_at, duration_ms, status, summary, error, workdir, environment
		 FROM apply_history
		 WHERE environment = ?
		 ORDER BY finished_at DESC
		 LIMIT ?`,
		environment,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// QueryOperations returns operations matching the filter.
func (s *Store) QueryOperations(filter OperationFilter) ([]OperationEntry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	query, args := buildOperationQuery(filter)

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanOperationEntries(rows)
}

func buildOperationQuery(filter OperationFilter) (string, []any) {
	query := `SELECT id, started_at, finished_at, duration_ms, action, command, exit_code, status, summary, user_name, environment, output_text, output_gzip
		FROM operations`
	var args []any
	var clauses []string

	appendClause := func(condition bool, clause string, value any) {
		if !condition {
			return
		}
		clauses = append(clauses, clause)
		args = append(args, value)
	}

	appendClause(!filter.After.IsZero(), "started_at >= ?", filter.After.UTC().Format(time.RFC3339Nano))
	appendClause(!filter.Before.IsZero(), "started_at <= ?", filter.Before.UTC().Format(time.RFC3339Nano))
	appendClause(strings.TrimSpace(filter.Action) != "", "action = ?", filter.Action)
	appendClause(strings.TrimSpace(filter.Environment) != "", "environment = ?", filter.Environment)
	appendClause(strings.TrimSpace(filter.User) != "", "user_name = ?", filter.User)

	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY started_at DESC LIMIT ?"
	args = append(args, filter.Limit)

	return query, args
}

func scanOperationEntries(rows *sql.Rows) ([]OperationEntry, error) {
	var entries []OperationEntry
	for rows.Next() {
		entry, err := scanOperationEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func scanOperationEntry(rows *sql.Rows) (OperationEntry, error) {
	var entry OperationEntry
	var started string
	var finished string
	var durationMs int64
	var status string
	var outputText sql.NullString
	var outputGzip []byte
	if err := rows.Scan(
		&entry.ID,
		&started,
		&finished,
		&durationMs,
		&entry.Action,
		&entry.Command,
		&entry.ExitCode,
		&status,
		&entry.Summary,
		&entry.User,
		&entry.Environment,
		&outputText,
		&outputGzip,
	); err != nil {
		return entry, err
	}
	entry.StartedAt = parseLocalTime(started)
	entry.FinishedAt = parseLocalTime(finished)
	entry.Duration = time.Duration(durationMs) * time.Millisecond
	entry.Status = Status(status)
	output, err := loadOutput(outputText, outputGzip)
	if err != nil {
		return entry, err
	}
	entry.Output = output
	return entry, nil
}

func parseLocalTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.Local()
}

// GetByID returns a history entry by ID.
func (s *Store) GetByID(id int64) (Entry, error) {
	var entry Entry
	if s == nil || s.db == nil {
		return entry, nil
	}
	ctx := context.Background()
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, started_at, finished_at, duration_ms, status, summary, error, workdir, environment, output_text
		 FROM apply_history
		 WHERE id = ?`,
		id,
	)
	var started string
	var finished string
	var durationMs int64
	var status string
	if err := row.Scan(&entry.ID, &started, &finished, &durationMs, &status, &entry.Summary, &entry.Error, &entry.WorkDir, &entry.Environment, &entry.Output); err != nil {
		return entry, err
	}
	startedAt, parseErr := time.Parse(time.RFC3339Nano, started)
	if parseErr == nil {
		entry.StartedAt = startedAt.Local()
	}
	finishedAt, parseErr := time.Parse(time.RFC3339Nano, finished)
	if parseErr == nil {
		entry.FinishedAt = finishedAt.Local()
	}
	entry.Duration = time.Duration(durationMs) * time.Millisecond
	entry.Status = Status(status)
	return entry, nil
}

// GetOperationsForApply returns plan+apply operations related to an apply entry.
// It queries operations within a time window of the entry's execution.
func (s *Store) GetOperationsForApply(entry Entry) ([]OperationEntry, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	// Time window: 1 hour before entry started to 5 minutes after finished
	windowStart := entry.StartedAt.Add(-1 * time.Hour)
	windowEnd := entry.FinishedAt.Add(5 * time.Minute)

	filter := OperationFilter{
		After:       windowStart,
		Before:      windowEnd,
		Environment: entry.Environment,
		Limit:       10,
	}

	return s.QueryOperations(filter)
}

// GetOperationByID returns an operation entry by ID.
func (s *Store) GetOperationByID(id int64) (OperationEntry, error) {
	var entry OperationEntry
	if s == nil || s.db == nil {
		return entry, nil
	}
	ctx := context.Background()
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, started_at, finished_at, duration_ms, action, command, exit_code, status, summary, user_name, environment, output_text, output_gzip
		 FROM operations
		 WHERE id = ?`,
		id,
	)
	var started string
	var finished string
	var durationMs int64
	var status string
	var outputText sql.NullString
	var outputGzip []byte
	if err := row.Scan(
		&entry.ID,
		&started,
		&finished,
		&durationMs,
		&entry.Action,
		&entry.Command,
		&entry.ExitCode,
		&status,
		&entry.Summary,
		&entry.User,
		&entry.Environment,
		&outputText,
		&outputGzip,
	); err != nil {
		return entry, err
	}
	startedAt, parseErr := time.Parse(time.RFC3339Nano, started)
	if parseErr == nil {
		entry.StartedAt = startedAt.Local()
	}
	finishedAt, parseErr := time.Parse(time.RFC3339Nano, finished)
	if parseErr == nil {
		entry.FinishedAt = finishedAt.Local()
	}
	entry.Duration = time.Duration(durationMs) * time.Millisecond
	entry.Status = Status(status)
	var err error
	entry.Output, err = loadOutput(outputText, outputGzip)
	if err != nil {
		return entry, err
	}
	return entry, nil
}

func (s *Store) ensureSchema() error {
	ctx := context.Background()
	_, err := s.db.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS apply_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			status TEXT NOT NULL,
			summary TEXT,
			error TEXT,
			workdir TEXT,
			environment TEXT,
			output_text TEXT
		)`,
	)
	if err != nil {
		return fmt.Errorf("init history schema: %w", err)
	}
	if err := s.ensureColumn("apply_history", "output_text", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("apply_history", "environment", "TEXT"); err != nil {
		return err
	}

	return s.ensureOperationsSchema()
}

func (s *Store) ensureOperationsSchema() error {
	ctx := context.Background()
	_, err := s.db.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS operations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			action TEXT NOT NULL,
			command TEXT,
			exit_code INTEGER,
			status TEXT,
			summary TEXT,
			user_name TEXT,
			environment TEXT,
			output_text TEXT,
			output_gzip BLOB
		)`,
	)
	if err != nil {
		return fmt.Errorf("init operations schema: %w", err)
	}
	if err := s.ensureColumn("operations", "output_gzip", "BLOB"); err != nil {
		return err
	}
	if err := s.ensureColumn("operations", "environment", "TEXT"); err != nil {
		return err
	}
	return s.ensureColumn("operations", "user_name", "TEXT")
}

func (s *Store) ensureColumn(table, column, columnType string) error {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
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

	_, err = s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnType))
	return err
}

func prepareOutput(output string, threshold int) (string, []byte, error) {
	if output == "" {
		return "", nil, nil
	}
	if len(output) < threshold {
		return output, nil, nil
	}
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write([]byte(output)); err != nil {
		if closeErr := writer.Close(); closeErr != nil {
			// Best effort close after gzip write failure.
			_ = closeErr
		}
		return "", nil, err
	}
	if err := writer.Close(); err != nil {
		return "", nil, err
	}
	return "", buf.Bytes(), nil
}

func loadOutput(outputText sql.NullString, outputGzip []byte) (string, error) {
	if len(outputGzip) > 0 {
		reader, err := gzip.NewReader(bytes.NewReader(outputGzip))
		if err != nil {
			return "", err
		}
		defer reader.Close()
		decoded, err := io.ReadAll(reader)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}
	if outputText.Valid {
		return outputText.String, nil
	}
	return "", nil
}
