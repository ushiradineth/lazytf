package components

import (
	"fmt"
	"strings"
	"time"

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
	styles      *styles.Styles
	width       int
	height      int
	// Session log history - stores all command outputs for the session
	sessionLogs []SessionLogEntry
}

// SessionLogEntry represents a single command log entry in the session.
type SessionLogEntry struct {
	Command   string
	Output    string
	Timestamp string
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

// SetParsedText is a no-op kept for API compatibility.
// The parsed text was previously stored but never displayed.
func (d *DiagnosticsPanel) SetParsedText(_ string) {
	// Intentionally empty - parsed text is not used
}

// SetShowRaw is a no-op kept for API compatibility.
// The show raw flag was previously stored but never used.
func (d *DiagnosticsPanel) SetShowRaw(_ bool) {
	// Intentionally empty - show raw flag is not used
}

// AppendSessionLog adds a new command log entry to the session history.
func (d *DiagnosticsPanel) AppendSessionLog(command, output string) {
	entry := SessionLogEntry{
		Command:   command,
		Output:    output,
		Timestamp: time.Now().Format("15:04:05"),
	}
	d.sessionLogs = append(d.sessionLogs, entry)
	d.updateViewport()
	d.viewport.GotoBottom()
}

// ClearSessionLogs clears all session log entries.
func (d *DiagnosticsPanel) ClearSessionLogs() {
	d.sessionLogs = nil
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

	var sections []string

	// Always show session logs first (all historic commands)
	if len(d.sessionLogs) > 0 {
		for _, entry := range d.sessionLogs {
			header := d.styles.Highlight.Render(fmt.Sprintf("[%s] %s", entry.Timestamp, entry.Command))
			sections = append(sections, header)
			if strings.TrimSpace(entry.Output) != "" {
				output := entry.Output
				if d.width > 0 {
					output = wrapText(output, d.width)
				}
				sections = append(sections, output)
			}
			sections = append(sections, "") // Empty line separator
		}
	}

	// Show diagnostics if any
	if len(d.diagnostics) > 0 {
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

		sections = append(sections, d.styles.Title.Render("Diagnostics"))
		if len(errors) > 0 {
			sections = append(sections, d.styles.Delete.Render("Errors"))
			for _, diag := range errors {
				sections = append(sections, formatDiagnostic(diag))
			}
		}
		if len(warnings) > 0 {
			sections = append(sections, d.styles.Update.Render("Warnings"))
			for _, diag := range warnings {
				sections = append(sections, formatDiagnostic(diag))
			}
		}
	}

	// If no session logs and no diagnostics, show current log text
	if len(sections) == 0 {
		content := d.logText
		if strings.TrimSpace(content) == "" {
			d.viewport.SetContent(d.styles.Dimmed.Render("No logs available."))
			return
		}
		if d.width > 0 {
			content = wrapText(content, d.width)
		}
		sections = append(sections, content)
	}

	d.viewport.SetContent(strings.TrimRight(strings.Join(sections, "\n"), "\n"))
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
