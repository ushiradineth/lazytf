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
		Description: "cancel/quit",
		Category:    "General",
	})

	// Help
	r.Register(Binding{
		Keys:        []string{"?"},
		Action:      ActionToggleHelp,
		Scope:       ScopeGlobal,
		Description: "keybinds",
		Category:    "General",
	})

	// Theme
	r.Register(Binding{
		Keys:        []string{"T"},
		Action:      ActionToggleTheme,
		Scope:       ScopeGlobal,
		Description: "theme",
		Category:    "General",
	})
}

//nolint:funlen // Binding registration is a flat declaration table for readability.
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
		Description: "next panel",
		Category:    "Panel Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"shift+tab"},
		Action:      ActionCycleFocusBack,
		Scope:       ScopeGlobal,
		Description: "prev panel",
		Category:    "Panel Navigation",
	})

	// Command log toggle
	r.Register(Binding{
		Keys:        []string{"L"},
		Action:      ActionToggleLog,
		Scope:       ScopeGlobal,
		Description: "command log",
		Category:    "Panel Navigation",
		Condition:   ConditionExecutionMode,
	})

	// Focus mode cycling
	r.Register(Binding{
		Keys:        []string{"+"},
		Action:      ActionFocusModeNext,
		Scope:       ScopeGlobal,
		Description: "focus mode +",
		Category:    "Panel Navigation",
		Condition:   ConditionExecutionMode,
	})
	r.Register(Binding{
		Keys:        []string{"_"},
		Action:      ActionFocusModePrev,
		Scope:       ScopeGlobal,
		Description: "focus mode -",
		Category:    "Panel Navigation",
		Condition:   ConditionExecutionMode,
	})

	// Escape
	r.Register(Binding{
		Keys:        []string{KeyEsc},
		Action:      ActionEscapeBack,
		Scope:       ScopeGlobal,
		Description: "back to resources",
		Category:    "Panel Navigation",
		Hidden:      true,
	})
}

func registerNavigationBindings(r *Registry) {
	registerDirectionalBindings(r)
	registerPagingBindings(r)
	registerSelectionBindings(r)
	registerMainTreeNavigationBindings(r)
}

func registerDirectionalBindings(r *Registry) {
	r.Register(Binding{
		Keys:        []string{"up", "k"},
		Action:      ActionMoveUp,
		Scope:       ScopeGlobal,
		Description: "up",
		Category:    "Navigation",
	})
	r.Register(Binding{
		Keys:        []string{KeyDown, "j"},
		Action:      ActionMoveDown,
		Scope:       ScopeGlobal,
		Description: "down",
		Category:    "Navigation",
	})
}

func registerPagingBindings(r *Registry) {
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
		Description: "top",
		Category:    "Navigation",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"end", "G"},
		Action:      ActionScrollEnd,
		Scope:       ScopeGlobal,
		Description: "bottom",
		Category:    "Navigation",
		Hidden:      true,
	})
}

