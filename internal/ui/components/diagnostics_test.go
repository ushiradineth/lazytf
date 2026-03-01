package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestDiagnosticsPanelEmptyAndSessionLogs(t *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())
	panel.SetSize(40, 5)

	if out := panel.View(); out == "" {
		t.Fatalf("expected view to render")
	}

	// Session logs take priority when present
	panel.AppendSessionLog("Planned", "terraform plan", "plan output")
	out := panel.View()
	if !strings.Contains(out, "terraform plan") {
		t.Fatalf("expected session log command in output")
	}
	if !strings.Contains(out, "Planned") {
		t.Fatalf("expected session log label in output")
	}
	if !strings.Contains(out, "plan output") {
		t.Fatalf("expected session log output in output")
	}

	// Clear session logs and use raw log text
	panel.ClearSessionLogs()
	panel.SetLogText("raw log content")
	out = panel.View()
	if !strings.Contains(out, "raw log content") {
		t.Fatalf("expected raw log content")
	}
}

func TestDiagnosticsPanelDiagnosticsList(t *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())
	panel.SetSize(40, 8)
	panel.SetDiagnostics([]terraform.Diagnostic{
		{Severity: "error", Summary: "bad", Detail: "detail"},
		{Severity: "warning", Summary: "warn"},
	})

	out := panel.View()
	if !strings.Contains(out, "Diagnostics") {
		t.Fatalf("expected diagnostics header")
	}
	if !strings.Contains(out, "bad") || !strings.Contains(out, "warn") {
		t.Fatalf("expected diagnostic content")
	}
}

func TestFormatDiagnosticAndWrapText(t *testing.T) {
	diag := terraform.Diagnostic{
		Summary: "summary",
		Detail:  "detail",
		Address: "aws_instance.example",
		Range: &terraform.DiagnosticRange{
			Filename: "main.tf",
			Start:    &terraform.LinePosition{Line: 3, Column: 2},
		},
	}
	line := formatDiagnostic(diag)
	if !strings.Contains(line, "summary") || !strings.Contains(line, "location: main.tf:3:2") {
		t.Fatalf("unexpected diagnostic line: %q", line)
	}

	wrapped := wrapText("line", 2)
	if !strings.Contains(wrapped, "li") {
		t.Fatalf("expected wrapped text")
	}
}

func TestDiagnosticsPanelUpdate(_ *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())
	panel.SetSize(40, 5)
	_, _ = panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
}

func TestDiagnosticsPanelUpdateNil(t *testing.T) {
	var panel *DiagnosticsPanel
	result, cmd := panel.Update(nil)
	if result != nil {
		t.Error("Expected nil result for nil panel")
	}
	if cmd != nil {
		t.Error("Expected nil cmd for nil panel")
	}
}

func TestDiagnosticsPanelViewNil(t *testing.T) {
	var panel *DiagnosticsPanel
	if panel.View() != "" {
		t.Error("Expected empty string for nil panel")
	}

	panel = &DiagnosticsPanel{}
	if panel.View() != "" {
		t.Error("Expected empty string for nil styles")
	}
}

func TestDiagnosticsPanelViewNoSize(_ *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())
	panel.SetLogText("some log content")
	// Without SetSize, width/height are 0 so we get viewport content directly
	out := panel.View()
	// The view returns viewport content even without size
	_ = out // Just verify it doesn't panic
}

func TestFormatDiagnosticNoDetails(t *testing.T) {
	diag := terraform.Diagnostic{}
	line := formatDiagnostic(diag)
	if !strings.Contains(line, "no details") {
		t.Errorf("Expected 'no details' for empty diagnostic, got %q", line)
	}
}

func TestFormatDiagnosticWithRange(t *testing.T) {
	diag := terraform.Diagnostic{
		Summary: "test",
		Range: &terraform.DiagnosticRange{
			Filename: "main.tf",
		},
	}
	line := formatDiagnostic(diag)
	if !strings.Contains(line, "main.tf") {
		t.Errorf("Expected filename in output, got %q", line)
	}
}

