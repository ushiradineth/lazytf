package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// ViewportPanel is a unified base component for viewport-based panels.
// It provides:
// - Rounded border with title and tabs
// - Scrollbar on the right side
// - Free-form scrollable content
// - Keyboard navigation (up/down/pgup/pgdown)
type ViewportPanel struct {
	styles   *styles.Styles
	frame    *PanelFrame
	viewport viewport.Model
	width    int
	height   int

	// Focus state
	focused bool

	// Panel configuration
	panelID   string   // e.g., "[0]"
	tabs      []string // Tab names
	activeTab int      // Currently active tab index
}

// NewViewportPanel creates a new viewport panel with the given panel ID.
func NewViewportPanel(panelID string, s *styles.Styles) *ViewportPanel {
	if s == nil {
		s = styles.DefaultStyles()
	}
	return &ViewportPanel{
		styles:   s,
		frame:    NewPanelFrame(s),
		viewport: viewport.New(0, 0),
		panelID:  panelID,
		tabs:     make([]string, 0),
	}
}

// SetSize updates the panel dimensions.
func (v *ViewportPanel) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.frame.SetSize(width, height)

	// Update viewport size (content area)
	contentHeight := v.contentHeight()
	contentWidth := v.contentWidth()
	v.viewport.Width = contentWidth
	v.viewport.Height = contentHeight
}

// SetFocused sets the focus state.
func (v *ViewportPanel) SetFocused(focused bool) {
	v.focused = focused
}

// IsFocused returns whether the panel is focused.
func (v *ViewportPanel) IsFocused() bool {
	return v.focused
}

// SetTabs sets the tab names for the panel.
func (v *ViewportPanel) SetTabs(tabs []string) {
	v.tabs = tabs
	if v.activeTab >= len(tabs) {
		v.activeTab = 0
	}
}

// SetActiveTab sets the active tab index.
func (v *ViewportPanel) SetActiveTab(index int) {
	if index >= 0 && index < len(v.tabs) {
		v.activeTab = index
	}
}

// GetActiveTab returns the currently active tab index.
func (v *ViewportPanel) GetActiveTab() int {
	return v.activeTab
}

// SetContent sets the viewport content.
func (v *ViewportPanel) SetContent(content string) {
	v.viewport.SetContent(content)
}

// GotoTop scrolls to the top.
func (v *ViewportPanel) GotoTop() {
	v.viewport.GotoTop()
}

// GotoBottom scrolls to the bottom.
func (v *ViewportPanel) GotoBottom() {
	v.viewport.GotoBottom()
}

// ScrollUp scrolls up one line.
func (v *ViewportPanel) ScrollUp() {
	v.viewport.LineUp(1)
}

// ScrollDown scrolls down one line.
func (v *ViewportPanel) ScrollDown() {
	v.viewport.LineDown(1)
}

// PageUp scrolls up one page.
func (v *ViewportPanel) PageUp() {
	v.viewport.ViewUp()
}

// PageDown scrolls down one page.
func (v *ViewportPanel) PageDown() {
	v.viewport.ViewDown()
}

// Update handles Bubble Tea messages.
func (v *ViewportPanel) Update(msg tea.Msg) (*ViewportPanel, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the viewport panel.
func (v *ViewportPanel) View() string {
	if v.styles == nil || v.height <= 0 || v.width <= 0 {
		return ""
	}

	// Calculate dimensions
	contentHeight := v.contentHeight()
	contentWidth := v.contentWidth()
	totalLines := v.viewport.TotalLineCount()
	hasScrollbar := totalLines > contentHeight

	// Calculate scroll position for frame
	scrollPos, thumbSize := v.calculateScrollParams(contentHeight, totalLines)

	// Configure the frame
	v.frame.SetConfig(PanelFrameConfig{
		PanelID:       v.panelID,
		Tabs:          v.tabs,
		ActiveTab:     v.activeTab,
		Focused:       v.focused,
		FooterText:    "", // Viewport panels typically don't show item count
		ScrollPos:     scrollPos,
		ThumbSize:     thumbSize,
		ShowScrollbar: hasScrollbar,
	})

	// Get viewport content and split into lines
	content := v.viewport.View()
	lines := strings.Split(content, "\n")

	// Ensure we have exactly contentHeight lines
	result := make([]string, contentHeight)
	for i := range contentHeight {
		if i < len(lines) {
			result[i] = v.padLine(lines[i], contentWidth)
		} else {
			result[i] = strings.Repeat(" ", contentWidth)
		}
	}

	return v.frame.RenderWithContent(result)
}

// contentHeight returns the height available for content (excluding borders).
func (v *ViewportPanel) contentHeight() int {
	h := v.height - 2 // top and bottom border
	if h < 1 {
		return 1
	}
	return h
}

// contentWidth returns the width available for content (excluding borders and scrollbar).
func (v *ViewportPanel) contentWidth() int {
	w := v.width - 2 // left and right border
	// Always reserve space for scrollbar if content might overflow
	totalLines := v.viewport.TotalLineCount()
	if totalLines > v.contentHeight() {
		w-- // scrollbar
	}
	if w < 1 {
		return 1
	}
	return w
}

// calculateScrollParams calculates scroll parameters for the frame.
func (v *ViewportPanel) calculateScrollParams(visibleHeight, totalLines int) (scrollPos, thumbSize float64) {
	if totalLines <= visibleHeight || visibleHeight <= 0 {
		return 0, 1.0
	}

	// Thumb size as fraction of track
	thumbSize = float64(visibleHeight) / float64(totalLines)
	if thumbSize > 1.0 {
		thumbSize = 1.0
	}

	// Scroll position as fraction
	scrollRange := totalLines - visibleHeight
	if scrollRange > 0 {
		scrollPos = float64(v.viewport.YOffset) / float64(scrollRange)
	}
	if scrollPos > 1.0 {
		scrollPos = 1.0
	}

	return scrollPos, thumbSize
}

// padLine pads a line to the given width.
func (v *ViewportPanel) padLine(line string, width int) string {
	// Use lipgloss.Width for accurate visual width calculation
	visibleWidth := len([]rune(line)) // Simple approximation
	if visibleWidth >= width {
		// Truncate if too long
		runes := []rune(line)
		if len(runes) > width {
			return string(runes[:width])
		}
		return line
	}
	return line + strings.Repeat(" ", width-visibleWidth)
}
