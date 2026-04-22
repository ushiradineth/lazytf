package keybinds

import (
	"strings"
	"testing"
)

func helpCtx(executionMode bool) *Context {
	return &Context{ExecutionMode: executionMode}
}

func helpCtxWithHistory(executionMode, historyEnabled bool) *Context {
	return &Context{ExecutionMode: executionMode, HistoryEnabled: historyEnabled}
}

func TestDefaultHintOptions(t *testing.T) {
	opts := DefaultHintOptions()

	if opts.MaxPrimary != 4 {
		t.Errorf("expected MaxPrimary 4, got %d", opts.MaxPrimary)
	}
	if opts.MaxSecondary != 2 {
		t.Errorf("expected MaxSecondary 2, got %d", opts.MaxSecondary)
	}
	if opts.Separator != " | " {
		t.Errorf("expected Separator ' | ', got %q", opts.Separator)
	}
}

func TestRegistry_ForStatusBar_Empty(t *testing.T) {
	r := NewRegistry()
	ctx := NewContext()
	opts := DefaultHintOptions()

	result := r.ForStatusBar(ctx, opts)

	if result != "" {
		t.Errorf("expected empty string for empty registry, got %q", result)
	}
}

func TestRegistry_ForStatusBar_HiddenFiltered(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Hidden:      true, // Should be filtered out
	})

	ctx := NewContext()
	opts := DefaultHintOptions()

	result := r.ForStatusBar(ctx, opts)

	if result != "" {
		t.Errorf("expected empty string when all bindings hidden, got %q", result)
	}
}

func TestRegistry_ForStatusBar_NoDescriptionFiltered(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:   []string{"q"},
		Action: ActionQuit,
		Scope:  ScopeGlobal,
		// No description - should be filtered out
	})

	ctx := NewContext()
	opts := DefaultHintOptions()

	result := r.ForStatusBar(ctx, opts)

	if result != "" {
		t.Errorf("expected empty string when all bindings lack description, got %q", result)
	}
}

func TestRegistry_ForStatusBar_DefaultSeparator(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Category:    "General",
	})
	r.Register(Binding{
		Keys:        []string{"?"},
		Action:      ActionToggleHelp,
		Scope:       ScopeGlobal,
		Description: "help",
		Category:    "General",
	})

	ctx := NewContext()
	opts := HintOptions{
		MaxPrimary:   4,
		MaxSecondary: 4,
		Separator:    "", // Empty - should default to " | "
	}

	result := r.ForStatusBar(ctx, opts)

	if !strings.Contains(result, " | ") {
		t.Errorf("expected default separator ' | ', got %q", result)
	}
}

func TestRegistry_ForStatusBar_PrimaryAndSecondary(t *testing.T) {
	r := NewRegistry()

	// Panel-scoped binding (primary)
	r.Register(Binding{
		Keys:        []string{"p"},
		Action:      ActionPlan,
		Scope:       ScopePanel,
		Panel:       PanelResources,
		Description: "plan",
		Category:    "Execution",
	})

	// Global binding (secondary)
	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Category:    "General",
	})

	ctx := &Context{FocusedPanel: PanelResources}
	opts := DefaultHintOptions()

	result := r.ForStatusBar(ctx, opts)

	if !strings.Contains(result, "p: plan") {
		t.Errorf("expected primary hint 'p: plan', got %q", result)
	}
	if !strings.Contains(result, "q: quit") {
		t.Errorf("expected secondary hint 'q: quit', got %q", result)
	}
}

func TestRegistry_ForStatusBar_MaxLimits(t *testing.T) {
	r := NewRegistry()

	// Add many panel bindings
	actions := []Action{
		ActionPlan, ActionApply, ActionRefresh, ActionValidate, ActionFormat,
		ActionMoveUp, ActionMoveDown, ActionPageUp, ActionPageDown, ActionScrollTop,
	}
	for i := 0; i < 10; i++ {
		r.Register(Binding{
			Keys:        []string{string(rune('a' + i))},
			Action:      actions[i],
			Scope:       ScopePanel,
			Panel:       PanelResources,
			Description: "action" + string(rune('a'+i)),
			Category:    "Test",
		})
	}

	ctx := &Context{FocusedPanel: PanelResources}
	opts := HintOptions{
		MaxPrimary:   2,
		MaxSecondary: 1,
		Separator:    ",",
	}

	result := r.ForStatusBar(ctx, opts)
	parts := strings.Split(result, ",")

	// Should only have MaxPrimary hints (no global bindings, so no secondary)
	if len(parts) > 2 {
		t.Errorf("expected at most 2 hints, got %d: %q", len(parts), result)
	}
}

