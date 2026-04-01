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

func TestNewUsesDesktopNotifierWhenEnabled(t *testing.T) {
	notifier, err := New(Config{Enabled: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := notifier.(*DesktopNotifier); !ok {
		t.Fatalf("expected DesktopNotifier, got %T", notifier)
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
