package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
	"github.com/ushiradineth/lazytf/internal/utils"
)

// Layout-related methods for Model

// renderStatusBar renders the bottom status bar.
func (m *Model) renderStatusBar() string {
	var parts []string

	// Add workspace/environment info if available
	if m.executionMode && m.envCurrent != "" {
		parts = append(parts, m.styles.Highlight.Render(m.envDisplayName()))
	}

	// Add resource summary
	if m.plan != nil && len(m.plan.Resources) > 0 {
		parts = append(parts, m.resourceSummaryText())
	}

	// Add help text
	parts = append(parts, m.statusHelpText())

	statusText := strings.Join(parts, " │ ")

	// Limit status bar to 1 line to prevent scrolling
	return m.styles.StatusBar.
		Width(m.width).
		MaxHeight(1).
		Render(statusText)
}

// resourceSummaryText returns a summary of resource changes like "+5 ~3 -2 ±2".
func (m *Model) resourceSummaryText() string {
	if m.plan == nil {
		return ""
	}
	create := m.countResourcesByAction(terraform.ActionCreate)
	update := m.countResourcesByAction(terraform.ActionUpdate)
	deleteCount := m.countResourcesByAction(terraform.ActionDelete)
	replace := m.countResourcesByAction(terraform.ActionReplace)

	total := create + update + deleteCount + replace
	if total == 0 {
		return "no changes"
	}

	var parts []string
	if create > 0 {
		parts = append(parts, styles.TfDiffAdd.Render(fmt.Sprintf("+%d", create)))
	}
	if update > 0 {
		parts = append(parts, styles.TfDiffChange.Render(fmt.Sprintf("~%d", update)))
	}
	if deleteCount > 0 {
		parts = append(parts, styles.TfDiffRemove.Render(fmt.Sprintf("-%d", deleteCount)))
	}
	if replace > 0 {
		parts = append(parts, styles.TfDiffChange.Render(fmt.Sprintf("±%d", replace)))
	}

	return fmt.Sprintf("%d changes (%s)", total, strings.Join(parts, " "))
}

func (m *Model) statusHelpText() string {
	if m.keybindRegistry == nil {
		return "?: keybinds | q: quit"
	}
	ctx := m.buildKeybindContext()
	opts := keybinds.DefaultHintOptions()
	return m.keybindRegistry.ForStatusBar(ctx, opts)
}

// countResourcesByAction counts resources of a specific action type.
func (m *Model) countResourcesByAction(action terraform.ActionType) int {
	if m.plan == nil {
		return 0
	}
	count := 0
	for _, resource := range m.plan.Resources {
		if resource.Action == action {
			count++
		}
	}
	return count
}

func (m *Model) renderMainContent() string {
	if m.panelManager == nil {
		return m.renderLegacyMainContent()
	}

	layout := m.panelManager.CalculateLayout(m.width, m.height)
	// Update panel sizes to match the current layout (handles focus-based size changes)
	m.panelManager.UpdatePanelSizes(layout)
	m.syncMainAreaSelection()
	contentHeight := contentAreaHeight(m.height)
	leftColumn := m.renderLeftColumn(layout, contentHeight)
	rightColumn := m.renderRightColumn(layout, contentHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)
}

func (m *Model) syncMainAreaSelection() {
	if m.mainArea != nil && m.resourceList != nil {
		m.mainArea.SetSelectedResource(m.resourceList.GetSelectedResource())
	}
}

func contentAreaHeight(totalHeight int) int {
	contentHeight := totalHeight - StatusBarHeight
	if contentHeight < 1 {
		return 1
	}
	return contentHeight
}

