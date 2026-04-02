package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
	tfparser "github.com/ushiradineth/lazytf/internal/terraform/parser"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/ui/views"
	"github.com/ushiradineth/lazytf/internal/utils"
)

const noChangesSummary = "No changes"

// Execution-related methods for Model

func (m *Model) beginPlan() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	m.err = nil
	planEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}
	planFlags, planFilePath := m.planFlagsForRun()
	m.planRunFlags = planFlags
	m.planFilePath = planFilePath
	// Cancel any previous execution before starting new one
	m.cancelExecution()
	ctx := m.beginTrackedOperation(0)
	m.planRunning = true
	m.planStartedAt = time.Now()

	// Keep in main view, switch MainArea to logs mode during plan
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeLogs)
	}

	// Show command log panel during operations
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
		m.updateLayout()
	}

	if m.applyView != nil {
		m.applyView.Reset()
		m.applyView.SetTitle("Running terraform plan...")
		m.applyView.SetStatus(views.ApplyRunning)
	}
	m.updateExecutionViewForStreaming()

	// Start progress indicator
	var progressCmd tea.Cmd
	if m.progressIndicator != nil {
		progressCmd = m.progressIndicator.Start(components.OperationPlan)
	}

	planCmd := func() tea.Msg {
		result, output, err := m.executor.Plan(ctx, terraform.PlanOptions{
			Flags: planFlags,
			Env:   planEnv,
		})
		return PlanStartMsg{Result: result, Output: output, Error: err}
	}

	if progressCmd != nil {
		return tea.Batch(planCmd, progressCmd)
	}
	return planCmd
}

func (m *Model) beginApply() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	m.err = nil
	applyEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}
	applyFlags, err := m.applyFlagsForRun()
	if err != nil {
		m.err = err
		return m.toastError(err.Error())
	}
	m.applyRunFlags = applyFlags
	// Cancel any previous execution before starting new one
	m.cancelExecution()
	ctx := m.beginTrackedOperation(0)
	m.applyRunning = true
	m.applyStartedAt = time.Now()

	// Keep in main view, switch MainArea to logs mode during apply
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeLogs)
	}

	// Show command log panel during operations
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
		m.updateLayout()
	}

	// Clear command log content - it will be populated when apply completes
	if m.commandLogPanel != nil {
		m.commandLogPanel.SetLogText("")
	}

	if m.applyView != nil {
		m.applyView.Reset()
		m.applyView.SetTitle("Applying changes...")
		m.applyView.SetStatus(views.ApplyRunning)
	}
	// Transition to main view from confirm view
	if m.execView == viewPlanConfirm {
		m.execView = viewMain
	}
	m.updateExecutionViewForStreaming()

	// Start progress indicator
	var progressCmd tea.Cmd
	if m.progressIndicator != nil {
		progressCmd = m.progressIndicator.Start(components.OperationApply)
	}

	applyCmd := func() tea.Msg {
		result, output, err := m.executor.Apply(ctx, terraform.ApplyOptions{
			Flags:       applyFlags,
			AutoApprove: true,
			Env:         applyEnv,
		})
		return ApplyStartMsg{Result: result, Output: output, Error: err}
	}

	if progressCmd != nil {
		return tea.Batch(applyCmd, progressCmd)
	}
	return applyCmd
}

func (m *Model) beginRefresh() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	m.err = nil
	refreshEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}
	// Cancel any previous execution before starting new one
	m.cancelExecution()
	ctx := m.beginTrackedOperation(0)
	m.refreshRunning = true
	m.refreshStartedAt = time.Now()

	// Keep in main view, switch MainArea to logs mode during refresh
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeLogs)
	}

	// Show command log panel during operations
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
		m.updateLayout()
	}

	if m.applyView != nil {
		m.applyView.Reset()
		m.applyView.SetTitle("Running terraform refresh...")
		m.applyView.SetStatus(views.ApplyRunning)
	}
	m.updateExecutionViewForStreaming()

	// Start progress indicator
	var progressCmd tea.Cmd
	if m.progressIndicator != nil {
		progressCmd = m.progressIndicator.Start(components.OperationRefresh)
	}

	refreshCmd := func() tea.Msg {
		result, output, err := m.executor.Refresh(ctx, terraform.RefreshOptions{
			Env: refreshEnv,
		})
		return RefreshStartMsg{Result: result, Output: output, Error: err}
	}

	if progressCmd != nil {
		return tea.Batch(refreshCmd, progressCmd)
	}
	return refreshCmd
}

func (m *Model) handleRefreshStart(msg RefreshStartMsg) (tea.Model, tea.Cmd) {
	return m.handleOperationStart(
		msg.Error,
		&m.refreshRunning,
		"Failed to start terraform refresh",
		"Refresh failed to start",
		msg.Output,
		m.waitRefreshCompleteCmd(msg.Result),
		m.streamRefreshOutputCmd(),
	)
}

func (m *Model) handleRefreshComplete(msg RefreshCompleteMsg) (tea.Model, tea.Cmd) {
	m.refreshRunning = false
	m.finishTrackedOperation()
	m.outputChan = nil

	// Switch MainArea back to diff mode when refresh completes
	if m.mainArea != nil {
		m.mainArea.SetMode(ModeDiff)
	}

	// Log to session history
	var refreshOutput string
	if msg.Result != nil {
		refreshOutput = msg.Result.Output
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog("Refreshed", "terraform apply -refresh-only", refreshOutput)
	}

	if msg.Error != nil || !msg.Success {
		// Mark progress indicator as failed
		if m.progressIndicator != nil {
			m.progressIndicator.Fail()
		}
		return m.handleRefreshFailure(msg)
	}

	// Reset progress indicator on success
	if m.progressIndicator != nil {
		m.progressIndicator.Reset()
	}

	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
	}
	// Route logs to command log panel
	parsed := ""
	if msg.Result != nil {
		parsed = utils.FormatLogOutput(msg.Result.Output)
	}
	if strings.TrimSpace(parsed) == "" {
		parsed = "Refresh complete"
	}
	m.setDiagnostics(nil)
	m.setLogText(parsed)
	m.updateExecutionViewForStreaming()
	var toastCmd tea.Cmd
	if m.toast != nil {
		toastCmd = m.toast.ShowSuccess("State refreshed successfully")
	}
	notifyCmd := m.notifyOperationCmd("refresh", m.flattenSummary(m.planSummary()), m.refreshStartedAt, msg.Result, nil)
	return m, tea.Batch(
		toastCmd,
		m.recordOperationCmd("refresh", nil, true, m.refreshStartedAt, msg.Result, "", nil),
		notifyCmd,
	)
}

