package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// HelpItem represents a single selectable item in the help modal.
type HelpItem struct {
	Key         string // Key binding (e.g., "j/k")
	Description string // Description (e.g., "move selection")
	IsHeader    bool   // True if this is a section header
}

// ModalAction represents an action button in confirm mode.
type ModalAction struct {
	Key   string // Key to trigger (e.g., "y")
	Label string // Display label (e.g., "Yes, apply")
}

// Modal renders a centered popup modal that overlays the main content.
// The modal displays content in the center of the screen while the
// background content remains visible around it. Supports scrolling and item selection.
type Modal struct {
	styles       *styles.Styles
	width        int
	height       int
	title        string
	contentLines []string
	scrollOffset int
	visible      bool

	// Item selection mode (for help modal)
	items         []HelpItem
	selectedIndex int
	itemMode      bool // Whether to use item selection mode

	// Confirm mode (for confirmation dialogs)
	confirmMode    bool
	confirmMessage string
	confirmActions []ModalAction
}

// NewModal creates a new modal component.
func NewModal(s *styles.Styles) *Modal {
	return &Modal{
		styles: s,
	}
}

// SetSize updates the available screen size.
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetTitle sets the modal title.
func (m *Modal) SetTitle(title string) {
	m.title = title
}

// SetContent sets the modal body content (for non-item mode).
func (m *Modal) SetContent(content string) {
	if content == "" {
		m.contentLines = nil
	} else {
		m.contentLines = strings.Split(content, "\n")
	}
	m.scrollOffset = 0
	m.itemMode = false
}

// SetItems sets the help items for item selection mode.
func (m *Modal) SetItems(items []HelpItem) {
	m.items = items
	m.itemMode = true
	m.confirmMode = false
	// Find first non-header item to select
	m.selectedIndex = 0
	for i, item := range items {
		if !item.IsHeader {
			m.selectedIndex = i
			break
		}
	}
	m.scrollOffset = 0
}

// SetConfirm sets up the modal for confirmation mode with actions.
func (m *Modal) SetConfirm(message string, actions []ModalAction) {
	m.confirmMessage = message
	m.confirmActions = actions
	m.confirmMode = true
	m.itemMode = false
	m.scrollOffset = 0
}

// IsConfirmMode returns whether the modal is in confirm mode.
func (m *Modal) IsConfirmMode() bool {
	return m.confirmMode
}

// GetConfirmActions returns the confirm actions for key handling.
func (m *Modal) GetConfirmActions() []ModalAction {
	return m.confirmActions
}

// Show makes the modal visible.
func (m *Modal) Show() {
	m.visible = true
	m.scrollOffset = 0
}

// Hide hides the modal.
func (m *Modal) Hide() {
	m.visible = false
	m.scrollOffset = 0
}

// GetSelectedIndex returns the current selected item index.
func (m *Modal) GetSelectedIndex() int {
	return m.selectedIndex
}

// SetSelectedIndex sets the selected item index and adjusts scroll.
func (m *Modal) SetSelectedIndex(index int) {
	if index < 0 || index >= len(m.items) {
		return
	}
	// Skip headers
	if m.items[index].IsHeader {
		return
	}
	m.selectedIndex = index
	m.ensureSelectionVisible()
}

// SetStyles updates the modal styles.
func (m *Modal) SetStyles(s *styles.Styles) {
	m.styles = s
}

// IsVisible returns whether the modal is currently visible.
func (m *Modal) IsVisible() bool {
	return m.visible
}

// ScrollUp scrolls the modal content up by one line (or moves selection up in item mode).
func (m *Modal) ScrollUp() {
	if m.itemMode {
		m.moveSelectionUp()
		return
	}
	if m.scrollOffset > 0 {
		m.scrollOffset--
	}
}

// ScrollDown scrolls the modal content down by one line (or moves selection down in item mode).
func (m *Modal) ScrollDown() {
	if m.itemMode {
		m.moveSelectionDown()
		return
	}
	maxScroll := m.maxScrollOffset()
	if m.scrollOffset < maxScroll {
		m.scrollOffset++
	}
}