func TestRegistry_ForStatusBar_ShortenedDescriptions(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"?"},
		Action:      ActionToggleHelp,
		Scope:       ScopeGlobal,
		Description: "toggle keybinds",
		Category:    "General",
	})
	ctx := NewContext()
	opts := DefaultHintOptions()

	result := r.ForStatusBar(ctx, opts)

	// Check shortened descriptions
	if !strings.Contains(result, "keybinds") {
		t.Errorf("expected shortened 'keybinds', got %q", result)
	}
}

func TestRegistry_ForStatusBar_ExcludesFocusModeHints(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"+"},
		Action:      ActionFocusModeNext,
		Scope:       ScopeGlobal,
		Description: "next focus mode",
		Category:    "Panel Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"_"},
		Action:      ActionFocusModePrev,
		Scope:       ScopeGlobal,
		Description: "previous focus mode",
		Category:    "Panel Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"L"},
		Action:      ActionToggleLog,
		Scope:       ScopeGlobal,
		Description: "toggle command log",
		Category:    "Panel Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"T"},
		Action:      ActionToggleTheme,
		Scope:       ScopeGlobal,
		Description: "change theme",
		Category:    "General",
	})
	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Category:    "General",
	})

	result := r.ForStatusBar(NewContext(), DefaultHintOptions())

	if strings.Contains(result, "focus mode") {
		t.Fatalf("expected focus mode hints to be excluded, got %q", result)
	}
	if strings.Contains(result, "command log") {
		t.Fatalf("expected command log hint to be excluded, got %q", result)
	}
	if strings.Contains(result, "theme") {
		t.Fatalf("expected theme hint to be excluded, got %q", result)
	}
	if !strings.Contains(result, "q: quit") {
		t.Fatalf("expected normal global hint to remain, got %q", result)
	}
}

func TestRegistry_ForStatusBar_IncludesToggleStatusHint(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, false)

	ctx := &Context{
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 0,
	}
	opts := HintOptions{MaxPrimary: 16, MaxSecondary: 16, Separator: " | "}

	result := r.ForStatusBar(ctx, opts)
	if !strings.Contains(result, "s: status") {
		t.Fatalf("expected status hint in status bar output, got %q", result)
	}
}

func TestRegistry_ForStatusBar_IncludesTargetModeHints(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	ctx := &Context{
		ExecutionMode:      true,
		FocusedPanel:       PanelResources,
		ResourcesActiveTab: 0,
		TargetMode:         true,
		TargetAvailable:    true,
	}
	opts := HintOptions{MaxPrimary: 16, MaxSecondary: 16, Separator: " | "}

	result := r.ForStatusBar(ctx, opts)
	if !strings.Contains(result, "target select") {
		t.Fatalf("expected target-selection hint in status bar output, got %q", result)
	}
	if !strings.Contains(result, "s: target all") {
		t.Fatalf("expected toggle-all-targets hint in status bar output, got %q", result)
	}
	if !strings.Contains(result, "a: apply") {
		t.Fatalf("expected target-mode apply hint in status bar output, got %q", result)
	}
}

func TestRegistry_ForHelpModal_Empty(t *testing.T) {
	r := NewRegistry()

	items := r.ForHelpModal(helpCtx(false))

	if len(items) != 0 {
		t.Errorf("expected 0 items for empty registry, got %d", len(items))
	}
}

func TestRegistry_ForHelpModal_Categories(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Category:    "General",
	})
	r.Register(Binding{
		Keys:        []string{"j"},
		Action:      ActionMoveDown,
		Scope:       ScopeGlobal,
		Description: "move down",
		Category:    "Navigation",
	})

	items := r.ForHelpModal(helpCtx(false))

	// Should have headers and items
	hasNavHeader := false
	hasGeneralHeader := false
	for _, item := range items {
		if item.IsHeader && item.Key == "Navigation" {
			hasNavHeader = true
		}
		if item.IsHeader && item.Key == "General" {
			hasGeneralHeader = true
		}
	}

	if !hasNavHeader {
		t.Error("expected Navigation header")
	}
	if !hasGeneralHeader {
		t.Error("expected General header")
	}
}

