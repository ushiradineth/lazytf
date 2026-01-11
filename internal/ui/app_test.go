package ui

import (
	"context"
	"errors"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/tftui/internal/environment"
	"github.com/ushiradineth/tftui/internal/history"
	"github.com/ushiradineth/tftui/internal/terraform"
	tfparser "github.com/ushiradineth/tftui/internal/terraform/parser"
	"github.com/ushiradineth/tftui/internal/ui/components"
	"github.com/ushiradineth/tftui/internal/ui/views"
	"github.com/ushiradineth/tftui/internal/utils"
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

func TestApplyWithoutPlanShowsToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !handled {
		t.Fatalf("expected key to be handled")
	}
	if m.err != nil {
		t.Fatalf("did not expect fatal error, got %v", m.err)
	}
	if m.toastMessage == "" || !m.toastIsError {
		t.Fatalf("expected error toast to be set")
	}
	if cmd == nil {
		t.Fatalf("expected toast clear command")
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
	m.modalState = ModalHelp

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
	if !strings.Contains(out, "tftui keybinds") {
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
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}, km.Keybinds) {
		t.Fatalf("expected keybinds binding to match '?'")
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
	if m.execView != viewApplyOutput {
		t.Fatalf("expected to stay on output view after success")
	}
	if m.plan == nil || len(m.plan.Resources) != 0 {
		t.Fatalf("expected plan to be cleared after apply")
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

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Fatalf("expected history key to be handled")
	}
	if cmd != nil {
		cmd()
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
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})
	m.historyStore = store
	m.envCurrent = "dev"

	entry := history.Entry{
		StartedAt:   time.Now(),
		FinishedAt:  time.Now(),
		Duration:    time.Second,
		Status:      history.StatusSuccess,
		Summary:     "ok",
		Environment: "dev",
		Output:      "output text",
	}
	if err := store.RecordApply(entry); err != nil {
		t.Fatalf("record history: %v", err)
	}
	entries, err := store.ListRecentForEnvironment("dev", 5)
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

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !handled || m.execView != viewPlanConfirm {
		t.Fatalf("expected plan confirm view")
	}
	if cmd != nil {
		cmd()
	}
	handled, cmd = m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !handled || m.execView != viewApplyOutput {
		t.Fatalf("expected apply output view")
	}
	if cmd != nil {
		cmd()
	}

	m.handleApplyComplete(ApplyCompleteMsg{Success: true, Result: &terraform.ExecutionResult{}})
	if m.execView != viewApplyOutput {
		t.Fatalf("expected to stay on output view after apply")
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
	m.modalState = ModalHelp
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

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyTab})
	if handled {
		t.Fatalf("expected tab to be ignored with no history entries")
	}
	if cmd != nil {
		cmd()
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
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})
	m.historyStore = store
	m.applyStartedAt = time.Now().Add(-time.Second)
	m.lastPlanOutput = "plan output"
	m.envCurrent = "dev"

	cmd := m.recordHistoryCmd(history.StatusSuccess, "summary", m.lastPlanOutput, &terraform.ExecutionResult{}, nil)
	if cmd == nil {
		t.Fatalf("expected history command")
	}
	cmd()

	entries, err := store.ListRecentForEnvironment("dev", 5)
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

func TestEnvStatusLabel(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envCurrent = ""
	m.envStrategy = environment.StrategyUnknown
	if got := m.envStatusLabel(); got != "unknown" {
		t.Fatalf("expected unknown label, got %q", got)
	}

	m.envCurrent = "dev"
	m.envStrategy = environment.StrategyWorkspace
	if got := m.envStatusLabel(); got != "dev (workspace)" {
		t.Fatalf("expected strategy label, got %q", got)
	}
}

func TestFormatLogTimestamp(t *testing.T) {
	ts := "2024-01-02T03:04:05Z"
	if got := formatLogTimestamp(ts); got != "2024-01-02 03:04:05 +00:00" {
		t.Fatalf("expected formatted timestamp, got %q", got)
	}

	raw := "not-a-timestamp"
	if got := formatLogTimestamp(raw); got != raw {
		t.Fatalf("expected raw fallback, got %q", got)
	}
}

func TestFormatLogOutput(t *testing.T) {
	input := strings.Join([]string{
		`{"@timestamp":"2024-01-02T03:04:05Z","@message":"hello"}`,
		`{"timestamp":"2024-01-02T03:04:05Z","message":"world"}`,
		`{"message":"just message"}`,
		" plain line ",
		"",
	}, "\n")
	expected := strings.Join([]string{
		"[2024-01-02 03:04:05 +00:00] hello",
		"[2024-01-02 03:04:05 +00:00] world",
		"just message",
		"plain line",
	}, "\n")

	if got := utils.FormatLogOutput(input); got != expected {
		t.Fatalf("unexpected log output:\n%s", got)
	}
}

func TestOperationStatus(t *testing.T) {
	if got := operationStatus(nil); got != history.StatusSuccess {
		t.Fatalf("expected success, got %s", got)
	}
	if got := operationStatus(context.Canceled); got != history.StatusCanceled {
		t.Fatalf("expected canceled, got %s", got)
	}
	if got := operationStatus(errors.New("boom")); got != history.StatusFailed {
		t.Fatalf("expected failed, got %s", got)
	}
}

func TestSelectOperationOutput(t *testing.T) {
	result := &terraform.ExecutionResult{
		Output: "output",
		Stdout: "stdout",
		Stderr: "stderr",
	}
	if got := selectOperationOutput("direct", result); got != "direct" {
		t.Fatalf("expected direct output, got %q", got)
	}
	if got := selectOperationOutput("", result); got != "output" {
		t.Fatalf("expected output field, got %q", got)
	}
	result.Output = ""
	if got := selectOperationOutput("", result); got != "stdout" {
		t.Fatalf("expected stdout field, got %q", got)
	}
	result.Stdout = ""
	if got := selectOperationOutput("", result); got != "stderr" {
		t.Fatalf("expected stderr field, got %q", got)
	}
	if got := selectOperationOutput("", nil); got != "" {
		t.Fatalf("expected empty output, got %q", got)
	}
}

