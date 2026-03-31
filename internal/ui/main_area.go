package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/ui/views"
)

// MainAreaMode represents the display mode of the main area.
type MainAreaMode int

const (
	ModeDiff          MainAreaMode = iota // Show diff viewer
	ModeLogs                              // Show operation logs
	ModeHistoryDetail                     // Show history detail (formatted logs)
	ModeStateShow                         // Show state resource details
	ModeAbout                             // Show about/info screen
)

// MainArea is a wrapper component that switches between diff view and logs.
type MainArea struct {
	styles       *styles.Styles
	frame        *components.PanelFrame
	height       int
	focused      bool
	mode         MainAreaMode
	previousMode MainAreaMode // Store previous mode for returning from history detail
	diffViewer   *components.DiffViewer
	applyView    *views.ApplyView
	planView     *views.PlanView
	historyView  *views.HistoryView
	stateView    *views.StateShowView
	aboutView    *views.AboutView

	// Current state for diff mode
	selectedResource *terraform.ResourceChange
}

// NewMainArea creates a new main area component.
func NewMainArea(s *styles.Styles, diffEngine *diff.Engine, applyView *views.ApplyView, planView *views.PlanView) *MainArea {
	return &MainArea{
		styles:      s,
		frame:       components.NewPanelFrame(s),
		mode:        ModeDiff,
		diffViewer:  components.NewDiffViewer(s, diffEngine),
		applyView:   applyView,
		planView:    planView,
		historyView: views.NewHistoryView(s),
		stateView:   views.NewStateShowView(s),
		aboutView:   views.NewAboutView(s),
	}
}

// SetSize updates the main area dimensions.
func (m *MainArea) SetSize(width, height int) {
	m.height = height
	m.frame.SetSize(width, height)

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
	if m.historyView != nil {
		m.historyView.SetSize(innerWidth, innerHeight)
	}
	if m.stateView != nil {
		m.stateView.SetSize(innerWidth, innerHeight)
	}
	if m.aboutView != nil {
		m.aboutView.SetSize(innerWidth, innerHeight)
	}
}

// SetFocused sets the focus state.
func (m *MainArea) SetFocused(focused bool) {
	m.focused = focused
}

// IsFocused returns whether the panel is focused.
func (m *MainArea) IsFocused() bool {
	return m.focused
}

// SetStyles updates the component styles.
func (m *MainArea) SetStyles(s *styles.Styles) {
	m.styles = s
	if m.frame != nil {
		m.frame.SetStyles(s)
	}
	if m.diffViewer != nil {
		m.diffViewer.SetStyles(s)
	}
	if m.historyView != nil {
		m.historyView.SetStyles(s)
	}
	if m.stateView != nil {
		m.stateView.SetStyles(s)
	}
	if m.aboutView != nil {
		m.aboutView.SetStyles(s)
	}
}

// SetMode switches the display mode.
func (m *MainArea) SetMode(mode MainAreaMode) {
	m.mode = mode
}

// GetMode returns the current display mode.
func (m *MainArea) GetMode() MainAreaMode {
	return m.mode
}

// EnterHistoryDetail switches to history detail mode, saving the current mode.
func (m *MainArea) EnterHistoryDetail() {
	m.previousMode = m.mode
	m.mode = ModeHistoryDetail
}

// ExitHistoryDetail returns to the previous mode.
func (m *MainArea) ExitHistoryDetail() {
	m.mode = m.previousMode
}

// SetHistoryContent sets the history detail content.
func (m *MainArea) SetHistoryContent(title, content string) {
	if m.historyView != nil {
		m.historyView.SetTitle(title)
		m.historyView.SetContent(content)
	}
}

// SetStateContent sets the state show content and switches to state mode.
func (m *MainArea) SetStateContent(address, content string) {
	if m.stateView == nil {
		return
	}
	m.stateView.SetAddress(address)
	m.stateView.SetContent(content)
	m.mode = ModeStateShow
}

// GetHistoryView returns the history view (for external updates).
func (m *MainArea) GetHistoryView() *views.HistoryView {
	return m.historyView
}