func (m *Model) streamRefreshOutputCmd() tea.Cmd {
	return func() tea.Msg {
		if m.outputChan == nil {
			return nil
		}
		line, ok := <-m.outputChan
		if !ok {
			return nil
		}
		return RefreshOutputMsg{Line: line}
	}
}

func (m *Model) waitRefreshCompleteCmd(result *terraform.ExecutionResult) tea.Cmd {
	return func() tea.Msg {
		if result == nil {
			return RefreshCompleteMsg{Success: false, Error: errors.New("refresh execution result missing")}
		}
		<-result.Done()
		if result.Error != nil {
			return RefreshCompleteMsg{Success: false, Error: result.Error, Result: result}
		}
		return RefreshCompleteMsg{Success: true, Result: result}
	}
}

func (m *Model) beginValidate() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	if m.toast != nil {
		m.toast.ShowInfo("Running terraform validate...")
	}
	validateEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	// Start progress indicator
	var progressCmd tea.Cmd
	if m.progressIndicator != nil {
		progressCmd = m.progressIndicator.Start(components.OperationValidate)
	}

	ctx := m.beginTrackedOperation(2 * time.Minute)

	validateCmd := func() tea.Msg {
		result, err := m.executor.Validate(ctx, terraform.ValidateOptions{
			Env: validateEnv,
		})
		if err != nil {
			return ValidateCompleteMsg{Error: err}
		}
		if result == nil {
			return ValidateCompleteMsg{Error: errors.New("validate execution result missing")}
		}
		rawOutput := strings.TrimSpace(result.Output)
		if result.Error != nil {
			return ValidateCompleteMsg{Error: result.Error, RawOutput: rawOutput, ExecResult: result}
		}
		// Parse JSON output
		var validateResult terraform.ValidateResult
		if result.Stdout != "" {
			if parseErr := json.Unmarshal([]byte(result.Stdout), &validateResult); parseErr != nil {
				return ValidateCompleteMsg{Error: parseErr, RawOutput: rawOutput, ExecResult: result}
			}
		}
		return ValidateCompleteMsg{Result: &validateResult, RawOutput: rawOutput, ExecResult: result}
	}

	if progressCmd != nil {
		return tea.Batch(validateCmd, progressCmd)
	}
	return validateCmd
}

//nolint:gocognit,gocyclo // Validation result handling has inherent complexity
func (m *Model) handleValidateComplete(msg ValidateCompleteMsg) (tea.Model, tea.Cmd) {
	m.finishTrackedOperation()

	// Log to session history
	output := msg.RawOutput
	if msg.Error != nil {
		if strings.TrimSpace(output) == "" {
			output = msg.Error.Error()
		}
	} else if msg.Result != nil {
		if msg.Result.Valid {
			output = "Configuration is valid"
		} else {
			output = fmt.Sprintf("%d errors, %d warnings", msg.Result.ErrorCount, msg.Result.WarningCount)
		}
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog("Validated", "terraform validate -json", output)
	}

	if msg.Error != nil {
		// Mark progress indicator as failed
		if m.progressIndicator != nil {
			m.progressIndicator.Fail()
		}
		m.addErrorDiagnostic("Validate failed", msg.Error, msg.RawOutput)
		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowError(fmt.Sprintf("Validate failed: %v", msg.Error))
		}
		return m, cmd
	}

	if msg.Result == nil {
		// Reset progress indicator (no error, just no result)
		if m.progressIndicator != nil {
			m.progressIndicator.Reset()
		}
		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowInfo("Validate completed (no result)")
		}
		return m, cmd
	}

	// Display diagnostics in command log panel
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
		m.updateLayout()
	}

	if len(msg.Result.Diagnostics) > 0 {
		m.setDiagnostics(msg.Result.Diagnostics)
	} else {
		m.setDiagnostics(nil)
	}

	var cmd tea.Cmd
	if msg.Result.Valid { //nolint:nestif // Validation logic requires nested checks
		// Reset progress indicator on success
		if m.progressIndicator != nil {
			m.progressIndicator.Reset()
		}
		if m.toast != nil {
			cmd = m.toast.ShowSuccess("Configuration is valid")
		}
	} else {
		// Mark progress indicator as failed for validation errors
		if m.progressIndicator != nil {
			m.progressIndicator.Fail()
		}
		if m.toast != nil {
			cmd = m.toast.ShowError(fmt.Sprintf("Validation failed: %d errors, %d warnings", msg.Result.ErrorCount, msg.Result.WarningCount))
		}
	}
	return m, cmd
}

//nolint:gocognit // Command setup with error handling has inherent complexity.
func (m *Model) beginFormat() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	if m.toast != nil {
		m.toast.ShowInfo("Running terraform fmt...")
	}
	formatEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	// Start progress indicator
	var progressCmd tea.Cmd
	if m.progressIndicator != nil {
		progressCmd = m.progressIndicator.Start(components.OperationFormat)
	}

	ctx := m.beginTrackedOperation(2 * time.Minute)

	formatCmd := func() tea.Msg {
		result, err := m.executor.Format(ctx, terraform.FormatOptions{
			Recursive: true,
			Env:       formatEnv,
		})
		if err != nil {
			return FormatCompleteMsg{Error: err}
		}
		if result == nil {
			return FormatCompleteMsg{Error: errors.New("format execution result missing")}
		}
		if result.Error != nil {
			return FormatCompleteMsg{Error: result.Error, ExecResult: result}
		}
		// Parse output - each line is a changed file
		var changedFiles []string
		if result.Stdout != "" {
			for _, line := range strings.Split(result.Stdout, "\n") {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					changedFiles = append(changedFiles, trimmed)
				}
			}
		}
		return FormatCompleteMsg{ChangedFiles: changedFiles, ExecResult: result}
	}

	if progressCmd != nil {
		return tea.Batch(formatCmd, progressCmd)
	}
	return formatCmd
}

func (m *Model) handleFormatComplete(msg FormatCompleteMsg) (tea.Model, tea.Cmd) {
	m.finishTrackedOperation()

	// Log to session history
	var formatOutput string
	switch {
	case msg.Error != nil:
		formatOutput = formatResultOutput(msg)
	case len(msg.ChangedFiles) == 0:
		formatOutput = "No files changed"
	default:
		formatOutput = fmt.Sprintf("Formatted %d file(s):\n%s", len(msg.ChangedFiles), strings.Join(msg.ChangedFiles, "\n"))
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog("Formatted", "terraform fmt -recursive", formatOutput)
	}

	if msg.Error != nil {
		// Mark progress indicator as failed
		if m.progressIndicator != nil {
			m.progressIndicator.Fail()
		}
		m.addErrorDiagnostic("Format failed", msg.Error, formatResultOutput(msg))
		cmd := m.toastError(fmt.Sprintf("Format failed: %v", msg.Error))
		return m, cmd
	}

	// Reset progress indicator on success
	if m.progressIndicator != nil {
		m.progressIndicator.Reset()
	}

	if len(msg.ChangedFiles) == 0 {
		m.setDiagnostics(nil)
		cmd := m.toastInfo("No files changed")
		return m, cmd
	}

	m.setDiagnostics(nil)
	m.showFormattedFiles(msg.ChangedFiles)
	cmd := m.toastSuccess(fmt.Sprintf("Formatted %d file(s)", len(msg.ChangedFiles)))
	return m, cmd
}

