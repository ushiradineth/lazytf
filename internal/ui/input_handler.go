package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// KeyMap defines the key bindings
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Expand    key.Binding
	ToggleAll key.Binding
	Filter    key.Binding
	Quit      key.Binding
	Keybinds  key.Binding
}

// DefaultKeyMap returns the default key bindings
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
			key.WithKeys("q", "ctrl+c"),
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
	key := msg.String()
	switch m.execView {
	case viewPlanConfirm:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "y", "Y":
			return true, m.beginApply()
		case "n", "N", "esc":
			m.execView = viewMain
			return true, nil
		case "ctrl+c":
			m.cancelExecution()
			return true, nil
		}
	case viewPlanOutput, viewApplyOutput:
		// Note: These views are deprecated, staying in viewMain now
		switch key {
		case "q":
			if !m.planRunning && !m.applyRunning {
				m.execView = viewMain
				return true, nil
			}
			m.quitting = true
			return true, tea.Quit
		case "ctrl+c":
			m.cancelExecution()
			return true, nil
		case "esc":
			if !m.planRunning && !m.applyRunning {
				m.execView = viewMain
				return true, nil
			}
		}
	case viewCommandLog:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "esc":
			m.execView = viewMain
			return true, nil
		}
	case viewStateList:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "esc":
			m.execView = viewMain
			return true, nil
		case "up", "k":
			if m.stateListView != nil {
				m.stateListView.MoveUp()
			}
			return true, nil
		case "down", "j":
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
		}
	case viewStateShow:
		switch key {
		case "q":
			m.quitting = true
			return true, tea.Quit
		case "esc":
			m.execView = viewStateList
			return true, nil
		}
		// Forward scroll keys to the view
		if m.stateShowView != nil {
			m.stateShowView, _ = m.stateShowView.Update(msg)
		}
		return true, nil
	default:
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
			if m.plan == nil {
				var cmd tea.Cmd
				if m.toast != nil {
					cmd = m.toast.ShowError("No plan loaded; run terraform plan first")
				}
				return true, cmd
			}
			if m.planView != nil {
				m.planView.SetSummary(m.planSummary())
			}
			m.execView = viewPlanConfirm
			return true, nil
		case "h":
			m.showHistory = !m.showHistory
			if !m.showHistory {
				m.historyFocused = false
			}
			m.updateLayout()
			return true, nil
		case "s":
			m.resourceList.SetShowStatus(!m.resourceList.ShowStatus())
			return true, nil
		case "D":
			// Focus the command log panel where diagnostics are shown
			if m.panelManager != nil && m.panelManager.IsCommandLogVisible() {
				return true, m.panelManager.SetFocus(PanelCommandLog)
			}
			// If command log not visible, show it and focus it
			if m.panelManager != nil {
				m.panelManager.SetCommandLogVisible(true)
				m.updateLayout()
				return true, m.panelManager.SetFocus(PanelCommandLog)
			}
			return true, nil
		case "tab":
			if m.showHistory && len(m.historyEntries) > 0 {
				m.historyFocused = !m.historyFocused
				m.syncHistorySelection()
				return true, nil
			}
		case "ctrl+c":
			if m.planRunning || m.applyRunning || m.refreshRunning {
				m.cancelExecution()
				return true, nil
			}
		case "esc":
			// Exit history detail mode when pressing esc
			if m.mainArea != nil && m.mainArea.GetMode() == ModeHistoryDetail {
				m.mainArea.ExitHistoryDetail()
				return true, nil
			}
		}
		// Handle history keys when history panel is focused (history is always visible in execution mode)
		if m.historyFocused {
			handled, cmd := m.handleHistoryKeys(key)
			if handled {
				return true, cmd
			}
		}
	}
	return false, nil
}