// SetSelectedResource updates the selected resource for diff view.
func (m *MainArea) SetSelectedResource(resource *terraform.ResourceChange) {
	if sameSelectedResource(m.selectedResource, resource) {
		m.selectedResource = resource
		return
	}

	m.selectedResource = resource
	// Reset scroll position when resource changes
	if m.diffViewer != nil {
		m.diffViewer.ResetScroll()
	}
}

func sameSelectedResource(a, b *terraform.ResourceChange) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a == b {
		return true
	}

	return a.Address == b.Address && a.Action == b.Action && a.Change == b.Change
}

// NextDiffHunk moves selection to the next diff hunk.
func (m *MainArea) NextDiffHunk() bool {
	if m.mode != ModeDiff || m.diffViewer == nil {
		return false
	}
	return m.diffViewer.NextHunk()
}

// PrevDiffHunk moves selection to the previous diff hunk.
func (m *MainArea) PrevDiffHunk() bool {
	if m.mode != ModeDiff || m.diffViewer == nil {
		return false
	}
	return m.diffViewer.PrevHunk()
}

// ToggleDiffHunk toggles fold state of the selected diff hunk.
func (m *MainArea) ToggleDiffHunk() bool {
	if m.mode != ModeDiff || m.diffViewer == nil {
		return false
	}
	return m.diffViewer.ToggleCurrentHunk()
}

// DiffTreeParent moves to parent node.
func (m *MainArea) DiffTreeParent() bool {
	if m.mode != ModeDiff || m.diffViewer == nil {
		return false
	}
	return m.diffViewer.TreeParent()
}

// DiffTreeChild expands current node or moves to first child node.
func (m *MainArea) DiffTreeChild() bool {
	if m.mode != ModeDiff || m.diffViewer == nil {
		return false
	}
	return m.diffViewer.TreeChild()
}

// Update handles Bubble Tea messages (implements Panel interface).
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
	case ModeHistoryDetail:
		if m.historyView != nil {
			m.historyView, cmd = m.historyView.Update(msg)
		}
	case ModeStateShow:
		if m.stateView != nil {
			m.stateView, cmd = m.stateView.Update(msg)
		}
	case ModeAbout:
		if m.aboutView != nil {
			m.aboutView, cmd = m.aboutView.Update(msg)
		}
	}

	return m, cmd
}

// HandleKey handles key events.
//
//nolint:gocognit,gocyclo // TUI key handling has inherent complexity from multiple modes
func (m *MainArea) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	// Forward key events to appropriate child based on mode
	if m.mode == ModeLogs {
		// Apply/Plan views handle scrolling (allow even when not focused,
		// so users can scroll logs while on Resources panel during operations)
		if m.applyView != nil {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "home", "end", "k", "j": //nolint:goconst // keyboard keys are clearer as literals
				_, cmd := m.Update(msg)
				return true, cmd
			}
		}
	}

	// Other modes require focus
	if !m.focused {
		return false, nil
	}

	switch m.mode {
	case ModeLogs:
		// Already handled above for unfocused state
	case ModeDiff:
		// Diff viewer scrolling
		if m.diffViewer != nil {
			switch msg.String() {
			case "up", "k":
				m.diffViewer.ScrollUp(1)
				return true, nil
			case "down", "j":
				m.diffViewer.ScrollDown(1)
				return true, nil
			case "pgup":
				m.diffViewer.ScrollUp(m.height / 2)
				return true, nil
			case "pgdown":
				m.diffViewer.ScrollDown(m.height / 2)
				return true, nil
			case "home", "g":
				m.diffViewer.ScrollToTop()
				return true, nil
			case "end", "G":
				m.diffViewer.ScrollToBottom()
				return true, nil
			}
		}
	case ModeHistoryDetail:
		// History view handles scrolling
		if m.historyView != nil {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "home", "end", "k", "j":
				_, cmd := m.Update(msg)
				return true, cmd
			}
		}
	case ModeStateShow:
		// State view handles scrolling
		if m.stateView != nil {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "home", "end", "k", "j":
				_, cmd := m.Update(msg)
				return true, cmd
			}
		}
	case ModeAbout:
		// About view handles scrolling
		if m.aboutView != nil {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "home", "end", "k", "j":
				_, cmd := m.Update(msg)
				return true, cmd
			}
		}
	}

	return false, nil
}

