package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestDiagnosticsPanelEmptyAndParsed(t *testing.T) {
	panel := NewDiagnosticsPanel(styles.DefaultStyles())
	panel.SetSize(40, 5)

	if out := panel.View(); out == "" {
		t.Fatalf("expected view to render")
	}

	panel.SetParsedText("parsed")
	out := panel.View()
	if !strings.Contains(out, "parsed") {
		t.Fatalf("expected parsed content")
	}

	panel.SetLogText("raw")
	panel.SetShowRaw(true)
	out = panel.View()
	if !strings.Contains(out, "raw") {
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
