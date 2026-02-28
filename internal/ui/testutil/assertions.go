package testutil

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// Height Assertions

// AssertHeight asserts that the output has exactly the expected number of lines.
func (r *RenderResult) AssertHeight(t *testing.T, expected int) *RenderResult {
	t.Helper()
	if r.LineCount != expected {
		t.Errorf("height mismatch: got %d lines, want %d", r.LineCount, expected)
	}
	return r
}

// AssertHeightAtMost asserts that the output has at most maxLines lines.
func (r *RenderResult) AssertHeightAtMost(t *testing.T, maxLines int) *RenderResult {
	t.Helper()
	if r.LineCount > maxLines {
		t.Errorf("height overflow: got %d lines, want at most %d", r.LineCount, maxLines)
	}
	return r
}

// AssertHeightAtLeast asserts that the output has at least minLines lines.
func (r *RenderResult) AssertHeightAtLeast(t *testing.T, minLines int) *RenderResult {
	t.Helper()
	if r.LineCount < minLines {
		t.Errorf("height underflow: got %d lines, want at least %d", r.LineCount, minLines)
	}
	return r
}

// Width Assertions

// AssertNoLineOverflow asserts that no line exceeds the configured width.
func (r *RenderResult) AssertNoLineOverflow(t *testing.T) *RenderResult {
	t.Helper()
	for i, line := range r.Lines {
		w := lipgloss.Width(line)
		if w > r.Width {
			t.Errorf("line %d overflows: width %d > configured %d\n  line: %q", i, w, r.Width, line)
		}
	}
	return r
}

// AssertMaxWidth asserts that no line exceeds the given width.
func (r *RenderResult) AssertMaxWidth(t *testing.T, maxWidth int) *RenderResult {
	t.Helper()
	for i, line := range r.Lines {
		w := lipgloss.Width(line)
		if w > maxWidth {
			t.Errorf("line %d overflows: width %d > max %d\n  line: %q", i, w, maxWidth, line)
		}
	}
	return r
}

