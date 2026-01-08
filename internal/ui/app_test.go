package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/tftui/internal/terraform"
)

func TestModelViewStates(t *testing.T) {
	plan := &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "aws_vpc.main", Action: terraform.ActionCreate},
		},
	}

	m := NewModel(plan)

	if got := m.View(); got != "Loading..." {
		t.Fatalf("expected loading view, got %q", got)
	}

	m.ready = true
	m.err = errors.New("boom")
	if got := m.View(); got != "Error: boom\n" {
		t.Fatalf("expected error view, got %q", got)
	}

	m.err = nil
	m.plan = nil
	if got := m.View(); got != "No plan loaded\n" {
		t.Fatalf("expected no plan view, got %q", got)
	}

	m.plan = plan
	m.quitting = true
	if got := m.View(); got != "Goodbye!\n" {
		t.Fatalf("expected goodbye view, got %q", got)
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	m.Update(msg)

	if !m.ready {
		t.Fatalf("expected model to be ready")
	}
	if m.width != 80 || m.height != 24 {
		t.Fatalf("unexpected size: %dx%d", m.width, m.height)
	}
}

func TestModelUpdateFilterToggles(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if m.filterCreate {
		t.Fatalf("expected create filter toggled off")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if m.filterUpdate {
		t.Fatalf("expected update filter toggled off")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.filterDelete {
		t.Fatalf("expected delete filter toggled off")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if m.filterReplace {
		t.Fatalf("expected replace filter toggled off")
	}
}

func TestModelRenderFilterBarCounts(t *testing.T) {
	plan := &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "a", Action: terraform.ActionCreate},
			{Address: "b", Action: terraform.ActionUpdate},
			{Address: "c", Action: terraform.ActionDelete},
			{Address: "d", Action: terraform.ActionReplace},
		},
	}
	m := NewModel(plan)
	m.ready = true
	m.width = 80

	bar := m.renderFilterBar()
	for _, want := range []string{"Create (1)", "Update (1)", "Delete (1)", "Replace (1)"} {
		if !strings.Contains(bar, want) {
			t.Fatalf("missing %q in filter bar: %q", want, bar)
		}
	}
}