func TestBuildCommandAddsFlags(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	cmd := m.buildCommand("plan", []string{"-var-file=dev.tfvars"}, true, true)
	if !strings.Contains(cmd, "-json") {
		t.Fatalf("expected json flag in command: %s", cmd)
	}
	if !strings.Contains(cmd, "-auto-approve") {
		t.Fatalf("expected auto-approve flag in command: %s", cmd)
	}

	cmd = m.buildCommand("apply", []string{"-json", "-auto-approve"}, true, true)
	if strings.Count(cmd, "-json") != 1 {
		t.Fatalf("expected single json flag in command: %s", cmd)
	}
	if strings.Count(cmd, "-auto-approve") != 1 {
		t.Fatalf("expected single auto-approve flag in command: %s", cmd)
	}
}

func TestContainsFlag(t *testing.T) {
	if !containsFlag([]string{"plan", "-json"}, "-json") {
		t.Fatalf("expected to find flag")
	}
	if containsFlag([]string{"plan", "-out=plan"}, "-json") {
		t.Fatalf("did not expect to find flag")
	}
}

func TestPlanOutputPath(t *testing.T) {
	if got := planOutputPath([]string{"-out", "plan.tfplan"}); got != "plan.tfplan" {
		t.Fatalf("expected plan output path, got %q", got)
	}
	if got := planOutputPath([]string{"-out=plan.tfplan"}); got != "plan.tfplan" {
		t.Fatalf("expected plan output path, got %q", got)
	}
	if got := planOutputPath([]string{"-out="}); got != "" {
		t.Fatalf("expected empty output path, got %q", got)
	}
}

func TestSetEnvironmentOptions(t *testing.T) {
	baseDir := t.TempDir()
	folderPath := filepath.Join(baseDir, "envs", "prod")
	result := environment.DetectionResult{
		BaseDir:     baseDir,
		Workspaces:  []string{"dev"},
		FolderPaths: []string{folderPath},
	}
	result.Environments = environment.BuildEnvironments(result, "")
	m := NewModel(&terraform.Plan{})
	m.envWorkDir = baseDir
	m.setEnvironmentOptions(result, environment.StrategyMixed, "")

	if len(m.envOptions) != 2 {
		t.Fatalf("expected 2 env options, got %d", len(m.envOptions))
	}
	if m.envOptions[0].Name != "dev" || m.envOptions[0].Strategy != environment.StrategyWorkspace {
		t.Fatalf("unexpected workspace option: %+v", m.envOptions[0])
	}
	if m.envOptions[1].Path != folderPath || m.envOptions[1].Strategy != environment.StrategyFolder {
		t.Fatalf("unexpected folder option: %+v", m.envOptions[1])
	}
}

func TestUpdateExecutionViewForStreaming(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewMain
	m.planRunning = true
	m.showCompactProgress = true
	m.updateExecutionViewForStreaming()
	if m.execView != viewCompactProgress {
		t.Fatalf("expected compact progress view, got %v", m.execView)
	}

	m.planRunning = false
	m.applyRunning = true
	m.showCompactProgress = false
	m.showDiagnostics = true
	m.updateExecutionViewForStreaming()
	if m.execView != viewDiagnostics {
		t.Fatalf("expected diagnostics view, got %v", m.execView)
	}

	m.showDiagnostics = false
	m.updateExecutionViewForStreaming()
	if m.execView != viewApplyOutput {
		t.Fatalf("expected apply output view, got %v", m.execView)
	}

	m.applyRunning = false
	m.execView = viewApplyOutput
	m.updateExecutionViewForStreaming()
	if m.execView != viewMain {
		t.Fatalf("expected return to main view, got %v", m.execView)
	}
}

func TestHandleStreamMessageUpdatesOperationState(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.operationState = terraform.NewOperationState()

	startMsg := terraform.StreamMessage{
		Type: terraform.MessageTypeApplyStart,
		Hook: &terraform.HookMessage{
			Resource: terraform.ResourceInstance{Address: "aws_instance.web"},
			Action:   "create",
		},
	}
	if cmd := m.handleStreamMessage(startMsg); cmd == nil {
		t.Fatalf("expected stream command")
	}
	op := m.operationState.GetResourceStatus("aws_instance.web")
	if op == nil || op.Status != terraform.StatusInProgress || op.Action != terraform.ActionCreate {
		t.Fatalf("expected in-progress create status")
	}

	completeMsg := terraform.StreamMessage{
		Type: terraform.MessageTypeApplyComplete,
		Hook: &terraform.HookMessage{
			Address: "aws_instance.web",
			IDValue: "i-123",
		},
	}
	m.handleStreamMessage(completeMsg)
	op = m.operationState.GetResourceStatus("aws_instance.web")
	if op == nil || op.Status != terraform.StatusComplete || op.IDValue != "i-123" {
		t.Fatalf("expected completed resource with id")
	}

	errorMsg := terraform.StreamMessage{
		Type: terraform.MessageTypeApplyErrored,
		Hook: &terraform.HookMessage{
			Address: "aws_instance.web",
			Error:   "boom",
		},
	}
	m.handleStreamMessage(errorMsg)
	op = m.operationState.GetResourceStatus("aws_instance.web")
	if op == nil || op.Status != terraform.StatusErrored || op.Error == "" {
		t.Fatalf("expected errored resource with error")
	}

	diagMsg := terraform.StreamMessage{
		Type: terraform.MessageTypeDiagnostic,
		Diagnostic: &terraform.Diagnostic{
			Severity: "error",
			Summary:  "bad",
		},
	}
	m.handleStreamMessage(diagMsg)
	diags := m.operationState.GetDiagnostics()
	if len(diags) != 1 || diags[0].Summary != "bad" {
		t.Fatalf("expected diagnostic added")
	}
}

func TestAddErrorDiagnostic(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.operationState = terraform.NewOperationState()

	m.addErrorDiagnostic("ignored", nil, "output")
	if len(m.operationState.GetDiagnostics()) != 0 {
		t.Fatalf("expected no diagnostics for nil error")
	}

	m.addErrorDiagnostic("Plan failed", errors.New("boom"), "details")
	diags := m.operationState.GetDiagnostics()
	if len(diags) != 1 {
		t.Fatalf("expected diagnostic to be recorded")
	}
	if diags[0].Summary != "Plan failed" || !strings.Contains(diags[0].Detail, "details") {
		t.Fatalf("unexpected diagnostic detail: %q", diags[0].Detail)
	}
}

