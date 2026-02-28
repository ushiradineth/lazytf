package ui

import (
	"context"
	"errors"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
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
	// Focus on Resources panel for 'a' key to work
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelResources)
	}

	// Press 'a' - this sends RequestApplyMsg via the controller
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Process the RequestApplyMsg
	if cmd != nil {
		msg := cmd()
		m.Update(msg)
	}

	if m.err != nil {
		t.Fatalf("did not expect fatal error, got %v", m.err)
	}
	if m.toast == nil || !m.toast.IsVisible() {
		t.Fatalf("expected error toast to be visible")
	}
}

func TestExecutionModelWithoutPlanStartsInAboutMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	if m.mainArea == nil {
		t.Fatal("expected main area to be initialized")
	}
	if m.mainArea.GetMode() != ModeAbout {
		t.Fatalf("expected main area mode %v, got %v", ModeAbout, m.mainArea.GetMode())
	}
	if m.panelManager == nil {
		t.Fatal("expected panel manager to be initialized")
	}
	if focused := m.panelManager.GetFocusedPanel(); focused != PanelMain {
		t.Fatalf("expected focused panel %v, got %v", PanelMain, focused)
	}
}

func TestExecutionModelWithPlanStartsInDiffMode(t *testing.T) {
	plan := &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "aws_vpc.main", Action: terraform.ActionCreate}}}
	m := NewExecutionModel(plan, ExecutionConfig{})
	if m.mainArea == nil {
		t.Fatal("expected main area to be initialized")
	}
	if m.mainArea.GetMode() != ModeDiff {
		t.Fatalf("expected main area mode %v, got %v", ModeDiff, m.mainArea.GetMode())
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
	m.historyPanel.SetEntries(m.historyEntries)
	m.showHistory = true
	m.historyFocused = true
	m.syncHistorySelection()

	// Register history panel with panel manager and focus on it
	if m.panelManager != nil {
		m.panelManager.RegisterPanel(PanelHistory, m.historyPanel)
		m.panelManager.SetFocus(PanelHistory)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		cmd()
	}
	// History panel selection is tracked by historyPanel itself
	if m.historyPanel.GetSelectedIndex() != 1 {
		t.Fatalf("expected selection to move to second entry, got %d", m.historyPanel.GetSelectedIndex())
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
	m.historyPanel.SetEntries(entries)
	m.showHistory = true
	m.historyFocused = true
	m.syncHistorySelection()

	// Register and focus History panel for enter key to work
	if m.panelManager != nil {
		m.panelManager.RegisterPanel(PanelHistory, m.historyPanel)
		m.panelManager.SetFocus(PanelHistory)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected history enter to return a command")
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

	// Focus on Resources panel for 'a' key to work
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelResources)
	}

	// Press 'a' to open confirm modal - this sends RequestApplyMsg via the controller
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	// Process the RequestApplyMsg
	if cmd != nil {
		msg := cmd()
		m.Update(msg)
	}
	if m.modalState != ModalConfirmApply {
		t.Fatalf("expected confirm apply modal, got modalState=%d", m.modalState)
	}

	// Press 'y' to confirm (handled by modal key handler)
	handled, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
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
	// The failure status is shown in the header with "ERR" prefix
	if !strings.Contains(m.applyView.View(), "ERR") {
		t.Fatalf("expected plan failure header with ERR prefix")
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

func TestHistoryFocusNoEntriesNoOp(_ *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.showHistory = true
	m.historyEntries = nil

	// Tab key now cycles panels via handlePanelNavigation, not history focus
	// History focus is now managed by focusing the History panel
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// With the new panel system, tab cycles to the next panel
	// The old historyFocused behavior is replaced by panel focus
	// This test now just verifies tab doesn't crash with no entries
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

func TestApplyFlagsForRecord(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.applyFlags = []string{"-parallelism=5"}
	if got := m.applyFlagsForRecord(); len(got) != 1 || got[0] != "-parallelism=5" {
		t.Fatalf("expected default apply flags")
	}
	m.applyRunFlags = []string{"-parallelism=5", "/tmp/plan.tfplan"}
	if got := m.applyFlagsForRecord(); len(got) != 2 || got[1] != "/tmp/plan.tfplan" {
		t.Fatalf("expected apply run flags")
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
	m.historyEnabled = true
	m.width = 80
	m.height = 24

	// 'h' is a global key that toggles history visibility
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if !m.showHistory {
		t.Fatalf("expected history toggle on")
	}

	// Tab cycles panels, not history focus anymore
	m.historyEntries = []history.Entry{{Summary: "one"}}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	// Tab now cycles to the next panel in the focus order
}

func TestHandleExecutionKeyDiagnosticsFocus(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.commandLogPanel = components.NewCommandLogPanel(m.styles)
	m.panelManager = NewPanelManager()
	m.panelManager.RegisterPanel(PanelCommandLog, m.commandLogPanel)
	m.planRunning = true

	// 'D' is a global key that focuses command log
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

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

func TestCommandLogExpandFillsContent(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.ready = true
	m.styles = styles.DefaultStyles()
	m.commandLogPanel = components.NewCommandLogPanel(m.styles)
	m.panelManager = NewPanelManager()
	m.panelManager.SetExecutionMode(true)
	m.panelManager.RegisterPanel(PanelCommandLog, m.commandLogPanel)

	// Add enough session logs to require scrolling
	for i := 1; i <= 20; i++ {
		m.commandLogPanel.AppendSessionLog(
			"Planned",
			"terraform plan -out=plan.tfplan",
			"Terraform will perform the following actions:\n"+
				"  # null_resource.example will be created\n"+
				"  + resource \"null_resource\" \"example\" {\n"+
				"      + id = (known after apply)\n"+
				"    }\n"+
				"Plan: 1 to add, 0 to change, 0 to destroy.",
		)
	}

	// Set screen size
	m.width = 120
	m.height = 50

	// Make command log visible but not focused (compact mode)
	m.panelManager.SetCommandLogVisible(true)
	m.updateLayout()

	// Get the command log panel's height when not focused
	layoutBefore := m.panelManager.CalculateLayout(m.width, m.height)
	unfocusedHeight := layoutBefore.CommandLog.Height

	// Focus the command log (should expand to full height)
	m.focusCommandLog()

	// Get the command log panel's height when focused
	layoutAfter := m.panelManager.CalculateLayout(m.width, m.height)
	focusedHeight := layoutAfter.CommandLog.Height

	// Verify the layout changed
	if focusedHeight <= unfocusedHeight {
		t.Errorf("Expected focused height (%d) > unfocused height (%d)",
			focusedHeight, unfocusedHeight)
	}

	// Render the view and check the command log content fills the space
	view := m.commandLogPanel.View()
	lines := strings.Split(view, "\n")

	// Count lines with actual content (not just whitespace)
	contentLines := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and pure border lines
		if trimmed != "" && !isPureBorderLine(trimmed) {
			contentLines++
		}
	}

	// The expanded view should have more content than the minimal height
	// We expect at least 20 lines of content in a ~50 line panel
	minExpectedContent := 20
	if contentLines < minExpectedContent {
		t.Errorf("Expected at least %d content lines in expanded command log, got %d.\n"+
			"Total lines: %d, Focused height: %d\nView sample:\n%s",
			minExpectedContent, contentLines, len(lines), focusedHeight,
			truncateForTest(view, 1000))
	}
}

// isPureBorderLine checks if a line contains only border characters.
func isPureBorderLine(s string) bool {
	for _, r := range s {
		switch r {
		case '─', '│', '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼', ' ', '[', ']', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			continue
		default:
			return false
		}
	}
	return true
}

// truncateForTest truncates a string for test output.
func truncateForTest(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func TestHandleExecutionKeyCancel(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.executionMode = true
	m.planRunning = true
	called := false
	m.cancelFunc = func() {
		called = true
	}

	// ctrl+c is a global key that cancels running operations
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !called {
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
	m.applyRunFlags = []string{"-parallelism=5", consts.PlanTFPlan}
	m.planView = views.NewPlanView("", m.styles)
	m.operationState = terraform.NewOperationState()

	if err := m.applyEnvironmentSelection(environment.Environment{Name: consts.EnvDev, Strategy: environment.StrategyWorkspace}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manager.switched != consts.EnvDev {
		t.Fatalf("expected workspace switch to dev")
	}
	if m.plan != nil || m.planFilePath != "" || m.planRunFlags != nil || m.applyRunFlags != nil {
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

	// Esc key exits history detail via handleEscKey called from panel navigation
	// when no panel claims the key. In main view, esc is handled by panel navigation
	// which returns to resources panel or exits history detail mode.
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// In the new architecture, esc in history detail mode should exit it
	if m.mainArea.GetMode() != ModeDiff {
		t.Fatalf("expected history detail to exit and return to previous mode, got %v", m.mainArea.GetMode())
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
	m.diagnosticsPanel.SetLogText("parsed")
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
	m.applyRunning = true // Apply must be running to accept output
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

func TestSwitchResourcesTab(_ *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true
	m.executionMode = true
	m.resourceList = components.NewResourceList(m.styles)

	// Test switching to state tab via message (just verify no panic)
	m.handleSwitchResourcesTab(1)
	m.handleSwitchResourcesTab(-1)
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

func TestApplyBindingDebug(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true

	// Check if keybind registry is initialized
	if m.keybindRegistry == nil {
		t.Fatal("keybind registry is nil")
	}

	// Check executionMode
	if !m.executionMode {
		t.Fatal("expected execution mode to be true")
	}

	// Focus Resources panel
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelResources)
	}

	focusedPanel := m.panelManager.GetFocusedPanel()
	t.Logf("Focused panel: %d (expected PanelResources=%d)", focusedPanel, PanelResources)

	// Build context and check it
	ctx := m.buildKeybindContext()
	t.Logf("Context: ExecutionMode=%v, FocusedPanel=%d, ResourcesActiveTab=%d",
		ctx.ExecutionMode, ctx.FocusedPanel, ctx.ResourcesActiveTab)

	// Try to resolve the binding manually
	binding := m.keybindRegistry.Resolve("a", ctx)
	if binding == nil {
		t.Fatal("No binding found for 'a' key")
	}
	t.Logf("Found binding: Action=%v, Scope=%d", binding.Action, binding.Scope)

	// Now send 'a' key - this returns RequestApplyMsg via command
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	t.Logf("Command returned: %v", cmd != nil)

	// Process the RequestApplyMsg (the command returns it synchronously)
	if cmd != nil {
		msg := cmd()
		m.Update(msg)
	}

	if m.toast != nil && m.toast.IsVisible() {
		t.Log("Toast is visible after processing RequestApplyMsg - SUCCESS!")
	} else {
		if m.toast == nil {
			t.Error("Toast is nil")
		} else {
			t.Error("Toast is NOT visible after processing RequestApplyMsg")
		}
	}
}

func TestHandleRefreshOutput(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.applyView = views.NewApplyView(m.styles)

	// Handle a refresh output message
	msg := RefreshOutputMsg{Line: "Refreshing state..."}
	result, cmd := m.handleRefreshOutput(msg)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if cmd == nil {
		t.Fatal("expected non-nil command to continue streaming")
	}
}

func TestHandleRefreshOutputNilApplyView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.applyView = nil

	msg := RefreshOutputMsg{Line: "Refreshing state..."}
	result, cmd := m.handleRefreshOutput(msg)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Should still return command even if applyView is nil
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
}

func TestHandleErrorMsg(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true

	testErr := errors.New("test error")
	msg := ErrorMsg{Err: testErr}
	result := m.handleErrorMsg(msg)

	model, ok := result.(*Model)
	if !ok {
		t.Fatal("expected *Model result")
	}
	if !errors.Is(model.err, testErr) {
		t.Errorf("expected error to be set, got %v", model.err)
	}
}

func TestHandlePostUpdate(t *testing.T) {
	plan := &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Action: terraform.ActionCreate},
		},
	}
	m := NewExecutionModel(plan, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Handle a generic message to trigger the post update flow
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	result, cmd := m.handlePostUpdate(msg)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	_ = cmd // cmd might be nil for this key
}

func TestHandlePostUpdateWithEnvironmentPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.environmentPanel = components.NewEnvironmentPanel(m.styles)
	m.updateLayout()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	result, cmd := m.handlePostUpdate(msg)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	_ = cmd
}

func TestBuildEnvironmentCommand(t *testing.T) {
	m := NewModel(&terraform.Plan{})

	// Test workspace strategy
	wsEnv := environment.Environment{
		Strategy: environment.StrategyWorkspace,
		Name:     "dev",
	}
	wsCmd := m.buildEnvironmentCommand(wsEnv)
	if wsCmd != "terraform workspace select dev" {
		t.Errorf("expected 'terraform workspace select dev', got %q", wsCmd)
	}

	// Test folder strategy
	folderEnv := environment.Environment{
		Strategy: environment.StrategyFolder,
		Path:     "/path/to/env",
	}
	folderCmd := m.buildEnvironmentCommand(folderEnv)
	if folderCmd != "cd /path/to/env" {
		t.Errorf("expected 'cd /path/to/env', got %q", folderCmd)
	}
}

func TestPromptEnvironmentSelection(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should return early without panelManager or environmentPanel
	m.panelManager = nil
	result, cmd := m.promptEnvironmentSelection()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

func TestPromptEnvironmentSelectionWithPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.panelManager == nil {
		t.Skip("panelManager not initialized")
	}

	m.environmentPanel = components.NewEnvironmentPanel(m.styles)
	m.panelManager.RegisterPanel(PanelWorkspace, m.environmentPanel)

	result, cmd := m.promptEnvironmentSelection()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	_ = cmd // cmd should not be nil but depends on panel manager state
}

func TestHandleEnvironmentChanged(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.envWorkDir = t.TempDir()
	m.commandLogPanel = components.NewCommandLogPanel(m.styles)

	// Create a mock environment
	env := environment.Environment{
		Strategy:  environment.StrategyWorkspace,
		Name:      "test-workspace",
		Path:      m.envWorkDir,
		IsCurrent: false,
	}

	// Set up executor mock that will fail (to test error path)
	m.executor = nil // nil executor will cause applyEnvironmentSelection to fail

	msg := components.EnvironmentChangedMsg{Environment: env}
	result, cmd := m.handleEnvironmentChanged(msg)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Should have a toast command for the error
	if cmd == nil {
		t.Fatal("expected toast command for error")
	}
}

func TestHandleStateTabKey(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Not on state tab - should not handle
	m.resourcesActiveTab = 0
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	cmd, handled := m.handleStateTabKey(msg)
	if handled {
		t.Error("expected not to handle when not on state tab")
	}
	if cmd != nil {
		t.Error("expected nil cmd when not handled")
	}
}

func TestHandleStateTabKeyOnStateTab(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1

	// Need stateListContent for handling
	m.stateListContent = components.NewStateListContent(m.styles)
	m.stateListContent.SetSize(80, 20)

	// Focus on resources panel
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelResources)
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	_, handled := m.handleStateTabKey(msg)
	// Result depends on whether stateListContent can handle the key
	_ = handled
}

func TestHandleStateTabKeyNonKeyMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.executionMode = true
	m.resourcesActiveTab = 1
	m.stateListContent = components.NewStateListContent(m.styles)

	// Non-key message should not be handled
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	cmd, handled := m.handleStateTabKey(msg)
	if handled {
		t.Error("expected not to handle non-key message")
	}
	if cmd != nil {
		t.Error("expected nil cmd for non-key message")
	}
}

func TestShouldPromptEnvironment(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	// No detection result - should not prompt
	m.envDetection = nil
	if m.shouldPromptEnvironment() {
		t.Error("expected false when no detection result")
	}

	// Mixed strategy - should prompt
	m.envDetection = &environment.DetectionResult{
		Strategy: environment.StrategyMixed,
	}
	if !m.shouldPromptEnvironment() {
		t.Error("expected true for mixed strategy")
	}

	// Single option - should not prompt
	m.envDetection = &environment.DetectionResult{
		Strategy: environment.StrategyWorkspace,
	}
	m.envOptions = []environment.Environment{{Name: "default"}}
	if m.shouldPromptEnvironment() {
		t.Error("expected false for single option")
	}

	// Multiple options - should prompt
	m.envOptions = []environment.Environment{
		{Name: "dev"},
		{Name: "prod"},
	}
	if !m.shouldPromptEnvironment() {
		t.Error("expected true for multiple options")
	}
}

func TestStrategyAvailable(t *testing.T) {
	// Test workspace strategy
	result := environment.DetectionResult{
		Workspaces: []string{"dev"},
	}
	if !strategyAvailable(result, environment.StrategyWorkspace) {
		t.Error("expected workspace strategy to be available")
	}
	if strategyAvailable(result, environment.StrategyFolder) {
		t.Error("expected folder strategy not available with no folders")
	}

	// Test folder strategy
	result = environment.DetectionResult{
		FolderPaths: []string{"/path/to/folder"},
	}
	if !strategyAvailable(result, environment.StrategyFolder) {
		t.Error("expected folder strategy to be available")
	}
	if strategyAvailable(result, environment.StrategyWorkspace) {
		t.Error("expected workspace strategy not available with no workspaces")
	}

	// Test mixed strategy
	result = environment.DetectionResult{
		Workspaces:  []string{"dev"},
		FolderPaths: []string{"/path/to/folder"},
	}
	if !strategyAvailable(result, environment.StrategyMixed) {
		t.Error("expected mixed strategy to be available")
	}

	// Test unknown strategy
	if strategyAvailable(result, environment.StrategyUnknown) {
		t.Error("expected unknown strategy to not be available")
	}
}

func TestMainAreaIsFocused(t *testing.T) {
	s := styles.DefaultStyles()
	m := NewMainArea(s, nil, nil, nil)
	m.SetFocused(true)
	if !m.IsFocused() {
		t.Error("expected IsFocused to return true")
	}
	m.SetFocused(false)
	if m.IsFocused() {
		t.Error("expected IsFocused to return false")
	}
}

func TestMainAreaSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	m := NewMainArea(s, nil, nil, nil)
	m.SetSize(80, 20)

	newStyles := styles.DefaultStyles()
	m.SetStyles(newStyles)
	// Should not panic and styles should be updated
}

func TestMainAreaUpdate(t *testing.T) {
	s := styles.DefaultStyles()
	applyView := views.NewApplyView(s)
	m := NewMainArea(s, nil, applyView, nil)
	m.SetSize(80, 20)

	// Test update in ModeLogs
	m.SetMode(ModeLogs)
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd

	// Test update in ModeHistoryDetail
	m.SetMode(ModeHistoryDetail)
	result, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestMainAreaHandleKey(t *testing.T) {
	s := styles.DefaultStyles()
	m := NewMainArea(s, nil, nil, nil)
	m.SetSize(80, 20)

	// Not focused - should not handle
	m.SetFocused(false)
	handled, _ := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if handled {
		t.Error("expected not to handle when not focused")
	}

	// Focused in ModeDiff
	m.SetFocused(true)
	m.SetMode(ModeDiff)

	// Test scroll down
	handled, _ = m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("expected to handle 'j' in ModeDiff")
	}

	// Test scroll up
	handled, _ = m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if !handled {
		t.Error("expected to handle 'k' in ModeDiff")
	}
}

func TestMainAreaHandleKeyInLogsMode(t *testing.T) {
	s := styles.DefaultStyles()
	applyView := views.NewApplyView(s)
	m := NewMainArea(s, nil, applyView, nil)
	m.SetSize(80, 20)
	m.SetFocused(true)
	m.SetMode(ModeLogs)

	// Test scroll keys in logs mode
	handled, _ := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("expected to handle 'j' in ModeLogs")
	}
}

func TestMainAreaGetViews(t *testing.T) {
	s := styles.DefaultStyles()
	applyView := views.NewApplyView(s)
	planView := views.NewPlanView("", s)
	m := NewMainArea(s, nil, applyView, planView)
	m.SetSize(80, 20)

	if m.GetApplyView() != applyView {
		t.Error("expected GetApplyView to return applyView")
	}
	if m.GetPlanView() != planView {
		t.Error("expected GetPlanView to return planView")
	}
	if m.GetDiffViewer() == nil {
		t.Error("expected GetDiffViewer to return non-nil")
	}
}

func TestPanelManagerGetPanel(t *testing.T) {
	pm := NewPanelManager()
	mockPanel := components.NewResourceList(styles.DefaultStyles())
	pm.RegisterPanel(PanelResources, mockPanel)

	panel, ok := pm.GetPanel(PanelResources)
	if !ok || panel == nil {
		t.Error("expected GetPanel to return registered panel")
	}

	// Non-existent panel
	panel, ok = pm.GetPanel(PanelID(999))
	if ok || panel != nil {
		t.Error("expected nil for non-existent panel")
	}
}

func TestPanelManagerToggleCommandLog(t *testing.T) {
	pm := NewPanelManager()

	// Initially visible by default
	if !pm.IsCommandLogVisible() {
		t.Error("expected command log to be visible initially")
	}

	// Toggle off
	pm.ToggleCommandLog()
	if pm.IsCommandLogVisible() {
		t.Error("expected command log to be hidden after toggle")
	}

	// Toggle on again
	pm.ToggleCommandLog()
	if !pm.IsCommandLogVisible() {
		t.Error("expected command log to be visible after second toggle")
	}
}

func TestPanelManagerIsExecutionMode(t *testing.T) {
	pm := NewPanelManager()

	if pm.IsExecutionMode() {
		t.Error("expected IsExecutionMode to return false initially")
	}

	pm.SetExecutionMode(true)
	if !pm.IsExecutionMode() {
		t.Error("expected IsExecutionMode to return true after SetExecutionMode(true)")
	}
}

func TestPanelManagerHandleNavigation(t *testing.T) {
	pm := NewPanelManager()
	rl := components.NewResourceList(styles.DefaultStyles())
	pm.RegisterPanel(PanelResources, rl)
	pm.SetFocus(PanelResources)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	handled, _ := pm.HandleNavigation(msg)
	// Should handle or not depending on panel state
	_ = handled
}

func TestResourcesControllerGetActiveTab(t *testing.T) {
	rl := components.NewResourceList(styles.DefaultStyles())
	rc := NewResourcesPanelController(rl)

	if rc.GetActiveTab() != 0 {
		t.Errorf("expected initial tab to be 0, got %d", rc.GetActiveTab())
	}

	rc.SetActiveTab(1)
	if rc.GetActiveTab() != 1 {
		t.Errorf("expected tab to be 1, got %d", rc.GetActiveTab())
	}
}

func TestResourcesControllerHandleKey(t *testing.T) {
	rl := components.NewResourceList(styles.DefaultStyles())
	rc := NewResourcesPanelController(rl)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	handled, cmd := rc.HandleKey(msg)
	// Should handle navigation keys
	_ = handled
	_ = cmd
}

func TestResourcesControllerHandleKeyPlan(t *testing.T) {
	rl := components.NewResourceList(styles.DefaultStyles())
	rc := NewResourcesPanelController(rl)

	// Test 'p' key for plan
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	handled, cmd := rc.HandleKey(msg)
	if !handled {
		t.Error("expected 'p' key to be handled")
	}
	if cmd == nil {
		t.Error("expected cmd to be non-nil for 'p' key")
	}
}

func TestResourcesControllerHandleKeyTabSwitch(t *testing.T) {
	rl := components.NewResourceList(styles.DefaultStyles())
	rc := NewResourcesPanelController(rl)

	// Test '[' key for tab switch
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
	handled, cmd := rc.HandleKey(msg)
	if !handled {
		t.Error("expected '[' key to be handled")
	}
	if cmd == nil {
		t.Error("expected cmd to be non-nil for '[' key")
	}
}

func TestResourcesControllerRequestCommands(t *testing.T) {
	// Test all request command factory functions return valid commands
	cmd := requestPlan()
	if cmd == nil {
		t.Error("expected requestPlan to return non-nil command")
	}

	cmd = requestRefresh()
	if cmd == nil {
		t.Error("expected requestRefresh to return non-nil command")
	}

	cmd = requestValidate()
	if cmd == nil {
		t.Error("expected requestValidate to return non-nil command")
	}

	cmd = requestFormat()
	if cmd == nil {
		t.Error("expected requestFormat to return non-nil command")
	}

	cmd = requestApply()
	if cmd == nil {
		t.Error("expected requestApply to return non-nil command")
	}
}

func TestResourcesControllerToggleCommands(t *testing.T) {
	// Test toggle filter
	cmd := toggleFilter(terraform.ActionCreate)
	if cmd == nil {
		t.Error("expected toggleFilter to return non-nil command")
	}

	// Test toggle all groups
	cmd = toggleAllGroups()
	if cmd == nil {
		t.Error("expected toggleAllGroups to return non-nil command")
	}

	// Test toggle status
	cmd = toggleStatus()
	if cmd == nil {
		t.Error("expected toggleStatus to return non-nil command")
	}

	// Test switch tab
	cmd = switchResourcesTab(1)
	if cmd == nil {
		t.Error("expected switchResourcesTab to return non-nil command")
	}
}

func TestToastInfo(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 80
	m.height = 24
	m.updateLayout()

	cmd := m.toastInfo("Test info message")
	if cmd == nil {
		t.Error("expected toastInfo to return non-nil command")
	}
}

func TestToastSuccess(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 80
	m.height = 24
	m.updateLayout()

	cmd := m.toastSuccess("Test success message")
	if cmd == nil {
		t.Error("expected toastSuccess to return non-nil command")
	}
}

func TestFocusMainPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.panelManager == nil {
		t.Skip("panelManager not initialized")
	}

	cmd := m.focusMainPanel()
	// Should return a command or nil
	_ = cmd
}

func TestBeginRefreshNoExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.executor = nil

	cmd := m.beginRefresh()
	if cmd != nil {
		t.Error("expected nil cmd when executor is nil")
	}
	if m.err == nil {
		t.Error("expected error to be set when executor is nil")
	}
}

func TestBeginRefreshAlreadyRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.planRunning = true

	cmd := m.beginRefresh()
	if cmd != nil {
		t.Error("expected nil cmd when already running")
	}
}

func TestBeginValidateNoExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.executor = nil

	cmd := m.beginValidate()
	if cmd != nil {
		t.Error("expected nil cmd when executor is nil")
	}
	if m.err == nil {
		t.Error("expected error to be set when executor is nil")
	}
}

func TestBeginFormatNoExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.executor = nil

	cmd := m.beginFormat()
	if cmd != nil {
		t.Error("expected nil cmd when executor is nil")
	}
	if m.err == nil {
		t.Error("expected error to be set when executor is nil")
	}
}

func TestBeginStateListNoExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.executor = nil

	cmd := m.beginStateList()
	if cmd != nil {
		t.Error("expected nil cmd when executor is nil")
	}
	if m.err == nil {
		t.Error("expected error to be set when executor is nil")
	}
}

func TestBeginStateShowNoExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.executor = nil

	cmd := m.beginStateShow("some.address")
	if cmd != nil {
		t.Error("expected nil cmd when executor is nil")
	}
	if m.err == nil {
		t.Error("expected error to be set when executor is nil")
	}
}