func registerSelectionBindings(r *Registry) {
	r.Register(Binding{
		Keys:        []string{"enter", " "},
		Action:      ActionSelect,
		Scope:       ScopeGlobal,
		Description: "select",
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

func registerMainTreeNavigationBindings(r *Registry) {
	r.Register(Binding{
		Keys:        []string{"up", "k"},
		Action:      ActionPrevHunk,
		Scope:       ScopePanel,
		Panel:       PanelMain,
		Description: "prev node",
		Category:    "Navigation",
	})
	r.Register(Binding{
		Keys:        []string{KeyDown, "j"},
		Action:      ActionNextHunk,
		Scope:       ScopePanel,
		Panel:       PanelMain,
		Description: "next node",
		Category:    "Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"left", "h"},
		Action:      ActionTreeParent,
		Scope:       ScopePanel,
		Panel:       PanelMain,
		Description: "parent",
		Category:    "Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"right", "l"},
		Action:      ActionTreeChild,
		Scope:       ScopePanel,
		Panel:       PanelMain,
		Description: "child",
		Category:    "Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"enter", " "},
		Action:      ActionToggleHunk,
		Scope:       ScopePanel,
		Panel:       PanelMain,
		Description: "toggle node",
		Category:    "Navigation",
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
		Description: "create filter",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"u"},
		Action:      ActionToggleUpdate,
		Scope:       ScopeGlobal,
		Description: "update filter",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"d"},
		Action:      ActionToggleDelete,
		Scope:       ScopeGlobal,
		Description: "delete filter",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"r"},
		Action:      ActionToggleReplace,
		Scope:       ScopeGlobal,
		Description: "replace filter",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"s"},
		Action:      ActionToggleStatus,
		Scope:       ScopeGlobal,
		Description: "status",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"t"},
		Action:      ActionToggleTargetMode,
		Scope:       ScopeGlobal,
		Description: "target mode",
		Category:    "Resources Panel",
		Condition: func(ctx *Context) bool {
			return ConditionExecutionMode(ctx) && ConditionTargetAvailable(ctx)
		},
	})

	// Panel-scoped filter bindings (higher priority, for execution mode on Resources tab)
	r.Register(Binding{
		Keys:        []string{"c"},
		Action:      ActionToggleCreate,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "create filter",
		Category:    "Resources Panel",
		Hidden:      true, // Don't show duplicate in help
	})
	r.Register(Binding{
		Keys:        []string{"u"},
		Action:      ActionToggleUpdate,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "update filter",
		Category:    "Resources Panel",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"d"},
		Action:      ActionToggleDelete,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "delete filter",
		Category:    "Resources Panel",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"r"},
		Action:      ActionToggleReplace,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "replace filter",
		Category:    "Resources Panel",
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"s"},
		Action:      ActionToggleStatus,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "status",
		Category:    "Resources Panel",
		Condition: func(ctx *Context) bool {
			return !ctx.TargetMode
		},
		Hidden: true,
	})
	r.Register(Binding{
		Keys:        []string{"t"},
		Action:      ActionToggleTargetMode,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "target mode",
		Category:    "Resources Panel",
		Condition: func(ctx *Context) bool {
			return ConditionExecutionMode(ctx) && ConditionTargetAvailable(ctx)
		},
	})
	r.Register(Binding{
		Keys:        []string{"enter", " "},
		Action:      ActionToggleTarget,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "target select",
		Category:    "Resources Panel",
		Condition: func(ctx *Context) bool {
			return ConditionExecutionMode(ctx) && ConditionTargetMode(ctx)
		},
	})
	r.Register(Binding{
		Keys:        []string{"s"},
		Action:      ActionToggleAllTargets,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "target all",
		Category:    "Resources Panel",
		Condition: func(ctx *Context) bool {
			return ConditionExecutionMode(ctx) && ConditionTargetMode(ctx)
		},
	})
	r.Register(Binding{
		Keys:        []string{"y"},
		Action:      ActionCopyAddress,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Description: "yank",
		Category:    "Resources Panel",
	})
	r.Register(Binding{
		Keys:        []string{"y"},
		Action:      ActionCopyAddress,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         1,
		Description: "yank",
		Category:    "Resources Panel",
		Condition:   ConditionExecutionMode,
	})

	// Tab switching
	r.Register(Binding{
		Keys:        []string{"["},
		Action:      ActionSwitchTabPrev,
		Scope:       ScopePanel,
		Panel:       PanelResources,
		Description: "prev tab",
		Category:    "Resources Panel",
		Condition:   ConditionExecutionMode,
		Hidden:      true,
	})
	r.Register(Binding{
		Keys:        []string{"]"},
		Action:      ActionSwitchTabNext,
		Scope:       ScopePanel,
		Panel:       PanelResources,
		Description: "next tab",
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
		Tab:         0,
		Condition:   ConditionExecutionMode,
		Description: "init",
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
		Keys:        []string{"I"},
		Action:      ActionInitUpgrade,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         0,
		Condition:   ConditionExecutionMode,
		Description: "init upgrade",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"I"},
		Action:      ActionInitUpgrade,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         1,
		Condition:   ConditionExecutionMode,
		Description: "init upgrade",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"r"},
		Action:      ActionRefresh,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         1,
		Condition:   ConditionExecutionMode,
		Description: "state reload",
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
		Description: "state rm",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"m"},
		Action:      ActionStateMove,
		Scope:       ScopePanelTab,
		Panel:       PanelResources,
		Tab:         1,
		Condition:   ConditionExecutionMode,
		Description: "state mv",
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
		Description: "history panel",
		Category:    "Execution",
	})

	// Focus command log from diagnostics
	r.Register(Binding{
		Keys:        []string{"D"},
		Action:      ActionFocusCommandLog,
		Scope:       ScopeGlobal,
		Condition:   ConditionExecutionMode,
		Description: "focus logs",
		Category:    "Execution",
		Hidden:      true,
	})
}

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

	// Theme modal
	r.Register(Binding{
		Keys:   []string{"T", "esc"},
		Action: ActionToggleTheme,
		Scope:  ScopeModal,
		Modal:  ModalTheme,
		Hidden: true,
	})
	r.Register(Binding{
		Keys:   []string{"enter"},
		Action: ActionSelect,
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