func (m *Model) beginInit(upgrade bool) tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	initEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	if m.toast != nil {
		if upgrade {
			m.toast.ShowInfo("Running terraform init -upgrade...")
		} else {
			m.toast.ShowInfo("Running terraform init...")
		}
	}

	var progressCmd tea.Cmd
	if m.progressIndicator != nil {
		progressCmd = m.progressIndicator.Start(components.OperationInit)
	}

	ctx := m.beginTrackedOperation(3 * time.Minute)

	initCmd := func() tea.Msg {
		result, err := m.executor.Init(ctx, terraform.InitOptions{Upgrade: upgrade, Env: initEnv})
		if err != nil {
			return InitCompleteMsg{Error: err, Upgrade: upgrade}
		}
		if result == nil {
			return InitCompleteMsg{Error: errors.New("init execution result missing"), Upgrade: upgrade}
		}
		return InitCompleteMsg{Output: strings.TrimSpace(result.Output), Result: result, Error: result.Error, Upgrade: upgrade}
	}

	if progressCmd != nil {
		return tea.Batch(initCmd, progressCmd)
	}
	return initCmd
}

func (m *Model) handleInitComplete(msg InitCompleteMsg) (tea.Model, tea.Cmd) {
	m.finishTrackedOperation()

	output := strings.TrimSpace(msg.Output)
	if output == "" && msg.Error != nil {
		output = msg.Error.Error()
	}
	if output == "" {
		output = "Terraform init completed"
	}

	if m.commandLogPanel != nil {
		command := "terraform init"
		if msg.Upgrade {
			command = "terraform init -upgrade"
		}
		m.commandLogPanel.AppendSessionLog("Initialized", command, output)
	}

	if msg.Error != nil {
		if m.progressIndicator != nil {
			m.progressIndicator.Fail()
		}
		m.addErrorDiagnostic("Init failed", msg.Error, output)
		cmd := m.toastError(fmt.Sprintf("Init failed: %v", msg.Error))
		return m, cmd
	}

	if m.progressIndicator != nil {
		m.progressIndicator.Reset()
	}
	m.setDiagnostics(nil)
	cmd := m.toastSuccess("Terraform initialized")
	return m, cmd
}

func (m *Model) beginStateList() tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	if m.toast != nil {
		m.toast.ShowInfo("Loading state list...")
	}
	stateEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	// Start progress indicator
	var progressCmd tea.Cmd
	if m.progressIndicator != nil {
		progressCmd = m.progressIndicator.Start(components.OperationStateList)
	}

	ctx := m.beginTrackedOperation(2 * time.Minute)

	stateListCmd := func() tea.Msg {
		result, err := m.executor.StateList(ctx, terraform.StateListOptions{
			Env: stateEnv,
		})
		if err != nil {
			return StateListCompleteMsg{Error: err}
		}
		if result.Error != nil {
			return StateListCompleteMsg{Error: result.Error, Output: stateListResultOutput(result)}
		}
		resources := parseStateListResources(result.Stdout)
		return StateListCompleteMsg{Resources: resources, Output: stateListResultOutput(result)}
	}

	if progressCmd != nil {
		return tea.Batch(stateListCmd, progressCmd)
	}
	return stateListCmd
}

func (m *Model) handleStateListComplete(msg StateListCompleteMsg) (tea.Model, tea.Cmd) {
	m.finishTrackedOperation()

	// Hide loading toast
	if m.toast != nil {
		m.toast.Hide()
	}

	m.appendSessionLog("State listed", "terraform state list", stateListSessionOutput(msg))

	if msg.Error != nil {
		// Mark progress indicator as failed
		if m.progressIndicator != nil {
			m.progressIndicator.Fail()
		}
		if m.stateListContent != nil {
			m.stateListContent.SetError(stateListErrorText(msg))
		}
		m.addErrorDiagnostic("State list failed", msg.Error, msg.Output)
		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowError(fmt.Sprintf("State list failed: %v", msg.Error))
		}
		return m, cmd
	}

	// Reset progress indicator on success
	if m.progressIndicator != nil {
		m.progressIndicator.Reset()
	}

	// Update state list content (for tab view)
	m.setDiagnostics(nil)
	if m.stateListContent != nil {
		m.stateListContent.SetResources(msg.Resources)
	}
	if m.stateMoveSource != "" && !containsStateAddress(msg.Resources, m.stateMoveSource) {
		m.stateMoveSource = ""
	}

	// Initialize state list view if needed (for full screen view)
	if m.stateListView == nil {
		m.stateListView = views.NewStateListView(m.styles)
	}
	m.stateListView.SetSize(m.width, m.height)
	m.stateListView.SetResources(msg.Resources)

	// Automatically show the first item's details if we have resources
	if len(msg.Resources) > 0 {
		cmd := m.showSelectedStateDetail()
		return m, cmd
	}

	if m.mainArea != nil && m.mainArea.GetMode() == ModeStateShow {
		m.mainArea.SetStateContent("", "No resources in state. Press 'r' to refresh.")
	}

	return m, nil
}

func (m *Model) beginStateShow(address string) tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	stateEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	ctx := m.beginTrackedOperation(2 * time.Minute)

	return func() tea.Msg {
		result, err := m.executor.StateShow(ctx, address, terraform.StateShowOptions{
			Env: stateEnv,
		})
		if err != nil {
			return StateShowCompleteMsg{Address: address, Error: err}
		}
		if result == nil {
			return StateShowCompleteMsg{Address: address, Error: errors.New("state show execution result missing")}
		}
		if result.Error != nil {
			return StateShowCompleteMsg{Address: address, Error: result.Error, Output: stateShowResultOutput(result)}
		}
		return StateShowCompleteMsg{Address: address, Output: stateShowResultOutput(result)}
	}
}