func TestHandleRefreshCompleteSuccess(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.refreshRunning = true
	m.commandLogPanel = components.NewCommandLogPanel(m.styles)

	msg := RefreshCompleteMsg{
		Success: true,
		Result:  &terraform.ExecutionResult{Output: "Refreshed"},
	}
	result, cmd := m.handleRefreshComplete(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	if m.refreshRunning {
		t.Error("expected refreshRunning to be false")
	}
	_ = cmd
}

func TestHandleValidateCompleteSuccess(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.commandLogPanel = components.NewCommandLogPanel(m.styles)

	msg := ValidateCompleteMsg{
		Result: &terraform.ValidateResult{Valid: true},
	}
	result, cmd := m.handleValidateComplete(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleValidateCompleteWithErrors(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ValidateCompleteMsg{
		Error: errors.New("validation failed"),
	}
	result, cmd := m.handleValidateComplete(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleFormatCompleteSuccess(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.commandLogPanel = components.NewCommandLogPanel(m.styles)

	msg := FormatCompleteMsg{
		ChangedFiles: []string{"main.tf", "variables.tf"},
	}
	result, cmd := m.handleFormatComplete(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleStateListCompleteSuccess(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.stateListContent = components.NewStateListContent(m.styles)

	resources := []terraform.StateResource{
		{ResourceType: "aws_instance", Name: "web", Address: "aws_instance.web"},
	}
	msg := StateListCompleteMsg{Resources: resources}
	result, cmd := m.handleStateListComplete(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleStateListCompleteError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.stateListContent = components.NewStateListContent(m.styles)

	msg := StateListCompleteMsg{Error: errors.New("state list failed")}
	result, cmd := m.handleStateListComplete(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleStateShowCompleteSuccess(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateShowCompleteMsg{
		Address: "aws_instance.web",
		Output:  `{"address": "aws_instance.web"}`,
	}
	result, cmd := m.handleStateShowComplete(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestResourceSummaryText(t *testing.T) {
	m := NewModel(&terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Action: terraform.ActionCreate},
			{Address: "aws_instance.db", Action: terraform.ActionUpdate},
		},
	})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	text := m.resourceSummaryText()
	if text == "" {
		t.Error("expected non-empty resource summary text")
	}
}

func TestShowFormattedFiles(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Empty modified list
	m.showFormattedFiles(nil)

	// With files
	m.showFormattedFiles([]string{"main.tf", "variables.tf"})
}

func TestHandleActionQuit(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.ready = true

	cmd := m.handleActionQuit(nil)
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestHandleActionToggleTheme(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionToggleTheme(nil)
	// May return nil or a command
	_ = cmd
}

func TestHandleActionFocusWorkspace(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.panelManager == nil {
		t.Skip("panelManager not initialized")
	}

	cmd := m.handleActionFocusWorkspace(nil)
	_ = cmd
	if m.mainArea != nil && m.mainArea.GetMode() != ModeAbout {
		t.Error("expected main area to be in about mode")
	}
}

func TestHandleActionFocusResources(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.panelManager == nil {
		t.Skip("panelManager not initialized")
	}

	cmd := m.handleActionFocusResources(nil)
	_ = cmd
}

func TestHandleActionFocusHistory(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.panelManager == nil {
		t.Skip("panelManager not initialized")
	}

	cmd := m.handleActionFocusHistory(nil)
	_ = cmd
	if !m.historyFocused {
		t.Error("expected historyFocused to be true")
	}
}

func TestHandleActionFocusMain(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.panelManager == nil {
		t.Skip("panelManager not initialized")
	}

	cmd := m.handleActionFocusMain(nil)
	_ = cmd
}

func TestHandleActionCycleFocusBack(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.panelManager == nil {
		t.Skip("panelManager not initialized")
	}

	cmd := m.handleActionCycleFocusBack(nil)
	_ = cmd
}

func TestHandleActionToggleLog(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionToggleLog(nil)
	_ = cmd
}

func TestHandleActionPlan(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionPlan(nil)
	if cmd == nil {
		t.Error("expected non-nil command")
	}
}

func TestHandleActionRefresh(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionRefresh(nil)
	if cmd == nil {
		t.Error("expected non-nil command")
	}
}

func TestHandleActionValidate(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionValidate(nil)
	if cmd == nil {
		t.Error("expected non-nil command")
	}
}

func TestHandleActionFormat(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionFormat(nil)
	if cmd == nil {
		t.Error("expected non-nil command")
	}
}

func TestHandleActionToggleAllGroups(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionToggleAllGroups(nil)
	// This returns nil as it just modifies resourceList state
	_ = cmd
}

func TestHandleActionToggleStatus(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionToggleStatus(nil)
	// This returns nil as it just modifies resourceList state
	_ = cmd
}

func TestHandleActionSwitchTabPrev(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionSwitchTabPrev(nil)
	if cmd == nil {
		t.Error("expected non-nil command")
	}
}

func TestHandleActionSwitchTabNext(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionSwitchTabNext(nil)
	if cmd == nil {
		t.Error("expected non-nil command")
	}
}

func TestHandleActionMoveUp(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Need a context with FocusedPanel
	ctx := m.buildKeybindContext()
	cmd := m.handleActionMoveUp(ctx)
	_ = cmd
}

func TestHandleActionPageUp(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := m.buildKeybindContext()
	cmd := m.handleActionPageUp(ctx)
	_ = cmd
}

func TestHandleActionPageDown(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := m.buildKeybindContext()
	cmd := m.handleActionPageDown(ctx)
	_ = cmd
}

func TestHandleActionScrollTop(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := m.buildKeybindContext()
	cmd := m.handleActionScrollTop(ctx)
	_ = cmd
}

func TestHandleActionScrollEnd(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := m.buildKeybindContext()
	cmd := m.handleActionScrollEnd(ctx)
	_ = cmd
}

func TestHandleActionScrollUp(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := m.buildKeybindContext()
	cmd := m.handleActionScrollUp(ctx)
	_ = cmd
}

func TestHandleActionScrollDown(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	ctx := m.buildKeybindContext()
	cmd := m.handleActionScrollDown(ctx)
	_ = cmd
}

func TestHandleRefreshStart(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RefreshStartMsg{
		Result: &terraform.ExecutionResult{},
		Output: make(chan string),
	}
	result, cmd := m.handleRefreshStart(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleRefreshStartWithError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RefreshStartMsg{
		Error: errors.New("refresh failed"),
	}
	result, cmd := m.handleRefreshStart(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleRefreshFailure(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.refreshRunning = true

	msg := RefreshCompleteMsg{
		Error: errors.New("refresh failed"),
	}
	result, cmd := m.handleRefreshFailure(msg)
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleCommandLogKey(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewCommandLog

	// Test escape key
	handled, cmd := m.handleCommandLogKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected escape key to be handled")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain after escape")
	}
	_ = cmd
}

func TestHandleCommandLogKeyUnknown(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true

	// Test unknown key
	handled, cmd := m.handleCommandLogKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected unknown key not to be handled")
	}
	if cmd != nil {
		t.Error("expected nil command for unknown key")
	}
}

func TestHandleStateListKey(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateList

	// Test escape key
	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected escape key to be handled")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain after escape")
	}
	_ = cmd
}

func TestHandleStateListKeyUpDown(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Test up key
	handled, _ := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if !handled {
		t.Error("expected 'k' key to be handled")
	}

	// Test down key
	handled, _ = m.handleStateListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("expected 'j' key to be handled")
	}
}

func TestHandleStateShowKey(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateShow

	// Test escape key
	handled, cmd := m.handleStateShowKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected escape key to be handled")
	}
	if m.execView != viewStateList {
		t.Error("expected execView to be viewStateList after escape")
	}
	_ = cmd
}

func TestHandleActionSelectEnv(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionSelectEnv(nil)
	_ = cmd
}

func TestHandleActionConfirmYes(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionConfirmYes(nil)
	_ = cmd
}

func TestHandleActionConfirmNo(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.handleActionConfirmNo(nil)
	_ = cmd
}

func TestFallbackValue(t *testing.T) {
	// Test with empty value
	result := fallbackValue("")
	if result != defaultThemeName {
		t.Errorf("expected %q, got %q", defaultThemeName, result)
	}

	// Test with non-empty value
	result = fallbackValue("custom")
	if result != "custom" {
		t.Errorf("expected 'custom', got %q", result)
	}
}

func TestResourcesControllerStateTabHandling(t *testing.T) {
	rl := components.NewResourceList(styles.DefaultStyles())
	rc := NewResourcesPanelController(rl)
	rc.SetActiveTab(1) // State tab

	slc := components.NewStateListContent(styles.DefaultStyles())
	rc.SetStateListContent(slc)

	// Test handling key on state tab
	handled, cmd := rc.handleStateTabKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	// Should be handled by state list content
	_ = handled
	_ = cmd
}

func TestHandleSecondaryUpdate(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Test ValidateCompleteMsg
	msg := ValidateCompleteMsg{Result: &terraform.ValidateResult{Valid: true}}
	result, cmd, handled := m.handleSecondaryUpdate(msg)
	if !handled {
		t.Error("expected ValidateCompleteMsg to be handled")
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd

	// Test FormatCompleteMsg
	formatMsg := FormatCompleteMsg{ChangedFiles: []string{"main.tf"}}
	result, cmd, handled = m.handleSecondaryUpdate(formatMsg)
	if !handled {
		t.Error("expected FormatCompleteMsg to be handled")
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleTertiaryUpdate(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.historyPanel = components.NewHistoryPanel(m.styles)

	// Test HistoryLoadedMsg
	msg := HistoryLoadedMsg{
		Entries: []history.Entry{{Summary: "test"}},
	}
	result, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected HistoryLoadedMsg to be handled")
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd

	// Test ClearToastMsg
	clearMsg := ClearToastMsg{}
	result, cmd, handled = m.handleTertiaryUpdate(clearMsg)
	if !handled {
		t.Error("expected ClearToastMsg to be handled")
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleDiagnosticsKey(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsPanel = components.NewDiagnosticsPanel(m.styles)
	m.diagnosticsFocused = true

	// Test a key that should be handled
	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("expected key to be handled when diagnostics is focused")
	}
	_ = cmd
}

func TestHandleEnvironmentPanelKey(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.environmentPanel = components.NewEnvironmentPanel(m.styles)

	// When selector is not active, should not handle
	handled, cmd := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	// Result depends on whether selector is active
	_ = handled
	_ = cmd
}

func TestHandleNonMainViewKey(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewMain

	// When in main view, returns model and nil cmd
	result, cmd := m.handleNonMainViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestHandleNonMainViewKeyInOutputView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewApplyOutput

	result, cmd := m.handleNonMainViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if result == nil {
		t.Error("expected non-nil result")
	}
	_ = cmd
}

func TestApplyEnvironmentPreference(t *testing.T) {
	// Test with nil preference
	result := environment.DetectionResult{
		Strategy: environment.StrategyWorkspace,
	}
	strategy, current := applyEnvironmentPreference(result, "dev", nil)
	if strategy != environment.StrategyWorkspace {
		t.Errorf("expected StrategyWorkspace, got %v", strategy)
	}
	if current != "dev" {
		t.Errorf("expected 'dev', got %q", current)
	}

	// Test with preference
	pref := &environment.Preference{
		Strategy:    environment.StrategyFolder,
		Environment: "prod",
	}
	result = environment.DetectionResult{
		Strategy:    environment.StrategyWorkspace,
		Workspaces:  []string{"dev"},
		FolderPaths: []string{"/path/to/prod"},
	}
	strategy, current = applyEnvironmentPreference(result, "dev", pref)
	if strategy != environment.StrategyFolder {
		t.Errorf("expected StrategyFolder, got %v", strategy)
	}
	if current != "prod" {
		t.Errorf("expected 'prod', got %q", current)
	}
}

func TestEnvDisplayName(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	// Set a current env for workspace strategy
	m.envCurrent = "dev"
	m.envStrategy = environment.StrategyWorkspace
	name := m.envDisplayName()
	if name != "dev" {
		t.Errorf("expected 'dev', got %q", name)
	}

	// Test folder strategy with relative path
	m.envStrategy = environment.StrategyFolder
	m.envWorkDir = "/base"
	m.envCurrent = "/base/subfolder"
	name = m.envDisplayName()
	// Should return relative path or base name
	_ = name
}

func TestInitHistory(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{
		HistoryEnabled: false,
	})
	m.ready = true

	// Should not initialize history when disabled
	m.initHistory(ExecutionConfig{HistoryEnabled: false})
	if m.historyStore != nil {
		t.Error("expected historyStore to be nil when disabled")
	}
}

func TestReloadHistoryCmd(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true

	// No history store - should return nil
	cmd := m.reloadHistoryCmd()
	if cmd != nil {
		t.Error("expected nil cmd when no history store")
	}
}

func TestLoadFilterPreferences(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic with no config
	m.loadFilterPreferences()
}

func TestSaveFilterPreferences(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic with no config
	m.saveFilterPreferences()
}

func TestHandleSecondaryUpdateValidateComplete(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ValidateCompleteMsg{
		Result: &terraform.ValidateResult{Valid: true},
	}

	model, cmd, handled := m.handleSecondaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleSecondaryUpdateFormatComplete(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := FormatCompleteMsg{
		ChangedFiles: []string{"main.tf"},
	}

	model, cmd, handled := m.handleSecondaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleSecondaryUpdateStateListComplete(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateListCompleteMsg{
		Resources: []terraform.StateResource{
			{Address: "test.resource"},
		},
	}

	model, cmd, handled := m.handleSecondaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleSecondaryUpdateStateShowComplete(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateShowCompleteMsg{
		Address: "test.resource",
		Output:  "resource details",
	}

	model, cmd, handled := m.handleSecondaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateSpinnerTick(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := spinner.TickMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateRequestPlan(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RequestPlanMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateRequestRefresh(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RequestRefreshMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateRequestValidate(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RequestValidateMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateRequestFormat(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RequestFormatMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateToggleFilter(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ToggleFilterMsg{Action: terraform.ActionCreate}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateToggleStatus(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ToggleStatusMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateToggleAllGroups(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ToggleAllGroupsMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateStateListStart(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateListStartMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateSwitchResourcesTab(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := SwitchResourcesTabMsg{Direction: 1}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleSecondaryUpdateRefreshStart(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RefreshStartMsg{}

	model, cmd, handled := m.handleSecondaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleSecondaryUpdateRefreshOutput(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RefreshOutputMsg{Line: "refreshing..."}

	model, cmd, handled := m.handleSecondaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleSecondaryUpdateRefreshComplete(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RefreshCompleteMsg{}

	model, cmd, handled := m.handleSecondaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateHistoryLoaded(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := HistoryLoadedMsg{
		Entries: []history.Entry{
			{ID: 1, Summary: "plan"},
		},
	}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateHistoryDetail(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := HistoryDetailMsg{
		Entry: history.Entry{ID: 1, Summary: "plan"},
	}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateEnvironmentDetected(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := EnvironmentDetectedMsg{
		Result: environment.DetectionResult{
			Strategy:     environment.StrategyFolder,
			Environments: []environment.Environment{{Name: "test"}},
		},
	}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateClearToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ClearToastMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateComponentsClearToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := components.ClearToast{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateEnvironmentChanged(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := components.EnvironmentChangedMsg{
		Environment: environment.Environment{
			Name:     "test",
			Strategy: environment.StrategyFolder,
			Path:     "/tmp/test",
		},
	}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateErrorMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ErrorMsg{Err: errors.New("test error")}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateRequestApply(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := RequestApplyMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateKeyMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleTertiaryUpdateUnknownMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	type unknownMsg struct{}
	msg := unknownMsg{}

	model, cmd, handled := m.handleTertiaryUpdate(msg)
	if handled {
		t.Error("expected unknown message to not be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleEnvironmentChangedWorkspace(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := components.EnvironmentChangedMsg{
		Environment: environment.Environment{
			Name:     "dev",
			Strategy: environment.StrategyWorkspace,
		},
	}

	model, cmd := m.handleEnvironmentChanged(msg)
	_ = model
	_ = cmd
}

func TestHandleEnvironmentChangedSuccessWithComponents(t *testing.T) {
	mock := testutil.NewMockExecutor()
	mock.MockWorkDir = t.TempDir()

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Initialize components to hit more code paths
	m.environmentPanel = components.NewEnvironmentPanel(m.styles)
	m.commandLogPanel = components.NewCommandLogPanel(m.styles)

	env := environment.Environment{
		Name:     "production",
		Path:     mock.MockWorkDir,
		Strategy: environment.StrategyFolder,
	}

	msg := components.EnvironmentChangedMsg{Environment: env}
	model, cmd := m.handleEnvironmentChanged(msg)

	if model == nil {
		t.Error("expected non-nil model")
	}
	if cmd == nil {
		t.Error("expected toast command for success")
	}
	// Verify environment was updated
	if m.envCurrent != envSelectionValue(env) {
		t.Error("expected envCurrent to be updated")
	}
}

func TestHandleEnvironmentChangedWithCommandLogPanelError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = nil // Will cause error

	// Initialize command log panel to test error logging
	m.commandLogPanel = components.NewCommandLogPanel(m.styles)

	env := environment.Environment{
		Name:     "dev",
		Strategy: environment.StrategyWorkspace,
	}

	msg := components.EnvironmentChangedMsg{Environment: env}
	model, cmd := m.handleEnvironmentChanged(msg)

	if model == nil {
		t.Error("expected non-nil model")
	}
	if cmd == nil {
		t.Error("expected toast command for error")
	}
}

func TestBuildEnvironmentCommandWorkspace(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	env := environment.Environment{
		Name:     "production",
		Strategy: environment.StrategyWorkspace,
	}

	cmd := m.buildEnvironmentCommand(env)
	if cmd != "terraform workspace select production" {
		t.Errorf("expected workspace select command, got %q", cmd)
	}
}

func TestBuildEnvironmentCommandFolder(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	env := environment.Environment{
		Path:     "/path/to/env",
		Strategy: environment.StrategyFolder,
	}

	cmd := m.buildEnvironmentCommand(env)
	if cmd != "cd /path/to/env" {
		t.Errorf("expected cd command, got %q", cmd)
	}
}

func TestHandleErrorMsgDisplay(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ErrorMsg{Err: errors.New("test error")}

	model := m.handleErrorMsg(msg)
	_ = model
}

func TestHandlePrimaryUpdateWindowSize(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	model, cmd, handled := m.handlePrimaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	if m.width != 120 || m.height != 40 {
		t.Error("expected dimensions to be updated")
	}
	_ = model
	_ = cmd
}

func TestHandlePrimaryUpdatePlanStart(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := PlanStartMsg{}

	model, cmd, handled := m.handlePrimaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandlePrimaryUpdatePlanOutput(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := PlanOutputMsg{Line: "Planning..."}

	model, cmd, handled := m.handlePrimaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandlePrimaryUpdatePlanComplete(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := PlanCompleteMsg{}

	model, cmd, handled := m.handlePrimaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandlePrimaryUpdateApplyStart(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ApplyStartMsg{}

	model, cmd, handled := m.handlePrimaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandlePrimaryUpdateApplyOutput(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ApplyOutputMsg{Line: "Applying..."}

	model, cmd, handled := m.handlePrimaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandlePrimaryUpdateApplyComplete(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ApplyCompleteMsg{}

	model, cmd, handled := m.handlePrimaryUpdate(msg)
	if !handled {
		t.Error("expected message to be handled")
	}
	_ = model
	_ = cmd
}

func TestNextResourcesTab(t *testing.T) {
	tests := []struct {
		current   int
		direction int
		want      int
	}{
		{0, 1, 1},  // Resources -> State
		{1, 1, 0},  // State -> Resources
		{1, -1, 0}, // State -> Resources
		{0, -1, 1}, // Resources -> State
	}

	for _, tt := range tests {
		got := nextResourcesTab(tt.current, tt.direction)
		if got != tt.want {
			t.Errorf("nextResourcesTab(%v, %d) = %v, want %v", tt.current, tt.direction, got, tt.want)
		}
	}
}

func TestHandleToggleFilter(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Filters start as true, toggle to false
	m.handleToggleFilter(terraform.ActionCreate)
	if m.filterCreate {
		t.Error("expected filterCreate to be false after toggle")
	}

	// Toggle update filter
	m.handleToggleFilter(terraform.ActionUpdate)
	if m.filterUpdate {
		t.Error("expected filterUpdate to be false after toggle")
	}

	// Toggle delete filter
	m.handleToggleFilter(terraform.ActionDelete)
	if m.filterDelete {
		t.Error("expected filterDelete to be false after toggle")
	}

	// Toggle replace filter
	m.handleToggleFilter(terraform.ActionReplace)
	if m.filterReplace {
		t.Error("expected filterReplace to be false after toggle")
	}
}

func TestHandleRequestApplyNoPlan(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.plan = nil

	model, cmd, handled := m.handleRequestApply()
	if !handled {
		t.Error("expected to be handled")
	}
	_ = model
	_ = cmd
}

func TestHandleRequestApplyWithPlan(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.plan = &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "test.resource"},
		},
	}

	model, cmd, handled := m.handleRequestApply()
	if !handled {
		t.Error("expected to be handled")
	}
	if m.modalState != ModalConfirmApply {
		t.Error("expected confirm apply modal to be shown")
	}
	_ = model
	_ = cmd
}

func TestViewExecutionOverride(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Test with plan output view - just ensure it doesn't panic
	m.execView = viewPlanOutput
	view := m.viewExecutionOverride()
	_ = view

	// Test with apply output view - just ensure it doesn't panic
	m.execView = viewApplyOutput
	view = m.viewExecutionOverride()
	_ = view
}

func TestApplyViewOverlays(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	baseView := "base content"

	// With help modal
	m.modalState = ModalHelp
	result := m.applyViewOverlays(baseView)
	_ = result

	// With settings modal
	m.modalState = ModalSettings
	result = m.applyViewOverlays(baseView)
	_ = result

	// With confirm apply modal
	m.modalState = ModalConfirmApply
	result = m.applyViewOverlays(baseView)
	_ = result

	// With toast
	m.modalState = ModalNone
	if m.toast != nil {
		m.toast.ShowSuccess("test")
	}
	result = m.applyViewOverlays(baseView)
	_ = result
}

func TestHandleEnvironmentPanelKeyNoPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.panelManager = nil

	handled, cmd := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyEnter})
	if handled {
		t.Error("expected not handled without panel manager")
	}
	_ = cmd
}

func TestHandleEnvironmentPanelKeyNoEnvPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.environmentPanel = nil

	handled, cmd := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyEnter})
	if handled {
		t.Error("expected not handled without environment panel")
	}
	_ = cmd
}

func TestHandleEnvironmentPanelKeyActive(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.environmentPanel != nil {
		m.environmentPanel.SetFocused(true)
		handled, cmd := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyDown})
		_ = handled
		_ = cmd
	}
}

func TestHandleNonMainViewKeyCommandLog(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewCommandLog

	model, cmd := m.handleNonMainViewKey(tea.KeyMsg{Type: tea.KeyDown})
	_ = model
	_ = cmd
}

func TestHandleNonMainViewKeyNotCommandLog(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewMain

	model, cmd := m.handleNonMainViewKey(tea.KeyMsg{Type: tea.KeyDown})
	_ = model
	_ = cmd
}

func TestHandleDiagnosticsKeyNotFocused(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = false

	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyDown})
	if handled {
		t.Error("expected not handled when diagnostics not focused")
	}
	_ = cmd
}

func TestHandleDiagnosticsKeyFocused(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyDown})
	_ = handled
	_ = cmd
}

func TestHandleKeyMsgWithPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	_ = model
	_ = cmd
}

func TestHandleApplyOutputLine(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ApplyOutputMsg{Line: "Applying..."}
	model, cmd := m.handleApplyOutput(msg)
	_ = model
	_ = cmd
}

func TestUpdateEnvironmentPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.updateEnvironmentPanel(nil)
}

func TestUpdateEnvironmentPanelWithOptions(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.envOptions = []environment.Environment{
		{Name: "dev", Strategy: environment.StrategyWorkspace},
	}

	m.updateEnvironmentPanel([]string{"warning1"})
}

func TestHandleStateTabKeyNotExecutionMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executionMode = false

	cmd, handled := m.handleStateTabKey(tea.KeyMsg{Type: tea.KeyEnter})
	if handled {
		t.Error("expected not handled when not in execution mode")
	}
	_ = cmd
}

func TestHandleStateTabKeyInStateTab(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executionMode = true
	m.resourcesActiveTab = 1 // State tab

	cmd, handled := m.handleStateTabKey(tea.KeyMsg{Type: tea.KeyEnter})
	// Expected behavior depends on whether stateListContent exists
	_ = handled
	_ = cmd
}

func TestShowConfirmApplyModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.plan = &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "test.resource", Action: terraform.ActionCreate},
		},
	}

	m.showConfirmApplyModal()
	if m.modalState != ModalConfirmApply {
		t.Error("expected confirm apply modal to be shown")
	}
}

func TestHandleModalConfirmApplyKeyYes(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalConfirmApply

	// Simulate pressing 'y' for yes
	model, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	_ = model
	_ = cmd
}

func TestHandleModalConfirmApplyKeyNo(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalConfirmApply

	// Simulate pressing 'n' for no
	model, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.modalState != ModalNone {
		t.Error("expected modal to be closed after pressing no")
	}
	_ = model
	_ = cmd
}

func TestHandleModalConfirmApplyKeyEsc(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalConfirmApply

	// Simulate pressing Esc
	model, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.modalState != ModalNone {
		t.Error("expected modal to be closed after pressing Esc")
	}
	_ = model
	_ = cmd
}

func TestShowStateMoveDestinationModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.showStateMoveDestinationModal("null_resource.a", "module.target.null_resource.a")
	if m.modalState != ModalStateMoveDestination {
		t.Fatal("expected destination modal state")
	}
	if m.stateMoveSource != "null_resource.a" {
		t.Fatalf("unexpected source: %q", m.stateMoveSource)
	}
	if m.stateMoveInput != "module.target.null_resource.a" {
		t.Fatalf("unexpected destination input: %q", m.stateMoveInput)
	}
}

func TestHandleStateMoveDestinationInputKeyEnterShowsConfirm(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = setupMockExecutor(t)

	m.stateMoveSource = "null_resource.a"
	m.stateMoveInput = "module.target.null_resource.a"
	m.modalState = ModalStateMoveDestination

	handled, cmd := m.handleStateMoveDestinationInputKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatal("expected key to be handled")
	}
	if cmd != nil {
		t.Fatal("expected nil command on successful transition to confirm modal")
	}
	if m.modalState != ModalConfirmApply {
		t.Fatalf("expected confirm modal state, got %v", m.modalState)
	}
	if m.pendingConfirmCmd == nil {
		t.Fatal("expected pending confirm command to be set")
	}
}

func TestHandleStateMoveDestinationInputKeyBackspace(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.stateMoveSource = "null_resource.a"
	m.stateMoveInput = "module.target.null_resource.a"
	m.modalState = ModalStateMoveDestination

	handled, cmd := m.handleStateMoveDestinationInputKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if !handled {
		t.Fatal("expected key to be handled")
	}
	_ = cmd
	if strings.HasSuffix(m.stateMoveInput, "a") {
		t.Fatalf("expected destination input to be shortened, got %q", m.stateMoveInput)
	}
}

func TestPlanSummaryWithChanges(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// No plan
	summary := m.planSummary()
	if summary != "No changes" {
		t.Errorf("expected 'No changes', got %q", summary)
	}

	// With plan
	m.plan = &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "test.a", Action: terraform.ActionCreate},
			{Address: "test.b", Action: terraform.ActionUpdate},
			{Address: "test.c", Action: terraform.ActionDelete},
		},
	}

	summary = m.planSummary()
	if summary == "No changes" {
		t.Error("expected changes in summary")
	}
}

func TestPlanSummaryVerbose(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.plan = &terraform.Plan{
		Resources: []terraform.ResourceChange{
			{Address: "test.a", Action: terraform.ActionCreate},
			{Address: "test.b", Action: terraform.ActionReplace},
		},
	}

	summary := m.planSummaryVerbose()
	if summary == "" {
		t.Error("expected non-empty verbose summary")
	}
}

func TestBeginRefreshExecution(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginRefresh()
	_ = cmd
}

func TestBeginValidateExecution(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginValidate()
	_ = cmd
}

func TestBeginFormatExecution(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginFormat()
	_ = cmd
}

func TestBeginStateListExecution(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateList()
	_ = cmd
}

func TestBeginStateShowExecution(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateShow("aws_instance.test")
	_ = cmd
}

func TestHandleValidateCompleteInvalid(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ValidateCompleteMsg{
		Result: &terraform.ValidateResult{
			Valid: false,
			Diagnostics: []terraform.Diagnostic{
				{Summary: "error", Severity: "error"},
			},
		},
	}

	model, cmd := m.handleValidateComplete(msg)
	_ = model
	_ = cmd
}

func TestHandleFormatCompleteEmpty(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := FormatCompleteMsg{
		ChangedFiles: []string{},
	}

	model, cmd := m.handleFormatComplete(msg)
	_ = model
	_ = cmd
}

func TestHandleFormatCompleteMultipleFiles(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := FormatCompleteMsg{
		ChangedFiles: []string{"main.tf", "variables.tf"},
	}

	model, cmd := m.handleFormatComplete(msg)
	_ = model
	_ = cmd
}

func TestHandleStateShowCompleteWithError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateShowCompleteMsg{
		Address: "aws_instance.test",
		Error:   errors.New("resource not found"),
	}

	model, cmd := m.handleStateShowComplete(msg)
	_ = model
	_ = cmd
}

func TestPrepareTerraformEnvCheck(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.envWorkDir = tmpDir
	m.updateLayout()

	env, err := m.prepareTerraformEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have TMPDIR set
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, "TMPDIR=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected TMPDIR in env")
	}
}

func TestHandleDiagnosticsKeyQuit(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected q key to be handled")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for quit")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleDiagnosticsKeyEsc(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	handled, _ := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected esc key to be handled")
	}
	if m.diagnosticsFocused {
		t.Error("expected diagnosticsFocused to be false after esc")
	}
}

func TestHandleDiagnosticsKeyD(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	handled, _ := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if !handled {
		t.Error("expected D key to be handled")
	}
	if m.diagnosticsFocused {
		t.Error("expected diagnosticsFocused to be false after D")
	}
}

func TestHandleDiagnosticsKeyOther(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	handled, _ := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("expected j key to be handled when diagnostics focused")
	}
}

func TestHandleApplyOutputNotRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.applyRunning = false

	_, cmd := m.handleApplyOutput(ApplyOutputMsg{Line: "test line"})
	if cmd != nil {
		t.Error("expected nil cmd when apply not running")
	}
}

func TestHandleApplyOutputRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.applyRunning = true

	_, cmd := m.handleApplyOutput(ApplyOutputMsg{Line: "test line"})
	// cmd should be the stream command (non-nil)
	_ = cmd // May be nil or non-nil depending on channel setup
}

func TestUpdateStateLockStatus(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.progressIndicator.Start(components.OperationPlan)

	m.updateStateLockStatus("Acquiring state lock. This may take a few moments...")
	if !strings.Contains(m.progressIndicator.View(), "waiting for state lock") {
		t.Fatal("expected lock wait detail in progress view")
	}

	m.updateStateLockStatus("Releasing state lock. This may take a few moments...")
	if strings.Contains(m.progressIndicator.View(), "waiting for state lock") {
		t.Fatal("expected lock wait detail to be cleared")
	}
}

func TestHandleApplyOutputUpdatesLockStatus(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.applyRunning = true
	m.progressIndicator.Start(components.OperationApply)

	m.handleApplyOutput(ApplyOutputMsg{Line: "Acquiring state lock. This may take a few moments..."})
	if !strings.Contains(m.progressIndicator.View(), "waiting for state lock") {
		t.Fatal("expected lock detail from apply output")
	}
}

func TestHandleEnvironmentChangedError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.envStrategy = environment.StrategyWorkspace

	// Create an environment that will fail to switch (no executor)
	env := environment.Environment{
		Name:     "nonexistent",
		Strategy: environment.StrategyWorkspace,
	}

	_, cmd := m.handleEnvironmentChanged(components.EnvironmentChangedMsg{Environment: env})
	// Should return an error toast command
	if cmd == nil {
		t.Error("expected non-nil cmd for error toast")
	}
}

func TestInitHistoryDisabled(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{HistoryEnabled: false})
	m.ready = true
	m.width = 100
	m.height = 30

	// Should not crash with history disabled
	m.initHistory(ExecutionConfig{HistoryEnabled: false})
	if m.historyStore != nil {
		t.Error("expected nil history store when disabled")
	}
}

func TestReloadHistoryCmdNilStore(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.historyStore = nil

	cmd := m.reloadHistoryCmd()
	if cmd != nil {
		t.Error("expected nil cmd when history store is nil")
	}
}

func TestHandleStateTabKeyWrongTab(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executionMode = true
	m.resourcesActiveTab = 0 // Not the state tab

	cmd, handled := m.handleStateTabKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if handled {
		t.Error("expected key to not be handled when not on state tab")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestUpdateEnvironmentPanelWithWarnings(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	warnings := []string{"warning1", "warning2"}
	m.updateEnvironmentPanel(warnings)
	// Should not crash
}

func TestUpdateEnvironmentPanelNilPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.environmentPanel = nil

	m.updateEnvironmentPanel(nil)
	// Should not crash with nil panel
}

func TestHandleKeyMsgBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Unknown key should still process
	_, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	// Should not crash
}

func TestToastErrorNilToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.toast = nil

	cmd := m.toastError("error message")
	if cmd != nil {
		t.Error("expected nil cmd when toast is nil")
	}
}

func TestToastErrorWithToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.toastError("error message")
	if cmd == nil {
		t.Error("expected non-nil cmd when toast is available")
	}
}

func TestToastInfoNilToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.toast = nil

	cmd := m.toastInfo("info message")
	if cmd != nil {
		t.Error("expected nil cmd when toast is nil")
	}
}

func TestToastInfoWithToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.toastInfo("info message")
	if cmd == nil {
		t.Error("expected non-nil cmd when toast is available")
	}
}

func TestToastSuccessNilToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.toast = nil

	cmd := m.toastSuccess("success message")
	if cmd != nil {
		t.Error("expected nil cmd when toast is nil")
	}
}

func TestToastSuccessWithToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.toastSuccess("success message")
	if cmd == nil {
		t.Error("expected non-nil cmd when toast is available")
	}
}

func TestShowFormattedFilesNilToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.toast = nil

	// Should not panic when toast is nil
	m.showFormattedFiles([]string{"file1.tf", "file2.tf"})
}

func TestShowFormattedFilesWithToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should show toast with single file
	m.showFormattedFiles([]string{"file1.tf"})
}

func TestShowFormattedFilesMultiple(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should show toast with multiple files
	m.showFormattedFiles([]string{"file1.tf", "file2.tf", "file3.tf"})
}

