package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// PanelID identifies each panel in the UI
type PanelID int

const (
	PanelMain       PanelID = 0 // Main area (diff or operation logs)
	PanelWorkspace  PanelID = 1 // Workspace/environment selector
	PanelResources  PanelID = 2 // Resource list
	PanelHistory    PanelID = 3 // History panel
	PanelCommandLog PanelID = 4 // Command log (diagnostics)
)

// Panel is the interface that all UI panels must implement
type Panel interface {
	// Update handles Bubble Tea messages and returns updated panel and command
	Update(msg tea.Msg) (any, tea.Cmd)

	// View renders the panel as a string
	View() string

	// SetSize updates the panel dimensions
	SetSize(width, height int)

	// SetFocused sets the focus state of the panel
	SetFocused(focused bool)

	// IsFocused returns whether the panel is currently focused
	IsFocused() bool

	// HandleKey handles key events and returns whether the key was handled
	HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd)
}

// PanelSpec defines the layout specification for a panel
type PanelSpec struct {
	X      int
	Y      int
	Width  int
	Height int
}

// LayoutSpec defines the complete layout for all panels
type LayoutSpec struct {
	FilterBarHeight  int
	StatusBarHeight  int
	LeftColumnWidth  int
	RightColumnWidth int

	Workspace  PanelSpec
	Resources  PanelSpec
	History    PanelSpec
	Main       PanelSpec
	CommandLog PanelSpec // Optional, only if visible

	CommandLogVisible bool
}

// Constants for layout calculations
const (
	FilterBarHeight    = 0
	StatusBarHeight    = 1
	WorkspaceHeight    = 6
	HistoryHeight      = 7
	CommandLogHeight   = 10
	CommandLogExpanded = 0.5 // 50% of screen height when expanded

	MinLeftColumnWidth = 40
	MaxLeftColumnWidth = 60
	LeftColumnRatio    = 0.35 // 35% of total width

	MinMainAreaWidth = 40
)