func (m *Model) handleStateShowComplete(msg StateShowCompleteMsg) (tea.Model, tea.Cmd) {
	m.finishTrackedOperation()

	m.appendSessionLog("State shown", "terraform state show "+msg.Address, stateShowSessionOutput(msg))

	if msg.Error != nil {
		m.addErrorDiagnostic("State show failed", msg.Error, msg.Output)
		var cmd tea.Cmd
		if m.toast != nil {
			cmd = m.toast.ShowError(fmt.Sprintf("State show failed: %v", msg.Error))
		}
		return m, cmd
	}

	// Show state in main area instead of full-screen view
	m.setDiagnostics(nil)
	if m.mainArea != nil {
		m.mainArea.SetStateContent(msg.Address, msg.Output)
	}

	return m, nil
}

func (m *Model) beginStateRm(address string) tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	stateEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	ctx := m.beginTrackedOperation(2 * time.Minute)

	return func() tea.Msg {
		backupPath, backupErr := m.backupStateToFile(ctx, stateEnv)
		if backupErr != nil {
			return StateRmCompleteMsg{Address: address, Error: fmt.Errorf("backup state: %w", backupErr)}
		}

		result, err := m.executor.StateRm(ctx, address, terraform.StateRmOptions{Env: stateEnv})
		if err != nil {
			return StateRmCompleteMsg{Address: address, BackupPath: backupPath, Error: err}
		}
		if result.Error != nil {
			return StateRmCompleteMsg{Address: address, BackupPath: backupPath, Output: result.Output, Error: result.Error, Result: result}
		}
		return StateRmCompleteMsg{Address: address, BackupPath: backupPath, Output: result.Output, Result: result}
	}
}

func (m *Model) handleStateRmComplete(msg StateRmCompleteMsg) (tea.Model, tea.Cmd) {
	m.finishTrackedOperation()

	m.stateMoveSource = ""
	m.stateMoveInput = ""
	m.pendingConfirmCmd = nil
	m.appendSessionLog("State removed", "terraform state rm "+msg.Address, stateMutationSessionOutput(msg.Output, msg.Error, msg.BackupPath))

	if msg.Error != nil {
		m.addErrorDiagnostic("State remove failed", msg.Error, msg.Output)
		cmd := m.toastError(fmt.Sprintf("State remove failed: %v", msg.Error))
		return m, cmd
	}

	m.setDiagnostics(nil)
	if strings.TrimSpace(msg.Output) != "" {
		m.setFormattedLogOutput(msg.Output)
	}
	cmds := make([]tea.Cmd, 0, 3)
	cmds = append(cmds, m.toastSuccess("State removed (backup: "+msg.BackupPath+")"))
	cmds = append(cmds, m.beginStateList())
	cmds = append(cmds, m.recordOperationCmd("state-rm", []string{msg.Address}, false, time.Now(), msg.Result, msg.Output, nil))
	return m, tea.Batch(cmds...)
}

func (m *Model) beginStateMv(source, destination string) tea.Cmd {
	if m.executor == nil {
		m.err = errors.New("terraform executor not configured")
		return nil
	}
	if m.isOperationRunning() {
		if m.toast != nil {
			return m.toast.ShowInfo("Operation already in progress")
		}
		return nil
	}
	stateEnv, err := m.prepareTerraformEnv()
	if err != nil {
		m.err = err
		return nil
	}

	m.cancelExecution()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	m.cancelFunc = cancel
	m.operationRunning = true

	return func() tea.Msg {
		backupPath, backupErr := m.backupStateToFile(ctx, stateEnv)
		if backupErr != nil {
			return StateMvCompleteMsg{Source: source, Destination: destination, Error: fmt.Errorf("backup state: %w", backupErr)}
		}

		result, err := m.executor.StateMv(ctx, source, destination, terraform.StateMvOptions{Env: stateEnv})
		if err != nil {
			return StateMvCompleteMsg{Source: source, Destination: destination, BackupPath: backupPath, Error: err}
		}
		if result.Error != nil {
			return StateMvCompleteMsg{Source: source, Destination: destination, BackupPath: backupPath, Output: result.Output, Error: result.Error, Result: result}
		}
		return StateMvCompleteMsg{Source: source, Destination: destination, BackupPath: backupPath, Output: result.Output, Result: result}
	}
}

func (m *Model) handleStateMvComplete(msg StateMvCompleteMsg) (tea.Model, tea.Cmd) {
	m.finishTrackedOperation()

	m.stateMoveSource = ""
	m.stateMoveInput = ""
	m.pendingConfirmCmd = nil
	cmdText := "terraform state mv " + msg.Source + " " + msg.Destination
	m.appendSessionLog("State moved", cmdText, stateMutationSessionOutput(msg.Output, msg.Error, msg.BackupPath))

	if msg.Error != nil {
		m.addErrorDiagnostic("State move failed", msg.Error, msg.Output)
		cmd := m.toastError(fmt.Sprintf("State move failed: %v", msg.Error))
		return m, cmd
	}

	m.setDiagnostics(nil)
	if strings.TrimSpace(msg.Output) != "" {
		m.setFormattedLogOutput(msg.Output)
	}
	cmds := make([]tea.Cmd, 0, 3)
	cmds = append(cmds, m.toastSuccess("State moved (backup: "+msg.BackupPath+")"))
	cmds = append(cmds, m.beginStateList())
	cmds = append(cmds, m.recordOperationCmd("state-mv", []string{msg.Source, msg.Destination}, false, time.Now(), msg.Result, msg.Output, nil))
	return m, tea.Batch(cmds...)
}

func stateMutationSessionOutput(output string, err error, backupPath string) string {
	trimmedOutput := strings.TrimSpace(output)
	if err != nil {
		result := strings.TrimSpace(err.Error())
		if trimmedOutput != "" {
			result = trimmedOutput
			if !strings.Contains(result, err.Error()) {
				result += "\n" + err.Error()
			}
		}
		if backupPath == "" {
			return result
		}
		return result + "\nbackup: " + backupPath
	}
	result := trimmedOutput
	if result == "" {
		result = "State operation complete"
	}
	if backupPath != "" {
		result += "\nbackup: " + backupPath
	}
	return result
}

func (m *Model) backupStateToFile(ctx context.Context, env []string) (string, error) {
	result, err := m.executor.StatePull(ctx, terraform.StatePullOptions{Env: env})
	if err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", result.Error
	}
	if strings.TrimSpace(result.Stdout) == "" {
		return "", errors.New("empty state pull output")
	}
	workDir := m.envWorkDir
	if m.executor != nil {
		workDir = m.executor.WorkDir()
	}
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	backupDir := filepath.Join(workDir, ".lazytf", "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("create state backup dir: %w", err)
	}
	backupPath := filepath.Join(backupDir, "state-"+time.Now().Format("20060102-150405")+".tfstate")
	if err := os.WriteFile(backupPath, []byte(result.Stdout), 0o600); err != nil {
		return "", fmt.Errorf("write state backup: %w", err)
	}
	return backupPath, nil
}

