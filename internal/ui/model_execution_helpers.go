package ui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

// handleRequestApply handles the RequestApplyMsg by showing confirmation modal.
func (m *Model) handleRequestApply() (tea.Model, tea.Cmd, bool) {
	if m.planRunning || m.applyRunning {
		if m.toast != nil {
			return m, m.toast.ShowInfo("Operation already in progress"), true
		}
		return m, nil, true
	}
	if m.plan == nil {
		if m.toast != nil {
			return m, m.toast.ShowError("No plan loaded; run terraform plan first"), true
		}
		return m, nil, true
	}
	if m.targetModeEnabled {
		targets := m.currentTargetSelection()
		if len(targets) == 0 {
			cmd := m.toastError("Target mode enabled but no resources selected")
			return m, cmd, true
		}
		sig := targetSelectionSignature(targets)
		if sig != m.targetPlanPinned {
			m.pendingTargetApply = true
			m.pendingTargetSig = sig
			message := "Target mode requires a targeted plan for the current selection before apply.\n\n" +
				"Run targeted plan now, then confirm apply?"
			m.showConfirmModal("Target plan required", message, "Yes, run plan", m.deferConfirmCommand(requestPlan))
			return m, nil, true
		}
	}
	m.showConfirmApplyModal()
	return m, nil, true
}

func (m *Model) handleToggleTargetMode() {
	m.targetModeEnabled = !m.targetModeEnabled
	if m.resourceList != nil {
		m.resourceList.SetTargetModeEnabled(m.targetModeEnabled)
	}
	m.invalidateTargetPlanPin()
	if !m.targetModeEnabled {
		m.handleClearTargetSelection()
	}
}

func (m *Model) handleToggleTargetSelection() (tea.Model, tea.Cmd, bool) {
	if !m.targetModeEnabled {
		cmd := m.toastInfo("Enable target mode first with 't'")
		return m, cmd, true
	}
	if m.resourceList == nil {
		return m, nil, true
	}
	if !m.resourceList.ToggleTargetSelectionAtSelected() {
		return m, nil, true
	}
	m.invalidateTargetPlanPin()
	return m, nil, true
}

func (m *Model) handleClearTargetSelection() {
	if m.resourceList != nil {
		m.resourceList.ClearTargetSelection()
	}
	m.invalidateTargetPlanPin()
}

func (m *Model) invalidateTargetPlanPin() {
	m.targetPlanPinned = ""
	m.planTargetSnapshot = ""
	m.clearPendingTargetPlanIntent()
}

func (m *Model) clearPendingTargetPlanIntent() {
	m.pendingTargetApply = false
	m.pendingTargetSig = ""
}

func (m *Model) currentTargetSelection() []string {
	if m.resourceList == nil {
		return nil
	}
	targets := m.resourceList.SelectedTargets()
	if len(targets) == 0 {
		return nil
	}
	sort.Strings(targets)
	return targets
}

func targetSelectionSignature(targets []string) string {
	if len(targets) == 0 {
		return ""
	}
	copyTargets := append([]string{}, targets...)
	sort.Strings(copyTargets)
	return strings.Join(copyTargets, "\n")
}

// handleSwitchResourcesTab handles switching the Resources panel tab.
func (m *Model) handleSwitchResourcesTab(direction int) (tea.Model, tea.Cmd, bool) {
	if !m.canSwitchResourcesTab() {
		return m, nil, true
	}
	m.resourcesActiveTab = nextResourcesTab(m.resourcesActiveTab, direction)

	if m.resourcesController != nil {
		m.resourcesController.SetActiveTab(m.resourcesActiveTab)
	}

	cmd := m.loadStateListIfNeeded()
	return m, cmd, true
}

// handleToggleFilter handles toggling an action filter.
func (m *Model) handleToggleFilter(action terraform.ActionType) {
	switch action {
	case terraform.ActionCreate:
		m.filterCreate = !m.filterCreate
		m.resourceList.SetFilter(action, m.filterCreate)
	case terraform.ActionUpdate:
		m.filterUpdate = !m.filterUpdate
		m.resourceList.SetFilter(action, m.filterUpdate)
	case terraform.ActionDelete:
		m.filterDelete = !m.filterDelete
		m.resourceList.SetFilter(action, m.filterDelete)
	case terraform.ActionReplace:
		m.filterReplace = !m.filterReplace
		m.resourceList.SetFilter(action, m.filterReplace)
	case terraform.ActionNoOp, terraform.ActionRead:
		return
	}
	m.saveFilterPreferences()
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) tea.Model {
	m.width = msg.Width
	m.height = msg.Height

	if !m.ready {
		m.ready = true
	}

	m.updateLayout()
	if m.applyView != nil {
		m.applyView.SetSize(m.width, m.height)
	}
	if m.planView != nil {
		m.planView.SetSize(m.width, m.height)
	}
	return m
}

func (m *Model) handlePlanOutput(msg PlanOutputMsg) (tea.Model, tea.Cmd) {
	if !m.planRunning {
		return m, nil
	}
	if m.applyView != nil {
		m.applyView.AppendLine(msg.Line)
	}
	m.updateStateLockStatus(msg.Line)
	cmd := m.streamPlanOutputCmd()
	return m, cmd
}

func (m *Model) handleApplyOutput(msg ApplyOutputMsg) (tea.Model, tea.Cmd) {
	if !m.applyRunning {
		return m, nil
	}
	if m.applyView != nil {
		m.applyView.AppendLine(msg.Line)
	}
	m.updateStateLockStatus(msg.Line)
	if m.operationState != nil {
		m.operationState.ParseApplyLine(msg.Line)
		if m.resourceList != nil {
			m.resourceList.Refresh()
		}
	}
	cmd := m.streamApplyOutputCmd()
	return m, cmd
}

func (m *Model) handleRefreshOutput(msg RefreshOutputMsg) (tea.Model, tea.Cmd) {
	if !m.refreshRunning {
		return m, nil
	}
	if m.applyView != nil {
		m.applyView.AppendLine(msg.Line)
	}
	m.updateStateLockStatus(msg.Line)
	cmd := m.streamRefreshOutputCmd()
	return m, cmd
}

func (m *Model) updateStateLockStatus(line string) {
	if m.progressIndicator == nil {
		return
	}
	_ = line
	m.progressIndicator.SetDetail("")
}

func (m *Model) canSwitchResourcesTab() bool {
	return m.executionMode && m.panelManager != nil && m.panelManager.GetFocusedPanel() == PanelResources
}

func nextResourcesTab(current, direction int) int {
	if direction < 0 {
		if current > 0 {
			return current - 1
		}
		return 1
	}
	if current < 1 {
		return current + 1
	}
	return 0
}

func (m *Model) loadStateListIfNeeded() tea.Cmd {
	if m.resourcesActiveTab != 1 || m.stateListContent == nil || m.stateListContent.ResourceCount() != 0 {
		return nil
	}
	if m.planRunning || m.applyRunning || m.refreshRunning {
		return m.beginStateList()
	}
	m.stateListContent.SetLoading(true)
	return m.beginStateList()
}
