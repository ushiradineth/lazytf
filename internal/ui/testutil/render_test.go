package testutil

import (
	"strings"
	"testing"
)

// mockRenderable is a simple mock for testing.
type mockRenderable struct {
	width, height int
	focused       bool
}

func (m *mockRenderable) View() string {
	lines := make([]string, m.height)
	for i := range lines {
		if i == 0 && m.focused {
			lines[i] = "\x1b[32m" + strings.Repeat("X", m.width) + "\x1b[0m"
		} else {
			lines[i] = strings.Repeat(".", m.width)
		}
	}
	return strings.Join(lines, "\n")
}

func (m *mockRenderable) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *mockRenderable) SetFocused(focused bool) {
	m.focused = focused
}

func TestRenderCapture(t *testing.T) {
	result := RenderCapture(t, func() string {
		return "line1\nline2\nline3"
	}, 80, 24)

	if result.LineCount != 3 {
		t.Errorf("expected 3 lines, got %d", result.LineCount)
	}
	if result.Line(0) != "line1" {
		t.Errorf("expected line1, got %q", result.Line(0))
	}
	if result.Line(1) != "line2" {
		t.Errorf("expected line2, got %q", result.Line(1))
	}
	if result.FirstLine() != "line1" {
		t.Errorf("expected line1, got %q", result.FirstLine())
	}
	if result.LastLine() != "line3" {
		t.Errorf("expected line3, got %q", result.LastLine())
	}
}

func TestRenderCaptureStripsANSI(t *testing.T) {
	result := RenderCapture(t, func() string {
		return "\x1b[31mred text\x1b[0m"
	}, 80, 24)

	if result.Plain != "red text" {
		t.Errorf("expected plain text, got %q", result.Plain)
	}
	if !strings.Contains(result.Raw, "\x1b[31m") {
		t.Errorf("expected raw to contain ANSI codes")
	}
}

func TestRenderComponent(t *testing.T) {
	mock := &mockRenderable{}
	result := RenderComponent(t, mock, 80, 20)

	if result.LineCount != 20 {
		t.Errorf("expected 20 lines, got %d", result.LineCount)
	}
	if mock.width != 80 || mock.height != 20 {
		t.Errorf("expected SetSize(80, 20), got (%d, %d)", mock.width, mock.height)
	}
}

func TestRenderWithFocus(t *testing.T) {
	mock := &mockRenderable{}

	focused := RenderWithFocus(t, mock, 80, 20, true)
	if !mock.focused {
		t.Error("expected SetFocused(true)")
	}
	if focused.LineCount != 20 {
		t.Errorf("expected 20 lines, got %d", focused.LineCount)
	}

	unfocused := RenderWithFocus(t, mock, 80, 20, false)
	if mock.focused {
		t.Error("expected SetFocused(false)")
	}
	if unfocused.LineCount != 20 {
		t.Errorf("expected 20 lines, got %d", unfocused.LineCount)
	}

	// The focused result should have ANSI codes
	if !focused.HasContent() {
		t.Error("expected focused content")
	}
	if !unfocused.HasContent() {
		t.Error("expected unfocused content")
	}
}

func TestRenderResultLine(t *testing.T) {
	result := RenderCapture(t, func() string {
		return "a\nb\nc"
	}, 80, 24)

	if result.Line(-1) != "" {
		t.Error("expected empty string for negative index")
	}
	if result.Line(100) != "" {
		t.Error("expected empty string for out of bounds index")
	}
	if result.Line(1) != "b" {
		t.Errorf("expected 'b', got %q", result.Line(1))
	}
}

func TestRenderResultVisualWidth(t *testing.T) {
	result := RenderCapture(t, func() string {
		return "hello\nworld!"
	}, 80, 24)

	if result.VisualWidth(0) != 5 {
		t.Errorf("expected width 5, got %d", result.VisualWidth(0))
	}
	if result.VisualWidth(1) != 6 {
		t.Errorf("expected width 6, got %d", result.VisualWidth(1))
	}
	if result.VisualWidth(100) != 0 {
		t.Errorf("expected width 0 for out of bounds, got %d", result.VisualWidth(100))
	}
}

func TestRenderResultMaxLineWidth(t *testing.T) {
	result := RenderCapture(t, func() string {
		return "short\nmuch longer line here"
	}, 80, 24)

	if result.MaxLineWidth != 21 {
		t.Errorf("expected max width 21, got %d", result.MaxLineWidth)
	}
}

func TestRenderResultHasContent(t *testing.T) {
	empty := RenderCapture(t, func() string {
		return "   \n\t\n  "
	}, 80, 24)
	if empty.HasContent() {
		t.Error("expected no content for whitespace-only")
	}

	content := RenderCapture(t, func() string {
		return "  hello  "
	}, 80, 24)
	if !content.HasContent() {
		t.Error("expected content")
	}
}

func TestRenderResultString(t *testing.T) {
	result := RenderCapture(t, func() string {
		return "\x1b[31mhello\x1b[0m"
	}, 80, 24)

	if result.String() != "hello" {
		t.Errorf("expected String() to return plain text, got %q", result.String())
	}
}
