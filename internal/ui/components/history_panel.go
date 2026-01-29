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
	when := formatHistoryTime(h.entry)
	statusPlain := historyStatusPlain(h.entry.Status)
	dur := formatDuration(h.entry.Duration)
	desc := historyDescription(h.entry)

	desc = trimString(desc, historyDescWidth(width, statusPlain, dur))

	if selected {
		line := historyLine(when, statusPlain, dur, desc)
		bg := s.SelectedLineBackground
		styled := s.LineItemText.Background(bg).Bold(true).Render(line)
		return PadLineWithBg(styled, width, bg)
	}

	statusText := historyStatusText(s, h.entry.Status, statusPlain)
	line := historyLine(when, statusText, dur, desc)
	return PadLine(line, width)
}

func formatHistoryTime(entry history.Entry) string {
	ts := entry.FinishedAt
	if ts.IsZero() {
		ts = entry.StartedAt
	}
	return formatTime(ts)
}

func historyStatusPlain(status history.Status) string {
	switch status {
	case history.StatusSuccess:
		return "ok"
	case history.StatusFailed:
		return "fail"
	case history.StatusCanceled:
		return "cancel"
	default:
		return string(status)
	}
}

func historyStatusText(s *styles.Styles, status history.Status, plain string) string {
	switch status {
	case history.StatusSuccess:
		return s.Create.Render(plain)
	case history.StatusFailed:
		return s.Delete.Render(plain)
	case history.StatusCanceled:
		return s.Update.Render(plain)
	default:
		return plain
	}
}

func historyDescription(entry history.Entry) string {
	desc := strings.TrimSpace(entry.Summary)
	if desc == "" {
		desc = strings.TrimSpace(entry.Environment)
	}
	if desc == "" {
		wd := strings.TrimSpace(entry.WorkDir)
		if idx := strings.LastIndex(wd, "/"); idx >= 0 && idx < len(wd)-1 {
			wd = wd[idx+1:]
		}
		desc = wd
	}
	if desc == "" {
		desc = "apply"
	}
	return desc
}

func historyDescWidth(width int, statusPlain, dur string) int {
	fixedLen := 6 + len(statusPlain) + 1
	if dur != "" {
		fixedLen += len(dur) + 1
	}
	return max(3, width-fixedLen)
}

func historyLine(when, statusText, dur, desc string) string {
	if dur != "" {
		return fmt.Sprintf("%s %s %s %s", when, statusText, dur, desc)
	}
	return fmt.Sprintf("%s %s %s", when, statusText, desc)
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

// SetFocused sets the focus state (implements Panel interface).
func (h *HistoryPanel) SetFocused(focused bool) {
	h.listPanel.SetFocused(focused)
}

// IsFocused returns whether the panel is focused (implements Panel interface).
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

// Update handles Bubble Tea messages (implements Panel interface).
func (h *HistoryPanel) Update(_ tea.Msg) (any, tea.Cmd) {
	return h, nil
}

// HandleKey handles key events (implements Panel interface).
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