func TestModalEnvSelectorErrorToast(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.modalState = ModalEnvSelector
	m.envOptions = []environment.Environment{
		{Name: "invalid", Strategy: environment.StrategyUnknown},
	}
	m.envView.SetMode(views.EnvViewEnvironments)
	m.envView.SetEnvironments(m.envOptions, environment.StrategyUnknown, "", ".")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected toast clear command")
	}
	if !strings.Contains(m.toastMessage, "Failed to switch environment") {
		t.Fatalf("expected environment error toast, got %q", m.toastMessage)
	}
	if !m.toastIsError {
		t.Fatalf("expected error toast")
	}
}

func TestPlanSummary(t *testing.T) {
	m := NewModel(nil)
	if got := m.planSummary(); got != "No changes" {
		t.Fatalf("expected no changes summary, got %q", got)
	}

	m.plan = &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "a", Action: terraform.ActionCreate},
			{Address: "b", Action: terraform.ActionUpdate},
			{Address: "c", Action: terraform.ActionDelete},
			{Address: "d", Action: terraform.ActionReplace},
		},
	}
	summary := m.planSummary()
	if !strings.Contains(summary, "+ 1 to create") || !strings.Contains(summary, "± 1 to replace") {
		t.Fatalf("unexpected summary output: %q", summary)
	}
}

func TestPlanFlagsForRecord(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.planFlags = []string{"-var", "foo=bar"}
	if got := m.planFlagsForRecord(); len(got) != 2 {
		t.Fatalf("expected default flags")
	}
	m.planRunFlags = []string{"-out=plan.tfplan"}
	if got := m.planFlagsForRecord(); len(got) != 1 || got[0] != "-out=plan.tfplan" {
		t.Fatalf("expected plan run flags")
	}
}

func TestMaxInt(t *testing.T) {
	if maxInt(1, 3) != 3 {
		t.Fatalf("expected maxInt to return larger value")
	}
	if maxInt(5, 2) != 5 {
		t.Fatalf("expected maxInt to return larger value")
	}
}

func TestTruncateOutput(t *testing.T) {
	if got := truncateOutput("short", 10); got != "short" {
		t.Fatalf("expected output unchanged, got %q", got)
	}
	if got := truncateOutput("truncate", 4); got != "trun" {
		t.Fatalf("expected truncated output, got %q", got)
	}
	if got := truncateOutput("keep", 0); got != "keep" {
		t.Fatalf("expected output unchanged for zero max, got %q", got)
	}
}

func TestChanToReader(t *testing.T) {
	ch := make(chan string, 2)
	ch <- "one"
	ch <- "two"
	close(ch)

	reader := chanToReader(ch)
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read data: %v", err)
	}
	if got := string(data); got != "one\ntwo\n" {
		t.Fatalf("unexpected reader output: %q", got)
	}
}

func TestWaitPlanCompleteCmdNilResult(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	msg := m.waitPlanCompleteCmd(nil)()
	planMsg, ok := msg.(PlanCompleteMsg)
	if !ok || planMsg.Error == nil {
		t.Fatalf("expected plan completion error")
	}
}

func TestWaitApplyCompleteCmdNilResult(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	msg := m.waitApplyCompleteCmd(nil)()
	applyMsg, ok := msg.(ApplyCompleteMsg)
	if !ok || applyMsg.Error == nil || applyMsg.Success {
		t.Fatalf("expected apply completion error")
	}
}

func TestRenderSettings(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.width = 80
	m.height = 24

	out := m.renderSettings()
	if !strings.Contains(out, "No configuration loaded.") {
		t.Fatalf("expected settings fallback text")
	}
}

func TestRenderEnvSelector(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.width = 80
	m.height = 24
	m.envOptions = []environment.Environment{
		{Name: "dev", Strategy: environment.StrategyWorkspace},
	}
	m.envCurrent = "dev"
	m.envStrategy = environment.StrategyWorkspace
	m.envDetection = &environment.DetectionResult{Warnings: []string{"warning"}}
	m.envView.SetMode(views.EnvViewEnvironments)
	m.refreshEnvSelector()

	out := m.renderEnvSelector()
	if !strings.Contains(out, "Select Environment") || !strings.Contains(out, "dev") {
		t.Fatalf("expected environment selector content")
	}
	if !strings.Contains(out, "Warnings:") {
		t.Fatalf("expected warning section")
	}
}

func TestRenderToast(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.width = 80
	m.height = 24

	out := m.renderToast("hello", true)
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected toast message")
	}
}

func TestDetectEnvironmentsCmdEmptyDir(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envWorkDir = t.TempDir()

	cmd := m.detectEnvironmentsCmd()
	if cmd == nil {
		t.Fatalf("expected detect environments command")
	}
	msg := cmd()
	typed, ok := msg.(EnvironmentDetectedMsg)
	if !ok {
		t.Fatalf("expected environment detected message")
	}
	if typed.Result.Strategy != environment.StrategyUnknown && typed.Result.Strategy != environment.StrategyWorkspace {
		t.Fatalf("unexpected strategy, got %s", typed.Result.Strategy)
	}
	if len(typed.Result.FolderPaths) != 0 {
		t.Fatalf("expected no folder paths")
	}
}

func TestCancelExecution(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	called := false
	m.cancelFunc = func() {
		called = true
	}
	m.cancelExecution()
	if !called {
		t.Fatalf("expected cancel func to be invoked")
	}
	if m.cancelFunc != nil {
		t.Fatalf("expected cancel func to be cleared")
	}
}

func TestStreamPlanOutputCmd(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	ch := make(chan string, 1)
	ch <- "line"
	close(ch)
	m.outputChan = ch

	msg := m.streamPlanOutputCmd()()
	planMsg, ok := msg.(PlanOutputMsg)
	if !ok || planMsg.Line != "line" {
		t.Fatalf("expected plan output message")
	}
}

func TestStreamApplyOutputCmd(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	ch := make(chan string, 1)
	ch <- "line"
	close(ch)
	m.outputChan = ch

	msg := m.streamApplyOutputCmd()()
	applyMsg, ok := msg.(ApplyOutputMsg)
	if !ok || applyMsg.Line != "line" {
		t.Fatalf("expected apply output message")
	}
}

func TestStreamJSONMessagesCmd(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	ch := make(chan terraform.StreamMessage, 1)
	ch <- terraform.StreamMessage{Type: terraform.MessageTypeDiagnostic}
	close(ch)
	m.streamMsgChan = ch

	msg := m.streamJSONMessagesCmd()()
	streamMsg, ok := msg.(StreamMessageMsg)
	if !ok || streamMsg.Message.Type != terraform.MessageTypeDiagnostic {
		t.Fatalf("expected stream message")
	}
}