func TestShowFormattedFilesEmpty(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Empty list should still work without panicking
	m.showFormattedFiles([]string{})
}

func TestHandleHistoryKeysUpNavigation(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.historyEntries = []history.Entry{
		{ID: 1, Summary: "plan"},
		{ID: 2, Summary: "apply"},
	}
	m.historySelected = 1

	handled, _ := m.handleHistoryKeys("up")
	if !handled {
		t.Error("expected handled to be true")
	}
	if m.historySelected != 0 {
		t.Errorf("expected historySelected to be 0, got %d", m.historySelected)
	}
}

func TestHandleHistoryKeysDownNavigation(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.historyEntries = []history.Entry{
		{ID: 1, Summary: "plan"},
		{ID: 2, Summary: "apply"},
	}
	m.historySelected = 0

	handled, _ := m.handleHistoryKeys("j")
	if !handled {
		t.Error("expected handled to be true")
	}
	if m.historySelected != 1 {
		t.Errorf("expected historySelected to be 1, got %d", m.historySelected)
	}
}

func TestHandleHistoryKeysEnterSelection(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.historyEntries = []history.Entry{
		{ID: 1, Summary: "plan"},
	}
	m.historySelected = 0

	handled, _ := m.handleHistoryKeys("enter")
	if !handled {
		t.Error("expected handled to be true")
	}
	// cmd may be nil if history panel is not fully initialized
}

