package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/tftui/internal/styles"
)

func TestApplyViewStatusAndFooter(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewApplyView(s)
	view.SetSize(60, 10)
	view.SetTitle("Applying changes...")
	view.SetStatus(ApplySuccess)

	out := view.View()
	if !strings.Contains(out, "Apply complete") {
		t.Fatalf("expected success footer")
	}
}

func TestApplyViewCapsLines(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewApplyView(s)
	view.maxLines = 3
	view.SetSize(60, 10)

	view.AppendLine("one")
	view.AppendLine("two")
	view.AppendLine("three")
	view.AppendLine("four")

	if len(view.outputLines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(view.outputLines))
	}
	if view.outputLines[0] != "two" {
		t.Fatalf("expected oldest line to be trimmed, got %q", view.outputLines[0])
	}
}

func TestApplyViewAutoScrolls(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewApplyView(s)
	view.SetSize(40, 6)

	for i := 0; i < 20; i++ {
		view.AppendLine("line")
	}
	if view.viewport.YOffset == 0 {
		t.Fatalf("expected viewport to scroll, got offset %d", view.viewport.YOffset)
	}
}

func TestApplyViewSetStatusTextAndProgress(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewApplyView(s)
	view.SetSize(40, 6)
	view.SetStatusText("run", "ok", "bad")
	view.SetStatus(ApplyRunning)
	view.SetProgress("2/5")

	out := view.View()
	if !strings.Contains(out, "Progress: 2/5") {
		t.Fatalf("expected progress text")
	}
}

func TestApplyViewResetClearsOutput(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewApplyView(s)
	view.SetSize(40, 6)
	view.AppendLine("line")
	view.SetStatus(ApplySuccess)

	view.Reset()
	if len(view.outputLines) != 0 {
		t.Fatalf("expected output lines cleared")
	}
	if view.status != ApplyPending {
		t.Fatalf("expected status reset to pending")
	}
}

func TestApplyViewSetOutput(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewApplyView(s)
	view.SetSize(40, 6)
	view.SetOutput("one\ntwo")

	if len(view.outputLines) != 2 {
		t.Fatalf("expected output lines to be set")
	}
	out := view.View()
	if !strings.Contains(out, "one") || !strings.Contains(out, "two") {
		t.Fatalf("expected output content")
	}
}

func TestApplyViewUpdateSpinnerTick(_ *testing.T) {
	s := styles.DefaultStyles()
	view := NewApplyView(s)
	view.SetStatus(ApplyRunning)

	_, _ = view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_, _ = view.Update(view.spinner.Tick())
}
