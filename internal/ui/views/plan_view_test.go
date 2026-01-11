package views

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestPlanViewRendersSummary(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewPlanView("+ 1 to create\n~ 0 to update\n- 0 to destroy", s)
	view.SetSize(60, 10)
	out := view.View()
	if !strings.Contains(out, "Confirm Apply") {
		t.Fatalf("expected confirm title")
	}
	if !strings.Contains(out, "+ 1 to create") {
		t.Fatalf("expected summary content")
	}
	if !strings.Contains(out, "[Y] Yes") {
		t.Fatalf("expected confirmation prompt")
	}
}

func TestPlanViewSmallSize(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewPlanView("+ 1 to create", s)
	view.SetSize(10, 4)
	out := view.View()
	if out == "" {
		t.Fatalf("expected non-empty view for small size")
	}
}

func TestPlanViewSetSummary(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewPlanView("", s)
	view.SetSize(40, 6)
	view.SetSummary("summary")

	out := view.View()
	if !strings.Contains(out, "summary") {
		t.Fatalf("expected updated summary")
	}
}
