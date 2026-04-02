package keybinds

import (
	"sort"
	"strings"
)

// categoryExecution is the name of the Execution category.
const categoryExecution = "Execution"

type helpSectionSpec struct {
	Category string
	Actions  []Action
}

var helpSectionOrder = []helpSectionSpec{
	{
		Category: "Panel Navigation",
		Actions: []Action{
			ActionFocusModeNext,
			ActionFocusModePrev,
			ActionCycleFocus,
			ActionCycleFocusBack,
			ActionToggleLog,
			ActionFocusWorkspace,
			ActionFocusResources,
			ActionFocusHistory,
			ActionFocusMain,
			ActionFocusCommandLog,
			ActionEscapeBack,
		},
	},
	{
		Category: "Navigation",
		Actions: []Action{
			ActionMoveUp,
			ActionMoveDown,
			ActionPageUp,
			ActionPageDown,
			ActionScrollTop,
			ActionScrollEnd,
			ActionTreeParent,
			ActionTreeChild,
			ActionSelect,
		},
	},
	{
		Category: "Resources Panel",
		Actions: []Action{
			ActionToggleCreate,
			ActionToggleReplace,
			ActionToggleUpdate,
			ActionToggleDelete,
			ActionCopyAddress,
			ActionToggleTargetMode,
			ActionToggleTarget,
			ActionToggleAllTargets,
			ActionToggleStatus,
			ActionSwitchTabPrev,
			ActionSwitchTabNext,
		},
	},
	{
		Category: categoryExecution,
		Actions: []Action{
			ActionPlan,
			ActionApply,
			ActionValidate,
			ActionFormat,
			ActionInit,
			ActionInitUpgrade,
			ActionRefresh,
			ActionToggleHistory,
		},
	},
	{
		Category: "General",
		Actions: []Action{
			ActionToggleHelp,
			ActionQuit,
			ActionCancelOp,
			ActionToggleTheme,
		},
	},
}

// HintOptions configures hint generation.
type HintOptions struct {
	// MaxPrimary is the maximum number of primary hints to show.
	MaxPrimary int
	// MaxSecondary is the maximum number of secondary hints to show.
	MaxSecondary int
	// Separator between hints.
	Separator string
}

// DefaultHintOptions returns sensible defaults for hint generation.
func DefaultHintOptions() HintOptions {
	return HintOptions{
		MaxPrimary:   4,
		MaxSecondary: 2,
		Separator:    " | ",
	}
}

// ForStatusBar generates a status bar hint string for the given context.
//
//nolint:gocognit,gocyclo // Prioritized filtering/sorting rules are clearer in one pass.
func (r *Registry) ForStatusBar(ctx *Context, opts HintOptions) string {
	if opts.Separator == "" {
		opts.Separator = " | "
	}

	bindings := r.GetBindingsForContext(ctx)
	if len(bindings) == 0 {
		return ""
	}

	// Filter out hidden bindings and those without descriptions
	var visible []Binding
	for _, b := range bindings {
		if b.Action == ActionFocusModeNext || b.Action == ActionFocusModePrev || b.Action == ActionToggleLog || b.Action == ActionToggleTheme {
			continue
		}
		if !b.Hidden && b.Description != "" {
			visible = append(visible, b)
		}
	}

	// Deduplicate by key after filtering so hidden high-priority duplicates
	// do not mask visible hints.
	visible = deduplicateByKey(visible)

	// Sort by priority (higher first), then by key
	sort.Slice(visible, func(i, j int) bool {
		pi, pj := visible[i].EffectivePriority(), visible[j].EffectivePriority()
		if pi != pj {
			return pi > pj
		}
		return visible[i].KeyString() < visible[j].KeyString()
	})

	// Build hints
	var hints []string

	// Primary hints (panel-specific or high priority)
	primaryCount := 0
	for _, b := range visible {
		if primaryCount >= opts.MaxPrimary {
			break
		}
		if b.Scope != ScopeGlobal {
			hints = append(hints, formatHint(b))
			primaryCount++
		}
	}

	// Secondary hints (global)
	secondaryCount := 0
	for _, b := range visible {
		if secondaryCount >= opts.MaxSecondary {
			break
		}
		if b.Scope == ScopeGlobal {
			hints = append(hints, formatHint(b))
			secondaryCount++
		}
	}

	return strings.Join(hints, opts.Separator)
}

