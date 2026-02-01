package views

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/components"
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
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		colorized := v.colorizeLine(line)
		// Skip empty lines generated from separator handling.
		if colorized != "" || strings.TrimSpace(line) == "" {
			result = append(result, colorized)
		}
	}

	return strings.Join(result, "\n")
}

// colorizeLine applies color to a single line based on its content.
// Uses terraform's actual output colors - only the prefix symbols get colored.
func (v *HistoryView) colorizeLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Section separator lines are now handled by RenderSectionHeader, skip them.
	if isSeparatorLine(trimmed) {
		return ""
	}

	// Section titles (Details, Plan Output, Apply Output) - render like diff viewer headers.
	if isSectionTitle(trimmed) {
		return components.RenderSectionHeader(trimmed, v.width, v.styles.DiffChange, v.styles.Theme.BorderColor)
	}

	// Metadata key-value lines (Status:, Time:, Environment:, Directory:).
	if isMetadataLine(trimmed) {
		return v.colorizeMetadataLine(line, trimmed)
	}

	// Find leading whitespace to preserve indentation.
	leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	// Diff lines - color only the prefix symbol, not the whole line.
	switch {
	case strings.HasPrefix(trimmed, "-/+"):
		return leadingSpace + v.styles.DiffChange.Render("-/+") + trimmed[3:]
	case strings.HasPrefix(trimmed, "+"):
		return leadingSpace + v.styles.DiffAdd.Render("+") + trimmed[1:]
	case strings.HasPrefix(trimmed, "-"):
		rest := trimmed[1:]
		// Color "-> null" suffix if present.
		if strings.HasSuffix(rest, "-> null") {
			rest = rest[:len(rest)-7] + v.styles.Dimmed.Render("-> null")
		}
		return leadingSpace + v.styles.DiffRemove.Render("-") + rest
	case strings.HasPrefix(trimmed, "~"):
		return leadingSpace + v.styles.DiffChange.Render("~") + trimmed[1:]
	}

	// Everything else stays default color.
	return line
}

// isMetadataLine checks if a line is a metadata key-value line.
func isMetadataLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "Status:") ||
		strings.HasPrefix(trimmed, "Time:") ||
		strings.HasPrefix(trimmed, "Environment:") ||
		strings.HasPrefix(trimmed, "Directory:")
}

// colorizeMetadataLine applies styling to metadata key-value lines.
func (v *HistoryView) colorizeMetadataLine(line, trimmed string) string {
	leadingSpace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	// Find the colon position to split key and value.
	colonIdx := strings.Index(trimmed, ":")
	if colonIdx == -1 {
		return line
	}

	key := trimmed[:colonIdx+1] // includes colon
	value := trimmed[colonIdx+1:]

	// Style: dimmed key, normal value, colored status icon.
	styledKey := v.styles.Dimmed.Render(key)

	// Special handling for status line - color the icon.
	if strings.HasPrefix(trimmed, "Status:") {
		value = v.colorizeStatusValue(value)
	}

	return leadingSpace + styledKey + value
}

// colorizeStatusValue applies color to the status value based on its content.
func (v *HistoryView) colorizeStatusValue(value string) string {
	value = strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(value, "●") || strings.Contains(value, "Success"):
		return " " + v.styles.DiffAdd.Render(value)
	case strings.HasPrefix(value, "✗") || strings.Contains(value, "Failed"):
		return " " + v.styles.DiffRemove.Render(value)
	case strings.HasPrefix(value, "○") || strings.Contains(value, "Canceled"):
		return " " + v.styles.DiffChange.Render(value)
	default:
		return " " + value
	}
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

// isSeparatorLine checks if a line consists only of horizontal line characters.
func isSeparatorLine(trimmed string) bool {
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if r != '─' {
			return false
		}
	}
	return true
}

// isSectionTitle checks if a line is a section title (Details, Plan Output, Apply Output).
func isSectionTitle(trimmed string) bool {
	return trimmed == "Details" || trimmed == "Plan Output" || trimmed == "Apply Output"
}
