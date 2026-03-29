package components

import (
	"strconv"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// ListPanelItem represents an item that can be rendered in a ListPanel.
type ListPanelItem interface {
	// Render returns the rendered string for this item.
	// width is the available content width (excluding scrollbar).
	// selected indicates if this item is currently selected.
	Render(styles *styles.Styles, width int, selected bool) string
}

// ListPanel is a unified base component for all list-based panels.
// It provides:
// - Rounded border with title and tabs
// - Scrollbar on the right side
// - Item count footer ("7 of 29")
// - Full-line selection highlighting
// - Keyboard navigation (up/down).
type ListPanel struct {
	styles *styles.Styles
	frame  *PanelFrame
	width  int
	height int

	// Focus state
	focused bool

	// Panel configuration
	panelID   string   // e.g., "[2]"
	tabs      []string // Tab names, e.g., ["Resources", "Modules"]
	activeTab int      // Currently active tab index

	// List state
	items         []ListPanelItem
	selectedIndex int
	scrollOffset  int
	lastMove      int // -1 for up, 1 for down, 0 for none
}

// NewListPanel creates a new list panel with the given panel ID.
func NewListPanel(panelID string, s *styles.Styles) *ListPanel {
	if s == nil {
		s = styles.DefaultStyles()
	}
	return &ListPanel{
		styles:  s,
		frame:   NewPanelFrame(s),
		panelID: panelID,
		tabs:    make([]string, 0),
	}
}

// SetSize updates the panel dimensions.
func (l *ListPanel) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.frame.SetSize(width, height)
	l.adjustScrollOffset()
}

// SetFocused sets the focus state.
func (l *ListPanel) SetFocused(focused bool) {
	l.focused = focused
}

// IsFocused returns whether the panel is focused.
func (l *ListPanel) IsFocused() bool {
	return l.focused
}

// SetStyles updates the component styles.
func (l *ListPanel) SetStyles(s *styles.Styles) {
	l.styles = s
	if l.frame != nil {
		l.frame.SetStyles(s)
	}
}

// SetTabs sets the tab names for the panel.
func (l *ListPanel) SetTabs(tabs []string) {
	l.tabs = tabs
	if l.activeTab >= len(tabs) {
		l.activeTab = 0
	}
}

// SetActiveTab sets the active tab index.
func (l *ListPanel) SetActiveTab(index int) {
	if index >= 0 && index < len(l.tabs) {
		l.activeTab = index
	}
}

// GetActiveTab returns the currently active tab index.
func (l *ListPanel) GetActiveTab() int {
	return l.activeTab
}

// NextTab switches to the next tab.
func (l *ListPanel) NextTab() {
	if len(l.tabs) > 1 {
		l.activeTab = (l.activeTab + 1) % len(l.tabs)
	}
}

// PrevTab switches to the previous tab.
func (l *ListPanel) PrevTab() {
	if len(l.tabs) > 1 {
		l.activeTab = (l.activeTab - 1 + len(l.tabs)) % len(l.tabs)
	}
}

// SetItems sets the list items.
func (l *ListPanel) SetItems(items []ListPanelItem) {
	l.items = items
	if l.selectedIndex >= len(items) {
		l.selectedIndex = max(0, len(items)-1)
	}
	l.adjustScrollOffset()
}

// GetSelectedIndex returns the currently selected index.
func (l *ListPanel) GetSelectedIndex() int {
	return l.selectedIndex
}

// SetSelectedIndex sets the selected index.
func (l *ListPanel) SetSelectedIndex(index int) {
	if index >= 0 && index < len(l.items) {
		l.selectedIndex = index
		l.adjustScrollOffset()
	}
}

// SelectVisibleRow sets selection by visible row index within the content area.
func (l *ListPanel) SelectVisibleRow(row int) bool {
	if row < 0 || row >= l.contentHeight() {
		return false
	}
	idx := l.scrollOffset + row
	if idx < 0 || idx >= len(l.items) {
		return false
	}
	l.selectedIndex = idx
	l.lastMove = 0
	l.adjustScrollOffset()
	return true
}

// MoveUp moves the selection up.
func (l *ListPanel) MoveUp() bool {
	if l.selectedIndex > 0 {
		l.selectedIndex--
		l.lastMove = -1
		l.adjustScrollOffset()
		return true
	}
	return false
}

// MoveDown moves the selection down.
func (l *ListPanel) MoveDown() bool {
	if l.selectedIndex < len(l.items)-1 {
		l.selectedIndex++
		l.lastMove = 1
		l.adjustScrollOffset()
		return true
	}
	return false
}

// PageUp moves the selection up by a page.
func (l *ListPanel) PageUp() bool {
	if l.selectedIndex > 0 {
		pageSize := l.contentHeight()
		l.selectedIndex = max(0, l.selectedIndex-pageSize)
		l.lastMove = -1
		l.adjustScrollOffset()
		return true
	}
	return false
}

