package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/tftui/internal/history"
	"github.com/ushiradineth/tftui/internal/styles"
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

// View renders the history panel.
func (h *HistoryPanel) View() string {
	if h.styles == nil || h.height <= 0 {
		return ""
	}

	title := h.styles.Title.Render("Apply history")
	lines := []string{title}

	available := h.height - 2
	if available < 1 {
		available = 1
	}

	if len(h.entries) == 0 {
		lines = append(lines, h.styles.Dimmed.Render("No history yet"))
	} else {
		for i := 0; i < len(h.entries) && i < available; i++ {
			lines = append(lines, h.renderEntry(h.entries[i], i == h.selected))
		}
	}

	content := strings.Join(lines, "\n")
	if h.width > 0 {
		content = lipgloss.NewStyle().Width(h.width).Render(content)
	}
	return h.styles.Border.Width(h.width).Render(content)
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

func trimString(val string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(val)
	if len(runes) <= max {
		return val
	}
	if max < 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
