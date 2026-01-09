package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/tftui/internal/styles"
	"github.com/ushiradineth/tftui/internal/terraform"
)

// DiagnosticsPanel renders diagnostics from streaming JSON output.
type DiagnosticsPanel struct {
	viewport    viewport.Model
	diagnostics []terraform.Diagnostic
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
		return d, nil
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
		d.viewport.SetContent(d.styles.Dimmed.Render("No diagnostics reported."))
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
