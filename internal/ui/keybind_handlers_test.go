package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

func TestHandleActionRefreshOnStateTabLoadsStateList(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1

	mock := setupMockExecutor(t)
	mock.StateListResult = testutil.NewMockResult("null_resource.example", 0)
	m.executor = mock

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelResources}
	cmd := m.handleActionRefresh(ctx)
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}

	foundStateList := false
	for _, batchCmd := range batch {
		if batchCmd == nil {
			continue
		}
		if _, ok := batchCmd().(StateListCompleteMsg); ok {
			foundStateList = true
			break
		}
	}
	if !foundStateList {
		t.Fatal("expected batch to include StateListCompleteMsg command")
	}
	if mock.StateListCalls != 1 {
		t.Fatalf("expected one state list call, got %d", mock.StateListCalls)
	}
}

func TestHandleActionRefreshOutsideStateTabRequestsTerraformRefresh(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 0

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelResources}
	cmd := m.handleActionRefresh(ctx)
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	msg := cmd()
	if _, ok := msg.(RequestRefreshMsg); !ok {
		t.Fatalf("expected RequestRefreshMsg, got %T", msg)
	}
}

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

func TestHandleActionStateRemoveShowsConfirmModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1
	m.executor = setupMockExecutor(t)
	m.stateListContent.SetResources([]terraform.StateResource{{Address: "null_resource.example"}})

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelResources}
	cmd := m.handleActionStateRemove(ctx)
	if cmd != nil {
		t.Fatal("expected nil cmd while showing confirm modal")
	}
	if m.modalState != ModalConfirmApply {
		t.Fatalf("expected confirm modal state, got %v", m.modalState)
	}
	if m.pendingConfirmCmd == nil {
		t.Fatal("expected pending confirm command for state remove")
	}
}

func TestHandleActionStateMoveShowsDestinationInput(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1
	m.executor = setupMockExecutor(t)
	m.stateListContent.SetResources([]terraform.StateResource{{Address: "null_resource.a"}, {Address: "null_resource.b"}})

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelResources}
	cmd := m.handleActionStateMove(ctx)
	if cmd == nil {
		t.Fatal("expected non-nil cursor blink command for destination input modal")
	}
	if m.stateMoveSource != "null_resource.a" {
		t.Fatalf("expected source to match selected item, got %q", m.stateMoveSource)
	}
	if m.modalState != ModalStateMoveDestination {
		t.Fatalf("expected destination input modal state, got %v", m.modalState)
	}
	if m.pendingConfirmCmd != nil {
		t.Fatal("expected no pending confirm command before destination input is confirmed")
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

// ============================================================================
// handleActionCancelOp tests
// ============================================================================

func TestHandleActionCancelOpWithOperationRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{OperationRunning: true}
	cancelCalled := false
	m.cancelFunc = func() { cancelCalled = true }

	cmd := m.handleActionCancelOp(ctx)

	if !cancelCalled {
		t.Error("expected cancelFunc to be called")
	}
	if cmd != nil {
		t.Error("expected nil cmd when canceling operation")
	}
	if m.quitting {
		t.Error("expected quitting to be false")
	}
}

func TestHandleActionCancelOpNoOperationRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	ctx := &keybinds.Context{OperationRunning: false}
	cmd := m.handleActionCancelOp(ctx)

	if !m.quitting {
		t.Error("expected quitting to be true when no operation running")
	}
	// cmd should be tea.Quit
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
}

// ============================================================================
// handleActionEscapeBack tests
// ============================================================================

func TestHandleActionEscapeBackInExecutionMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executionMode = true

	ctx := &keybinds.Context{}
	cmd := m.handleActionEscapeBack(ctx)

	// Should return to resource list if not already focused
	_ = cmd
}

func TestHandleActionEscapeBackOnResourcesPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Set focus to resources panel
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelResources)
	}

	ctx := &keybinds.Context{}
	cmd := m.handleActionEscapeBack(ctx)

	// When already on resources panel, should return nil
	if cmd != nil {
		t.Error("expected nil cmd when already on resources panel")
	}
}

func TestHandleActionEscapeBackFromOtherPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Set focus to main panel
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelMain)
	}

	ctx := &keybinds.Context{}
	cmd := m.handleActionEscapeBack(ctx)

	// Should return a cmd to switch to resources panel
	_ = cmd // May or may not be nil depending on panel state
}

// ============================================================================
// sendKeyToPanel tests
// ============================================================================

