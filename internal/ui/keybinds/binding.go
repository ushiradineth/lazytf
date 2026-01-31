package keybinds

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Binding links a key to an action with scope and conditions.
type Binding struct {
	// Keys are the key strings that trigger this binding (e.g., "p", "ctrl+c").
	Keys []string

	// Action is the operation to perform.
	Action Action

	// Scope determines where this binding is active.
	Scope Scope

	// Priority determines precedence when multiple bindings match.
	// If 0, defaults based on Scope.
	Priority Priority

	// Panel restricts this binding to a specific panel (for ScopePanel/ScopePanelTab).
	Panel PanelID

	// Tab restricts this binding to a specific tab within the panel (for ScopePanelTab).
	Tab int

	// Modal restricts this binding to a specific modal (for ScopeModal).
	Modal ModalID

	// View restricts this binding to a specific view (for ScopeView).
	View ViewID

	// Condition is an additional check that must pass for the binding to be active.
	Condition Condition

	// Description is shown in help text.
	Description string

	// Category groups related bindings in help display.
	Category string

	// Hidden bindings are not shown in help text.
	Hidden bool
}

// Matches checks if this binding matches the given key and context.
func (b *Binding) Matches(key string, ctx *Context) bool {
	// Check if key matches
	if !slices.Contains(b.Keys, key) {
		return false
	}

	// Check scope-specific conditions
	switch b.Scope {
	case ScopeGlobal:
		// Global bindings always match (unless condition fails)
	case ScopePanel:
		if ctx.FocusedPanel != b.Panel {
			return false
		}
	case ScopePanelTab:
		if ctx.FocusedPanel != b.Panel {
			return false
		}
		if b.Panel == PanelResources && ctx.ResourcesActiveTab != b.Tab {
			return false
		}
	case ScopeModal:
		if ctx.ActiveModal != b.Modal {
			return false
		}
	case ScopeView:
		if ctx.CurrentView != b.View {
			return false
		}
	}

	// Check additional condition
	if b.Condition != nil && !b.Condition(ctx) {
		return false
	}

	return true
}

// EffectivePriority returns the priority to use for this binding.
func (b *Binding) EffectivePriority() Priority {
	if b.Priority != 0 {
		return b.Priority
	}

	switch b.Scope {
	case ScopeGlobal:
		return PriorityGlobal
	case ScopePanel:
		return PriorityPanel
	case ScopePanelTab:
		return PriorityPanelTab
	case ScopeView:
		return PriorityView
	case ScopeModal:
		return PriorityModal
	default:
		return PriorityGlobal
	}
}

// Handler is a function that executes an action and returns a command.
type Handler func(ctx *Context) tea.Cmd

// KeyString returns the first key for display purposes.
func (b *Binding) KeyString() string {
	if len(b.Keys) == 0 {
		return ""
	}
	return b.Keys[0]
}

// AllKeysString returns all keys formatted for display.
func (b *Binding) AllKeysString() string {
	if len(b.Keys) == 0 {
		return ""
	}
	if len(b.Keys) == 1 {
		return b.Keys[0]
	}

	return strings.Join(b.Keys, "/")
}
