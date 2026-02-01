package ui

import (
	"testing"

	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

func TestConvertPanelID(t *testing.T) {
	tests := []struct {
		name string
		p    PanelID
		want keybinds.PanelID
	}{
		{"workspace", PanelWorkspace, keybinds.PanelWorkspace},
		{"resources", PanelResources, keybinds.PanelResources},
		{"history", PanelHistory, keybinds.PanelHistory},
		{"main", PanelMain, keybinds.PanelMain},
		{"command log", PanelCommandLog, keybinds.PanelCommandLog},
		{"unknown", PanelID(999), keybinds.PanelNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertPanelID(tt.p)
			if got != tt.want {
				t.Errorf("convertPanelID(%v) = %v, want %v", tt.p, got, tt.want)
			}
		})
	}
}

func TestConvertModalState(t *testing.T) {
	tests := []struct {
		name string
		s    ModalState
		want keybinds.ModalID
	}{
		{"help", ModalHelp, keybinds.ModalHelp},
		{"settings", ModalSettings, keybinds.ModalSettings},
		{"confirm apply", ModalConfirmApply, keybinds.ModalConfirmApply},
		{"theme", ModalTheme, keybinds.ModalTheme},
		{"none", ModalNone, keybinds.ModalNone},
		{"unknown", ModalState(999), keybinds.ModalNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertModalState(tt.s)
			if got != tt.want {
				t.Errorf("convertModalState(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestConvertExecView(t *testing.T) {
	tests := []struct {
		name string
		v    executionView
		want keybinds.ViewID
	}{
		{"plan output", viewPlanOutput, keybinds.ViewPlanOutput},
		{"apply output", viewApplyOutput, keybinds.ViewPlanOutput},
		{"command log", viewCommandLog, keybinds.ViewCommandLog},
		{"state list", viewStateList, keybinds.ViewStateList},
		{"state show", viewStateShow, keybinds.ViewStateShow},
		{"main", viewMain, keybinds.ViewMain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertExecView(tt.v)
			if got != tt.want {
				t.Errorf("convertExecView(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestBuildKeybindContext(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := m.buildKeybindContext()
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if !ctx.ExecutionMode {
		t.Error("expected ExecutionMode to be true")
	}
}

func TestBuildKeybindContextNonExecution(t *testing.T) {
	m := NewModel(nil)
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := m.buildKeybindContext()
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if ctx.ExecutionMode {
		t.Error("expected ExecutionMode to be false")
	}
}

func TestRegisterKeybindHandlersNilRegistry(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.keybindRegistry = nil
	// Should not panic
	m.registerKeybindHandlers()
}

func TestHandleActionEscapeBackNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil
	cmd := m.handleActionEscapeBack(nil)
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

func TestHandleActionToggleLogNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil
	cmd := m.handleActionToggleLog(nil)
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

func TestHandleActionToggleHistory(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.showHistory = false
	m.handleActionToggleHistory(nil)
	if !m.showHistory {
		t.Error("expected showHistory to be true")
	}

	m.showHistory = true
	m.historyFocused = true
	m.handleActionToggleHistory(nil)
	if m.showHistory {
		t.Error("expected showHistory to be false")
	}
	if m.historyFocused {
		t.Error("expected historyFocused to be false")
	}
}

func TestSendKeyToPanelNilComponents(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.mainArea = nil
	m.commandLogPanel = nil

	// Should not panic
	cmd := m.sendKeyToPanel(keybinds.PanelMain, 0)
	if cmd != nil {
		t.Error("expected nil cmd when mainArea is nil")
	}

	cmd = m.sendKeyToPanel(keybinds.PanelCommandLog, 0)
	if cmd != nil {
		t.Error("expected nil cmd when commandLogPanel is nil")
	}

	// Other panels should return nil
	cmd = m.sendKeyToPanel(keybinds.PanelWorkspace, 0)
	if cmd != nil {
		t.Error("expected nil cmd for PanelWorkspace")
	}
}

func TestHandleVerticalNavigationNoPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.mainArea = nil
	m.commandLogPanel = nil
	m.historyPanel = nil

	// PanelNone and PanelWorkspace should return nil
	cmd := m.handleVerticalNavigation(keybinds.PanelNone, true)
	if cmd != nil {
		t.Error("expected nil cmd for PanelNone")
	}

	cmd = m.handleVerticalNavigation(keybinds.PanelWorkspace, true)
	if cmd != nil {
		t.Error("expected nil cmd for PanelWorkspace")
	}
}

func TestHandleActionSelectPanelNone(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelNone}
	cmd := m.handleActionSelect(ctx)
	if cmd != nil {
		t.Error("expected nil cmd for PanelNone")
	}
}

func TestHandleActionScrollUpNilModals(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.helpModal = nil
	m.settingsModal = nil
	m.themeModal = nil

	tests := []keybinds.ModalID{
		keybinds.ModalHelp,
		keybinds.ModalSettings,
		keybinds.ModalTheme,
		keybinds.ModalNone,
	}

	for _, modal := range tests {
		ctx := &keybinds.Context{ActiveModal: modal}
		// Should not panic
		cmd := m.handleActionScrollUp(ctx)
		if cmd != nil {
			t.Errorf("expected nil cmd for modal %v", modal)
		}
	}
}

func TestHandleActionScrollDownNilModals(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.helpModal = nil
	m.settingsModal = nil
	m.themeModal = nil

	tests := []keybinds.ModalID{
		keybinds.ModalHelp,
		keybinds.ModalSettings,
		keybinds.ModalTheme,
		keybinds.ModalNone,
	}

	for _, modal := range tests {
		ctx := &keybinds.Context{ActiveModal: modal}
		// Should not panic
		cmd := m.handleActionScrollDown(ctx)
		if cmd != nil {
			t.Errorf("expected nil cmd for modal %v", modal)
		}
	}
}

func TestHandleVerticalNavigationResourcesPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 0

	// Should not panic and should navigate resource list
	cmd := m.handleVerticalNavigation(keybinds.PanelResources, true)
	if cmd != nil {
		t.Error("expected nil cmd for resource list navigation")
	}

	cmd = m.handleVerticalNavigation(keybinds.PanelResources, false)
	if cmd != nil {
		t.Error("expected nil cmd for resource list navigation")
	}
}

func TestHandleVerticalNavigationHistoryPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// historyPanel should be nil in basic setup, test should not panic
	m.historyPanel = nil
	cmd := m.handleVerticalNavigation(keybinds.PanelHistory, true)
	if cmd != nil {
		t.Error("expected nil cmd when historyPanel is nil")
	}
}

func TestHandleVerticalNavigationMainPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should handle main panel navigation
	cmd := m.handleVerticalNavigation(keybinds.PanelMain, true)
	_ = cmd // May or may not be nil depending on setup
}

func TestHandleVerticalNavigationCommandLogPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should handle command log panel navigation
	cmd := m.handleVerticalNavigation(keybinds.PanelCommandLog, false)
	_ = cmd // May or may not be nil depending on setup
}

func TestHandleActionSelectResourcesPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 0

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelResources}
	// Should toggle group in resource list
	cmd := m.handleActionSelect(ctx)
	if cmd != nil {
		t.Error("expected nil cmd for resource list select")
	}
}

func TestHandleActionSelectHistoryPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelHistory}
	// Should focus main panel and show detail
	m.historyFocused = true
	cmd := m.handleActionSelect(ctx)
	if m.historyFocused {
		t.Error("expected historyFocused to be false after select")
	}
	_ = cmd // May or may not be nil
}