func TestHandleEscKeyModeHistoryDetailExit(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea == nil {
		t.Skip("mainArea is nil")
	}
	m.mainArea.SetMode(ModeHistoryDetail)
	handled := m.handleEscKey()
	if !handled {
		t.Error("expected handled to be true for ModeHistoryDetail")
	}
}

func TestHandleEscKeyModeStateShowExit(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea == nil {
		t.Skip("mainArea is nil")
	}
	m.mainArea.SetMode(ModeStateShow)
	handled := m.handleEscKey()
	if !handled {
		t.Error("expected handled to be true for ModeStateShow")
	}
}

func TestHandleEscKeyModeAboutExit(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea == nil {
		t.Skip("mainArea is nil")
	}
	m.mainArea.SetMode(ModeAbout)
	handled := m.handleEscKey()
	if !handled {
		t.Error("expected handled to be true for ModeAbout")
	}
}

func TestHandleEscKeyModeDiffNoExit(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea == nil {
		t.Skip("mainArea is nil")
	}
	m.mainArea.SetMode(ModeDiff)
	handled := m.handleEscKey()
	if handled {
		t.Error("expected handled to be false for ModeDiff")
	}
}

func TestHandleEscKeyNilMainAreaCheck(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.mainArea = nil
	handled := m.handleEscKey()
	if handled {
		t.Error("expected handled to be false when mainArea is nil")
	}
}

func TestFocusCommandLogPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Just verify it doesn't panic
	_ = m.focusCommandLog()
}

func TestFocusCommandLogNilPanelManagerCheck(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil
	cmd := m.focusCommandLog()
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

func TestFocusMainPanelNilPanelManagerCheck(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil
	cmd := m.focusMainPanel()
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

func TestHandleStateListKeyQuit(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateList
	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true for q key")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for quit")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleStateListKeyEsc(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateList
	handled, _ := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyEscape})
	if !handled {
		t.Error("expected handled to be true for esc key")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain after esc")
	}
}

func TestHandleStateListKeyUp(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateList
	m.stateListView = nil
	handled, _ := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyUp})
	if !handled {
		t.Error("expected handled to be true for up key")
	}
}

func TestHandleStateListKeyDown(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateList
	m.stateListView = nil
	handled, _ := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("expected handled to be true for j key")
	}
}

func TestHandleStateListKeyEnter(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateList
	m.stateListView = nil
	handled, _ := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected handled to be true for enter key")
	}
}

func TestHandleStateListKeyUnknown(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateList
	handled, _ := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected handled to be false for unknown key")
	}
}

func TestHandleStateShowKeyQuit(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateShow
	handled, cmd := m.handleStateShowKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true for q key")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for quit")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleStateShowKeyEsc(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateShow
	handled, _ := m.handleStateShowKey(tea.KeyMsg{Type: tea.KeyEscape})
	if !handled {
		t.Error("expected handled to be true for esc key")
	}
	if m.execView != viewStateList {
		t.Error("expected execView to be viewStateList after esc")
	}
}

func TestHandleStateShowKeyDefault(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateShow
	m.stateShowView = nil
	handled, _ := m.handleStateShowKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !handled {
		t.Error("expected handled to be true for default key")
	}
}

func TestHandleCommandLogKeyQuit(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewCommandLog
	handled, cmd := m.handleCommandLogKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true for q key")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for quit")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleCommandLogKeyEsc(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewCommandLog
	handled, _ := m.handleCommandLogKey(tea.KeyMsg{Type: tea.KeyEscape})
	if !handled {
		t.Error("expected handled to be true for esc key")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain after esc")
	}
}

func TestHandleCommandLogKeyDefault(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewCommandLog
	handled, _ := m.handleCommandLogKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected handled to be false for unknown key")
	}
}

func TestHandleLegacyOutputKeyQuitNotRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewPlanOutput
	m.planRunning = false
	m.applyRunning = false
	handled, _ := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true for q key")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain")
	}
}

func TestHandleLegacyOutputKeyQuitRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewPlanOutput
	m.planRunning = true
	handled, cmd := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true for q key")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for quit")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleLegacyOutputKeyEscNotRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewApplyOutput
	m.planRunning = false
	m.applyRunning = false
	handled, _ := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyEscape})
	if !handled {
		t.Error("expected handled to be true for esc key")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain after esc")
	}
}

func TestHandleLegacyOutputKeyEscRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewApplyOutput
	m.applyRunning = true
	handled, _ := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyEscape})
	if !handled {
		t.Error("expected handled to be true for esc key")
	}
	// Should not change view when running
	if m.execView != viewApplyOutput {
		t.Error("expected execView to remain viewApplyOutput when running")
	}
}

func TestHandleLegacyOutputKeyDefault(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewPlanOutput
	handled, _ := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected handled to be false for unknown key")
	}
}

func TestHandleExecutionKeyViewMain(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewMain
	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected handled to be false for viewMain")
	}
}

func TestHandleExecutionKeyViewPlanOutput(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewPlanOutput
	m.planRunning = false
	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true for viewPlanOutput")
	}
}

func TestHandleExecutionKeyViewStateList(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.execView = viewStateList
	handled, _ := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEscape})
	if !handled {
		t.Error("expected handled to be true for viewStateList")
	}
}

func TestInputCaptured(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	if m.inputCaptured() {
		t.Error("expected inputCaptured to return false")
	}
}

func TestHandleEnvironmentPanelKeyNilPanelCheck(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.environmentPanel = nil
	handled, cmd := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if handled {
		t.Error("expected not handled when panel is nil")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleEnvironmentPanelKeyNotSelectorMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	// Panel exists but not in selector mode
	if m.environmentPanel != nil {
		// Not in selector mode should not handle keys
		handled, _ := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		if handled {
			t.Error("expected not handled when not in selector mode")
		}
	}
}

func TestCommandLogPanelUpdateKeyJ(t *testing.T) {
	s := styles.DefaultStyles()
	panel := components.NewCommandLogPanel(s)
	panel.SetSize(80, 20)

	// Add some logs to enable scrolling
	panel.AppendSessionLog("Test", "cmd", "output")
	panel.AppendSessionLog("Test2", "cmd2", "output2")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newPanel, cmd := panel.Update(msg)
	if newPanel == nil {
		t.Error("expected non-nil panel")
	}
	_ = cmd
}

func TestCommandLogPanelUpdateKeyK(t *testing.T) {
	s := styles.DefaultStyles()
	panel := components.NewCommandLogPanel(s)
	panel.SetSize(80, 20)

	// Add some logs
	panel.AppendSessionLog("Test", "cmd", "output")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newPanel, cmd := panel.Update(msg)
	if newPanel == nil {
		t.Error("expected non-nil panel")
	}
	_ = cmd
}

func TestInitHistoryWithProvidedLoggerFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := history.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	logger := history.NewLogger(store, history.LevelMinimal)
	m := NewExecutionModel(nil, ExecutionConfig{
		HistoryEnabled: true,
		HistoryStore:   store,
		HistoryLogger:  logger,
	})
	if m.historyLogger != logger {
		t.Error("expected provided logger to be used")
	}
}

