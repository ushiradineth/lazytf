package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
)

// ResourcesPanelController manages the Resources panel, which has two tabs:
// Resources (index 0) and State (index 1). It routes keyboard input to the
// appropriate tab and handles tab-specific operations like execution commands.
type ResourcesPanelController struct {
	resourceList *components.ResourceList
	stateList    *components.StateListContent
	activeTab    int // 0=Resources, 1=State
}

// NewResourcesPanelController creates a new resources panel controller.
func NewResourcesPanelController(resourceList *components.ResourceList) *ResourcesPanelController {
	return &ResourcesPanelController{
		resourceList: resourceList,
		activeTab:    0,
	}
}

// SetStateListContent sets the state list content for the State tab.
func (c *ResourcesPanelController) SetStateListContent(stateList *components.StateListContent) {
	c.stateList = stateList
}

// SetActiveTab sets the active tab index.
func (c *ResourcesPanelController) SetActiveTab(tab int) {
	if tab >= 0 && tab <= 1 {
		c.activeTab = tab
	}
}

// GetActiveTab returns the active tab index.
func (c *ResourcesPanelController) GetActiveTab() int {
	return c.activeTab
}

// HandleKey handles keyboard input for the Resources panel.
// Returns (handled, cmd) where handled indicates if the key was consumed.
func (c *ResourcesPanelController) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Tab switch keys work on both tabs - delegate to model via message
	switch msg.String() {
	case "[":
		return true, switchResourcesTab(-1)
	case "]":
		return true, switchResourcesTab(1)
	}

	// Delegate to active tab
	if c.activeTab == 0 {
		return c.handleResourcesTabKey(msg)
	}
	return c.handleStateTabKey(msg)
}

// handleResourcesTabKey handles keys when the Resources tab is active.
func (c *ResourcesPanelController) handleResourcesTabKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	// Execution actions - only work on Resources tab
	case "p":
		return true, requestPlan()
	case "a":
		return true, requestApply()
	case "f":
		return true, requestRefresh()
	case "v":
		return true, requestValidate()
	case "F":
		return true, requestFormat()

	// Filter toggles - only work on Resources tab
	case "c":
		return true, toggleFilter(terraform.ActionCreate)
	case "u":
		return true, toggleFilter(terraform.ActionUpdate)
	case "d":
		return true, toggleFilter(terraform.ActionDelete)
	case "r":
		return true, toggleFilter(terraform.ActionReplace)
	case "s":
		return true, toggleStatus()
	}

	// Navigation keys - delegate to ResourceList
	if c.resourceList != nil {
		return c.resourceList.HandleKey(msg)
	}
	return false, nil
}

// handleStateTabKey handles keys when the State tab is active.
func (c *ResourcesPanelController) handleStateTabKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if c.stateList != nil {
		return c.stateList.HandleKey(msg)
	}
	return false, nil
}

// Command factory functions that return tea.Cmd.

func requestPlan() tea.Cmd {
	return func() tea.Msg {
		return RequestPlanMsg{}
	}
}

func requestApply() tea.Cmd {
	return func() tea.Msg {
		return RequestApplyMsg{}
	}
}

func requestRefresh() tea.Cmd {
	return func() tea.Msg {
		return RequestRefreshMsg{}
	}
}

func requestValidate() tea.Cmd {
	return func() tea.Msg {
		return RequestValidateMsg{}
	}
}

func requestFormat() tea.Cmd {
	return func() tea.Msg {
		return RequestFormatMsg{}
	}
}

func toggleFilter(action terraform.ActionType) tea.Cmd {
	return func() tea.Msg {
		return ToggleFilterMsg{Action: action}
	}
}

func toggleAllGroups() tea.Cmd {
	return func() tea.Msg {
		return ToggleAllGroupsMsg{}
	}
}

func toggleStatus() tea.Cmd {
	return func() tea.Msg {
		return ToggleStatusMsg{}
	}
}

func switchResourcesTab(direction int) tea.Cmd {
	return func() tea.Msg {
		return SwitchResourcesTabMsg{Direction: direction}
	}
}
