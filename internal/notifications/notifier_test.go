package notifications

import (
	"testing"
)

func TestNewReturnsNopWhenDisabled(t *testing.T) {
	notifier, err := New(Config{Enabled: false})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := notifier.(NopNotifier); !ok {
		t.Fatalf("expected NopNotifier, got %T", notifier)
	}
}

func TestNewReturnsErrorOnUnsupportedProtocol(t *testing.T) {
	_, err := New(Config{Enabled: true, Protocol: "smtp", URL: "https://example.com/hook"})
	if err == nil {
		t.Fatalf("expected error for unsupported protocol")
	}
}

func TestNewUsesCloudEventsHTTPByDefault(t *testing.T) {
	notifier, err := New(Config{Enabled: true, URL: "https://example.com/hook"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := notifier.(*CloudEventsHTTPNotifier); !ok {
		t.Fatalf("expected CloudEventsHTTPNotifier, got %T", notifier)
	}
}

func TestEventTypeSanitizesSegments(t *testing.T) {
	typeValue := EventType("state rm", StatusSuccess)
	if typeValue != "io.lazytf.operation.state-rm.success" {
		t.Fatalf("unexpected event type: %s", typeValue)
	}
}

func TestOperationEventValidate(t *testing.T) {
	invalidAction := OperationEvent{Status: StatusSuccess}
	if err := invalidAction.validate(); err == nil {
		t.Fatalf("expected validation error for empty action")
	}

	invalidStatus := OperationEvent{Action: "plan", Status: "wat"}
	if err := invalidStatus.validate(); err == nil {
		t.Fatalf("expected validation error for invalid status")
	}

	valid := OperationEvent{Action: "plan", Status: StatusSuccess}
	if err := valid.validate(); err != nil {
		t.Fatalf("expected valid event, got %v", err)
	}
}
