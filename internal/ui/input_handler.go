package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

func (m *Model) inputCaptured() bool {
	return false
}

// handleExecutionKey handles keys for non-main execution views.
// Returns (handled, cmd) - only returns handled=true for non-main views.
// Note: For main view (viewMain), keys are delegated to panels via handleKeyMsg.
func (m *Model) handleExecutionKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch m.execView {
	case viewPlanOutput, viewApplyOutput:
		return m.handleLegacyOutputKey(msg)
	case viewCommandLog:
		return m.handleCommandLogKey(msg)
	case viewStateList:
		return m.handleStateListKey(msg)
	case viewStateShow:
		return m.handleStateShowKey(msg)
	default:
		// viewMain - not handled here, delegate to panels
		return false, nil
	}
}

func (m *Model) handleLegacyOutputKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "q":
		if !m.planRunning && !m.applyRunning {
			m.execView = viewMain
			return true, nil
		}
		m.quitting = true
		return true, tea.Quit
	case keybinds.KeyCtrlC:
		m.cancelExecution()
		return true, nil
	case keybinds.KeyEsc:
		if !m.planRunning && !m.applyRunning {
			m.execView = viewMain
			return true, nil
		}
		return true, nil
	default:
		return false, nil
	}
}

func (m *Model) handleCommandLogKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return true, tea.Quit
	case keybinds.KeyEsc:
		m.execView = viewMain
		return true, nil
	default:
		return false, nil
	}
}

func (m *Model) handleStateListKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return true, tea.Quit
	case keybinds.KeyEsc:
		m.execView = viewMain
		return true, nil
	case "up", "k":
		if m.stateListView != nil {
			m.stateListView.MoveUp()
		}
		return true, nil
	case keybinds.KeyDown, "j":
		if m.stateListView != nil {
			m.stateListView.MoveDown()
		}
		return true, nil
	case "enter":
		if m.stateListView != nil {
			if res := m.stateListView.GetSelected(); res != nil {
				return true, m.beginStateShow(res.Address)
			}
		}
		return true, nil
	default:
		return false, nil
	}
}

func (m *Model) handleStateShowKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return true, tea.Quit
	case keybinds.KeyEsc:
		m.execView = viewStateList
		return true, nil
	default:
		if m.stateShowView != nil {
			m.stateShowView, _ = m.stateShowView.Update(msg)
		}
		return true, nil
	}
}

// handleEscKey handles escape key for exiting history detail or state show mode.
func (m *Model) handleEscKey() bool {
	if m.mainArea == nil {
		return false
	}
	mode := m.mainArea.GetMode()
	switch mode {
	case ModeHistoryDetail:
		m.mainArea.ExitHistoryDetail()
		return true
	case ModeStateShow:
		m.mainArea.SetMode(ModeDiff)
		return true
	case ModeAbout:
		m.mainArea.SetMode(ModeDiff)
		return true
	case ModeDiff, ModeLogs:
		// Nothing to exit from these modes
		return false
	}
	return false
}

func (m *Model) focusCommandLog() tea.Cmd {
	if m.panelManager == nil {
		return nil
	}
	if !m.panelManager.IsCommandLogVisible() {
		m.panelManager.SetCommandLogVisible(true)
	}
	cmd := m.panelManager.SetFocus(PanelCommandLog)
	m.updateLayout()
	return cmd
}
