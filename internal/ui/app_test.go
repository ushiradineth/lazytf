package ui

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/tftui/internal/history"
	"github.com/ushiradineth/tftui/internal/terraform"
	"github.com/ushiradineth/tftui/internal/ui/components"
	"github.com/ushiradineth/tftui/internal/ui/views"
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

func TestApplyCompleteTransitions(t *testing.T) {
	plan := &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Action: terraform.ActionCreate},
		},
	}
	m := NewModel(plan)
	m.executionMode = true
	m.applyView = views.NewApplyView(m.styles)
	m.execView = viewApplyOutput

	m.handleApplyComplete(ApplyCompleteMsg{Success: true, Result: &terraform.ExecutionResult{}})
	if m.execView != viewMain {
		t.Fatalf("expected viewMain after success")
	}
	if m.plan == nil || len(m.plan.Resources) != 0 {
		t.Fatalf("expected plan to be cleared after apply")
	}
	if m.toastMessage == "" {
		t.Fatalf("expected toast message after apply")
	}
}

func TestApplyCompleteFailureStaysOnOutput(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.applyView = views.NewApplyView(m.styles)
	m.execView = viewApplyOutput

	m.handleApplyComplete(ApplyCompleteMsg{Success: false, Error: errors.New("fail"), Result: &terraform.ExecutionResult{}})
	if m.execView != viewApplyOutput {
		t.Fatalf("expected to remain on output view on failure")
	}
}

func TestHistoryFocusNavigation(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.historyPanel = components.NewHistoryPanel(m.styles)
	m.historyEntries = []history.Entry{
		{Summary: "first"},
		{Summary: "second"},
	}
	m.showHistory = true
	m.historyFocused = true
	m.syncHistorySelection()

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Fatalf("expected history key to be handled")
	}
	if m.historySelected != 1 {
		t.Fatalf("expected selection to move to second entry")
	}
}

func TestHistoryDetailOpenFlow(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.historyPanel = components.NewHistoryPanel(m.styles)
	m.historyView = views.NewHistoryView(m.styles)
	m.historyView.SetSize(80, 20)
	store, err := history.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	m.historyStore = store

	entry := history.Entry{
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Duration:   time.Second,
		Status:     history.StatusSuccess,
		Summary:    "ok",
		Output:     "output text",
	}
	if err := store.RecordApply(entry); err != nil {
		t.Fatalf("record history: %v", err)
	}
	entries, err := store.ListRecent(5)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	m.historyEntries = entries
	m.showHistory = true
	m.historyFocused = true
	m.syncHistorySelection()

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled || cmd == nil {
		t.Fatalf("expected history enter to be handled")
	}
	msg := cmd()
	m.Update(msg)
	if m.execView != viewHistoryDetail {
		t.Fatalf("expected history detail view")
	}
	if m.historyDetail == nil || m.historyDetail.Output != "output text" {
		t.Fatalf("expected history detail output")
	}
}

func TestPlanConfirmApplySuccessFlow(t *testing.T) {
	plan := &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Action: terraform.ActionCreate},
		},
	}
	m := NewModel(plan)
	m.executionMode = true
	m.applyView = views.NewApplyView(m.styles)
	m.planView = views.NewPlanView("", m.styles)
	m.executor = &terraform.Executor{}

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !handled || m.execView != viewPlanConfirm {
		t.Fatalf("expected plan confirm view")
	}
	handled, _ = m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !handled || m.execView != viewApplyOutput {
		t.Fatalf("expected apply output view")
	}

	m.handleApplyComplete(ApplyCompleteMsg{Success: true, Result: &terraform.ExecutionResult{}})
	if m.execView != viewMain {
		t.Fatalf("expected viewMain after apply")
	}
	if m.toastMessage == "" {
		t.Fatalf("expected toast after apply")
	}
}

func TestPlanFailureStaysOnOutputView(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.applyView = views.NewApplyView(m.styles)
	m.executor = &terraform.Executor{}

	m.beginPlan()
	m.handlePlanComplete(PlanCompleteMsg{Error: errors.New("plan failed")})
	if m.execView != viewPlanOutput {
		t.Fatalf("expected to stay on plan output view")
	}
	if !strings.Contains(m.applyView.View(), "Plan failed") {
		t.Fatalf("expected plan failure footer")
	}
}

func TestHelpOverlayBlocksHistoryToggle(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.showHelp = true
	m.showHistory = false
	m.historyEntries = []history.Entry{{Summary: "one"}}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.showHistory {
		t.Fatalf("expected history toggle blocked by help")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.historyFocused {
		t.Fatalf("expected history focus blocked by help")
	}
}

func TestHistoryFocusNoEntriesNoOp(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.showHistory = true
	m.historyEntries = nil

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyTab})
	if handled {
		t.Fatalf("expected tab to be ignored with no history entries")
	}
	if m.historyFocused {
		t.Fatalf("expected history not focused")
	}
}

func TestHistoryDetailFallbackContent(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.historyView = views.NewHistoryView(m.styles)
	m.historyView.SetSize(80, 10)

	m.Update(HistoryDetailMsg{Entry: history.Entry{Output: ""}})
	out := m.historyView.View()
	if !strings.Contains(out, "No stored output") {
		t.Fatalf("expected fallback text in history view")
	}
}

func TestRecordHistoryAfterApply(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	store, err := history.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	m.historyStore = store
	m.applyStartedAt = time.Now().Add(-time.Second)
	m.lastPlanOutput = "plan output"

	cmd := m.recordHistoryCmd(history.StatusSuccess, "summary", m.lastPlanOutput, &terraform.ExecutionResult{}, nil)
	if cmd == nil {
		t.Fatalf("expected history command")
	}
	cmd()

	entries, err := store.ListRecent(5)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected stored output")
	}
	loaded, err := store.GetByID(entries[0].ID)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if loaded.Output != "plan output" {
		t.Fatalf("expected stored output")
	}
}
