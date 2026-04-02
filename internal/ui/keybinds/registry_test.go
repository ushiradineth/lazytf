package keybinds

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRegistry_Resolve_GlobalBinding(t *testing.T) {
	r := NewRegistry()
	r.Register(Binding{
		Keys:   []string{"q"},
		Action: ActionQuit,
		Scope:  ScopeGlobal,
	})

	ctx := NewContext()
	binding := r.Resolve("q", ctx)

	if binding == nil {
		t.Fatal("expected binding to be found")
	}
	if binding.Action != ActionQuit {
		t.Errorf("expected action %v, got %v", ActionQuit, binding.Action)
	}
}

func TestRegistry_Resolve_NoMatch(t *testing.T) {
	r := NewRegistry()
	r.Register(Binding{
		Keys:   []string{"q"},
		Action: ActionQuit,
		Scope:  ScopeGlobal,
	})

	ctx := NewContext()
	binding := r.Resolve("x", ctx)

	if binding != nil {
		t.Error("expected no binding to be found")
	}
}

func TestRegistry_Resolve_PriorityOrder(t *testing.T) {
	r := NewRegistry()

	// Register global binding (lower priority)
	r.Register(Binding{
		Keys:   []string{"p"},
		Action: ActionMoveUp,
		Scope:  ScopeGlobal,
	})

	// Register panel binding (higher priority)
	r.Register(Binding{
		Keys:   []string{"p"},
		Action: ActionPlan,
		Scope:  ScopePanel,
		Panel:  PanelResources,
	})

	// When resources panel is focused, panel binding should win
	ctx := &Context{FocusedPanel: PanelResources}
	binding := r.Resolve("p", ctx)

	if binding == nil {
		t.Fatal("expected binding to be found")
	}
	if binding.Action != ActionPlan {
		t.Errorf("expected panel binding to win, got %v", binding.Action)
	}

	// When different panel is focused, only global matches
	ctx = &Context{FocusedPanel: PanelHistory}
	binding = r.Resolve("p", ctx)

	if binding == nil {
		t.Fatal("expected binding to be found")
	}
	if binding.Action != ActionMoveUp {
		t.Errorf("expected global binding, got %v", binding.Action)
	}
}

func TestRegistry_Resolve_ModalWins(t *testing.T) {
	r := NewRegistry()

	// Register global binding
	r.Register(Binding{
		Keys:   []string{"q"},
		Action: ActionQuit,
		Scope:  ScopeGlobal,
	})

	// Register modal binding
	r.Register(Binding{
		Keys:   []string{"q"},
		Action: ActionConfirmNo,
		Scope:  ScopeModal,
		Modal:  ModalConfirmApply,
	})

	// When modal is active, modal binding should win
	ctx := &Context{ActiveModal: ModalConfirmApply}
	binding := r.Resolve("q", ctx)

	if binding == nil {
		t.Fatal("expected binding to be found")
	}
	if binding.Action != ActionConfirmNo {
		t.Errorf("expected modal binding to win, got %v", binding.Action)
	}

	// When no modal is active, global wins
	ctx = &Context{ActiveModal: ModalNone}
	binding = r.Resolve("q", ctx)

	if binding == nil {
		t.Fatal("expected binding to be found")
	}
	if binding.Action != ActionQuit {
		t.Errorf("expected global binding, got %v", binding.Action)
	}
}

func TestRegistry_Resolve_ConditionCheck(t *testing.T) {
	r := NewRegistry()

	// Register binding with execution mode condition
	r.Register(Binding{
		Keys:      []string{"p"},
		Action:    ActionPlan,
		Scope:     ScopeGlobal,
		Condition: ConditionExecutionMode,
	})

	// When not in execution mode, binding should not match
	ctx := &Context{ExecutionMode: false}
	binding := r.Resolve("p", ctx)

	if binding != nil {
		t.Error("expected binding to not match when condition fails")
	}

	// When in execution mode, binding should match
	ctx = &Context{ExecutionMode: true}
	binding = r.Resolve("p", ctx)

	if binding == nil {
		t.Fatal("expected binding to match when condition passes")
	}
}

