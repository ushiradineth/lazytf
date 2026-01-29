package ui

import (
	"math"

	tea "github.com/charmbracelet/bubbletea"
)

// PanelManager manages panel registration, focus, and layout
type PanelManager struct {
	panels            map[PanelID]Panel
	focusedPanel      PanelID
	commandLogVisible bool
	commandLogFocused bool
	executionMode     bool
}

// NewPanelManager creates a new panel manager
func NewPanelManager() *PanelManager {
	return &PanelManager{
		panels:            make(map[PanelID]Panel),
		focusedPanel:      PanelResources, // Default to resource list
		commandLogVisible: true,           // Command log visible by default
		commandLogFocused: false,
	}
}

// RegisterPanel adds a panel to the manager
func (pm *PanelManager) RegisterPanel(id PanelID, panel Panel) {
	pm.panels[id] = panel
}

// GetPanel retrieves a panel by ID
func (pm *PanelManager) GetPanel(id PanelID) (Panel, bool) {
	panel, ok := pm.panels[id]
	return panel, ok
}

// SetFocus changes the focused panel
func (pm *PanelManager) SetFocus(id PanelID) tea.Cmd {
	// Unfocus current panel
	if currentPanel, ok := pm.panels[pm.focusedPanel]; ok {
		currentPanel.SetFocused(false)
	}

	// Special handling for command log
	if id == PanelCommandLog {
		pm.commandLogFocused = true
		if panel, ok := pm.panels[PanelCommandLog]; ok {
			panel.SetFocused(true)
		}
	} else {
		pm.commandLogFocused = false
		if logPanel, ok := pm.panels[PanelCommandLog]; ok {
			logPanel.SetFocused(false)
		}
	}

	// Focus new panel
	pm.focusedPanel = id
	if newPanel, ok := pm.panels[id]; ok {
		newPanel.SetFocused(true)
	}

	return nil
}

// GetFocusedPanel returns the currently focused panel ID
func (pm *PanelManager) GetFocusedPanel() PanelID {
	if pm.commandLogFocused {
		return PanelCommandLog
	}
	return pm.focusedPanel
}

// CycleFocus cycles to the next panel
func (pm *PanelManager) CycleFocus(reverse bool) tea.Cmd {
	// Define focus order: Workspace -> Resources -> History -> Main
	focusOrder := []PanelID{PanelWorkspace, PanelResources, PanelHistory, PanelMain}

	current := pm.focusedPanel
	if pm.commandLogFocused {
		current = PanelCommandLog
	}

	// Find current index
	currentIdx := -1
	for i, id := range focusOrder {
		if id == current {
			currentIdx = i
			break
		}
	}

	// If command log is focused, go to resources when cycling forward
	if pm.commandLogFocused {
		if reverse {
			return pm.SetFocus(PanelMain)
		}
		return pm.SetFocus(PanelWorkspace)
	}

	// Calculate next index
	var nextIdx int
	if reverse {
		if currentIdx <= 0 {
			nextIdx = len(focusOrder) - 1
		} else {
			nextIdx = currentIdx - 1
		}
	} else {
		if currentIdx < 0 || currentIdx >= len(focusOrder)-1 {
			nextIdx = 0
		} else {
			nextIdx = currentIdx + 1
		}
	}

	return pm.SetFocus(focusOrder[nextIdx])
}

// ToggleCommandLog toggles command log visibility
func (pm *PanelManager) ToggleCommandLog() bool {
	pm.commandLogVisible = !pm.commandLogVisible
	// Also update the panel's visibility
	if panel, ok := pm.panels[PanelCommandLog]; ok {
		if cmdLogPanel, ok := panel.(interface{ SetVisible(bool) }); ok {
			cmdLogPanel.SetVisible(pm.commandLogVisible)
		}
	}
	return pm.commandLogVisible
}

// SetCommandLogVisible sets command log visibility
func (pm *PanelManager) SetCommandLogVisible(visible bool) {
	pm.commandLogVisible = visible
	// Also update the panel's visibility
	if panel, ok := pm.panels[PanelCommandLog]; ok {
		if cmdLogPanel, ok := panel.(interface{ SetVisible(bool) }); ok {
			cmdLogPanel.SetVisible(visible)
		}
	}
}

// IsCommandLogVisible returns whether command log is visible
func (pm *PanelManager) IsCommandLogVisible() bool {
	return pm.commandLogVisible
}

// SetExecutionMode sets whether the app is in execution mode (affects layout)
func (pm *PanelManager) SetExecutionMode(mode bool) {
	pm.executionMode = mode
}

