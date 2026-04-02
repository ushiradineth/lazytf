package keybinds

// Action represents a named operation that can be triggered by a keybind.
type Action string

// Global actions (work everywhere).
const (
	ActionQuit        Action = "quit"
	ActionCancelOp    Action = "cancel_operation"
	ActionToggleHelp  Action = "toggle_help"
	ActionToggleTheme Action = "toggle_theme"
)

// Panel navigation actions.
const (
	ActionFocusWorkspace  Action = "focus_workspace"
	ActionFocusResources  Action = "focus_resources"
	ActionFocusHistory    Action = "focus_history"
	ActionFocusMain       Action = "focus_main"
	ActionFocusCommandLog Action = "focus_command_log"
	ActionCycleFocus      Action = "cycle_focus"
	ActionCycleFocusBack  Action = "cycle_focus_back"
	ActionToggleLog       Action = "toggle_command_log"
	ActionFocusModeNext   Action = "focus_mode_next"
	ActionFocusModePrev   Action = "focus_mode_prev"
	ActionEscapeBack      Action = "escape_back"
	ActionToggleHistory   Action = "toggle_history"
)

// Execution actions (require execution mode).
const (
	ActionInit        Action = "init"
	ActionInitUpgrade Action = "init_upgrade"
	ActionPlan        Action = "plan"
	ActionApply       Action = "apply"
	ActionRefresh     Action = "refresh"
	ActionValidate    Action = "validate"
	ActionFormat      Action = "format"
)

// Filter actions (resources tab only).
const (
	ActionToggleCreate    Action = "toggle_filter_create"
	ActionToggleUpdate    Action = "toggle_filter_update"
	ActionToggleDelete    Action = "toggle_filter_delete"
	ActionToggleReplace   Action = "toggle_filter_replace"
	ActionToggleAllGroups Action = "toggle_all_groups"
	ActionToggleStatus    Action = "toggle_status"
	ActionStateRemove     Action = "state_remove"
	ActionStateMove       Action = "state_move"
	ActionCopyAddress     Action = "copy_selected_address"
)

// Tab actions.
const (
	ActionSwitchTabPrev Action = "switch_tab_prev"
	ActionSwitchTabNext Action = "switch_tab_next"
)

// Navigation actions.
const (
	ActionMoveUp     Action = "move_up"
	ActionMoveDown   Action = "move_down"
	ActionPageUp     Action = "page_up"
	ActionPageDown   Action = "page_down"
	ActionScrollTop  Action = "scroll_top"
	ActionScrollEnd  Action = "scroll_end"
	ActionPrevHunk   Action = "prev_hunk"
	ActionNextHunk   Action = "next_hunk"
	ActionToggleHunk Action = "toggle_hunk"
	ActionTreeParent Action = "tree_parent"
	ActionTreeChild  Action = "tree_child"
	ActionSelect     Action = "select"
	ActionScrollUp   Action = "scroll_up"
	ActionScrollDown Action = "scroll_down"
)

// Environment actions.
const (
	ActionSelectEnv Action = "select_environment"
)

// Modal actions.
const (
	ActionConfirmYes Action = "confirm_yes"
	ActionConfirmNo  Action = "confirm_no"
)
