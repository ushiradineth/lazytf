package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/tftui/internal/styles"
)

// HistoryView renders stored apply output.
type HistoryView struct {
	viewport viewport.Model
	styles   *styles.Styles
	title    string
	width    int
	height   int
}

// NewHistoryView creates a new history view.
func NewHistoryView(s *styles.Styles) *HistoryView {
	return &HistoryView{
		viewport: viewport.New(0, 0),
		styles:   s,
	}
}

// SetSize updates the layout size.
func (v *HistoryView) SetSize(width, height int) {
	v.width = width
	v.height = height
	headerHeight := 1
	footerHeight := 1
	bodyHeight := height - headerHeight - footerHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	v.viewport.Width = width
	v.viewport.Height = bodyHeight
}

// SetTitle updates the header title.
func (v *HistoryView) SetTitle(title string) {
	v.title = title
}

// SetContent sets the history output text.
func (v *HistoryView) SetContent(content string) {
	v.viewport.SetContent(strings.TrimRight(content, "\n"))
	v.viewport.GotoTop()
}

// Update handles viewport messages.
func (v *HistoryView) Update(msg tea.Msg) (*HistoryView, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the history output.
func (v *HistoryView) View() string {
	if v.styles == nil {
		return ""
	}
	title := v.title
	if title == "" {
		title = "Apply details"
	}
	header := v.styles.Title.Width(v.width).Render(title)
	body := v.viewport.View()
	if v.width > 0 {
		body = lipgloss.NewStyle().Width(v.width).Height(v.viewport.Height).Render(body)
	}
	footer := v.styles.StatusBar.Width(v.width).Render("esc: back | q: quit")
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