func TestRegistry_Resolve_PanelTabScope(t *testing.T) {
	r := NewRegistry()

	// Register panel+tab binding
	r.Register(Binding{
		Keys:   []string{"p"},
		Action: ActionPlan,
		Scope:  ScopePanelTab,
		Panel:  PanelResources,
		Tab:    0, // Resources tab
	})

	// When on Resources tab, should match
	ctx := &Context{
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 0,
	}
	binding := r.Resolve("p", ctx)

	if binding == nil {
		t.Fatal("expected binding to match on Resources tab")
	}

	// When on State tab, should not match
	ctx = &Context{
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 1,
	}
	binding = r.Resolve("p", ctx)

	if binding != nil {
		t.Error("expected binding to not match on State tab")
	}
}

func TestBinding_EffectivePriority(t *testing.T) {
	tests := []struct {
		name     string
		binding  Binding
		expected Priority
	}{
		{
			name:     "global scope",
			binding:  Binding{Scope: ScopeGlobal},
			expected: PriorityGlobal,
		},
		{
			name:     "panel scope",
			binding:  Binding{Scope: ScopePanel},
			expected: PriorityPanel,
		},
		{
			name:     "panel tab scope",
			binding:  Binding{Scope: ScopePanelTab},
			expected: PriorityPanelTab,
		},
		{
			name:     "view scope",
			binding:  Binding{Scope: ScopeView},
			expected: PriorityView,
		},
		{
			name:     "modal scope",
			binding:  Binding{Scope: ScopeModal},
			expected: PriorityModal,
		},
		{
			name:     "explicit priority overrides",
			binding:  Binding{Scope: ScopeGlobal, Priority: PriorityModal},
			expected: PriorityModal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.binding.EffectivePriority()
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestBinding_AllKeysString(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		expected string
	}{
		{
			name:     "empty",
			keys:     nil,
			expected: "",
		},
		{
			name:     "single key",
			keys:     []string{"q"},
			expected: "q",
		},
		{
			name:     "multiple keys",
			keys:     []string{"up", "k"},
			expected: "up/k",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Binding{Keys: tt.keys}
			got := b.AllKeysString()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRegistry_GetBindingsForContext(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:   []string{"q"},
		Action: ActionQuit,
		Scope:  ScopeGlobal,
	})
	r.Register(Binding{
		Keys:   []string{"p"},
		Action: ActionPlan,
		Scope:  ScopePanel,
		Panel:  PanelResources,
	})
	r.Register(Binding{
		Keys:   []string{"a"},
		Action: ActionApply,
		Scope:  ScopePanel,
		Panel:  PanelHistory,
	})

	ctx := &Context{FocusedPanel: PanelResources}
	bindings := r.GetBindingsForContext(ctx)

	// Should get global + resources panel bindings
	if len(bindings) != 2 {
		t.Errorf("expected 2 bindings, got %d", len(bindings))
	}

	// Verify the bindings
	actions := make(map[Action]bool)
	for _, b := range bindings {
		actions[b.Action] = true
	}
	if !actions[ActionQuit] {
		t.Error("expected ActionQuit in bindings")
	}
	if !actions[ActionPlan] {
		t.Error("expected ActionPlan in bindings")
	}
	if actions[ActionApply] {
		t.Error("did not expect ActionApply in bindings")
	}
}

func TestRegistry_Resolve_ApplyInExecutionMode(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)
	RegisterWorkspacePanelBindings(r)

	ctx := &Context{
		ExecutionMode:      true,
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 0,
	}

	binding := r.Resolve("a", ctx)

	if binding == nil {
		t.Fatal("expected binding for 'a' in execution mode with Resources panel focused")
	}
	if binding.Action != ActionApply {
		t.Errorf("expected ActionApply, got %v", binding.Action)
	}
}

func TestRegistry_Resolve_MainPanelToggleHunkOverridesSelect(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	ctx := &Context{FocusedPanel: PanelMain}
	binding := r.Resolve("enter", ctx)

	if binding == nil {
		t.Fatal("expected binding for enter on main panel")
	}
	if binding.Action != ActionToggleHunk {
		t.Fatalf("expected ActionToggleHunk, got %v", binding.Action)
	}
}

func TestRegistry_Resolve_MainPanelTreeArrowKeys(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	ctx := &Context{FocusedPanel: PanelMain}
	prev := r.Resolve("up", ctx)
	if prev == nil || prev.Action != ActionPrevHunk {
		t.Fatalf("expected 'up' to resolve to ActionPrevHunk, got %#v", prev)
	}
	next := r.Resolve("down", ctx)
	if next == nil || next.Action != ActionNextHunk {
		t.Fatalf("expected 'down' to resolve to ActionNextHunk, got %#v", next)
	}
	left := r.Resolve("left", ctx)
	if left == nil || left.Action != ActionTreeParent {
		t.Fatalf("expected 'left' to resolve to ActionTreeParent, got %#v", left)
	}
	right := r.Resolve("right", ctx)
	if right == nil || right.Action != ActionTreeChild {
		t.Fatalf("expected 'right' to resolve to ActionTreeChild, got %#v", right)
	}
}

func TestRegistry_Resolve_TargetModeBindingsOnResourcesTab(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	ctx := &Context{
		ExecutionMode:      true,
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 0,
		TargetMode:         true,
	}

	if binding := r.Resolve("t", ctx); binding == nil || binding.Action != ActionToggleTargetMode {
		t.Fatalf("expected t -> ActionToggleTargetMode, got %#v", binding)
	}
	if binding := r.Resolve("a", ctx); binding == nil || binding.Action != ActionToggleAllTargets {
		t.Fatalf("expected a -> ActionToggleAllTargets in target mode, got %#v", binding)
	}
	if binding := r.Resolve("A", ctx); binding == nil || binding.Action != ActionApply {
		t.Fatalf("expected A -> ActionApply in target mode, got %#v", binding)
	}
}

func TestRegistry_Resolve_NoSettingsBindingForComma(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	if binding := r.Resolve(",", NewContext()); binding != nil {
		t.Fatalf("expected no binding for ',' after settings removal, got %v", binding.Action)
	}

	modalCtx := &Context{ActiveModal: ModalHelp}
	if binding := r.Resolve(",", modalCtx); binding != nil {
		t.Fatalf("expected no modal binding for ',' after settings removal, got %v", binding.Action)
	}
}

func TestBinding_KeyString(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		expected string
	}{
		{
			name:     "empty",
			keys:     nil,
			expected: "",
		},
		{
			name:     "single key",
			keys:     []string{"q"},
			expected: "q",
		},
		{
			name:     "multiple keys returns first",
			keys:     []string{"up", "k"},
			expected: "up",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Binding{Keys: tt.keys}
			got := b.KeyString()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRegistry_RegisterHandler(t *testing.T) {
	r := NewRegistry()

	called := false
	handler := func(_ *Context) tea.Cmd {
		called = true
		return nil
	}

	r.RegisterHandler(ActionPlan, handler)

	// Verify handler is registered by using it
	r.Register(Binding{
		Keys:   []string{"p"},
		Action: ActionPlan,
		Scope:  ScopeGlobal,
	})

	ctx := NewContext()
	cmd, handled := r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}, ctx)

	if !handled {
		t.Error("expected key to be handled")
	}
	// cmd is nil because the handler returns nil
	_ = cmd
	// The handler was invoked
	if !called {
		t.Error("expected handler to be called")
	}
}

func TestRegistry_Handle_ModalBlocking(t *testing.T) {
	r := NewRegistry()

	// Register a global binding
	globalCalled := false
	r.Register(Binding{
		Keys:   []string{"x"},
		Action: ActionMoveUp,
		Scope:  ScopeGlobal,
	})
	r.RegisterHandler(ActionMoveUp, func(_ *Context) tea.Cmd {
		globalCalled = true
		return nil
	})

	// When modal is active, non-modal keys should be consumed but not execute global handler
	ctx := &Context{ActiveModal: ModalHelp}
	_, handled := r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}, ctx)

	if !handled {
		t.Error("expected key to be consumed when modal is active")
	}
	if globalCalled {
		t.Error("expected global handler NOT to be called when modal is active")
	}
}

