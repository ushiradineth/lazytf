package notifications

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gen2brain/beeep"
)

type desktopNotifyFunc func(title, message string, icon any) error

// DesktopNotifier sends operation events as local desktop notifications.
type DesktopNotifier struct {
	notifyFn desktopNotifyFunc
}

var changeSummaryPattern = regexp.MustCompile(`^\s*\+(\d+)\s+~(\d+)\s+-(\d+)\s*$`)

// NewDesktopNotifier creates a library-backed desktop notifier.
func NewDesktopNotifier() (*DesktopNotifier, error) {
	return &DesktopNotifier{notifyFn: beeep.Notify}, nil
}

// Notify sends an operation event as a desktop notification.
func (n *DesktopNotifier) Notify(ctx context.Context, event OperationEvent) error {
	if err := event.validate(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("desktop notifier is nil")
	}
	if n.notifyFn == nil {
		return errors.New("desktop notifier notify function is nil")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	title, message := formatDesktopNotification(event)
	if err := n.notifyFn(title, message, ""); err != nil {
		return fmt.Errorf("send desktop notification: %w", err)
	}

	return nil
}

func formatDesktopNotification(event OperationEvent) (string, string) {
	action := strings.TrimSpace(event.Action)
	if action == "" {
		action = "operation"
	}
	title := fmt.Sprintf("Terraform %s %s", action, statusWord(event.Status))

	lines := buildNotificationLines(event)
	return title, strings.Join(lines, "\n")
}

func buildNotificationLines(event OperationEvent) []string {
	lines := make([]string, 0, 3)

	if primary := primaryDetailLine(event); primary != "" {
		lines = append(lines, primary)
	}
	if environmentLine := environmentContextLine(event); environmentLine != "" {
		lines = append(lines, environmentLine)
	}
	if workDirLine := workDirContextLine(event); workDirLine != "" {
		lines = append(lines, workDirLine)
	}

	if len(lines) == 0 {
		return []string{"Completed successfully"}
	}
	if len(lines) > 3 {
		return lines[:3]
	}

	return lines
}

func primaryDetailLine(event OperationEvent) string {
	switch event.Status {
	case StatusSuccess:
		if summary := humanizeSummary(event.Action, event.Summary); summary != "" {
			return summary
		}
		return "Completed successfully"
	case StatusFailed:
		if errText := strings.TrimSpace(event.Error); errText != "" {
			return errText
		}
		if summary := humanizeSummary(event.Action, event.Summary); summary != "" {
			return summary
		}
		return "Operation failed"
	case StatusCanceled:
		return "Operation canceled"
	default:
		return "Operation completed"
	}
}

func humanizeSummary(action, summary string) string {
	trimmed := strings.TrimSpace(summary)
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "no changes") {
		if strings.EqualFold(strings.TrimSpace(action), "plan") {
			return "Planned changes: no changes"
		}
		return "No changes"
	}

	matches := changeSummaryPattern.FindStringSubmatch(trimmed)
	if len(matches) != 4 {
		return trimmed
	}

	adds, errAdd := strconv.Atoi(matches[1])
	changes, errChange := strconv.Atoi(matches[2])
	destroys, errDestroy := strconv.Atoi(matches[3])
	if errAdd != nil || errChange != nil || errDestroy != nil {
		return trimmed
	}

	parts := make([]string, 0, 3)
	if adds > 0 {
		parts = append(parts, fmt.Sprintf("%d to add", adds))
	}
	if changes > 0 {
		parts = append(parts, fmt.Sprintf("%d to change", changes))
	}
	if destroys > 0 {
		parts = append(parts, fmt.Sprintf("%d to destroy", destroys))
	}
	if len(parts) == 0 {
		if strings.EqualFold(strings.TrimSpace(action), "plan") {
			return "Planned changes: no changes"
		}
		return "No changes"
	}

	if strings.EqualFold(strings.TrimSpace(action), "plan") {
		return "Planned changes: " + strings.Join(parts, ", ")
	}
	return "Changes: " + strings.Join(parts, ", ")
}

func environmentContextLine(event OperationEvent) string {
	environment := strings.TrimSpace(event.Environment)
	if environment == "" {
		return ""
	}
	if strings.EqualFold(environment, "default") {
		return ""
	}
	if looksLikePath(environment) {
		return ""
	}
	if workDir := strings.TrimSpace(event.WorkDir); workDir != "" {
		if strings.EqualFold(environment, workDir) {
			return ""
		}
		if strings.EqualFold(environment, filepath.Base(workDir)) {
			return ""
		}
	}
	return "Environment: " + environment
}

func workDirContextLine(event OperationEvent) string {
	if event.Status != StatusFailed {
		return ""
	}
	workDir := strings.TrimSpace(event.WorkDir)
	if workDir == "" {
		return ""
	}
	return "Workdir: " + shortenPath(workDir)
}

func statusWord(status OperationStatus) string {
	switch status {
	case StatusSuccess:
		return "succeeded"
	case StatusFailed:
		return "failed"
	case StatusCanceled:
		return "canceled"
	default:
		return "completed"
	}
}

func looksLikePath(value string) bool {
	return strings.Contains(value, "/") || strings.Contains(value, `\`) || strings.HasPrefix(value, "~") || strings.HasPrefix(value, ".")
}

func shortenPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	shortened := trimmed
	if home, err := os.UserHomeDir(); err == nil {
		home = filepath.Clean(home)
		if home != "." && strings.HasPrefix(filepath.Clean(trimmed), home) {
			rel := strings.TrimPrefix(filepath.Clean(trimmed), home)
			if rel == "" {
				shortened = "~"
			} else {
				shortened = "~" + rel
			}
		}
	}
	if len(shortened) <= 48 {
		return shortened
	}
	return "..." + shortened[len(shortened)-45:]
}
