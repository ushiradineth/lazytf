package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

// buildKeybindContext creates a keybinds.Context from the current model state.
func (m *Model) buildKeybindContext() *keybinds.Context {
	ctx := keybinds.NewContext()

	// Mode state
	ctx.ExecutionMode = m.executionMode
	ctx.OperationRunning = m.planRunning || m.applyRunning || m.refreshRunning
	ctx.PlanRunning = m.planRunning
	ctx.ApplyRunning = m.applyRunning
	ctx.RefreshRunning = m.refreshRunning

	// Focus state
	if m.panelManager != nil {
		ctx.FocusedPanel = convertPanelID(m.panelManager.GetFocusedPanel())
	}

	// Modal state
	ctx.ActiveModal = convertModalState(m.modalState)

	// View state
	ctx.CurrentView = convertExecView(m.execView)

	// Tab state
	ctx.ResourcesActiveTab = m.resourcesActiveTab

	// Selector state
	if m.environmentPanel != nil {
		ctx.SelectorActive = m.environmentPanel.SelectorActive()
	}

	return ctx
}

// convertPanelID converts PanelID to keybinds.PanelID.
func convertPanelID(p PanelID) keybinds.PanelID {
	switch p {
	case PanelWorkspace:
		return keybinds.PanelWorkspace
	case PanelResources:
		return keybinds.PanelResources
	case PanelHistory:
		return keybinds.PanelHistory
	case PanelMain:
		return keybinds.PanelMain
	case PanelCommandLog:
		return keybinds.PanelCommandLog
	default:
		return keybinds.PanelNone
	}
}

// convertModalState converts ModalState to keybinds.ModalID.
func convertModalState(s ModalState) keybinds.ModalID {
	switch s {
	case ModalHelp:
		return keybinds.ModalHelp
	case ModalSettings:
		return keybinds.ModalSettings
	case ModalConfirmApply:
		return keybinds.ModalConfirmApply
	case ModalTheme:
		return keybinds.ModalTheme
	default:
		return keybinds.ModalNone
	}
}

// convertExecView converts executionView to keybinds.ViewID.
func convertExecView(v executionView) keybinds.ViewID {
	switch v {
	case viewPlanOutput, viewApplyOutput:
		return keybinds.ViewPlanOutput
	case viewCommandLog:
		return keybinds.ViewCommandLog
	case viewStateList:
		return keybinds.ViewStateList
	case viewStateShow:
		return keybinds.ViewStateShow
	default:
		return keybinds.ViewMain
	}
}

// registerKeybindHandlers registers all action handlers with the keybind registry.
func (m *Model) registerKeybindHandlers() {
	if m.keybindRegistry == nil {
		return
	}

	r := m.keybindRegistry

	// Global actions
	r.RegisterHandler(keybinds.ActionQuit, m.handleActionQuit)
	r.RegisterHandler(keybinds.ActionCancelOp, m.handleActionCancelOp)
	r.RegisterHandler(keybinds.ActionToggleHelp, m.handleActionToggleHelp)
	r.RegisterHandler(keybinds.ActionToggleConfig, m.handleActionToggleConfig)
	r.RegisterHandler(keybinds.ActionToggleTheme, m.handleActionToggleTheme)

	// Panel navigation
	r.RegisterHandler(keybinds.ActionFocusWorkspace, m.handleActionFocusWorkspace)
	r.RegisterHandler(keybinds.ActionFocusResources, m.handleActionFocusResources)
	r.RegisterHandler(keybinds.ActionFocusHistory, m.handleActionFocusHistory)
	r.RegisterHandler(keybinds.ActionFocusMain, m.handleActionFocusMain)
	r.RegisterHandler(keybinds.ActionFocusCommandLog, m.handleActionFocusCommandLog)
	r.RegisterHandler(keybinds.ActionCycleFocus, m.handleActionCycleFocus)
	r.RegisterHandler(keybinds.ActionCycleFocusBack, m.handleActionCycleFocusBack)
	r.RegisterHandler(keybinds.ActionToggleLog, m.handleActionToggleLog)
	r.RegisterHandler(keybinds.ActionEscapeBack, m.handleActionEscapeBack)
	r.RegisterHandler(keybinds.ActionToggleHistory, m.handleActionToggleHistory)

	// Execution actions
	r.RegisterHandler(keybinds.ActionPlan, m.handleActionPlan)
	r.RegisterHandler(keybinds.ActionApply, m.handleActionApply)
	r.RegisterHandler(keybinds.ActionRefresh, m.handleActionRefresh)
	r.RegisterHandler(keybinds.ActionValidate, m.handleActionValidate)
	r.RegisterHandler(keybinds.ActionFormat, m.handleActionFormat)

	// Filter actions
	r.RegisterHandler(keybinds.ActionToggleCreate, m.handleActionToggleCreate)
	r.RegisterHandler(keybinds.ActionToggleUpdate, m.handleActionToggleUpdate)
	r.RegisterHandler(keybinds.ActionToggleDelete, m.handleActionToggleDelete)
	r.RegisterHandler(keybinds.ActionToggleReplace, m.handleActionToggleReplace)
	r.RegisterHandler(keybinds.ActionToggleAllGroups, m.handleActionToggleAllGroups)
	r.RegisterHandler(keybinds.ActionToggleStatus, m.handleActionToggleStatus)

	// Tab actions
	r.RegisterHandler(keybinds.ActionSwitchTabPrev, m.handleActionSwitchTabPrev)
	r.RegisterHandler(keybinds.ActionSwitchTabNext, m.handleActionSwitchTabNext)

	// Navigation actions
	r.RegisterHandler(keybinds.ActionMoveUp, m.handleActionMoveUp)
	r.RegisterHandler(keybinds.ActionMoveDown, m.handleActionMoveDown)
	r.RegisterHandler(keybinds.ActionSelect, m.handleActionSelect)
	r.RegisterHandler(keybinds.ActionScrollUp, m.handleActionScrollUp)
	r.RegisterHandler(keybinds.ActionScrollDown, m.handleActionScrollDown)

	// Environment actions
	r.RegisterHandler(keybinds.ActionSelectEnv, m.handleActionSelectEnv)

	// Modal actions
	r.RegisterHandler(keybinds.ActionConfirmYes, m.handleActionConfirmYes)
	r.RegisterHandler(keybinds.ActionConfirmNo, m.handleActionConfirmNo)
}

