package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

// DiagnosticsPanel renders diagnostics alongside command output.
type DiagnosticsPanel struct {
	viewport    viewport.Model
	diagnostics []terraform.Diagnostic
	logText     string
	parsedText  string
	showRaw     bool
	styles      *styles.Styles
	width       int
	height      int
}

// NewDiagnosticsPanel creates a diagnostics panel.
func NewDiagnosticsPanel(styles *styles.Styles) *DiagnosticsPanel {
	return &DiagnosticsPanel{
		viewport: viewport.New(0, 0),
		styles:   styles,
	}
}

// SetSize updates the panel size.
func (d *DiagnosticsPanel) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.viewport.Width = width
	d.viewport.Height = height
	d.updateViewport()
}

// SetDiagnostics replaces the diagnostics list.
func (d *DiagnosticsPanel) SetDiagnostics(items []terraform.Diagnostic) {
	d.diagnostics = append([]terraform.Diagnostic{}, items...)
	d.updateViewport()
}

// SetLogText sets raw log output to show when no diagnostics are available.
func (d *DiagnosticsPanel) SetLogText(text string) {
	d.logText = strings.TrimRight(text, "\n")
	d.updateViewport()
	d.viewport.GotoBottom()
}

// SetParsedText sets the parsed summary text for display.
func (d *DiagnosticsPanel) SetParsedText(text string) {
	d.parsedText = strings.TrimRight(text, "\n")
	d.updateViewport()
	d.viewport.GotoBottom()
}

// SetShowRaw toggles between raw logs and parsed summary when no diagnostics exist.
func (d *DiagnosticsPanel) SetShowRaw(show bool) {
	d.showRaw = show
	d.updateViewport()
}

// View renders the diagnostics panel.
func (d *DiagnosticsPanel) View() string {
	if d == nil || d.styles == nil {
		return ""
	}
	content := d.viewport.View()
	if d.width > 0 && d.height > 0 {
		return lipgloss.NewStyle().Width(d.width).Height(d.height).Render(content)
	}
	return content
}

// Update forwards messages to the viewport for scrolling.
func (d *DiagnosticsPanel) Update(msg tea.Msg) (*DiagnosticsPanel, tea.Cmd) {
	if d == nil {
		return nil, nil
	}
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d *DiagnosticsPanel) updateViewport() {
	if d == nil || d.styles == nil {
		return
	}
	if len(d.diagnostics) == 0 {
		content := d.parsedText
		title := "Parsed"
		if d.showRaw {
			content = d.logText
			title = "Logs"
		}
		if strings.TrimSpace(content) == "" {
			if strings.TrimSpace(d.logText) != "" {
				content = d.logText
				title = "Logs"
			}
		}
		if strings.TrimSpace(content) == "" {
			d.viewport.SetContent(d.styles.Dimmed.Render("No diagnostics reported."))
			return
		}
		logs := content
		if d.width > 0 {
			logs = wrapText(logs, d.width)
		}
		lines := []string{
			d.styles.Title.Render(title),
			logs,
		}
		d.viewport.SetContent(strings.TrimRight(strings.Join(lines, "\n"), "\n"))
		return
	}

	var errors []terraform.Diagnostic
	var warnings []terraform.Diagnostic
	for _, diag := range d.diagnostics {
		switch strings.ToLower(diag.Severity) {
		case "error":
			errors = append(errors, diag)
		default:
			warnings = append(warnings, diag)
		}
	}

	var lines []string
	lines = append(lines, d.styles.Title.Render("Diagnostics"))

	if len(errors) > 0 {
		lines = append(lines, d.styles.Delete.Render("Errors"))
		for _, diag := range errors {
			lines = append(lines, formatDiagnostic(diag))
		}
	}
	if len(warnings) > 0 {
		lines = append(lines, d.styles.Update.Render("Warnings"))
		for _, diag := range warnings {
			lines = append(lines, formatDiagnostic(diag))
		}
	}

	d.viewport.SetContent(strings.TrimRight(strings.Join(lines, "\n"), "\n"))
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	wrapped := make([]string, 0, 32)
	wrapStyle := lipgloss.NewStyle().Width(width)
	for _, line := range strings.Split(text, "\n") {
		wrapped = append(wrapped, strings.TrimRight(wrapStyle.Render(line), " "))
	}
	return strings.TrimRight(strings.Join(wrapped, "\n"), "\n")
}

func formatDiagnostic(diag terraform.Diagnostic) string {
	var parts []string
	if diag.Summary != "" {
		parts = append(parts, diag.Summary)
	}
	if diag.Detail != "" {
		parts = append(parts, diag.Detail)
	}
	if diag.Address != "" {
		parts = append(parts, "address: "+diag.Address)
	}
	if diag.Range != nil && diag.Range.Filename != "" {
		location := diag.Range.Filename
		if diag.Range.Start != nil && diag.Range.Start.Line > 0 {
			location = fmt.Sprintf("%s:%d:%d", location, diag.Range.Start.Line, diag.Range.Start.Column)
		}
		parts = append(parts, "location: "+location)
	}
	if len(parts) == 0 {
		return "- (no details)"
	}
	return "- " + strings.Join(parts, " | ")
}