func TestRegistry_Handle_ModalQuitAllowed(t *testing.T) {
	r := NewRegistry()

	quitCalled := false
	r.Register(Binding{
		Keys:   []string{"q"},
		Action: ActionQuit,
		Scope:  ScopeGlobal,
	})
	r.RegisterHandler(ActionQuit, func(_ *Context) tea.Cmd {
		quitCalled = true
		return nil
	})

	// q should work even when modal is active
	ctx := &Context{ActiveModal: ModalHelp}
	_, handled := r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, ctx)

	if !handled {
		t.Error("expected q to be handled even when modal is active")
	}
	if !quitCalled {
		t.Error("expected quit handler to be called")
	}
}

func TestRegistry_Handle_ModalCtrlCAllowed(t *testing.T) {
	r := NewRegistry()

	cancelCalled := false
	r.Register(Binding{
		Keys:   []string{"ctrl+c"},
		Action: ActionCancelOp,
		Scope:  ScopeGlobal,
	})
	r.RegisterHandler(ActionCancelOp, func(_ *Context) tea.Cmd {
		cancelCalled = true
		return nil
	})

	// ctrl+c should work even when modal is active
	ctx := &Context{ActiveModal: ModalHelp}
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, handled := r.Handle(msg, ctx)

	if !handled {
		t.Error("expected ctrl+c to be handled even when modal is active")
	}
	if !cancelCalled {
		t.Error("expected cancel handler to be called")
	}
}

