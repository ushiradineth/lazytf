package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestHistoryViewRendersContent(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(60, 10)
	view.SetTitle("Apply details")
	view.SetContent("line one\nline two")

	out := view.View()
	if !strings.Contains(out, "Apply details") {
		t.Fatalf("expected title in output")
	}
	if !strings.Contains(out, "line one") || !strings.Contains(out, "line two") {
		t.Fatalf("expected content in output")
	}
}

func TestHistoryViewUpdateHandlesKeys(_ *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(40, 6)
	view.SetTitle("Title")
	view.SetContent("line")

	_, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
}

func TestHistoryViewSetStyles(t *testing.T) {
	view := NewHistoryView(styles.DefaultStyles())
	view.SetSize(60, 20)

	newStyles := styles.DefaultStyles()
	view.SetStyles(newStyles)

	if view.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestHistoryViewViewContent(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 20)
	view.SetContent("line one\nline two\nline three")

	content := view.ViewContent()
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestHistoryViewViewContentNilStyles(t *testing.T) {
	view := &HistoryView{}
	content := view.ViewContent()
	if content != "" {
		t.Error("expected empty content for nil styles")
	}
}

func TestHistoryViewViewContentZeroWidth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(0, 20)
	view.SetContent("test content")

	content := view.ViewContent()
	// Should still return something even with zero width
	_ = content
}

func TestHistoryViewGetTitle(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)

	// Initially empty
	if view.GetTitle() != "" {
		t.Error("expected empty title initially")
	}

	view.SetTitle("Apply Output")
	if view.GetTitle() != "Apply Output" {
		t.Errorf("expected 'Apply Output', got %q", view.GetTitle())
	}
}

func TestHistoryViewViewWithStatusContent(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)
	view.SetTitle("Operation Details")

	// Content with status metadata that exercises colorizeMetadataLine
	content := `Status:       Success
Started:      2024-01-15
Duration:     5m30s
Command:      terraform apply

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.`

	view.SetContent(content)
	out := view.View()

	if !strings.Contains(out, "Operation Details") {
		t.Error("expected title in output")
	}
}

func TestHistoryViewViewWithFailedStatus(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)
	view.SetTitle("Failed Operation")

	// Content with failed status
	content := `Status:       Failed
Started:      2024-01-15
Duration:     2m15s
Command:      terraform apply

Error: Error creating resource`

	view.SetContent(content)
	out := view.View()

	if !strings.Contains(out, "Failed Operation") {
		t.Error("expected title in output")
	}
}

func TestHistoryViewViewWithRunningStatus(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewHistoryView(s)
	view.SetSize(80, 30)
	view.SetTitle("Running Operation")

	// Content with running status
	content := `Status:       Running
Started:      2024-01-15
Duration:     0s
Command:      terraform plan

Planning...`

	view.SetContent(content)
	out := view.View()

	if !strings.Contains(out, "Running Operation") {
		t.Error("expected title in output")
	}
}
