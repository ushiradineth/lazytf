package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout     = 3 * time.Second
	cloudEventsVersion = "1.0"
)

type cloudEventsEnvelope struct {
	SpecVersion     string              `json:"specversion"`
	Type            string              `json:"type"`
	Source          string              `json:"source"`
	ID              string              `json:"id"`
	Time            string              `json:"time"`
	Subject         string              `json:"subject,omitempty"`
	DataContentType string              `json:"datacontenttype"`
	Data            cloudEventsDataBody `json:"data"`
}

type cloudEventsDataBody struct {
	Action      string `json:"action"`
	Status      string `json:"status"`
	Summary     string `json:"summary,omitempty"`
	Command     string `json:"command,omitempty"`
	Environment string `json:"environment,omitempty"`
	WorkDir     string `json:"workdir,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
	DurationMS  int64  `json:"duration_ms,omitempty"`
	ExitCode    int    `json:"exit_code,omitempty"`
	Error       string `json:"error,omitempty"`
}

// CloudEventsHTTPNotifier sends CloudEvents over HTTP(S) webhook.
type CloudEventsHTTPNotifier struct {
	endpoint string
	source   string
	client   *http.Client
	nowFn    func() time.Time
	idFn     func() string
}

// NewCloudEventsHTTPNotifier creates a CloudEvents notifier.
func NewCloudEventsHTTPNotifier(endpoint string, timeout time.Duration, source string, client *http.Client) (*CloudEventsHTTPNotifier, error) {
	trimmedEndpoint := strings.TrimSpace(endpoint)
	if trimmedEndpoint == "" {
		return nil, errors.New("notification sink url is required")
	}
	parsed, err := url.Parse(trimmedEndpoint)
	if err != nil {
		return nil, fmt.Errorf("parse notification sink url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("notification sink url must use http or https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, errors.New("notification sink url must include host")
	}

	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	n := &CloudEventsHTTPNotifier{
		endpoint: parsed.String(),
		source:   strings.TrimSpace(source),
		client:   client,
		nowFn:    time.Now,
	}
	if n.source == "" {
		n.source = defaultSource
	}
	n.idFn = func() string {
		return strconv.FormatInt(n.nowFn().UnixNano(), 10)
	}
	return n, nil
}

// Notify sends the event in CloudEvents 1.0 JSON format.
func (n *CloudEventsHTTPNotifier) Notify(ctx context.Context, event OperationEvent) error {
	if err := event.validate(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("cloud events notifier is nil")
	}

	finishedAt := event.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = n.nowFn().UTC()
	}

	body := cloudEventsEnvelope{
		SpecVersion:     cloudEventsVersion,
		Type:            EventType(event.Action, event.Status),
		Source:          n.source,
		ID:              n.idFn(),
		Time:            finishedAt.Format(time.RFC3339Nano),
		Subject:         strings.TrimSpace(event.Environment),
		DataContentType: "application/json",
		Data: cloudEventsDataBody{
			Action:      event.Action,
			Status:      string(event.Status),
			Summary:     event.Summary,
			Command:     event.Command,
			Environment: event.Environment,
			WorkDir:     event.WorkDir,
			DurationMS:  event.Duration.Milliseconds(),
			ExitCode:    event.ExitCode,
			Error:       event.Error,
		},
	}
	if !event.StartedAt.IsZero() {
		body.Data.StartedAt = event.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if !event.FinishedAt.IsZero() {
		body.Data.FinishedAt = event.FinishedAt.UTC().Format(time.RFC3339Nano)
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal cloud event payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create notification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/cloudevents+json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send notification request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(resp.Body)
		trimmedBody := strings.TrimSpace(string(responseBody))
		if trimmedBody == "" {
			return fmt.Errorf("notification request returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("notification request returned status %d: %s", resp.StatusCode, trimmedBody)
	}

	return nil
}
