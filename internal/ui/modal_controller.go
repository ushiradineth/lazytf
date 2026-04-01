package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/ui/components"
)

// Modal-related methods for Model

func (m *Model) updateHelpModalContent() {
	if m.helpModal == nil {
		return
	}

	// Get help items from the keybind registry
	kbItems := m.keybindRegistry.ForHelpModal(m.executionMode)

	// Convert keybinds.HelpItem to components.HelpItem
	items := make([]components.HelpItem, len(kbItems)+1)
	for i, kbItem := range kbItems {
		items[i] = components.HelpItem{
			Key:         kbItem.Key,
			Description: kbItem.Description,
			IsHeader:    kbItem.IsHeader,
		}
	}
	// Add footer item
	items[len(kbItems)] = components.HelpItem{Key: "esc", Description: "close", IsHeader: false}

	m.helpModal.SetTitle("Keybinds")
	m.helpModal.SetItems(items)
	m.helpModal.Show()
}

func (m *Model) updateSettingsModalContent() {
	if m.settingsModal == nil {
		return
	}

	var items []components.HelpItem

	if m.config == nil {
		items = make([]components.HelpItem, 0, 2)
		items = append(items, components.HelpItem{Key: "No configuration loaded.", IsHeader: true})
		items = append(items, components.HelpItem{Key: "", IsHeader: true})
	} else {
		cfg := m.config
		items = make([]components.HelpItem, 0, 20)

		// General section
		items = append(items, components.HelpItem{Key: "General", IsHeader: true})
		items = append(items, components.HelpItem{Key: "default env", Description: fallbackValue(cfg.DefaultEnvironment)})

		// Theme section
		items = append(items, components.HelpItem{Key: "", IsHeader: true})
		items = append(items, components.HelpItem{Key: "Theme", IsHeader: true})
		items = append(items, components.HelpItem{Key: "name", Description: cfg.Theme.Name})

		// Terraform section
		items = append(items, components.HelpItem{Key: "", IsHeader: true})
		items = append(items, components.HelpItem{Key: "Terraform", IsHeader: true})
		items = append(items, components.HelpItem{Key: "binary", Description: fallbackValue(cfg.Terraform.Binary)})
		items = append(items, components.HelpItem{Key: "working dir", Description: fallbackValue(cfg.Terraform.WorkingDir)})
		items = append(items, components.HelpItem{Key: "timeout", Description: cfg.Terraform.Timeout.String()})
		items = append(items, components.HelpItem{Key: "parallelism", Description: strconv.Itoa(cfg.Terraform.Parallelism)})
		items = append(items, components.HelpItem{Key: "default flags", Description: strings.Join(cfg.Terraform.DefaultFlags, " ")})

		// History section
		items = append(items, components.HelpItem{Key: "", IsHeader: true})
		items = append(items, components.HelpItem{Key: "History", IsHeader: true})
		items = append(items, components.HelpItem{Key: "enabled", Description: strconv.FormatBool(cfg.History.Enabled)})
		items = append(items, components.HelpItem{Key: "level", Description: cfg.History.Level})
		items = append(items, components.HelpItem{Key: "path", Description: fallbackValue(cfg.History.Path)})

		// Footer
		items = append(items, components.HelpItem{Key: "", IsHeader: true})
		items = append(items, components.HelpItem{Key: "esc: back", IsHeader: true})
	}

	m.settingsModal.SetTitle("Settings")
	m.settingsModal.SetItems(items)
	m.settingsModal.SetSize(m.width, m.height)
	m.settingsModal.Show()
}

// defaultThemeName is the name of the default theme.
const defaultThemeName = "default"

const themeNameMonochrome = "monochrome"

func fallbackValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return defaultThemeName
	}
	return value
}

// Theme modal methods

// availableThemes returns the list of available theme names.
var availableThemes = styles.BuiltInThemeNames()

// themeDisplayName returns a user-friendly display name for a theme.
func themeDisplayName(name string) string {
	switch name {
	case defaultThemeName:
		return "Default"
	case "terraform-cloud":
		return "Terraform Cloud"
	case "monokai":
		return "Monokai"
	case themeNameMonochrome:
		return "Monochrome"
	case "nord":
		return "Nord"
	case "github-dark":
		return "GitHub Dark"
	default:
		return name
	}
}

// toggleThemeModal toggles the theme selection modal.
func (m *Model) toggleThemeModal() tea.Cmd {
	if m.modalState == ModalTheme {
		// Restore original styles if closing without selection
		if m.originalStyles != nil {
			m.applyStyles(m.originalStyles)
			m.originalStyles = nil
			m.previewThemeName = ""
		}
		m.modalState = ModalNone
		return nil
	}
	m.modalState = ModalTheme
	// Save current styles for potential revert
	m.originalStyles = m.styles
	m.previewThemeName = m.styles.Theme.Name
	m.updateThemeModalContent()
	return nil
}

// updateThemeModalContent populates the theme modal with available themes.
func (m *Model) updateThemeModalContent() {
	if m.themeModal == nil {
		return
	}

	currentTheme := m.styles.Theme.Name
	items := make([]components.HelpItem, 0, len(availableThemes)+2) // Pre-allocate for themes + footer hints
	currentIndex := 0

	for i, themeName := range availableThemes {
		displayName := themeDisplayName(themeName)
		if themeName == currentTheme {
			displayName += " (current)"
			currentIndex = i
		}
		items = append(items, components.HelpItem{
			Key:         displayName,
			Description: "",
			IsHeader:    false,
		})
	}

	// Add footer hints (as headers so they're not selectable)
	items = append(items, components.HelpItem{Key: "", IsHeader: true})
	items = append(items, components.HelpItem{Key: "enter: select  esc: cancel", IsHeader: true})

	m.themeModal.SetTitle("Select Theme")
	m.themeModal.SetItems(items)
	m.themeModal.SetSelectedIndex(currentIndex)
	m.themeModal.SetSize(m.width, m.height)
	m.themeModal.Show()
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

func (m *Model) commitSelectedTheme() tea.Cmd {
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
		return m.toastError(fmt.Sprintf("Theme apply failed: %v", err))
	}

	m.applyStyles(styles.NewStyles(theme))
	m.previewThemeName = themeName
	if m.config != nil {
		m.config.Theme.Name = themeName
		if m.configView != nil {
			m.configView.SetConfig(m.config)
		}
	}

	if m.themeModal != nil {
		m.themeModal.Hide()
	}
	m.modalState = ModalNone
	m.originalStyles = nil
	m.previewThemeName = ""

	if m.config == nil || m.configManager == nil {
		return m.toastInfo("Theme applied for this session only")
	}
	if err := m.configManager.Save(*m.config); err != nil {
		return m.toastInfo(fmt.Sprintf("Theme applied for this session only: %v", err))
	}
	return m.toastSuccess("Theme saved: " + themeDisplayName(themeName))
}

// applyStyles updates all components with new styles.
//
//nolint:gocyclo // Updating all components requires many nil checks
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
	if m.settingsModal != nil {
		m.settingsModal.SetStyles(newStyles)
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
