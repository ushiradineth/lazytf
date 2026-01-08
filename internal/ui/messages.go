package ui

import (
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