// moveSelectionUp moves to the previous non-header item.
func (m *Modal) moveSelectionUp() {
	for i := m.selectedIndex - 1; i >= 0; i-- {
		if !m.items[i].IsHeader {
			m.selectedIndex = i
			m.ensureSelectionVisible()
			return
		}
	}
}

// moveSelectionDown moves to the next non-header item.
func (m *Modal) moveSelectionDown() {
	for i := m.selectedIndex + 1; i < len(m.items); i++ {
		if !m.items[i].IsHeader {
			m.selectedIndex = i
			m.ensureSelectionVisible()
			return
		}
	}
}

// ensureSelectionVisible adjusts scroll offset to keep selection visible.
func (m *Modal) ensureSelectionVisible() {
	viewportHeight := m.viewportHeight()
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	} else if m.selectedIndex >= m.scrollOffset+viewportHeight {
		m.scrollOffset = m.selectedIndex - viewportHeight + 1
	}
}

// GetScrollInfo returns debug info about scroll state.
func (m *Modal) GetScrollInfo() (offset, maxOffset, viewport, totalLines int) {
	return m.scrollOffset, m.maxScrollOffset(), m.viewportHeight(), m.totalLines()
}

// totalLines returns the total number of content lines.
func (m *Modal) totalLines() int {
	if m.itemMode {
		return len(m.items)
	}
	return len(m.contentLines)
}

// maxScrollOffset returns the maximum scroll offset based on content and viewport size.
func (m *Modal) maxScrollOffset() int {
	viewportHeight := m.viewportHeight()
	total := m.totalLines()
	if total <= viewportHeight {
		return 0
	}
	return total - viewportHeight
}

// viewportHeight returns the number of content lines that can be displayed.
func (m *Modal) viewportHeight() int {
	// 70% of screen height, minus border (2), padding (2), title (2 if present), footer (1)
	// Cap at 20 lines max to ensure scrollability on large terminals
	overhead := 5
	if m.title != "" {
		overhead += 2
	}
	calculated := int(float64(m.height)*0.7) - overhead
	return max(min(calculated, 20), 3)
}

// View renders the modal box (without background).
// Use Overlay() to render on top of existing content.
func (m *Modal) View() string {
	if m.styles == nil || !m.visible {
		return ""
	}

	return m.renderBox()
}

