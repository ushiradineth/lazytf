package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/styles"
)

// HistoryPanel renders recent apply history entries.
type HistoryPanel struct {
	styles   *styles.Styles
	width    int
	height   int
	entries  []history.Entry
	selected int
	focused  bool
}

// NewHistoryPanel creates a new history panel.
func NewHistoryPanel(s *styles.Styles) *HistoryPanel {
	return &HistoryPanel{styles: s}
}

// SetSize updates the panel dimensions.
func (h *HistoryPanel) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// SetEntries updates the history entries.
func (h *HistoryPanel) SetEntries(entries []history.Entry) {
	h.entries = entries
}

// SetSelection updates the selected entry index and focus state.
func (h *HistoryPanel) SetSelection(index int, focused bool) {
	h.selected = index
	h.focused = focused
}

// SetFocused sets the focus state (implements Panel interface)
func (h *HistoryPanel) SetFocused(focused bool) {
	h.focused = focused
}

// IsFocused returns whether the panel is focused (implements Panel interface)
func (h *HistoryPanel) IsFocused() bool {
	return h.focused
}

// Update handles Bubble Tea messages (implements Panel interface)
func (h *HistoryPanel) Update(_ tea.Msg) (any, tea.Cmd) {
	return h, nil
}

// HandleKey handles key events (implements Panel interface)
func (h *HistoryPanel) HandleKey(_ tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	// History panel doesn't handle keys directly in panel mode
	// Navigation is handled by the app
	return false, nil
}

// View renders the history panel.
func (h *HistoryPanel) View() string {
	if h.styles == nil || h.height <= 0 || h.width <= 0 {
		return ""
	}

	// Determine border style based on focus
	borderStyle := h.styles.Border
	titleStyle := h.styles.PanelTitle
	if h.focused {
		borderStyle = h.styles.FocusedBorder
		titleStyle = h.styles.FocusedPanelTitle
	}

	// Build content
	available := h.height - 2
	if available < 1 {
		available = 1
	}

	// Content width is panel width minus border chars
	contentWidth := h.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	var lines []string
	if len(h.entries) == 0 {
		lines = append(lines, h.styles.Dimmed.Render("No history yet"))
	} else {
		for i := 0; i < len(h.entries) && i < available; i++ {
			lines = append(lines, h.renderEntry(h.entries[i], i == h.selected))
		}
	}

	// Truncate lines to fit content width
	for i := range lines {
		lines[i] = trimString(lines[i], contentWidth)
	}

	content := strings.Join(lines, "\n")

	// Build panel with border
	panel := borderStyle.
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Width(contentWidth).
		Height(h.height - 2).
		Render(content)

	// Add title to border - use simple title without extra styling that might cause width issues
	titleText := " [3] Recent Applies "
	title := titleStyle.Render(titleText)

	// Add title to border
	panelLines := strings.Split(panel, "\n")
	if len(panelLines) > 0 && h.width > 4 {
		if line, ok := RenderPanelTitleLine(h.width, borderStyle, title); ok {
			panelLines[0] = line
		}
	}

	return strings.Join(panelLines, "\n")
}

func (h *HistoryPanel) renderEntry(entry history.Entry, selected bool) string {
	status := string(entry.Status)
	switch entry.Status {
	case history.StatusSuccess:
		status = h.styles.Create.Render("ok")
	case history.StatusFailed:
		status = h.styles.Delete.Render("fail")
	case history.StatusCanceled:
		status = h.styles.Update.Render("cancel")
	}

	ts := entry.FinishedAt
	if ts.IsZero() {
		ts = entry.StartedAt
	}
	when := formatTime(ts)
	label := strings.TrimSpace(entry.Summary)
	if label == "" {
		label = entry.WorkDir
	}
	if label == "" {
		label = "apply"
	}
	dur := formatDuration(entry.Duration)
	if dur != "" {
		dur = " " + dur
	}

	line := fmt.Sprintf("%s %s%s %s", when, status, dur, trimString(label, h.width-14))
	if selected && h.focused {
		return h.styles.Selected.Render(line)
	}
	return line
}

func formatTime(ts time.Time) string {
	if ts.IsZero() {
		return "--:--"
	}
	return ts.Format("15:04")
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

func trimString(val string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(val)
	if len(runes) <= maxLen {
		return val
	}
	if maxLen < 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}
