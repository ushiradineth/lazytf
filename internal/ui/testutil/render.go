// Package testutil provides utilities for testing TUI rendering,
// including dimension handling, selection states, height correctness,
// keybind hints, and layout regression detection.
package testutil

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/terraform/parser"
)

// Renderable is an interface for components that can be rendered.
// This mirrors components.Panel but is defined locally to avoid import cycles.
type Renderable interface {
	View() string
	SetSize(width, height int)
}

// Focusable extends Renderable with focus control.
type Focusable interface {
	Renderable
	SetFocused(focused bool)
}

// Styleable extends Renderable with style control.
type Styleable interface {
	Renderable
	SetStyles(s any)
}

// RenderResult captures rendered output for analysis.
type RenderResult struct {
	// Raw is the original output with ANSI escape sequences.
	Raw string

	// Plain is the ANSI-stripped output.
	Plain string

	// Lines contains the output split into lines.
	Lines []string

	// LineCount is the number of lines in the output.
	LineCount int

	// MaxLineWidth is the maximum visual width of any line.
	MaxLineWidth int

	// Width is the configured render width.
	Width int

	// Height is the configured render height.
	Height int

	t *testing.T
}

// cleaner is a reusable ANSI cleaner instance.
var cleaner = parser.NewCleaner()

// RenderCapture captures the output of a view function for analysis.
func RenderCapture(t *testing.T, view func() string, width, height int) *RenderResult {
	t.Helper()

	raw := view()
	plain := cleaner.StripANSI(raw)
	lines := strings.Split(plain, "\n")

	maxWidth := 0
	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > maxWidth {
			maxWidth = w
		}
	}

	return &RenderResult{
		Raw:          raw,
		Plain:        plain,
		Lines:        lines,
		LineCount:    len(lines),
		MaxLineWidth: maxWidth,
		Width:        width,
		Height:       height,
		t:            t,
	}
}

// RenderComponent renders a Renderable component and captures the output.
func RenderComponent(t *testing.T, r Renderable, width, height int) *RenderResult {
	t.Helper()

	r.SetSize(width, height)
	return RenderCapture(t, r.View, width, height)
}

// RenderWithFocus renders a Focusable component with a specific focus state.
func RenderWithFocus(t *testing.T, f Focusable, width, height int, focused bool) *RenderResult {
	t.Helper()

	f.SetSize(width, height)
	f.SetFocused(focused)
	return RenderCapture(t, f.View, width, height)
}

// Line returns a specific line from the output (0-indexed).
// Returns empty string if index is out of bounds.
func (r *RenderResult) Line(index int) string {
	if index < 0 || index >= len(r.Lines) {
		return ""
	}
	return r.Lines[index]
}

// FirstLine returns the first line of the output.
func (r *RenderResult) FirstLine() string {
	return r.Line(0)
}

// LastLine returns the last line of the output.
func (r *RenderResult) LastLine() string {
	return r.Line(len(r.Lines) - 1)
}

// HasContent returns true if the output contains non-whitespace content.
func (r *RenderResult) HasContent() bool {
	return strings.TrimSpace(r.Plain) != ""
}

// String returns the plain text output.
func (r *RenderResult) String() string {
	return r.Plain
}

// VisualWidth returns the visual width of a specific line.
func (r *RenderResult) VisualWidth(lineIndex int) int {
	if lineIndex < 0 || lineIndex >= len(r.Lines) {
		return 0
	}
	return lipgloss.Width(r.Lines[lineIndex])
}

// Overlayer is an interface for components that overlay on base content.
type Overlayer interface {
	Overlay(baseView string) string
	SetSize(width, height int)
}

// RenderOverlay renders an overlay component on a blank base.
func RenderOverlay(t *testing.T, o Overlayer, width, height int) *RenderResult {
	t.Helper()

	o.SetSize(width, height)

	// Create a blank base view
	base := createBlankBase(width, height)
	raw := o.Overlay(base)
	plain := cleaner.StripANSI(raw)
	lines := strings.Split(plain, "\n")

	maxWidth := 0
	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > maxWidth {
			maxWidth = w
		}
	}

	return &RenderResult{
		Raw:          raw,
		Plain:        plain,
		Lines:        lines,
		LineCount:    len(lines),
		MaxLineWidth: maxWidth,
		Width:        width,
		Height:       height,
		t:            t,
	}
}

// createBlankBase creates a blank base view of the given dimensions.
func createBlankBase(width, height int) string {
	lines := make([]string, height)
	blankLine := strings.Repeat(" ", width)
	for i := range lines {
		lines[i] = blankLine
	}
	return strings.Join(lines, "\n")
}

// Viewable is an interface for components that have just a View method (used for Modal).
type Viewable interface {
	View() string
	SetSize(width, height int)
}

// RenderViewable renders a component that implements Viewable.
func RenderViewable(t *testing.T, v Viewable, width, height int) *RenderResult {
	t.Helper()

	v.SetSize(width, height)
	return RenderCapture(t, v.View, width, height)
}