// showSelectedStateDetail loads and shows the currently selected state resource in the main area.
func (m *Model) showSelectedStateDetail() tea.Cmd {
	if m.stateListContent == nil {
		return nil
	}
	selected := m.stateListContent.GetSelected()
	if selected == nil {
		return nil
	}
	return m.beginStateShow(selected.Address)
}

func stateListSessionOutput(msg StateListCompleteMsg) string {
	if msg.Error != nil {
		return stateListErrorText(msg)
	}
	output := strings.TrimSpace(msg.Output)
	if output != "" {
		return output
	}
	if len(msg.Resources) == 0 {
		return "0 resources"
	}
	addresses := make([]string, len(msg.Resources))
	for i, r := range msg.Resources {
		addresses[i] = r.Address
	}
	return fmt.Sprintf("%d resources\n%s", len(msg.Resources), strings.Join(addresses, "\n"))
}

func stateListErrorText(msg StateListCompleteMsg) string {
	if msg.Error == nil {
		return ""
	}
	output := strings.TrimSpace(msg.Output)
	if output == "" {
		return msg.Error.Error()
	}
	if output == msg.Error.Error() {
		return output
	}
	return msg.Error.Error() + "\n\n" + output
}

func stateListResultOutput(result *terraform.ExecutionResult) string {
	if result == nil {
		return ""
	}
	stderr := strings.TrimSpace(result.Stderr)
	if stderr != "" {
		return stderr
	}
	stdout := strings.TrimSpace(result.Stdout)
	if stdout != "" {
		return stdout
	}
	return strings.TrimSpace(result.Output)
}

func parseStateListResources(stdout string) []terraform.StateResource {
	if strings.TrimSpace(stdout) == "" {
		return nil
	}
	resources := make([]terraform.StateResource, 0)
	for _, line := range strings.Split(stdout, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			resources = append(resources, terraform.StateResource{Address: trimmed})
		}
	}
	return resources
}

func containsStateAddress(resources []terraform.StateResource, address string) bool {
	for _, resource := range resources {
		if resource.Address == address {
			return true
		}
	}
	return false
}

func stateShowSessionOutput(msg StateShowCompleteMsg) string {
	if msg.Error != nil {
		output := strings.TrimSpace(msg.Output)
		if output == "" {
			return msg.Error.Error()
		}
		if output == msg.Error.Error() {
			return output
		}
		return msg.Error.Error() + "\n\n" + output
	}
	return msg.Output
}

func stateShowResultOutput(result *terraform.ExecutionResult) string {
	if result == nil {
		return ""
	}
	stderr := strings.TrimSpace(result.Stderr)
	if stderr != "" {
		return stderr
	}
	stdout := strings.TrimSpace(result.Stdout)
	if stdout != "" {
		return stdout
	}
	return strings.TrimSpace(result.Output)
}

func formatResultOutput(msg FormatCompleteMsg) string {
	if msg.ExecResult != nil {
		output := strings.TrimSpace(msg.ExecResult.Output)
		if output != "" {
			return output
		}
	}
	if msg.Error != nil {
		return msg.Error.Error()
	}
	return ""
}

func (m *Model) appendSessionLog(title, command, output string) {
	if m.commandLogPanel == nil {
		return
	}
	m.commandLogPanel.AppendSessionLog(title, command, output)
}

func (m *Model) prepareTerraformEnv() ([]string, error) {
	workDir := m.currentWorkDir()
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	tmpDir := filepath.Join(workDir, ".lazytf", "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	return []string{"TMPDIR=" + tmpDir}, nil
}

func (m *Model) currentWorkDir() string {
	workDir := m.envWorkDir
	if m.executor != nil {
		workDir = m.executor.WorkDir()
	}
	if strings.TrimSpace(workDir) == "" {
		return "."
	}
	return workDir
}

func (m *Model) clearSavedPlanState() {
	m.planFilePath = ""
	m.planRunFlags = nil
	m.applyRunFlags = nil
	m.planEnvironment = ""
	m.planWorkDir = ""
}

func (m *Model) clearPlanResultState() {
	m.setPlan(nil)
	m.lastPlanOutput = ""
	if m.planView != nil {
		m.planView.SetSummary(m.planSummary())
	}
}

func (m *Model) cancelExecution() {
	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}
}

func (m *Model) beginTrackedOperation(timeout time.Duration) context.Context {
	m.cancelExecution()
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	m.cancelFunc = cancel
	m.operationRunning = true
	return ctx
}

func (m *Model) finishTrackedOperation() {
	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}
	m.operationRunning = false
}

func (m *Model) handlePlanStart(msg PlanStartMsg) (tea.Model, tea.Cmd) {
	m.clearPlanResultState()
	if msg.Error != nil {
		m.clearSavedPlanState()
	}
	return m.handleOperationStart(
		msg.Error,
		&m.planRunning,
		"Failed to start terraform plan",
		"Plan failed to start",
		msg.Output,
		m.waitPlanCompleteCmd(msg.Result),
		m.streamPlanOutputCmd(),
	)
}

func (m *Model) handlePlanComplete(msg PlanCompleteMsg) (tea.Model, tea.Cmd) {
	m.planRunning = false
	m.finishTrackedOperation()
	m.outputChan = nil

	// Log to session history
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog("Planned", m.buildCommand("plan", m.planRunFlags, false), msg.Output)
	}

	if msg.Error != nil {
		// Mark progress indicator as failed
		if m.progressIndicator != nil {
			m.progressIndicator.Fail()
		}
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("Plan failed: %v", msg.Error))
		}
		m.clearSavedPlanState()
		// Clear operation state on plan failure to avoid stale resource states
		if m.operationState != nil {
			m.operationState.InitializeFromPlan(nil)
		}
		m.addErrorDiagnostic("Plan failed", msg.Error, msg.Output)
		// Route logs to command log panel
		m.setFormattedLogOutput(msg.Output)
		summary := m.flattenSummary(m.planSummary())
		recordCmd := m.recordOperationCmd("plan", m.planFlagsForRecord(), false, m.planStartedAt, msg.Result, msg.Output, msg.Error)
		notifyCmd := m.notifyOperationCmd("plan", summary, m.planStartedAt, msg.Result, msg.Error)
		return m, tea.Batch(recordCmd, notifyCmd)
	}

	// Reset progress indicator on success
	if m.progressIndicator != nil {
		m.progressIndicator.Reset()
	}

	if msg.Plan != nil {
		m.setPlan(msg.Plan)
		if m.operationState != nil {
			m.operationState.InitializeFromPlan(msg.Plan)
		}
		if m.planView != nil {
			m.planView.SetSummary(m.planSummary())
		}
	}
	m.planEnvironment = m.envCurrent
	m.planWorkDir = m.currentWorkDir()
	m.setDiagnostics(nil)
	if msg.Output != "" {
		m.lastPlanOutput = msg.Output
		// Route logs to command log panel
		m.setFormattedLogOutput(msg.Output)
	}
	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
	}

	// Only switch to diff mode if there are actual changes, otherwise stay on logs
	if m.mainArea != nil && m.hasChanges() {
		m.mainArea.SetMode(ModeDiff)
	}

	m.updateExecutionViewForStreaming()
	summary := m.flattenSummary(m.planSummary())
	recordCmd := m.recordOperationCmd("plan", m.planFlagsForRecord(), false, m.planStartedAt, msg.Result, msg.Output, nil)
	notifyCmd := m.notifyOperationCmd("plan", summary, m.planStartedAt, msg.Result, nil)
	return m, tea.Batch(recordCmd, notifyCmd)
}

