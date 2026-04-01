package notifications

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDesktopNotifierNotifySendsFormattedNotification(t *testing.T) {
	t.Parallel()

	var (
		called  bool
		title   string
		message string
		icon    any
	)

	notifier := &DesktopNotifier{
		notifyFn: func(capturedTitle, capturedMessage string, capturedIcon any) error {
			called = true
			title = capturedTitle
			message = capturedMessage
			icon = capturedIcon
			return nil
		},
	}

	err := notifier.Notify(context.Background(), OperationEvent{
		Action:      "apply",
		Status:      StatusSuccess,
		Summary:     "+1 ~0 -0",
		Environment: "dev",
		WorkDir:     "/tmp/lazytf",
	})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}
	if !called {
		t.Fatalf("expected notifier backend to be called")
	}
	if title != "Terraform apply succeeded" {
		t.Fatalf("unexpected notification title: %q", title)
	}
	if !strings.Contains(message, "Changes: 1 to add") {
		t.Fatalf("expected humanized summary in message, got %q", message)
	}
	if !strings.Contains(message, "Environment: dev") {
		t.Fatalf("expected environment in message, got %q", message)
	}
	if strings.Contains(message, "Workdir:") {
		t.Fatalf("did not expect workdir in success message, got %q", message)
	}
	iconValue, ok := icon.(string)
	if !ok {
		t.Fatalf("expected icon payload to be string, got %T", icon)
	}
	if iconValue != "" {
		t.Fatalf("expected empty icon payload, got %q", iconValue)
	}
}

func TestDesktopNotifierNotifyPropagatesBackendError(t *testing.T) {
	t.Parallel()

	backendErr := errors.New("desktop notifications unavailable")
	notifier := &DesktopNotifier{
		notifyFn: func(string, string, any) error {
			return backendErr
		},
	}

	err := notifier.Notify(context.Background(), OperationEvent{Action: "plan", Status: StatusFailed})
	if err == nil {
		t.Fatalf("expected notify error")
	}
	if !errors.Is(err, backendErr) {
		t.Fatalf("expected wrapped backend error, got %v", err)
	}
}

func TestDesktopNotifierNotifyHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	called := false
	notifier := &DesktopNotifier{
		notifyFn: func(string, string, any) error {
			called = true
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := notifier.Notify(ctx, OperationEvent{Action: "refresh", Status: StatusCanceled})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
	if called {
		t.Fatalf("expected backend not to be called after context cancellation")
	}
}

func TestFormatDesktopNotificationFallbackMessage(t *testing.T) {
	t.Parallel()

	title, message := formatDesktopNotification(OperationEvent{Action: "plan", Status: StatusSuccess})
	if title != "Terraform plan succeeded" {
		t.Fatalf("unexpected title: %q", title)
	}
	if message != "Completed successfully" {
		t.Fatalf("unexpected fallback message: %q", message)
	}
}

func TestFormatDesktopNotificationHumanizesPlanSummary(t *testing.T) {
	t.Parallel()

	_, message := formatDesktopNotification(OperationEvent{Action: "plan", Status: StatusSuccess, Summary: "+9 ~5 -4"})
	if !strings.Contains(message, "Planned changes: 9 to add, 5 to change, 4 to destroy") {
		t.Fatalf("expected humanized plan summary, got %q", message)
	}
}

func TestFormatDesktopNotificationSuppressesPathLikeEnvironment(t *testing.T) {
	t.Parallel()

	_, message := formatDesktopNotification(OperationEvent{
		Action:      "plan",
		Status:      StatusSuccess,
		Summary:     "+1 ~0 -0",
		Environment: "/Users/shu/Code/tmpkube",
		WorkDir:     "/Users/shu/Code/tmpkube",
	})
	if strings.Contains(message, "Environment:") {
		t.Fatalf("expected path-like environment to be suppressed, got %q", message)
	}
	if strings.Contains(message, "Workdir:") {
		t.Fatalf("expected workdir to be omitted for success, got %q", message)
	}
}

func TestFormatDesktopNotificationIncludesWorkdirForFailures(t *testing.T) {
	t.Parallel()

	_, message := formatDesktopNotification(OperationEvent{
		Action:  "apply",
		Status:  StatusFailed,
		Error:   "provider error",
		WorkDir: "/Users/shu/Code/infra/production",
	})
	if !strings.Contains(message, "provider error") {
		t.Fatalf("expected error detail, got %q", message)
	}
	if !strings.Contains(message, "Workdir:") {
		t.Fatalf("expected workdir context for failure, got %q", message)
	}
}