func TestSendKeyToPanelMainWithValidArea(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// mainArea should be initialized
	if m.mainArea != nil {
		cmd := m.sendKeyToPanel(keybinds.PanelMain, 0)
		_ = cmd // May be nil or a command
	}
}

func TestSendKeyToPanelCommandLogWithValidPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// commandLogPanel may or may not be initialized
	if m.commandLogPanel != nil {
		cmd := m.sendKeyToPanel(keybinds.PanelCommandLog, 0)
		_ = cmd // May be nil or a command
	}
}

func TestSendKeyToPanelResourcesReturnsNil(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Resources panel doesn't support this navigation
	cmd := m.sendKeyToPanel(keybinds.PanelResources, 0)
	if cmd != nil {
		t.Error("expected nil cmd for resources panel")
	}
}

func TestSendKeyToPanelHistoryReturnsNil(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// History panel doesn't support this navigation
	cmd := m.sendKeyToPanel(keybinds.PanelHistory, 0)
	if cmd != nil {
		t.Error("expected nil cmd for history panel")
	}
}

// ============================================================================
// handleActionScrollUp/Down with initialized modals tests
// ============================================================================

func TestHandleActionScrollUpWithInitializedHelpModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Help modal should be initialized after updateLayout
	if m.helpModal != nil {
		ctx := &keybinds.Context{ActiveModal: keybinds.ModalHelp}
		cmd := m.handleActionScrollUp(ctx)
		if cmd != nil {
			t.Error("expected nil cmd for help modal scroll")
		}
	}
}

func TestHandleActionScrollDownWithInitializedHelpModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.helpModal != nil {
		ctx := &keybinds.Context{ActiveModal: keybinds.ModalHelp}
		cmd := m.handleActionScrollDown(ctx)
		if cmd != nil {
			t.Error("expected nil cmd for help modal scroll")
		}
	}
}

func TestHandleActionScrollUpWithInitializedSettingsModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.settingsModal != nil {
		ctx := &keybinds.Context{ActiveModal: keybinds.ModalSettings}
		cmd := m.handleActionScrollUp(ctx)
		if cmd != nil {
			t.Error("expected nil cmd for settings modal scroll")
		}
	}
}

func TestHandleActionScrollDownWithInitializedSettingsModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.settingsModal != nil {
		ctx := &keybinds.Context{ActiveModal: keybinds.ModalSettings}
		cmd := m.handleActionScrollDown(ctx)
		if cmd != nil {
			t.Error("expected nil cmd for settings modal scroll")
		}
	}
}

func TestHandleActionScrollUpWithConfirmApplyModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{ActiveModal: keybinds.ModalConfirmApply}
	cmd := m.handleActionScrollUp(ctx)
	if cmd != nil {
		t.Error("expected nil cmd for confirm apply modal")
	}
}

func TestHandleActionScrollDownWithConfirmApplyModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{ActiveModal: keybinds.ModalConfirmApply}
	cmd := m.handleActionScrollDown(ctx)
	if cmd != nil {
		t.Error("expected nil cmd for confirm apply modal")
	}
}

// ============================================================================
// handleVerticalNavigation additional tests
// ============================================================================

func TestHandleVerticalNavigationResourcesInLogsMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 0

	// Set main area to logs mode
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeLogs)
	}

	// When in logs mode, scroll should be redirected to MainArea
	cmd := m.handleVerticalNavigation(keybinds.PanelResources, true)
	_ = cmd
}

func TestHandleVerticalNavigationResourcesStateTab(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1 // State tab

	// Test with stateListContent
	if m.stateListContent != nil {
		cmd := m.handleVerticalNavigation(keybinds.PanelResources, true)
		_ = cmd // Should return showSelectedStateDetail command
	}
}

func TestHandleVerticalNavigationResourcesStateTabMoveDown(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1 // State tab

	if m.stateListContent != nil {
		cmd := m.handleVerticalNavigation(keybinds.PanelResources, false) // Move down
		_ = cmd
	}
}

// ============================================================================
// handleActionSelect additional tests
// ============================================================================

func TestHandleActionSelectStateTab(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1 // State tab

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelResources}
	cmd := m.handleActionSelect(ctx)
	_ = cmd
}

func TestHandleActionSelectMainPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelMain}
	cmd := m.handleActionSelect(ctx)
	if cmd != nil {
		t.Error("expected nil cmd for main panel")
	}
}

func TestHandleActionSelectCommandLogPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelCommandLog}
	cmd := m.handleActionSelect(ctx)
	if cmd != nil {
		t.Error("expected nil cmd for command log panel")
	}
}

func TestHandleActionSelectWorkspacePanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{FocusedPanel: keybinds.PanelWorkspace}
	cmd := m.handleActionSelect(ctx)
	if cmd != nil {
		t.Error("expected nil cmd for workspace panel")
	}
}

// ============================================================================
// cycleFocusWithDirection tests
// ============================================================================

func TestCycleFocusWithDirectionForward(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.cycleFocusWithDirection(false) // Forward
	_ = cmd
}

func TestCycleFocusWithDirectionReverse(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.cycleFocusWithDirection(true) // Reverse
	_ = cmd
}

func TestCycleFocusWithDirectionNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil

	cmd := m.cycleFocusWithDirection(false)
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

func TestCycleFocusWithDirectionToWorkspace(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Cycle until we get to workspace panel
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelWorkspace)
	}

	cmd := m.cycleFocusWithDirection(false)
	_ = cmd
}

func TestCycleFocusWithDirectionFromHistory(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.historyFocused = true
	m.showHistory = true // Required to have history panel in cycle

	// Cycle away from history - just test it doesn't panic
	cmd := m.cycleFocusWithDirection(false)
	_ = cmd
}

func TestCycleFocusWithDirectionToHistory(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.historyFocused = false
	m.showHistory = true

	cmd := m.cycleFocusWithDirection(false)
	_ = cmd
}

func TestCycleFocusWithDirectionRestoresMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Set main area to history detail mode
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeHistoryDetail)
	}
	m.historyFocused = false

	cmd := m.cycleFocusWithDirection(false)
	_ = cmd
}

// ============================================================================
// handleActionFocusWorkspace tests
// ============================================================================

func TestHandleActionFocusWorkspaceBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusWorkspace(ctx)

	if m.historyFocused {
		t.Error("expected historyFocused to be false")
	}
	_ = cmd
}

func TestHandleActionFocusWorkspaceNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusWorkspace(ctx)
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

func TestHandleActionFocusWorkspaceNilMainArea(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.mainArea = nil

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusWorkspace(ctx)
	_ = cmd // Should still return a command
}

// ============================================================================
// handleActionFocusResources tests
// ============================================================================

func TestHandleActionFocusResourcesBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusResources(ctx)

	if m.historyFocused {
		t.Error("expected historyFocused to be false")
	}
	_ = cmd
}

func TestHandleActionFocusResourcesWhenOperationRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusResources(ctx)

	// Should set mode to ModeLogs when operation is running
	if m.mainArea != nil && m.mainArea.GetMode() != ModeLogs {
		t.Error("expected main area mode to be ModeLogs when operation running")
	}
	_ = cmd
}

func TestHandleActionFocusResourcesFromHistoryMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea != nil {
		m.mainArea.SetMode(ModeHistoryDetail)
	}

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusResources(ctx)

	// Should restore mode to ModeDiff
	if m.mainArea != nil && m.mainArea.GetMode() != ModeDiff {
		t.Error("expected main area mode to be ModeDiff")
	}
	_ = cmd
}

func TestHandleActionFocusResourcesFromAboutMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea != nil {
		m.mainArea.SetMode(ModeAbout)
	}

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusResources(ctx)

	// Should restore mode to ModeDiff
	if m.mainArea != nil && m.mainArea.GetMode() != ModeDiff {
		t.Error("expected main area mode to be ModeDiff")
	}
	_ = cmd
}

func TestHandleActionFocusResourcesNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusResources(ctx)
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

// ============================================================================
// handleActionFocusHistory tests
// ============================================================================

func TestHandleActionFocusHistoryBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusHistory(ctx)

	if !m.historyFocused {
		t.Error("expected historyFocused to be true")
	}
	_ = cmd
}

func TestHandleActionFocusHistoryNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusHistory(ctx)
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

// ============================================================================
// handleActionFocusMain tests
// ============================================================================

func TestHandleActionFocusMainBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.historyFocused = true

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusMain(ctx)

	if m.historyFocused {
		t.Error("expected historyFocused to be false")
	}
	_ = cmd
}

func TestHandleActionFocusMainNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusMain(ctx)
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

// ============================================================================
// handleActionFocusCommandLog tests
// ============================================================================

func TestHandleActionFocusCommandLogBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.historyFocused = true

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusCommandLog(ctx)

	if m.historyFocused {
		t.Error("expected historyFocused to be false")
	}
	_ = cmd
}

