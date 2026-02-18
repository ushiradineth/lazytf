package keybinds

// RegisterDefaults registers all default keybindings.
// The executionMode parameter is kept for backward compatibility but execution
// bindings are always registered. They use ConditionExecutionMode to only match
// when the context has ExecutionMode=true.
func RegisterDefaults(r *Registry, _ bool) {
	registerGlobalBindings(r)
	registerNavigationBindings(r)
	registerPanelNavigationBindings(r)
	registerResourcesPanelBindings(r)
	registerModalBindings(r)
	// Always register execution bindings - they have ConditionExecutionMode
	// which will prevent them from matching when not in execution mode
	registerExecutionBindings(r)
}

func registerGlobalBindings(r *Registry) {
	// Quit
	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Category:    "General",
	})
	r.Register(Binding{
		Keys:        []string{KeyCtrlC},
		Action:      ActionCancelOp,
		Scope:       ScopeGlobal,
		Description: "quit / cancel operation",
		Category:    "General",
	})

	// Help
	r.Register(Binding{
		Keys:        []string{"?"},
		Action:      ActionToggleHelp,
		Scope:       ScopeGlobal,
		Description: "toggle keybinds",
		Category:    "General",
	})

	// Settings
	r.Register(Binding{
		Keys:        []string{","},
		Action:      ActionToggleConfig,
		Scope:       ScopeGlobal,
		Description: "open settings",
		Category:    "General",
	})

	// Theme
	r.Register(Binding{
		Keys:        []string{"T"},
		Action:      ActionToggleTheme,
		Scope:       ScopeGlobal,
		Description: "change theme",
		Category:    "General",
	})
}

func registerPanelNavigationBindings(r *Registry) {
	// Number keys for panel focus - hidden since panel titles show [1], [2], etc.
	r.Register(Binding{
		Keys:      []string{"1"},
		Action:    ActionFocusWorkspace,
		Scope:     ScopeGlobal,
		Category:  "Panel Navigation",
		Condition: ConditionExecutionMode,
		Hidden:    true,
	})
	r.Register(Binding{
		Keys:     []string{"2"},
		Action:   ActionFocusResources,
		Scope:    ScopeGlobal,
		Category: "Panel Navigation",
		Hidden:   true,
	})
	r.Register(Binding{
		Keys:     []string{"3"},
		Action:   ActionFocusHistory,
		Scope:    ScopeGlobal,
		Category: "Panel Navigation",
		Condition: func(ctx *Context) bool {
			return ctx.ExecutionMode && ctx.HistoryEnabled
		},
		Hidden: true,
	})
	r.Register(Binding{
		Keys:      []string{"4"},
		Action:    ActionFocusCommandLog,
		Scope:     ScopeGlobal,
		Category:  "Panel Navigation",
		Condition: ConditionExecutionMode,
		Hidden:    true,
	})
	r.Register(Binding{
		Keys:     []string{"0"},
		Action:   ActionFocusMain,
		Scope:    ScopeGlobal,
		Category: "Panel Navigation",
		Hidden:   true,
	})

	// Tab cycling
	r.Register(Binding{
		Keys:        []string{"tab"},
		Action:      ActionCycleFocus,
		Scope:       ScopeGlobal,
		Description: "cycle panels",
		Category:    "Panel Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"shift+tab"},
		Action:      ActionCycleFocusBack,
		Scope:       ScopeGlobal,
		Description: "cycle panels (reverse)",
		Category:    "Panel Navigation",
		Hidden:      true,
	})

	// Command log toggle
	r.Register(Binding{
		Keys:        []string{"L"},
		Action:      ActionToggleLog,
		Scope:       ScopeGlobal,
		Description: "toggle command log",
		Category:    "Panel Navigation",
		Condition:   ConditionExecutionMode,
	})

	// Escape
	r.Register(Binding{
		Keys:        []string{KeyEsc},
		Action:      ActionEscapeBack,
		Scope:       ScopeGlobal,
		Description: "return to resource list",
		Category:    "Panel Navigation",
		Hidden:      true,
	})
}

