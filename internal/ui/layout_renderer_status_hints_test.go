package ui

import (
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func newStatusHintsModel(t *testing.T) *Model {
	t.Helper()
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.executionMode = true
	return m
}

func TestStatusHelpTextWorkspacePanel(t *testing.T) {
	m := newStatusHintsModel(t)
	m.panelManager.SetFocus(PanelWorkspace)

	got := m.statusHelpText()
	if !strings.Contains(got, "select: enter") {
		t.Fatalf("expected workspace select hint, got %q", got)
	}
	if !strings.Contains(got, "keybinds: ?") {
		t.Fatalf("expected help hint, got %q", got)
	}
}

func TestStatusHelpTextResourcesTabNoPlan(t *testing.T) {
	m := newStatusHintsModel(t)
	m.panelManager.SetFocus(PanelResources)
	m.resourcesActiveTab = 0
	m.resourceList.SetResources(nil)

	got := m.statusHelpText()
	for _, hint := range []string{"plan: p", "format: f", "validate: v", "init: i", "init upgrade: I", "keybinds: ?"} {
		if !strings.Contains(got, hint) {
			t.Fatalf("expected hint %q in %q", hint, got)
		}
	}
}

func TestStatusHelpTextResourcesStateTabIncludesInitHints(t *testing.T) {
	m := newStatusHintsModel(t)
	m.panelManager.SetFocus(PanelResources)
	m.resourcesActiveTab = 1

	got := m.statusHelpText()
	for _, hint := range []string{"select: enter", "yank: y", "init: i", "init upgrade: I", "keybinds: ?"} {
		if !strings.Contains(got, hint) {
			t.Fatalf("expected hint %q in %q", hint, got)
		}
	}
}

func TestStatusHelpTextResourcesTabWithPlanNotTargetMode(t *testing.T) {
	m := newStatusHintsModel(t)
	m.panelManager.SetFocus(PanelResources)
	m.resourcesActiveTab = 0
	m.resourceList.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})
	m.targetModeEnabled = false

	got := m.statusHelpText()
	for _, hint := range []string{"apply: a", "target: t", "yank: y", "keybinds: ?"} {
		if !strings.Contains(got, hint) {
			t.Fatalf("expected hint %q in %q", hint, got)
		}
	}
}

func TestStatusHelpTextResourcesTabWithPlanTargetMode(t *testing.T) {
	m := newStatusHintsModel(t)
	m.panelManager.SetFocus(PanelResources)
	m.resourcesActiveTab = 0
	m.resourceList.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})
	m.targetModeEnabled = true

	got := m.statusHelpText()
	for _, hint := range []string{"apply: a", "yank: y", "keybinds: ?"} {
		if !strings.Contains(got, hint) {
			t.Fatalf("expected hint %q in %q", hint, got)
		}
	}
	for _, moved := range []string{"target select: space", "target all: s", "target: t"} {
		if strings.Contains(got, moved) {
			t.Fatalf("did not expect moved target hint %q in global footer %q", moved, got)
		}
	}

	ordered := []string{"apply: a", "yank: y", "keybinds: ?"}
	last := -1
	for _, hint := range ordered {
		idx := strings.Index(got, hint)
		if idx == -1 {
			t.Fatalf("expected ordered hint %q in %q", hint, got)
		}
		if idx < last {
			t.Fatalf("expected order %v, got %q", ordered, got)
		}
		last = idx
	}
}

func TestStatusHelpTextMainAndCommandLogPanels(t *testing.T) {
	m := newStatusHintsModel(t)

	m.panelManager.SetFocus(PanelMain)
	mainHints := m.statusHelpText()
	if mainHints != "keybinds: ?" {
		t.Fatalf("expected only help hint on main panel, got %q", mainHints)
	}

	m.panelManager.SetFocus(PanelCommandLog)
	logHints := m.statusHelpText()
	for _, hint := range []string{"command log: L", "keybinds: ?"} {
		if !strings.Contains(logHints, hint) {
			t.Fatalf("expected hint %q in %q", hint, logHints)
		}
	}
}

func TestRenderStatusBarDoesNotIncludeTargetModeIndicator(t *testing.T) {
	m := newStatusHintsModel(t)
	m.width = 120
	m.panelManager.SetFocus(PanelResources)
	m.resourceList.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})
	m.targetModeEnabled = true
	m.resourceList.SetTargetModeEnabled(true)

	if ok := m.resourceList.ToggleTargetSelectionAtSelected(); !ok {
		t.Fatal("expected target selection to toggle")
	}

	got := m.renderStatusBar()
	if strings.Contains(got, "TARGET MODE") {
		t.Fatalf("did not expect target mode status text in footer, got %q", got)
	}
}

func TestRenderStatusBarShowsVersionTagOnRightSide(t *testing.T) {
	oldVersion := consts.Version
	consts.Version = "0.6.1"
	t.Cleanup(func() {
		consts.Version = oldVersion
	})

	m := newStatusHintsModel(t)
	m.width = 120
	m.panelManager.SetFocus(PanelResources)
	m.resourceList.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})

	got := m.renderStatusBar()
	if !strings.Contains(got, "v0.6.1") {
		t.Fatalf("expected version tag in footer, got %q", got)
	}
}

func TestRenderResourcesPanelWithTabsShowsTargetBadge(t *testing.T) {
	m := newStatusHintsModel(t)
	m.panelManager.SetFocus(PanelResources)
	m.resourcesActiveTab = 0
	m.resourceList.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})
	m.targetModeEnabled = true
	m.resourceList.SetTargetModeEnabled(true)

	got := m.renderResourcesPanelWithTabs(80, 10)
	if !strings.Contains(got, "[TARGET]") {
		t.Fatalf("expected target badge in resources panel title, got %q", got)
	}
}