func TestRegistry_Handle_ModalScopedBinding(t *testing.T) {
	r := NewRegistry()

	modalCalled := false
	r.Register(Binding{
		Keys:   []string{"esc"},
		Action: ActionToggleHelp,
		Scope:  ScopeModal,
		Modal:  ModalHelp,
	})
	r.RegisterHandler(ActionToggleHelp, func(_ *Context) tea.Cmd {
		modalCalled = true
		return nil
	})

	ctx := &Context{ActiveModal: ModalHelp}
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, handled := r.Handle(msg, ctx)

	if !handled {
		t.Error("expected modal binding to be handled")
	}
	if !modalCalled {
		t.Error("expected modal handler to be called")
	}
}

func TestRegistry_Handle_NoMatch(t *testing.T) {
	r := NewRegistry()

	ctx := NewContext()
	_, handled := r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}, ctx)

	if handled {
		t.Error("expected unregistered key to not be handled")
	}
}

func TestRegistry_Handle_NoHandler(t *testing.T) {
	r := NewRegistry()

	// Register binding but no handler
	r.Register(Binding{
		Keys:   []string{"p"},
		Action: ActionPlan,
		Scope:  ScopeGlobal,
	})

	ctx := NewContext()
	_, handled := r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}, ctx)

	if handled {
		t.Error("expected key without handler to not be handled")
	}
}

func TestRegistry_GetBindingsByCategory(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Category:    "General",
		Description: "quit",
	})
	r.Register(Binding{
		Keys:        []string{"p"},
		Action:      ActionPlan,
		Scope:       ScopeGlobal,
		Category:    "Execution",
		Description: "plan",
	})
	r.Register(Binding{
		Keys:     []string{"x"},
		Action:   ActionMoveUp,
		Scope:    ScopeGlobal,
		Category: "Navigation",
		Hidden:   true, // Should be excluded
	})
	r.Register(Binding{
		Keys:        []string{"y"},
		Action:      ActionMoveDown,
		Scope:       ScopeGlobal,
		Description: "test", // No category - should go to "Other"
	})

	byCategory := r.GetBindingsByCategory()

	if len(byCategory["General"]) != 1 {
		t.Errorf("expected 1 General binding, got %d", len(byCategory["General"]))
	}
	if len(byCategory["Execution"]) != 1 {
		t.Errorf("expected 1 Execution binding, got %d", len(byCategory["Execution"]))
	}
	if len(byCategory["Navigation"]) != 0 {
		t.Error("expected Navigation to be empty (hidden binding)")
	}
	if len(byCategory["Other"]) != 1 {
		t.Errorf("expected 1 Other binding, got %d", len(byCategory["Other"]))
	}
}