func TestWrapTextZeroWidth(t *testing.T) {
	result := wrapText("some text", 0)
	if result != "some text" {
		t.Errorf("Expected unchanged text for zero width, got %q", result)
	}

	result = wrapText("some text", -1)
	if result != "some text" {
		t.Errorf("Expected unchanged text for negative width, got %q", result)
	}
}

func TestDiagnosticsPanelSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewDiagnosticsPanel(s)

	newStyles := styles.DefaultStyles()
	panel.SetStyles(newStyles)

	if panel.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestDiagnosticsPanelExpandFillsContent(t *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())

	// Add enough content to require scrolling in a small panel
	var logs []string
	for i := 1; i <= 50; i++ {
		logs = append(logs, strings.Repeat("x", 40))
	}
	panel.SetLogText(strings.Join(logs, "\n"))

	// Start with a small size (like compact command log)
	panel.SetSize(80, 8)
	smallView := panel.View()
	smallLines := strings.Split(smallView, "\n")

	// Expand to a larger size (like focused command log)
	panel.SetSize(80, 40)
	expandedView := panel.View()
	expandedLines := strings.Split(expandedView, "\n")

	// Count non-empty lines with actual content (not just whitespace)
	countContentLines := func(lines []string) int {
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		return count
	}

	smallContentLines := countContentLines(smallLines)
	expandedContentLines := countContentLines(expandedLines)

	// The expanded view should show more content lines than the small view
	if expandedContentLines <= smallContentLines {
		t.Errorf("Expected expanded panel to show more content lines than small panel.\n"+
			"Small panel content lines: %d\n"+
			"Expanded panel content lines: %d\n"+
			"Small view:\n%s\n\nExpanded view:\n%s",
			smallContentLines, expandedContentLines, smallView, expandedView)
	}

	// The expanded view should have content filling most of the height
	// (allowing some margin for the viewport behavior)
	minExpectedContentLines := 30 // At least 30 lines of content in a 40-line panel
	if expandedContentLines < minExpectedContentLines {
		t.Errorf("Expected at least %d content lines in expanded panel, got %d.\n"+
			"Expanded view:\n%s",
			minExpectedContentLines, expandedContentLines, expandedView)
	}
}

func TestWrapTextPreservesAllCharacters(t *testing.T) {
	text := "terraform apply -compact-warnings /very/long/path/to/.lazytf/tmp/plan.tfplan"
	wrapped := wrapText(text, 16)
	reconstructed := strings.ReplaceAll(wrapped, "\n", "")
	if reconstructed != text {
		t.Fatalf("expected wrapped text to preserve all characters\nwant: %q\n got: %q", text, reconstructed)
	}
}

func TestDiagnosticsPanelSessionCommandWrapKeepsTail(t *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())
	panel.SetSize(28, 8)
	panel.AppendSessionLog("Applied", "terraform apply -compact-warnings /tmp/plan.tfplan", "ok")

	view := panel.View()
	if !strings.Contains(view, "plan.tfplan") {
		t.Fatalf("expected wrapped command tail to remain visible, got %q", view)
	}
}

func TestDiagnosticsPanelGetScrollInfo(t *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())
	panel.SetSize(30, 4)

	lines := make([]string, 0, 40)
	for range 40 {
		lines = append(lines, "line")
	}
	panel.SetLogText(strings.Join(lines, "\n"))

	_, _, show := panel.GetScrollInfo()
	if !show {
		t.Fatal("expected scrollbar to be shown for long content")
	}

	_, _ = panel.Update(tea.KeyMsg{Type: tea.KeyDown})
	scrollPos, thumbSize, show := panel.GetScrollInfo()
	if !show {
		t.Fatal("expected scrollbar to remain visible after scroll")
	}
	if scrollPos <= 0 {
		t.Fatalf("expected positive scroll position after moving down, got %f", scrollPos)
	}
	if thumbSize <= 0 || thumbSize >= 1 {
		t.Fatalf("expected thumb size between 0 and 1, got %f", thumbSize)
	}
}