func TestRenderResourcesPanelWithTabsHidesTargetBadgeOnStateTab(t *testing.T) {
	m := newStatusHintsModel(t)
	m.panelManager.SetFocus(PanelResources)
	m.resourcesActiveTab = 1
	m.resourceList.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})
	m.targetModeEnabled = true
	m.resourceList.SetTargetModeEnabled(true)

	got := m.renderResourcesPanelWithTabs(80, 10)
	if strings.Contains(got, "[TARGET]") {
		t.Fatalf("did not expect target badge on state tab, got %q", got)
	}
}

func TestRenderResourcesPanelWithTabsHidesTargetBadgeWhenTargetModeDisabled(t *testing.T) {
	m := newStatusHintsModel(t)
	m.panelManager.SetFocus(PanelResources)
	m.resourcesActiveTab = 0
	m.resourceList.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})
	m.targetModeEnabled = false
	m.resourceList.SetTargetModeEnabled(false)

	got := m.renderResourcesPanelWithTabs(80, 10)
	if strings.Contains(got, "[TARGET]") {
		t.Fatalf("did not expect target badge when target mode is disabled, got %q", got)
	}
}

func TestRenderStatusBarReadOnlyShowsPlanPathAndNoChip(t *testing.T) {
	plan := &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}}}
	m := NewReadOnlyModelWithStyles(plan, ExecutionConfig{PreloadedPlanPath: "/tmp/plan.tfplan"}, nil)
	m.width = 140

	got := m.renderStatusBar()
	if !strings.Contains(got, "/tmp/plan.tfplan") {
		t.Fatalf("expected read-only plan path in footer, got %q", got)
	}
	if strings.Contains(got, "read-only") {
		t.Fatalf("did not expect legacy read-only chip in footer, got %q", got)
	}
}

func TestRenderStatusBarReadOnlyShowsPlanPathInLeftStatusSection(t *testing.T) {
	plan := &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}}}
	m := NewReadOnlyModelWithStyles(plan, ExecutionConfig{PreloadedPlanPath: "/tmp/plan.tfplan"}, nil)
	m.width = 140

	got := m.renderStatusBar()
	pathIdx := strings.Index(got, "/tmp/plan.tfplan")
	summaryIdx := strings.Index(got, "1 changes")
	if pathIdx == -1 || summaryIdx == -1 {
		t.Fatalf("expected plan path and summary in footer, got %q", got)
	}
	if pathIdx > summaryIdx {
		t.Fatalf("expected plan path in left status section before summary, got %q", got)
	}
}

func TestRenderResourcesPanelReadOnlyShowsTitleBadge(t *testing.T) {
	plan := &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}}}
	m := NewReadOnlyModelWithStyles(plan, ExecutionConfig{}, nil)
	m.width = 80
	m.height = 20
	m.updateLayout()

	got := m.renderResourcesPanelWithTabs(80, 10)
	if !strings.Contains(got, "[READONLY]") {
		t.Fatalf("expected [READONLY] badge in resources title, got %q", got)
	}
}

func TestTruncateFooterPathKeepsSuffix(t *testing.T) {
	got := truncateFooterPath("/very/long/path/to/plan.tfplan", 12)
	if got == "" || !strings.Contains(got, "plan.tfplan") {
		t.Fatalf("expected truncated footer path to keep tail, got %q", got)
	}
}

func TestRenderStatusBarReadOnlyNarrowWidthKeepsUsefulPathSuffix(t *testing.T) {
	plan := &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}}}
	m := NewReadOnlyModelWithStyles(plan, ExecutionConfig{PreloadedPlanPath: "/Users/example/projects/infra/live/prod/very/long/path/plan.tfplan"}, nil)
	m.width = 40

	got := m.renderStatusBar()
	if !strings.Contains(got, "plan.tfplan") {
		t.Fatalf("expected narrow footer path to keep useful suffix, got %q", got)
	}
	if strings.Contains(got, "read-only") {
		t.Fatalf("did not expect legacy read-only chip in narrow footer, got %q", got)
	}
}

func TestRenderStatusBarReadOnlyVeryNarrowWidthStillKeepsPathTail(t *testing.T) {
	plan := &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}}}
	m := NewReadOnlyModelWithStyles(plan, ExecutionConfig{PreloadedPlanPath: "/Users/example/projects/infra/live/prod/very/long/path/plan.tfplan"}, nil)
	m.width = 24

	got := m.renderStatusBar()
	if !strings.Contains(got, "tfplan") {
		t.Fatalf("expected very narrow footer to keep plan path tail, got %q", got)
	}
}

func TestRenderStatusBarReadOnlyUltraNarrowWidthDoesNotRegress(t *testing.T) {
	plan := &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}}}
	m := NewReadOnlyModelWithStyles(plan, ExecutionConfig{PreloadedPlanPath: "/Users/example/projects/infra/live/prod/very/long/path/plan.tfplan"}, nil)
	m.width = 16

	got := m.renderStatusBar()
	if strings.TrimSpace(got) == "" {
		t.Fatal("expected non-empty status bar at ultra narrow width")
	}
	if strings.Contains(got, "read-only") {
		t.Fatalf("did not expect legacy read-only chip at ultra narrow width, got %q", got)
	}
}
