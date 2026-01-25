package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/ui/views"
)

// MainAreaMode represents the display mode of the main area
type MainAreaMode int

const (
	ModeDiff MainAreaMode = iota // Show diff viewer
	ModeLogs                     // Show operation logs
)

// MainArea is a wrapper component that switches between diff view and logs
type MainArea struct {
	styles     *styles.Styles
	width      int
	height     int
	focused    bool
	mode       MainAreaMode
	diffViewer *components.DiffViewer
	applyView  *views.ApplyView
	planView   *views.PlanView

	// Current state for diff mode
	selectedResource *terraform.ResourceChange
}

// NewMainArea creates a new main area component
func NewMainArea(s *styles.Styles, diffEngine *diff.Engine, applyView *views.ApplyView, planView *views.PlanView) *MainArea {
	return &MainArea{
		styles:     s,
		mode:       ModeDiff,
		diffViewer: components.NewDiffViewer(s, diffEngine),
		applyView:  applyView,
		planView:   planView,
	}
}

// SetSize updates the main area dimensions
func (m *MainArea) SetSize(width, height int) {
	m.width = width
	m.height = height

	// Calculate inner dimensions (accounting for border)
	innerWidth := width - 2
	innerHeight := height - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Update child components with inner dimensions
	if m.diffViewer != nil {
		m.diffViewer.SetSize(innerWidth, innerHeight)
	}
	if m.applyView != nil {
		m.applyView.SetSize(innerWidth, innerHeight)
	}
	if m.planView != nil {
		m.planView.SetSize(innerWidth, innerHeight)
	}
}

// SetFocused sets the focus state
func (m *MainArea) SetFocused(focused bool) {
	m.focused = focused
}

// IsFocused returns whether the panel is focused
func (m *MainArea) IsFocused() bool {
	return m.focused
}

// SetMode switches the display mode
func (m *MainArea) SetMode(mode MainAreaMode) {
	m.mode = mode
}

// GetMode returns the current display mode
func (m *MainArea) GetMode() MainAreaMode {
	return m.mode
}

// SetSelectedResource updates the selected resource for diff view
func (m *MainArea) SetSelectedResource(resource *terraform.ResourceChange) {
	m.selectedResource = resource
}

// Update handles Bubble Tea messages (implements Panel interface)
func (m *MainArea) Update(msg tea.Msg) (any, tea.Cmd) {
	var cmd tea.Cmd

	// Forward messages to appropriate child component
	switch m.mode {
	case ModeLogs:
		if m.applyView != nil {
			_, cmd = m.applyView.Update(msg)
		}
	case ModeDiff:
		// DiffViewer doesn't have Update method, it's stateless
	}

	return m, cmd
}

// HandleKey handles key events
func (m *MainArea) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	if !m.focused {
		return false, nil
	}

	// Forward key events to appropriate child based on mode
	switch m.mode {
	case ModeLogs:
		// Apply/Plan views handle scrolling
		if m.applyView != nil {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "home", "end":
				// These are typically handled by viewport inside applyView
				_, cmd := m.Update(msg)
				return true, cmd
			}
		}
	case ModeDiff:
		// Diff viewer is stateless, no key handling needed
	}

	return false, nil
}

// View renders the main area
func (m *MainArea) View() string {
	if m.styles == nil {
		return "[DEBUG: styles nil]"
	}
	if m.height <= 0 {
		return fmt.Sprintf("[DEBUG: height=%d width=%d]", m.height, m.width)
	}

	// Determine border style and title based on focus
	borderStyle := m.styles.Border
	titleStyle := m.styles.PanelTitle
	if m.focused {
		borderStyle = m.styles.FocusedBorder
		titleStyle = m.styles.FocusedPanelTitle
	}

	var content string
	var title string

	switch m.mode {
	case ModeLogs:
		// Show operation logs (plan or apply)
		title = "[0] Operation Logs"
		switch {
		case m.applyView != nil:
			content = m.applyView.View()
		case m.planView != nil:
			content = m.planView.View()
		default:
			content = m.styles.Dimmed.Render("No logs available")
		}

	case ModeDiff:
		// Show diff viewer
		title = "[0] Diff View"
		if m.diffViewer != nil {
			content = m.diffViewer.View(m.selectedResource)
		} else {
			content = m.styles.Dimmed.Render("No diff available")
		}
	}

	// Wrap in border
	panel := borderStyle.
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Width(m.width - 2).
		Height(m.height - 2).
		Render(content)

	// Add title to border
	if title != "" {
		titleRendered := titleStyle.Render(" " + title + " ")
		lines := strings.Split(panel, "\n")
		if len(lines) > 0 && m.width > 4 {
			// Use the same panel title rendering function as other panels
			if line, ok := components.RenderPanelTitleLine(m.width, borderStyle, titleRendered); ok {
				lines[0] = line
			}
		}
		panel = strings.Join(lines, "\n")
	}

	return panel
}

// GetApplyView returns the apply view (for external updates)
func (m *MainArea) GetApplyView() *views.ApplyView {
	return m.applyView
}

// GetPlanView returns the plan view (for external updates)
func (m *MainArea) GetPlanView() *views.PlanView {
	return m.planView
}

// GetDiffViewer returns the diff viewer (for external updates)
func (m *MainArea) GetDiffViewer() *components.DiffViewer {
	return m.diffViewer
}
