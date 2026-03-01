package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

// CommandLogPanel wraps DiagnosticsPanel with a PanelFrame for consistent styling.
type CommandLogPanel struct {
	frame            *PanelFrame
	diagnosticsPanel *DiagnosticsPanel
	styles           *styles.Styles
	height           int
	focused          bool
	visible          bool
}

// NewCommandLogPanel creates a new command log panel.
func NewCommandLogPanel(s *styles.Styles) *CommandLogPanel {
	if s == nil {
		s = styles.DefaultStyles()
	}
	return &CommandLogPanel{
		frame:            NewPanelFrame(s),
		diagnosticsPanel: NewDiagnosticsPanel(s),
		styles:           s,
		visible:          true, // Visible by default
	}
}

// SetSize updates the panel dimensions.
func (c *CommandLogPanel) SetSize(width, height int) {
	c.height = height
	c.frame.SetSize(width, height)
	if c.diagnosticsPanel != nil {
		// Reserve space for border
		innerWidth := max(1, width-2)
		innerHeight := max(1, height-2)
		c.diagnosticsPanel.SetSize(innerWidth, innerHeight)
	}
}

// SetFocused sets the focus state.
func (c *CommandLogPanel) SetFocused(focused bool) {
	c.focused = focused
}

// IsFocused returns whether the panel is focused.
func (c *CommandLogPanel) IsFocused() bool {
	return c.focused
}

// SetVisible sets the visibility state.
func (c *CommandLogPanel) SetVisible(visible bool) {
	c.visible = visible
}

// IsVisible returns whether the panel is visible.
func (c *CommandLogPanel) IsVisible() bool {
	return c.visible
}

// SetDiagnostics updates the diagnostics list.
func (c *CommandLogPanel) SetDiagnostics(items []terraform.Diagnostic) {
	if c.diagnosticsPanel != nil {
		c.diagnosticsPanel.SetDiagnostics(items)
	}
}

// SetLogText sets raw log output.
func (c *CommandLogPanel) SetLogText(text string) {
	if c.diagnosticsPanel != nil {
		c.diagnosticsPanel.SetLogText(text)
	}
}

// SetStyles updates the panel styles.
func (c *CommandLogPanel) SetStyles(s *styles.Styles) {
	c.styles = s
	if c.diagnosticsPanel != nil {
		c.diagnosticsPanel.SetStyles(s)
	}
	if c.frame != nil {
		c.frame.SetStyles(s)
	}
}

// Update handles Bubble Tea messages (implements Panel interface).
func (c *CommandLogPanel) Update(msg tea.Msg) (any, tea.Cmd) {
	if c.diagnosticsPanel == nil {
		return c, nil
	}

	var cmd tea.Cmd
	c.diagnosticsPanel, cmd = c.diagnosticsPanel.Update(msg)
	return c, cmd
}

// HandleKey handles key events.
func (c *CommandLogPanel) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	if !c.focused {
		return false, nil
	}

	// Handle scrolling when focused
	switch msg.String() {
	case "up", "k":
		_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyUp})
		return true, cmd
	case keybinds.KeyDown, "j":
		_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyDown})
		return true, cmd
	case "pgup":
		_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		return true, cmd
	case "pgdown":
		_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		return true, cmd
	case "home":
		_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyHome})
		return true, cmd
	case "end":
		_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnd})
		return true, cmd
	}

	return false, nil
}

// View renders the command log panel.
func (c *CommandLogPanel) View() string {
	if c.styles == nil || c.height <= 0 || !c.visible {
		return ""
	}

	// Get content from diagnostics panel
	content := ""
	if c.diagnosticsPanel != nil {
		content = c.diagnosticsPanel.View()
	}
	if content == "" {
		content = c.styles.Dimmed.Render("No logs available.")
	}

	// Calculate content dimensions
	contentHeight := max(1, c.height-2)

	// Split content into lines
	contentLines := strings.Split(content, "\n")

	scrollPos := 0.0
	thumbSize := 1.0
	showScrollbar := len(contentLines) > contentHeight
	if c.diagnosticsPanel != nil {
		scrollPos, thumbSize, showScrollbar = c.diagnosticsPanel.GetScrollInfo()
	}

	// Configure the frame
	c.frame.SetConfig(PanelFrameConfig{
		PanelID:       "[4]",
		Tabs:          []string{"Command Log"},
		ActiveTab:     0,
		Focused:       c.focused,
		FooterText:    "",
		ShowScrollbar: showScrollbar,
		ScrollPos:     scrollPos,
		ThumbSize:     thumbSize,
	})

	// Pad content lines to fill panel
	result := make([]string, contentHeight)
	contentW := c.frame.ContentWidth()
	emptyLine := GetPadding(contentW)
	for i := range contentHeight {
		if i < len(contentLines) {
			result[i] = c.padLine(contentLines[i], contentW)
		} else {
			result[i] = emptyLine
		}
	}

	return c.frame.RenderWithContent(result)
}

// calculateThumbSize calculates the thumb size for the frame.
func (c *CommandLogPanel) calculateThumbSize(visibleHeight, totalLines int) float64 {
	if totalLines <= visibleHeight || visibleHeight <= 0 {
		return 1.0
	}
	thumbSize := float64(visibleHeight) / float64(totalLines)
	if thumbSize > 1.0 {
		thumbSize = 1.0
	}
	return thumbSize
}

// padLine pads a line to the given width.
func (c *CommandLogPanel) padLine(line string, width int) string {
	runes := []rune(line)
	if len(runes) >= width {
		if len(runes) > width {
			return string(runes[:width])
		}
		return line
	}
	return line + GetPadding(width-len(runes))
}

// GetDiagnosticsPanel returns the underlying diagnostics panel.
func (c *CommandLogPanel) GetDiagnosticsPanel() *DiagnosticsPanel {
	return c.diagnosticsPanel
}

// AppendSessionLog adds a command log entry to the session history.
func (c *CommandLogPanel) AppendSessionLog(label, command, output string) {
	if c.diagnosticsPanel != nil {
		c.diagnosticsPanel.AppendSessionLog(label, command, output)
	}
}

// ClearSessionLogs clears all session log entries.
func (c *CommandLogPanel) ClearSessionLogs() {
	if c.diagnosticsPanel != nil {
		c.diagnosticsPanel.ClearSessionLogs()
	}
}
