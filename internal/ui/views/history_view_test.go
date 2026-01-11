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
