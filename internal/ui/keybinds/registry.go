package keybinds

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

// Registry manages keybind definitions and action handlers.
type Registry struct {
	bindings []Binding
	handlers map[Action]Handler
}

// NewRegistry creates a new keybind registry.
func NewRegistry() *Registry {
	return &Registry{
		bindings: make([]Binding, 0),
		handlers: make(map[Action]Handler),
	}
}

// Register adds a binding to the registry.
func (r *Registry) Register(b Binding) {
	r.bindings = append(r.bindings, b)
}

// RegisterHandler associates a handler with an action.
func (r *Registry) RegisterHandler(action Action, handler Handler) {
	r.handlers[action] = handler
}

// Resolve finds the highest-priority binding that matches the key and context.
func (r *Registry) Resolve(key string, ctx *Context) *Binding {
	var matches []*Binding

	for i := range r.bindings {
		if r.bindings[i].Matches(key, ctx) {
			matches = append(matches, &r.bindings[i])
		}
	}

	if len(matches) == 0 {
		return nil
	}

	// Sort by priority (descending)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].EffectivePriority() > matches[j].EffectivePriority()
	})

	return matches[0]
}

// Handle resolves a key and executes the corresponding handler.
// Returns (cmd, handled).
func (r *Registry) Handle(msg tea.KeyMsg, ctx *Context) (tea.Cmd, bool) {
	key := msg.String()

	// When a modal is open, only process modal-scoped bindings
	// All other keys are consumed but not executed (blocking other handlers)
	if ctx.ActiveModal != ModalNone { //nolint:nestif // Modal handling requires nested checks
		// Always allow q and ctrl+c to quit
		if key == "q" || key == "ctrl+c" {
			binding := r.Resolve(key, ctx)
			if binding != nil {
				if handler, ok := r.handlers[binding.Action]; ok {
					return handler(ctx), true
				}
			}
			return nil, false
		}

		// Find a modal-scoped binding for this key
		binding := r.Resolve(key, ctx)
		if binding != nil && binding.Scope == ScopeModal {
			if handler, ok := r.handlers[binding.Action]; ok {
				return handler(ctx), true
			}
		}
		// Consume the key even if no modal binding exists (blocks other handlers)
		return nil, true
	}

	binding := r.Resolve(key, ctx)
	if binding == nil {
		return nil, false
	}

	handler, ok := r.handlers[binding.Action]
	if !ok {
		return nil, false
	}

	cmd := handler(ctx)
	return cmd, true
}

// GetBindingsForContext returns all bindings active in the given context.
func (r *Registry) GetBindingsForContext(ctx *Context) []Binding {
	var result []Binding
	for _, b := range r.bindings {
		if r.bindingActiveInContext(&b, ctx) {
			result = append(result, b)
		}
	}
	return result
}

// bindingActiveInContext checks if a binding would be active (ignoring key).
func (r *Registry) bindingActiveInContext(b *Binding, ctx *Context) bool {
	switch b.Scope {
	case ScopeGlobal:
		// Global bindings always active unless condition fails
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

	if b.Condition != nil && !b.Condition(ctx) {
		return false
	}

	return true
}

// GetBindingsByCategory groups bindings by category.
func (r *Registry) GetBindingsByCategory() map[string][]Binding {
	result := make(map[string][]Binding)
	for _, b := range r.bindings {
		if b.Hidden {
			continue
		}
		category := b.Category
		if category == "" {
			category = "Other"
		}
		result[category] = append(result[category], b)
	}
	return result
}

// GetAllBindings returns all registered bindings.
func (r *Registry) GetAllBindings() []Binding {
	result := make([]Binding, len(r.bindings))
	copy(result, r.bindings)
	return result
}

// BindingCount returns the number of registered bindings.
func (r *Registry) BindingCount() int {
	return len(r.bindings)
}

// deduplicateByKey removes duplicate bindings for the same key, keeping highest priority.
func deduplicateByKey(bindings []Binding) []Binding {
	seen := make(map[string]*Binding)
	for i := range bindings {
		b := &bindings[i]
		key := b.KeyString()
		if existing, ok := seen[key]; ok {
			if b.EffectivePriority() > existing.EffectivePriority() {
				seen[key] = b
			}
		} else {
			seen[key] = b
		}
	}

	result := make([]Binding, 0, len(seen))
	for _, b := range seen {
		result = append(result, *b)
	}
	return result
}
