package views

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
)

// ansiPattern matches ANSI escape codes for stripping colors.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// HistoryView renders stored apply output.
type HistoryView struct {
	viewport viewport.Model
	styles   *styles.Styles
	title    string
	width    int
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
	headerHeight := 1
	footerHeight := 1
	bodyHeight := max(1, height-headerHeight-footerHeight)
	v.viewport.Width = width
	v.viewport.Height = bodyHeight
}

// SetTitle updates the header title.
func (v *HistoryView) SetTitle(title string) {
	v.title = title
}

// SetContent sets the history output text.
func (v *HistoryView) SetContent(content string) {
	// Strip ANSI escape codes for clean display.
	content = ansiPattern.ReplaceAllString(content, "")
	// Apply syntax highlighting (terraform output is already properly indented).
	content = v.colorizeOutput(content)
	v.viewport.SetContent(strings.TrimRight(content, "\n"))
	v.viewport.GotoTop()
}

// colorizeOutput applies syntax highlighting to terraform plan output.
func (v *HistoryView) colorizeOutput(content string) string {
	if v.styles == nil {
		return content
	}

	lines := strings.Split(content, "\n")
	result := make([]string, len(lines))

	for i, line := range lines {
		result[i] = v.colorizeLine(line)
	}

	return strings.Join(result, "\n")
}

// colorizeLine applies color to a single line based on its content.
// Uses terraform's actual output colors - only the prefix symbols get colored.
func (v *HistoryView) colorizeLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Section separators (from smart formatter).
	if strings.Contains(trimmed, "────") {
		return styles.TfDimmed.Render(line)
	}

	// Find leading whitespace to preserve indentation.
	leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	// Diff lines - color only the prefix symbol, not the whole line.
	switch {
	case strings.HasPrefix(trimmed, "-/+"):
		return leadingSpace + styles.TfDiffChange.Render("-/+") + trimmed[3:]
	case strings.HasPrefix(trimmed, "+"):
		return leadingSpace + styles.TfDiffAdd.Render("+") + trimmed[1:]
	case strings.HasPrefix(trimmed, "-"):
		rest := trimmed[1:]
		// Color "-> null" suffix if present.
		if strings.HasSuffix(rest, "-> null") {
			rest = rest[:len(rest)-7] + styles.TfDimmed.Render("-> null")
		}
		return leadingSpace + styles.TfDiffRemove.Render("-") + rest
	case strings.HasPrefix(trimmed, "~"):
		return leadingSpace + styles.TfDiffChange.Render("~") + trimmed[1:]
	}

	// Everything else stays default color.
	return line
}

// SetStyles updates the view styles.
func (v *HistoryView) SetStyles(s *styles.Styles) {
	v.styles = s
}

// Update handles viewport messages.
func (v *HistoryView) Update(msg tea.Msg) (*HistoryView, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the history output with full-screen header/footer.
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

// ViewContent renders just the viewport content (for embedding in MainArea).
func (v *HistoryView) ViewContent() string {
	if v.styles == nil {
		return ""
	}
	body := v.viewport.View()
	if v.width > 0 {
		body = lipgloss.NewStyle().Width(v.width).Height(v.viewport.Height).Render(body)
	}
	return body
}

// GetTitle returns the current title.
func (v *HistoryView) GetTitle() string {
	return v.title
}
