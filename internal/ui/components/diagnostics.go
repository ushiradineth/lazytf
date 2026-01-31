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
	Label     string // Human-readable action (e.g., "Planned", "Applied")
	Command   string // The actual terraform command
	Output    string // Command output (optional)
	Timestamp string // For reference, not displayed
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

// SetStyles updates the panel styles.
func (d *DiagnosticsPanel) SetStyles(s *styles.Styles) {
	d.styles = s
}

// AppendSessionLog adds a new command log entry to the session history.
func (d *DiagnosticsPanel) AppendSessionLog(label, command, output string) {
	entry := SessionLogEntry{
		Label:     label,
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

	sections := buildSessionSections(d.styles, d.sessionLogs, d.width)
	sections = append(sections, buildDiagnosticSections(d.styles, d.diagnostics)...)
	sections = append(sections, buildFallbackSection(d.styles, d.logText, d.width)...)

	d.viewport.SetContent(strings.TrimRight(strings.Join(sections, "\n"), "\n"))
}

func buildSessionSections(styles *styles.Styles, logs []SessionLogEntry, width int) []string {
	if len(logs) == 0 {
		return nil
	}
	sections := make([]string, 0, len(logs)*3)
	for _, entry := range logs {
		// Render label in highlight style
		label := styles.Highlight.Render(entry.Label)
		sections = append(sections, label)
		// Render command indented with dimmed style
		command := styles.Dimmed.Render("  " + entry.Command)
		sections = append(sections, command)
		// Render output if present
		if strings.TrimSpace(entry.Output) != "" {
			output := entry.Output
			if width > 0 {
				output = wrapText(output, width)
			}
			sections = append(sections, output)
		}
		sections = append(sections, "")
	}
	return sections
}

func buildDiagnosticSections(styles *styles.Styles, diagnostics []terraform.Diagnostic) []string {
	if len(diagnostics) == 0 {
		return nil
	}
	errors, warnings := splitDiagnostics(diagnostics)
	sections := []string{styles.Title.Render("Diagnostics")}
	if len(errors) > 0 {
		sections = append(sections, styles.Delete.Render("Errors"))
		for _, diag := range errors {
			sections = append(sections, formatDiagnostic(diag))
		}
	}
	if len(warnings) > 0 {
		sections = append(sections, styles.Update.Render("Warnings"))
		for _, diag := range warnings {
			sections = append(sections, formatDiagnostic(diag))
		}
	}
	return sections
}

func splitDiagnostics(diagnostics []terraform.Diagnostic) ([]terraform.Diagnostic, []terraform.Diagnostic) {
	var errors []terraform.Diagnostic
	var warnings []terraform.Diagnostic
	for _, diag := range diagnostics {
		switch strings.ToLower(diag.Severity) {
		case "error":
			errors = append(errors, diag)
		default:
			warnings = append(warnings, diag)
		}
	}
	return errors, warnings
}

func buildFallbackSection(styles *styles.Styles, logText string, width int) []string {
	if strings.TrimSpace(logText) == "" {
		return []string{styles.Dimmed.Render("No logs available.")}
	}
	content := logText
	if width > 0 {
		content = wrapText(content, width)
	}
	return []string{content}
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