func TestRegistry_ForHelpModal_ExecutionModeFilter(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"p"},
		Action:      ActionPlan,
		Scope:       ScopeGlobal,
		Description: "plan",
		Category:    "Execution",
	})
	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Category:    "General",
	})

	// When not in execution mode, Execution category should be filtered
	items := r.ForHelpModal(helpCtx(false))

	hasExecutionHeader := false
	for _, item := range items {
		if item.IsHeader && item.Key == "Execution" {
			hasExecutionHeader = true
		}
	}

	if hasExecutionHeader {
		t.Error("expected Execution category to be filtered when not in execution mode")
	}

	// When in execution mode, Execution category should appear
	items = r.ForHelpModal(helpCtx(true))

	hasExecutionHeader = false
	for _, item := range items {
		if item.IsHeader && item.Key == "Execution" {
			hasExecutionHeader = true
		}
	}

	if !hasExecutionHeader {
		t.Error("expected Execution category when in execution mode")
	}
}

func TestRegistry_ForHelpModal_HiddenFiltered(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Category:    "General",
	})
	r.Register(Binding{
		Keys:        []string{"x"},
		Action:      ActionMoveUp,
		Scope:       ScopeGlobal,
		Description: "hidden action",
		Category:    "General",
		Hidden:      true,
	})

	items := r.ForHelpModal(helpCtx(false))

	for _, item := range items {
		if !item.IsHeader && item.Key == "x" {
			t.Error("expected hidden binding to be filtered")
		}
	}
}

func TestRegistry_ForHelpModal_TrailingEmptyLineRemoved(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Category:    "General",
	})

	items := r.ForHelpModal(helpCtx(false))

	if len(items) > 0 {
		last := items[len(items)-1]
		if last.IsHeader && last.Key == "" {
			t.Error("expected trailing empty line to be removed")
		}
	}
}

func TestDeduplicateByKey(t *testing.T) {
	bindings := []Binding{
		{
			Keys:     []string{"p"},
			Action:   ActionMoveUp,
			Scope:    ScopeGlobal,
			Priority: PriorityGlobal,
		},
		{
			Keys:     []string{"p"},
			Action:   ActionPlan,
			Scope:    ScopePanel,
			Priority: PriorityPanel, // Higher priority - should win
		},
	}

	result := deduplicateByKey(bindings)

	if len(result) != 1 {
		t.Errorf("expected 1 binding after dedup, got %d", len(result))
	}
	if result[0].Action != ActionPlan {
		t.Errorf("expected higher priority binding to win, got %v", result[0].Action)
	}
}

func TestDeduplicateByKey_KeepsFirst(t *testing.T) {
	// When priorities are equal, first one should be kept
	bindings := []Binding{
		{
			Keys:     []string{"p"},
			Action:   ActionMoveUp,
			Scope:    ScopeGlobal,
			Priority: PriorityGlobal,
		},
		{
			Keys:     []string{"p"},
			Action:   ActionPlan,
			Scope:    ScopeGlobal,
			Priority: PriorityGlobal, // Same priority
		},
	}

	result := deduplicateByKey(bindings)

	if len(result) != 1 {
		t.Errorf("expected 1 binding after dedup, got %d", len(result))
	}
	// First one with same priority wins
	if result[0].Action != ActionMoveUp {
		t.Errorf("expected first binding to win on equal priority, got %v", result[0].Action)
	}
}

func TestFormatHint_ToggleKeybindsHelp(t *testing.T) {
	b := Binding{
		Keys:        []string{"?"},
		Description: "toggle keybinds help",
	}

	result := formatHint(b)

	if result != "?: keybinds" {
		t.Errorf("expected '?: keybinds', got %q", result)
	}
}

func TestFormatHint_RegularDescription(t *testing.T) {
	b := Binding{
		Keys:        []string{"p"},
		Description: "run plan",
	}

	result := formatHint(b)

	if result != "p: run plan" {
		t.Errorf("expected 'p: run plan', got %q", result)
	}
}