func TestProcessJSONStream(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	output := make(chan string)
	close(output)

	cmd := m.processJSONStream(output)
	if cmd == nil {
		t.Fatalf("expected stream command")
	}
	if m.streamMsgChan == nil || m.streamDone == nil || m.streamParser == nil {
		t.Fatalf("expected stream state to be initialized")
	}
	select {
	case <-m.streamDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected stream to complete")
	}
}

func TestHandlePlanStartError(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.applyView = views.NewApplyView(m.styles)
	m.operationState = terraform.NewOperationState()
	m.planRunning = true

	updated, cmd := m.handlePlanStart(PlanStartMsg{Error: errors.New("boom")})
	if cmd != nil {
		t.Fatalf("expected no command on error")
	}
	model := updated.(*Model)
	if model.planRunning {
		t.Fatalf("expected plan running to be false")
	}
	if len(model.operationState.GetDiagnostics()) == 0 {
		t.Fatalf("expected diagnostic to be recorded")
	}
}

func TestHandleApplyStartError(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.applyView = views.NewApplyView(m.styles)
	m.operationState = terraform.NewOperationState()
	m.applyRunning = true

	updated, cmd := m.handleApplyStart(ApplyStartMsg{Error: errors.New("boom")})
	if cmd != nil {
		t.Fatalf("expected no command on error")
	}
	model := updated.(*Model)
	if model.applyRunning {
		t.Fatalf("expected apply running to be false")
	}
	if len(model.operationState.GetDiagnostics()) == 0 {
		t.Fatalf("expected diagnostic to be recorded")
	}
}

func TestAppendDiagnostics(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.diagnosticsPanel.SetSize(40, 5)
	m.diagnosticsPanel.SetParsedText("parsed")
	m.diagnosticsHeight = 5

	content := m.appendDiagnostics("base")
	if !strings.Contains(content, "parsed") {
		t.Fatalf("expected diagnostics content to be appended")
	}
}

func TestHandleExecutionKeyToggles(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.width = 80
	m.height = 24

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if !handled || !m.resourceList.ShowStatus() {
		t.Fatalf("expected status column toggle")
	}

	handled, _ = m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if !handled || !m.showHistory {
		t.Fatalf("expected history toggle on")
	}

	m.historyEntries = []history.Entry{{Summary: "one"}}
	handled, _ = m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyTab})
	if !handled || !m.historyFocused {
		t.Fatalf("expected history focus toggle")
	}
}

func TestHandleExecutionKeyDiagnosticsFocus(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.useJSON = true
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.planRunning = true

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if !handled || !m.diagnosticsFocused || !m.showDiagnostics {
		t.Fatalf("expected diagnostics focus")
	}
	if m.execView != viewDiagnostics {
		t.Fatalf("expected diagnostics view")
	}
}

func TestHandleExecutionKeyCancel(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.planRunning = true
	called := false
	m.cancelFunc = func() {
		called = true
	}

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !handled || !called {
		t.Fatalf("expected cancel execution")
	}
}

func TestApplyEnvironmentSelectionErrors(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.planRunning = true
	if err := m.applyEnvironmentSelection(environment.Environment{Strategy: environment.StrategyFolder}); err == nil {
		t.Fatalf("expected error when command running")
	}

	m.planRunning = false
	if err := m.applyEnvironmentSelection(environment.Environment{Strategy: environment.StrategyFolder}); err == nil {
		t.Fatalf("expected error when executor missing")
	}

	if err := m.applyEnvironmentSelection(environment.Environment{Strategy: environment.StrategyUnknown}); err == nil {
		t.Fatalf("expected error for unsupported strategy")
	}
}

func TestCurrentUserName(t *testing.T) {
	if got := currentUserName(); got == "" {
		t.Fatalf("expected current user name")
	}
}

func TestApplyEnvironmentSelectionWorkspace(t *testing.T) {
	origNewWorkspaceManager := newWorkspaceManager
	defer func() {
		newWorkspaceManager = origNewWorkspaceManager
	}()

	manager := &fakeWorkspaceManager{}
	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return manager, nil
	}

	m := NewModel(&terraform.Plan{Resources: []terraform.ResourceChange{{Address: "a", Action: terraform.ActionCreate}}})
	m.planFilePath = "plan.tfplan"
	m.planRunFlags = []string{"-out=plan.tfplan"}
	m.planView = views.NewPlanView("", m.styles)
	m.operationState = terraform.NewOperationState()

	if err := m.applyEnvironmentSelection(environment.Environment{Name: "dev", Strategy: environment.StrategyWorkspace}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manager.switched != "dev" {
		t.Fatalf("expected workspace switch to dev")
	}
	if m.plan != nil || m.planFilePath != "" || m.planRunFlags != nil {
		t.Fatalf("expected plan state to reset")
	}
}

func TestCurrentUserNameFallbacks(t *testing.T) {
	orig := currentUserFunc
	currentUserFunc = func() (*user.User, error) {
		return nil, errors.New("no user")
	}
	t.Cleanup(func() {
		currentUserFunc = orig
	})

	t.Setenv("USER", "tester")
	t.Setenv("USERNAME", "")
	if got := currentUserName(); got != "tester" {
		t.Fatalf("expected USER fallback, got %q", got)
	}

	t.Setenv("USER", "")
	t.Setenv("USERNAME", "alt")
	if got := currentUserName(); got != "alt" {
		t.Fatalf("expected USERNAME fallback, got %q", got)
	}
}

func TestUpdatePlanStartAndApplyStart(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.applyView = views.NewApplyView(m.styles)
	m.operationState = terraform.NewOperationState()

	planResult := terraform.NewExecutionResult()
	planResult.Finish()
	planOutput := make(chan string)
	close(planOutput)

	m.Update(PlanStartMsg{Result: planResult, Output: planOutput})
	if m.outputChan == nil {
		t.Fatalf("expected plan output channel set")
	}

	applyResult := terraform.NewExecutionResult()
	applyResult.Finish()
	applyOutput := make(chan string)
	close(applyOutput)

	m.Update(ApplyStartMsg{Result: applyResult, Output: applyOutput})
	if m.outputChan == nil {
		t.Fatalf("expected apply output channel set")
	}
}

