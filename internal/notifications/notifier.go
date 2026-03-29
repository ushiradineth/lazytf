package notifications

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	// ProtocolCloudEventsHTTP sends CloudEvents JSON over HTTP(S).
	ProtocolCloudEventsHTTP = "cloudevents-http"
	defaultSource           = "https://github.com/ushiradineth/lazytf"
)

var eventTypeSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

// Config controls notifier construction.
type Config struct {
	Enabled  bool
	Protocol string
	URL      string
	Timeout  time.Duration
	Source   string
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
	protocol := strings.TrimSpace(strings.ToLower(cfg.Protocol))
	if protocol == "" {
		protocol = ProtocolCloudEventsHTTP
	}
	if protocol != ProtocolCloudEventsHTTP {
		return nil, fmt.Errorf("unsupported notification protocol: %s", cfg.Protocol)
	}
	return NewCloudEventsHTTPNotifier(cfg.URL, cfg.Timeout, cfg.Source, nil)
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

// EventType builds a stable CloudEvents type value.
func EventType(action string, status OperationStatus) string {
	actionPart := sanitizeSegment(action)
	if actionPart == "" {
		actionPart = "unknown"
	}
	statusPart := sanitizeSegment(string(status))
	if statusPart == "" {
		statusPart = "unknown"
	}
	return "io.lazytf.operation." + actionPart + "." + statusPart
}

func sanitizeSegment(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = eventTypeSanitizer.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	return normalized
}
