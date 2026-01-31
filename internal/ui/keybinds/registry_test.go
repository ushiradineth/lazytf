package keybinds

import (
	"testing"
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