func TestUpdateLayoutWithHistoryAndDiagnostics(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.width = 120
	m.height = 40
	m.showHistory = true
	m.historyPanel = components.NewHistoryPanel(m.styles)
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.diagnosticsHeight = 8
	m.diagnosticsFocused = true

	m.updateLayout()
	if m.resourceList.View() == "" {
		t.Fatalf("expected resource list to render")
	}
}

func TestDetectEnvironmentsCmdUsesWorkspaceManager(t *testing.T) {
	origNewWorkspaceManager := newWorkspaceManager
	origNewEnvironmentDetector := newEnvironmentDetector
	defer func() {
		newWorkspaceManager = origNewWorkspaceManager
		newEnvironmentDetector = origNewEnvironmentDetector
	}()

	detector := &fakeDetector{
		result: environment.DetectionResult{
			Strategy:   environment.StrategyWorkspace,
			Workspaces: []string{"dev"},
		},
	}
	newEnvironmentDetector = func(_ string) (environmentDetector, error) {
		return detector, nil
	}
	manager := &fakeWorkspaceManager{current: "dev"}
	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return manager, nil
	}

	m := NewModel(&terraform.Plan{})
	m.envWorkDir = t.TempDir()

	msg := m.detectEnvironmentsCmd()()
	typed, ok := msg.(EnvironmentDetectedMsg)
	if !ok {
		t.Fatalf("expected environment detected message")
	}
	if typed.Current != "dev" {
		t.Fatalf("expected current workspace, got %q", typed.Current)
	}
}

func TestDetectEnvironmentsCmdUsesFolderMatch(t *testing.T) {
	origNewEnvironmentDetector := newEnvironmentDetector
	defer func() {
		newEnvironmentDetector = origNewEnvironmentDetector
	}()

	workDir := t.TempDir()
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		t.Fatalf("abs workdir: %v", err)
	}
	detector := &fakeDetector{
		result: environment.DetectionResult{
			Strategy:    environment.StrategyFolder,
			FolderPaths: []string{absWorkDir},
		},
	}
	newEnvironmentDetector = func(_ string) (environmentDetector, error) {
		return detector, nil
	}

	m := NewModel(&terraform.Plan{})
	m.envWorkDir = workDir

	msg := m.detectEnvironmentsCmd()()
	typed, ok := msg.(EnvironmentDetectedMsg)
	if !ok {
		t.Fatalf("expected environment detected message")
	}
	if typed.Current != absWorkDir {
		t.Fatalf("expected current folder to match abs workdir, got %q", typed.Current)
	}
}

func TestRecordOperationCmd(t *testing.T) {
	store, err := history.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close history store: %v", closeErr)
		}
	})

	m := NewModel(&terraform.Plan{})
	m.historyLogger = history.NewLogger(store, history.LevelStandard)
	m.envCurrent = "dev"
	m.applyStartedAt = time.Now().Add(-time.Second)
	result := &terraform.ExecutionResult{
		ExitCode: 1,
		Duration: time.Second,
		Output:   "output",
	}

	cmd := m.recordOperationCmd("apply", []string{"-auto-approve"}, true, m.applyStartedAt, result, "", errors.New("boom"))
	if cmd == nil {
		t.Fatalf("expected operation command")
	}
	cmd()

	entries, err := store.QueryOperations(history.OperationFilter{Action: "apply", Limit: 5})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if entries[0].Environment != "dev" || entries[0].Status != history.StatusFailed {
		t.Fatalf("unexpected operation entry data")
	}
}

func TestWaitPlanCompleteCmdError(t *testing.T) {
	result := terraform.NewExecutionResult()
	result.Error = errors.New("plan failed")
	result.Finish()

	m := NewModel(&terraform.Plan{})
	msg := m.waitPlanCompleteCmd(result)()
	planMsg, ok := msg.(PlanCompleteMsg)
	if !ok || planMsg.Error == nil {
		t.Fatalf("expected plan error")
	}
}

func TestWaitApplyCompleteCmdError(t *testing.T) {
	result := terraform.NewExecutionResult()
	result.Error = errors.New("apply failed")
	result.Finish()

	m := NewModel(&terraform.Plan{})
	msg := m.waitApplyCompleteCmd(result)()
	applyMsg, ok := msg.(ApplyCompleteMsg)
	if !ok || applyMsg.Error == nil || applyMsg.Success {
		t.Fatalf("expected apply error")
	}
}

func TestWaitPlanCompleteCmdNoChanges(t *testing.T) {
	result := terraform.NewExecutionResult()
	result.Output = "No changes. Infrastructure is up-to-date."
	result.Finish()

	m := NewModel(&terraform.Plan{})
	msg := m.waitPlanCompleteCmd(result)()
	planMsg, ok := msg.(PlanCompleteMsg)
	if !ok || planMsg.Error != nil {
		t.Fatalf("expected plan without error")
	}
	if planMsg.Plan == nil {
		t.Fatalf("expected plan to be parsed")
	}
}

func TestWaitApplyCompleteCmdSuccess(t *testing.T) {
	result := terraform.NewExecutionResult()
	result.Output = "ok"
	result.Finish()

	m := NewModel(&terraform.Plan{})
	msg := m.waitApplyCompleteCmd(result)()
	applyMsg, ok := msg.(ApplyCompleteMsg)
	if !ok || applyMsg.Error != nil || !applyMsg.Success {
		t.Fatalf("expected apply success")
	}
}

func TestUpdateEnvironmentDetectedError(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true

	_, cmd := m.Update(EnvironmentDetectedMsg{Error: errors.New("boom")})
	if cmd == nil {
		t.Fatalf("expected toast clear command")
	}
	if !strings.Contains(m.toastMessage, "Environment detection failed") {
		t.Fatalf("expected error toast, got %q", m.toastMessage)
	}
}

func TestUpdateHistoryLoadedError(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true

	_, cmd := m.Update(HistoryLoadedMsg{Error: errors.New("boom")})
	if cmd != nil {
		cmd()
	}
	if m.err == nil || m.err.Error() != "boom" {
		t.Fatalf("expected history error to be recorded")
	}
}

func TestUpdateClearToast(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.toastMessage = "hi"
	m.toastIsError = true

	m.Update(ClearToastMsg{})
	if m.toastMessage != "" || m.toastIsError {
		t.Fatalf("expected toast to be cleared")
	}
}

