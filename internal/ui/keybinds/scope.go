package keybinds

// Scope defines the context in which a keybind is active.
type Scope int

const (
	// ScopeGlobal bindings are active everywhere.
	ScopeGlobal Scope = iota
	// ScopePanel bindings are active when a specific panel is focused.
	ScopePanel
	// ScopePanelTab bindings are active when a specific panel+tab is focused.
	ScopePanelTab
	// ScopeModal bindings are active when a modal is open.
	ScopeModal
	// ScopeView bindings are active in a specific view (e.g., state list, command log fullscreen).
	ScopeView
)

// Priority determines which binding takes precedence when multiple match.
// Higher priority wins.
type Priority int

const (
	// PriorityGlobal is for global keybinds (lowest priority).
	PriorityGlobal Priority = 0
	// PriorityPanel is for panel-specific keybinds.
	PriorityPanel Priority = 10
	// PriorityPanelTab is for panel+tab-specific keybinds.
	PriorityPanelTab Priority = 15
	// PriorityView is for view-specific keybinds.
	PriorityView Priority = 20
	// PriorityModal is for modal keybinds (highest priority).
	PriorityModal Priority = 100
)

// PanelID identifies a UI panel.
type PanelID int

const (
	PanelNone PanelID = iota
	PanelWorkspace
	PanelResources
	PanelHistory
	PanelMain
	PanelCommandLog
)

// ModalID identifies a modal type.
type ModalID int

const (
	ModalNone ModalID = iota
	ModalHelp
	ModalSettings
	ModalConfirmApply
	ModalTheme
)

// ViewID identifies a view.
type ViewID int

const (
	ViewMain ViewID = iota
	ViewPlanOutput
	ViewApplyOutput
	ViewCommandLog
	ViewStateList
	ViewStateShow
)

// Condition is a function that checks if a binding is active in the current context.
type Condition func(ctx *Context) bool

// ConditionNone always returns true (no condition).
var ConditionNone Condition

// ConditionExecutionMode checks if the app is in execution mode.
var ConditionExecutionMode Condition = func(ctx *Context) bool {
	return ctx.ExecutionMode
}

// ConditionResourcesTab checks if the Resources tab (not State) is active.
var ConditionResourcesTab Condition = func(ctx *Context) bool {
	return ctx.FocusedPanel == PanelResources && ctx.ResourcesActiveTab == 0
}

// ConditionStateTab checks if the State tab is active.
var ConditionStateTab Condition = func(ctx *Context) bool {
	return ctx.FocusedPanel == PanelResources && ctx.ResourcesActiveTab == 1
}

// ConditionSelectorActive checks if the environment selector is active.
var ConditionSelectorActive Condition = func(ctx *Context) bool {
	return ctx.SelectorActive
}

// ConditionOperationRunning checks if a terraform operation is running.
var ConditionOperationRunning Condition = func(ctx *Context) bool {
	return ctx.OperationRunning
}

// Context represents the current UI state for keybind resolution.
type Context struct {
	// Focus state
	FocusedPanel PanelID
	ActiveModal  ModalID
	CurrentView  ViewID

	// Mode state
	ExecutionMode  bool
	SelectorActive bool

	// Tab state
	ResourcesActiveTab int // 0 = Resources, 1 = State

	// Operation state
	OperationRunning bool
	PlanRunning      bool
	ApplyRunning     bool
	RefreshRunning   bool
}

// NewContext creates an empty context.
func NewContext() *Context {
	return &Context{}
}
