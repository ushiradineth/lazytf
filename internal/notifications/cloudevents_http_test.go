package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewCloudEventsHTTPNotifierValidatesURL(t *testing.T) {
	if _, err := NewCloudEventsHTTPNotifier("", 0, "", nil); err == nil {
		t.Fatalf("expected error for empty URL")
	}
	if _, err := NewCloudEventsHTTPNotifier("ftp://example.com/hook", 0, "", nil); err == nil {
		t.Fatalf("expected error for invalid URL scheme")
	}
	if _, err := NewCloudEventsHTTPNotifier("http:///hook", 0, "", nil); err == nil {
		t.Fatalf("expected error for missing host")
	}
}

func TestCloudEventsHTTPNotifierNotifySendsPayload(t *testing.T) {
	var (
		capturedBody        cloudEventsEnvelope
		capturedContentType string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	notifier, err := NewCloudEventsHTTPNotifier(server.URL, 2*time.Second, "https://example.com/source", nil)
	if err != nil {
		t.Fatalf("new notifier: %v", err)
	}
	notifier.nowFn = func() time.Time {
		return time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)
	}
	notifier.idFn = func() string { return "evt-1" }

	err = notifier.Notify(context.Background(), OperationEvent{
		Action:      "apply",
		Status:      StatusSuccess,
		Summary:     "+1 ~0 -0",
		Command:     "terraform apply -auto-approve",
		Environment: "default",
		WorkDir:     "/tmp/example",
		Duration:    1500 * time.Millisecond,
		ExitCode:    0,
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}

	if capturedContentType != "application/cloudevents+json" {
		t.Fatalf("unexpected content type: %s", capturedContentType)
	}
	if capturedBody.SpecVersion != "1.0" {
		t.Fatalf("unexpected specversion: %s", capturedBody.SpecVersion)
	}
	if capturedBody.ID != "evt-1" {
		t.Fatalf("unexpected id: %s", capturedBody.ID)
	}
	if capturedBody.Type != "io.lazytf.operation.apply.success" {
		t.Fatalf("unexpected type: %s", capturedBody.Type)
	}
	if capturedBody.Source != "https://example.com/source" {
		t.Fatalf("unexpected source: %s", capturedBody.Source)
	}
	if capturedBody.Data.Action != "apply" || capturedBody.Data.Status != "success" {
		t.Fatalf("unexpected data action/status: %+v", capturedBody.Data)
	}
	if capturedBody.Data.DurationMS != 1500 {
		t.Fatalf("expected duration_ms=1500, got %d", capturedBody.Data.DurationMS)
	}
}

func TestCloudEventsHTTPNotifierNotifyNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream failed"))
	}))
	defer server.Close()

	notifier, err := NewCloudEventsHTTPNotifier(server.URL, 2*time.Second, "", nil)
	if err != nil {
		t.Fatalf("new notifier: %v", err)
	}

	err = notifier.Notify(context.Background(), OperationEvent{Action: "plan", Status: StatusFailed})
	if err == nil {
		t.Fatalf("expected non-2xx error")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("expected status code in error, got %v", err)
	}
}
