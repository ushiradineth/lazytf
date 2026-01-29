package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/utils"
)

// Modal-related methods for Model

type helpRow struct {
	keys string
	desc string
}

type helpSection struct {
	title string
	rows  []helpRow
}

func (m *Model) updateHelpModalContent() {
	if m.helpModal == nil {
		return
	}

	sections := helpSections(m.executionMode)
	items := helpItems(sections)
	// Add footer item
	items = append(items, components.HelpItem{Key: "esc", Description: "close", IsHeader: false})

	m.helpModal.SetTitle("Keybinds")
	m.helpModal.SetItems(items)
	m.helpModal.Show()
}

// helpItems converts help sections to HelpItem slice for item selection mode.
func helpItems(sections []helpSection) []components.HelpItem {
	// Calculate total items: headers + rows + blank lines between sections
	totalItems := 0
	for _, section := range sections {
		totalItems += 1 + len(section.rows) // header + rows
	}
	totalItems += len(sections) - 1 // blank lines between sections

	items := make([]components.HelpItem, 0, totalItems)
	for i, section := range sections {
		// Add section header
		items = append(items, components.HelpItem{
			Key:      section.title,
			IsHeader: true,
		})
		// Add section rows
		for _, row := range section.rows {
			items = append(items, components.HelpItem{
				Key:         row.keys,
				Description: row.desc,
				IsHeader:    false,
			})
		}
		// Add empty line between sections (except last)
		if i < len(sections)-1 {
			items = append(items, components.HelpItem{
				Key:      "",
				IsHeader: true, // Use header style for blank lines (no selection)
			})
		}
	}
	return items
}