func TestRegistry_GetAllBindings(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{Keys: []string{"a"}, Action: ActionApply})
	r.Register(Binding{Keys: []string{"b"}, Action: ActionPlan})
	r.Register(Binding{Keys: []string{"c"}, Action: ActionQuit})

	bindings := r.GetAllBindings()

	if len(bindings) != 3 {
		t.Errorf("expected 3 bindings, got %d", len(bindings))
	}

	// Verify it's a copy
	bindings[0].Keys = []string{"modified"}
	original := r.GetAllBindings()
	if original[0].Keys[0] == "modified" {
		t.Error("GetAllBindings should return a copy, not the original slice")
	}
}

func TestRegistry_BindingCount(t *testing.T) {
	r := NewRegistry()

	if r.BindingCount() != 0 {
		t.Errorf("expected 0 bindings, got %d", r.BindingCount())
	}

	r.Register(Binding{Keys: []string{"a"}, Action: ActionApply})
	r.Register(Binding{Keys: []string{"b"}, Action: ActionPlan})

	if r.BindingCount() != 2 {
		t.Errorf("expected 2 bindings, got %d", r.BindingCount())
	}
}

func TestNewContext(t *testing.T) {
	ctx := NewContext()

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if ctx.FocusedPanel != PanelNone {
		t.Errorf("expected PanelNone, got %v", ctx.FocusedPanel)
	}
	if ctx.ActiveModal != ModalNone {
		t.Errorf("expected ModalNone, got %v", ctx.ActiveModal)
	}
	if ctx.ExecutionMode {
		t.Error("expected ExecutionMode to be false")
	}
}

func TestRegistry_bindingActiveInContext_ViewScope(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:   []string{"s"},
		Action: ActionSelect,
		Scope:  ScopeView,
		View:   ViewStateList,
	})

	// When in StateList view, binding should be active
	ctx := &Context{CurrentView: ViewStateList}
	bindings := r.GetBindingsForContext(ctx)
	if len(bindings) != 1 {
		t.Errorf("expected 1 binding in StateList view, got %d", len(bindings))
	}

	// When in different view, binding should not be active
	ctx = &Context{CurrentView: ViewMain}
	bindings = r.GetBindingsForContext(ctx)
	if len(bindings) != 0 {
		t.Errorf("expected 0 bindings in Main view, got %d", len(bindings))
	}
}

func TestBinding_Matches_ViewScope(t *testing.T) {
	b := Binding{
		Keys:  []string{"s"},
		Scope: ScopeView,
		View:  ViewCommandLog,
	}

	// Should match when in CommandLog view
	ctx := &Context{CurrentView: ViewCommandLog}
	if !b.Matches("s", ctx) {
		t.Error("expected binding to match in CommandLog view")
	}

	// Should not match in different view
	ctx = &Context{CurrentView: ViewMain}
	if b.Matches("s", ctx) {
		t.Error("expected binding to NOT match in Main view")
	}
}

