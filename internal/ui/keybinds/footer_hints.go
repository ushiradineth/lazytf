package keybinds

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

type footerHintSpec struct {
	Action       Action
	PreferredKey string
}

const footerSpaceKey = "space"

// FooterHints returns ordered footer hints for the active context.
func (r *Registry) FooterHints(ctx *Context) []string {
	if ctx == nil {
		ctx = NewContext()
	}

	active := r.GetBindingsForContext(ctx)
	spec := footerSpecsForContext(ctx)
	spec = append(spec, footerHintSpec{Action: ActionToggleHelp})

	hints := make([]string, 0, len(spec))
	for _, hintSpec := range spec {
		binding := selectBindingForAction(active, hintSpec)
		if binding == nil {
			continue
		}
		desc := footerDescription(*binding)
		if strings.TrimSpace(desc) == "" {
			continue
		}
		key := footerKey(*binding, hintSpec.PreferredKey)
		hints = append(hints, fmt.Sprintf("%s: %s", desc, key))
	}
	return hints
}

func footerSpecsForContext(ctx *Context) []footerHintSpec {
	switch ctx.FocusedPanel {
	case PanelWorkspace:
		return []footerHintSpec{{Action: ActionSelect}}
	case PanelResources:
		return resourcesFooterSpecs(ctx)
	case PanelHistory:
		return []footerHintSpec{{Action: ActionSelect}}
	case PanelCommandLog:
		return []footerHintSpec{{Action: ActionToggleLog}}
	case PanelMain, PanelNone:
		return nil
	default:
		return nil
	}
}

func resourcesFooterSpecs(ctx *Context) []footerHintSpec {
	if ctx.ResourcesActiveTab == 1 {
		return []footerHintSpec{
			{Action: ActionSelect},
			{Action: ActionCopyAddress},
			{Action: ActionInit},
			{Action: ActionInitUpgrade},
		}
	}
	if !ctx.ExecutionMode {
		return nil
	}
	if !ctx.TargetAvailable {
		return []footerHintSpec{
			{Action: ActionPlan},
			{Action: ActionFormat},
			{Action: ActionValidate},
			{Action: ActionInit},
			{Action: ActionInitUpgrade},
		}
	}
	if ctx.TargetMode {
		return []footerHintSpec{
			{Action: ActionApply},
			{Action: ActionCopyAddress},
		}
	}
	return []footerHintSpec{
		{Action: ActionApply},
		{Action: ActionToggleTargetMode},
		{Action: ActionCopyAddress},
	}
}

func selectBindingForAction(bindings []Binding, spec footerHintSpec) *Binding {
	candidates := make([]Binding, 0, len(bindings))
	for _, b := range bindings {
		if b.Hidden || b.Action != spec.Action {
			continue
		}
		candidates = append(candidates, b)
	}
	if len(candidates) == 0 {
		return nil
	}
	if spec.PreferredKey != "" {
		for i := range candidates {
			if hasDisplayKey(candidates[i], spec.PreferredKey) {
				return &candidates[i]
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].EffectivePriority() > candidates[j].EffectivePriority()
	})
	return &candidates[0]
}

func hasDisplayKey(b Binding, key string) bool {
	if key == footerSpaceKey {
		return slices.Contains(b.Keys, " ")
	}
	return slices.Contains(b.Keys, key)
}

func footerKey(b Binding, preferred string) string {
	if preferred == footerSpaceKey && slices.Contains(b.Keys, " ") {
		return footerSpaceKey
	}
	key := b.KeyString()
	if key == " " {
		return footerSpaceKey
	}
	return key
}

func footerDescription(b Binding) string {
	return strings.TrimSpace(b.Description)
}