func (m *Model) handleApplyStart(msg ApplyStartMsg) (tea.Model, tea.Cmd) {
	// Show status column during apply
	if m.resourceList != nil {
		m.resourceList.SetShowStatus(true)
	}
	return m.handleOperationStart(
		msg.Error,
		&m.applyRunning,
		"Failed to start terraform apply",
		"Apply failed to start",
		msg.Output,
		m.waitApplyCompleteCmd(msg.Result),
		m.streamApplyOutputCmd(),
	)
}

func (m *Model) handleApplyComplete(msg ApplyCompleteMsg) (tea.Model, tea.Cmd) {
	m.applyRunning = false
	m.finishTrackedOperation()
	m.outputChan = nil

	// Keep status column visible after apply - user can see final state

	// Keep MainArea in logs mode to show apply output (don't switch to diff)

	// Log to session history
	output := ""
	if msg.Result != nil {
		output = msg.Result.Output
	}
	if m.commandLogPanel != nil {
		m.commandLogPanel.AppendSessionLog("Applied", m.buildCommand("apply", m.applyFlagsForRecord(), true), output)
	}

	if msg.Error != nil || !msg.Success {
		// Mark progress indicator as failed
		if m.progressIndicator != nil {
			m.progressIndicator.Fail()
		}
		return m.handleApplyFailure(msg)
	}

	// Reset progress indicator on success
	if m.progressIndicator != nil {
		m.progressIndicator.Reset()
	}

	// Set full output to applyView so [0] Operation Logs shows everything
	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplySuccess)
		if msg.Result != nil && msg.Result.Output != "" {
			m.applyView.SetOutput(msg.Result.Output)
		}
	}
	summary := m.planSummary()
	// Route logs to command log panel
	parsed := ""
	if msg.Result != nil {
		parsed = utils.FormatLogOutput(msg.Result.Output)
	}
	if strings.TrimSpace(parsed) == "" {
		parsed = "Apply complete"
	}
	m.setDiagnostics(nil)
	m.setLogText(parsed)
	// Stay in main view with panel layout
	m.setPlan(&terraform.Plan{Resources: nil})
	m.clearSavedPlanState()
	m.updateExecutionViewForStreaming()

	// Build commands - recordHistoryCmd will record and reload entries
	// Always add explicit reload as safety measure to ensure UI is updated
	recordCmd := m.recordHistoryCmd(history.StatusSuccess, m.flattenSummary(summary), m.lastPlanOutput, msg.Result, nil)
	operationCmd := m.recordOperationCmd("apply", m.applyFlagsForRecord(), true, m.applyStartedAt, msg.Result, "", nil)
	reloadCmd := m.reloadHistoryCmd()
	notifyCmd := m.notifyOperationCmd("apply", m.flattenSummary(summary), m.applyStartedAt, msg.Result, nil)

	return m, tea.Batch(recordCmd, operationCmd, reloadCmd, notifyCmd)
}

func (m *Model) handleOperationStart(
	err error,
	running *bool,
	failureLine string,
	diagnosticSummary string,
	output <-chan string,
	waitCmd, streamCmd tea.Cmd,
) (tea.Model, tea.Cmd) {
	if err != nil {
		*running = false
		m.finishTrackedOperation()
		if m.applyView != nil {
			m.applyView.SetStatus(views.ApplyFailed)
			m.applyView.AppendLine(fmt.Sprintf("%s: %v", failureLine, err))
		}
		m.addErrorDiagnostic(diagnosticSummary, err, "")
		return m, nil
	}

	m.outputChan = output
	cmds := []tea.Cmd{waitCmd, streamCmd}
	if m.applyView != nil {
		cmds = append(cmds, m.applyView.Tick())
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleRefreshFailure(msg RefreshCompleteMsg) (tea.Model, tea.Cmd) {
	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplyFailed)
		if msg.Error != nil {
			m.applyView.AppendLine(fmt.Sprintf("Refresh failed: %v", msg.Error))
		}
	}
	// Route logs to command log panel.
	if msg.Result != nil {
		m.setFormattedLogOutput(msg.Result.Output)
	}
	if msg.Error != nil {
		output := ""
		if msg.Result != nil {
			output = msg.Result.Output
		}
		m.addErrorDiagnostic("Refresh failed", msg.Error, output)
	}
	opErr := msg.Error
	if opErr == nil && !msg.Success {
		opErr = errors.New("refresh failed")
	}
	m.updateExecutionViewForStreaming()
	recordCmd := m.recordOperationCmd("refresh", nil, true, m.refreshStartedAt, msg.Result, "", msg.Error)
	notifyCmd := m.notifyOperationCmd("refresh", m.flattenSummary(m.planSummary()), m.refreshStartedAt, msg.Result, opErr)
	return m, tea.Batch(recordCmd, notifyCmd)
}