// Handler implementations

func (m *Model) handleActionQuit(_ *keybinds.Context) tea.Cmd {
	m.quitting = true
	return tea.Quit
}

func (m *Model) handleActionCancelOp(ctx *keybinds.Context) tea.Cmd {
	if ctx.OperationRunning {
		m.cancelExecution()
		return nil
	}
	m.quitting = true
	return tea.Quit
}

func (m *Model) handleActionToggleHelp(_ *keybinds.Context) tea.Cmd {
	m.toggleHelpModal()
	return nil
}

func (m *Model) handleActionToggleConfig(_ *keybinds.Context) tea.Cmd {
	m.toggleSettingsModal()
	return nil
}

func (m *Model) handleActionToggleTheme(_ *keybinds.Context) tea.Cmd {
	return m.toggleThemeModal()
}

func (m *Model) handleActionFocusWorkspace(_ *keybinds.Context) tea.Cmd {
	if m.panelManager == nil {
		return nil
	}
	// Reset main area to diff mode when leaving history
	if m.historyFocused && m.mainArea != nil {
		m.mainArea.SetMode(ModeDiff)
	}
	m.historyFocused = false
	m.updateLayout()
	return m.panelManager.SetFocus(PanelWorkspace)
}

func (m *Model) handleActionFocusResources(_ *keybinds.Context) tea.Cmd {
	if m.panelManager == nil {
		return nil
	}
	// Reset main area to diff mode when leaving history detail
	if m.mainArea != nil && m.mainArea.GetMode() == ModeHistoryDetail {
		m.mainArea.SetMode(ModeDiff)
	}
	m.historyFocused = false
	m.updateLayout()
	return m.panelManager.SetFocus(PanelResources)
}

func (m *Model) handleActionFocusHistory(_ *keybinds.Context) tea.Cmd {
	if m.panelManager == nil {
		return nil
	}
	m.updateLayout()
	cmd := m.panelManager.SetFocus(PanelHistory)
	m.historyFocused = true
	historyCmd := m.showSelectedHistoryDetail()
	return tea.Batch(cmd, historyCmd)
}

func (m *Model) handleActionFocusMain(_ *keybinds.Context) tea.Cmd {
	if m.panelManager == nil {
		return nil
	}
	// Keep current mode - user explicitly chose to focus main panel
	m.historyFocused = false
	m.updateLayout()
	return m.panelManager.SetFocus(PanelMain)
}

func (m *Model) handleActionFocusCommandLog(_ *keybinds.Context) tea.Cmd {
	// Reset main area to diff mode when leaving history detail
	if m.mainArea != nil && m.mainArea.GetMode() == ModeHistoryDetail {
		m.mainArea.SetMode(ModeDiff)
	}
	m.historyFocused = false
	return m.focusCommandLog()
}