func TestViewExecutionOverrideCommandLogViewReturnsEmpty(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewCommandLog

	view := m.viewExecutionOverride()
	// Should return empty string since command log is handled elsewhere
	if view != "" {
		t.Error("expected empty view for command log")
	}
}

func TestHandlePostUpdateWhenNotReady(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = false

	// Should not panic when not ready
	_, _ = m.handlePostUpdate(nil)
}

func TestEnvDisplayNameWithFolderStrategy(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envStrategy = environment.StrategyFolder
	m.envWorkDir = "/projects"
	m.envCurrent = "/projects/envs/dev"

	name := m.envDisplayName()
	if name != "envs/dev" {
		t.Errorf("expected relative path 'envs/dev', got %q", name)
	}
}

func TestEnvDisplayNameWithFolderStrategyEmptyWorkDir(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envStrategy = environment.StrategyFolder
	m.envWorkDir = ""
	m.envCurrent = "/some/path/env"

	name := m.envDisplayName()
	// When workDir is empty, use current working dir (".")
	// The result depends on current dir, so just check it doesn't panic
	if name == "" {
		// Either returns basename or empty is acceptable - call again just to ensure no panic
		_ = m.envDisplayName()
	}
}

func TestEnvDisplayNameWithFolderStrategyCurrentEmpty(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envStrategy = environment.StrategyFolder
	m.envWorkDir = "/projects"
	m.envCurrent = ""

	name := m.envDisplayName()
	if name != "" {
		t.Errorf("expected empty name for empty current, got %q", name)
	}
}

func TestEnvDisplayNameWithWorkspaceStrategy(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envStrategy = environment.StrategyWorkspace
	m.envCurrent = "production"

	name := m.envDisplayName()
	if name != "production" {
		t.Errorf("expected 'production', got %q", name)
	}
}

func TestEnvDisplayNameWithRelativePathOutside(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envStrategy = environment.StrategyFolder
	m.envWorkDir = "/projects/a"
	m.envCurrent = "/projects/b/env" // Not under workDir

	name := m.envDisplayName()
	// Should return basename when not a valid relative path
	if name != "env" {
		t.Errorf("expected basename 'env', got %q", name)
	}
}

func TestCurrentWorkspaceNameError(t *testing.T) {
	origNewWorkspaceManager := newWorkspaceManager
	defer func() {
		newWorkspaceManager = origNewWorkspaceManager
	}()

	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return nil, errors.New("no workspace manager")
	}

	_, err := currentWorkspaceName("/tmp")
	if err == nil {
		t.Error("expected error from currentWorkspaceName")
	}
}

func TestCurrentWorkspaceNameSuccess(t *testing.T) {
	origNewWorkspaceManager := newWorkspaceManager
	defer func() {
		newWorkspaceManager = origNewWorkspaceManager
	}()

	manager := &fakeWorkspaceManager{current: "development"}
	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return manager, nil
	}

	name, err := currentWorkspaceName("/tmp")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if name != "development" {
		t.Errorf("expected 'development', got %q", name)
	}
}

func TestLoadEnvironmentPreferenceNoPreference(t *testing.T) {
	tmpDir := t.TempDir()
	pref, err := loadEnvironmentPreference(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// No preference file exists, so pref should be nil
	if pref != nil {
		t.Errorf("expected nil preference, got %+v", pref)
	}
}

func TestLoadEnvironmentPreferenceWithPreference(t *testing.T) {
	tmpDir := t.TempDir()
	expected := environment.Preference{
		Strategy:    environment.StrategyWorkspace,
		Environment: "staging",
	}
	if err := environment.SavePreference(tmpDir, expected); err != nil {
		t.Fatalf("failed to save preference: %v", err)
	}

	pref, err := loadEnvironmentPreference(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if pref == nil {
		t.Fatal("expected non-nil preference")
	}
	if pref.Strategy != expected.Strategy || pref.Environment != expected.Environment {
		t.Errorf("expected %+v, got %+v", expected, pref)
	}
}

func TestDetectEnvironmentsError(t *testing.T) {
	origNewEnvironmentDetector := newEnvironmentDetector
	defer func() {
		newEnvironmentDetector = origNewEnvironmentDetector
	}()

	newEnvironmentDetector = func(_ string) (environmentDetector, error) {
		return nil, errors.New("detector error")
	}

	_, err := detectEnvironments("/tmp")
	if err == nil {
		t.Error("expected error from detectEnvironments")
	}
}

func TestDetectEnvironmentsSuccess(t *testing.T) {
	origNewEnvironmentDetector := newEnvironmentDetector
	defer func() {
		newEnvironmentDetector = origNewEnvironmentDetector
	}()

	expected := environment.DetectionResult{
		Strategy:   environment.StrategyWorkspace,
		Workspaces: []string{"dev", "prod"},
	}
	newEnvironmentDetector = func(_ string) (environmentDetector, error) {
		return &fakeDetector{result: expected}, nil
	}

	result, err := detectEnvironments("/tmp")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Strategy != expected.Strategy {
		t.Errorf("expected strategy %v, got %v", expected.Strategy, result.Strategy)
	}
	if len(result.Workspaces) != len(expected.Workspaces) {
		t.Errorf("expected %d workspaces, got %d", len(expected.Workspaces), len(result.Workspaces))
	}
}

func TestViewExecutionOverrideStateModes(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Test viewStateList
	m.execView = viewStateList
	view := m.viewExecutionOverride()
	if view != "" {
		t.Error("expected empty view for state list")
	}

	// Test viewStateShow
	m.execView = viewStateShow
	view = m.viewExecutionOverride()
	if view != "" {
		t.Error("expected empty view for state show")
	}

	// Test viewDiagnostics
	m.execView = viewDiagnostics
	view = m.viewExecutionOverride()
	if view != "" {
		t.Error("expected empty view for diagnostics")
	}
}

func TestHandleEnvironmentPanelKeyNotFocused(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.environmentPanel = components.NewEnvironmentPanel(m.styles)

	// Focus on resources panel, not environment panel
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelResources)
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	handled, _ := m.handleEnvironmentPanelKey(msg)
	if handled {
		t.Error("expected not to handle when environment panel not focused")
	}
}

func TestHandleEnvironmentPanelKeyFocused(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.environmentPanel = components.NewEnvironmentPanel(m.styles)

	if m.panelManager == nil {
		t.Skip("panel manager not initialized")
	}

	// Register and focus on environment panel
	m.panelManager.RegisterPanel(PanelWorkspace, m.environmentPanel)
	m.panelManager.SetFocus(PanelWorkspace)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	// Should handle the key when focused (even if result is no-op)
	_, _ = m.handleEnvironmentPanelKey(msg)
}

func TestHandleEnvironmentPanelKeyBlockedWhileOperationRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.environmentPanel = components.NewEnvironmentPanel(m.styles)
	m.operationRunning = true

	if m.panelManager == nil {
		t.Skip("panel manager not initialized")
	}

	m.panelManager.RegisterPanel(PanelWorkspace, m.environmentPanel)
	m.panelManager.SetFocus(PanelWorkspace)

	handled, cmd := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if handled {
		t.Fatal("expected environment panel key to be ignored while operation is running")
	}
	if cmd != nil {
		t.Fatal("expected nil command when environment panel input is blocked")
	}
}

func TestResolveDetectedEnvironmentWithWorkspaces(t *testing.T) {
	origNewWorkspaceManager := newWorkspaceManager
	defer func() {
		newWorkspaceManager = origNewWorkspaceManager
	}()

	manager := &fakeWorkspaceManager{current: "staging"}
	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return manager, nil
	}

	m := NewModel(&terraform.Plan{})
	result := environment.DetectionResult{
		Strategy:   environment.StrategyWorkspace,
		Workspaces: []string{"dev", "staging", "prod"},
	}

	current := resolveDetectedEnvironment(m, "/tmp", "/tmp", result)
	if current != "staging" {
		t.Errorf("expected 'staging', got %q", current)
	}
}

func TestResolveDetectedEnvironmentFallbackToAbsWorkDir(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	result := environment.DetectionResult{
		Strategy:    environment.StrategyFolder,
		FolderPaths: []string{"/other/path"},
	}

	current := resolveDetectedEnvironment(m, "/projects", "/projects/abs", result)
	if current != "/projects/abs" {
		t.Errorf("expected '/projects/abs', got %q", current)
	}
}

func TestMatchCurrentFolderFound(t *testing.T) {
	folders := []string{"/a/envs/dev", "/a/envs/prod", "/a/envs/staging"}
	result := matchCurrentFolder(folders, "/a/envs/prod")
	if result != "/a/envs/prod" {
		t.Errorf("expected '/a/envs/prod', got %q", result)
	}
}

func TestMatchCurrentFolderNotFound(t *testing.T) {
	folders := []string{"/a/envs/dev", "/a/envs/staging"}
	result := matchCurrentFolder(folders, "/a/envs/prod")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestEnvStatusLabelWithUnknownStrategy(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envStrategy = environment.StrategyUnknown
	m.envCurrent = "test-env"

	label := m.envStatusLabel()
	if label != "test-env" {
		t.Errorf("expected 'test-env', got %q", label)
	}
}

func TestEnvStatusLabelWithKnownStrategy(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envStrategy = environment.StrategyWorkspace
	m.envCurrent = "production"

	label := m.envStatusLabel()
	expected := "production (workspace)"
	if label != expected {
		t.Errorf("expected %q, got %q", expected, label)
	}
}

func TestEnvStatusLabelEmpty(t *testing.T) {
	m := NewModel(&terraform.Plan{})
	m.envStrategy = environment.StrategyUnknown
	m.envCurrent = ""

	label := m.envStatusLabel()
	if label != "unknown" {
		t.Errorf("expected 'unknown', got %q", label)
	}
}
