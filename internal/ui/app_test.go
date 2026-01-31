package ui

import (
	"context"
	"errors"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/ui/views"
	"github.com/ushiradineth/lazytf/internal/utils"
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
	if m.toast == nil || !m.toast.IsVisible() {
		t.Fatalf("expected error toast to be visible")
	}
	if cmd == nil {
		t.Fatalf("expected toast clear command")
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
	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil init command in non-execution mode")
	}

	m.executionMode = true
	if cmd := m.Init(); cmd == nil {
		t.Fatalf("expected init command to be set in execution mode")
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
	// With new panel layout, main content includes panel borders and layout
	if split == "" {
		t.Fatalf("expected non-empty split layout")
	}

	m.width = 80
	m.updateLayout()
	single := m.renderMainContent()
	// New panel system always renders panel layout, even with narrow width
	if single == "" {
		t.Fatalf("expected non-empty single layout")
	}
	// Verify that the resource list content is present in the output
	if !strings.Contains(single, "Resources") {
		t.Fatalf("expected Resources panel title in output")
	}
}

func TestHelpModalContent(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.width = 80
	m.height = 24
	m.ready = true
	m.modalState = ModalHelp

	// Trigger the help modal content update
	m.updateHelpModalContent()

	if m.helpModal == nil {
		t.Fatalf("expected help modal to be initialized")
	}
	if !m.helpModal.IsVisible() {
		t.Fatalf("expected help modal to be visible")
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
	if utils.MinInt(5, 2) != 2 {
		t.Fatalf("expected MinInt to return smaller value")
	}
	if utils.MinInt(2, 5) != 2 {
		t.Fatalf("expected MinInt to return smaller value")
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
	m.applyRunning = true
	m.execView = viewMain

	m.handleApplyComplete(ApplyCompleteMsg{Success: true, Result: &terraform.ExecutionResult{}})
	// In new layout, we stay in viewMain after operations complete
	if m.execView != viewMain {
		t.Fatalf("expected to stay in main view after success, got %d", m.execView)
	}
	if m.plan == nil || len(m.plan.Resources) != 0 {
		t.Fatalf("expected plan to be cleared after apply")
	}
}

func TestApplyCompleteFailureStaysOnOutput(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.applyView = views.NewApplyView(m.styles)
	m.applyRunning = true
	m.execView = viewMain

	m.handleApplyComplete(ApplyCompleteMsg{Success: false, Error: errors.New("fail"), Result: &terraform.ExecutionResult{}})
	// In new layout, we stay in viewMain even on failure
	if m.execView != viewMain {
		t.Fatalf("expected to remain in main view on failure, got %d", m.execView)
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
	// History view is now embedded in mainArea
	m.mainArea = NewMainArea(m.styles, nil, nil, nil)
	m.mainArea.SetSize(80, 20)
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
	m.envCurrent = consts.EnvDev

	entry := history.Entry{
		StartedAt:   time.Now(),
		FinishedAt:  time.Now(),
		Duration:    time.Second,
		Status:      history.StatusSuccess,
		Summary:     "ok",
		Environment: consts.EnvDev,
		Output:      "output text",
	}
	if err := store.RecordApply(entry); err != nil {
		t.Fatalf("record history: %v", err)
	}
	entries, err := store.ListRecentForEnvironment(consts.EnvDev, 5)
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
	// History detail is now shown in mainArea [0], not as a full-screen view
	if m.mainArea == nil || m.mainArea.GetMode() != ModeHistoryDetail {
		t.Fatalf("expected mainArea to be in history detail mode")
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

	// Press 'a' to open confirm modal
	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !handled || m.modalState != ModalConfirmApply {
		t.Fatalf("expected confirm apply modal, got modalState=%d", m.modalState)
	}
	if cmd != nil {
		cmd()
	}

	// Press 'y' to confirm (handled by modal key handler)
	handled, cmd = m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !handled {
		t.Fatalf("expected key to be handled")
	}
	if m.modalState != ModalNone {
		t.Fatalf("expected modal to be closed after confirm")
	}
	if cmd != nil {
		cmd()
	}

	// In the new layout, we stay in viewMain and main area stays in logs mode
	// to show apply output (not switching immediately to diff)
	m.handleApplyComplete(ApplyCompleteMsg{Success: true, Result: &terraform.ExecutionResult{}})
	if m.execView != viewMain {
		t.Fatalf("expected to stay in main view after apply, got %d", m.execView)
	}
	// Apply completes should keep logs visible, not switch to diff
	if m.mainArea != nil && m.mainArea.GetMode() != ModeLogs {
		t.Fatalf("expected main area to stay in logs mode after apply, got %d", m.mainArea.GetMode())
	}
}

func TestPlanFailureStaysOnOutputView(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.applyView = views.NewApplyView(m.styles)
	m.executor = &terraform.Executor{}

	m.beginPlan()
	m.handlePlanComplete(PlanCompleteMsg{Error: errors.New("plan failed")})
	// In the new layout, we stay in viewMain with main area in diff mode after operations complete
	if m.execView != viewMain {
		t.Fatalf("expected to stay in main view, got %d", m.execView)
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
	// History view is now embedded in mainArea
	m.mainArea = NewMainArea(m.styles, nil, nil, nil)
	m.mainArea.SetSize(80, 10)

	m.Update(HistoryDetailMsg{Entry: history.Entry{Output: ""}})
	m.mainArea.EnterHistoryDetail() // Ensure we're in history detail mode
	out := m.mainArea.GetHistoryView().View()
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
	m.envCurrent = consts.EnvDev

	cmd := m.recordHistoryCmd(history.StatusSuccess, "summary", m.lastPlanOutput, &terraform.ExecutionResult{}, nil)
	if cmd == nil {
		t.Fatalf("expected history command")
	}
	cmd()

	entries, err := store.ListRecentForEnvironment(consts.EnvDev, 5)
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

	m.envCurrent = consts.EnvDev
	m.envStrategy = environment.StrategyWorkspace
	if got := m.envStatusLabel(); got != "dev (workspace)" {
		t.Fatalf("expected strategy label, got %q", got)
	}
}

func TestFormatLogTimestamp(t *testing.T) {
	ts := "2024-01-02T03:04:05Z"
	if got := utils.FormatLogTimestamp(ts); got != "2024-01-02 03:04:05 +00:00" {
		t.Fatalf("expected formatted timestamp, got %q", got)
	}

	raw := "not-a-timestamp"
	if got := utils.FormatLogTimestamp(raw); got != raw {
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
	cmd := m.buildCommand("apply", []string{"-var-file=dev.tfvars"}, true)
	if !strings.Contains(cmd, "-auto-approve") {
		t.Fatalf("expected auto-approve flag in command: %s", cmd)
	}
	if !strings.Contains(cmd, "-var-file=dev.tfvars") {
		t.Fatalf("expected custom flag in command: %s", cmd)
	}

	cmd = m.buildCommand("apply", []string{"-auto-approve"}, true)
	if strings.Count(cmd, "-auto-approve") != 1 {
		t.Fatalf("expected single auto-approve flag in command: %s", cmd)
	}
}

func TestContainsFlag(t *testing.T) {
	if !containsFlag([]string{"apply", "-auto-approve"}, "-auto-approve") {
		t.Fatalf("expected to find flag")
	}
	if containsFlag([]string{"plan", "-out=plan"}, "-auto-approve") {
		t.Fatalf("did not expect to find flag")
	}
}

func TestPlanOutputPath(t *testing.T) {
	if got := planOutputPath([]string{"-out", consts.PlanTFPlan}); got != consts.PlanTFPlan {
		t.Fatalf("expected plan output path, got %q", got)
	}
	if got := planOutputPath([]string{"-out=plan.tfplan"}); got != consts.PlanTFPlan {
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
		Workspaces:  []string{consts.EnvDev},
		FolderPaths: []string{folderPath},
	}
	result.Environments = environment.BuildEnvironments(result, "")
	m := NewModel(&terraform.Plan{})
	m.envWorkDir = baseDir
	m.setEnvironmentOptions(result, environment.StrategyMixed, "")

	if len(m.envOptions) != 2 {
		t.Fatalf("expected 2 env options, got %d", len(m.envOptions))
	}
	if m.envOptions[0].Name != consts.EnvDev || m.envOptions[0].Strategy != environment.StrategyWorkspace {
		t.Fatalf("unexpected workspace option: %+v", m.envOptions[0])
	}
	if m.envOptions[1].Path != folderPath || m.envOptions[1].Strategy != environment.StrategyFolder {
		t.Fatalf("unexpected folder option: %+v", m.envOptions[1])
	}
}

func TestUpdateExecutionViewForStreaming(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewPlanConfirm
	m.updateExecutionViewForStreaming()
	if m.execView != viewPlanConfirm {
		t.Fatalf("expected plan confirm view to remain, got %v", m.execView)
	}

	// History detail is now shown in mainArea [0], not as a full-screen view
	m.mainArea = NewMainArea(m.styles, nil, nil, nil)
	m.mainArea.EnterHistoryDetail()
	m.execView = viewMain
	m.updateExecutionViewForStreaming()
	if m.mainArea.GetMode() != ModeHistoryDetail {
		t.Fatalf("expected mainArea to remain in history detail mode")
	}
	if m.execView != viewMain {
		t.Fatalf("expected main view, got %v", m.execView)
	}

	m.mainArea.ExitHistoryDetail()
	m.execView = viewCommandLog
	m.updateExecutionViewForStreaming()
	if m.execView != viewMain {
		t.Fatalf("expected main view, got %v", m.execView)
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
	if !strings.Contains(summary, "+1") || !strings.Contains(summary, "±1") {
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
	if utils.MaxInt(1, 3) != 3 {
		t.Fatalf("expected MaxInt to return larger value")
	}
	if utils.MaxInt(5, 2) != 5 {
		t.Fatalf("expected MaxInt to return larger value")
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

func TestSettingsModal(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.width = 80
	m.height = 24

	m.updateSettingsModalContent()
	if m.settingsModal == nil {
		t.Fatalf("expected settings modal to be initialized")
	}
	if !m.settingsModal.IsVisible() {
		t.Fatalf("expected settings modal to be visible")
	}
	out := m.settingsModal.View()
	if !strings.Contains(out, "No configuration loaded.") {
		t.Fatalf("expected settings fallback text")
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

func TestHandlePlanStartError(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.applyView = views.NewApplyView(m.styles)
	m.operationState = terraform.NewOperationState()
	m.planRunning = true

	updated, cmd := m.handlePlanStart(PlanStartMsg{Error: errors.New("boom")})
	if cmd != nil {
		t.Fatalf("expected no command on error")
	}
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected *Model type")
	}
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
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected *Model type")
	}
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
	m.diagnosticsPanel.AppendSessionLog("Planned", "terraform plan", "session log output")
	m.diagnosticsHeight = 5

	content := m.appendDiagnostics("base")
	if !strings.Contains(content, "session log output") {
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
	m.commandLogPanel = components.NewCommandLogPanel(m.styles)
	m.panelManager = NewPanelManager()
	m.panelManager.RegisterPanel(PanelCommandLog, m.commandLogPanel)
	m.planRunning = true

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if !handled {
		t.Fatalf("expected D key to be handled")
	}
	// In new layout, D focuses the command log panel instead of switching views
	if !m.panelManager.IsCommandLogVisible() {
		t.Fatalf("expected command log to be visible")
	}
	if m.panelManager.GetFocusedPanel() != PanelCommandLog {
		t.Fatalf("expected command log panel to be focused")
	}
	// We stay in viewMain
	if m.execView != viewMain {
		t.Fatalf("expected to stay in main view, got %v", m.execView)
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
	m.planFilePath = consts.PlanTFPlan
	m.planRunFlags = []string{"-out=plan.tfplan"}
	m.planView = views.NewPlanView("", m.styles)
	m.operationState = terraform.NewOperationState()

	if err := m.applyEnvironmentSelection(environment.Environment{Name: consts.EnvDev, Strategy: environment.StrategyWorkspace}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manager.switched != consts.EnvDev {
		t.Fatalf("expected workspace switch to dev")
	}
	if m.plan != nil || m.planFilePath != "" || m.planRunFlags != nil {
		t.Fatalf("expected plan state to reset")
	}
}

func TestApplyEnvironmentSelectionSavesPreference(t *testing.T) {
	origNewWorkspaceManager := newWorkspaceManager
	defer func() {
		newWorkspaceManager = origNewWorkspaceManager
	}()

	manager := &fakeWorkspaceManager{}
	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return manager, nil
	}

	// Create a temp directory for testing
	tmpDir := t.TempDir()

	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.envWorkDir = tmpDir

	// Apply a workspace selection
	env := environment.Environment{Name: "production", Strategy: environment.StrategyWorkspace}
	if err := m.applyEnvironmentSelection(env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify preference was saved
	pref, err := environment.LoadPreference(tmpDir)
	if err != nil {
		t.Fatalf("failed to load preference: %v", err)
	}
	if pref.Strategy != environment.StrategyWorkspace {
		t.Fatalf("expected strategy workspace, got %s", pref.Strategy)
	}
	if pref.Environment != "production" {
		t.Fatalf("expected environment production, got %s", pref.Environment)
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
			Workspaces: []string{consts.EnvDev},
		},
	}
	newEnvironmentDetector = func(_ string) (environmentDetector, error) {
		return detector, nil
	}
	manager := &fakeWorkspaceManager{current: consts.EnvDev}
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
	if typed.Current != consts.EnvDev {
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
	m.envCurrent = consts.EnvDev
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
	if entries[0].Environment != consts.EnvDev || entries[0].Status != history.StatusFailed {
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
	if m.toast == nil || !m.toast.IsVisible() {
		t.Fatalf("expected error toast to be visible")
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
	if m.toast != nil {
		m.toast.ShowInfo("hi")
	}

	m.Update(ClearToastMsg{})
	if m.toast != nil && m.toast.IsVisible() {
		t.Fatalf("expected toast to be cleared")
	}
}

func TestViewModalStates(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.width = 80
	m.height = 24
	m.updateLayout() // Update layout to set overlay component sizes

	m.modalState = ModalSettings
	m.updateSettingsModalContent() // Populate settings modal content
	if out := m.View(); !strings.Contains(out, "Settings") {
		t.Fatalf("expected settings view")
	}

	m.modalState = ModalHelp
	m.updateHelpModalContent() // Populate help modal content
	if out := m.View(); !strings.Contains(out, "Keybinds") {
		t.Fatalf("expected help view")
	}
}

func TestHandleExecutionKeyPlanConfirm(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.modalState = ModalConfirmApply

	// Test 'n' closes the modal
	handled, _ := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !handled || m.modalState != ModalNone {
		t.Fatalf("expected confirm modal to close on 'n'")
	}

	// Test ctrl+c cancels and closes modal
	m.modalState = ModalConfirmApply
	called := false
	m.cancelFunc = func() {
		called = true
	}
	handled, _ = m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !handled || !called || m.modalState != ModalNone {
		t.Fatalf("expected cancel execution and modal close")
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

func TestHandleExecutionKeyHistoryDetailExit(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.execView = viewMain
	// History detail is now shown in mainArea [0]
	m.mainArea = NewMainArea(m.styles, nil, nil, nil)
	m.mainArea.EnterHistoryDetail()

	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled || m.mainArea.GetMode() != ModeDiff {
		t.Fatalf("expected history detail to exit and return to previous mode")
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
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected *Model type")
	}
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
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected *Model type")
	}
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
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected *Model type")
	}
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
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.diagnosticsPanel.SetSize(40, 5)
	m.diagnosticsPanel.SetParsedText("parsed")
	// History view is now embedded in mainArea
	m.mainArea = NewMainArea(m.styles, nil, nil, nil)
	m.mainArea.SetSize(80, 20)

	m.execView = viewPlanConfirm
	if out := m.View(); out == "" {
		t.Fatalf("expected plan confirm view")
	}

	m.execView = viewApplyOutput
	if out := m.View(); out == "" {
		t.Fatalf("expected apply output view")
	}

	m.execView = viewDiagnostics
	if out := m.View(); out == "" {
		t.Fatalf("expected diagnostics view")
	}

	// History detail is now shown in mainArea [0], not as a full-screen view
	m.execView = viewMain
	m.mainArea.EnterHistoryDetail()
	m.mainArea.SetHistoryContent("Test", "History content")
	if out := m.View(); out == "" {
		t.Fatalf("expected main view with history detail")
	}
}

func TestUpdatePlanOutputAppendsLine(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
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
	if m.toast == nil || !m.toast.IsVisible() {
		t.Fatalf("expected history error toast to be visible")
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

func TestUpdateEnvironmentDetectedSuccess(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envWorkDir = t.TempDir()

	result := environment.DetectionResult{
		BaseDir:    m.envWorkDir,
		Strategy:   environment.StrategyWorkspace,
		Workspaces: []string{consts.EnvDev},
	}
	result.Environments = environment.BuildEnvironments(result, "")
	m.Update(EnvironmentDetectedMsg{Result: result, Current: consts.EnvDev})
	if m.envCurrent != consts.EnvDev {
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
	m.execView = viewMain
	// History detail is now shown in mainArea [0]
	m.mainArea = NewMainArea(m.styles, nil, nil, nil)
	m.mainArea.EnterHistoryDetail()

	// Pressing q in mainArea history detail mode should still allow normal quit
	// (q key handling is in the main view default case, not specific to history detail)
	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	// The 'q' key in main view is handled elsewhere (not in handleExecutionKey)
	// so this test now expects it NOT to be handled here
	if handled {
		t.Fatalf("q key should not be handled by handleExecutionKey in main view")
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

func TestStreamCmdsHandleNilChannels(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	if msg := m.streamPlanOutputCmd()(); msg != nil {
		t.Fatalf("expected nil plan output message")
	}
	if msg := m.streamApplyOutputCmd()(); msg != nil {
		t.Fatalf("expected nil apply output message")
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

func TestHandleExecutionKeyPlanConfirmQuit(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.modalState = ModalConfirmApply

	handled, _ := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled || !m.quitting {
		t.Fatalf("expected quitting in confirm modal")
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

func TestToggleHelpModal(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.width = 80
	m.height = 24

	// Toggle help modal on
	m.toggleHelpModal()
	if m.modalState != ModalHelp {
		t.Errorf("expected modal state ModalHelp, got %v", m.modalState)
	}

	// Toggle help modal off
	m.toggleHelpModal()
	if m.modalState != ModalNone {
		t.Errorf("expected modal state ModalNone, got %v", m.modalState)
	}
}

func TestToggleSettingsModal(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.width = 80
	m.height = 24

	// Toggle settings modal on
	m.toggleSettingsModal()
	if m.modalState != ModalSettings {
		t.Errorf("expected modal state ModalSettings, got %v", m.modalState)
	}

	// Toggle settings modal off
	m.toggleSettingsModal()
	if m.modalState != ModalNone {
		t.Errorf("expected modal state ModalNone, got %v", m.modalState)
	}
}

func TestSwitchResourcesTab(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.executionMode = true
	m.resourceList = components.NewResourceList(m.styles)

	// Test switching to state tab (just verify no panic)
	m.switchResourcesTab(1)
	m.switchResourcesTab(0)
}

func TestCanSwitchResourcesTab(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true

	// Should be able to switch
	if !m.canSwitchResourcesTab() {
		t.Error("expected to be able to switch tabs in execution mode")
	}

	// Read-only mode - can't switch
	m.executionMode = false
	if m.canSwitchResourcesTab() {
		t.Error("expected not to be able to switch tabs in read-only mode")
	}
}
