package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestNewAboutView(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewAboutView(s)
	if view == nil {
		t.Fatal("expected non-nil view")
	}
	if view.styles != s {
		t.Error("styles not set correctly")
	}
}

func TestAboutViewSetSize(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewAboutView(s)
	view.SetSize(80, 24)

	if view.width != 80 {
		t.Errorf("expected width 80, got %d", view.width)
	}
	if view.height != 24 {
		t.Errorf("expected height 24, got %d", view.height)
	}
	if view.viewport.Width != 80 {
		t.Errorf("expected viewport width 80, got %d", view.viewport.Width)
	}
	if view.viewport.Height != 24 {
		t.Errorf("expected viewport height 24, got %d", view.viewport.Height)
	}
}

func TestAboutViewSetStyles(t *testing.T) {
	view := NewAboutView(styles.DefaultStyles())
	view.SetSize(60, 20)

	newStyles := styles.DefaultStyles()
	view.SetStyles(newStyles)

	if view.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestAboutViewSetStylesNil(t *testing.T) {
	view := NewAboutView(nil)
	view.SetSize(60, 20)
	view.SetStyles(nil)
	// Should not panic
}

func TestAboutViewUpdate(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewAboutView(s)
	view.SetSize(60, 20)

	// Test that Update returns view and cmd
	updated, cmd := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated != view {
		t.Error("expected same view instance")
	}
	_ = cmd
}

func TestAboutViewViewContent(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewAboutView(s)
	view.SetSize(80, 30)

	content := view.ViewContent()
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestAboutViewViewContentNilStyles(t *testing.T) {
	view := &AboutView{}
	content := view.ViewContent()
	if content != "" {
		t.Error("expected empty content for nil styles")
	}
}

func TestAboutViewViewContentZeroWidth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewAboutView(s)
	view.SetSize(0, 20)

	content := view.ViewContent()
	// Should still return something even with zero width
	_ = content
}

func TestAboutViewUpdateContent(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewAboutView(s)
	view.SetSize(100, 40)

	// Manually trigger updateContent
	view.updateContent()

	content := view.ViewContent()

	// Verify content includes expected sections
	if !strings.Contains(content, "lazytf") {
		t.Error("expected lazytf in content")
	}
	if !strings.Contains(content, "GitHub") {
		t.Error("expected GitHub link in content")
	}
	if !strings.Contains(content, "Keybindings") {
		t.Error("expected keybindings info in content")
	}
}

func TestAboutViewUpdateContentNilStyles(t *testing.T) {
	view := &AboutView{}
	// Should not panic
	view.updateContent()
}

func TestAboutViewLogoConstant(t *testing.T) {
	// Verify the logo is not empty
	if lazytfLogo == "" {
		t.Error("expected non-empty logo")
	}
	// Verify it contains underscores (part of ASCII art)
	if !strings.Contains(lazytfLogo, "_") {
		t.Error("expected logo to contain ASCII art characters")
	}
}