func (m *Model) handleActionCycleFocus(_ *keybinds.Context) tea.Cmd {
	return m.cycleFocusWithDirection(false)
}

func (m *Model) handleActionCycleFocusBack(_ *keybinds.Context) tea.Cmd {
	return m.cycleFocusWithDirection(true)
}

// cycleFocusWithDirection handles panel focus cycling in either direction.
func (m *Model) cycleFocusWithDirection(reverse bool) tea.Cmd {
	if m.panelManager == nil {
		return nil
	}
	wasHistoryFocused := m.historyFocused
	m.updateLayout()
	cmd := m.panelManager.CycleFocus(reverse)

	focusedPanel := m.panelManager.GetFocusedPanel()
	m.historyFocused = focusedPanel == PanelHistory

	// When history panel gains focus, show the selected history detail
	if m.historyFocused && !wasHistoryFocused {
		historyCmd := m.showSelectedHistoryDetail()
		return tea.Batch(cmd, historyCmd)
	}
	// When leaving history, switch back to diff mode
	if !m.historyFocused && m.mainArea != nil && m.mainArea.GetMode() == ModeHistoryDetail {
		m.mainArea.SetMode(ModeDiff)
	}
	return cmd
}

func (m *Model) handleActionToggleLog(_ *keybinds.Context) tea.Cmd {
	if m.panelManager == nil {
		return nil
	}
	visible := m.panelManager.ToggleCommandLog()
	m.updateLayout()
	if !visible && m.panelManager.GetFocusedPanel() == PanelCommandLog {
		return m.panelManager.SetFocus(PanelResources)
	}
	return nil
}

func (m *Model) handleActionEscapeBack(_ *keybinds.Context) tea.Cmd {
	// Try to exit history detail mode first
	if m.executionMode && m.handleEscKey() {
		return nil
	}
	// Otherwise return to resource list
	if m.panelManager == nil {
		return nil
	}
	if m.panelManager.GetFocusedPanel() != PanelResources {
		m.updateLayout()
		return m.panelManager.SetFocus(PanelResources)
	}
	return nil
}

func (m *Model) handleActionToggleHistory(_ *keybinds.Context) tea.Cmd {
	m.showHistory = !m.showHistory
	if !m.showHistory {
		m.historyFocused = false
	}
	m.updateLayout()
	return nil
}

func (m *Model) handleActionPlan(_ *keybinds.Context) tea.Cmd {
	return requestPlan()
}

func (m *Model) handleActionApply(_ *keybinds.Context) tea.Cmd {
	return requestApply()
}

func (m *Model) handleActionRefresh(_ *keybinds.Context) tea.Cmd {
	return requestRefresh()
}

func (m *Model) handleActionValidate(_ *keybinds.Context) tea.Cmd {
	return requestValidate()
}

func (m *Model) handleActionFormat(_ *keybinds.Context) tea.Cmd {
	return requestFormat()
}

func (m *Model) handleActionToggleCreate(_ *keybinds.Context) tea.Cmd {
	m.handleToggleFilter(terraform.ActionCreate)
	return nil
}

func (m *Model) handleActionToggleUpdate(_ *keybinds.Context) tea.Cmd {
	m.handleToggleFilter(terraform.ActionUpdate)
	return nil
}

func (m *Model) handleActionToggleDelete(_ *keybinds.Context) tea.Cmd {
	m.handleToggleFilter(terraform.ActionDelete)
	return nil
}

func (m *Model) handleActionToggleReplace(_ *keybinds.Context) tea.Cmd {
	m.handleToggleFilter(terraform.ActionReplace)
	return nil
}

func (m *Model) handleActionToggleAllGroups(_ *keybinds.Context) tea.Cmd {
	m.resourceList.ToggleAllGroups()
	return nil
}

func (m *Model) handleActionToggleStatus(_ *keybinds.Context) tea.Cmd {
	m.resourceList.SetShowStatus(!m.resourceList.ShowStatus())
	return nil
}

func (m *Model) handleActionSwitchTabPrev(_ *keybinds.Context) tea.Cmd {
	return switchResourcesTab(-1)
}

func (m *Model) handleActionSwitchTabNext(_ *keybinds.Context) tea.Cmd {
	return switchResourcesTab(1)
}

func (m *Model) handleActionMoveUp(ctx *keybinds.Context) tea.Cmd {
	return m.handleVerticalNavigation(ctx.FocusedPanel, true)
}

