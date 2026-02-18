package ui

import (
	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

// PlanLoadedMsg is sent when a plan has been successfully loaded.
type PlanLoadedMsg struct {
	Plan  *terraform.Plan
	Error error
}

// FilterChangedMsg is sent when the action filter changes.
type FilterChangedMsg struct {
	Action  terraform.ActionType
	Enabled bool
}

// ToggleResourceMsg is sent to toggle a resource's expanded state.
type ToggleResourceMsg struct {
	Address string
}

// ErrorMsg represents an error that should be displayed to the user.
type ErrorMsg struct {
	Err error
}

// QuitMsg is sent to quit the application.
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
	Result *terraform.ExecutionResult
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

// RefreshStartMsg is sent when a refresh execution begins.
type RefreshStartMsg struct {
	Result *terraform.ExecutionResult
	Output <-chan string
	Error  error
}

// RefreshOutputMsg streams refresh output lines.
type RefreshOutputMsg struct{ Line string }

// RefreshCompleteMsg signals refresh completion.
type RefreshCompleteMsg struct {
	Success bool
	Error   error
	Result  *terraform.ExecutionResult
}

// ValidateStartMsg is sent when validation begins.
type ValidateStartMsg struct{}

// ValidateCompleteMsg signals validation completion.
type ValidateCompleteMsg struct {
	Result     *terraform.ValidateResult
	RawOutput  string
	Error      error
	ExecResult *terraform.ExecutionResult
}

// FormatStartMsg is sent when formatting begins.
type FormatStartMsg struct{}

// FormatCompleteMsg signals formatting completion.
type FormatCompleteMsg struct {
	ChangedFiles []string
	Error        error
	ExecResult   *terraform.ExecutionResult
}

// InitCompleteMsg signals init completion.
type InitCompleteMsg struct {
	Output string
	Error  error
	Result *terraform.ExecutionResult
}

// StateListStartMsg is sent when state list begins.
type StateListStartMsg struct{}

// StateListCompleteMsg signals state list completion.
type StateListCompleteMsg struct {
	Resources []terraform.StateResource
	Error     error
}

// StateShowStartMsg is sent when state show begins.
type StateShowStartMsg struct {
	Address string
}

// StateShowCompleteMsg signals state show completion.
type StateShowCompleteMsg struct {
	Address string
	Output  string
	Error   error
}

// StateRmCompleteMsg signals state rm completion.
type StateRmCompleteMsg struct {
	Address    string
	BackupPath string
	Output     string
	Error      error
	Result     *terraform.ExecutionResult
}

// StateMvCompleteMsg signals state mv completion.
type StateMvCompleteMsg struct {
	Source      string
	Destination string
	BackupPath  string
	Output      string
	Error       error
	Result      *terraform.ExecutionResult
}

// HistoryLoadedMsg delivers history entries.
type HistoryLoadedMsg struct {
	Entries []history.Entry
	Error   error
}

// HistoryDetailMsg delivers a single entry with output text and related operations.
type HistoryDetailMsg struct {
	Entry      history.Entry
	Operations []history.OperationEntry
	Error      error
}

// ClearToastMsg clears transient notifications.
type ClearToastMsg struct{}

// EnvironmentDetectedMsg delivers detected environment data.
type EnvironmentDetectedMsg struct {
	Result     environment.DetectionResult
	Current    string
	Preference *environment.Preference
	Error      error
}

// PanelFocusChangedMsg is sent when the focused panel changes.
type PanelFocusChangedMsg struct {
	PanelID PanelID
}

// ToggleCommandLogMsg toggles command log visibility.
type ToggleCommandLogMsg struct{}

// SetCommandLogVisibleMsg sets command log visibility.
type SetCommandLogVisibleMsg struct {
	Visible bool
}

// Action request messages - sent by panels to request operations.
// These allow panels to trigger actions without direct access to the executor.

// RequestPlanMsg requests a terraform plan execution.
type RequestPlanMsg struct{}

// RequestApplyMsg requests a terraform apply execution.
type RequestApplyMsg struct{}

// RequestRefreshMsg requests a terraform refresh execution.
type RequestRefreshMsg struct{}

// RequestValidateMsg requests a terraform validate execution.
type RequestValidateMsg struct{}

// RequestFormatMsg requests a terraform fmt execution.
type RequestFormatMsg struct{}

// ToggleFilterMsg toggles an action filter.
type ToggleFilterMsg struct {
	Action terraform.ActionType
}

// ToggleStatusMsg toggles the status column in the resource list.
type ToggleStatusMsg struct{}

// ToggleAllGroupsMsg toggles all groups expanded/collapsed.
type ToggleAllGroupsMsg struct{}

// SwitchResourcesTabMsg switches the Resources panel tab.
type SwitchResourcesTabMsg struct {
	Direction int // -1 for previous, +1 for next
}

// StateMoveCursorBlinkMsg toggles cursor visibility in state move input modal.
type StateMoveCursorBlinkMsg struct{}
