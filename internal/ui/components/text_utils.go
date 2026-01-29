package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// TruncateWithANSI truncates a string that may contain ANSI codes to the given visible width.
// Unlike runewidth.Truncate, this properly handles ANSI escape sequences.
func TruncateWithANSI(s string, width int) string {
	if width <= 0 {
		return ""
	}

	var result strings.Builder
	currentWidth := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		// Regular character - check if it fits
		charWidth := runewidth.RuneWidth(r)
		if currentWidth+charWidth > width {
			break
		}
		result.WriteRune(r)
		currentWidth += charWidth
	}

	return result.String()
}

// PadLine pads or truncates a line to the given width.
// Handles ANSI escape sequences properly.
func PadLine(line string, width int) string {
	visibleWidth := lipgloss.Width(line)
	if visibleWidth == width {
		// Already exact width, return as-is (avoids ANSI truncation issues)
		return line
	}
	if visibleWidth > width {
		// Need to truncate - use ANSI-aware truncation
		return TruncateWithANSI(line, width)
	}
	return line + strings.Repeat(" ", width-visibleWidth)
}

// PadLineWithBg pads a styled string to width with the given background color.
// The padding spaces will have the same background color as the content.
func PadLineWithBg(styled string, width int, bg lipgloss.AdaptiveColor) string {
	visible := lipgloss.Width(styled)
	if visible >= width {
		// Use ANSI-aware truncation if needed
		return TruncateWithANSI(styled, width)
	}
	padding := strings.Repeat(" ", width-visible)
	return styled + lipgloss.NewStyle().Background(bg).Render(padding)
}

// PadLineWithStyle pads a styled string to width using the given style for padding.
// This is useful when you want the padding to have the same style as the content.
func PadLineWithStyle(styled string, width int, paddingStyle lipgloss.Style) string {
	visible := lipgloss.Width(styled)
	if visible >= width {
		return TruncateWithANSI(styled, width)
	}
	padding := strings.Repeat(" ", width-visible)
	return styled + paddingStyle.Render(padding)
}

// ANSICutLeft returns the portion of s after skipping the first n visual characters.
// It properly handles ANSI escape sequences, preserving the active styling at the cut point.
func ANSICutLeft(s string, n int) string {
	if n <= 0 {
		return s
	}

	var result strings.Builder
	var activeANSI strings.Builder // Track active ANSI codes
	visualPos := 0
	i := 0
	runes := []rune(s)

	// Skip first n visual characters while tracking ANSI codes
	for i < len(runes) && visualPos < n {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Capture ANSI escape sequence
			j := i + 2
			for j < len(runes) && !IsANSITerminator(runes[j]) {
				j++
			}
			if j < len(runes) {
				j++ // Include terminator
			}
			// Track this ANSI code - it might affect remaining text
			ansiCode := string(runes[i:j])
			// Check if it's a reset code
			if ansiCode == "\x1b[0m" || ansiCode == "\x1b[m" {
				activeANSI.Reset()
			} else {
				activeANSI.WriteString(ansiCode)
			}
			i = j
		} else {
			// Regular character - count it
			visualPos++
			i++
		}
	}

	// Prepend active ANSI codes to maintain styling
	if activeANSI.Len() > 0 {
		result.WriteString(activeANSI.String())
	}

	// Collect remaining characters
	for i < len(runes) {
		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// IsANSITerminator checks if a rune is an ANSI escape sequence terminator.
func IsANSITerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// VisibleWidth returns the visible width of a string, ignoring ANSI codes.
// This is equivalent to lipgloss.Width but provided here for convenience.
func VisibleWidth(s string) int {
	return lipgloss.Width(s)
}
