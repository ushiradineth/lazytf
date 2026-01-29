package diff

import (
	"strconv"
	"strings"
)

// Formatter formats diffs for display.
type Formatter struct{}

// NewFormatter creates a new formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatDiffs formats a slice of diffs as multi-line string.
func (f *Formatter) FormatDiffs(diffs []MinimalDiff) string {
	if len(diffs) == 0 {
		return "  (no changes)"
	}

	lines := make([]string, len(diffs))
	for i, diff := range diffs {
		lines[i] = FormatDiff(diff)
	}

	return strings.Join(lines, "\n")
}

// FormatResourceSummary creates a one-line summary for collapsed view.
func (f *Formatter) FormatResourceSummary(address string, action string, changeCount int) string {
	if changeCount == 0 {
		return action + " " + address
	}
	if changeCount == 1 {
		return action + " " + address + "  (1 change)"
	}
	return action + " " + address + "  (" + strconv.Itoa(changeCount) + " changes)"
}