// IsExecutionMode returns whether the app is in execution mode
func (pm *PanelManager) IsExecutionMode() bool {
	return pm.executionMode
}

// CalculateLayout computes layout specifications for all panels
func (pm *PanelManager) CalculateLayout(width, height int) LayoutSpec {
	layout := LayoutSpec{
		FilterBarHeight:   FilterBarHeight,
		StatusBarHeight:   StatusBarHeight,
		CommandLogVisible: pm.commandLogVisible,
	}

	// Calculate available height (subtract filter bar and status bar)
	availableHeight := height - FilterBarHeight - StatusBarHeight

	// Calculate left column width (35% of total, min 40, max 60)
	leftWidth := int(float64(width) * LeftColumnRatio)
	if leftWidth < MinLeftColumnWidth {
		leftWidth = MinLeftColumnWidth
	}
	if leftWidth > MaxLeftColumnWidth {
		leftWidth = MaxLeftColumnWidth
	}
	if leftWidth > width-MinMainAreaWidth {
		leftWidth = width - MinMainAreaWidth
	}

	layout.LeftColumnWidth = leftWidth
	layout.RightColumnWidth = width - leftWidth

	// Calculate vertical layout for left column (uses full available height)
	workspaceHeight, resourcesHeight, historyHeight := pm.leftColumnHeights(availableHeight)

	layout.Workspace = PanelSpec{
		X:      0,
		Y:      FilterBarHeight,
		Width:  leftWidth,
		Height: workspaceHeight,
	}

	layout.Resources = PanelSpec{
		X:      0,
		Y:      FilterBarHeight + workspaceHeight,
		Width:  leftWidth,
		Height: resourcesHeight,
	}

	layout.History = PanelSpec{
		X:      0,
		Y:      FilterBarHeight + workspaceHeight + resourcesHeight,
		Width:  leftWidth,
		Height: historyHeight,
	}

	// Right column: Main area + Command log
	// Calculate command log height if visible
	commandLogHeight := 0
	if pm.commandLogVisible {
		if pm.commandLogFocused {
			// Expanded mode: 50% of right column
			commandLogHeight = int(float64(availableHeight) * CommandLogExpanded)
		} else {
			// Compact mode: fixed height
			commandLogHeight = CommandLogHeight
		}
		if commandLogHeight > availableHeight-10 {
			commandLogHeight = availableHeight - 10
		}
		if commandLogHeight < 3 {
			commandLogHeight = 3
		}
	}

	// Main area gets remaining height in right column
	mainAreaHeight := availableHeight - commandLogHeight

	layout.Main = PanelSpec{
		X:      leftWidth,
		Y:      FilterBarHeight,
		Width:  layout.RightColumnWidth,
		Height: mainAreaHeight,
	}

	// Command log in right column under main area
	if pm.commandLogVisible {
		layout.CommandLog = PanelSpec{
			X:      leftWidth,
			Y:      FilterBarHeight + mainAreaHeight,
			Width:  layout.RightColumnWidth,
			Height: commandLogHeight,
		}
	}

	return layout
}

func (pm *PanelManager) workspaceSelectorActive() bool {
	panel, ok := pm.panels[PanelWorkspace]
	if !ok || panel == nil {
		return false
	}
	type selectorState interface {
		SelectorActive() bool
	}
	if s, ok := panel.(selectorState); ok {
		return s.SelectorActive()
	}
	return false
}