func (m *Model) handleActionMoveDown(ctx *keybinds.Context) tea.Cmd {
	return m.handleVerticalNavigation(ctx.FocusedPanel, false)
}

// handleVerticalNavigation handles up/down navigation within panels.
func (m *Model) handleVerticalNavigation(panel keybinds.PanelID, moveUp bool) tea.Cmd {
	switch panel {
	case keybinds.PanelResources:
		if m.resourcesActiveTab == 0 {
			if moveUp {
				m.resourceList.MoveUp()
			} else {
				m.resourceList.MoveDown()
			}
		} else if m.stateListContent != nil {
			if moveUp {
				m.stateListContent.MoveUp()
			} else {
				m.stateListContent.MoveDown()
			}
		}
	case keybinds.PanelHistory:
		if m.historyPanel != nil {
			if moveUp {
				m.historyPanel.MoveUp()
			} else {
				m.historyPanel.MoveDown()
			}
			m.historySelected = m.historyPanel.GetSelectedIndex()
			return m.showSelectedHistoryDetail()
		}
	case keybinds.PanelMain:
		if m.mainArea != nil {
			keyType := tea.KeyDown
			if moveUp {
				keyType = tea.KeyUp
			}
			_, cmd := m.mainArea.HandleKey(tea.KeyMsg{Type: keyType})
			return cmd
		}
	case keybinds.PanelCommandLog:
		if m.commandLogPanel != nil {
			keyType := tea.KeyDown
			if moveUp {
				keyType = tea.KeyUp
			}
			_, cmd := m.commandLogPanel.HandleKey(tea.KeyMsg{Type: keyType})
			return cmd
		}
	case keybinds.PanelNone, keybinds.PanelWorkspace:
		// No vertical navigation for these panels
	}
	return nil
}

func (m *Model) handleActionSelect(ctx *keybinds.Context) tea.Cmd {
	switch ctx.FocusedPanel {
	case keybinds.PanelResources:
		if m.resourcesActiveTab == 0 {
			m.resourceList.ToggleGroup()
		} else if m.stateListContent != nil {
			if m.stateListContent.OnSelect != nil {
				selected := m.stateListContent.GetSelected()
				if selected != nil {
					return m.stateListContent.OnSelect(selected.Address)
				}
			}
		}
	case keybinds.PanelHistory:
		// Focus main panel and show detail
		m.historyFocused = false
		if m.panelManager != nil {
			m.panelManager.SetFocus(PanelMain)
		}
		return m.showSelectedHistoryDetail()
	case keybinds.PanelCommandLog:
		m.execView = viewCommandLog
	case keybinds.PanelNone, keybinds.PanelWorkspace, keybinds.PanelMain:
		// No select action for these panels
	}
	return nil
}

func (m *Model) handleActionScrollUp(ctx *keybinds.Context) tea.Cmd {
	switch ctx.ActiveModal {
	case keybinds.ModalHelp:
		if m.helpModal != nil {
			m.helpModal.ScrollUp()
		}
	case keybinds.ModalSettings:
		if m.settingsModal != nil {
			m.settingsModal.ScrollUp()
		}
	case keybinds.ModalTheme:
		if m.themeModal != nil {
			m.themeModal.ScrollUp()
			m.previewSelectedTheme()
		}
	case keybinds.ModalNone, keybinds.ModalConfirmApply:
		// No scroll for these modals
	}
	return nil
}

func (m *Model) handleActionScrollDown(ctx *keybinds.Context) tea.Cmd {
	switch ctx.ActiveModal {
	case keybinds.ModalHelp:
		if m.helpModal != nil {
			m.helpModal.ScrollDown()
		}
	case keybinds.ModalSettings:
		if m.settingsModal != nil {
			m.settingsModal.ScrollDown()
		}
	case keybinds.ModalTheme:
		if m.themeModal != nil {
			m.themeModal.ScrollDown()
			m.previewSelectedTheme()
		}
	case keybinds.ModalNone, keybinds.ModalConfirmApply:
		// No scroll for these modals
	}
	return nil
}

func (m *Model) handleActionSelectEnv(_ *keybinds.Context) tea.Cmd {
	if m.environmentPanel != nil {
		m.environmentPanel.ActivateSelector()
		m.updateLayout()
	}
	return nil
}

func (m *Model) handleActionConfirmYes(_ *keybinds.Context) tea.Cmd {
	m.modalState = ModalNone
	return m.beginApply()
}

func (m *Model) handleActionConfirmNo(_ *keybinds.Context) tea.Cmd {
	m.modalState = ModalNone
	return nil
}
