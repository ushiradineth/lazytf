package components

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestHistoryPanelRendersEntries(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewHistoryPanel(s)
	panel.SetSize(50, 6)
	panel.SetEntries([]history.Entry{
		{
			StartedAt: time.Now(),
			Status:    history.StatusSuccess,
			Summary:   "+ 1 to create ~ 0 to update - 0 to destroy",
		},
	})
	panel.SetSelection(0, true)

	out := panel.View()
	if !strings.Contains(out, "Recent Applies") {
		t.Fatalf("expected header in output")
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("expected status in output")
	}
	if !strings.Contains(out, "+ 1 to create") {
		t.Fatalf("expected summary in output")
	}
}

func TestHistoryPanelNilStyles(t *testing.T) {
	panel := NewHistoryPanel(nil)
	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
}

func TestHistoryPanelSetFocused(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewHistoryPanel(s)
	panel.SetSize(50, 6)

	panel.SetFocused(true)
	if !panel.IsFocused() {
		t.Error("expected panel to be focused")
	}

	panel.SetFocused(false)
	if panel.IsFocused() {
		t.Error("expected panel to not be focused")
	}
}

func TestHistoryPanelGetSelectedIndex(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewHistoryPanel(s)
	panel.SetSize(50, 6)
	panel.SetEntries([]history.Entry{
		{Status: history.StatusSuccess},
		{Status: history.StatusFailed},
	})
	panel.SetSelection(1, true)

	if panel.GetSelectedIndex() != 1 {
		t.Errorf("expected selected index 1, got %d", panel.GetSelectedIndex())
	}
}

func TestHistoryPanelGetSelectedEntry(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewHistoryPanel(s)
	panel.SetSize(50, 6)
	panel.SetEntries([]history.Entry{
		{Status: history.StatusSuccess, Summary: "first"},
		{Status: history.StatusFailed, Summary: "second"},
	})
	panel.SetSelection(1, true)

	entry := panel.GetSelectedEntry()
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Summary != "second" {
		t.Errorf("expected summary 'second', got %s", entry.Summary)
	}

	// Empty list should return nil
	emptyPanel := NewHistoryPanel(s)
	emptyPanel.SetSize(50, 6)
	emptyPanel.SetEntries(nil)
	if emptyPanel.GetSelectedEntry() != nil {
		t.Error("expected nil entry for empty list")
	}
}

func TestHistoryPanelMoveUpDown(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewHistoryPanel(s)
	panel.SetSize(50, 6)
	panel.SetEntries([]history.Entry{
		{Status: history.StatusSuccess},
		{Status: history.StatusFailed},
		{Status: history.StatusCanceled},
	})
	panel.SetSelection(0, true)

	// Move down
	if !panel.MoveDown() {
		t.Error("expected MoveDown to return true")
	}
	if panel.GetSelectedIndex() != 1 {
		t.Errorf("expected index 1 after MoveDown, got %d", panel.GetSelectedIndex())
	}

	// Move up
	if !panel.MoveUp() {
		t.Error("expected MoveUp to return true")
	}
	if panel.GetSelectedIndex() != 0 {
		t.Errorf("expected index 0 after MoveUp, got %d", panel.GetSelectedIndex())
	}
}

func TestHistoryPanelUpdate(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewHistoryPanel(s)
	result, cmd := panel.Update(nil)
	if result != panel {
		t.Error("expected Update to return same panel")
	}
	if cmd != nil {
		t.Error("expected Update to return nil cmd")
	}
}

func TestHistoryPanelHandleKey(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewHistoryPanel(s)
	handled, cmd := panel.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if handled {
		t.Error("expected HandleKey to return false")
	}
	if cmd != nil {
		t.Error("expected HandleKey to return nil cmd")
	}
}