func TestRegistry_ForStatusBar_Sorting(t *testing.T) {
	r := NewRegistry()

	// Add bindings with different priorities - panel scope should come first
	r.Register(Binding{
		Keys:        []string{"z"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "quit",
		Category:    "General",
	})
	r.Register(Binding{
		Keys:        []string{"a"},
		Action:      ActionPlan,
		Scope:       ScopePanel,
		Panel:       PanelResources,
		Description: "plan",
		Category:    "Execution",
	})

	ctx := &Context{FocusedPanel: PanelResources}
	opts := DefaultHintOptions()

	result := r.ForStatusBar(ctx, opts)

	// Panel binding should appear before global
	planIdx := strings.Index(result, "a: plan")
	quitIdx := strings.Index(result, "z: quit")

	if planIdx == -1 || quitIdx == -1 {
		t.Fatalf("expected both hints, got %q", result)
	}
	if planIdx > quitIdx {
		t.Errorf("expected panel hint before global, got %q", result)
	}
}

func TestRegistry_ForHelpModal_UnknownCategory(t *testing.T) {
	r := NewRegistry()

	// Register with unknown category - should not appear (not in categoryOrder)
	r.Register(Binding{
		Keys:        []string{"x"},
		Action:      ActionMoveUp,
		Scope:       ScopeGlobal,
		Description: "test",
		Category:    "UnknownCategory",
	})

	items := r.ForHelpModal(helpCtx(false))

	hasUnknownCategory := false
	for _, item := range items {
		if item.IsHeader && item.Key == "UnknownCategory" {
			hasUnknownCategory = true
		}
	}

	if hasUnknownCategory {
		t.Error("unexpected category should not appear in help modal")
	}
}

func TestRegistry_ForHelpModal_EmptyDescription(t *testing.T) {
	r := NewRegistry()

	// Binding with no description - still included
	r.Register(Binding{
		Keys:        []string{"q"},
		Action:      ActionQuit,
		Scope:       ScopeGlobal,
		Description: "", // Empty description
		Category:    "General",
	})

	items := r.ForHelpModal(helpCtx(false))

	// Should have General header and the binding
	hasGeneral := false
	hasBinding := false
	for _, item := range items {
		if item.IsHeader && item.Key == "General" {
			hasGeneral = true
		}
		if !item.IsHeader && item.Key == "q" {
			hasBinding = true
		}
	}

	if !hasGeneral {
		t.Error("expected General header")
	}
	if !hasBinding {
		t.Error("expected binding with empty description")
	}
}

func TestRegistry_ForHelpModal_AllKeysString(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"up", "k"},
		Action:      ActionMoveUp,
		Scope:       ScopeGlobal,
		Description: "move up",
		Category:    "Navigation",
	})

	items := r.ForHelpModal(helpCtx(false))

	found := false
	for _, item := range items {
		if !item.IsHeader && item.Key == "up/k" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected binding with combined keys 'up/k'")
	}
}

