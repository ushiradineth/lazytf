package ui

import (
	"context"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/notifications"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

// History-related methods for Model

func (m *Model) recordHistoryCmd(status history.Status, summary string, planOutput string, result *terraform.ExecutionResult, err error) tea.Cmd {
	if m.historyStore == nil {
		return nil
	}
	entry := history.Entry{
		StartedAt:   m.applyStartedAt,
		FinishedAt:  time.Now(),
		Duration:    time.Since(m.applyStartedAt),
		Status:      status,
		Summary:     summary,
		Environment: m.envCurrent,
	}
	if m.executor != nil {
		entry.WorkDir = m.executor.WorkDir()
	}
	if err != nil {
		entry.Error = err.Error()
	}
	if planOutput != "" {
		entry.Output = truncateOutput(planOutput, 2*1024*1024)
	} else if result != nil {
		entry.Output = truncateOutput(result.Output, 2*1024*1024)
	}

	return func() tea.Msg {
		if recordErr := m.historyStore.RecordApply(entry); recordErr != nil {
			return HistoryLoadedMsg{Error: recordErr}
		}
		entries, listErr := m.loadHistoryEntries()
		if listErr != nil {
			return HistoryLoadedMsg{Error: listErr}
		}
		return HistoryLoadedMsg{Entries: entries}
	}
}

func (m *Model) recordOperationCmd(action string, flags []string, autoApprove bool, startedAt time.Time, result *terraform.ExecutionResult, output string, opErr error) tea.Cmd {
	if m.historyLogger == nil {
		return nil
	}
	entry := history.OperationEntry{
		StartedAt:   startedAt,
		Action:      action,
		Command:     m.buildCommand(action, flags, autoApprove),
		Summary:     m.flattenSummary(m.planSummary()),
		User:        currentUserName(),
		WorkDir:     m.currentHistoryWorkDir(),
		Environment: m.envCurrent,
		Output:      selectOperationOutput(output, result),
	}
	if result != nil {
		entry.ExitCode = result.ExitCode
		entry.Duration = result.Duration
	}
	entry.Status = operationStatus(opErr)
	return func() tea.Msg {
		if err := m.historyLogger.RecordOperation(entry); err != nil {
			return HistoryLoadedMsg{Error: err}
		}
		return nil
	}
}

func (m *Model) loadHistoryEntries() ([]history.Entry, error) {
	if m.historyStore == nil {
		return nil, nil
	}
	// Large limit - the panel is scrollable so no practical need to restrict
	const maxEntries = 1000
	return m.historyStore.ListRecentForScope(m.envCurrent, m.currentHistoryWorkDir(), maxEntries)
}

func (m *Model) currentHistoryWorkDir() string {
	workDir := m.envWorkDir
	if m.executor != nil {
		workDir = m.executor.WorkDir()
	}
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return ""
	}
	abs, err := filepath.Abs(workDir)
	if err != nil {
		return workDir
	}
	return abs
}

func (m *Model) loadHistoryDetailCmd(id int64) tea.Cmd {
	if m.historyStore == nil {
		return nil
	}
	return func() tea.Msg {
		entry, err := m.historyStore.GetByID(id)
		if err != nil {
			return HistoryDetailMsg{Error: err}
		}

		// Fetch related operations (plan + apply) within time window
		var operations []history.OperationEntry
		if ops, opsErr := m.historyStore.GetOperationsForApply(entry); opsErr == nil {
			operations = ops
		}

		return HistoryDetailMsg{
			Entry:      entry,
			Operations: operations,
		}
	}
}

func (m *Model) syncHistorySelection() {
	if m.historyPanel == nil {
		return
	}
	if len(m.historyEntries) == 0 {
		m.historySelected = 0
		m.historyPanel.SetSelection(0, m.historyFocused)
		return
	}
	if m.historySelected >= len(m.historyEntries) {
		m.historySelected = len(m.historyEntries) - 1
	}
	if m.historySelected < 0 {
		m.historySelected = 0
	}
	m.historyPanel.SetSelection(m.historySelected, m.historyFocused)
}