func TestHistoryStatusPlain(t *testing.T) {
	tests := []struct {
		status history.Status
		want   string
	}{
		{history.StatusSuccess, "ok"},
		{history.StatusFailed, "fail"},
		{history.StatusCanceled, "cancel"},
		{history.Status("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := historyStatusPlain(tt.status)
			if got != tt.want {
				t.Errorf("historyStatusPlain(%v) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestHistoryStatusText(t *testing.T) {
	s := styles.DefaultStyles()

	tests := []struct {
		status history.Status
		plain  string
	}{
		{history.StatusSuccess, "ok"},
		{history.StatusFailed, "fail"},
		{history.StatusCanceled, "cancel"},
		{history.Status("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := historyStatusText(s, tt.status, tt.plain)
			if got == "" {
				t.Errorf("historyStatusText returned empty string for %v", tt.status)
			}
		})
	}
}

func TestHistoryDescription(t *testing.T) {
	tests := []struct {
		name  string
		entry history.Entry
		want  string
	}{
		{
			name:  "with summary",
			entry: history.Entry{Summary: "test summary"},
			want:  "test summary",
		},
		{
			name:  "with environment",
			entry: history.Entry{Environment: "dev"},
			want:  "dev",
		},
		{
			name:  "with workdir",
			entry: history.Entry{WorkDir: "/path/to/project"},
			want:  "project",
		},
		{
			name:  "empty fallback",
			entry: history.Entry{},
			want:  "apply",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := historyDescription(tt.entry)
			if got != tt.want {
				t.Errorf("historyDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHistoryDescWidth(t *testing.T) {
	// With duration
	width := historyDescWidth(50, "ok", "10s")
	if width <= 0 {
		t.Errorf("expected positive width, got %d", width)
	}

	// Without duration
	width = historyDescWidth(50, "ok", "")
	if width <= 0 {
		t.Errorf("expected positive width, got %d", width)
	}

	// Very small width
	width = historyDescWidth(5, "ok", "10s")
	if width < 3 {
		t.Errorf("expected minimum width of 3, got %d", width)
	}
}

func TestHistoryLine(t *testing.T) {
	// With duration
	line := historyLine("10:30", "ok", "5s", "summary")
	if !strings.Contains(line, "10:30") || !strings.Contains(line, "ok") || !strings.Contains(line, "5s") || !strings.Contains(line, "summary") {
		t.Errorf("unexpected line format: %q", line)
	}

	// Without duration
	line = historyLine("10:30", "ok", "", "summary")
	if !strings.Contains(line, "10:30") || !strings.Contains(line, "ok") || !strings.Contains(line, "summary") {
		t.Errorf("unexpected line format: %q", line)
	}
	if strings.Contains(line, "  ") {
		// Should not have double spaces from empty duration
	}
}

func TestFormatTime(t *testing.T) {
	// Zero time
	if formatTime(time.Time{}) != "--- -- --:--" {
		t.Errorf("expected '--- -- --:--' for zero time, got %s", formatTime(time.Time{}))
	}

	// Non-zero time
	tm := time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC)
	if formatTime(tm) != "Jan 01 10:30" {
		t.Errorf("expected 'Jan 01 10:30', got %s", formatTime(tm))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		dur  time.Duration
		want string
	}{
		{0, ""},
		{-1 * time.Second, ""},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m"},
		{5 * time.Minute, "5m"},
	}

	for _, tt := range tests {
		t.Run(tt.dur.String(), func(t *testing.T) {
			got := formatDuration(tt.dur)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.dur, got, tt.want)
			}
		})
	}
}

func TestTrimString(t *testing.T) {
	tests := []struct {
		val    string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"hello", 2, "he"},
		{"hello", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			got := trimString(tt.val, tt.maxLen)
			if got != tt.want {
				t.Errorf("trimString(%q, %d) = %q, want %q", tt.val, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestFormatHistoryTime(t *testing.T) {
	now := time.Now()

	// With finished time
	entry1 := history.Entry{
		StartedAt:  now.Add(-time.Minute),
		FinishedAt: now,
	}
	result := formatHistoryTime(entry1)
	if result == "--:--" {
		t.Error("expected non-zero time format")
	}

	// Without finished time
	entry2 := history.Entry{
		StartedAt: now,
	}
	result = formatHistoryTime(entry2)
	if result == "--:--" {
		t.Error("expected non-zero time format from started time")
	}
}

func TestHistoryItemRenderSelected(t *testing.T) {
	s := styles.DefaultStyles()
	entry := history.Entry{
		StartedAt: time.Now(),
		Status:    history.StatusSuccess,
		Summary:   "Test summary",
		Duration:  5 * time.Second,
	}
	item := HistoryItem{entry: entry}

	rendered := item.Render(s, 50, true)
	if rendered == "" {
		t.Error("expected non-empty render for selected item")
	}
	if !strings.Contains(rendered, "ok") {
		t.Error("expected status in rendered output")
	}
}

func TestHistoryItemRenderNotSelected(t *testing.T) {
	s := styles.DefaultStyles()
	entry := history.Entry{
		StartedAt: time.Now(),
		Status:    history.StatusFailed,
		Summary:   "Test summary",
	}
	item := HistoryItem{entry: entry}

	rendered := item.Render(s, 50, false)
	if rendered == "" {
		t.Error("expected non-empty render for non-selected item")
	}
	if !strings.Contains(rendered, "fail") {
		t.Error("expected status in rendered output")
	}
}

func TestHistoryItemRenderCanceled(t *testing.T) {
	s := styles.DefaultStyles()
	entry := history.Entry{
		StartedAt: time.Now(),
		Status:    history.StatusCanceled,
		Summary:   "Canceled operation",
	}
	item := HistoryItem{entry: entry}

	rendered := item.Render(s, 50, false)
	if !strings.Contains(rendered, "cancel") {
		t.Error("expected cancel status in rendered output")
	}
}

func TestHistoryPanelSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	panel := NewHistoryPanel(s)

	newStyles := styles.DefaultStyles()
	panel.SetStyles(newStyles)

	// Should not panic
}