func enforceDimensions(view string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			lines[i] = line + strings.Repeat(" ", width-lineWidth)
			continue
		}
		if lineWidth > width {
			lines[i] = lipgloss.NewStyle().MaxWidth(width).Render(line)
		}
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderLeftColumn(layout LayoutSpec, contentHeight int) string {
	var leftPanels []string

	// Always render the workspace/environment area to fill its allocated space
	if layout.Workspace.Height > 0 {
		workspaceView := ""
		if m.environmentPanel != nil {
			workspaceView = m.environmentPanel.View()
		}
		leftPanels = append(leftPanels, enforceDimensions(workspaceView, layout.LeftColumnWidth, layout.Workspace.Height))
	}

	// Resources panel
	leftPanels = append(leftPanels, enforceDimensions(
		m.renderResourcesPanelWithTabs(layout.Resources.Width, layout.Resources.Height),
		layout.LeftColumnWidth,
		layout.Resources.Height,
	))

	// History panel (only in execution mode with allocated height)
	if m.executionMode && layout.History.Height > 0 {
		historyView := ""
		if m.historyPanel != nil {
			historyView = m.historyPanel.View()
		}
		leftPanels = append(leftPanels, enforceDimensions(historyView, layout.LeftColumnWidth, layout.History.Height))
	}

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, leftPanels...)
	return lipgloss.NewStyle().
		Width(layout.LeftColumnWidth).
		MaxWidth(layout.LeftColumnWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(leftColumn)
}

func (m *Model) renderRightColumn(layout LayoutSpec, contentHeight int) string {
	var rightPanels []string

	// Only render main area if it has height allocated (hidden when command log is focused)
	if layout.Main.Height > 0 && m.mainArea != nil {
		rightPanels = append(rightPanels, enforceDimensions(m.mainArea.View(), layout.RightColumnWidth, layout.Main.Height))
	}

	// Render command log when visible
	if m.panelManager.IsCommandLogVisible() && layout.CommandLog.Height > 0 {
		// Always render the command log space when it's visible to prevent layout gaps.
		// If the panel returns empty, create empty space to fill the reserved height.
		commandLogView := ""
		if m.commandLogPanel != nil {
			commandLogView = m.commandLogPanel.View()
		}
		rightPanels = append(rightPanels, enforceDimensions(commandLogView, layout.RightColumnWidth, layout.CommandLog.Height))
	}

	rightColumn := lipgloss.JoinVertical(lipgloss.Left, rightPanels...)
	return lipgloss.NewStyle().
		Width(layout.RightColumnWidth).
		MaxWidth(layout.RightColumnWidth).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(rightColumn)
}

// renderResourcesPanelWithTabs renders the resources panel with tabs (Resources / State).
func (m *Model) renderResourcesPanelWithTabs(width, height int) string {
	if m.resourceList == nil {
		return ""
	}

	// In non-execution mode, just show the resource list (no tabs)
	if !m.executionMode {
		return m.resourceList.View()
	}

	// Determine which content to show based on active tab
	if m.resourcesActiveTab == 0 {
		// Resources tab - use the resource list's view but we'll modify the title
		// ResourceList already renders with border, so use full dimensions
		m.resourceList.SetSize(width, height)
		content := m.resourceList.View()
		// The resource list already renders with border and title, so return it with modified title
		return m.addTabsToPanel(content, width, []string{"Resources", "State"}, m.resourcesActiveTab)
	}

	// State tab - use PanelFrame properly
	if m.stateListContent == nil {
		m.stateListContent = components.NewStateListContent(m.styles)
		m.stateListContent.OnSelect = func(address string) tea.Cmd {
			return m.beginStateShow(address)
		}
	}

	focused := m.panelManager != nil && m.panelManager.GetFocusedPanel() == PanelResources
	m.stateListContent.SetFocused(focused)

	// Content area is panel size minus borders (2 for border)
	contentWidth := width - 2
	contentHeight := height - 2
	m.stateListContent.SetSize(contentWidth, contentHeight)

	// Get scroll info for the frame
	scrollPos, thumbSize, hasScrollbar := m.stateListContent.GetScrollInfo(contentHeight)

	// Build tab title for the frame
	var tabParts []string
	tabParts = append(tabParts, "Resources")
	tabParts = append(tabParts, "State")

	// Create and configure the frame
	frame := components.NewPanelFrame(m.styles)
	frame.SetSize(width, height)
	frame.SetConfig(components.PanelFrameConfig{
		PanelID:       "[2]",
		Tabs:          tabParts,
		ActiveTab:     1, // State tab is active
		Focused:       focused,
		FooterText:    m.stateListContent.GetFooterText(),
		ShowScrollbar: hasScrollbar,
		ScrollPos:     scrollPos,
		ThumbSize:     thumbSize,
	})

	// Get content lines from StateListContent
	stateView := m.stateListContent.View()
	contentLines := strings.Split(stateView, "\n")

	// Pad content lines to fill panel
	result := make([]string, contentHeight)
	for i := range contentHeight {
		if i < len(contentLines) {
			result[i] = components.PadLine(contentLines[i], frame.ContentWidth())
		} else {
			result[i] = strings.Repeat(" ", frame.ContentWidth())
		}
	}

	return frame.RenderWithContent(result)
}

