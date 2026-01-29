package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/styles"
)

// HistoryItem implements ListPanelItem for history entries.
type HistoryItem struct {
	entry history.Entry
}

// Render renders the history item.
func (h HistoryItem) Render(s *styles.Styles, width int, selected bool) string {
	// Get timestamp
	ts := h.entry.FinishedAt
	if ts.IsZero() {
		ts = h.entry.StartedAt
	}
	when := formatTime(ts)

	// Get plain status text for width calculation
	var statusPlain string
	switch h.entry.Status {
	case history.StatusSuccess:
		statusPlain = "ok"
	case history.StatusFailed:
		statusPlain = "fail"
	case history.StatusCanceled:
		statusPlain = "cancel"
	default:
		statusPlain = string(h.entry.Status)
	}

	// Get duration
	dur := formatDuration(h.entry.Duration)

	// Get description - prefer Summary (shows what happened), fall back to Environment/WorkDir
	desc := strings.TrimSpace(h.entry.Summary)
	if desc == "" {
		desc = strings.TrimSpace(h.entry.Environment)
	}
	if desc == "" {
		// Get last component of WorkDir for brevity
		wd := strings.TrimSpace(h.entry.WorkDir)
		if idx := strings.LastIndex(wd, "/"); idx >= 0 && idx < len(wd)-1 {
			wd = wd[idx+1:]
		}
		desc = wd
	}
	if desc == "" {
		desc = "apply"
	}

	// Build line: "HH:MM ok 5s + 1 to create" or "HH:MM fail prod"
	// Calculate available space for description
	// Fixed parts: "HH:MM " (6) + status (max 6 visual chars) + space (1) = 13 chars
	fixedLen := 6 + len(statusPlain) + 1
	if dur != "" {
		fixedLen += len(dur) + 1 // duration + space
	}
	descWidth := max(3, width-fixedLen)
	desc = trimString(desc, descWidth)

	// Apply styling and ensure full-width
	if selected {
		// For selected: build plain text line, then apply full background
		var line string
		if dur != "" {
			line = fmt.Sprintf("%s %s %s %s", when, statusPlain, dur, desc)
		} else {
			line = fmt.Sprintf("%s %s %s", when, statusPlain, desc)
		}
		bg := s.SelectedLineBackground
		styled := s.LineItemText.Background(bg).Bold(true).Render(line)
		return PadLineWithBg(styled, width, bg)
	}

	// Non-selected: apply status color styling
	var statusText string
	switch h.entry.Status {
	case history.StatusSuccess:
		statusText = s.Create.Render(statusPlain)
	case history.StatusFailed:
		statusText = s.Delete.Render(statusPlain)
	case history.StatusCanceled:
		statusText = s.Update.Render(statusPlain)
	default:
		statusText = statusPlain
	}

	var line string
	if dur != "" {
		line = fmt.Sprintf("%s %s %s %s", when, statusText, dur, desc)
	} else {
		line = fmt.Sprintf("%s %s %s", when, statusText, desc)
	}
	return PadLine(line, width)
}

// HistoryPanel renders recent apply history entries using ListPanel.
type HistoryPanel struct {
	listPanel *ListPanel
	entries   []history.Entry
}

// NewHistoryPanel creates a new history panel.
func NewHistoryPanel(s *styles.Styles) *HistoryPanel {
	if s == nil {
		s = styles.DefaultStyles()
	}
	panel := NewListPanel("[3]", s)
	panel.SetTabs([]string{"Recent Applies"})
	return &HistoryPanel{
		listPanel: panel,
	}
}

// SetSize updates the panel dimensions.
func (h *HistoryPanel) SetSize(width, height int) {
	h.listPanel.SetSize(width, height)
}

// SetEntries updates the history entries.
func (h *HistoryPanel) SetEntries(entries []history.Entry) {
	h.entries = entries
	items := make([]ListPanelItem, len(entries))
	for i, entry := range entries {
		items[i] = HistoryItem{entry: entry}
	}
	h.listPanel.SetItems(items)
}

// SetSelection updates the selected entry index and focus state.
func (h *HistoryPanel) SetSelection(index int, focused bool) {
	h.listPanel.SetSelectedIndex(index)
	h.listPanel.SetFocused(focused)
}

// SetFocused sets the focus state (implements Panel interface)
func (h *HistoryPanel) SetFocused(focused bool) {
	h.listPanel.SetFocused(focused)
}

// IsFocused returns whether the panel is focused (implements Panel interface)
func (h *HistoryPanel) IsFocused() bool {
	return h.listPanel.IsFocused()
}

// GetSelectedIndex returns the currently selected index.
func (h *HistoryPanel) GetSelectedIndex() int {
	return h.listPanel.GetSelectedIndex()
}

// GetSelectedEntry returns the currently selected entry.
func (h *HistoryPanel) GetSelectedEntry() *history.Entry {
	idx := h.listPanel.GetSelectedIndex()
	if idx >= 0 && idx < len(h.entries) {
		return &h.entries[idx]
	}
	return nil
}

// MoveUp moves the selection up.
func (h *HistoryPanel) MoveUp() bool {
	return h.listPanel.MoveUp()
}

// MoveDown moves the selection down.
func (h *HistoryPanel) MoveDown() bool {
	return h.listPanel.MoveDown()
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
	return h.listPanel.View()
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
