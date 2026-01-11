package history

import "time"

// This file implements the Logger type which controls history verbosity
// and output size limits for terraform operations.

// Level controls how much history data is stored.
type Level string

const (
	LevelMinimal  Level = "minimal"
	LevelStandard Level = "standard"
	LevelVerbose  Level = "verbose"
)

const (
	maxOutputStandard = 256 * 1024
	maxOutputVerbose  = 2 * 1024 * 1024
)

// Logger stores terraform operations with a configured verbosity.
type Logger struct {
	store *Store
	level Level
}

// NewLogger creates a history Logger with the provided store and level.
func NewLogger(store *Store, level Level) *Logger {
	if level == "" {
		level = LevelStandard
	}
	return &Logger{store: store, level: level}
}

// RecordOperation stores an operation entry based on the logger level.
func (l *Logger) RecordOperation(entry OperationEntry) error {
	if l == nil {
		// Silently skip if logger is nil (history disabled)
		return nil
	}
	if l.store == nil {
		// Silently skip if store is nil (history disabled)
		return nil
	}
	entry.Output = l.prepareOutput(entry.Output)
	if entry.FinishedAt.IsZero() {
		entry.FinishedAt = time.Now()
	}
	if entry.Duration == 0 && !entry.StartedAt.IsZero() {
		entry.Duration = entry.FinishedAt.Sub(entry.StartedAt)
	}
	return l.store.RecordOperation(entry)
}

func (l *Logger) prepareOutput(output string) string {
	switch l.level {
	case LevelMinimal:
		return ""
	case LevelVerbose:
		return truncateOutput(output, maxOutputVerbose)
	default:
		return truncateOutput(output, maxOutputStandard)
	}
}

func truncateOutput(output string, maxBytes int) string {
	if maxBytes <= 0 || len(output) <= maxBytes {
		return output
	}
	return output[:maxBytes]
}