// Overlay renders the modal centered on top of the base view.
// The base view remains visible around the modal.
func (m *Modal) Overlay(baseView string) string {
	if m.styles == nil || !m.visible || m.width == 0 || m.height == 0 {
		return baseView
	}

	modalBox := m.renderBox()
	modalWidth := lipgloss.Width(modalBox)
	modalHeight := lipgloss.Height(modalBox)

	// Calculate position to center the modal
	startRow := (m.height - modalHeight) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (m.width - modalWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	baseLines := strings.Split(baseView, "\n")
	modalLines := strings.Split(modalBox, "\n")

	// Ensure we have enough lines in base view
	emptyLine := GetPadding(m.width)
	for len(baseLines) < m.height {
		baseLines = append(baseLines, emptyLine)
	}

	// Overlay modal on base view
	for i, modalLine := range modalLines {
		row := startRow + i
		if row < 0 || row >= len(baseLines) {
			continue
		}

		baseLine := baseLines[row]

		// Ensure base line is wide enough (visual width)
		baseLineWidth := ansi.StringWidth(baseLine)
		if baseLineWidth < m.width {
			baseLine = baseLine + GetPadding(m.width-baseLineWidth)
		}

		// Build the new line using ANSI-aware functions:
		// [left part][modal line][right part]
		left := ansi.Truncate(baseLine, startCol, "")
		right := ANSICutLeft(baseLine, startCol+modalWidth)

		baseLines[row] = left + modalLine + right
	}

	// Return only up to m.height lines
	if len(baseLines) > m.height {
		baseLines = baseLines[:m.height]
	}

	return strings.Join(baseLines, "\n")
}

func (m *Modal) renderBox() string {
	// Calculate modal dimensions (max 80% of screen width, min 30, max 80)
	maxWidth := min(max(int(float64(m.width)*0.8), 30), 80)

	// Confirm mode has its own rendering
	if m.confirmMode {
		return m.renderConfirmBox(maxWidth)
	}

	viewportHeight := m.viewportHeight()

	var lines []string

	// Add title if present
	if m.title != "" {
		lines = append(lines, m.styles.Highlight.Render(m.title))
		lines = append(lines, "")
	}

	// Add visible content based on scroll offset
	if m.itemMode {
		lines = append(lines, m.renderItemContent(viewportHeight, maxWidth-6)...) // -6 for padding
	} else {
		lines = append(lines, m.renderTextContent(viewportHeight)...)
	}

	// Add scroll indicator if content is scrollable
	total := m.totalLines()
	if total > viewportHeight {
		// Pad to fill viewport
		titleOffset := 0
		if m.title != "" {
			titleOffset = 2
		}
		for len(lines)-titleOffset < viewportHeight {
			lines = append(lines, "")
		}
		// Add position indicator
		currentVisible := min(viewportHeight, total-m.scrollOffset)
		position := fmt.Sprintf("%d of %d", m.scrollOffset+currentVisible, total)
		indicator := m.styles.Dimmed.Render(position)
		lines = append(lines, "")
		lines = append(lines, indicator)
	}

	content := strings.Join(lines, "\n")

	// Render with border
	box := m.styles.FocusedBorder.
		Width(maxWidth).
		Padding(1, 2).
		Render(content)

	return box
}

// renderConfirmBox renders the confirmation dialog box.
func (m *Modal) renderConfirmBox(maxWidth int) string {
	var lines []string

	// Add title if present
	if m.title != "" {
		lines = append(lines, m.styles.Highlight.Render(m.title))
		lines = append(lines, "")
	}

	// Add message lines
	if m.confirmMessage != "" {
		lines = append(lines, strings.Split(m.confirmMessage, "\n")...)
		lines = append(lines, "")
	}

	// Add action buttons
	if len(m.confirmActions) > 0 {
		actions := make([]string, 0, len(m.confirmActions))
		for _, action := range m.confirmActions {
			actionStr := fmt.Sprintf("[%s] %s", m.styles.HelpKey.Render(action.Key), action.Label)
			actions = append(actions, actionStr)
		}
		lines = append(lines, strings.Join(actions, "    "))
	}

	content := strings.Join(lines, "\n")

	// Render with border
	box := m.styles.FocusedBorder.
		Width(maxWidth).
		Padding(1, 2).
		Render(content)

	return box
}

// renderTextContent renders text content (non-item mode).
func (m *Modal) renderTextContent(viewportHeight int) []string {
	totalLines := len(m.contentLines)
	if totalLines == 0 {
		return nil
	}
	endIdx := min(m.scrollOffset+viewportHeight, totalLines)
	return m.contentLines[m.scrollOffset:endIdx]
}

// renderItemContent renders help items with selection highlighting.
func (m *Modal) renderItemContent(viewportHeight, contentWidth int) []string {
	if len(m.items) == 0 {
		return nil
	}

	// Calculate key column width
	keyWidth := 8
	for _, item := range m.items {
		if !item.IsHeader && len(item.Key) > keyWidth {
			keyWidth = len(item.Key)
		}
	}

	var lines []string
	endIdx := min(m.scrollOffset+viewportHeight, len(m.items))

	for i := m.scrollOffset; i < endIdx; i++ {
		item := m.items[i]
		if item.IsHeader {
			// Render header (section title)
			lines = append(lines, m.styles.Highlight.Render(item.Key))
		} else {
			// Render key-value pair with selection highlight
			isSelected := i == m.selectedIndex
			line := m.renderHelpItem(item, keyWidth, contentWidth, isSelected)
			lines = append(lines, line)
		}
	}

	return lines
}

// renderHelpItem renders a single help item with optional selection highlighting.
func (m *Modal) renderHelpItem(item HelpItem, keyWidth, contentWidth int, isSelected bool) string {
	keyText := fmt.Sprintf("%-*s", keyWidth, item.Key)

	if isSelected {
		// Full-width selection highlight
		fullLine := keyText + "  " + item.Description
		// Pad to content width
		if len(fullLine) < contentWidth {
			fullLine += GetPadding(contentWidth - len(fullLine))
		}
		return m.styles.SelectedLine.Render(fullLine)
	}

	// Normal rendering
	left := m.styles.HelpKey.Render(keyText)
	right := m.styles.HelpValue.Render(item.Description)
	return left + "  " + right
}