func (m *Model) handleApplyFailure(msg ApplyCompleteMsg) (tea.Model, tea.Cmd) {
	// Clear plan-related state on apply failure
	m.clearSavedPlanState()

	// Set full output to applyView so [0] Operation Logs shows everything
	if m.applyView != nil {
		m.applyView.SetStatus(views.ApplyFailed)
		if msg.Result != nil && msg.Result.Output != "" {
			m.applyView.SetOutput(msg.Result.Output)
		}
	}
	// Route logs to command log panel.
	if msg.Result != nil {
		m.setFormattedLogOutput(msg.Result.Output)
		// Parse the full output for status updates - some lines (like Error:) may not
		// have been streamed but are in the final result
		if m.operationState != nil {
			for _, line := range strings.Split(msg.Result.Output, "\n") {
				m.operationState.ParseApplyLine(line)
			}
			if m.resourceList != nil {
				m.resourceList.Refresh()
			}
		}
	}
	output := ""
	if msg.Result != nil {
		output = msg.Result.Output
	}
	if msg.Error != nil {
		m.addErrorDiagnostic("Apply failed", msg.Error, output)
	} else if !msg.Success {
		m.addErrorDiagnostic("Apply failed", errors.New("apply failed"), output)
	}
	status := history.StatusFailed
	if errors.Is(msg.Error, context.Canceled) {
		status = history.StatusCanceled
	}
	opErr := msg.Error
	if opErr == nil && !msg.Success {
		opErr = errors.New("apply failed")
	}
	summary := m.flattenSummary(m.planSummary())
	m.updateExecutionViewForStreaming()

	// Build commands - recordHistoryCmd will record and reload entries
	// Always add explicit reload as safety measure to ensure UI is updated
	recordCmd := m.recordHistoryCmd(status, summary, m.lastPlanOutput, msg.Result, msg.Error)
	operationCmd := m.recordOperationCmd("apply", m.applyFlagsForRecord(), true, m.applyStartedAt, msg.Result, "", opErr)
	reloadCmd := m.reloadHistoryCmd()
	notifyCmd := m.notifyOperationCmd("apply", summary, m.applyStartedAt, msg.Result, opErr)

	return m, tea.Batch(recordCmd, operationCmd, reloadCmd, notifyCmd)
}

func (m *Model) showFormattedFiles(changedFiles []string) {
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
		m.updateLayout()
	}

	output := "Formatted files:\n" + strings.Join(changedFiles, "\n")
	m.setLogText(output)
}

func (m *Model) setLogText(text string) {
	if m.commandLogPanel != nil {
		m.commandLogPanel.SetLogText(text)
		return
	}
	if m.diagnosticsPanel != nil {
		m.diagnosticsPanel.SetLogText(text)
	}
}

func (m *Model) setFormattedLogOutput(raw string) {
	m.setLogText(utils.FormatLogOutput(raw))
}

func (m *Model) setDiagnostics(diagnostics []terraform.Diagnostic) {
	if m.commandLogPanel != nil {
		m.commandLogPanel.SetDiagnostics(diagnostics)
		return
	}
	if m.diagnosticsPanel != nil {
		m.diagnosticsPanel.SetDiagnostics(diagnostics)
	}
}

func (m *Model) toastError(message string) tea.Cmd {
	if m.toast == nil {
		return nil
	}
	return m.toast.ShowError(message)
}

func (m *Model) toastInfo(message string) tea.Cmd {
	if m.toast == nil {
		return nil
	}
	return m.toast.ShowInfo(message)
}

func (m *Model) toastSuccess(message string) tea.Cmd {
	if m.toast == nil {
		return nil
	}
	return m.toast.ShowSuccess(message)
}

func (m *Model) updateExecutionViewForStreaming() {
	if m.execView == viewPlanConfirm {
		return
	}
	// Don't interrupt history detail mode when showing in main area
	if m.mainArea != nil && m.mainArea.GetMode() == ModeHistoryDetail {
		return
	}
	if m.execView != viewMain {
		m.execView = viewMain
	}
}

func (m *Model) addErrorDiagnostic(summary string, err error, output string) {
	if err == nil {
		return
	}
	detail := err.Error()
	if strings.TrimSpace(output) != "" {
		detail = detail + "\n\n" + output
	}
	diag := terraform.Diagnostic{
		Severity: "error",
		Summary:  summary,
		Detail:   detail,
	}

	// Ensure command log is visible when errors occur
	if m.panelManager != nil {
		m.panelManager.SetCommandLogVisible(true)
	}

	if m.operationState != nil {
		m.operationState.AddDiagnostic(diag)
		// Route diagnostics to command log panel
		m.setDiagnostics(m.operationState.GetDiagnostics())
		return
	}
	// Route diagnostics to command log panel
	m.setDiagnostics([]terraform.Diagnostic{diag})
}

func (m *Model) streamPlanOutputCmd() tea.Cmd {
	return func() tea.Msg {
		if m.outputChan == nil {
			return nil
		}
		line, ok := <-m.outputChan
		if !ok {
			return nil
		}
		return PlanOutputMsg{Line: line}
	}
}

func (m *Model) streamApplyOutputCmd() tea.Cmd {
	return func() tea.Msg {
		if m.outputChan == nil {
			return nil
		}
		line, ok := <-m.outputChan
		if !ok {
			return nil
		}
		return ApplyOutputMsg{Line: line}
	}
}

func (m *Model) waitPlanCompleteCmd(result *terraform.ExecutionResult) tea.Cmd {
	return func() tea.Msg {
		if result == nil {
			return PlanCompleteMsg{Error: errors.New("plan execution result missing")}
		}
		<-result.Done()
		if result.Error != nil {
			return PlanCompleteMsg{Error: result.Error, Result: result, Output: result.Output}
		}

		output := result.Output
		if output == "" {
			output = result.Stdout
		}

		parseInput, showWarning := m.resolvePlanParseInput(output)
		output = appendWarningOutput(output, showWarning)

		textParser := tfparser.NewTextParser()
		plan, err := textParser.Parse(strings.NewReader(parseInput))
		if err != nil {
			return PlanCompleteMsg{Error: fmt.Errorf("parse plan output: %w", err), Result: result, Output: output}
		}
		return PlanCompleteMsg{Plan: plan, Result: result, Output: output}
	}
}

func (m *Model) resolvePlanParseInput(defaultOutput string) (string, string) {
	parseInput := defaultOutput
	if m.executor == nil || m.planFilePath == "" {
		return parseInput, ""
	}

	planEnv, err := m.prepareTerraformEnv()
	if err != nil {
		planEnv = nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	showResult, showErr := m.executor.Show(ctx, m.planFilePath, terraform.ShowOptions{Env: planEnv})
	cancel()

	if showErr == nil && showResult != nil && strings.TrimSpace(showResult.Output) != "" {
		return showResult.Output, ""
	}

	if showErr != nil {
		return parseInput, fmt.Sprintf("Warning: terraform show failed for %s: %v", m.planFilePath, showErr)
	}
	if showResult == nil {
		return parseInput, "Warning: terraform show returned no result for " + m.planFilePath
	}
	if strings.TrimSpace(showResult.Output) == "" {
		return parseInput, "Warning: terraform show returned empty output for " + m.planFilePath
	}

	return parseInput, ""
}

func appendWarningOutput(output, warning string) string {
	if strings.TrimSpace(warning) == "" {
		return output
	}
	if strings.TrimSpace(output) == "" {
		return warning
	}
	return output + "\n\n" + warning
}

func (m *Model) waitApplyCompleteCmd(result *terraform.ExecutionResult) tea.Cmd {
	return func() tea.Msg {
		if result == nil {
			return ApplyCompleteMsg{Success: false, Error: errors.New("apply execution result missing")}
		}
		<-result.Done()
		if result.Error != nil {
			return ApplyCompleteMsg{Success: false, Error: result.Error, Result: result}
		}
		return ApplyCompleteMsg{Success: true, Result: result}
	}
}

func (m *Model) setPlan(plan *terraform.Plan) {
	m.plan = plan
	// Hide status column when loading a new plan
	if m.resourceList != nil {
		m.resourceList.SetShowStatus(false)
	}
	if plan == nil {
		m.resourceList.SetResources(nil)
		return
	}
	m.resourceList.SetResources(plan.Resources)
	if m.operationState != nil {
		m.operationState.InitializeFromPlan(plan)
	}
}

func (m *Model) renderToast(message string, isError bool) string {
	if m.styles == nil {
		return ""
	}
	style := m.styles.Highlight
	if isError {
		style = m.styles.DiffRemove
	}
	content := style.Render(message)
	box := m.styles.Border.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *Model) clearToastCmd(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(_ time.Time) tea.Msg {
		return ClearToastMsg{}
	})
}

