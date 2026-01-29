package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/utils"
)

// Modal-related methods for Model

func (m *Model) updateHelpModalContent() {
	if m.helpModal == nil {
		return
	}

	type helpRow struct {
		keys string
		desc string
	}
	type helpSection struct {
		title string
		rows  []helpRow
	}

	sections := []helpSection{
		{
			title: "Panel Navigation",
			rows: []helpRow{
				{keys: "1", desc: "focus workspace panel"},
				{keys: "2", desc: "focus resource list"},
				{keys: "3", desc: "focus history"},
				{keys: "0", desc: "focus main area"},
				{keys: "4", desc: "focus command log (enter for full screen)"},
				{keys: "tab", desc: "cycle panels"},
				{keys: "L", desc: "toggle command log"},
			},
		},
		{
			title: "Navigation",
			rows: []helpRow{
				{keys: "↑/↓ or j/k", desc: "move selection"},
				{keys: "enter/space", desc: "toggle group"},
				{keys: "t", desc: "toggle all groups"},
			},
		},
		{
			title: "Filters",
			rows: []helpRow{
				{keys: "c", desc: "toggle create"},
				{keys: "u", desc: "toggle update"},
				{keys: "d", desc: "toggle delete"},
				{keys: "r", desc: "toggle replace"},
			},
		},
		{
			title: "Search",
			rows: []helpRow{
				{keys: "/", desc: "focus search"},
				{keys: "esc", desc: "clear search"},
			},
		},
		{
			title: "General",
			rows: []helpRow{
				{keys: "1 then e", desc: "select environment"},
				{keys: ",", desc: "open settings"},
				{keys: "?", desc: "toggle keybinds"},
				{keys: "q or ctrl+c", desc: "quit"},
			},
		},
	}
	if m.executionMode {
		sections = append(sections, helpSection{
			title: "Execution",
			rows: []helpRow{
				{keys: "p", desc: "run terraform plan"},
				{keys: "f", desc: "refresh state"},
				{keys: "v", desc: "validate configuration"},
				{keys: "F", desc: "format code (fmt)"},
				{keys: "a", desc: "confirm apply"},
				{keys: "h", desc: "toggle history panel"},
				{keys: "tab", desc: "focus history panel"},
				{keys: "ctrl+c", desc: "cancel running command"},
				{keys: "s", desc: "toggle status column"},
				{keys: "C", desc: "toggle compact progress view"},
				{keys: "D", desc: "focus logs panel"},
				{keys: "[/]", desc: "switch tabs in panel"},
			},
		})
	}

	keyWidth := 0
	for _, section := range sections {
		for _, row := range section.rows {
			if len(row.keys) > keyWidth {
				keyWidth = len(row.keys)
			}
		}
	}
	if keyWidth < 8 {
		keyWidth = 8
	}

	// Estimate number of lines: 2 per row (key-desc + empty) + section titles
	totalRows := 0
	for _, section := range sections {
		totalRows += len(section.rows) + 2 // rows + title + separator
	}
	lines := make([]string, 0, totalRows+1) // +1 for closing help text
	for _, section := range sections {
		lines = append(lines, m.styles.Highlight.Render(section.title))
		for _, row := range section.rows {
			keyText := fmt.Sprintf("%-*s", keyWidth, row.keys)
			left := m.styles.HelpKey.Render(keyText)
			right := m.styles.HelpValue.Render(row.desc)
			lines = append(lines, left+"  "+right)
		}
		lines = append(lines, "")
	}
	lines = append(lines, m.styles.Dimmed.Render("esc: close"))

	m.helpModal.SetTitle("Keybinds")
	m.helpModal.SetContent(strings.TrimRight(strings.Join(lines, "\n"), "\n"))
	m.helpModal.Show()
}

func (m *Model) renderSettings() string {
	if m.styles == nil {
		return ""
	}
	if m.configView == nil {
		lines := []string{
			m.styles.Highlight.Render("Settings"),
			"",
			"No configuration loaded.",
			"",
			"esc: back",
		}
		content := strings.TrimRight(strings.Join(lines, "\n"), "\n")
		box := m.styles.Border.Width(utils.MinInt(50, m.width-4)).Render(content)
		if m.width == 0 || m.height == 0 {
			return box
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	m.configView.SetConfig(m.config)
	return m.configView.View()
}
