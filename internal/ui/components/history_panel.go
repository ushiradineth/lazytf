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
	coloredDesc := colorizeSummary(s, desc)
	line := historyLine(when, statusText, dur, coloredDesc)
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

// colorizeSummary converts summary text to compact colored format.
// Handles both old verbose format ("+ 1 to create ~ 0 to update") and new compact format ("+1 ~0 -2").
func colorizeSummary(s *styles.Styles, summary string) string {
	// Check if it's the old verbose format
	if strings.Contains(summary, " to ") {
		return convertVerboseToCompact(s, summary)
	}

	// Handle compact format "+1 ~0 -2 ±1"
	parts := strings.Fields(summary)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, "+"):
			result = append(result, s.DiffAdd.Render(part))
		case strings.HasPrefix(part, "-"):
			result = append(result, s.DiffRemove.Render(part))
		case strings.HasPrefix(part, "~"), strings.HasPrefix(part, "±"):
			result = append(result, s.DiffChange.Render(part))
		default:
			result = append(result, part)
		}
	}
	return strings.Join(result, " ")
}

// convertVerboseToCompact converts "+ 1 to create ~ 0 to update - 2 to destroy ± 1 to replace" to colored "+1 ~0 -2 ±1".
func convertVerboseToCompact(s *styles.Styles, summary string) string {
	var result []string

	// Parse patterns like "+ 1 to create", "~ 0 to update", "- 2 to destroy", "± 1 to replace"
	type pattern struct {
		prefix string
		symbol string
		color  string // "add", "change", "remove"
	}
	patterns := []pattern{
		{"+ ", "+", "add"},
		{"~ ", "~", "change"},
		{"- ", "-", "remove"},
		{"± ", "±", "change"},
	}

	for _, p := range patterns {
		if idx := strings.Index(summary, p.prefix); idx != -1 {
			// Find the number after the prefix
			rest := summary[idx+len(p.prefix):]
			numEnd := 0
			for numEnd < len(rest) && rest[numEnd] >= '0' && rest[numEnd] <= '9' {
				numEnd++
			}
			if numEnd > 0 {
				num := rest[:numEnd]
				text := p.symbol + num
				switch p.color {
				case "add":
					result = append(result, s.DiffAdd.Render(text))
				case "remove":
					result = append(result, s.DiffRemove.Render(text))
				case "change":
					result = append(result, s.DiffChange.Render(text))
				}
			}
		}
	}

	if len(result) == 0 {
		return summary
	}
	return strings.Join(result, " ")
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

// SetStyles updates the panel styles.
func (h *HistoryPanel) SetStyles(s *styles.Styles) {
	h.listPanel.styles = s
	if h.listPanel.frame != nil {
		h.listPanel.frame.SetStyles(s)
	}
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
func (h *HistoryPanel) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	switch msg.String() {
	case "j", keyDown:
		h.listPanel.MoveDown()
		return true, nil
	case "k", "up":
		h.listPanel.MoveUp()
		return true, nil
	}
	return false, nil
}

// View renders the history panel.
func (h *HistoryPanel) View() string {
	return h.listPanel.View()
}

func formatTime(ts time.Time) string {
	if ts.IsZero() {
		return "--- -- --:--"
	}
	// Show "Jan 02 15:04" format for compact date+time
	return ts.Format("Jan 02 15:04")
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