// View renders the main area.
//
//nolint:gocognit,gocyclo,funlen // Rendering multiple view modes requires branching
func (m *MainArea) View() string {
	if m.styles == nil {
		return ""
	}
	if m.height <= 0 {
		return ""
	}

	var content string
	var title string
	var tabs []string

	switch m.mode {
	case ModeLogs:
		// Show operation logs (plan or apply)
		title = "Operation Logs"
		tabs = []string{title}
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
		title = "Diff View"
		tabs = []string{title}
		if m.diffViewer != nil {
			content = m.diffViewer.View(m.selectedResource)
		} else {
			content = m.styles.Dimmed.Render("No diff available")
		}

	case ModeHistoryDetail:
		// Show history detail (formatted logs)
		if m.historyView != nil {
			historyTitle := m.historyView.GetTitle()
			if historyTitle != "" {
				title = historyTitle
			} else {
				title = "History Detail"
			}
			content = m.historyView.ViewContent()
		} else {
			title = "History Detail"
			content = m.styles.Dimmed.Render("No history detail available")
		}
		tabs = []string{title}

	case ModeStateShow:
		// Show terraform state details
		if m.stateView != nil {
			title = "State Details"
			content = m.stateView.ViewContent()
		} else {
			title = "State Details"
			content = m.styles.Dimmed.Render("No state data available")
		}
		tabs = []string{title}

	case ModeAbout:
		// Show about/info screen
		title = "About"
		tabs = []string{title}
		if m.aboutView != nil {
			content = m.aboutView.ViewContent()
		} else {
			content = m.styles.Dimmed.Render("lazytf")
		}
	}

	// Set footer text based on mode
	var footerText string
	switch m.mode {
	case ModeDiff, ModeLogs, ModeHistoryDetail, ModeStateShow, ModeAbout:
		footerText = ""
	}

	// Split content into lines for frame rendering
	contentLines := strings.Split(content, "\n")
	contentHeight := max(1, m.height-2)

	// Configure the frame
	m.frame.SetConfig(components.PanelFrameConfig{
		PanelID:       "[0]",
		Tabs:          tabs,
		ActiveTab:     0,
		Focused:       m.focused,
		FooterText:    footerText,
		ShowScrollbar: len(contentLines) > contentHeight,
		ScrollPos:     0, // Content providers handle their own scrolling
		ThumbSize:     m.calculateThumbSize(contentHeight, len(contentLines)),
	})

	// Pad content lines to fill panel
	result := make([]string, contentHeight)
	contentW := m.frame.ContentWidth()
	emptyLine := components.GetPadding(contentW)
	for i := range contentHeight {
		if i < len(contentLines) {
			result[i] = m.padLine(contentLines[i], contentW)
		} else {
			result[i] = emptyLine
		}
	}

	return m.frame.RenderWithContent(result)
}

// calculateThumbSize calculates the thumb size for the scrollbar.
func (m *MainArea) calculateThumbSize(visibleHeight, totalLines int) float64 {
	if totalLines <= visibleHeight || visibleHeight <= 0 {
		return 1.0
	}
	thumbSize := float64(visibleHeight) / float64(totalLines)
	if thumbSize > 1.0 {
		thumbSize = 1.0
	}
	return thumbSize
}

// padLine pads a line to the given width.
func (m *MainArea) padLine(line string, width int) string {
	return components.PadLine(line, width)
}

// GetApplyView returns the apply view (for external updates).
func (m *MainArea) GetApplyView() *views.ApplyView {
	return m.applyView
}

// GetPlanView returns the plan view (for external updates).
func (m *MainArea) GetPlanView() *views.PlanView {
	return m.planView
}

// GetDiffViewer returns the diff viewer (for external updates).
func (m *MainArea) GetDiffViewer() *components.DiffViewer {
	return m.diffViewer
}