func (pm *PanelManager) leftColumnHeights(panelsHeight int) (int, int, int) {
	if panelsHeight <= 0 {
		return 0, 0, 0
	}

	// When not in execution mode, history panel is not shown
	// Divide space between workspace and resources only
	if !pm.executionMode {
		workspaceRatio := 0.25
		if pm.focusedPanel == PanelWorkspace {
			workspaceRatio = 0.40
		}
		if pm.focusedPanel == PanelWorkspace && pm.workspaceSelectorActive() {
			workspaceRatio = 0.50
		}
		workspaceHeight := int(math.Round(float64(panelsHeight) * workspaceRatio))
		minHeight := 3
		if workspaceHeight < minHeight {
			workspaceHeight = minHeight
		}
		if workspaceHeight > panelsHeight-minHeight {
			workspaceHeight = panelsHeight - minHeight
		}
		resourcesHeight := panelsHeight - workspaceHeight
		return workspaceHeight, resourcesHeight, 0
	}

	type ratios struct {
		workspace float64
		resources float64
		history   float64
	}
	current := ratios{workspace: 0.20, resources: 0.60, history: 0.20}
	switch pm.focusedPanel {
	case PanelWorkspace:
		current = ratios{workspace: 0.60, resources: 0.20, history: 0.20}
	case PanelResources:
		current = ratios{workspace: 0.20, resources: 0.60, history: 0.20}
	case PanelHistory:
		current = ratios{workspace: 0.20, resources: 0.20, history: 0.60}
	}
	if pm.focusedPanel == PanelWorkspace && pm.workspaceSelectorActive() {
		current = ratios{workspace: 0.60, resources: 0.20, history: 0.20}
	}

	minHeight := 3
	workspaceHeight := int(math.Round(float64(panelsHeight) * current.workspace))
	resourcesHeight := int(math.Round(float64(panelsHeight) * current.resources))
	historyHeight := int(math.Round(float64(panelsHeight) * current.history))

	if workspaceHeight < minHeight {
		workspaceHeight = minHeight
	}
	if resourcesHeight < minHeight {
		resourcesHeight = minHeight
	}
	if historyHeight < minHeight {
		historyHeight = minHeight
	}

	sum := workspaceHeight + resourcesHeight + historyHeight
	diff := panelsHeight - sum

	add := func(index int) {
		switch index {
		case 0:
			workspaceHeight++
		case 1:
			resourcesHeight++
		case 2:
			historyHeight++
		}
	}
	subtract := func(index int) bool {
		switch index {
		case 0:
			if workspaceHeight > minHeight {
				workspaceHeight--
				return true
			}
		case 1:
			if resourcesHeight > minHeight {
				resourcesHeight--
				return true
			}
		case 2:
			if historyHeight > minHeight {
				historyHeight--
				return true
			}
		}
		return false
	}

	focusIndex := 1
	switch pm.focusedPanel {
	case PanelWorkspace:
		focusIndex = 0
	case PanelResources:
		focusIndex = 1
	case PanelHistory:
		focusIndex = 2
	}

	for diff > 0 {
		add(focusIndex)
		diff--
	}
	for diff < 0 {
		if !subtract(focusIndex) {
			if focusIndex != 0 && subtract(0) {
				diff++
				continue
			}
			if focusIndex != 1 && subtract(1) {
				diff++
				continue
			}
			if focusIndex != 2 && subtract(2) {
				diff++
				continue
			}
			break
		}
		diff++
	}

	return workspaceHeight, resourcesHeight, historyHeight
}

// HandleNavigation handles navigation keys (number keys, tab)
// Returns true if the key was handled
func (pm *PanelManager) HandleNavigation(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "1":
		return true, pm.SetFocus(PanelWorkspace)
	case "2":
		return true, pm.SetFocus(PanelResources)
	case "3":
		return true, pm.SetFocus(PanelHistory)
	case "0":
		return true, pm.SetFocus(PanelMain)
	case "4":
		// Focus command log if visible
		if pm.commandLogVisible {
			return true, pm.SetFocus(PanelCommandLog)
		}
		// If not visible, show it and focus it
		pm.SetCommandLogVisible(true)
		return true, pm.SetFocus(PanelCommandLog)
	case "tab":
		return true, pm.CycleFocus(false)
	case "shift+tab":
		return true, pm.CycleFocus(true)
	case "L":
		visible := pm.ToggleCommandLog()
		// If we're showing the log, don't focus it yet
		// If we're hiding it and it was focused, move focus to resources
		if !visible && pm.commandLogFocused {
			return true, pm.SetFocus(PanelResources)
		}
		return true, nil
	case "esc":
		// Return to resource list
		if pm.focusedPanel != PanelResources || pm.commandLogFocused {
			return true, pm.SetFocus(PanelResources)
		}
		return false, nil
	}

	return false, nil
}

// UpdatePanelSizes updates all panel sizes based on current layout
func (pm *PanelManager) UpdatePanelSizes(layout LayoutSpec) {
	if panel, ok := pm.panels[PanelWorkspace]; ok && panel != nil {
		panel.SetSize(layout.Workspace.Width, layout.Workspace.Height)
	}
	if panel, ok := pm.panels[PanelResources]; ok && panel != nil {
		panel.SetSize(layout.Resources.Width, layout.Resources.Height)
	}
	if panel, ok := pm.panels[PanelHistory]; ok && panel != nil {
		panel.SetSize(layout.History.Width, layout.History.Height)
	}
	if panel, ok := pm.panels[PanelMain]; ok && panel != nil {
		panel.SetSize(layout.Main.Width, layout.Main.Height)
	}
	if pm.commandLogVisible {
		if panel, ok := pm.panels[PanelCommandLog]; ok && panel != nil {
			panel.SetSize(layout.CommandLog.Width, layout.CommandLog.Height)
		}
	}
}
