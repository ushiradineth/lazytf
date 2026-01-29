package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestNewStateShowView(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewStateShowView(s)
	if view == nil {
		t.Fatal("expected non-nil view")
	}
	if view.styles != s {
		t.Error("styles not set correctly")
	}
}

func TestStateShowViewSetSize(t *testing.T) {
	view := NewStateShowView(styles.DefaultStyles())
	view.SetSize(80, 24)

	if view.width != 80 {
		t.Errorf("expected width 80, got %d", view.width)
	}
	// Viewport height should be height - header - footer
	if view.viewport.Height != 22 {
		t.Errorf("expected viewport height 22, got %d", view.viewport.Height)
	}
	if view.viewport.Width != 80 {
		t.Errorf("expected viewport width 80, got %d", view.viewport.Width)
	}
}

func TestStateShowViewSetSizeSmall(t *testing.T) {
	view := NewStateShowView(styles.DefaultStyles())
	view.SetSize(80, 2) // Very small

	// Should have minimum height of 1
	if view.viewport.Height != 1 {
		t.Errorf("expected minimum viewport height 1, got %d", view.viewport.Height)
	}
}

func TestStateShowViewSetAddress(t *testing.T) {
	view := NewStateShowView(styles.DefaultStyles())
	view.SetAddress("aws_instance.web")
	if view.address != "aws_instance.web" {
		t.Errorf("expected address 'aws_instance.web', got %q", view.address)
	}
}

func TestStateShowViewSetContent(t *testing.T) {
	view := NewStateShowView(styles.DefaultStyles())
	view.SetSize(80, 10)
	view.SetContent("resource content\nline 2\n\n")

	// Content should be trimmed of trailing newlines
	// and viewport should be at top
	if view.viewport.YOffset != 0 {
		t.Errorf("expected viewport at top, got offset %d", view.viewport.YOffset)
	}
}

func TestStateShowViewUpdate(t *testing.T) {
	view := NewStateShowView(styles.DefaultStyles())
	view.SetSize(80, 10)
	view.SetContent("line 1\nline 2\nline 3\nline 4\nline 5")

	// Test that Update returns view and cmd
	updated, cmd := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated != view {
		t.Error("expected same view instance")
	}
	// cmd may be nil for simple key presses
	_ = cmd
}

func TestStateShowViewView(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewStateShowView(s)
	view.SetSize(60, 10)
	view.SetAddress("aws_instance.web")
	view.SetContent("resource \"aws_instance\" \"web\" {\n  ami = \"ami-12345\"\n}")

	out := view.View()

	if !strings.Contains(out, "State: aws_instance.web") {
		t.Error("expected title with address in output")
	}
	if !strings.Contains(out, "scroll") {
		t.Error("expected footer with scroll help")
	}
}

func TestStateShowViewViewEmptyAddress(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewStateShowView(s)
	view.SetSize(60, 10)
	view.SetContent("some content")

	out := view.View()

	if !strings.Contains(out, "State Details") {
		t.Error("expected default title when address is empty")
	}
}

func TestStateShowViewViewNilStyles(t *testing.T) {
	view := &StateShowView{}
	out := view.View()
	if out != "" {
		t.Error("expected empty output for nil styles")
	}
}

func TestStateShowViewViewWithWidth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewStateShowView(s)
	view.SetSize(40, 8)
	view.SetContent("test content")

	out := view.View()
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestStateShowViewViewZeroWidth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewStateShowView(s)
	view.SetSize(0, 10)
	view.SetContent("test content")

	out := view.View()
	// Should still render but without width constraint on body
	if out == "" {
		t.Error("expected non-empty output even with zero width")
	}
}
