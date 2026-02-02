package history

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoggerRespectsLevels(t *testing.T) {
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

	output := strings.Repeat("log line\n", 50000)
	entry := OperationEntry{
		StartedAt:  time.Now().Add(-time.Second),
		FinishedAt: time.Now(),
		Action:     "apply",
		Command:    "terraform apply",
		ExitCode:   1,
		Status:     StatusFailed,
		Summary:    "apply failed",
		User:       "bob",
		Output:     output,
	}

	logger := NewLogger(store, LevelMinimal)
	if err := logger.RecordOperation(entry); err != nil {
		t.Fatalf("record minimal: %v", err)
	}

	entries, err := store.QueryOperations(OperationFilter{Action: "apply", Limit: 5})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Output != "" {
		t.Fatalf("expected minimal level to drop output")
	}

	verboseLogger := NewLogger(store, LevelVerbose)
	entry.User = "carol"
	if err := verboseLogger.RecordOperation(entry); err != nil {
		t.Fatalf("record verbose: %v", err)
	}

	entries, err = store.QueryOperations(OperationFilter{User: "carol", Limit: 5})
	if err != nil {
		t.Fatalf("query verbose entry: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Output == "" {
		t.Fatalf("expected verbose level to keep output")
	}
}

func TestLoggerDefaultLevelAndNilHandling(t *testing.T) {
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

	logger := NewLogger(store, "")
	if logger.level != LevelStandard {
		t.Fatalf("expected default level standard")
	}

	entry := OperationEntry{Action: "plan"}
	if err := logger.RecordOperation(entry); err != nil {
		t.Fatalf("record operation: %v", err)
	}

	var nilLogger *Logger
	if err := nilLogger.RecordOperation(entry); err != nil {
		t.Fatalf("expected nil logger to be ignored")
	}

	disabled := &Logger{}
	if err := disabled.RecordOperation(entry); err != nil {
		t.Fatalf("expected nil store to be ignored")
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		maxBytes int
		expected string
	}{
		{"empty output", "", 100, ""},
		{"short output", "hello", 100, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hello"},
		{"zero max", "hello", 0, "hello"},
		{"negative max", "hello", -1, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateOutput(tt.output, tt.maxBytes)
			if result != tt.expected {
				t.Errorf("truncateOutput(%q, %d) = %q, want %q", tt.output, tt.maxBytes, result, tt.expected)
			}
		})
	}
}

func TestPrepareOutputLevels(t *testing.T) {
	// Large output to test truncation
	output := strings.Repeat("x", 3*1024*1024) // 3MB

	// Test with minimal level
	minimalLogger := &Logger{level: LevelMinimal}
	result := minimalLogger.prepareOutput(output)
	if result != "" {
		t.Error("minimal output should be empty")
	}

	// Test with verbose level
	verboseLogger := &Logger{level: LevelVerbose}
	result = verboseLogger.prepareOutput(output)
	if len(result) > maxOutputVerbose {
		t.Error("verbose output should be truncated")
	}

	// Test with standard level
	standardLogger := &Logger{level: LevelStandard}
	result = standardLogger.prepareOutput(output)
	if len(result) > maxOutputStandard {
		t.Error("standard output should be truncated")
	}
}