func TestViewModalStates(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.width = 80
	m.height = 24

	m.modalState = ModalSettings
	if out := m.View(); !strings.Contains(out, "Settings") {
		t.Fatalf("expected settings view")
	}

	m.modalState = ModalHelp
	if out := m.View(); !strings.Contains(out, "tftui keybinds") {
		t.Fatalf("expected help view")
	}

	m.modalState = ModalEnvSelector
	m.envOptions = []environment.Environment{{Name: "dev", Strategy: environment.StrategyWorkspace}}
	m.envStrategy = environment.StrategyWorkspace
	m.envView.SetMode(views.EnvViewEnvironments)
	if out := m.View(); !strings.Contains(out, "Select Environment") {
		t.Fatalf("expected env selector view")
	}
}

func TestHandleExecutionKeyPlanConfirm(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewPlanConfirm

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !handled || m.execView != viewMain {
		t.Fatalf("expected plan confirm to return to main view")
	}

	m.execView = viewPlanConfirm
	called := false
	m.cancelFunc = func() {
		called = true
	}
	handled, _ = m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !handled || !called {
		t.Fatalf("expected cancel execution")
	}
}

func TestHandleExecutionKeyPlanOutputExit(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewPlanOutput

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled || m.execView != viewMain {
		t.Fatalf("expected plan output to return to main view")
	}
}

func TestHandleExecutionKeyCompactProgressExit(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewCompactProgress
	m.showDiagnostics = true

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled || m.execView != viewMain || m.showDiagnostics {
		t.Fatalf("expected compact progress to return to main and clear diagnostics")
	}
}

func TestHandleExecutionKeyDiagnosticsToggleCompact(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.useJSON = true
	m.execView = viewDiagnostics
	m.planRunning = true
	m.showDiagnostics = true

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	if !handled || !m.showCompactProgress {
		t.Fatalf("expected compact progress toggle")
	}
	if m.execView != viewCompactProgress {
		t.Fatalf("expected compact progress view")
	}
}

func TestHandleExecutionKeyHistoryDetailExit(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewHistoryDetail

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled || m.execView != viewMain {
		t.Fatalf("expected history detail to return to main")
	}
}

func TestHandleExecutionKeyRawLogsToggle(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.showRawLogs = false

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	if !handled || !m.showRawLogs {
		t.Fatalf("expected raw logs toggle")
	}
}

func TestBeginPlanWithoutExecutor(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	cmd := m.beginPlan()
	if cmd != nil {
		t.Fatalf("expected no command without executor")
	}
	if m.err == nil {
		t.Fatalf("expected error when executor missing")
	}
}

func TestHandlePlanStartSuccess(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.applyView = views.NewApplyView(m.styles)
	m.operationState = terraform.NewOperationState()

	output := make(chan string)
	close(output)
	result := terraform.NewExecutionResult()
	result.Finish()

	updated, cmd := m.handlePlanStart(PlanStartMsg{Result: result, Output: output})
	if cmd == nil {
		t.Fatalf("expected command batch")
	}
	model := updated.(*Model)
	if model.outputChan == nil {
		t.Fatalf("expected output channel to be set")
	}
}

func TestHandleApplyStartSuccess(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.applyView = views.NewApplyView(m.styles)
	m.operationState = terraform.NewOperationState()

	output := make(chan string)
	close(output)
	result := terraform.NewExecutionResult()
	result.Finish()

	updated, cmd := m.handleApplyStart(ApplyStartMsg{Result: result, Output: output})
	if cmd == nil {
		t.Fatalf("expected command batch")
	}
	model := updated.(*Model)
	if model.outputChan == nil {
		t.Fatalf("expected output channel to be set")
	}
}

func TestHandlePlanCompleteSuccess(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.applyView = views.NewApplyView(m.styles)
	m.operationState = terraform.NewOperationState()
	plan := &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "a", Action: terraform.ActionCreate}}}

	updated, cmd := m.handlePlanComplete(PlanCompleteMsg{Plan: plan, Output: "No changes."})
	if cmd != nil {
		cmd()
	}
	model := updated.(*Model)
	if model.plan == nil || len(model.plan.Resources) != 1 {
		t.Fatalf("expected plan to be set")
	}
	if model.applyView == nil {
		t.Fatalf("expected apply view to exist")
	}
}

