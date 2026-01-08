package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
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

func TestSearchFocusAndClear(t *testing.T) {
	m := NewModel(&terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Action: terraform.ActionCreate},
		},
	})
	m.ready = true

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.searching {
		t.Fatalf("expected searching to be true")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if m.searchInput.Value() == "" {
		t.Fatalf("expected search input to update")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.searching {
		t.Fatalf("expected searching to be false")
	}
	if m.searchInput.Value() != "" {
		t.Fatalf("expected search input cleared")
	}
}

func TestHelpBlocksInput(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.showHelp = true

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if !m.filterCreate {
		t.Fatalf("expected filters unchanged while help open")
	}
}

func TestModelInitReturnsCommand(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	if cmd := m.Init(); cmd == nil {
		t.Fatalf("expected init command to be set")
	}
}

func TestRenderMainContentSplitAndSingle(t *testing.T) {
	plan := &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Action: terraform.ActionCreate},
		},
	}
	m := NewModel(plan)
	m.ready = true
	m.width = 120
	m.height = 30
	m.updateLayout()

	split := m.renderMainContent()
	if split == m.resourceList.View() {
		t.Fatalf("expected split layout to differ from list-only view")
	}

	m.width = 80
	m.updateLayout()
	single := m.renderMainContent()
	if single != m.resourceList.View() {
		t.Fatalf("expected list-only view when width is narrow")
	}
}

func TestRenderHelpContent(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.width = 80
	m.height = 24

	out := m.renderHelp()
	if !strings.Contains(out, "tftui help") {
		t.Fatalf("expected help title in output")
	}
	if !strings.Contains(out, "Navigation") {
		t.Fatalf("expected navigation line in help output")
	}
}

func TestUpdateLayoutUsesMinimumHeight(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.width = 80
	m.height = 3
	m.updateLayout()

	if m.renderMainContent() == "" {
		t.Fatalf("expected main content to render with minimum height")
	}
}

func TestMinInt(t *testing.T) {
	if minInt(5, 2) != 2 {
		t.Fatalf("expected minInt to return smaller value")
	}
	if minInt(2, 5) != 2 {
		t.Fatalf("expected minInt to return smaller value")
	}
}

func TestDefaultKeyMapBindings(t *testing.T) {
	km := DefaultKeyMap()
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, km.Up) {
		t.Fatalf("expected up binding to match 'k'")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}, km.Help) {
		t.Fatalf("expected help binding to match '?'")
	}
}
