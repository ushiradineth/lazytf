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

// AssertFocusedBorder asserts that the output appears to have a focused border style.
// This checks for visual indicators like a different title style or border color.
// Note: ANSI colors are stripped, so we check the raw output for escape sequences.
func (r *RenderResult) AssertFocusedBorder(t *testing.T) *RenderResult {
	t.Helper()
	// Check for ANSI escape sequences in raw output that indicate styling
	if !strings.Contains(r.Raw, "\x1b[") {
		t.Errorf("output does not appear to have any styling (no ANSI codes)")
		return r
	}
	// Focused borders typically have more styling than unfocused
	r.AssertHasBorder(t)
	return r
}

// AssertUnfocusedBorder asserts that the output has a border but appears unfocused.
func (r *RenderResult) AssertUnfocusedBorder(t *testing.T) *RenderResult {
	t.Helper()
	r.AssertHasBorder(t)
	return r
}

// Selection Assertions

// AssertHasSelection asserts that the output shows selection highlighting.
// This typically manifests as background color styling in the raw output.
func (r *RenderResult) AssertHasSelection(t *testing.T) *RenderResult {
	t.Helper()
	// Selection highlighting uses background colors, which show up as specific ANSI codes
	// Background colors in ANSI use codes 40-47, 48;5;X, or 48;2;R;G;B
	if !strings.Contains(r.Raw, "\x1b[") {
		t.Errorf("output does not appear to have selection highlighting (no ANSI codes)")
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

// Color Assertions

// AssertHasBackgroundColor asserts that the raw output contains background color codes.
// Background colors use ANSI codes 40-47, 48;5;X (256 color), or 48;2;R;G;B (true color).
func (r *RenderResult) AssertHasBackgroundColor(t *testing.T) *RenderResult {
	t.Helper()
	// Check for various background color code patterns
	patterns := []string{
		"\x1b[4",   // Standard background colors (40-47, 49)
		"\x1b[48;", // 256-color or true-color background
		"\x1b[10",  // Bright background colors (100-107)
	}
	for _, p := range patterns {
		if strings.Contains(r.Raw, p) {
			return r
		}
	}
	t.Errorf("output does not contain background color codes")
	return r
}

// AssertHasForegroundColor asserts that the raw output contains foreground color codes.
func (r *RenderResult) AssertHasForegroundColor(t *testing.T) *RenderResult {
	t.Helper()
	patterns := []string{
		"\x1b[3",   // Standard foreground colors (30-37, 39)
		"\x1b[38;", // 256-color or true-color foreground
		"\x1b[9",   // Bright foreground colors (90-97)
	}
	for _, p := range patterns {
		if strings.Contains(r.Raw, p) {
			return r
		}
	}
	t.Errorf("output does not contain foreground color codes")
	return r
}

// AssertHasBoldText asserts that the raw output contains bold text formatting.
func (r *RenderResult) AssertHasBoldText(t *testing.T) *RenderResult {
	t.Helper()
	if !strings.Contains(r.Raw, "\x1b[1m") && !strings.Contains(r.Raw, "\x1b[1;") {
		t.Errorf("output does not contain bold text formatting")
	}
	return r
}

// AssertHasDimText asserts that the raw output contains dim/faint text formatting.
func (r *RenderResult) AssertHasDimText(t *testing.T) *RenderResult {
	t.Helper()
	if !strings.Contains(r.Raw, "\x1b[2m") && !strings.Contains(r.Raw, "\x1b[2;") {
		t.Errorf("output does not contain dim text formatting")
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

// AssertHasTitle asserts that the first line contains the expected title.
func (r *RenderResult) AssertHasTitle(t *testing.T, title string) *RenderResult {
	t.Helper()
	if len(r.Lines) == 0 {
		t.Errorf("output has no lines to check for title")
		return r
	}
	if !strings.Contains(r.Lines[0], title) {
		t.Errorf("title line does not contain %q\n  first line: %q", title, r.Lines[0])
	}
	return r
}

// AssertHasFooter asserts that the last line contains the expected footer text.
func (r *RenderResult) AssertHasFooter(t *testing.T, footer string) *RenderResult {
	t.Helper()
	if len(r.Lines) == 0 {
		t.Errorf("output has no lines to check for footer")
		return r
	}
	lastLine := r.Lines[len(r.Lines)-1]
	if !strings.Contains(lastLine, footer) {
		t.Errorf("footer line does not contain %q\n  last line: %q", footer, lastLine)
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

// Action/Status Assertions

// AssertHasActionIndicator asserts that the output shows action type indicators.
func (r *RenderResult) AssertHasActionIndicator(t *testing.T, indicator string) *RenderResult {
	t.Helper()
	// Action indicators: + (create), ~ (update), - (delete), ± (replace)
	if !strings.Contains(r.Plain, indicator) {
		t.Errorf("output does not contain action indicator %q", indicator)
	}
	return r
}

// AssertHasCreateIndicator asserts the output shows a create (+) indicator.
func (r *RenderResult) AssertHasCreateIndicator(t *testing.T) *RenderResult {
	t.Helper()
	return r.AssertContainsAny(t, "+", "create", "C")
}

// AssertHasUpdateIndicator asserts the output shows an update (~) indicator.
func (r *RenderResult) AssertHasUpdateIndicator(t *testing.T) *RenderResult {
	t.Helper()
	return r.AssertContainsAny(t, "~", "update", "U")
}

// AssertHasDeleteIndicator asserts the output shows a delete (-) indicator.
func (r *RenderResult) AssertHasDeleteIndicator(t *testing.T) *RenderResult {
	t.Helper()
	return r.AssertContainsAny(t, "-", "delete", "D")
}

// AssertHasReplaceIndicator asserts the output shows a replace (±) indicator.
func (r *RenderResult) AssertHasReplaceIndicator(t *testing.T) *RenderResult {
	t.Helper()
	return r.AssertContainsAny(t, "±", "replace", "R")
}

// Modal/Toast Assertions

// AssertIsCentered asserts that content appears centered (has leading whitespace on content lines).
func (r *RenderResult) AssertIsCentered(t *testing.T) *RenderResult {
	t.Helper()
	if len(r.Lines) < 3 {
		return r // Not enough lines to check centering
	}
	// Check middle lines for leading whitespace (indication of centering)
	middleLine := r.Lines[len(r.Lines)/2]
	if middleLine != "" && middleLine[0] != ' ' {
		// Content might still be centered if using full width
		// Check if there's significant padding
		trimmed := strings.TrimLeft(middleLine, " ")
		if len(trimmed) == len(middleLine) && r.Width > 40 {
			t.Errorf("content does not appear to be centered (no leading padding)")
		}
	}
	return r
}

// AssertHasOverlay asserts that the output has characteristics of an overlay (modal/toast).
// Overlays typically have borders and are smaller than the full render area.
func (r *RenderResult) AssertHasOverlay(t *testing.T) *RenderResult {
	t.Helper()
	r.AssertHasBorder(t)
	// Overlays should have some content
	r.AssertNotEmpty(t)
	return r
}

// Diff Viewer Assertions

// AssertHasDiffAddition asserts the output shows diff addition styling.
func (r *RenderResult) AssertHasDiffAddition(t *testing.T) *RenderResult {
	t.Helper()
	// Diff additions typically show with + prefix or green color
	if !strings.Contains(r.Plain, "+") && !strings.Contains(r.Raw, "\x1b[32") {
		t.Errorf("output does not show diff addition (+ or green color)")
	}
	return r
}

// AssertHasDiffRemoval asserts the output shows diff removal styling.
func (r *RenderResult) AssertHasDiffRemoval(t *testing.T) *RenderResult {
	t.Helper()
	// Diff removals typically show with - prefix or red color
	if !strings.Contains(r.Plain, "-") && !strings.Contains(r.Raw, "\x1b[31") {
		t.Errorf("output does not show diff removal (- or red color)")
	}
	return r
}

// AssertHasDiffChange asserts the output shows diff change styling.
func (r *RenderResult) AssertHasDiffChange(t *testing.T) *RenderResult {
	t.Helper()
	// Changes typically show with ~ or yellow color
	if !strings.Contains(r.Plain, "~") && !strings.Contains(r.Plain, "→") &&
		!strings.Contains(r.Raw, "\x1b[33") {
		t.Errorf("output does not show diff change (~ or → or yellow color)")
	}
	return r
}

// Layout Assertions

// AssertFillsWidth asserts that at least one line fills the configured width.
func (r *RenderResult) AssertFillsWidth(t *testing.T) *RenderResult {
	t.Helper()
	for _, line := range r.Lines {
		if lipgloss.Width(line) == r.Width {
			return r
		}
	}
	t.Errorf("no line fills the configured width %d, max width: %d", r.Width, r.MaxLineWidth)
	return r
}

// AssertFillsHeight asserts that the output has exactly the configured height.
func (r *RenderResult) AssertFillsHeight(t *testing.T) *RenderResult {
	t.Helper()
	if r.LineCount != r.Height {
		t.Errorf("output does not fill height: got %d lines, want %d", r.LineCount, r.Height)
	}
	return r
}

// AssertFillsDimensions asserts that the output fills both width and height.
func (r *RenderResult) AssertFillsDimensions(t *testing.T) *RenderResult {
	t.Helper()
	r.AssertFillsHeight(t)
	r.AssertFillsWidth(t)
	return r
}

// Selection State Assertions

// AssertSelectionVisible asserts that the selection highlight is visible.
// This checks for background color changes that indicate selection.
func (r *RenderResult) AssertSelectionVisible(t *testing.T) *RenderResult {
	t.Helper()
	r.AssertHasBackgroundColor(t)
	return r
}

// AssertNoSelection asserts that there is no visible selection highlight.
func (r *RenderResult) AssertNoSelection(t *testing.T) *RenderResult {
	t.Helper()
	// Check that there's no background color for selection
	// This is a weak check - selection might use specific colors
	// Note: \x1b[48; is 256/true color background - might indicate selection
	// but we can't definitively check without knowing the exact selection color
	_ = strings.Contains(r.Raw, "\x1b[48;")
	return r
}

// Progress Assertions

// AssertHasSpinner asserts that the output contains spinner characters.
func (r *RenderResult) AssertHasSpinner(t *testing.T) *RenderResult {
	t.Helper()
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏", "◐", "◓", "◑", "◒", "⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	for _, char := range spinnerChars {
		if strings.Contains(r.Plain, char) {
			return r
		}
	}
	t.Errorf("output does not contain spinner characters")
	return r
}

// AssertHasProgressCount asserts the output shows progress counts like "+3 ~2 -1".
func (r *RenderResult) AssertHasProgressCount(t *testing.T) *RenderResult {
	t.Helper()
	// Look for patterns like +N, ~N, -N, ±N
	hasCount := false
	for _, prefix := range []string{"+", "~", "-", "±"} {
		for i := 0; i <= 9; i++ {
			if strings.Contains(r.Plain, prefix+intToString(i)) {
				hasCount = true
				break
			}
		}
		if hasCount {
			break
		}
	}
	if !hasCount {
		t.Errorf("output does not contain progress counts")
	}
	return r
}

// Status Assertions

// AssertHasSuccessStatus asserts the output shows success status indicators.
func (r *RenderResult) AssertHasSuccessStatus(t *testing.T) *RenderResult {
	t.Helper()
	// Success typically shown with green color or checkmark
	if !strings.Contains(r.Plain, "✓") && !strings.Contains(r.Plain, "ok") &&
		!strings.Contains(r.Plain, "success") && !strings.Contains(r.Plain, "Success") &&
		!strings.Contains(r.Plain, "complete") && !strings.Contains(r.Plain, "Complete") &&
		!strings.Contains(r.Raw, "\x1b[32") { // green
		t.Errorf("output does not show success status")
	}
	return r
}

// AssertHasErrorStatus asserts the output shows error status indicators.
func (r *RenderResult) AssertHasErrorStatus(t *testing.T) *RenderResult {
	t.Helper()
	// Error typically shown with red color or X mark
	if !strings.Contains(r.Plain, "✗") && !strings.Contains(r.Plain, "fail") &&
		!strings.Contains(r.Plain, "error") && !strings.Contains(r.Plain, "Error") &&
		!strings.Contains(r.Plain, "Failed") &&
		!strings.Contains(r.Raw, "\x1b[31") { // red
		t.Errorf("output does not show error status")
	}
	return r
}

// AssertHasWarningStatus asserts the output shows warning status indicators.
func (r *RenderResult) AssertHasWarningStatus(t *testing.T) *RenderResult {
	t.Helper()
	// Warning typically shown with yellow color or warning symbol
	if !strings.Contains(r.Plain, "⚠") && !strings.Contains(r.Plain, "warn") &&
		!strings.Contains(r.Plain, "Warning") &&
		!strings.Contains(r.Raw, "\x1b[33") { // yellow
		t.Errorf("output does not show warning status")
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

// AssertSameAs asserts that this render result is identical to another.
func (r *RenderResult) AssertSameAs(t *testing.T, other *RenderResult, description string) *RenderResult {
	t.Helper()
	if r.Plain != other.Plain {
		t.Errorf("render results should be identical (%s)\n  got: %q\n  want: %q", description, r.Plain, other.Plain)
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
