package keybinds

import (
	"sort"
	"strings"
)

// categoryExecution is the name of the Execution category.
const categoryExecution = "Execution"

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
func (r *Registry) ForStatusBar(ctx *Context, opts HintOptions) string {
	if opts.Separator == "" {
		opts.Separator = " | "
	}

	bindings := r.GetBindingsForContext(ctx)
	if len(bindings) == 0 {
		return ""
	}

	// Deduplicate by key
	bindings = deduplicateByKey(bindings)

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
	case "open settings", "toggle settings":
		desc = "settings"
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
func (r *Registry) ForHelpModal(executionMode bool) []HelpItem {
	items := make([]HelpItem, 0, 32) // Pre-allocate with reasonable capacity

	// Define category order
	categoryOrder := []string{
		"Panel Navigation",
		"Navigation",
		"Resources Panel",
		categoryExecution,
		"Search",
		"General",
	}

	byCategory := r.GetBindingsByCategory()

	for _, category := range categoryOrder {
		bindings, ok := byCategory[category]
		if !ok || len(bindings) == 0 {
			continue
		}

		// Filter execution bindings when not in execution mode
		if !executionMode && category == categoryExecution {
			continue
		}

		// Add header
		items = append(items, HelpItem{
			Key:      category,
			IsHeader: true,
		})

		// Deduplicate and sort bindings
		bindings = deduplicateByKey(bindings)
		sort.Slice(bindings, func(i, j int) bool {
			return bindings[i].KeyString() < bindings[j].KeyString()
		})

		// Add bindings
		for _, b := range bindings {
			if b.Hidden {
				continue
			}
			items = append(items, HelpItem{
				Key:         b.AllKeysString(),
				Description: b.Description,
				IsHeader:    false,
			})
		}

		// Add empty line after section
		items = append(items, HelpItem{
			Key:      "",
			IsHeader: true,
		})
	}

	// Remove trailing empty line
	if len(items) > 0 && items[len(items)-1].Key == "" {
		items = items[:len(items)-1]
	}

	return items
}
