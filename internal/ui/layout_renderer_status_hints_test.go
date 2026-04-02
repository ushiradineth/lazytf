package ui

import (
	"strings"
	"testing"

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
	if !strings.Contains(got, "enter: select") {
		t.Fatalf("expected workspace select hint, got %q", got)
	}
	if !strings.Contains(got, "?: kbd") {
		t.Fatalf("expected help hint, got %q", got)
	}
}

func TestStatusHelpTextResourcesTabNoPlan(t *testing.T) {
	m := newStatusHintsModel(t)
	m.panelManager.SetFocus(PanelResources)
	m.resourcesActiveTab = 0
	m.resourceList.SetResources(nil)

	got := m.statusHelpText()
	for _, hint := range []string{"p: plan", "f: format", "v: validate", "i: init", "?: kbd"} {
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
	for _, hint := range []string{"a: apply", "t: enter target mode", "x: reset plan", "?: kbd"} {
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
	for _, hint := range []string{"A: apply", "t: exit target mode", "a: toggle all", "?: kbd"} {
		if !strings.Contains(got, hint) {
			t.Fatalf("expected hint %q in %q", hint, got)
		}
	}
}

func TestStatusHelpTextMainAndCommandLogPanels(t *testing.T) {
	m := newStatusHintsModel(t)

	m.panelManager.SetFocus(PanelMain)
	mainHints := m.statusHelpText()
	if mainHints != "?: kbd" {
		t.Fatalf("expected only help hint on main panel, got %q", mainHints)
	}

	m.panelManager.SetFocus(PanelCommandLog)
	logHints := m.statusHelpText()
	for _, hint := range []string{"L: toggle", "?: kbd"} {
		if !strings.Contains(logHints, hint) {
			t.Fatalf("expected hint %q in %q", hint, logHints)
		}
	}
}
