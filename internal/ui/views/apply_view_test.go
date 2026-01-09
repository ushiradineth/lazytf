package views

import (
	"strings"
	"testing"

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
