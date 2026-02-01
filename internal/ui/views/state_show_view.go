package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// StateShowView renders terraform state show output.
type StateShowView struct {
	viewport viewport.Model
	styles   *styles.Styles
	address  string
	width    int
}

// NewStateShowView creates a new state show view.
func NewStateShowView(s *styles.Styles) *StateShowView {
	return &StateShowView{
		viewport: viewport.New(0, 0),
		styles:   s,
	}
}

// SetStyles updates the component styles.
func (v *StateShowView) SetStyles(s *styles.Styles) {
	v.styles = s
}

// SetSize updates the layout size.
func (v *StateShowView) SetSize(width, height int) {
	v.width = width
	headerHeight := 1
	footerHeight := 1
	bodyHeight := height - headerHeight - footerHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	v.viewport.Width = width
	v.viewport.Height = bodyHeight
}

// SetAddress sets the resource address being shown.
func (v *StateShowView) SetAddress(address string) {
	v.address = address
}

// SetContent sets the state output text.
func (v *StateShowView) SetContent(content string) {
	v.viewport.SetContent(strings.TrimRight(content, "\n"))
	v.viewport.GotoTop()
}

// Update handles viewport messages.
func (v *StateShowView) Update(msg tea.Msg) (*StateShowView, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the state show output.
func (v *StateShowView) View() string {
	if v.styles == nil {
		return ""
	}
	title := "State: " + v.address
	if v.address == "" {
		title = "State Details"
	}
	header := v.styles.Title.Width(v.width).Render(title)
	body := v.viewport.View()
	if v.width > 0 {
		body = lipgloss.NewStyle().Width(v.width).Height(v.viewport.Height).Render(body)
	}
	footer := v.styles.StatusBar.Width(v.width).Render("↑↓/jk: scroll | esc: back to list | q: quit")
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// ViewContent renders just the viewport content (for embedding in MainArea).
func (v *StateShowView) ViewContent() string {
	if v.styles == nil {
		return ""
	}
	body := v.viewport.View()
	if v.width > 0 {
		body = lipgloss.NewStyle().Width(v.width).Height(v.viewport.Height).Render(body)
	}
	return body
}

// GetAddress returns the current resource address.
func (v *StateShowView) GetAddress() string {
	return v.address
}