// addTabsToPanel adds tab indicators to the first line of a panel.
func (m *Model) addTabsToPanel(panel string, width int, tabs []string, activeTab int) string {
	lines := strings.Split(panel, "\n")
	if len(lines) == 0 {
		return panel
	}

	// Build tab title
	focused := m.panelManager != nil && m.panelManager.GetFocusedPanel() == PanelResources
	titleStyle := m.styles.PanelTitle
	if focused {
		titleStyle = m.styles.FocusedPanelTitle
	}

	// Build title: [2] ActiveTab - InactiveTab
	// Active tab gets title color (blue when focused), inactive tabs are white
	var tabParts []string
	for i, tab := range tabs {
		if i == activeTab {
			// Active tab uses title style (blue when focused)
			tabParts = append(tabParts, titleStyle.Render(tab))
		} else {
			// Inactive tabs are white (plain text)
			tabParts = append(tabParts, tab)
		}
	}
	titleRendered := titleStyle.Render("[2]") + " " + strings.Join(tabParts, " - ")

	// Try to replace the title in the first line
	firstLine := lines[0]
	// Find position after first border character
	if len(firstLine) > 2 {
		// Replace title section
		borderStyle := m.styles.Border
		if focused {
			borderStyle = m.styles.FocusedBorder
		}
		if newLine, ok := components.RenderPanelTitleLine(width, borderStyle, titleRendered); ok {
			lines[0] = newLine
		}
	}

	return strings.Join(lines, "\n")
}

func (m *Model) renderLegacyMainContent() string {
	leftContent := m.resourceList.View()
	if m.executionMode && m.showHistory && m.historyPanel != nil && m.historyPanel.View() != "" {
		leftContent = lipgloss.JoinVertical(lipgloss.Left, leftContent, m.historyPanel.View())
	}
	if m.showSplit && m.width >= 100 {
		right := lipgloss.NewStyle().MarginLeft(1).Render(
			m.diffViewer.View(m.resourceList.GetSelectedResource()),
		)
		main := lipgloss.JoinHorizontal(lipgloss.Top, leftContent, right)
		return m.appendDiagnostics(main)
	}
	return m.appendDiagnostics(leftContent)
}

func (m *Model) appendDiagnostics(content string) string {
	if !m.executionMode || m.diagnosticsPanel == nil || m.diagnosticsHeight == 0 {
		return content
	}
	panel := m.diagnosticsPanel.View()
	if panel == "" {
		return content
	}
	return lipgloss.JoinVertical(lipgloss.Left, content, panel)
}