func registerNavigationBindings(r *Registry) {
	// Up/Down
	r.Register(Binding{
		Keys:        []string{"up", "k"},
		Action:      ActionMoveUp,
		Scope:       ScopeGlobal,
		Description: "move selection up",
		Category:    "Navigation",
	})
	r.Register(Binding{
		Keys:        []string{KeyDown, "j"},
		Action:      ActionMoveDown,
		Scope:       ScopeGlobal,
		Description: "move selection down",
		Category:    "Navigation",
	})

	// Page Up/Down
	r.Register(Binding{
		Keys:        []string{"pgup"},
		Action:      ActionPageUp,
		Scope:       ScopeGlobal,
		Description: "page up",
		Category:    "Navigation",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"pgdown"},
		Action:      ActionPageDown,
		Scope:       ScopeGlobal,
		Description: "page down",
		Category:    "Navigation",
		Hidden:      true,
	})

	// Home/End
	r.Register(Binding{
		Keys:        []string{"home", "g"},
		Action:      ActionScrollTop,
		Scope:       ScopeGlobal,
		Description: "scroll to top",
		Category:    "Navigation",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"end", "G"},
		Action:      ActionScrollEnd,
		Scope:       ScopeGlobal,
		Description: "scroll to bottom",
		Category:    "Navigation",
		Hidden:      true,
	})

	// Select/Toggle
	r.Register(Binding{
		Keys:        []string{"enter", " "},
		Action:      ActionSelect,
		Scope:       ScopeGlobal,
		Description: "toggle group / select",
		Category:    "Navigation",
	})

	// Panel-specific select hints
	r.Register(Binding{
		Keys:        []string{"enter"},
		Action:      ActionSelect,
		Scope:       ScopePanel,
		Panel:       PanelWorkspace,
		Description: "select",
		Category:    "Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"enter"},
		Action:      ActionSelect,
		Scope:       ScopePanel,
		Panel:       PanelHistory,
		Description: "select",
		Category:    "Navigation",
		Condition:   ConditionExecutionMode,
	})
}

//nolint:funlen // Keybind registration is naturally verbose
func registerResourcesPanelBindings(r *Registry) {
	// Filter toggles - registered as global in non-execution mode, panel-scoped in execution mode
	// The execution mode variant (ScopePanelTab) has higher priority and will take precedence
	// when the Resources panel is focused in execution mode.

	// Global filter bindings (work in non-execution mode, or when other panels are focused)
	r.Register(Binding{
		Keys:        []string{"c"},
		Action:      ActionToggleCreate,
		Scope:       ScopeGlobal,
		Description: "toggle create filter",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"u"},
		Action:      ActionToggleUpdate,
		Scope:       ScopeGlobal,
		Description: "toggle update filter",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"d"},
		Action:      ActionToggleDelete,
		Scope:       ScopeGlobal,
		Description: "toggle delete filter",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"r"},
		Action:      ActionToggleReplace,
		Scope:       ScopeGlobal,
		Description: "toggle replace filter",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"t"},
		Action:      ActionToggleAllGroups,
		Scope:       ScopeGlobal,
		Description: "toggle all groups",
		Category:    "Resources Panel",
	})

	// Panel-scoped filter bindings (higher priority, for execution mode on Resources tab)
	r.Register(Binding{
		Keys:        []string{"c"},
		Action:      ActionToggleCreate,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "toggle create filter",
		Category:    "Resources Panel",
		Hidden:      true, // Don't show duplicate in help
	})
	r.Register(Binding{
		Keys:        []string{"u"},
		Action:      ActionToggleUpdate,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "toggle update filter",
		Category:    "Resources Panel",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"d"},
		Action:      ActionToggleDelete,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "toggle delete filter",
		Category:    "Resources Panel",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"r"},
		Action:      ActionToggleReplace,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "toggle replace filter",
		Category:    "Resources Panel",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"t"},
		Action:      ActionToggleAllGroups,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "toggle all groups",
		Category:    "Resources Panel",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"y"},
		Action:      ActionCopyAddress,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "copy selected address",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"y"},
		Action:      ActionCopyAddress,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         1,
		Description: "copy selected address",
		Category:    "Resources Panel",
		Condition:   ConditionExecutionMode,
	})

	// Tab switching
	r.Register(Binding{
		Keys:        []string{"["},
		Action:      ActionSwitchTabPrev,
		Scope:       ScopePanel,
		Panel:       PanelResources,
		Description: "switch to previous tab",
		Category:    "Resources Panel",
		Condition:   ConditionExecutionMode,
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"]"},
		Action:      ActionSwitchTabNext,
		Scope:       ScopePanel,
		Panel:       PanelResources,
		Description: "switch to next tab",
		Category:    "Resources Panel",
		Condition:   ConditionExecutionMode,
		Hidden:      true,
	})
}