func (m *Model) flattenSummary(summary string) string {
	parts := strings.Fields(summary)
	return strings.Join(parts, " ")
}

func (m *Model) buildCommand(action string, flags []string, autoApprove bool) string {
	args := []string{action}
	args = append(args, flags...)
	if autoApprove && !containsFlag(args, "-auto-approve") {
		args = append(args, "-auto-approve")
	}
	return "terraform " + strings.Join(args, " ")
}

func containsFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func planOutputPath(flags []string) string {
	for i, flag := range flags {
		if flag == "-out" && i+1 < len(flags) {
			return flags[i+1]
		}
		if strings.HasPrefix(flag, "-out=") {
			value := strings.TrimPrefix(flag, "-out=")
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func (m *Model) planFlagsForRun() ([]string, string) {
	planFlags := append([]string{}, m.planFlags...)
	planFilePath := planOutputPath(planFlags)
	if planFilePath != "" {
		return planFlags, planFilePath
	}
	workDir := m.envWorkDir
	if m.executor != nil {
		workDir = m.executor.WorkDir()
	}
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	planFilePath = filepath.Join(workDir, ".lazytf", "tmp", "plan.tfplan")
	planFlags = append(planFlags, "-out="+planFilePath)
	return planFlags, planFilePath
}

func (m *Model) planFlagsForRecord() []string {
	if len(m.planRunFlags) > 0 {
		return m.planRunFlags
	}
	return m.planFlags
}

func (m *Model) applyFlagsForRun() ([]string, error) {
	flags := append([]string{}, m.applyFlags...)
	if strings.TrimSpace(m.planFilePath) == "" {
		return flags, nil
	}
	if m.planEnvironment != "" && m.planEnvironment != m.envCurrent {
		return nil, fmt.Errorf(
			"saved plan belongs to environment %q, current environment is %q. Run terraform plan again",
			m.planEnvironment,
			m.envCurrent,
		)
	}
	if m.planWorkDir != "" {
		currentWorkDir := filepath.Clean(m.currentWorkDir())
		savedPlanWorkDir := filepath.Clean(m.planWorkDir)
		if savedPlanWorkDir != currentWorkDir {
			return nil, fmt.Errorf(
				"saved plan belongs to workdir %q, current workdir is %q. Run terraform plan again",
				savedPlanWorkDir,
				currentWorkDir,
			)
		}
	}
	if _, err := os.Stat(m.planFilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("saved plan file missing at %q, run terraform plan again", m.planFilePath)
		}
		return nil, fmt.Errorf("check saved plan file %q: %w", m.planFilePath, err)
	}
	flags = removePlanFileArgs(flags)
	flags = append(flags, m.planFilePath)
	return flags, nil
}

func removePlanFileArgs(flags []string) []string {
	if len(flags) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(flags))
	for _, flag := range flags {
		if isPlanFileArg(flag) {
			continue
		}
		filtered = append(filtered, flag)
	}
	return filtered
}

func isPlanFileArg(arg string) bool {
	trimmed := strings.TrimSpace(arg)
	if trimmed == "" || strings.HasPrefix(trimmed, "-") {
		return false
	}
	return strings.HasSuffix(trimmed, ".tfplan")
}

func (m *Model) applyFlagsForRecord() []string {
	if len(m.applyRunFlags) > 0 {
		return m.applyRunFlags
	}
	return m.applyFlags
}

func (m *Model) planSummary() string {
	if m.plan == nil {
		return noChangesSummary
	}
	create := m.countResourcesByAction(terraform.ActionCreate)
	update := m.countResourcesByAction(terraform.ActionUpdate)
	deleteCount := m.countResourcesByAction(terraform.ActionDelete)
	replace := m.countResourcesByAction(terraform.ActionReplace)

	parts := []string{
		fmt.Sprintf("+%d", create),
		fmt.Sprintf("~%d", update),
		fmt.Sprintf("-%d", deleteCount),
	}
	if replace > 0 {
		parts = append(parts, fmt.Sprintf("±%d", replace))
	}
	return strings.Join(parts, " ")
}

// planSummaryVerbose returns a multi-line summary with labels and colors for the confirm dialog.
func (m *Model) planSummaryVerbose() string {
	if m.plan == nil {
		return noChangesSummary
	}
	create := m.countResourcesByAction(terraform.ActionCreate)
	update := m.countResourcesByAction(terraform.ActionUpdate)
	deleteCount := m.countResourcesByAction(terraform.ActionDelete)
	replace := m.countResourcesByAction(terraform.ActionReplace)

	lines := []string{
		m.styles.DiffAdd.Render("+") + fmt.Sprintf(" %d to create", create),
		m.styles.DiffChange.Render("~") + fmt.Sprintf(" %d to update", update),
		m.styles.DiffRemove.Render("-") + fmt.Sprintf(" %d to destroy", deleteCount),
	}
	if replace > 0 {
		lines = append(lines, m.styles.DiffChange.Render("±")+fmt.Sprintf(" %d to replace", replace))
	}
	return strings.Join(lines, "\n")
}

// hasChanges returns true if the current plan has any resources that will be modified.
func (m *Model) hasChanges() bool {
	if m.plan == nil {
		return false
	}
	create := m.countResourcesByAction(terraform.ActionCreate)
	update := m.countResourcesByAction(terraform.ActionUpdate)
	deleteCount := m.countResourcesByAction(terraform.ActionDelete)
	replace := m.countResourcesByAction(terraform.ActionReplace)
	return create+update+deleteCount+replace > 0
}
