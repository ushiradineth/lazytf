package ui

import "time"

// Layout constants for the UI.
const (
	// MinSplitWidth is the minimum terminal width to enable split view.
	MinSplitWidth = 100

	// MinListWidth is the minimum width for the resource list panel.
	MinListWidth = 40

	// MinDiffWidth is the minimum width for the diff viewer panel.
	MinDiffWidth = 20

	// ListWidthRatio is the proportion of width allocated to list in split view.
	ListWidthRatio = 0.45

	// MaxFocusedDiagnosticsHeight is the maximum height for diagnostics panel when focused.
	MaxFocusedDiagnosticsHeight = 12

	// MinPanelHeight is the minimum height for any panel.
	MinPanelHeight = 3

	// DefaultHistoryHeight is the default height for the history panel.
	DefaultHistoryHeight = 6
)

// Toast duration constants.
const (
	ToastShortDuration  = 2 * time.Second
	ToastNormalDuration = 3 * time.Second
	ToastLongDuration   = 5 * time.Second
)

// Command log constants.
const (
	MaxCommandLogs = 100
)