// AssertMinWidth asserts that at least one line has the given minimum width.
func (r *RenderResult) AssertMinWidth(t *testing.T, minWidth int) *RenderResult {
	t.Helper()
	found := false
	for _, line := range r.Lines {
		if lipgloss.Width(line) >= minWidth {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no line has width >= %d, max width found: %d", minWidth, r.MaxLineWidth)
	}
	return r
}

// Content Assertions

// AssertContains asserts that the output contains the given substring.
func (r *RenderResult) AssertContains(t *testing.T, substr string) *RenderResult {
	t.Helper()
	if !strings.Contains(r.Plain, substr) {
		t.Errorf("output does not contain %q\n  output: %q", substr, r.Plain)
	}
	return r
}

// AssertNotContains asserts that the output does not contain the given substring.
func (r *RenderResult) AssertNotContains(t *testing.T, substr string) *RenderResult {
	t.Helper()
	if strings.Contains(r.Plain, substr) {
		t.Errorf("output unexpectedly contains %q\n  output: %q", substr, r.Plain)
	}
	return r
}

// AssertContainsAll asserts that the output contains all given substrings.
func (r *RenderResult) AssertContainsAll(t *testing.T, substrs ...string) *RenderResult {
	t.Helper()
	for _, substr := range substrs {
		if !strings.Contains(r.Plain, substr) {
			t.Errorf("output does not contain %q\n  output: %q", substr, r.Plain)
		}
	}
	return r
}

// AssertContainsAny asserts that the output contains at least one of the given substrings.
func (r *RenderResult) AssertContainsAny(t *testing.T, substrs ...string) *RenderResult {
	t.Helper()
	for _, substr := range substrs {
		if strings.Contains(r.Plain, substr) {
			return r
		}
	}
	t.Errorf("output does not contain any of %v\n  output: %q", substrs, r.Plain)
	return r
}

// AssertLineContains asserts that a specific line contains the given substring.
func (r *RenderResult) AssertLineContains(t *testing.T, lineIndex int, substr string) *RenderResult {
	t.Helper()
	if lineIndex < 0 || lineIndex >= len(r.Lines) {
		t.Errorf("line index %d out of range [0, %d)", lineIndex, len(r.Lines))
		return r
	}
	if !strings.Contains(r.Lines[lineIndex], substr) {
		t.Errorf("line %d does not contain %q\n  line: %q", lineIndex, substr, r.Lines[lineIndex])
	}
	return r
}

// AssertEmpty asserts that the output is empty or contains only whitespace.
func (r *RenderResult) AssertEmpty(t *testing.T) *RenderResult {
	t.Helper()
	if strings.TrimSpace(r.Plain) != "" {
		t.Errorf("expected empty output, got: %q", r.Plain)
	}
	return r
}

// AssertNotEmpty asserts that the output is not empty.
func (r *RenderResult) AssertNotEmpty(t *testing.T) *RenderResult {
	t.Helper()
	if strings.TrimSpace(r.Plain) == "" {
		t.Error("expected non-empty output")
	}
	return r
}

// Visual Assertions

// AssertHasBorder asserts that the output has border characters.
func (r *RenderResult) AssertHasBorder(t *testing.T) *RenderResult {
	t.Helper()
	borderChars := []string{"│", "─", "╭", "╮", "╰", "╯", "|", "-", "+"}
	for _, char := range borderChars {
		if strings.Contains(r.Plain, char) {
			return r
		}
	}
	t.Errorf("output does not appear to have a border\n  output: %q", r.Plain)
	return r
}

// AssertHasRoundedBorder asserts that the output has rounded border corners.
func (r *RenderResult) AssertHasRoundedBorder(t *testing.T) *RenderResult {
	t.Helper()
	roundedCorners := []string{"╭", "╮", "╰", "╯"}
	foundCorners := 0
	for _, corner := range roundedCorners {
		if strings.Contains(r.Plain, corner) {
			foundCorners++
		}
	}
	if foundCorners < 2 {
		t.Errorf("output does not appear to have rounded border corners\n  output: %q", r.Plain)
	}
	return r
}

// AssertHasScrollbar asserts that the output has a scrollbar character.
func (r *RenderResult) AssertHasScrollbar(t *testing.T) *RenderResult {
	t.Helper()
	// Scrollbar thumb character used in PanelFrame
	if !strings.Contains(r.Plain, "▐") {
		t.Errorf("output does not contain scrollbar thumb (▐)\n  output: %q", r.Plain)
	}
	return r
}

// AssertNoScrollbar asserts that the output does not have a scrollbar.
func (r *RenderResult) AssertNoScrollbar(t *testing.T) *RenderResult {
	t.Helper()
	if strings.Contains(r.Plain, "▐") {
		t.Errorf("output unexpectedly contains scrollbar thumb (▐)\n  output: %q", r.Plain)
	}
	return r
}

// Raw Output Assertions

// AssertRawContains asserts that the raw output (with ANSI) contains the given substring.
func (r *RenderResult) AssertRawContains(t *testing.T, substr string) *RenderResult {
	t.Helper()
	if !strings.Contains(r.Raw, substr) {
		t.Errorf("raw output does not contain %q", substr)
	}
	return r
}

// AssertHasANSI asserts that the raw output contains ANSI escape sequences.
func (r *RenderResult) AssertHasANSI(t *testing.T) *RenderResult {
	t.Helper()
	if !strings.Contains(r.Raw, "\x1b[") {
		t.Errorf("output does not contain any ANSI escape sequences")
	}
	return r
}

// Panel Assertions

// AssertHasPanelID asserts that the output contains a panel ID like [1], [2], etc.
func (r *RenderResult) AssertHasPanelID(t *testing.T, id string) *RenderResult {
	t.Helper()
	if !strings.Contains(r.Plain, id) {
		t.Errorf("output does not contain panel ID %q\n  output: %q", id, r.Plain)
	}
	return r
}

// AssertHasItemCount asserts that the output contains an item count like "7 of 29".
func (r *RenderResult) AssertHasItemCount(t *testing.T) *RenderResult {
	t.Helper()
	if !strings.Contains(r.Plain, " of ") {
		t.Errorf("output does not contain item count (expected ' of ')\n  output: %q", r.Plain)
	}
	return r
}

// Comparison Assertions

// AssertDifferentFrom asserts that this render result differs from another.
func (r *RenderResult) AssertDifferentFrom(t *testing.T, other *RenderResult, description string) *RenderResult {
	t.Helper()
	if r.Plain == other.Plain {
		t.Errorf("render results should be different (%s)\n  both: %q", description, r.Plain)
	}
	return r
}

// AssertStyleDifferentFrom asserts that this result has different styling than another.
// The plain text may be the same, but the ANSI codes should differ.
func (r *RenderResult) AssertStyleDifferentFrom(t *testing.T, other *RenderResult, description string) *RenderResult {
	t.Helper()
	if r.Raw == other.Raw {
		t.Errorf("render styling should be different (%s)", description)
	}
	return r
}