// PageDown moves the selection down by a page.
func (l *ListPanel) PageDown() bool {
	if l.selectedIndex < len(l.items)-1 {
		pageSize := l.contentHeight()
		l.selectedIndex = min(len(l.items)-1, l.selectedIndex+pageSize)
		l.lastMove = 1
		l.adjustScrollOffset()
		return true
	}
	return false
}

// Home moves to the first item.
func (l *ListPanel) Home() bool {
	if l.selectedIndex != 0 && len(l.items) > 0 {
		l.selectedIndex = 0
		l.lastMove = -1
		l.adjustScrollOffset()
		return true
	}
	return false
}

// End moves to the last item.
func (l *ListPanel) End() bool {
	if len(l.items) > 0 && l.selectedIndex != len(l.items)-1 {
		l.selectedIndex = len(l.items) - 1
		l.lastMove = 1
		l.adjustScrollOffset()
		return true
	}
	return false
}

// ItemCount returns the total number of items.
func (l *ListPanel) ItemCount() int {
	return len(l.items)
}

// View renders the list panel.
func (l *ListPanel) View() string {
	if l.styles == nil || l.height <= 0 || l.width <= 0 {
		return ""
	}

	// Calculate dimensions
	contentHeight := l.contentHeight()
	contentWidth := l.contentWidth()
	hasScrollbar := len(l.items) > contentHeight

	// Configure the frame
	scrollPos, thumbSize := CalculateScrollParams(l.scrollOffset, contentHeight, len(l.items))
	l.frame.SetConfig(PanelFrameConfig{
		PanelID:       l.panelID,
		Tabs:          l.tabs,
		ActiveTab:     l.activeTab,
		Focused:       l.focused,
		FooterText:    l.buildFooterText(),
		ScrollPos:     scrollPos,
		ThumbSize:     thumbSize,
		ShowScrollbar: hasScrollbar,
	})

	// Render content lines
	lines := l.renderContent(contentWidth, contentHeight)

	return l.frame.RenderWithContent(lines)
}

// RenderContentLines returns just the content lines without frame.
// Useful when the caller handles the frame/border separately.
func (l *ListPanel) RenderContentLines(width, height int) []string {
	if l.styles == nil || height <= 0 || width <= 0 {
		return nil
	}
	return l.renderContent(width, height)
}

// GetScrollInfo returns scroll position and thumb size for external scrollbar rendering.
func (l *ListPanel) GetScrollInfo(height int) (scrollPos, thumbSize float64, hasScrollbar bool) {
	hasScrollbar = len(l.items) > height
	scrollPos, thumbSize = CalculateScrollParams(l.scrollOffset, height, len(l.items))
	return
}

// contentHeight returns the height available for content (excluding borders).
func (l *ListPanel) contentHeight() int {
	h := l.height - 2 // top and bottom border
	if h < 1 {
		return 1
	}
	return h
}

// contentWidth returns the width available for content (excluding borders and scrollbar).
func (l *ListPanel) contentWidth() int {
	w := l.width - 2 // left and right border
	if len(l.items) > l.contentHeight() {
		w-- // scrollbar
	}
	if w < 1 {
		return 1
	}
	return w
}

// adjustScrollOffset ensures the selected item is visible.
func (l *ListPanel) adjustScrollOffset() {
	contentHeight := l.contentHeight()
	if contentHeight <= 0 || len(l.items) == 0 {
		l.scrollOffset = 0
		return
	}

	// Anchor positions for smooth scrolling
	anchorTop := min(2, contentHeight-1)
	anchorBottom := max(contentHeight-3, anchorTop)
	l.scrollOffset = adjustScrollOffset(
		l.scrollOffset,
		l.selectedIndex,
		len(l.items),
		contentHeight,
		l.lastMove,
		anchorTop,
		anchorBottom,
	)
}

// renderContent renders the visible content lines.
func (l *ListPanel) renderContent(width, height int) []string {
	lines := make([]string, height)

	emptyLine := GetPadding(width)
	if len(l.items) == 0 {
		emptyMsg := l.styles.Dimmed.Render("No items")
		lines[0] = l.padLine(emptyMsg, width)
		for i := range height - 1 {
			lines[i+1] = emptyLine
		}
		return lines
	}

	// Render visible items
	// Only show selection highlight when focused
	for i := range height {
		itemIndex := l.scrollOffset + i
		if itemIndex < len(l.items) {
			isSelected := l.focused && itemIndex == l.selectedIndex
			item := l.items[itemIndex]
			rendered := item.Render(l.styles, width, isSelected)
			lines[i] = l.padLine(rendered, width)
		} else {
			lines[i] = emptyLine
		}
	}

	return lines
}

// buildFooterText builds the footer text with item count.
func (l *ListPanel) buildFooterText() string {
	if len(l.items) == 0 {
		return ""
	}
	return FormatItemCount(l.selectedIndex+1, len(l.items))
}

// padLine pads or truncates a line to the given width.
func (l *ListPanel) padLine(line string, width int) string {
	return PadLine(line, width)
}

func intToString(n int) string {
	return strconv.Itoa(n)
}
