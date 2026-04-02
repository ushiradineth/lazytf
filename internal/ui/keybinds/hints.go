package keybinds

import (
	"sort"
	"strings"
)

// categoryExecution is the name of the Execution category.
const categoryExecution = "Execution"

var helpCategoryOrder = []string{
	"Panel Navigation",
	"Navigation",
	"Resources Panel",
	categoryExecution,
	"Search",
	"General",
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
//nolint:gocognit // Prioritized filtering/sorting rules are clearer in one pass.
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
		if b.Action == ActionFocusModeNext || b.Action == ActionFocusModePrev {
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
	items := make([]HelpItem, 0, 32) // Pre-allocate with reasonable capacity

	byCategory := r.GetBindingsByCategory()

	for _, category := range helpCategoryOrder {
		bindings, ok := byCategory[category]
		if !ok || len(bindings) == 0 {
			continue
		}

		// Filter execution bindings when not in execution mode
		if !ctx.ExecutionMode && category == categoryExecution {
			continue
		}

		sectionItems := helpSectionItems(bindings, ctx)

		if len(sectionItems) == 0 {
			continue
		}

		items = appendHelpSection(items, category, sectionItems)
	}

	// Remove trailing empty line
	if len(items) > 0 && items[len(items)-1].Key == "" {
		items = items[:len(items)-1]
	}

	return items
}

func helpSectionItems(bindings []Binding, ctx *Context) []HelpItem {
	bindings = deduplicateByKey(bindings)
	sort.Slice(bindings, func(i, j int) bool {
		return bindings[i].KeyString() < bindings[j].KeyString()
	})

	sectionItems := make([]HelpItem, 0, len(bindings))
	for _, b := range bindings {
		if b.Hidden {
			continue
		}
		if b.Condition != nil && !b.Condition(ctx) {
			continue
		}
		sectionItems = append(sectionItems, HelpItem{
			Key:         b.AllKeysString(),
			Description: b.Description,
			IsHeader:    false,
		})
	}

	return sectionItems
}

func appendHelpSection(items []HelpItem, category string, sectionItems []HelpItem) []HelpItem {
	items = append(items, HelpItem{Key: category, IsHeader: true})
	items = append(items, sectionItems...)
	items = append(items, HelpItem{Key: "", IsHeader: true})
	return items
}
