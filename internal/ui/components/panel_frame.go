package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/styles"
)

// PanelFrameConfig holds configuration for a panel frame.
type PanelFrameConfig struct {
	PanelID       string   // e.g., "[2]"
	Tabs          []string // Tab names, e.g., ["Resources", "Modules"]
	ActiveTab     int      // Currently active tab index
	Focused       bool     // Whether the panel is focused
	FooterText    string   // Footer text, e.g., "7 of 29"
	ScrollPos     float64  // 0.0-1.0 for scrollbar thumb position
	ThumbSize     float64  // 0.0-1.0 for scrollbar thumb size (fraction of track)
	ShowScrollbar bool     // Whether to show a scrollbar
}

// PanelFrame renders the visual frame (borders, title, scrollbar, footer) for panels.
// It is a pure rendering component with no state of its own.
type PanelFrame struct {
	styles *styles.Styles
	config PanelFrameConfig
	width  int
	height int
}

// NewPanelFrame creates a new panel frame.
func NewPanelFrame(s *styles.Styles) *PanelFrame {
	if s == nil {
		s = styles.DefaultStyles()
	}
	return &PanelFrame{
		styles: s,
	}
}

// SetSize updates the frame dimensions.
func (f *PanelFrame) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// SetConfig updates the frame configuration.
func (f *PanelFrame) SetConfig(config PanelFrameConfig) {
	f.config = config
}

// SetStyles updates the frame styles.
func (f *PanelFrame) SetStyles(s *styles.Styles) {
	f.styles = s
}

// ContentWidth returns the width available for content (excluding borders and scrollbar).
func (f *PanelFrame) ContentWidth() int {
	w := f.width - 2 // left and right border
	if f.config.ShowScrollbar {
		w-- // scrollbar
	}
	if w < 1 {
		return 1
	}
	return w
}

// ContentHeight returns the height available for content (excluding borders).
func (f *PanelFrame) ContentHeight() int {
	h := f.height - 2 // top and bottom border
	if h < 1 {
		return 1
	}
	return h
}

// RenderWithContent renders the frame around the given content lines.
// The content should be a slice of strings, one per line, each pre-padded to ContentWidth().
func (f *PanelFrame) RenderWithContent(contentLines []string) string {
	if f.styles == nil || f.height <= 0 || f.width <= 0 {
		return ""
	}

	// Determine border style based on focus
	borderStyle := f.styles.Border.BorderStyle(lipgloss.RoundedBorder())
	titleStyle := f.styles.PanelTitle
	if f.config.Focused {
		borderStyle = f.styles.FocusedBorder.BorderStyle(lipgloss.RoundedBorder())
		titleStyle = f.styles.FocusedPanelTitle
	}

	contentHeight := f.ContentHeight()

	// Ensure we have exactly contentHeight lines
	lines := make([]string, contentHeight)
	contentWidth := f.ContentWidth()
	for i := range contentHeight {
		if i < len(contentLines) {
			lines[i] = f.padLine(contentLines[i], contentWidth)
		} else {
			lines[i] = strings.Repeat(" ", contentWidth)
		}
	}

	return f.buildPanel(lines, borderStyle, titleStyle, contentHeight)
}

// buildPanel constructs the final panel with borders, title, scrollbar, and footer.
func (f *PanelFrame) buildPanel(contentLines []string, borderStyle, titleStyle lipgloss.Style, contentHeight int) string {
	border, _, _, _, _ := borderStyle.GetBorder()

	// Get border characters
	topLeft := border.TopLeft
	topRight := border.TopRight
	bottomLeft := border.BottomLeft
	bottomRight := border.BottomRight
	horizontal := border.Top
	vertical := border.Left

	if topLeft == "" {
		topLeft = "╭"
	}
	if topRight == "" {
		topRight = "╮"
	}
	if bottomLeft == "" {
		bottomLeft = "╰"
	}
	if bottomRight == "" {
		bottomRight = "╯"
	}
	if horizontal == "" {
		horizontal = "─"
	}
	if vertical == "" {
		vertical = consts.VerticalBar
	}

	// Build title line
	title := f.buildTitle(titleStyle)
	titleLine := f.buildTitleLine(topLeft, topRight, horizontal, title, borderStyle)

	// Build content lines with scrollbar
	outputLines := make([]string, 0, len(contentLines)+2)
	outputLines = append(outputLines, titleLine)

	scrollbarChars := f.calculateScrollbar(contentHeight)

	for i, line := range contentLines {
		var scrollbar string
		if f.config.ShowScrollbar {
			scrollbar = borderLine(borderStyle).Render(scrollbarChars[i])
		}
		lineContent := borderLine(borderStyle).Render(vertical) + line + scrollbar + borderLine(borderStyle).Render(vertical)
		outputLines = append(outputLines, lineContent)
	}

	// Build footer line
	footer := f.buildFooter()
	footerLine := f.buildFooterLine(bottomLeft, bottomRight, horizontal, footer, borderStyle)
	outputLines = append(outputLines, footerLine)

	return strings.Join(outputLines, "\n")
}

