package notifications

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Config controls notifier construction.
type Config struct {
	Enabled bool
}

// OperationStatus is the normalized outcome of an operation.
type OperationStatus string

const (
	StatusSuccess  OperationStatus = "success"
	StatusFailed   OperationStatus = "failed"
	StatusCanceled OperationStatus = "canceled"
)

// OperationEvent represents one operation completion event.
type OperationEvent struct {
	Action      string
	Status      OperationStatus
	Summary     string
	Command     string
	Environment string
	WorkDir     string
	StartedAt   time.Time
	FinishedAt  time.Time
	Duration    time.Duration
	ExitCode    int
	Error       string
}

// Notifier sends operation events.
type Notifier interface {
	Notify(ctx context.Context, event OperationEvent) error
}

// NopNotifier intentionally drops events.
type NopNotifier struct{}

// Notify satisfies Notifier for NopNotifier.
func (NopNotifier) Notify(context.Context, OperationEvent) error {
	return nil
}

// New creates a notifier from config.
func New(cfg Config) (Notifier, error) {
	if !cfg.Enabled {
		return NopNotifier{}, nil
	}
	return NewDesktopNotifier()
}

func (e OperationEvent) validate() error {
	if strings.TrimSpace(e.Action) == "" {
		return errors.New("notification action cannot be empty")
	}
	switch e.Status {
	case StatusSuccess, StatusFailed, StatusCanceled:
		return nil
	default:
		return fmt.Errorf("invalid notification status: %s", e.Status)
	}
}
