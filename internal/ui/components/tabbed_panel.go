package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

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
	frame       *PanelFrame
	focused     bool
	panelID     string // For display purposes like "[2]"
}

// NewTabbedPanel creates a new tabbed panel.
func NewTabbedPanel(panelID string, s *styles.Styles) *TabbedPanel {
	if s == nil {
		s = styles.DefaultStyles()
	}
	return &TabbedPanel{
		tabs:    make([]Tab, 0),
		styles:  s,
		frame:   NewPanelFrame(s),
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
	t.frame.SetSize(width, height)
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

// SetStyles updates the component styles.
func (t *TabbedPanel) SetStyles(s *styles.Styles) {
	t.styles = s
	if t.frame != nil {
		t.frame.SetStyles(s)
	}
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

	// Get active content
	var content string
	if t.activeIndex >= 0 && t.activeIndex < len(t.tabs) {
		if tab := t.tabs[t.activeIndex]; tab.Content != nil {
			content = tab.Content.View()
		}
	}

	// Build tab names array for frame
	tabNames := make([]string, len(t.tabs))
	for i, tab := range t.tabs {
		tabNames[i] = tab.Name
	}

	// Configure frame
	t.frame.SetConfig(PanelFrameConfig{
		PanelID:       t.panelID,
		Tabs:          tabNames,
		ActiveTab:     t.activeIndex,
		Focused:       t.focused,
		FooterText:    "",
		ShowScrollbar: false,
	})

	// Split content into lines for frame
	contentHeight := t.frame.ContentHeight()
	contentWidth := t.frame.ContentWidth()
	contentLines := strings.Split(content, "\n")

	// Pad content lines to fill panel
	result := make([]string, contentHeight)
	for i := range contentHeight {
		if i < len(contentLines) {
			result[i] = PadLine(contentLines[i], contentWidth)
		} else {
			result[i] = strings.Repeat(" ", contentWidth)
		}
	}

	return t.frame.RenderWithContent(result)
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