func (m *Model) renderFullScreenCommandLog() string {
	if m.commandLogPanel == nil {
		return m.styles.Dimmed.Render("No logs available")
	}

	// Get diagnostics panel content
	diagPanel := m.commandLogPanel.GetDiagnosticsPanel()
	if diagPanel == nil {
		return m.styles.Dimmed.Render("No logs available")
	}

	// Calculate dimensions for full screen
	// Reserve space for title (1), border (2), and footer (1)
	contentHeight := m.height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}
	contentWidth := m.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	// Update diagnostics panel size to full screen
	diagPanel.SetSize(contentWidth, contentHeight)

	// Get the content
	content := diagPanel.View()

	// Build title
	title := " Command Log (Full Screen) "
	titleRendered := m.styles.FocusedPanelTitle.Render(title)

	// Build footer with help text
	footer := m.styles.Dimmed.Render("Press 'esc' to exit | arrow keys to scroll")

	// Wrap in border
	panel := m.styles.FocusedBorder.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)

	// Add title to top border
	lines := strings.Split(panel, "\n")
	if len(lines) > 0 {
		// Replace first line with title
		if line, ok := components.RenderPanelTitleLine(contentWidth+2, m.styles.FocusedBorder, titleRendered); ok {
			lines[0] = line
		}
	}
	panel = strings.Join(lines, "\n")

	// Join panel and footer
	return lipgloss.JoinVertical(lipgloss.Left, panel, footer)
}

func (m *Model) updateLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Update overlay component sizes
	if m.helpModal != nil {
		m.helpModal.SetSize(m.width, m.height)
	}
	if m.toast != nil {
		m.toast.SetSize(m.width, m.height)
	}

	// Use panel manager if available
	if m.panelManager != nil {
		// Calculate layout using panel manager (it will handle status bar internally)
		layout := m.panelManager.CalculateLayout(m.width, m.height)

		// Update all panel sizes
		m.panelManager.UpdatePanelSizes(layout)
		return
	}

	// Legacy layout code
	m.updateLegacyLayout()
}

func (m *Model) updateLegacyLayout() {
	reserved := lipgloss.Height(m.renderStatusBar())
	listHeight := m.height - reserved
	if listHeight < 1 {
		listHeight = 1
	}
	historyHeight := 0
	if m.executionMode && m.showHistory && m.historyPanel != nil {
		historyHeight = m.historyHeight
		if historyHeight >= listHeight {
			historyHeight = 0
		}
	}
	diagnosticsHeight := 0
	if m.executionMode && m.diagnosticsPanel != nil {
		diagnosticsHeight = m.diagnosticsHeight
		if m.diagnosticsFocused {
			diagnosticsHeight = utils.MaxInt(diagnosticsHeight, utils.MinInt(m.height/2, 12))
		}
		if diagnosticsHeight >= listHeight {
			diagnosticsHeight = 0
		}
	}
	listHeight -= historyHeight
	listHeight -= diagnosticsHeight
	if listHeight < 1 {
		listHeight = 1
	}

	if m.showSplit && m.width >= 100 {
		m.updateLegacyLayoutSplit(listHeight, historyHeight, diagnosticsHeight)
		return
	}
	m.updateLegacyLayoutSingle(listHeight, historyHeight, diagnosticsHeight)
}

func (m *Model) updateLegacyLayoutSplit(listHeight, historyHeight, diagnosticsHeight int) {
	listWidth := utils.MaxInt(40, int(float64(m.width)*0.45))
	diffWidth := m.width - listWidth - 1
	if diffWidth < 20 {
		diffWidth = 20
		listWidth = m.width - diffWidth - 1
	}
	m.resourceList.SetSize(listWidth, listHeight)
	if m.historyPanel != nil {
		m.historyPanel.SetSize(listWidth, historyHeight)
	}
	if m.diagnosticsPanel != nil {
		m.diagnosticsPanel.SetSize(m.width, diagnosticsHeight)
	}
	m.diffViewer.SetSize(diffWidth, listHeight)
}

func (m *Model) updateLegacyLayoutSingle(listHeight, historyHeight, diagnosticsHeight int) {
	m.resourceList.SetSize(m.width, listHeight)
	if m.historyPanel != nil {
		m.historyPanel.SetSize(m.width, historyHeight)
	}
	if m.diagnosticsPanel != nil {
		m.diagnosticsPanel.SetSize(m.width, diagnosticsHeight)
	}
	m.diffViewer.SetSize(m.width, listHeight)
}
