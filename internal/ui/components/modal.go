package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// Modal renders a centered popup modal that overlays the main content.
// The modal displays content in the center of the screen while the
// background content remains visible around it. Supports scrolling.
type Modal struct {
	styles       *styles.Styles
	width        int
	height       int
	title        string
	contentLines []string
	scrollOffset int
	visible      bool
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

// SetContent sets the modal body content.
func (m *Modal) SetContent(content string) {
	if content == "" {
		m.contentLines = nil
	} else {
		m.contentLines = strings.Split(content, "\n")
	}
	m.scrollOffset = 0
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

// IsVisible returns whether the modal is currently visible.
func (m *Modal) IsVisible() bool {
	return m.visible
}

// ScrollUp scrolls the modal content up by one line.
func (m *Modal) ScrollUp() {
	if m.scrollOffset > 0 {
		m.scrollOffset--
	}
}

// ScrollDown scrolls the modal content down by one line.
func (m *Modal) ScrollDown() {
	maxScroll := m.maxScrollOffset()
	if m.scrollOffset < maxScroll {
		m.scrollOffset++
	}
}

// GetScrollInfo returns debug info about scroll state.
func (m *Modal) GetScrollInfo() (offset, maxOffset, viewport, totalLines int) {
	return m.scrollOffset, m.maxScrollOffset(), m.viewportHeight(), len(m.contentLines)
}

// maxScrollOffset returns the maximum scroll offset based on content and viewport size.
func (m *Modal) maxScrollOffset() int {
	viewportHeight := m.viewportHeight()
	if len(m.contentLines) <= viewportHeight {
		return 0
	}
	return len(m.contentLines) - viewportHeight
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
	for len(baseLines) < m.height {
		baseLines = append(baseLines, strings.Repeat(" ", m.width))
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
			baseLine = baseLine + strings.Repeat(" ", m.width-baseLineWidth)
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
	viewportHeight := m.viewportHeight()

	var lines []string

	// Add title if present
	if m.title != "" {
		lines = append(lines, m.styles.Highlight.Render(m.title))
		lines = append(lines, "")
	}

	// Add visible content based on scroll offset
	totalLines := len(m.contentLines)
	if totalLines > 0 {
		endIdx := min(m.scrollOffset+viewportHeight, totalLines)
		visibleLines := m.contentLines[m.scrollOffset:endIdx]
		lines = append(lines, visibleLines...)
	}

	// Add scroll indicator if content is scrollable
	if totalLines > viewportHeight {
		// Pad to fill viewport
		currentVisible := min(viewportHeight, totalLines-m.scrollOffset)
		for len(lines)-2 < viewportHeight { // -2 for title and blank line
			lines = append(lines, "")
		}
		// Add position indicator
		position := fmt.Sprintf("%d of %d", m.scrollOffset+currentVisible, totalLines)
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
