package ui

import (
	"github.com/ushiradineth/tftui/internal/history"
	"github.com/ushiradineth/tftui/internal/terraform"
)

// PlanLoadedMsg is sent when a plan has been successfully loaded
type PlanLoadedMsg struct {
	Plan  *terraform.Plan
	Error error
}

// FilterChangedMsg is sent when the action filter changes
type FilterChangedMsg struct {
	Action  terraform.ActionType
	Enabled bool
}

// ToggleResourceMsg is sent to toggle a resource's expanded state
type ToggleResourceMsg struct {
	Address string
}

// ErrorMsg represents an error that should be displayed to the user
type ErrorMsg struct {
	Err error
}

// QuitMsg is sent to quit the application
type QuitMsg struct{}

// PlanStartMsg is sent when a plan execution begins.
type PlanStartMsg struct {
	Result *terraform.ExecutionResult
	Output <-chan string
	Error  error
}

// PlanOutputMsg streams plan output lines.
type PlanOutputMsg struct{ Line string }

// PlanCompleteMsg signals plan completion.
type PlanCompleteMsg struct {
	Plan   *terraform.Plan
	Error  error
	Output string
}

// ApplyStartMsg is sent when an apply execution begins.
type ApplyStartMsg struct {
	Result *terraform.ExecutionResult
	Output <-chan string
	Error  error
}

// ApplyOutputMsg streams apply output lines.
type ApplyOutputMsg struct{ Line string }

// ApplyCompleteMsg signals apply completion.
type ApplyCompleteMsg struct {
	Success bool
	Error   error
	Result  *terraform.ExecutionResult
}

// HistoryLoadedMsg delivers history entries.
type HistoryLoadedMsg struct {
	Entries []history.Entry
	Error   error
}

// HistoryDetailMsg delivers a single entry with output text.
type HistoryDetailMsg struct {
	Entry history.Entry
	Error error
}

// ClearToastMsg clears transient notifications.
type ClearToastMsg struct{}