func TestConditions(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		ctx       *Context
		expected  bool
	}{
		{
			name:      "ConditionExecutionMode true",
			condition: ConditionExecutionMode,
			ctx:       &Context{ExecutionMode: true},
			expected:  true,
		},
		{
			name:      "ConditionExecutionMode false",
			condition: ConditionExecutionMode,
			ctx:       &Context{ExecutionMode: false},
			expected:  false,
		},
		{
			name:      "ConditionResourcesTab true",
			condition: ConditionResourcesTab,
			ctx:       &Context{FocusedPanel: PanelResources, ResourcesActiveTab: 0},
			expected:  true,
		},
		{
			name:      "ConditionResourcesTab wrong tab",
			condition: ConditionResourcesTab,
			ctx:       &Context{FocusedPanel: PanelResources, ResourcesActiveTab: 1},
			expected:  false,
		},
		{
			name:      "ConditionResourcesTab wrong panel",
			condition: ConditionResourcesTab,
			ctx:       &Context{FocusedPanel: PanelHistory, ResourcesActiveTab: 0},
			expected:  false,
		},
		{
			name:      "ConditionStateTab true",
			condition: ConditionStateTab,
			ctx:       &Context{FocusedPanel: PanelResources, ResourcesActiveTab: 1},
			expected:  true,
		},
		{
			name:      "ConditionStateTab wrong tab",
			condition: ConditionStateTab,
			ctx:       &Context{FocusedPanel: PanelResources, ResourcesActiveTab: 0},
			expected:  false,
		},
		{
			name:      "ConditionSelectorActive true",
			condition: ConditionSelectorActive,
			ctx:       &Context{SelectorActive: true},
			expected:  true,
		},
		{
			name:      "ConditionSelectorActive false",
			condition: ConditionSelectorActive,
			ctx:       &Context{SelectorActive: false},
			expected:  false,
		},
		{
			name:      "ConditionOperationRunning true",
			condition: ConditionOperationRunning,
			ctx:       &Context{OperationRunning: true},
			expected:  true,
		},
		{
			name:      "ConditionOperationRunning false",
			condition: ConditionOperationRunning,
			ctx:       &Context{OperationRunning: false},
			expected:  false,
		},
		{
			name:      "ConditionHistoryEnabled true",
			condition: ConditionHistoryEnabled,
			ctx:       &Context{HistoryEnabled: true},
			expected:  true,
		},
		{
			name:      "ConditionHistoryEnabled false",
			condition: ConditionHistoryEnabled,
			ctx:       &Context{HistoryEnabled: false},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.condition(tt.ctx)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestBinding_EffectivePriority_DefaultScope(t *testing.T) {
	// Test the default case in EffectivePriority switch
	b := Binding{Scope: Scope(99)} // Unknown scope
	got := b.EffectivePriority()
	if got != PriorityGlobal {
		t.Errorf("expected PriorityGlobal for unknown scope, got %v", got)
	}
}

func TestRegistry_Handle_ModalNoHandler(t *testing.T) {
	r := NewRegistry()

	// Register modal binding without handler
	r.Register(Binding{
		Keys:   []string{"x"},
		Action: ActionMoveUp,
		Scope:  ScopeModal,
		Modal:  ModalHelp,
	})

	ctx := &Context{ActiveModal: ModalHelp}
	_, handled := r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}, ctx)

	// Should be handled (consumed) but no command executed
	if !handled {
		t.Error("expected modal key to be consumed even without handler")
	}
}

func TestRegistry_Handle_QuitNoBinding(t *testing.T) {
	r := NewRegistry()

	// When modal is active but q has no binding
	ctx := &Context{ActiveModal: ModalHelp}
	_, handled := r.Handle(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, ctx)

	if handled {
		t.Error("expected q without binding to not be handled")
	}
}

func TestRegisterWorkspacePanelBindings(t *testing.T) {
	r := NewRegistry()
	initialCount := r.BindingCount()

	RegisterWorkspacePanelBindings(r)

	// Currently this function does nothing, so count should be the same
	if r.BindingCount() != initialCount {
		t.Errorf("expected binding count unchanged, got %d", r.BindingCount())
	}
}

func TestBindingActiveInContext_AllScopes(t *testing.T) {
	tests := []struct {
		name     string
		binding  Binding
		ctx      *Context
		expected bool
	}{
		{
			name: "ScopeModal matches",
			binding: Binding{
				Keys:  []string{"q"},
				Scope: ScopeModal,
				Modal: ModalHelp,
			},
			ctx:      &Context{ActiveModal: ModalHelp},
			expected: true,
		},
		{
			name: "ScopeModal not matches",
			binding: Binding{
				Keys:  []string{"q"},
				Scope: ScopeModal,
				Modal: ModalHelp,
			},
			ctx:      &Context{ActiveModal: ModalTheme},
			expected: false,
		},
		{
			name: "ScopePanelTab wrong panel",
			binding: Binding{
				Keys:  []string{"p"},
				Scope: ScopePanelTab,
				Panel: PanelResources,
				Tab:   0,
			},
			ctx:      &Context{FocusedPanel: PanelHistory},
			expected: false,
		},
		{
			name: "ScopePanelTab wrong tab",
			binding: Binding{
				Keys:  []string{"p"},
				Scope: ScopePanelTab,
				Panel: PanelResources,
				Tab:   0,
			},
			ctx:      &Context{FocusedPanel: PanelResources, ResourcesActiveTab: 1},
			expected: false,
		},
		{
			name: "condition fails",
			binding: Binding{
				Keys:      []string{"p"},
				Scope:     ScopeGlobal,
				Condition: func(_ *Context) bool { return false },
			},
			ctx:      NewContext(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			r.Register(tt.binding)
			bindings := r.GetBindingsForContext(tt.ctx)
			found := len(bindings) > 0
			if found != tt.expected {
				t.Errorf("expected active=%v, got active=%v", tt.expected, found)
			}
		})
	}
}