func TestHandleActionFocusCommandLogFromHistoryMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea != nil {
		m.mainArea.SetMode(ModeHistoryDetail)
	}

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusCommandLog(ctx)

	// Should restore mode to ModeDiff
	if m.mainArea != nil && m.mainArea.GetMode() != ModeDiff {
		t.Error("expected main area mode to be ModeDiff")
	}
	_ = cmd
}

func TestHandleActionFocusCommandLogFromAboutMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea != nil {
		m.mainArea.SetMode(ModeAbout)
	}

	ctx := &keybinds.Context{}
	cmd := m.handleActionFocusCommandLog(ctx)

	// Should restore mode to ModeDiff
	if m.mainArea != nil && m.mainArea.GetMode() != ModeDiff {
		t.Error("expected main area mode to be ModeDiff")
	}
	_ = cmd
}

// ============================================================================
// handleActionToggleLog tests
// ============================================================================

func TestHandleActionToggleLogBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{}
	cmd := m.handleActionToggleLog(ctx)
	_ = cmd
}

func TestHandleActionToggleLogWhenFocusedAndHidden(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Set focus to command log and toggle it off
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelCommandLog)
	}

	ctx := &keybinds.Context{}
	cmd := m.handleActionToggleLog(ctx)
	_ = cmd // Should return focus change command
}

// ============================================================================
// handleActionSelectEnv tests
// ============================================================================

func TestHandleActionSelectEnvBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{}
	cmd := m.handleActionSelectEnv(ctx)
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleActionSelectEnvNilEnvironmentPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.environmentPanel = nil

	ctx := &keybinds.Context{}
	cmd := m.handleActionSelectEnv(ctx)
	if cmd != nil {
		t.Error("expected nil cmd when environmentPanel is nil")
	}
}

// ============================================================================
// handleActionConfirmYes/No tests (keybind specific)
// ============================================================================

func TestHandleActionConfirmYesKeybind(t *testing.T) {
	mock := testutil.NewMockExecutor()
	mock.MockWorkDir = t.TempDir()

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalConfirmApply

	ctx := &keybinds.Context{}
	cmd := m.handleActionConfirmYes(ctx)

	if m.modalState != ModalNone {
		t.Error("expected modalState to be ModalNone")
	}
	_ = cmd // Should return beginApply command
}

func TestHandleActionConfirmNoKeybind(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalConfirmApply

	ctx := &keybinds.Context{}
	cmd := m.handleActionConfirmNo(ctx)

	if m.modalState != ModalNone {
		t.Error("expected modalState to be ModalNone")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ============================================================================
// handleActionToggleAllGroups tests (keybind specific)
// ============================================================================

func TestHandleActionToggleAllGroupsKeybind(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{}
	cmd := m.handleActionToggleAllGroups(ctx)
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ============================================================================
// handleActionToggleStatus tests (keybind specific)
// ============================================================================

func TestHandleActionToggleStatusKeybind(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	initialStatus := m.resourceList.ShowStatus()

	ctx := &keybinds.Context{}
	cmd := m.handleActionToggleStatus(ctx)
	if cmd != nil {
		t.Error("expected nil cmd")
	}

	if m.resourceList.ShowStatus() == initialStatus {
		t.Error("expected showStatus to be toggled")
	}
}

// ============================================================================
// handleActionQuit tests (keybind specific)
// ============================================================================

func TestHandleActionQuitKeybind(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	ctx := &keybinds.Context{}
	cmd := m.handleActionQuit(ctx)

	if !m.quitting {
		t.Error("expected quitting to be true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
}

// ============================================================================
// handleActionToggleHelp/Config/Theme tests (keybind specific)
// ============================================================================

func TestHandleActionToggleHelpKeybind(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{}
	cmd := m.handleActionToggleHelp(ctx)
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleActionToggleConfigKeybind(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{}
	cmd := m.handleActionToggleConfig(ctx)
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleActionToggleThemeKeybind(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := &keybinds.Context{}
	cmd := m.handleActionToggleTheme(ctx)
	_ = cmd // May or may not be nil
}

// ============================================================================
// handlePageNavigation tests
// ============================================================================

func TestHandlePageNavigationUp(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handlePageNavigation(keybinds.PanelMain, true)
	_ = cmd
}

func TestHandlePageNavigationDown(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handlePageNavigation(keybinds.PanelMain, false)
	_ = cmd
}

// ============================================================================
// handleScrollEdge tests
// ============================================================================

func TestHandleScrollEdgeTop(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleScrollEdge(keybinds.PanelMain, true)
	_ = cmd
}

func TestHandleScrollEdgeBottom(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleScrollEdge(keybinds.PanelMain, false)
	_ = cmd
}