func helpSections(executionMode bool) []helpSection {
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

	if executionMode {
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

	return sections
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

// Theme modal methods

// availableThemes returns the list of available theme names.
var availableThemes = []string{
	"default",
	"terraform-cloud",
	"monokai",
	"nord",
	"github-dark",
}

// themeDisplayName returns a user-friendly display name for a theme.
func themeDisplayName(name string) string {
	switch name {
	case "default":
		return "Default"
	case "terraform-cloud":
		return "Terraform Cloud"
	case "monokai":
		return "Monokai"
	case "nord":
		return "Nord"
	case "github-dark":
		return "GitHub Dark"
	default:
		return name
	}
}

// toggleThemeModal toggles the theme selection modal.
func (m *Model) toggleThemeModal() (tea.Model, tea.Cmd) {
	if m.modalState == ModalTheme {
		// Restore original styles if closing without selection
		if m.originalStyles != nil {
			m.applyStyles(m.originalStyles)
			m.originalStyles = nil
			m.previewThemeName = ""
		}
		m.modalState = ModalNone
		return m, nil
	}
	m.modalState = ModalTheme
	// Save current styles for potential revert
	m.originalStyles = m.styles
	m.previewThemeName = m.styles.Theme.Name
	m.updateThemeModalContent()
	return m, nil
}

// updateThemeModalContent populates the theme modal with available themes.
func (m *Model) updateThemeModalContent() {
	if m.themeModal == nil {
		return
	}

	currentTheme := m.styles.Theme.Name
	var items []components.HelpItem

	for _, themeName := range availableThemes {
		displayName := themeDisplayName(themeName)
		if themeName == currentTheme {
			displayName += " (current)"
		}
		items = append(items, components.HelpItem{
			Key:         displayName,
			Description: "",
			IsHeader:    false,
		})
	}

	// Add footer
	items = append(items, components.HelpItem{Key: "", IsHeader: true})
	items = append(items, components.HelpItem{Key: "enter", Description: "select theme", IsHeader: false})
	items = append(items, components.HelpItem{Key: "esc", Description: "cancel", IsHeader: false})

	m.themeModal.SetTitle("Select Theme")
	m.themeModal.SetItems(items)
	m.themeModal.SetSize(m.width, m.height)
	m.themeModal.Show()
}

// handleModalThemeKey handles key events when the theme modal is active.
func (m *Model) handleModalThemeKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.modalState != ModalTheme {
		return false, nil
	}

	switch msg.String() {
	case "q", consts.KeyCtrlC:
		m.quitting = true
		return true, tea.Quit
	case consts.KeyEsc, "T":
		// Cancel and restore original styles
		if m.originalStyles != nil {
			m.applyStyles(m.originalStyles)
			m.originalStyles = nil
			m.previewThemeName = ""
		}
		m.modalState = ModalNone
		return true, nil
	case "j", consts.KeyDown:
		if m.themeModal != nil {
			m.themeModal.ScrollDown()
			m.previewSelectedTheme()
		}
		return true, nil
	case "k", "up":
		if m.themeModal != nil {
			m.themeModal.ScrollUp()
			m.previewSelectedTheme()
		}
		return true, nil
	case consts.KeyEnter, " ":
		return true, m.confirmThemeSelection()
	default:
		return true, nil
	}
}

// previewSelectedTheme applies the currently selected theme as a preview.
func (m *Model) previewSelectedTheme() {
	if m.themeModal == nil {
		return
	}

	// Get selected index from modal
	selectedIdx := m.themeModal.GetSelectedIndex()
	if selectedIdx < 0 || selectedIdx >= len(availableThemes) {
		return
	}

	themeName := availableThemes[selectedIdx]
	if themeName == m.previewThemeName {
		return // Already previewing this theme
	}

	theme, err := styles.ResolveTheme(themeName)
	if err != nil {
		return
	}

	m.previewThemeName = themeName
	newStyles := styles.NewStyles(theme)
	m.applyStyles(newStyles)
}

// confirmThemeSelection confirms the selected theme and persists it.
func (m *Model) confirmThemeSelection() tea.Cmd {
	if m.themeModal == nil {
		return nil
	}

	selectedIdx := m.themeModal.GetSelectedIndex()
	if selectedIdx < 0 || selectedIdx >= len(availableThemes) {
		return nil
	}

	themeName := availableThemes[selectedIdx]
	theme, err := styles.ResolveTheme(themeName)
	if err != nil {
		return m.toastError("Failed to apply theme: " + err.Error())
	}

	// Apply the theme
	newStyles := styles.NewStyles(theme)
	m.applyStyles(newStyles)

	// Persist to config
	if m.config != nil {
		m.config.Theme.Name = themeName
		if m.configManager != nil {
			if err := m.configManager.Save(*m.config); err != nil {
				m.modalState = ModalNone
				m.originalStyles = nil
				m.previewThemeName = ""
				return m.toastError("Theme applied but failed to save: " + err.Error())
			}
		}
	}

	// Clear modal state
	m.modalState = ModalNone
	m.originalStyles = nil
	m.previewThemeName = ""

	return m.toastSuccess("Theme changed to " + themeDisplayName(themeName))
}

// applyStyles updates all components with new styles.
func (m *Model) applyStyles(newStyles *styles.Styles) {
	if newStyles == nil {
		return
	}

	m.styles = newStyles

	// Update all components that use styles
	if m.resourceList != nil {
		m.resourceList.SetStyles(newStyles)
	}
	if m.diffViewer != nil {
		m.diffViewer.SetStyles(newStyles)
	}
	if m.helpModal != nil {
		m.helpModal.SetStyles(newStyles)
	}
	if m.themeModal != nil {
		m.themeModal.SetStyles(newStyles)
	}
	if m.toast != nil {
		m.toast.SetStyles(newStyles)
	}
	if m.environmentPanel != nil {
		m.environmentPanel.SetStyles(newStyles)
	}
	if m.historyPanel != nil {
		m.historyPanel.SetStyles(newStyles)
	}
	if m.diagnosticsPanel != nil {
		m.diagnosticsPanel.SetStyles(newStyles)
	}
	if m.mainArea != nil {
		m.mainArea.SetStyles(newStyles)
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.SetStyles(newStyles)
	}
	if m.applyView != nil {
		m.applyView.SetStyles(newStyles)
	}
	if m.planView != nil {
		m.planView.SetStyles(newStyles)
	}
	if m.configView != nil {
		m.configView.SetStyles(newStyles)
	}
	if m.stateListView != nil {
		m.stateListView.SetStyles(newStyles)
	}
	if m.stateShowView != nil {
		m.stateShowView.SetStyles(newStyles)
	}
	if m.stateListContent != nil {
		m.stateListContent.SetStyles(newStyles)
	}
}