func TestBinding_Matches_AllBranches(t *testing.T) {
	tests := []struct {
		name     string
		binding  Binding
		key      string
		ctx      *Context
		expected bool
	}{
		{
			name:     "key not in binding",
			binding:  Binding{Keys: []string{"a", "b"}},
			key:      "c",
			ctx:      NewContext(),
			expected: false,
		},
		{
			name: "ScopePanel wrong panel",
			binding: Binding{
				Keys:  []string{"p"},
				Scope: ScopePanel,
				Panel: PanelResources,
			},
			key:      "p",
			ctx:      &Context{FocusedPanel: PanelHistory},
			expected: false,
		},
		{
			name: "ScopePanelTab for non-resources panel",
			binding: Binding{
				Keys:  []string{"x"},
				Scope: ScopePanelTab,
				Panel: PanelHistory, // Not PanelResources
				Tab:   0,
			},
			key:      "x",
			ctx:      &Context{FocusedPanel: PanelHistory},
			expected: true, // Tab check only applies to PanelResources
		},
		{
			name: "ScopeModal wrong modal",
			binding: Binding{
				Keys:  []string{"q"},
				Scope: ScopeModal,
				Modal: ModalHelp,
			},
			key:      "q",
			ctx:      &Context{ActiveModal: ModalTheme},
			expected: false,
		},
		{
			name: "ScopeView wrong view",
			binding: Binding{
				Keys:  []string{"s"},
				Scope: ScopeView,
				View:  ViewStateList,
			},
			key:      "s",
			ctx:      &Context{CurrentView: ViewMain},
			expected: false,
		},
		{
			name: "condition passes",
			binding: Binding{
				Keys:      []string{"p"},
				Scope:     ScopeGlobal,
				Condition: func(_ *Context) bool { return true },
			},
			key:      "p",
			ctx:      NewContext(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.binding.Matches(tt.key, tt.ctx)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestRegisterDefaults_InNonExecutionMode(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, false)

	// Should still register all bindings (execution flag is ignored now)
	count := r.BindingCount()
	if count == 0 {
		t.Error("expected bindings to be registered")
	}

	// Check that execution bindings are registered (they use conditions)
	ctx := &Context{
		ExecutionMode:      true,
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 0,
	}
	binding := r.Resolve("p", ctx)
	if binding == nil || binding.Action != ActionPlan {
		t.Error("expected plan binding to be resolvable in execution mode")
	}
}

func TestConditionsWithHistory(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	// Test history condition on focus history binding
	ctx := &Context{
		ExecutionMode:  true,
		HistoryEnabled: true,
	}
	binding := r.Resolve("3", ctx)
	if binding == nil {
		t.Error("expected '3' to resolve when history enabled")
	}

	ctx = &Context{
		ExecutionMode:  true,
		HistoryEnabled: false,
	}
	binding = r.Resolve("3", ctx)
	if binding != nil {
		t.Error("expected '3' to NOT resolve when history disabled")
	}
}

func TestHistoryToggleCondition(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	// 'h' toggles history - requires both ExecutionMode and HistoryEnabled
	tests := []struct {
		name           string
		executionMode  bool
		historyEnabled bool
		shouldResolve  bool
	}{
		{"both enabled", true, true, true},
		{"execution only", true, false, false},
		{"history only", false, true, false},
		{"neither", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{
				ExecutionMode:  tt.executionMode,
				HistoryEnabled: tt.historyEnabled,
			}
			binding := r.Resolve("h", ctx)
			resolved := binding != nil && binding.Action == ActionToggleHistory
			if resolved != tt.shouldResolve {
				t.Errorf("expected resolved=%v, got resolved=%v", tt.shouldResolve, resolved)
			}
		})
	}
}

func TestBindingActiveInContext_NonResourcesPanelTab(t *testing.T) {
	r := NewRegistry()

	// Register a ScopePanelTab binding for a non-Resources panel
	r.Register(Binding{
		Keys:  []string{"x"},
		Scope: ScopePanelTab,
		Panel: PanelHistory,
		Tab:   0,
	})

	// Should be active when History panel is focused (tab check only applies to Resources)
	ctx := &Context{FocusedPanel: PanelHistory}
	bindings := r.GetBindingsForContext(ctx)
	if len(bindings) != 1 {
		t.Errorf("expected 1 binding for History panel, got %d", len(bindings))
	}

	// Should NOT be active when different panel is focused
	ctx = &Context{FocusedPanel: PanelResources}
	bindings = r.GetBindingsForContext(ctx)
	if len(bindings) != 0 {
		t.Errorf("expected 0 bindings for Resources panel, got %d", len(bindings))
	}
}

func TestBinding_Matches_PanelTabResourcesMatchingTab(t *testing.T) {
	// Test the case where PanelResources tab matches
	b := Binding{
		Keys:  []string{"p"},
		Scope: ScopePanelTab,
		Panel: PanelResources,
		Tab:   0,
	}

	// Should match when on correct panel and tab
	ctx := &Context{
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 0,
	}
	if !b.Matches("p", ctx) {
		t.Error("expected binding to match on Resources tab 0")
	}

	// Should not match on wrong tab
	ctx = &Context{
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 1,
	}
	if b.Matches("p", ctx) {
		t.Error("expected binding to NOT match on Resources tab 1")
	}
}

func TestFocusModeBindingsResolveOnlyInExecutionMode(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	execCtx := &Context{ExecutionMode: true}
	next := r.Resolve("+", execCtx)
	if next == nil || next.Action != ActionFocusModeNext {
		t.Fatalf("expected '+' to resolve to ActionFocusModeNext in execution mode")
	}
	prev := r.Resolve("_", execCtx)
	if prev == nil || prev.Action != ActionFocusModePrev {
		t.Fatalf("expected '_' to resolve to ActionFocusModePrev in execution mode")
	}

	nonExecCtx := &Context{ExecutionMode: false}
	if binding := r.Resolve("+", nonExecCtx); binding != nil {
		t.Fatalf("expected '+' to not resolve outside execution mode")
	}
	if binding := r.Resolve("_", nonExecCtx); binding != nil {
		t.Fatalf("expected '_' to not resolve outside execution mode")
	}
}

func TestRegistry_Resolve_ToggleStatusBinding(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, false)

	ctx := &Context{
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 0,
	}
	binding := r.Resolve("s", ctx)
	if binding == nil {
		t.Fatal("expected 's' binding to resolve")
	}
	if binding.Action != ActionToggleStatus {
		t.Fatalf("expected ActionToggleStatus, got %v", binding.Action)
	}
	if binding.Scope != ScopePanelTab {
		t.Fatalf("expected panel-tab binding precedence for resources tab, got scope %v", binding.Scope)
	}

	nonResourcesCtx := &Context{
		FocusedPanel:       PanelWorkspace,
		ResourcesActiveTab: 0,
	}
	nonResourcesBinding := r.Resolve("s", nonResourcesCtx)
	if nonResourcesBinding == nil {
		t.Fatal("expected 's' binding to resolve outside resources panel too")
	}
	if nonResourcesBinding.Scope != ScopeGlobal {
		t.Fatalf("expected global binding outside resources panel, got scope %v", nonResourcesBinding.Scope)
	}
}

func TestBinding_Matches_GlobalWithCondition(t *testing.T) {
	// Test global binding with condition that passes
	conditionCalled := false
	b := Binding{
		Keys:  []string{"g"},
		Scope: ScopeGlobal,
		Condition: func(_ *Context) bool {
			conditionCalled = true
			return true
		},
	}

	ctx := NewContext()
	result := b.Matches("g", ctx)

	if !conditionCalled {
		t.Error("expected condition to be called")
	}
	if !result {
		t.Error("expected binding to match when condition passes")
	}
}