func (m *Model) handleHistoryKeys(key string) (bool, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.historySelected > 0 {
			m.historySelected--
		}
	case keybinds.KeyDown, "j":
		if m.historySelected < len(m.historyEntries)-1 {
			m.historySelected++
		}
	case "enter":
		// Enter loads the detail and focuses the main area for scrolling
		m.syncHistorySelection()
		loadCmd := m.showSelectedHistoryDetail()
		focusCmd := m.focusMainPanel()
		return true, tea.Batch(loadCmd, focusCmd)
	default:
		return false, nil
	}
	m.syncHistorySelection()
	// Show history detail in main area on scroll
	return true, m.showSelectedHistoryDetail()
}

// focusMainPanel switches focus to the main panel.
func (m *Model) focusMainPanel() tea.Cmd {
	if m.panelManager == nil {
		return nil
	}
	return m.panelManager.SetFocus(PanelMain)
}

// showSelectedHistoryDetail loads and shows the currently selected history item in the main area.
func (m *Model) showSelectedHistoryDetail() tea.Cmd {
	if len(m.historyEntries) == 0 || m.historyStore == nil {
		return nil
	}
	entry := m.historyEntries[m.historySelected]
	// Show history detail in main area
	if m.mainArea != nil {
		m.mainArea.EnterHistoryDetail()
		m.mainArea.SetHistoryContent("Apply details", "Loading...")
	}
	return m.loadHistoryDetailCmd(entry.ID)
}

// Helper functions for history operations

func operationStatus(err error) history.Status {
	if err == nil {
		return history.StatusSuccess
	}
	if errors.Is(err, context.Canceled) {
		return history.StatusCanceled
	}
	return history.StatusFailed
}

func selectOperationOutput(output string, result *terraform.ExecutionResult) string {
	if strings.TrimSpace(output) != "" {
		return output
	}
	if result == nil {
		return ""
	}
	if result.Output != "" {
		return result.Output
	}
	if result.Stdout != "" {
		return result.Stdout
	}
	return result.Stderr
}

var currentUserFunc = user.Current

func currentUserName() string {
	if current, err := currentUserFunc(); err == nil && current != nil && current.Username != "" {
		return current.Username
	}
	if value := os.Getenv("USER"); value != "" {
		return value
	}
	return os.Getenv("USERNAME")
}

func truncateOutput(output string, maxBytes int) string {
	if maxBytes <= 0 || len(output) <= maxBytes {
		return output
	}
	return output[:maxBytes]
}

func (m *Model) notifyOperationCmd(action, summary string, startedAt time.Time, result *terraform.ExecutionResult, opErr error) tea.Cmd {
	if m.notifier == nil {
		return nil
	}
	finishedAt := time.Now()
	duration := computeOperationDuration(startedAt, finishedAt, result)
	event := notifications.OperationEvent{
		Action:      action,
		Status:      notificationStatus(opErr),
		Summary:     summary,
		Environment: m.envCurrent,
		WorkDir:     m.currentHistoryWorkDir(),
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		Duration:    duration,
	}
	if result != nil {
		event.ExitCode = result.ExitCode
	}
	if opErr != nil {
		event.Error = opErr.Error()
	}
	notifier := m.notifier
	return func() tea.Msg {
		if err := notifier.Notify(context.Background(), event); err != nil {
			return NotificationFailedMsg{Action: action, Error: err}
		}
		return nil
	}
}

func notificationStatus(err error) notifications.OperationStatus {
	if err == nil {
		return notifications.StatusSuccess
	}
	if errors.Is(err, context.Canceled) {
		return notifications.StatusCanceled
	}
	return notifications.StatusFailed
}

func computeOperationDuration(startedAt, finishedAt time.Time, result *terraform.ExecutionResult) time.Duration {
	if result != nil && result.Duration > 0 {
		return result.Duration
	}
	if startedAt.IsZero() {
		return 0
	}
	if finishedAt.Before(startedAt) {
		return 0
	}
	return finishedAt.Sub(startedAt)
}