func TestUpdateModalDismissals(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true

	m.modalState = ModalSettings
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.modalState != ModalSettings {
		t.Fatalf("expected settings modal to ignore unrelated keys")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.modalState != ModalNone {
		t.Fatalf("expected settings modal to close on esc")
	}

	m.modalState = ModalHelp
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.modalState != ModalNone {
		t.Fatalf("expected help modal to close")
	}
}

func TestUpdateExecViewRoutesToApplyView(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewApplyOutput
	m.applyView = views.NewApplyView(m.styles)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated == nil {
		t.Fatalf("expected update to return model")
	}
}

func TestViewExecutionModes(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.executionMode = true
	m.width = 80
	m.height = 24
	m.applyView = views.NewApplyView(m.styles)
	m.planView = views.NewPlanView("", m.styles)
	m.progressCompact = components.NewProgressCompact(terraform.NewOperationState(), m.styles)
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.diagnosticsPanel.SetSize(40, 5)
	m.diagnosticsPanel.SetParsedText("parsed")
	m.historyView = views.NewHistoryView(m.styles)

	m.execView = viewPlanConfirm
	if out := m.View(); out == "" {
		t.Fatalf("expected plan confirm view")
	}

	m.execView = viewApplyOutput
	if out := m.View(); out == "" {
		t.Fatalf("expected apply output view")
	}

	m.execView = viewCompactProgress
	if out := m.View(); out == "" {
		t.Fatalf("expected compact progress view")
	}

	m.execView = viewDiagnostics
	if out := m.View(); out == "" {
		t.Fatalf("expected diagnostics view")
	}

	m.execView = viewHistoryDetail
	if out := m.View(); out == "" {
		t.Fatalf("expected history detail view")
	}
}

func TestUpdatePlanOutputAppendsLine(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.useJSON = false
	m.applyView = views.NewApplyView(m.styles)
	m.applyView.SetSize(80, 10)

	_, cmd := m.Update(PlanOutputMsg{Line: "hello"})
	if cmd == nil {
		t.Fatalf("expected stream command")
	}
	if !strings.Contains(m.applyView.View(), "hello") {
		t.Fatalf("expected output line in apply view")
	}
}

func TestUpdateApplyOutputAppendsLine(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.useJSON = false
	m.applyView = views.NewApplyView(m.styles)
	m.applyView.SetSize(80, 10)

	_, cmd := m.Update(ApplyOutputMsg{Line: "world"})
	if cmd == nil {
		t.Fatalf("expected stream command")
	}
	if !strings.Contains(m.applyView.View(), "world") {
		t.Fatalf("expected output line in apply view")
	}
}

func TestUpdateHistoryDetailError(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true

	_, cmd := m.Update(HistoryDetailMsg{Error: errors.New("boom")})
	if cmd == nil {
		t.Fatalf("expected toast clear command")
	}
	if !strings.Contains(m.toastMessage, "History error") {
		t.Fatalf("expected history error toast")
	}
}

func TestUpdateHistoryLoadedSuccess(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.historyPanel = components.NewHistoryPanel(m.styles)
	entries := []history.Entry{{Summary: "one"}}

	m.Update(HistoryLoadedMsg{Entries: entries})
	if len(m.historyEntries) != 1 {
		t.Fatalf("expected history entries to update")
	}
}

func TestUpdateStreamMessage(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.operationState = terraform.NewOperationState()

	msg := terraform.StreamMessage{
		Type: terraform.MessageTypeDiagnostic,
		Diagnostic: &terraform.Diagnostic{
			Severity: "warning",
			Summary:  "note",
		},
	}
	_, cmd := m.Update(StreamMessageMsg{Message: msg})
	if cmd == nil {
		t.Fatalf("expected stream command")
	}
	if len(m.operationState.GetDiagnostics()) != 1 {
		t.Fatalf("expected diagnostic to be recorded")
	}
}

func TestUpdateOperationStateUpdate(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.operationState = terraform.NewOperationState()
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.diagnosticsPanel.SetSize(40, 5)

	m.operationState.AddDiagnostic(terraform.Diagnostic{Severity: "error", Summary: "bad"})
	m.Update(OperationStateUpdateMsg{})
	if m.diagnosticsPanel.View() == "" {
		t.Fatalf("expected diagnostics to be rendered")
	}
}

func TestUpdateEnvironmentDetectedSuccess(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envWorkDir = t.TempDir()

	result := environment.DetectionResult{
		BaseDir:    m.envWorkDir,
		Strategy:   environment.StrategyWorkspace,
		Workspaces: []string{"dev"},
	}
	result.Environments = environment.BuildEnvironments(result, "")
	m.Update(EnvironmentDetectedMsg{Result: result, Current: "dev"})
	if m.envCurrent != "dev" {
		t.Fatalf("expected current environment to be set")
	}
	if len(m.envOptions) == 0 {
		t.Fatalf("expected environment options")
	}
}

func TestClearToastCmd(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	cmd := m.clearToastCmd(1 * time.Millisecond)
	msg := cmd()
	if _, ok := msg.(ClearToastMsg); !ok {
		t.Fatalf("expected ClearToastMsg")
	}
}

func TestInitExecutionModeAutoPlan(t *testing.T) {
	origNewEnvironmentDetector := newEnvironmentDetector
	defer func() {
		newEnvironmentDetector = origNewEnvironmentDetector
	}()

	newEnvironmentDetector = func(_ string) (environmentDetector, error) {
		return &fakeDetector{result: environment.DetectionResult{}}, nil
	}

	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.autoPlan = true
	m.executor = &terraform.Executor{}
	m.envWorkDir = t.TempDir()

	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected init command")
	}
	if !m.planRunning {
		t.Fatalf("expected plan to start")
	}
}

func TestWaitPlanCompleteCmdUsesStdout(t *testing.T) {
	result := terraform.NewExecutionResult()
	result.Stdout = "No changes. Infrastructure is up-to-date."
	result.Finish()

	m := NewModel(&terraform.Plan{})
	msg := m.waitPlanCompleteCmd(result)()
	planMsg, ok := msg.(PlanCompleteMsg)
	if !ok || planMsg.Error != nil {
		t.Fatalf("expected plan without error")
	}
	if planMsg.Output != result.Stdout {
		t.Fatalf("expected stdout to be used")
	}
}

func TestWaitPlanCompleteCmdUsesStreamParser(t *testing.T) {
	stream := `{"type":"planned_change","resource":{"addr":"aws_instance.web","resource_type":"aws_instance","resource_name":"web"},"change":{"actions":["create"],"before":null,"after":{}}}`
	parser := tfparser.NewStreamParser()
	if err := parser.Parse(strings.NewReader(stream), nil); err != nil {
		t.Fatalf("parse stream: %v", err)
	}

	done := make(chan struct{})
	close(done)

	result := terraform.NewExecutionResult()
	result.Finish()

	m := NewModel(&terraform.Plan{})
	m.useJSON = true
	m.streamParser = parser
	m.streamDone = done

	msg := m.waitPlanCompleteCmd(result)()
	planMsg, ok := msg.(PlanCompleteMsg)
	if !ok || planMsg.Plan == nil {
		t.Fatalf("expected plan from stream parser")
	}
	if len(planMsg.Plan.Resources) != 1 {
		t.Fatalf("expected one planned resource")
	}
}

func TestWaitPlanCompleteCmdStreamFallback(t *testing.T) {
	stream := `{"type":"planned_change","resource":{"addr":"aws_instance.web","resource_type":"aws_instance","resource_name":"web"},"change":{"actions":["create"],"before":null,"after":{}}}`
	result := terraform.NewExecutionResult()
	result.Output = stream
	result.Finish()

	m := NewModel(&terraform.Plan{})
	m.useJSON = true
	m.streamParser = tfparser.NewStreamParser()

	msg := m.waitPlanCompleteCmd(result)()
	planMsg, ok := msg.(PlanCompleteMsg)
	if !ok || planMsg.Plan == nil {
		t.Fatalf("expected plan from stream fallback")
	}
}

func TestWaitPlanCompleteCmdJSONFallback(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "plans", "sample.json"))
	if err != nil {
		t.Fatalf("read sample plan: %v", err)
	}
	result := terraform.NewExecutionResult()
	result.Output = string(data)
	result.Finish()

	m := NewModel(&terraform.Plan{})
	msg := m.waitPlanCompleteCmd(result)()
	planMsg, ok := msg.(PlanCompleteMsg)
	if !ok || planMsg.Plan == nil {
		t.Fatalf("expected plan from json fallback")
	}
	if len(planMsg.Plan.Resources) == 0 {
		t.Fatalf("expected plan resources")
	}
}