// buildTitle builds the title string with panel ID and tabs.
func (f *PanelFrame) buildTitle(titleStyle lipgloss.Style) string {
	if len(f.config.Tabs) == 0 {
		return titleStyle.Render(f.config.PanelID)
	}

	if len(f.config.Tabs) == 1 {
		return titleStyle.Render(f.config.PanelID + " " + f.config.Tabs[0])
	}

	// Multiple tabs: [2] ActiveTab - InactiveTab
	// Active tab gets the title color (blue when focused)
	// Inactive tabs are white (not dimmed)
	var tabParts []string
	for i, tab := range f.config.Tabs {
		if i == f.config.ActiveTab {
			// Active tab uses the title style (blue when focused)
			tabParts = append(tabParts, titleStyle.Render(tab))
		} else {
			// Inactive tabs are white (normal text, not dimmed)
			tabParts = append(tabParts, tab)
		}
	}
	return titleStyle.Render(f.config.PanelID) + " " + strings.Join(tabParts, " - ")
}

// buildTitleLine builds the top border line with title.
func (f *PanelFrame) buildTitleLine(topLeft, topRight, horizontal, title string, borderStyle lipgloss.Style) string {
	titleWidth := lipgloss.Width(title)
	availableWidth := f.width - 2 // minus corners

	if titleWidth > availableWidth {
		// Title too wide - just render border
		return borderLine(borderStyle).Render(topLeft + strings.Repeat(horizontal, availableWidth) + topRight)
	}

	paddingNeeded := availableWidth - titleWidth
	return borderLine(borderStyle).Render(topLeft) + title + borderLine(borderStyle).Render(strings.Repeat(horizontal, paddingNeeded)+topRight)
}

// buildFooter builds the footer string.
func (f *PanelFrame) buildFooter() string {
	if f.config.FooterText == "" {
		return ""
	}
	return f.styles.Dimmed.Render(f.config.FooterText)
}

// buildFooterLine builds the bottom border line with footer.
func (f *PanelFrame) buildFooterLine(bottomLeft, bottomRight, horizontal, footer string, borderStyle lipgloss.Style) string {
	footerWidth := lipgloss.Width(footer)
	availableWidth := f.width - 2 // minus corners

	if footer == "" || footerWidth > availableWidth {
		// No footer or too wide - just render border
		return borderLine(borderStyle).Render(bottomLeft + strings.Repeat(horizontal, availableWidth) + bottomRight)
	}

	leftPadding := availableWidth - footerWidth
	return borderLine(borderStyle).Render(bottomLeft+strings.Repeat(horizontal, leftPadding)) + footer + borderLine(borderStyle).Render(bottomRight)
}

// calculateScrollbar returns the scrollbar characters for each content line.
func (f *PanelFrame) calculateScrollbar(height int) []string {
	chars := make([]string, height)

	if !f.config.ShowScrollbar {
		for i := range chars {
			chars[i] = "│"
		}
		return chars
	}

	// Calculate thumb size (minimum 1 line)
	thumbSize := int(f.config.ThumbSize * float64(height))
	thumbSize = max(1, min(thumbSize, height))

	// Calculate thumb position
	thumbRange := height - thumbSize
	thumbPos := int(f.config.ScrollPos * float64(thumbRange))
	thumbPos = max(0, min(thumbPos, thumbRange))

	// Fill scrollbar characters
	for i := range chars {
		if i >= thumbPos && i < thumbPos+thumbSize {
			chars[i] = "▐" // Scrollbar thumb
		} else {
			chars[i] = "│" // Scrollbar track (same as border)
		}
	}

	return chars
}

// padLine pads or truncates a line to the given width.
func (f *PanelFrame) padLine(line string, width int) string {
	return PadLine(line, width)
}

// FormatItemCount formats an item count string like " 7 of 29 ".
func FormatItemCount(current, total int) string {
	if total == 0 {
		return ""
	}
	return " " + intToString(current) + " of " + intToString(total) + " "
}

// CalculateScrollParams calculates scrollbar parameters from list state.
// Returns scrollPos (0.0-1.0) and thumbSize (0.0-1.0).
func CalculateScrollParams(scrollOffset, visibleHeight, totalItems int) (scrollPos, thumbSize float64) {
	if totalItems <= visibleHeight || visibleHeight <= 0 {
		return 0, 1.0
	}

	// Thumb size as fraction of track
	thumbSize = float64(visibleHeight) / float64(totalItems)
	if thumbSize > 1.0 {
		thumbSize = 1.0
	}

	// Scroll position as fraction
	scrollRange := totalItems - visibleHeight
	if scrollRange > 0 {
		scrollPos = float64(scrollOffset) / float64(scrollRange)
	}
	if scrollPos > 1.0 {
		scrollPos = 1.0
	}

	return scrollPos, thumbSize
}
