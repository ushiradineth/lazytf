package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// Tab represents a single tab in a TabbedPanel.
type Tab struct {
	Name    string
	Content TabContent
}

// TabContent is the interface that tab content must implement.
type TabContent interface {
	View() string
	SetSize(width, height int)
	HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd)
}

// TabbedPanel is a panel that can display multiple tabs.
type TabbedPanel struct {
	tabs        []Tab
	activeIndex int
	styles      *styles.Styles
	width       int
	height      int
	focused     bool
	panelID     string // For display purposes like "[2]"
}

// NewTabbedPanel creates a new tabbed panel.
func NewTabbedPanel(panelID string, s *styles.Styles) *TabbedPanel {
	return &TabbedPanel{
		tabs:    make([]Tab, 0),
		styles:  s,
		panelID: panelID,
	}
}

// AddTab adds a new tab to the panel.
func (t *TabbedPanel) AddTab(name string, content TabContent) {
	t.tabs = append(t.tabs, Tab{Name: name, Content: content})
}

// SetActiveTab sets the active tab by index.
func (t *TabbedPanel) SetActiveTab(index int) {
	if index >= 0 && index < len(t.tabs) {
		t.activeIndex = index
	}
}

// GetActiveTab returns the currently active tab index.
func (t *TabbedPanel) GetActiveTab() int {
	return t.activeIndex
}

// GetActiveTabName returns the name of the currently active tab.
func (t *TabbedPanel) GetActiveTabName() string {
	if t.activeIndex >= 0 && t.activeIndex < len(t.tabs) {
		return t.tabs[t.activeIndex].Name
	}
	return ""
}

// NextTab switches to the next tab.
func (t *TabbedPanel) NextTab() {
	if len(t.tabs) > 0 {
		t.activeIndex = (t.activeIndex + 1) % len(t.tabs)
	}
}

// PrevTab switches to the previous tab.
func (t *TabbedPanel) PrevTab() {
	if len(t.tabs) > 0 {
		t.activeIndex = (t.activeIndex - 1 + len(t.tabs)) % len(t.tabs)
	}
}

// SetSize updates the panel dimensions.
func (t *TabbedPanel) SetSize(width, height int) {
	t.width = width
	t.height = height
	// Update content sizes (account for border)
	contentWidth := width - 2
	contentHeight := height - 2
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}
	for _, tab := range t.tabs {
		if tab.Content != nil {
			tab.Content.SetSize(contentWidth, contentHeight)
		}
	}
}

// SetFocused sets the focus state.
func (t *TabbedPanel) SetFocused(focused bool) {
	t.focused = focused
}

// IsFocused returns whether the panel is focused.
func (t *TabbedPanel) IsFocused() bool {
	return t.focused
}

// HandleKey handles key events.
func (t *TabbedPanel) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	switch msg.String() {
	case "[":
		t.PrevTab()
		return true, nil
	case "]":
		t.NextTab()
		return true, nil
	}

	// Forward to active tab content
	if t.activeIndex >= 0 && t.activeIndex < len(t.tabs) {
		if content := t.tabs[t.activeIndex].Content; content != nil {
			return content.HandleKey(msg)
		}
	}
	return false, nil
}

// Update handles Bubble Tea messages.
func (t *TabbedPanel) Update(msg tea.Msg) (any, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		handled, cmd := t.HandleKey(keyMsg)
		if handled {
			return t, cmd
		}
	}
	return t, nil
}

// View renders the tabbed panel.
func (t *TabbedPanel) View() string {
	if t.styles == nil || len(t.tabs) == 0 {
		return ""
	}

	// Determine border style based on focus
	borderStyle := t.styles.Border
	titleStyle := t.styles.PanelTitle
	if t.focused {
		borderStyle = t.styles.FocusedBorder
		titleStyle = t.styles.FocusedPanelTitle
	}

	// Get active content
	var content string
	if t.activeIndex >= 0 && t.activeIndex < len(t.tabs) {
		if tab := t.tabs[t.activeIndex]; tab.Content != nil {
			content = tab.Content.View()
		}
	}

	// Build panel with border
	panel := borderStyle.
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Width(t.width - 2).
		Height(t.height - 2).
		Render(content)

	// Build title with tabs
	titleText := t.buildTitleWithTabs(titleStyle)
	titleRendered := titleStyle.Render(" " + titleText + " ")

	lines := strings.Split(panel, "\n")
	if len(lines) > 0 && t.width > 4 {
		if line, ok := RenderPanelTitleLine(t.width, borderStyle, titleRendered); ok {
			lines[0] = line
		}
	}

	return strings.Join(lines, "\n")
}

// buildTitleWithTabs builds the title string with tab indicators.
func (t *TabbedPanel) buildTitleWithTabs(_ lipgloss.Style) string {
	if len(t.tabs) <= 1 {
		// Single tab, just show the panel ID and tab name
		name := ""
		if len(t.tabs) == 1 {
			name = t.tabs[0].Name
		}
		return t.panelID + " " + name
	}

	// Multiple tabs - show tab bar
	var parts []string
	parts = append(parts, t.panelID)

	for i, tab := range t.tabs {
		if i == t.activeIndex {
			parts = append(parts, "["+tab.Name+"]")
		} else {
			parts = append(parts, t.styles.Dimmed.Render(tab.Name))
		}
	}

	return strings.Join(parts, " ")
}

// GetContent returns the content for a specific tab index.
func (t *TabbedPanel) GetContent(index int) TabContent {
	if index >= 0 && index < len(t.tabs) {
		return t.tabs[index].Content
	}
	return nil
}

// GetActiveContent returns the currently active tab's content.
func (t *TabbedPanel) GetActiveContent() TabContent {
	return t.GetContent(t.activeIndex)
}

// TabCount returns the number of tabs.
func (t *TabbedPanel) TabCount() int {
	return len(t.tabs)
}