//nolint:funlen // Keybind registration is naturally verbose
func registerExecutionBindings(r *Registry) {
	// Terraform commands (Resources tab only in execution mode)
	r.Register(Binding{
		Keys:        []string{"p"},
		Action:      ActionPlan,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Condition:   ConditionExecutionMode,
		Description: "plan",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"a"},
		Action:      ActionApply,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Condition:   ConditionExecutionMode,
		Description: "apply",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"i"},
		Action:      ActionInit,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         1,
		Condition:   ConditionExecutionMode,
		Description: "init",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"r"},
		Action:      ActionRefresh,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         1,
		Condition:   ConditionExecutionMode,
		Description: "reload state list",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"v"},
		Action:      ActionValidate,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Condition:   ConditionExecutionMode,
		Description: "validate",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"f"},
		Action:      ActionFormat,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Condition:   ConditionExecutionMode,
		Description: "format",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"x"},
		Action:      ActionStateRemove,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         1,
		Condition:   ConditionExecutionMode,
		Description: "remove selected state",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"m"},
		Action:      ActionStateMove,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         1,
		Condition:   ConditionExecutionMode,
		Description: "move selected state",
		Category:    "Execution",
	})
	// History toggle (global in execution mode with history enabled)
	r.Register(Binding{
		Keys:   []string{"h"},
		Action: ActionToggleHistory,
		Scope:  ScopeGlobal,
		Condition: func(ctx *Context) bool {
			return ctx.ExecutionMode && ctx.HistoryEnabled
		},
		Description: "toggle history panel",
		Category:    "Execution",
	})

	// Focus command log from diagnostics
	r.Register(Binding{
		Keys:        []string{"D"},
		Action:      ActionFocusCommandLog,
		Scope:       ScopeGlobal,
		Condition:   ConditionExecutionMode,
		Description: "focus logs panel",
		Category:    "Execution",
		Hidden:      true,
	})
}

//nolint:funlen // Keybind registration is naturally verbose
func registerModalBindings(r *Registry) {
	// Help modal
	r.Register(Binding{
		Keys:   []string{"?", "esc"},
		Action: ActionToggleHelp,
		Scope:  ScopeModal,
		Modal:  ModalHelp,
		Hidden: true,
	})
	r.Register(Binding{
		Keys:   []string{"j", "down"},
		Action: ActionScrollDown,
		Scope:  ScopeModal,
		Modal:  ModalHelp,
		Hidden: true,
	})
	r.Register(Binding{
		Keys:   []string{"k", "up"},
		Action: ActionScrollUp,
		Scope:  ScopeModal,
		Modal:  ModalHelp,
		Hidden: true,
	})

	// Settings modal
	r.Register(Binding{
		Keys:   []string{",", "esc"},
		Action: ActionToggleConfig,
		Scope:  ScopeModal,
		Modal:  ModalSettings,
		Hidden: true,
	})
	r.Register(Binding{
		Keys:   []string{"j", "down"},
		Action: ActionScrollDown,
		Scope:  ScopeModal,
		Modal:  ModalSettings,
		Hidden: true,
	})
	r.Register(Binding{
		Keys:   []string{"k", "up"},
		Action: ActionScrollUp,
		Scope:  ScopeModal,
		Modal:  ModalSettings,
		Hidden: true,
	})

	// Theme modal
	r.Register(Binding{
		Keys:   []string{"T", "esc"},
		Action: ActionToggleTheme,
		Scope:  ScopeModal,
		Modal:  ModalTheme,
		Hidden: true,
	})
	r.Register(Binding{
		Keys:   []string{"j", "down"},
		Action: ActionScrollDown,
		Scope:  ScopeModal,
		Modal:  ModalTheme,
		Hidden: true,
	})
	r.Register(Binding{
		Keys:   []string{"k", "up"},
		Action: ActionScrollUp,
		Scope:  ScopeModal,
		Modal:  ModalTheme,
		Hidden: true,
	})

	// Confirm apply modal
	r.Register(Binding{
		Keys:        []string{"y", "Y"},
		Action:      ActionConfirmYes,
		Scope:       ScopeModal,
		Modal:       ModalConfirmApply,
		Description: "confirm apply",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"n", "N", "esc"},
		Action:      ActionConfirmNo,
		Scope:       ScopeModal,
		Modal:       ModalConfirmApply,
		Description: "cancel apply",
		Hidden:      true,
	})
}

// RegisterWorkspacePanelBindings registers workspace panel keybindings.
func RegisterWorkspacePanelBindings(_ *Registry) {
	// No bindings currently - environment selection handled by the panel itself
}