func TestWaitPlanCompleteCmdParseError(t *testing.T) {
	result := terraform.NewExecutionResult()
	result.Output = "not a plan"
	result.Finish()

	m := NewModel(&terraform.Plan{})
	msg := m.waitPlanCompleteCmd(result)()
	planMsg, ok := msg.(PlanCompleteMsg)
	if !ok || planMsg.Error == nil {
		t.Fatalf("expected parse error")
	}
}

func TestSyncHistorySelectionClamps(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.historyPanel = components.NewHistoryPanel(m.styles)
	m.historyEntries = []history.Entry{{Summary: "one"}, {Summary: "two"}}
	m.historySelected = 5
	m.syncHistorySelection()
	if m.historySelected != 1 {
		t.Fatalf("expected selection to clamp to last entry")
	}

	m.historySelected = -2
	m.syncHistorySelection()
	if m.historySelected != 0 {
		t.Fatalf("expected selection to clamp to zero")
	}
}

func TestRecordHistoryCmdNilStore(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	if cmd := m.recordHistoryCmd(history.StatusSuccess, "summary", "", nil, nil); cmd != nil {
		t.Fatalf("expected nil command when store is nil")
	}
}

func TestLoadHistoryDetailCmdNilStore(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	if cmd := m.loadHistoryDetailCmd(1); cmd != nil {
		t.Fatalf("expected nil command when store is nil")
	}
}

func TestChanToReaderWriteError(_ *testing.T) {
	ch := make(chan string, 1)
	reader := chanToReader(ch)
	if closer, ok := reader.(io.Closer); ok {
		_ = closer.Close()
	}
	ch <- "line"
	close(ch)
}

func TestHandleExecutionKeyPlanOutputQuitWhileRunning(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewPlanOutput
	m.planRunning = true

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled || !m.quitting {
		t.Fatalf("expected quit while running")
	}
}

func TestHandleExecutionKeyHistoryDetailQuit(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewHistoryDetail

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled || !m.quitting {
		t.Fatalf("expected quit in history detail")
	}
}

func TestUpdateSearchClearWhenInactive(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.searchInput.SetValue("query")

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.searchInput.Value() != "" {
		t.Fatalf("expected search to clear")
	}
}

func TestUpdateEnvSelectorNavigation(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.modalState = ModalEnvSelector
	m.envOptions = []environment.Environment{
		{Name: "one", Strategy: environment.StrategyWorkspace},
		{Name: "two", Strategy: environment.StrategyWorkspace},
	}
	m.envView.SetMode(views.EnvViewEnvironments)
	m.envView.SetEnvironments(m.envOptions, environment.StrategyWorkspace, "", ".")
	m.envView.SetSize(60, 10)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if selected := m.envView.SelectedEnvironment(); selected == nil || selected.Name != "two" {
		t.Fatalf("expected selection to move down")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if selected := m.envView.SelectedEnvironment(); selected == nil || selected.Name != "one" {
		t.Fatalf("expected selection to move up")
	}
}

func TestUpdateExecViewAppliesToApplyView(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewPlanOutput
	m.applyView = views.NewApplyView(m.styles)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if updated == nil {
		t.Fatalf("expected model update")
	}
}

func TestHandleExecutionKeyCompactToggleInPlanOutput(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewPlanOutput
	m.useJSON = true

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	if !handled || !m.showCompactProgress {
		t.Fatalf("expected compact progress toggle")
	}
}

func TestRecordHistoryCmdError(t *testing.T) {
	store, err := history.Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close history store: %v", err)
	}
	m := NewModel(&terraform.Plan{})
	m.historyStore = store

	cmd := m.recordHistoryCmd(history.StatusSuccess, "summary", "", nil, nil)
	if cmd == nil {
		t.Fatalf("expected history command")
	}
	msg := cmd()
	if _, ok := msg.(HistoryLoadedMsg); !ok {
		t.Fatalf("expected HistoryLoadedMsg")
	}
}

func TestSyncHistorySelectionNilPanel(_ *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.historyPanel = nil
	m.syncHistorySelection()
}

func TestRenderSearchBarPlaceholder(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.width = 80
	out := m.renderSearchBar()
	if !strings.Contains(out, "Search:") {
		t.Fatalf("expected search placeholder")
	}
}

func TestStreamCmdsHandleNilChannels(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	if msg := m.streamPlanOutputCmd()(); msg != nil {
		t.Fatalf("expected nil plan output message")
	}
	if msg := m.streamApplyOutputCmd()(); msg != nil {
		t.Fatalf("expected nil apply output message")
	}
	if msg := m.streamJSONMessagesCmd()(); msg != nil {
		t.Fatalf("expected nil stream message")
	}
}

func TestStreamApplyOutputCmdClosed(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	ch := make(chan string)
	close(ch)
	m.outputChan = ch
	if msg := m.streamApplyOutputCmd()(); msg != nil {
		t.Fatalf("expected nil apply output message on closed channel")
	}
}

func TestHandleHistoryKeysUnknown(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	handled, cmd := m.handleHistoryKeys("x")
	if handled {
		t.Fatalf("expected unhandled key")
	}
	if cmd != nil {
		t.Fatalf("expected no command")
	}
}

func TestUpdatePlanOutputWhenJSON(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.useJSON = true
	m.applyView = views.NewApplyView(m.styles)

	_, cmd := m.Update(PlanOutputMsg{Line: "line"})
	if cmd != nil {
		t.Fatalf("expected nil cmd when JSON streaming")
	}
}

func TestHandleExecutionKeyPlanConfirmQuit(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewPlanConfirm

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled || !m.quitting {
		t.Fatalf("expected quitting in plan confirm")
	}
}

type fakeWorkspaceManager struct {
	current    string
	currentErr error
	switched   string
	switchErr  error
}

func (f *fakeWorkspaceManager) Current(_ context.Context) (string, error) {
	return f.current, f.currentErr
}

func (f *fakeWorkspaceManager) Switch(_ context.Context, name string) error {
	f.switched = name
	return f.switchErr
}

type fakeDetector struct {
	result environment.DetectionResult
	err    error
}

func (f *fakeDetector) Detect(_ context.Context) (environment.DetectionResult, error) {
	return f.result, f.err
}
