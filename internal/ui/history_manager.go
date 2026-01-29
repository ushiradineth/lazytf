package ui

import (
	"context"
	"errors"
	"os"
	"os/user"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
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
	if m.envCurrent == "" {
		return m.historyStore.ListRecent(5)
	}
	return m.historyStore.ListRecentForEnvironment(m.envCurrent, 5)
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
		return HistoryDetailMsg{Entry: entry}
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
	case "down", "j":
		if m.historySelected < len(m.historyEntries)-1 {
			m.historySelected++
		}
	case "enter":
		if len(m.historyEntries) == 0 || m.historyStore == nil {
			return true, nil
		}
		entry := m.historyEntries[m.historySelected]
		// Show history detail in main area instead of full-screen view
		if m.mainArea != nil {
			m.mainArea.EnterHistoryDetail()
			m.mainArea.SetHistoryContent("Apply details", "Loading...")
		}
		return true, m.loadHistoryDetailCmd(entry.ID)
	default:
		return false, nil
	}
	m.syncHistorySelection()
	return true, nil
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
