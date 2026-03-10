package ui

import (
	"math"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

// PanelManager manages panel registration, focus, and layout.
type PanelManager struct {
	panels            map[PanelID]Panel
	focusedPanel      PanelID
	activeLeftPanel   PanelID
	commandLogVisible bool
	commandLogFocused bool
	executionMode     bool
	focusMode         FocusMode
}

// FocusMode controls which high-level panel layout is active.
type FocusMode int

const (
	FocusModeL1 FocusMode = iota // Default layout
	FocusModeL2                  // 50/50: selected left panel + right stack
	FocusModeL3                  // 100% selected side: left panel OR right stack
)

// NewPanelManager creates a new panel manager.
func NewPanelManager() *PanelManager {
	return &PanelManager{
		panels:            make(map[PanelID]Panel),
		focusedPanel:      PanelResources, // Default to resource list
		activeLeftPanel:   PanelResources,
		commandLogVisible: true, // Command log visible by default
		commandLogFocused: false,
		focusMode:         FocusModeL1,
	}
}

func isLeftPanel(id PanelID) bool {
	return id == PanelWorkspace || id == PanelResources || id == PanelHistory
}

func (pm *PanelManager) historyAvailable() bool {
	panel, ok := pm.panels[PanelHistory]
	return ok && panel != nil
}

func (pm *PanelManager) normalizedLeftPanel() PanelID {
	if !isLeftPanel(pm.activeLeftPanel) {
		return PanelResources
	}
	if pm.activeLeftPanel == PanelHistory && !pm.historyAvailable() {
		return PanelResources
	}
	return pm.activeLeftPanel
}

func (pm *PanelManager) isRightSideFocused() bool {
	focused := pm.GetFocusedPanel()
	return focused == PanelMain || focused == PanelCommandLog
}

// GetFocusMode returns the current focus mode.
func (pm *PanelManager) GetFocusMode() FocusMode {
	return pm.focusMode
}

// NextFocusMode advances to the next focus mode.
func (pm *PanelManager) NextFocusMode() FocusMode {
	pm.focusMode = (pm.focusMode + 1) % 3
	return pm.focusMode
}

// PrevFocusMode moves to the previous focus mode.
func (pm *PanelManager) PrevFocusMode() FocusMode {
	if pm.focusMode == FocusModeL1 {
		pm.focusMode = FocusModeL3
		return pm.focusMode
	}
	pm.focusMode--
	return pm.focusMode
}

// IsPanelVisible reports whether the given panel is currently visible.
//
//nolint:gocognit,gocyclo,funlen // Visibility depends on focus mode, side focus, and execution mode.
func (pm *PanelManager) IsPanelVisible(id PanelID) bool {
	if id == PanelHistory && !pm.historyAvailable() {
		return false
	}

	switch id {
	case PanelResources:
		if !pm.executionMode {
			return true
		}
		switch pm.focusMode {
		case FocusModeL1:
			return true
		case FocusModeL2:
			return pm.normalizedLeftPanel() == PanelResources
		case FocusModeL3:
			return !pm.isRightSideFocused() && pm.normalizedLeftPanel() == PanelResources
		}
		return true
	case PanelWorkspace:
		if !pm.executionMode {
			return false
		}
		switch pm.focusMode {
		case FocusModeL1:
			return true
		case FocusModeL2:
			return pm.normalizedLeftPanel() == PanelWorkspace
		case FocusModeL3:
			return !pm.isRightSideFocused() && pm.normalizedLeftPanel() == PanelWorkspace
		}
		return false
	case PanelHistory:
		if !pm.executionMode {
			return false
		}
		switch pm.focusMode {
		case FocusModeL1:
			return true
		case FocusModeL2:
			return pm.normalizedLeftPanel() == PanelHistory
		case FocusModeL3:
			return !pm.isRightSideFocused() && pm.normalizedLeftPanel() == PanelHistory
		}
		return false
	case PanelMain:
		if !pm.executionMode {
			return false
		}
		switch pm.focusMode {
		case FocusModeL1, FocusModeL2:
			return true
		case FocusModeL3:
			return pm.isRightSideFocused()
		}
		return false
	case PanelCommandLog:
		if !pm.executionMode || !pm.commandLogVisible {
			return false
		}
		switch pm.focusMode {
		case FocusModeL1, FocusModeL2:
			return true
		case FocusModeL3:
			return pm.isRightSideFocused()
		}
		return false
	default:
		return false
	}
}

// RegisterPanel adds a panel to the manager.
func (pm *PanelManager) RegisterPanel(id PanelID, panel Panel) {
	pm.panels[id] = panel
}

// GetPanel retrieves a panel by ID.
func (pm *PanelManager) GetPanel(id PanelID) (Panel, bool) {
	panel, ok := pm.panels[id]
	return panel, ok
}

// SetFocus changes the focused panel.
func (pm *PanelManager) SetFocus(id PanelID) tea.Cmd {
	if id == PanelHistory && !pm.historyAvailable() {
		return nil
	}
	if isLeftPanel(id) {
		pm.activeLeftPanel = id
	}

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
	if newPanel, ok := pm.panels[id]; ok && newPanel != nil {
		newPanel.SetFocused(true)
	}

	return nil
}

// GetFocusedPanel returns the currently focused panel ID.
func (pm *PanelManager) GetFocusedPanel() PanelID {
	if pm.commandLogFocused {
		return PanelCommandLog
	}
	return pm.focusedPanel
}

// CycleFocus cycles to the next panel.
func (pm *PanelManager) CycleFocus(reverse bool) tea.Cmd {
	// Define focus order and keep only visible panels.
	focusOrder := make([]PanelID, 0, 5)
	for _, panelID := range []PanelID{PanelWorkspace, PanelResources, PanelHistory, PanelMain, PanelCommandLog} {
		if pm.IsPanelVisible(panelID) {
			focusOrder = append(focusOrder, panelID)
		}
	}
	if len(focusOrder) == 0 {
		focusOrder = append(focusOrder, PanelResources)
	}

	current := pm.GetFocusedPanel()

	// Find current index
	currentIdx := -1
	for i, id := range focusOrder {
		if id == current {
			currentIdx = i
			break
		}
	}

	nextIdx := nextFocusIndex(reverse, currentIdx, len(focusOrder))
	return pm.SetFocus(focusOrder[nextIdx])
}

func nextFocusIndex(reverse bool, currentIdx, total int) int {
	if total == 0 {
		return 0
	}
	if reverse {
		if currentIdx <= 0 {
			return total - 1
		}
		return currentIdx - 1
	}
	if currentIdx < 0 || currentIdx >= total-1 {
		return 0
	}
	return currentIdx + 1
}

// ToggleCommandLog toggles command log visibility.
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

// SetCommandLogVisible sets command log visibility.
func (pm *PanelManager) SetCommandLogVisible(visible bool) {
	pm.commandLogVisible = visible
	// Also update the panel's visibility
	if panel, ok := pm.panels[PanelCommandLog]; ok {
		if cmdLogPanel, ok := panel.(interface{ SetVisible(bool) }); ok {
			cmdLogPanel.SetVisible(visible)
		}
	}
}

// IsCommandLogVisible returns whether command log is visible.
func (pm *PanelManager) IsCommandLogVisible() bool {
	return pm.commandLogVisible
}

// SetExecutionMode sets whether the app is in execution mode (affects layout).
func (pm *PanelManager) SetExecutionMode(mode bool) {
	pm.executionMode = mode
}

// IsExecutionMode returns whether the app is in execution mode.
func (pm *PanelManager) IsExecutionMode() bool {
	return pm.executionMode
}

// CalculateLayout computes layout specifications for all panels.
//
//nolint:gocognit,gocyclo,funlen // Layout math branches by focus mode and panel visibility states.
func (pm *PanelManager) CalculateLayout(width, height int) LayoutSpec {
	rightStackActive := pm.focusMode != FocusModeL3 || pm.isRightSideFocused()
	commandLogVisible := pm.executionMode && pm.commandLogVisible && rightStackActive

	layout := LayoutSpec{
		FilterBarHeight:   FilterBarHeight,
		StatusBarHeight:   StatusBarHeight,
		CommandLogVisible: commandLogVisible,
	}

	// Calculate available height (subtract filter bar and status bar)
	availableHeight := height - FilterBarHeight - StatusBarHeight

	// Calculate left column width by mode.
	leftWidth := width
	switch pm.focusMode {
	case FocusModeL1:
		leftWidth = int(float64(width) * LeftColumnRatio)
		if leftWidth < MinLeftColumnWidth {
			leftWidth = MinLeftColumnWidth
		}
		if leftWidth > MaxLeftColumnWidth {
			leftWidth = MaxLeftColumnWidth
		}
		if leftWidth > width-MinMainAreaWidth {
			leftWidth = width - MinMainAreaWidth
		}
	case FocusModeL2:
		leftWidth = width / 2
	case FocusModeL3:
		if rightStackActive {
			leftWidth = 0
		} else {
			leftWidth = width
		}
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

	// Right column: Main area + Command log.
	// Height behavior stays identical, width changes by mode.
	commandLogHeight := 0
	mainAreaHeight := 0
	if rightStackActive {
		mainAreaHeight = availableHeight
	}
	if commandLogVisible { //nolint:nestif // Layout calculation requires nested checks
		if pm.commandLogFocused {
			// Full screen mode: command log takes 100%, main area hidden
			commandLogHeight = availableHeight
			mainAreaHeight = 0
		} else {
			// Compact mode: fixed height
			commandLogHeight = CommandLogHeight
			if commandLogHeight > availableHeight-10 {
				commandLogHeight = availableHeight - 10
			}
			if commandLogHeight < 3 {
				commandLogHeight = 3
			}
			mainAreaHeight = availableHeight - commandLogHeight
		}
	}

	layout.Main = PanelSpec{
		X:      leftWidth,
		Y:      FilterBarHeight,
		Width:  layout.RightColumnWidth,
		Height: mainAreaHeight,
	}

	// Command log in right column under main area (only in execution mode)
	if commandLogVisible {
		layout.CommandLog = PanelSpec{
			X:      leftWidth,
			Y:      FilterBarHeight + mainAreaHeight,
			Width:  layout.RightColumnWidth,
			Height: commandLogHeight,
		}
	}

	return layout
}

//nolint:gocognit,gocyclo,funlen // Height allocation intentionally handles multiple mode/availability cases.
func (pm *PanelManager) leftColumnHeights(panelsHeight int) (int, int, int) {
	if panelsHeight <= 0 {
		return 0, 0, 0
	}

	// When not in execution mode, only show resources panel (no workspace or history)
	if !pm.executionMode {
		return 0, panelsHeight, 0
	}

	if pm.focusMode == FocusModeL2 || pm.focusMode == FocusModeL3 {
		switch pm.normalizedLeftPanel() {
		case PanelWorkspace:
			return panelsHeight, 0, 0
		case PanelHistory:
			return 0, 0, panelsHeight
		default:
			return 0, panelsHeight, 0
		}
	}

	// When history panel is not registered, divide space between workspace and resources only
	historyPanel, hasHistory := pm.panels[PanelHistory]
	historyEnabled := hasHistory && historyPanel != nil
	if !historyEnabled {
		// Environment: 5% when inactive, 20% when active
		workspaceRatio := 0.05
		if pm.focusedPanel == PanelWorkspace {
			workspaceRatio = 0.20
		}
		workspaceHeight := int(math.Round(float64(panelsHeight) * workspaceRatio))
		minHeight := MinPanelHeight
		if workspaceHeight < minHeight {
			workspaceHeight = minHeight
		}
		if workspaceHeight > panelsHeight-minHeight {
			workspaceHeight = panelsHeight - minHeight
		}
		resourcesHeight := panelsHeight - workspaceHeight
		return workspaceHeight, resourcesHeight, 0
	}

	// Environment: 5% when inactive, 20% when active
	// History: fixed at 20%
	// Resources: takes the remainder
	type ratios struct {
		workspace float64
		resources float64
		history   float64
	}
	current := ratios{workspace: 0.05, resources: 0.75, history: 0.20}
	if pm.focusedPanel == PanelWorkspace {
		current = ratios{workspace: 0.20, resources: 0.60, history: 0.20}
	}

	minHeight := MinPanelHeight
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

	// Always adjust rounding differences on resources panel to keep other panels stable
	for diff > 0 {
		resourcesHeight++
		diff--
	}
	for diff < 0 {
		if resourcesHeight <= minHeight {
			break
		}
		resourcesHeight--
		diff++
	}

	return workspaceHeight, resourcesHeight, historyHeight
}

// HandleNavigation handles navigation keys (number keys, tab)
// Returns true if the key was handled.
//
//nolint:gocyclo // Navigation logic has inherent complexity from multiple panels
func (pm *PanelManager) HandleNavigation(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "1":
		return true, pm.SetFocus(PanelWorkspace)
	case "2":
		return true, pm.SetFocus(PanelResources)
	case "3":
		// Only focus history if it exists
		if panel, ok := pm.panels[PanelHistory]; ok && panel != nil {
			return true, pm.SetFocus(PanelHistory)
		}
		return true, nil
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
	case keybinds.KeyEsc:
		// Return to resource list
		if pm.focusedPanel != PanelResources || pm.commandLogFocused {
			return true, pm.SetFocus(PanelResources)
		}
		return false, nil
	}

	return false, nil
}

// UpdatePanelSizes updates all panel sizes based on current layout.
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
