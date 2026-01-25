package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderPanelTitleLine renders a panel border line with an embedded title
func RenderPanelTitleLine(width int, borderStyle lipgloss.Style, title string) (string, bool) {
	if width <= 0 {
		return "", false
	}
	border, _, _, _, _ := borderStyle.GetBorder()
	topLeft := border.TopLeft
	if topLeft == "" {
		topLeft = "┌"
	}
	topRight := border.TopRight
	if topRight == "" {
		topRight = "┐"
	}
	top := border.Top
	if top == "" {
		top = "─"
	}

	titleLen := lipgloss.Width(title)
	availableWidth := width - 2 // minus corners
	if titleLen > availableWidth {
		// Title too wide - truncate if possible
		// For now, skip title and just render border
		return borderLine(borderStyle).Render(topLeft + strings.Repeat(top, availableWidth) + topRight), true
	}

	paddingNeeded := availableWidth - titleLen
	return borderLine(borderStyle).Render(topLeft) + title + borderLine(borderStyle).Render(strings.Repeat(top, paddingNeeded)+topRight), true
}

func borderLine(borderStyle lipgloss.Style) lipgloss.Style {
	borderColor := borderStyle.GetBorderTopForeground()
	return lipgloss.NewStyle().Foreground(borderColor)
}
