package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/utils"
)

// Layout-related methods for Model

// renderStatusBar renders the bottom status bar.
func (m *Model) renderStatusBar() string {
	var parts []string

	// Add read-only indicator for non-execution mode
	if !m.executionMode {
		parts = append(parts, m.styles.Dimmed.Render("read-only"))
	}

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

	// Add progress indicator on the right side
	progressView := ""
	if m.progressIndicator != nil {
		progressView = m.progressIndicator.View()
	}

	if progressView != "" {
		progressWidth := lipgloss.Width(progressView)
		statusWidth := lipgloss.Width(statusText)
		gap := m.width - statusWidth - progressWidth
		if gap > 0 {
			statusText = statusText + components.GetPadding(gap) + progressView
		}
	}

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
		parts = append(parts, m.styles.DiffAdd.Render(fmt.Sprintf("+%d", create)))
	}
	if update > 0 {
		parts = append(parts, m.styles.DiffChange.Render(fmt.Sprintf("~%d", update)))
	}
	if deleteCount > 0 {
		parts = append(parts, m.styles.DiffRemove.Render(fmt.Sprintf("-%d", deleteCount)))
	}
	if replace > 0 {
		parts = append(parts, m.styles.DiffChange.Render(fmt.Sprintf("±%d", replace)))
	}

	return fmt.Sprintf("%d changes (%s)", total, strings.Join(parts, " "))
}

func (m *Model) statusHelpText() string {
	hints := m.staticStatusHints()
	if len(hints) == 0 {
		return "?: kbd"
	}
	return strings.Join(hints, " | ")
}

func (m *Model) staticStatusHints() []string {
	hints := make([]string, 0, 6)

	switch m.focusedPanelID() {
	case PanelWorkspace:
		hints = append(hints, "enter: select")
	case PanelResources:
		hints = append(hints, m.staticResourcesPanelHints()...)
	case PanelHistory:
		hints = append(hints, "enter: select")
	case PanelCommandLog:
		hints = append(hints, "L: toggle")
	case PanelMain:
		// No panel-specific hint.
	}

	hints = append(hints, "?: kbd")
	return hints
}

func (m *Model) staticResourcesPanelHints() []string {
	if m.resourcesActiveTab != 0 {
		return []string{"enter: select", "i: init", "I: init upgrade"}
	}
	if !m.executionMode {
		return nil
	}
	hasResources := m.resourceList != nil && m.resourceList.HasResources()
	if !hasResources {
		return []string{"p: plan", "f: format", "v: validate", "i: init", "I: init upgrade"}
	}
	if m.targetModeEnabled {
		return []string{"A: apply", "t: exit target mode", "a: toggle all"}
	}
	return []string{"a: apply", "t: enter target mode", "x: reset plan"}
}

func (m *Model) focusedPanelID() PanelID {
	if m.panelManager == nil {
		return PanelMain
	}
	return m.panelManager.GetFocusedPanel()
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
	emptyLine := components.GetPadding(width) // Cache for reuse
	for i, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			lines[i] = line + components.GetPadding(width-lineWidth)
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
		lines = append(lines, emptyLine)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderLeftColumn(layout LayoutSpec, contentHeight int) string {
	var leftPanels []string

	// Render workspace/environment panel only in execution mode with allocated height
	if m.executionMode && layout.Workspace.Height > 0 {
		workspaceView := ""
		if m.environmentPanel != nil {
			workspaceView = m.environmentPanel.View()
		}
		leftPanels = append(leftPanels, enforceDimensions(workspaceView, layout.LeftColumnWidth, layout.Workspace.Height))
	}

	// Resources panel
	if layout.Resources.Height > 0 {
		leftPanels = append(leftPanels, enforceDimensions(
			m.renderResourcesPanelWithTabs(layout.Resources.Width, layout.Resources.Height),
			layout.LeftColumnWidth,
			layout.Resources.Height,
		))
	}

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

	// Render command log when visible (only in execution mode)
	if m.executionMode && m.panelManager.IsCommandLogVisible() && layout.CommandLog.Height > 0 {
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
	tabParts := make([]string, 0, 2)
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
	contentW := frame.ContentWidth()
	emptyLine := components.GetPadding(contentW)
	for i := range contentHeight {
		if i < len(contentLines) {
			result[i] = components.PadLine(contentLines[i], contentW)
		} else {
			result[i] = emptyLine
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
	if m.showSplit && m.width >= MinSplitWidth {
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
			diagnosticsHeight = utils.MaxInt(diagnosticsHeight, utils.MinInt(m.height/2, MaxFocusedDiagnosticsHeight))
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

	if m.showSplit && m.width >= MinSplitWidth {
		m.updateLegacyLayoutSplit(listHeight, historyHeight, diagnosticsHeight)
		return
	}
	m.updateLegacyLayoutSingle(listHeight, historyHeight, diagnosticsHeight)
}

func (m *Model) updateLegacyLayoutSplit(listHeight, historyHeight, diagnosticsHeight int) {
	listWidth := utils.MaxInt(MinListWidth, int(float64(m.width)*ListWidthRatio))
	diffWidth := m.width - listWidth - 1
	if diffWidth < MinDiffWidth {
		diffWidth = MinDiffWidth
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
