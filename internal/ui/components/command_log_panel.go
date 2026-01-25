package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

// CommandLogPanel wraps DiagnosticsPanel and adds Panel interface support
type CommandLogPanel struct {
	diagnosticsPanel *DiagnosticsPanel
	styles           *styles.Styles
	width            int
	height           int
	focused          bool
	visible          bool
}

// NewCommandLogPanel creates a new command log panel
func NewCommandLogPanel(s *styles.Styles) *CommandLogPanel {
	return &CommandLogPanel{
		diagnosticsPanel: NewDiagnosticsPanel(s),
		styles:           s,
		visible:          false,
	}
}

// SetSize updates the panel dimensions
func (c *CommandLogPanel) SetSize(width, height int) {
	c.width = width
	c.height = height
	if c.diagnosticsPanel != nil {
		// Reserve space for border
		innerHeight := height - 2
		if innerHeight < 1 {
			innerHeight = 1
		}
		c.diagnosticsPanel.SetSize(width-2, innerHeight)
	}
}

// SetFocused sets the focus state
func (c *CommandLogPanel) SetFocused(focused bool) {
	c.focused = focused
}

// IsFocused returns whether the panel is focused
func (c *CommandLogPanel) IsFocused() bool {
	return c.focused
}

// SetVisible sets the visibility state
func (c *CommandLogPanel) SetVisible(visible bool) {
	c.visible = visible
}

// IsVisible returns whether the panel is visible
func (c *CommandLogPanel) IsVisible() bool {
	return c.visible
}

// SetDiagnostics updates the diagnostics list
func (c *CommandLogPanel) SetDiagnostics(items []terraform.Diagnostic) {
	if c.diagnosticsPanel != nil {
		c.diagnosticsPanel.SetDiagnostics(items)
	}
}

// SetLogText sets raw log output
func (c *CommandLogPanel) SetLogText(text string) {
	if c.diagnosticsPanel != nil {
		c.diagnosticsPanel.SetLogText(text)
	}
}

// SetParsedText sets the parsed summary text
func (c *CommandLogPanel) SetParsedText(text string) {
	if c.diagnosticsPanel != nil {
		c.diagnosticsPanel.SetParsedText(text)
	}
}

// SetShowRaw toggles between raw logs and parsed summary
func (c *CommandLogPanel) SetShowRaw(show bool) {
	if c.diagnosticsPanel != nil {
		c.diagnosticsPanel.SetShowRaw(show)
	}
}

// Update handles Bubble Tea messages (implements Panel interface)
func (c *CommandLogPanel) Update(msg tea.Msg) (any, tea.Cmd) {
	if c.diagnosticsPanel == nil {
		return c, nil
	}

	var cmd tea.Cmd
	c.diagnosticsPanel, cmd = c.diagnosticsPanel.Update(msg)
	return c, cmd
}

// HandleKey handles key events
func (c *CommandLogPanel) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	if !c.focused {
		return false, nil
	}

	// Handle scrolling when focused
	switch msg.String() {
	case "up", "k":
		_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyUp})
		return true, cmd
	case "down", "j":
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

// View renders the command log panel
func (c *CommandLogPanel) View() string {
	if c.styles == nil || c.height <= 0 || !c.visible {
		return ""
	}

	// Determine border style based on focus
	borderStyle := c.styles.Border
	titleStyle := c.styles.PanelTitle
	if c.focused {
		borderStyle = c.styles.FocusedBorder
		titleStyle = c.styles.FocusedPanelTitle
	}

	// Get content from diagnostics panel
	content := ""
	if c.diagnosticsPanel != nil {
		content = c.diagnosticsPanel.View()
	} else {
		content = c.styles.Dimmed.Render("No logs available")
	}

	// Build panel with border
	panel := borderStyle.
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Width(c.width - 2).
		Height(c.height - 2).
		Render(content)

	// Add title to border
	title := " [4] Command Log "
	title = titleStyle.Render(title)

	lines := strings.Split(panel, "\n")
	if len(lines) > 0 {
		firstLine := lines[0]
		// Insert title after the first border character
		// Note: title may contain ANSI codes, so we need to be careful with length
		if len(firstLine) > len(title)+1 {
			lines[0] = string(firstLine[0]) + title + firstLine[len(title)+1:]
		} else if len(firstLine) > 1 {
			// If not enough space, just append title and truncate
			lines[0] = string(firstLine[0]) + title
		}
	}

	return strings.Join(lines, "\n")
}

// GetDiagnosticsPanel returns the underlying diagnostics panel
func (c *CommandLogPanel) GetDiagnosticsPanel() *DiagnosticsPanel {
	return c.diagnosticsPanel
}
