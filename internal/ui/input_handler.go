package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/consts"
)

// KeyMap defines the key bindings.
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Expand    key.Binding
	ToggleAll key.Binding
	Filter    key.Binding
	Quit      key.Binding
	Keybinds  key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Expand: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "toggle group"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle all groups"),
		),
		Filter: key.NewBinding(
			key.WithKeys("c", "u", "d", "r"),
			key.WithHelp("c/u/d/r", "toggle filters"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", consts.KeyCtrlC),
			key.WithHelp("q", "quit"),
		),
		Keybinds: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle keybinds"),
		),
	}
}

func (m *Model) inputCaptured() bool {
	return false
}

func (m *Model) handleExecutionKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch m.execView {
	case viewPlanConfirm:
		return m.handlePlanConfirmKey(msg)
	case viewPlanOutput, viewApplyOutput:
		return m.handleLegacyOutputKey(msg)
	case viewCommandLog:
		return m.handleCommandLogKey(msg)
	case viewStateList:
		return m.handleStateListKey(msg)
	case viewStateShow:
		return m.handleStateShowKey(msg)
	default:
		return m.handleMainExecutionKey(msg)
	}
}

func (m *Model) handlePlanConfirmKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quitting = true
		return true, tea.Quit
	case "y", "Y":
		return true, m.beginApply()
	case "n", "N", consts.KeyEsc:
		m.execView = viewMain
		return true, nil
	case consts.KeyCtrlC:
		m.cancelExecution()
		return true, nil
	default:
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
	case consts.KeyCtrlC:
		m.cancelExecution()
		return true, nil
	case consts.KeyEsc:
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
	case consts.KeyEsc:
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
	case consts.KeyEsc:
		m.execView = viewMain
		return true, nil
	case "up", "k":
		if m.stateListView != nil {
			m.stateListView.MoveUp()
		}
		return true, nil
	case consts.KeyDown, "j":
		if m.stateListView != nil {
			m.stateListView.MoveDown()
		}
		return true, nil
	case consts.KeyEnter:
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
	case consts.KeyEsc:
		m.execView = viewStateList
		return true, nil
	default:
		if m.stateShowView != nil {
			m.stateShowView, _ = m.stateShowView.Update(msg)
		}
		return true, nil
	}
}

func (m *Model) handleMainExecutionKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	key := msg.String()
	if handled, cmd := m.handleExecutionActionKey(key); handled {
		return true, cmd
	}
	if handled, cmd := m.handleExecutionToggleKey(key); handled {
		return true, cmd
	}
	if handled := m.handleExecutionNavKey(key); handled {
		return true, nil
	}
	if m.historyFocused {
		if handled, cmd := m.handleHistoryKeys(key); handled {
			return true, cmd
		}
	}
	return false, nil
}

func (m *Model) handleExecutionActionKey(key string) (bool, tea.Cmd) {
	switch key {
	case "p":
		return true, m.beginPlan()
	case "f":
		return true, m.beginRefresh()
	case "v":
		return true, m.beginValidate()
	case "F":
		return true, m.beginFormat()
	case "a":
		return m.handleApplyKey()
	default:
		return false, nil
	}
}

func (m *Model) handleExecutionToggleKey(key string) (bool, tea.Cmd) {
	switch key {
	case "h":
		return m.handleHistoryToggle()
	case "s":
		m.resourceList.SetShowStatus(!m.resourceList.ShowStatus())
		return true, nil
	case "D":
		return m.focusCommandLog()
	default:
		return false, nil
	}
}

func (m *Model) handleExecutionNavKey(key string) bool {
	switch key {
	case "tab":
		if m.showHistory && len(m.historyEntries) > 0 {
			m.historyFocused = !m.historyFocused
			m.syncHistorySelection()
			return true
		}
	case consts.KeyCtrlC:
		if m.planRunning || m.applyRunning || m.refreshRunning {
			m.cancelExecution()
			return true
		}
	case consts.KeyEsc:
		if m.mainArea != nil && m.mainArea.GetMode() == ModeHistoryDetail {
			m.mainArea.ExitHistoryDetail()
			return true
		}
	}
	return false
}

func (m *Model) handleApplyKey() (bool, tea.Cmd) {
	if m.plan == nil {
		if m.toast != nil {
			return true, m.toast.ShowError("No plan loaded; run terraform plan first")
		}
		return true, nil
	}
	if m.planView != nil {
		m.planView.SetSummary(m.planSummary())
	}
	m.execView = viewPlanConfirm
	return true, nil
}

func (m *Model) handleHistoryToggle() (bool, tea.Cmd) {
	m.showHistory = !m.showHistory
	if !m.showHistory {
		m.historyFocused = false
	}
	m.updateLayout()
	return true, nil
}

func (m *Model) focusCommandLog() (bool, tea.Cmd) {
	if m.panelManager == nil {
		return true, nil
	}
	if m.panelManager.IsCommandLogVisible() {
		return true, m.panelManager.SetFocus(PanelCommandLog)
	}
	m.panelManager.SetCommandLogVisible(true)
	m.updateLayout()
	return true, m.panelManager.SetFocus(PanelCommandLog)
}