// formatHint formats a single hint.
func formatHint(b Binding) string {
	key := b.KeyString()
	desc := b.Description

	// Shorten common descriptions
	switch desc {
	case "toggle keybinds", "toggle keybinds help":
		desc = "keybinds"
	}

	return key + ": " + desc
}

// HelpItem represents a single item in the help display.
type HelpItem struct {
	Key         string
	Description string
	IsHeader    bool
}

// ForHelpModal returns all bindings formatted for the help modal.
func (r *Registry) ForHelpModal(ctx *Context) []HelpItem {
	if ctx == nil {
		ctx = NewContext()
	}

	bindings := visibleHelpBindings(r.GetBindingsForContext(ctx))
	if len(bindings) == 0 {
		return nil
	}
	byCategory := groupHelpBindingsByCategory(bindings)

	items := make([]HelpItem, 0, 32) // Pre-allocate with reasonable capacity

	for _, section := range helpSectionOrder {
		if !ctx.ExecutionMode && section.Category == categoryExecution {
			continue
		}

		categoryBindings, ok := byCategory[section.Category]
		if !ok || len(categoryBindings) == 0 {
			continue
		}
		sectionItems := helpSectionItems(categoryBindings, section.Actions)

		if len(sectionItems) == 0 {
			continue
		}

		items = appendHelpSection(items, section.Category, sectionItems)
	}

	// Remove trailing empty line
	if len(items) > 0 && items[len(items)-1].Key == "" {
		items = items[:len(items)-1]
	}

	return items
}

func visibleHelpBindings(bindings []Binding) []Binding {
	visible := make([]Binding, 0, len(bindings))
	for _, b := range bindings {
		if b.Hidden {
			continue
		}
		visible = append(visible, b)
	}
	return visible
}

func groupHelpBindingsByCategory(bindings []Binding) map[string][]Binding {
	grouped := make(map[string][]Binding)
	for _, b := range bindings {
		category := b.Category
		if category == "" {
			category = "Other"
		}
		grouped[category] = append(grouped[category], b)
	}
	return grouped
}

func helpSectionItems(bindings []Binding, orderedActions []Action) []HelpItem {
	bindings = deduplicateByKey(bindings)

	sectionItems := make([]HelpItem, 0, len(bindings))
	used := make([]bool, len(bindings))

	for _, action := range orderedActions {
		for i, b := range bindings {
			if used[i] || b.Action != action {
				continue
			}
			sectionItems = append(sectionItems, HelpItem{
				Key:         formatHelpKeys(b.Keys),
				Description: b.Description,
				IsHeader:    false,
			})
			used[i] = true
		}
	}

	remaining := make([]Binding, 0, len(bindings))
	for i, b := range bindings {
		if used[i] {
			continue
		}
		remaining = append(remaining, b)
	}
	sort.Slice(remaining, func(i, j int) bool {
		return formatHelpKeys(remaining[i].Keys) < formatHelpKeys(remaining[j].Keys)
	})

	for _, b := range remaining {
		sectionItems = append(sectionItems, HelpItem{
			Key:         formatHelpKeys(b.Keys),
			Description: b.Description,
			IsHeader:    false,
		})
	}

	return sectionItems
}

func formatHelpKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == " " {
			parts = append(parts, "space")
			continue
		}
		parts = append(parts, key)
	}
	return strings.Join(parts, "/")
}

func appendHelpSection(items []HelpItem, category string, sectionItems []HelpItem) []HelpItem {
	items = append(items, HelpItem{Key: category, IsHeader: true})
	items = append(items, sectionItems...)
	items = append(items, HelpItem{Key: "", IsHeader: true})
	return items
}
