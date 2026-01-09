package views

import (
	"strings"
	"testing"

	"github.com/ushiradineth/tftui/internal/styles"
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