func TestDeduplicateByKey_Empty(t *testing.T) {
	result := deduplicateByKey(nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestDeduplicateByKey_SingleItem(t *testing.T) {
	bindings := []Binding{
		{Keys: []string{"p"}, Action: ActionPlan},
	}
	result := deduplicateByKey(bindings)
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
}

func TestDeduplicateByKey_DifferentKeys(t *testing.T) {
	bindings := []Binding{
		{Keys: []string{"p"}, Action: ActionPlan},
		{Keys: []string{"a"}, Action: ActionApply},
	}
	result := deduplicateByKey(bindings)
	if len(result) != 2 {
		t.Errorf("expected 2 items (different keys), got %d", len(result))
	}
}

func TestRegistry_ForStatusBar_MaxSecondaryLimit(t *testing.T) {
	r := NewRegistry()

	// Add many global bindings to test secondary limit
	for i := 0; i < 5; i++ {
		r.Register(Binding{
			Keys:        []string{string(rune('a' + i))},
			Action:      ActionQuit,
			Scope:       ScopeGlobal,
			Description: "action" + string(rune('a'+i)),
			Category:    "General",
		})
	}

	ctx := NewContext()
	opts := HintOptions{
		MaxPrimary:   4,
		MaxSecondary: 2,
		Separator:    ",",
	}

	result := r.ForStatusBar(ctx, opts)
	parts := strings.Split(result, ",")

	// Should only have MaxSecondary hints (no panel bindings)
	if len(parts) > 2 {
		t.Errorf("expected at most 2 secondary hints, got %d: %q", len(parts), result)
	}
}

func TestRegistry_ForHelpModal_FallsBackToKeyOrder(t *testing.T) {
	r := NewRegistry()

	// Add bindings out of order
	r.Register(Binding{
		Keys:        []string{"z"},
		Action:      ActionMoveDown,
		Scope:       ScopeGlobal,
		Description: "z action",
		Category:    "General",
	})
	r.Register(Binding{
		Keys:        []string{"a"},
		Action:      ActionMoveDown,
		Scope:       ScopeGlobal,
		Description: "a action",
		Category:    "General",
	})

	items := r.ForHelpModal(helpCtx(false))

	// Find the two bindings and verify order
	var aIdx, zIdx int
	for i, item := range items {
		if !item.IsHeader && item.Key == "a" {
			aIdx = i
		}
		if !item.IsHeader && item.Key == "z" {
			zIdx = i
		}
	}

	if aIdx >= zIdx {
		t.Error("expected 'a' to come before 'z' in sorted list")
	}
}

func TestRegistry_ForHelpModal_PanelNavigationUsesHumanOrder(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	ctx := &Context{ExecutionMode: true}
	items := r.ForHelpModal(ctx)

	plusIdx, minusIdx, tabIdx, logIdx := -1, -1, -1, -1
	for i, item := range items {
		if item.IsHeader {
			continue
		}
		switch item.Key {
		case "+":
			plusIdx = i
		case "_":
			minusIdx = i
		case "tab":
			tabIdx = i
		case "L":
			logIdx = i
		}
	}

	if plusIdx == -1 || minusIdx == -1 || tabIdx == -1 || logIdx == -1 {
		t.Fatalf("expected +, _, tab, and L in help modal, got %+v", items)
	}
	if plusIdx >= minusIdx || minusIdx >= tabIdx || tabIdx >= logIdx {
		t.Fatalf("expected order +, _, tab, L; got indexes +:%d _:%d tab:%d L:%d", plusIdx, minusIdx, tabIdx, logIdx)
	}
}

func TestRegistry_ForHelpModal_HiddenInCategory(t *testing.T) {
	r := NewRegistry()

	// Add both visible and hidden bindings to same category
	r.Register(Binding{
		Keys:        []string{"a"},
		Action:      ActionMoveUp,
		Scope:       ScopeGlobal,
		Description: "visible",
		Category:    "Navigation",
	})
	r.Register(Binding{
		Keys:        []string{"b"},
		Action:      ActionMoveDown,
		Scope:       ScopeGlobal,
		Description: "hidden",
		Category:    "Navigation",
		Hidden:      true,
	})

	items := r.ForHelpModal(helpCtx(false))

	// Should have Navigation header and only visible binding
	hasA := false
	hasB := false
	for _, item := range items {
		if !item.IsHeader && item.Key == "a" {
			hasA = true
		}
		if !item.IsHeader && item.Key == "b" {
			hasB = true
		}
	}

	if !hasA {
		t.Error("expected visible binding 'a'")
	}
	if hasB {
		t.Error("expected hidden binding 'b' to be filtered")
	}
}

func TestRegistry_ForHelpModal_RespectsExecutionConditionOutsideExecutionCategory(t *testing.T) {
	r := NewRegistry()

	r.Register(Binding{
		Keys:        []string{"L"},
		Action:      ActionToggleLog,
		Scope:       ScopeGlobal,
		Description: "toggle command log",
		Category:    "Panel Navigation",
		Condition:   ConditionExecutionMode,
	})

	nonExecItems := r.ForHelpModal(helpCtx(false))
	for _, item := range nonExecItems {
		if !item.IsHeader && item.Key == "L" {
			t.Fatalf("expected L binding to be hidden when not in execution mode")
		}
	}

	execItems := r.ForHelpModal(helpCtx(true))
	found := false
	for _, item := range execItems {
		if !item.IsHeader && item.Key == "L" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected L binding to appear in execution mode")
	}
}

func TestRegistry_ForHelpModal_RespectsHistoryEnabledCondition(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, true)

	withoutHistory := r.ForHelpModal(helpCtxWithHistory(true, false))
	for _, item := range withoutHistory {
		if !item.IsHeader && item.Description == "history panel" {
			t.Fatalf("expected history toggle to be hidden when history is disabled")
		}
	}

	withHistory := r.ForHelpModal(helpCtxWithHistory(true, true))
	found := false
	for _, item := range withHistory {
		if !item.IsHeader && item.Description == "history panel" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected history toggle to appear when history is enabled")
	}
}
