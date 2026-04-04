package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/styles"
)

// ASCII art for lazytf logo.
const lazytfLogo = `
  _                     _    __
 | |                   | |  / _|
 | |  __ _  ____ _   _ | |_| |_
 | | / _` + "`" + ` ||_  /| | | || __|  _|
 | || (_| | / / | |_| || |_| |
 |_| \__,_|/___| \__, | \__|_|
                  __/ |
                 |___/
`

// AboutView renders the about/info screen.
type AboutView struct {
	viewport viewport.Model
	styles   *styles.Styles
	width    int
	height   int
}

// NewAboutView creates a new about view.
func NewAboutView(s *styles.Styles) *AboutView {
	return &AboutView{
		viewport: viewport.New(0, 0),
		styles:   s,
	}
}

// SetSize updates the layout size.
func (v *AboutView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height
	v.updateContent()
}

// SetStyles updates the view styles.
func (v *AboutView) SetStyles(s *styles.Styles) {
	v.styles = s
	v.updateContent()
}

// Update handles viewport messages.
func (v *AboutView) Update(msg tea.Msg) (*AboutView, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// ViewContent renders the about content (for embedding in MainArea).
func (v *AboutView) ViewContent() string {
	if v.styles == nil {
		return ""
	}
	body := v.viewport.View()
	if v.width > 0 {
		body = lipgloss.NewStyle().Width(v.width).Height(v.viewport.Height).Render(body)
	}
	return body
}

// updateContent generates and sets the about content.
func (v *AboutView) updateContent() {
	if v.styles == nil {
		return
	}

	var sb strings.Builder

	// Logo with styling
	logoStyle := lipgloss.NewStyle().
		Foreground(v.styles.Theme.HighlightColor).
		Bold(true)
	sb.WriteString(logoStyle.Render(lazytfLogo))
	sb.WriteString("\n\n")

	textStyle := v.styles.Bold
	labelStyle := v.styles.Bold

	// Links section
	links := []struct {
		label string
		value string
	}{
		{"Keybindings", "Press ? for keybinds"},
		{"GitHub", "https://github.com/ushiradineth/lazytf"},
		{"Raise an Issue", "https://github.com/ushiradineth/lazytf/issues"},
		{"Release Notes", "https://github.com/ushiradineth/lazytf/releases"},
	}

	for _, link := range links {
		sb.WriteString(labelStyle.Render(link.label+":") + " ")
		sb.WriteString(textStyle.Render(link.value))
		sb.WriteString("\n\n")
	}

	// Version
	sb.WriteString(labelStyle.Render("Version:") + " ")
	sb.WriteString(v.styles.DiffAdd.Render(consts.Version))
	sb.WriteString("\n")

	v.viewport.SetContent(sb.String())
}
