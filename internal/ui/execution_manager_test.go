package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/environment"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/notifications"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/ui/components"
	"github.com/ushiradineth/lazytf/internal/ui/testutil"
)

// setupMockExecutor creates a mock executor with a valid temp workdir.
func setupMockExecutor(t *testing.T) *testutil.MockExecutor {
	t.Helper()
	mock := testutil.NewMockExecutor()
	mock.MockWorkDir = t.TempDir()
	return mock
}

type recordingNotifier struct {
	events []notifications.OperationEvent
	err    error
}

func (n *recordingNotifier) Notify(_ context.Context, event notifications.OperationEvent) error {
	n.events = append(n.events, event)
	return n.err
}

// ============================================================================
// beginRefresh tests
// ============================================================================

func TestBeginRefreshWithMockSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.RefreshOutput = testutil.NewMockOutputChannel("Refreshing...", "Done")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginRefresh()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Verify synchronous state changes
	if !m.refreshRunning {
		t.Error("expected refreshRunning to be true")
	}
}

func TestBeginRefreshWithMockNilExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = nil
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginRefresh()
	if cmd != nil {
		t.Error("expected nil cmd for nil executor")
	}

	if m.err == nil {
		t.Error("expected error to be set on model")
	}
}

func TestBeginRefreshWithMockAlreadyRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.refreshRunning = true

	_ = m.beginRefresh()
	// Should return a toast cmd or nil, but not call executor
	if mock.RefreshCalls != 0 {
		t.Error("expected no refresh calls when already running")
	}
}

func TestBeginRefreshWithMockPlanRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.planRunning = true

	_ = m.beginRefresh()
	if mock.RefreshCalls != 0 {
		t.Error("expected no refresh calls when plan is running")
	}
}

func TestBeginRefreshWithMockApplyRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.applyRunning = true

	_ = m.beginRefresh()
	if mock.RefreshCalls != 0 {
		t.Error("expected no refresh calls when apply is running")
	}
}

// ============================================================================
// beginValidate tests
// ============================================================================

func TestBeginValidateWithMockSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.ValidateResult = testutil.NewMockResult(`{
		"format_version": "1.0",
		"valid": true,
		"error_count": 0,
		"warning_count": 0,
		"diagnostics": []
	}`, 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginValidate()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	// Note: beginValidate returns a batched command (validate + progress indicator)
	// The executor is called when the batched command is executed
}

func TestBeginValidateWithMockNilExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = nil

	cmd := m.beginValidate()
	if cmd != nil {
		t.Error("expected nil cmd for nil executor")
	}

	if m.err == nil {
		t.Error("expected error to be set on model")
	}
}

func TestBeginValidateWithMockAlreadyRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.planRunning = true

	_ = m.beginValidate()
	if mock.ValidateCalls != 0 {
		t.Error("expected no validate calls when plan is running")
	}
}

func TestBeginValidateWithMockApplyRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.applyRunning = true

	_ = m.beginValidate()
	if mock.ValidateCalls != 0 {
		t.Error("expected no validate calls when apply is running")
	}
}

func TestBeginValidateWithMockRefreshRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.refreshRunning = true

	_ = m.beginValidate()
	if mock.ValidateCalls != 0 {
		t.Error("expected no validate calls when refresh is running")
	}
}

// ============================================================================
// beginFormat tests
// ============================================================================

func TestBeginFormatWithMockSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.FormatResult = testutil.NewMockResult("main.tf\nvariables.tf", 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginFormat()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	// Note: beginFormat returns a batched command (format + progress indicator)
}

func TestBeginFormatWithMockNilExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = nil

	cmd := m.beginFormat()
	if cmd != nil {
		t.Error("expected nil cmd for nil executor")
	}

	if m.err == nil {
		t.Error("expected error to be set on model")
	}
}

func TestBeginFormatWithMockAlreadyRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.applyRunning = true

	_ = m.beginFormat()
	if mock.FormatCalls != 0 {
		t.Error("expected no format calls when apply is running")
	}
}

func TestBeginFormatWithMockPlanRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.planRunning = true

	_ = m.beginFormat()
	if mock.FormatCalls != 0 {
		t.Error("expected no format calls when plan is running")
	}
}

func TestBeginFormatWithMockRefreshRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.refreshRunning = true

	_ = m.beginFormat()
	if mock.FormatCalls != 0 {
		t.Error("expected no format calls when refresh is running")
	}
}

// ============================================================================
// beginStateList tests
// ============================================================================

func TestBeginStateListWithMockSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateListResult = testutil.NewMockResult("aws_instance.web\naws_s3_bucket.logs", 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateList()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	// Note: beginStateList returns a batched command (stateList + progress indicator)
}

func TestBeginStateListWithMockNilExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = nil

	cmd := m.beginStateList()
	if cmd != nil {
		t.Error("expected nil cmd for nil executor")
	}

	if m.err == nil {
		t.Error("expected error to be set on model")
	}
}

func TestBeginStateListWithMockAlreadyRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.refreshRunning = true

	_ = m.beginStateList()
	if mock.StateListCalls != 0 {
		t.Error("expected no state list calls when refresh is running")
	}
}

func TestBeginStateListWithMockPlanRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.planRunning = true

	_ = m.beginStateList()
	if mock.StateListCalls != 0 {
		t.Error("expected no state list calls when plan is running")
	}
}

func TestBeginStateListWithMockApplyRunning(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.applyRunning = true

	_ = m.beginStateList()
	if mock.StateListCalls != 0 {
		t.Error("expected no state list calls when apply is running")
	}
}

// ============================================================================
// beginStateShow tests
// Note: beginStateShow returns a single command (not batched), so we can
// execute it and verify the executor is called.
// ============================================================================

func TestBeginStateShowWithMockSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateShowResult = testutil.NewMockResult(`# aws_instance.example:
resource "aws_instance" "example" {
    ami = "ami-12345"
}`, 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateShow("aws_instance.example")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Execute the command to trigger executor call
	msg := cmd()

	// Verify the executor was called
	if mock.StateShowCalls != 1 {
		t.Errorf("expected 1 state show call, got %d", mock.StateShowCalls)
	}

	if mock.LastStateShowAddr != "aws_instance.example" {
		t.Errorf("unexpected address: %s", mock.LastStateShowAddr)
	}

	// Verify message type
	completeMsg, ok := msg.(StateShowCompleteMsg)
	if !ok {
		t.Fatalf("expected StateShowCompleteMsg, got %T", msg)
	}
	if completeMsg.Error != nil {
		t.Errorf("unexpected error: %v", completeMsg.Error)
	}
	if completeMsg.Output == "" {
		t.Error("expected non-empty output")
	}
}

func TestBeginStateShowWithMockNilExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = nil

	cmd := m.beginStateShow("aws_instance.example")
	if cmd != nil {
		t.Error("expected nil cmd for nil executor")
	}

	if m.err == nil {
		t.Error("expected error to be set on model")
	}
}

func TestBeginStateShowWithMockExecutorError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateShowErr = errors.New("resource not found")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateShow("nonexistent.resource")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Execute the command to trigger executor call
	msg := cmd()

	// Verify the executor was called
	if mock.StateShowCalls != 1 {
		t.Errorf("expected 1 state show call, got %d", mock.StateShowCalls)
	}

	// Verify error message was returned
	completeMsg, ok := msg.(StateShowCompleteMsg)
	if !ok {
		t.Fatalf("expected StateShowCompleteMsg, got %T", msg)
	}
	if completeMsg.Error == nil {
		t.Error("expected error in message")
	}
}

func TestBeginStateShowWithMockResultErrorIncludesOutput(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateShowResult = testutil.NewMockErrorResult("resource not found in state", errors.New("exit status 1"))

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateShow("nonexistent.resource")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	completeMsg, ok := msg.(StateShowCompleteMsg)
	if !ok {
		t.Fatalf("expected StateShowCompleteMsg, got %T", msg)
	}
	if completeMsg.Error == nil {
		t.Fatal("expected error in message")
	}
	if !strings.Contains(completeMsg.Output, "resource not found in state") {
		t.Fatalf("expected stderr output, got %q", completeMsg.Output)
	}
}

func TestBeginStateShowWithMockDifferentAddresses(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	testCases := []string{
		"aws_instance.web",
		"module.vpc.aws_subnet.main",
		"aws_s3_bucket.data[0]",
	}

	for _, addr := range testCases {
		mock.Reset()
		mock.MockWorkDir = t.TempDir()

		cmd := m.beginStateShow(addr)
		if cmd == nil {
			t.Fatalf("expected non-nil cmd for address %s", addr)
		}

		_ = cmd()
		m.handleStateShowComplete(StateShowCompleteMsg{Address: addr, Output: "ok"})

		if mock.LastStateShowAddr != addr {
			t.Errorf("expected address %s, got %s", addr, mock.LastStateShowAddr)
		}
	}
}

// ============================================================================
// streamRefreshOutputCmd and waitRefreshCompleteCmd tests
// ============================================================================

func TestStreamRefreshOutputCmdWithChannel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	outputChan := make(chan string, 3)
	outputChan <- "line1"
	outputChan <- "line2"
	close(outputChan)
	m.outputChan = outputChan

	cmd := m.streamRefreshOutputCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	outputMsg, ok := msg.(RefreshOutputMsg)
	if !ok {
		t.Fatalf("expected RefreshOutputMsg, got %T", msg)
	}

	if outputMsg.Line != "line1" {
		t.Errorf("expected line1, got %s", outputMsg.Line)
	}
}

func TestStreamRefreshOutputCmdNilChannel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.outputChan = nil

	// streamRefreshOutputCmd always returns a command,
	// but the command returns nil when channel is nil
	cmd := m.streamRefreshOutputCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// When executed with nil channel, should return nil
	msg := cmd()
	if msg != nil {
		t.Errorf("expected nil message for nil channel, got %T", msg)
	}
}

func TestStreamRefreshOutputCmdClosedChannel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	// Create and immediately close channel
	outputChan := make(chan string)
	close(outputChan)
	m.outputChan = outputChan

	cmd := m.streamRefreshOutputCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// When executed with closed channel, should return nil
	msg := cmd()
	if msg != nil {
		t.Errorf("expected nil message for closed channel, got %T", msg)
	}
}

func TestWaitRefreshCompleteCmdSuccess(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	result := testutil.NewMockResult("Refresh complete", 0)

	cmd := m.waitRefreshCompleteCmd(result)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	completeMsg, ok := msg.(RefreshCompleteMsg)
	if !ok {
		t.Fatalf("expected RefreshCompleteMsg, got %T", msg)
	}

	if !completeMsg.Success {
		t.Error("expected success=true")
	}
}

func TestWaitRefreshCompleteCmdNilResult(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	// waitRefreshCompleteCmd always returns a command,
	// but returns error message when result is nil
	cmd := m.waitRefreshCompleteCmd(nil)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	completeMsg, ok := msg.(RefreshCompleteMsg)
	if !ok {
		t.Fatalf("expected RefreshCompleteMsg, got %T", msg)
	}

	if completeMsg.Success {
		t.Error("expected success=false for nil result")
	}
	if completeMsg.Error == nil {
		t.Error("expected error for nil result")
	}
}

func TestWaitRefreshCompleteCmdWithError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	result := testutil.NewMockErrorResult("Error refreshing", errors.New("refresh failed"))

	cmd := m.waitRefreshCompleteCmd(result)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	completeMsg, ok := msg.(RefreshCompleteMsg)
	if !ok {
		t.Fatalf("expected RefreshCompleteMsg, got %T", msg)
	}

	if completeMsg.Success {
		t.Error("expected success=false")
	}
	if completeMsg.Error == nil {
		t.Error("expected non-nil error")
	}
}

// ============================================================================
// Mock executor helper tests
// ============================================================================

func TestNewMockExecutorDefaults(t *testing.T) {
	mock := testutil.NewMockExecutor()

	if mock.MockWorkDir != "/mock/workdir" {
		t.Errorf("unexpected default workdir: %s", mock.MockWorkDir)
	}

	if mock.VersionResult != "1.5.0" {
		t.Errorf("unexpected default version: %s", mock.VersionResult)
	}
}

func TestMockExecutorWorkDir(t *testing.T) {
	mock := testutil.NewMockExecutor()
	mock.MockWorkDir = "/custom/workdir"

	if mock.WorkDir() != "/custom/workdir" {
		t.Errorf("unexpected workdir: %s", mock.WorkDir())
	}
}

func TestMockExecutorVersion(t *testing.T) {
	mock := testutil.NewMockExecutor()

	version, err := mock.Version()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if version != "1.5.0" {
		t.Errorf("unexpected version: %s", version)
	}

	// Test version error
	mock.VersionErr = errors.New("version error")
	_, err = mock.Version()
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockExecutorReset(t *testing.T) {
	mock := setupMockExecutor(t)

	// Manually increment some counters
	mock.ValidateCalls = 5
	mock.FormatCalls = 3
	mock.StateShowCalls = 2
	mock.LastStateShowAddr = "test.resource"

	// Reset
	mock.Reset()

	if mock.ValidateCalls != 0 {
		t.Errorf("expected ValidateCalls=0, got %d", mock.ValidateCalls)
	}
	if mock.FormatCalls != 0 {
		t.Errorf("expected FormatCalls=0, got %d", mock.FormatCalls)
	}
	if mock.StateShowCalls != 0 {
		t.Errorf("expected StateShowCalls=0, got %d", mock.StateShowCalls)
	}
	if mock.LastStateShowAddr != "" {
		t.Errorf("expected LastStateShowAddr='', got %s", mock.LastStateShowAddr)
	}
}

func TestNewMockResult(t *testing.T) {
	result := testutil.NewMockResult("test output", 0)

	if result.Stdout != "test output" {
		t.Errorf("unexpected stdout: %s", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Errorf("unexpected exit code: %d", result.ExitCode)
	}

	// Wait for result to finish
	<-result.Done()
}

func TestNewMockErrorResult(t *testing.T) {
	result := testutil.NewMockErrorResult("error output", errors.New("test error"))

	if result.Stderr != "error output" {
		t.Errorf("unexpected stderr: %s", result.Stderr)
	}
	if result.ExitCode != 1 {
		t.Errorf("unexpected exit code: %d", result.ExitCode)
	}
	if result.Error == nil {
		t.Error("expected error to be set")
	}

	// Wait for result to finish
	<-result.Done()
}

func TestNewMockOutputChannel(t *testing.T) {
	ch := testutil.NewMockOutputChannel("line1", "line2", "line3")

	lines := make([]string, 0, 3)
	for line := range ch {
		lines = append(lines, line)
	}

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

// ============================================================================
// handleValidateComplete tests
// ============================================================================

func TestHandleValidateCompleteValidConfig(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ValidateCompleteMsg{
		Result: &terraform.ValidateResult{
			Valid:        true,
			ErrorCount:   0,
			WarningCount: 0,
		},
		RawOutput: `{"valid":true}`,
	}

	model, cmd := m.handleValidateComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// Command may be nil or a toast command
	_ = cmd
}

func TestHandleValidateCompleteWithError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ValidateCompleteMsg{
		Error:     errors.New("validation failed"),
		RawOutput: "error output",
	}

	model, _ := m.handleValidateComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleValidateCompleteInvalidConfig(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ValidateCompleteMsg{
		Result: &terraform.ValidateResult{
			Valid:        false,
			ErrorCount:   2,
			WarningCount: 1,
			Diagnostics: []terraform.Diagnostic{
				{Severity: "error", Summary: "Missing required argument"},
				{Severity: "error", Summary: "Invalid resource name"},
				{Severity: "warning", Summary: "Deprecated attribute"},
			},
		},
		RawOutput: `{"valid":false}`,
	}

	model, _ := m.handleValidateComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleValidateCompleteNilResult(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ValidateCompleteMsg{
		Result: nil,
	}

	model, _ := m.handleValidateComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// handleFormatComplete tests
// ============================================================================

func TestHandleFormatCompleteWithFiles(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := FormatCompleteMsg{
		ChangedFiles: []string{"main.tf", "variables.tf"},
	}

	model, _ := m.handleFormatComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleFormatCompleteNoChanges(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := FormatCompleteMsg{
		ChangedFiles: []string{},
	}

	model, _ := m.handleFormatComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleFormatCompleteWithError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := FormatCompleteMsg{
		Error: errors.New("format failed"),
	}

	model, _ := m.handleFormatComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// handleStateListComplete tests
// ============================================================================

func TestHandleStateListCompleteWithResources(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateListCompleteMsg{
		Resources: []terraform.StateResource{
			{Address: "aws_instance.web", ResourceType: "aws_instance", Name: "web"},
			{Address: "aws_s3_bucket.logs", ResourceType: "aws_s3_bucket", Name: "logs"},
		},
	}

	model, _ := m.handleStateListComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleStateListCompleteEmpty(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateListCompleteMsg{
		Resources: []terraform.StateResource{},
	}

	model, _ := m.handleStateListComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleStateListCompleteEmptyClearsStaleStateDetails(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea == nil {
		t.Fatal("expected non-nil main area")
	}
	m.mainArea.SetStateContent("null_resource.example", "stale content")

	msg := StateListCompleteMsg{Resources: []terraform.StateResource{}}
	model, _ := m.handleStateListComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}

	view := m.mainArea.View()
	if !strings.Contains(view, "No resources in state. Press 'r' to refresh.") {
		t.Fatal("expected main area to show empty state message")
	}
}

func TestHandleStateListCompleteWithError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateListCompleteMsg{
		Error: errors.New("no state file"),
	}

	model, _ := m.handleStateListComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestStateListSessionOutputIncludesStderrOnError(t *testing.T) {
	msg := StateListCompleteMsg{
		Error:  errors.New("exit status 1"),
		Output: "No state file was found\nRun terraform init first",
	}

	out := stateListSessionOutput(msg)
	if !strings.Contains(out, "No state file was found") {
		t.Fatalf("expected stderr details in session output, got %q", out)
	}
}

func TestStateListResultOutputPrefersStderr(t *testing.T) {
	result := &terraform.ExecutionResult{
		Stdout: "stdout content",
		Stderr: "stderr content",
		Output: "combined content",
	}

	out := stateListResultOutput(result)
	if out != "stderr content" {
		t.Fatalf("expected stderr content, got %q", out)
	}
}

// ============================================================================
// handleStateShowComplete tests
// ============================================================================

func TestHandleStateShowCompleteWithOutput(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateShowCompleteMsg{
		Address: "aws_instance.example",
		Output: `# aws_instance.example:
resource "aws_instance" "example" {
    ami = "ami-12345"
}`,
	}

	model, _ := m.handleStateShowComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleStateShowCompleteResourceNotFound(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := StateShowCompleteMsg{
		Address: "nonexistent.resource",
		Error:   errors.New("resource not found"),
	}

	model, _ := m.handleStateShowComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestStateShowSessionOutputIncludesErrorOutput(t *testing.T) {
	msg := StateShowCompleteMsg{
		Address: "nonexistent.resource",
		Error:   errors.New("exit status 1"),
		Output:  "resource does not exist",
	}

	out := stateShowSessionOutput(msg)
	if !strings.Contains(out, "resource does not exist") {
		t.Fatalf("expected stderr details in session output, got %q", out)
	}
}

func TestBeginStateRmCreatesBackupAndRunsRemove(t *testing.T) {
	mock := setupMockExecutor(t)
	workDir := t.TempDir()
	mock.MockWorkDir = workDir
	mock.StatePullResult = testutil.NewMockResult(`{"version":4}`, 0)
	mock.StateRmResult = testutil.NewMockResult("Removed", 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateRm("null_resource.example")
	if cmd == nil {
		t.Fatal("expected non-nil state rm cmd")
	}
	raw := cmd()
	msg, ok := raw.(StateRmCompleteMsg)
	if !ok {
		t.Fatalf("expected StateRmCompleteMsg, got %T", raw)
	}
	if msg.Error != nil {
		t.Fatalf("unexpected state rm error: %v", msg.Error)
	}
	if msg.BackupPath == "" {
		t.Fatal("expected backup path")
	}
	if _, err := os.Stat(msg.BackupPath); err != nil {
		t.Fatalf("expected backup file to exist: %v", err)
	}
	if mock.StatePullCalls != 1 {
		t.Fatalf("expected one state pull call, got %d", mock.StatePullCalls)
	}
	if mock.StateRmCalls != 1 {
		t.Fatalf("expected one state rm call, got %d", mock.StateRmCalls)
	}
}

func TestBeginStateMvCreatesBackupAndRunsMove(t *testing.T) {
	mock := setupMockExecutor(t)
	workDir := t.TempDir()
	mock.MockWorkDir = workDir
	mock.StatePullResult = testutil.NewMockResult(`{"version":4}`, 0)
	mock.StateMvResult = testutil.NewMockResult("Moved", 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateMv("null_resource.old", "null_resource.new")
	if cmd == nil {
		t.Fatal("expected non-nil state mv cmd")
	}
	raw := cmd()
	msg, ok := raw.(StateMvCompleteMsg)
	if !ok {
		t.Fatalf("expected StateMvCompleteMsg, got %T", raw)
	}
	if msg.Error != nil {
		t.Fatalf("unexpected state mv error: %v", msg.Error)
	}
	if msg.BackupPath == "" {
		t.Fatal("expected backup path")
	}
	if _, err := os.Stat(msg.BackupPath); err != nil {
		t.Fatalf("expected backup file to exist: %v", err)
	}
	if mock.StatePullCalls != 1 {
		t.Fatalf("expected one state pull call, got %d", mock.StatePullCalls)
	}
	if mock.StateMvCalls != 1 {
		t.Fatalf("expected one state mv call, got %d", mock.StateMvCalls)
	}
}

// ============================================================================
// handleRefreshComplete tests
// ============================================================================

func TestHandleRefreshCompleteSuccessful(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.refreshRunning = true

	msg := RefreshCompleteMsg{
		Success: true,
		Result:  testutil.NewMockResult("Refresh complete", 0),
	}

	model, _ := m.handleRefreshComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// Verify refreshRunning is reset
	if m.refreshRunning {
		t.Error("expected refreshRunning to be false after completion")
	}
}

func TestHandleRefreshCompleteWithError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.refreshRunning = true

	msg := RefreshCompleteMsg{
		Success: false,
		Error:   errors.New("refresh failed"),
	}

	model, _ := m.handleRefreshComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// handleRequestApply tests
// ============================================================================

func TestHandleRequestApplyWhenPlanRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true

	model, _, handled := m.handleRequestApply()
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
}

func TestHandleRequestApplyWhenApplyRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.applyRunning = true

	model, _, handled := m.handleRequestApply()
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
}

func TestHandleRequestApplyNoPlanLoaded(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.plan = nil

	model, _, handled := m.handleRequestApply()
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
}

func TestHandleRequestApplyWithLoadedPlan(t *testing.T) {
	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Change: &terraform.Change{Actions: []string{"create"}}},
		},
	}
	m := NewExecutionModel(plan, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _, handled := m.handleRequestApply()
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
}

// ============================================================================
// buildEnvironmentCommand tests
// ============================================================================

func TestBuildEnvCommandWorkspaceStrategy(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	env := environment.Environment{
		Strategy: environment.StrategyWorkspace,
		Name:     "production",
	}

	cmd := m.buildEnvironmentCommand(env)
	expected := "terraform workspace select production"
	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

func TestBuildEnvCommandFolderStrategy(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	env := environment.Environment{
		Strategy: environment.StrategyFolder,
		Path:     "/path/to/env",
	}

	cmd := m.buildEnvironmentCommand(env)
	expected := "cd /path/to/env"
	if cmd != expected {
		t.Errorf("expected %q, got %q", expected, cmd)
	}
}

// ============================================================================
// showFormattedFiles tests
// ============================================================================

func TestShowFormattedFilesEmptyList(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic with empty list
	m.showFormattedFiles([]string{})
}

func TestShowFormattedFilesWithFiles(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic with files
	m.showFormattedFiles([]string{"main.tf", "variables.tf", "outputs.tf"})
}

// ============================================================================
// toastError/Info/Success tests
// ============================================================================

func TestToastErrorNoToastComponent(t *testing.T) {
	m := NewModel(nil)
	m.toast = nil

	cmd := m.toastError("error message")
	if cmd != nil {
		t.Error("expected nil cmd when toast is nil")
	}
}

func TestToastInfoNoToastComponent(t *testing.T) {
	m := NewModel(nil)
	m.toast = nil

	cmd := m.toastInfo("info message")
	if cmd != nil {
		t.Error("expected nil cmd when toast is nil")
	}
}

func TestToastSuccessNoToastComponent(t *testing.T) {
	m := NewModel(nil)
	m.toast = nil

	cmd := m.toastSuccess("success message")
	if cmd != nil {
		t.Error("expected nil cmd when toast is nil")
	}
}

func TestToastErrorWithToastComponent(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.toastError("error message")
	// Toast exists, so cmd should be non-nil
	if cmd == nil {
		t.Error("expected non-nil cmd when toast exists")
	}
}

func TestToastInfoWithToastComponent(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.toastInfo("info message")
	if cmd == nil {
		t.Error("expected non-nil cmd when toast exists")
	}
}

func TestToastSuccessWithToastComponent(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.toastSuccess("success message")
	if cmd == nil {
		t.Error("expected non-nil cmd when toast exists")
	}
}

// ============================================================================
// addErrorDiagnostic tests
// ============================================================================

func TestAddErrorDiagnosticBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic
	m.addErrorDiagnostic("Test error", errors.New("test error"), "")
}

func TestAddErrorDiagnosticWithOutput(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic
	m.addErrorDiagnostic("Test error", errors.New("test error"), "additional output")
}

func TestAddErrorDiagnosticNilError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic with nil error
	m.addErrorDiagnostic("Test error", nil, "")
}

// ============================================================================
// prepareTerraformEnv tests
// ============================================================================

func TestPrepareTerraformEnvWithExecutor(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	env, err := m.prepareTerraformEnv()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(env) == 0 {
		t.Error("expected non-empty env")
	}
}

func TestPrepareTerraformEnvNilExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = nil
	m.envWorkDir = t.TempDir()
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	env, err := m.prepareTerraformEnv()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(env) == 0 {
		t.Error("expected non-empty env")
	}
}

// ============================================================================
// cancelExecution tests
// ============================================================================

func TestCancelExecutionNilFunc(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.cancelFunc = nil

	// Should not panic
	m.cancelExecution()
}

func TestCancelExecutionWithFunc(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	called := false
	m.cancelFunc = func() { called = true }

	m.cancelExecution()

	if !called {
		t.Error("expected cancel function to be called")
	}
	if m.cancelFunc != nil {
		t.Error("expected cancelFunc to be nil after cancel")
	}
}

// ============================================================================
// initHistory tests
// ============================================================================

func TestInitHistoryDisabledExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cfg := ExecutionConfig{
		HistoryEnabled: false,
	}

	// Should return early without error
	m.initHistory(cfg)

	if m.historyStore != nil {
		t.Error("expected historyStore to remain nil when disabled")
	}
}

func TestInitHistoryWithProvidedStore(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Create a temp directory for the test store
	tmpDir := t.TempDir()
	store, err := history.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}
	defer store.Close()

	cfg := ExecutionConfig{
		HistoryEnabled: true,
		HistoryStore:   store,
	}

	m.initHistory(cfg)

	if m.historyStore != store {
		t.Error("expected historyStore to be set to provided store")
	}
}

// ============================================================================
// reloadHistoryCmd tests
// ============================================================================

func TestReloadHistoryCmdNilStoreExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.historyStore = nil

	cmd := m.reloadHistoryCmd()
	if cmd != nil {
		t.Error("expected nil cmd when historyStore is nil")
	}
}

func TestReloadHistoryCmdWithStore(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Create a temp directory for the test store
	tmpDir := t.TempDir()
	store, err := history.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}
	defer store.Close()

	m.historyStore = store

	cmd := m.reloadHistoryCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd when historyStore is set")
	}

	// Execute the command
	msg := cmd()
	historyMsg, ok := msg.(HistoryLoadedMsg)
	if !ok {
		t.Fatalf("expected HistoryLoadedMsg, got %T", msg)
	}

	// Empty store should return empty entries without error
	if historyMsg.Error != nil {
		t.Errorf("unexpected error: %v", historyMsg.Error)
	}
}

// ============================================================================
// handleEnvironmentChanged tests
// ============================================================================

func TestHandleEnvironmentChangedBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Create a valid temp directory for the environment
	tmpDir := t.TempDir()
	msg := components.EnvironmentChangedMsg{
		Environment: environment.Environment{
			Strategy: environment.StrategyFolder,
			Path:     tmpDir,
			Name:     "test-env",
		},
	}

	model, _ := m.handleEnvironmentChanged(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// updateHistoryDetailContentWithOperations tests
// ============================================================================

func TestUpdateHistoryDetailContentWithOperationsNilMainArea(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.mainArea = nil

	entry := history.Entry{
		ID:      1,
		Summary: "test apply",
	}

	// Should not panic with nil mainArea
	m.updateHistoryDetailContentWithOperations(entry, nil)
}

func TestUpdateHistoryDetailContentWithOperationsBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	entry := history.Entry{
		ID:      1,
		Summary: "test apply",
		WorkDir: "/path/to/workdir",
		Status:  "success",
	}

	operations := []history.OperationEntry{
		{
			ID:      1,
			Action:  "plan",
			Command: "terraform plan",
			Status:  history.StatusSuccess,
			Summary: "Plan: 1 to add, 0 to change, 0 to destroy",
		},
	}

	// Should not panic
	m.updateHistoryDetailContentWithOperations(entry, operations)
}

// ============================================================================
// handleStateTabKey tests
// ============================================================================

func TestHandleStateTabKeyNotExecutionModeExec(t *testing.T) {
	m := NewModel(nil)
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executionMode = false

	cmd, handled := m.handleStateTabKey(tea.KeyMsg{Type: tea.KeyEnter})
	if handled {
		t.Error("expected handled to be false when not in execution mode")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleStateTabKeyWrongTabExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 0 // Not the state tab

	cmd, handled := m.handleStateTabKey(tea.KeyMsg{Type: tea.KeyEnter})
	if handled {
		t.Error("expected handled to be false when not on state tab")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleStateTabKeyNotKeyMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1

	// Pass a non-KeyMsg
	cmd, handled := m.handleStateTabKey(tea.WindowSizeMsg{})
	if handled {
		t.Error("expected handled to be false for non-KeyMsg")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ============================================================================
// loadStateListIfNeeded tests
// ============================================================================

func TestLoadStateListIfNeededNotExecutionMode(t *testing.T) {
	m := NewModel(nil)
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executionMode = false

	cmd := m.loadStateListIfNeeded()
	if cmd != nil {
		t.Error("expected nil cmd when not in execution mode")
	}
}

func TestLoadStateListIfNeededWrongTab(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 0 // Not the state tab

	cmd := m.loadStateListIfNeeded()
	if cmd != nil {
		t.Error("expected nil cmd when not on state tab")
	}
}

func TestLoadStateListIfNeededAlreadyLoaded(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1

	// Set up resources in stateListContent to simulate "already loaded"
	if m.stateListContent != nil {
		m.stateListContent.SetResources([]terraform.StateResource{
			{Address: "aws_instance.web", ResourceType: "aws_instance", Name: "web"},
		})
	}

	cmd := m.loadStateListIfNeeded()
	if cmd != nil {
		t.Error("expected nil cmd when state list already loaded")
	}
}

// ============================================================================
// handleSwitchResourcesTab tests
// ============================================================================

func TestHandleSwitchResourcesTabCannotSwitch(t *testing.T) {
	m := NewModel(nil)
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	// Cannot switch when plan is nil and not in execution mode

	model, cmd, handled := m.handleSwitchResourcesTab(1)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
	_ = cmd
}

func TestHandleSwitchResourcesTabWithPlan(t *testing.T) {
	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web"},
		},
	}
	m := NewModel(plan)
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 0

	model, _, handled := m.handleSwitchResourcesTab(1)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
}

// ============================================================================
// beginValidate command execution tests
// ============================================================================

func TestBeginValidateExecuteCommandSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.ValidateResult = testutil.NewMockResult(`{
		"format_version": "1.0",
		"valid": true,
		"error_count": 0,
		"warning_count": 0,
		"diagnostics": []
	}`, 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil // Disable to get simpler command

	cmd := m.beginValidate()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Execute the command to run the inner closure
	msg := cmd()
	completeMsg, ok := msg.(ValidateCompleteMsg)
	if !ok {
		t.Fatalf("expected ValidateCompleteMsg, got %T", msg)
	}

	if completeMsg.Error != nil {
		t.Errorf("unexpected error: %v", completeMsg.Error)
	}
	if completeMsg.Result == nil {
		t.Fatal("expected non-nil result")
	}
	if !completeMsg.Result.Valid {
		t.Error("expected valid=true")
	}
}

func TestBeginValidateExecuteCommandInvalid(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.ValidateResult = testutil.NewMockResult(`{
		"format_version": "1.0",
		"valid": false,
		"error_count": 2,
		"warning_count": 1,
		"diagnostics": [
			{"severity": "error", "summary": "Missing argument"},
			{"severity": "error", "summary": "Invalid type"}
		]
	}`, 1)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginValidate()
	msg := cmd()
	completeMsg, ok := msg.(ValidateCompleteMsg)
	if !ok {
		t.Fatalf("expected ValidateCompleteMsg, got %T", msg)
	}

	if completeMsg.Result == nil {
		t.Fatal("expected non-nil result")
	}
	if completeMsg.Result.Valid {
		t.Error("expected valid=false")
	}
	if completeMsg.Result.ErrorCount != 2 {
		t.Errorf("expected 2 errors, got %d", completeMsg.Result.ErrorCount)
	}
}

func TestBeginValidateExecuteCommandExecutorError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.ValidateErr = errors.New("validate failed")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginValidate()
	msg := cmd()
	completeMsg, ok := msg.(ValidateCompleteMsg)
	if !ok {
		t.Fatalf("expected ValidateCompleteMsg, got %T", msg)
	}

	if completeMsg.Error == nil {
		t.Error("expected error")
	}
}

func TestBeginValidateExecuteCommandResultErrorIncludesOutput(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.ValidateResult = testutil.NewMockErrorResult(
		"Error: Missing required provider",
		errors.New("exit status 1"),
	)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginValidate()
	msg := cmd()
	completeMsg, ok := msg.(ValidateCompleteMsg)
	if !ok {
		t.Fatalf("expected ValidateCompleteMsg, got %T", msg)
	}
	if completeMsg.Error == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(completeMsg.RawOutput, "Missing required provider") {
		t.Fatalf("expected stderr in raw output, got %q", completeMsg.RawOutput)
	}
}

func TestBeginValidateExecuteCommandParseError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.ValidateResult = testutil.NewMockResult(`invalid json`, 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginValidate()
	msg := cmd()
	completeMsg, ok := msg.(ValidateCompleteMsg)
	if !ok {
		t.Fatalf("expected ValidateCompleteMsg, got %T", msg)
	}

	// Parse error should be in the Error field
	if completeMsg.Error == nil {
		t.Error("expected parse error")
	}
	if completeMsg.RawOutput != "invalid json" {
		t.Errorf("expected raw output to be preserved, got %q", completeMsg.RawOutput)
	}
}

func TestBeginValidatePrepareEnvError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = nil // Will cause prepareTerraformEnv to work but we need executor
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Set up mock executor but with invalid work dir
	mock := testutil.NewMockExecutor()
	mock.MockWorkDir = "" // Empty work dir
	m.executor = mock
	m.envWorkDir = "" // Also empty

	cmd := m.beginValidate()
	// Should still return a command since prepareTerraformEnv only sets up env vars
	if cmd == nil {
		// If nil, verify error was set
		if m.err == nil {
			t.Error("expected either command or error")
		}
	}
}

// ============================================================================
// beginFormat command execution tests
// ============================================================================

func TestBeginFormatExecuteCommandSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.FormatResult = testutil.NewMockResult("main.tf\nvariables.tf", 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginFormat()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	completeMsg, ok := msg.(FormatCompleteMsg)
	if !ok {
		t.Fatalf("expected FormatCompleteMsg, got %T", msg)
	}

	if completeMsg.Error != nil {
		t.Errorf("unexpected error: %v", completeMsg.Error)
	}
	if len(completeMsg.ChangedFiles) != 2 {
		t.Errorf("expected 2 changed files, got %d", len(completeMsg.ChangedFiles))
	}
}

func TestBeginFormatExecuteCommandNoChanges(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.FormatResult = testutil.NewMockResult("", 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginFormat()
	msg := cmd()
	completeMsg, ok := msg.(FormatCompleteMsg)
	if !ok {
		t.Fatalf("expected FormatCompleteMsg, got %T", msg)
	}

	if len(completeMsg.ChangedFiles) != 0 {
		t.Errorf("expected 0 changed files, got %d", len(completeMsg.ChangedFiles))
	}
}

func TestBeginFormatExecuteCommandError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.FormatErr = errors.New("format failed")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginFormat()
	msg := cmd()
	completeMsg, ok := msg.(FormatCompleteMsg)
	if !ok {
		t.Fatalf("expected FormatCompleteMsg, got %T", msg)
	}

	if completeMsg.Error == nil {
		t.Error("expected error")
	}
}

func TestBeginFormatExecuteCommandResultError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.FormatResult = testutil.NewMockErrorResult("Error: invalid path", errors.New("exit status 1"))

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginFormat()
	msg := cmd()
	completeMsg, ok := msg.(FormatCompleteMsg)
	if !ok {
		t.Fatalf("expected FormatCompleteMsg, got %T", msg)
	}
	if completeMsg.Error == nil {
		t.Fatal("expected format error")
	}
	if completeMsg.ExecResult == nil {
		t.Fatal("expected execution result to be included")
	}
	if !strings.Contains(completeMsg.ExecResult.Output, "invalid path") {
		t.Fatalf("expected stderr output, got %q", completeMsg.ExecResult.Output)
	}
}

// ============================================================================
// beginStateList command execution tests
// ============================================================================

func TestBeginStateListExecuteCommandSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateListResult = testutil.NewMockResult("aws_instance.web\naws_s3_bucket.logs\naws_vpc.main", 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginStateList()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	completeMsg, ok := msg.(StateListCompleteMsg)
	if !ok {
		t.Fatalf("expected StateListCompleteMsg, got %T", msg)
	}

	if completeMsg.Error != nil {
		t.Errorf("unexpected error: %v", completeMsg.Error)
	}
	if len(completeMsg.Resources) != 3 {
		t.Errorf("expected 3 resources, got %d", len(completeMsg.Resources))
	}
}

func TestBeginStateListExecuteCommandEmpty(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateListResult = testutil.NewMockResult("", 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginStateList()
	msg := cmd()
	completeMsg, ok := msg.(StateListCompleteMsg)
	if !ok {
		t.Fatalf("expected StateListCompleteMsg, got %T", msg)
	}

	if len(completeMsg.Resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(completeMsg.Resources))
	}
}

func TestBeginStateListExecuteCommandError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateListErr = errors.New("no state file")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginStateList()
	msg := cmd()
	completeMsg, ok := msg.(StateListCompleteMsg)
	if !ok {
		t.Fatalf("expected StateListCompleteMsg, got %T", msg)
	}

	if completeMsg.Error == nil {
		t.Error("expected error")
	}
}

func TestBeginStateListExecuteCommandResultError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateListResult = testutil.NewMockErrorResult("state not initialized", errors.New("state error"))

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginStateList()
	msg := cmd()
	completeMsg, ok := msg.(StateListCompleteMsg)
	if !ok {
		t.Fatalf("expected StateListCompleteMsg, got %T", msg)
	}

	if completeMsg.Error == nil {
		t.Error("expected error from result.Error")
	}
}

// ============================================================================
// handleRefreshFailure tests
// ============================================================================

func TestHandleRefreshFailureBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.refreshRunning = true

	msg := RefreshCompleteMsg{
		Success: false,
		Error:   errors.New("refresh failed"),
	}

	model, _ := m.handleRefreshFailure(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// Note: handleRefreshFailure doesn't reset refreshRunning - that's done in handleRefreshComplete
}

func TestHandleRefreshFailureWithHistoryStore(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.refreshRunning = true

	// Create a temp directory for the test store
	tmpDir := t.TempDir()
	store, err := history.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}
	defer store.Close()
	m.historyStore = store

	msg := RefreshCompleteMsg{
		Success: false,
		Error:   errors.New("refresh failed"),
	}

	model, _ := m.handleRefreshFailure(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleRefreshFailureWithResult(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	result := testutil.NewMockResult("Refresh output", 1)
	result.Output = "Refresh output with details"

	msg := RefreshCompleteMsg{
		Success: false,
		Error:   errors.New("refresh failed"),
		Result:  result,
	}

	model, _ := m.handleRefreshFailure(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// waitPlanCompleteCmd tests
// ============================================================================

func TestWaitPlanCompleteCmdSuccessExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	// Provide valid plan output that the text parser can parse
	planOutput := `Terraform will perform the following actions:

  # aws_instance.example will be created
  + resource "aws_instance" "example" {
      + ami           = "ami-12345"
      + instance_type = "t2.micro"
    }

Plan: 1 to add, 0 to change, 0 to destroy.`

	result := testutil.NewMockResult(planOutput, 0)
	result.Output = planOutput

	cmd := m.waitPlanCompleteCmd(result)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	completeMsg, ok := msg.(PlanCompleteMsg)
	if !ok {
		t.Fatalf("expected PlanCompleteMsg, got %T", msg)
	}

	// PlanCompleteMsg doesn't have Success field - check Error instead
	if completeMsg.Error != nil {
		t.Errorf("unexpected error: %v", completeMsg.Error)
	}
	if completeMsg.Plan == nil {
		t.Error("expected non-nil plan")
	}
}

func TestWaitPlanCompleteCmdNilResultExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	cmd := m.waitPlanCompleteCmd(nil)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	completeMsg, ok := msg.(PlanCompleteMsg)
	if !ok {
		t.Fatalf("expected PlanCompleteMsg, got %T", msg)
	}

	if completeMsg.Error == nil {
		t.Error("expected error for nil result")
	}
}

func TestWaitPlanCompleteCmdWithErrorExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	result := testutil.NewMockErrorResult("Plan failed", errors.New("plan error"))

	cmd := m.waitPlanCompleteCmd(result)
	msg := cmd()
	completeMsg, ok := msg.(PlanCompleteMsg)
	if !ok {
		t.Fatalf("expected PlanCompleteMsg, got %T", msg)
	}

	if completeMsg.Error == nil {
		t.Error("expected non-nil error")
	}
}

// ============================================================================
// handleApplyFailure tests
// ============================================================================

func TestHandleApplyFailureBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ApplyCompleteMsg{
		Success: false,
		Error:   errors.New("apply failed"),
	}

	model, _ := m.handleApplyFailure(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}

	// Verify plan state is cleared
	if m.planFilePath != "" {
		t.Error("expected planFilePath to be cleared")
	}
}

func TestHandleApplyFailureWithResult(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	result := testutil.NewMockResult("Apply output with details", 1)
	result.Output = "Apply output with error details\nError: something went wrong"

	msg := ApplyCompleteMsg{
		Success: false,
		Error:   errors.New("apply failed"),
		Result:  result,
	}

	model, _ := m.handleApplyFailure(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleApplyFailureNoErrorButFailed(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ApplyCompleteMsg{
		Success: false,
		Error:   nil, // No error but not successful
	}

	model, _ := m.handleApplyFailure(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleApplyFailureCanceled(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := ApplyCompleteMsg{
		Success: false,
		Error:   context.Canceled,
	}

	model, _ := m.handleApplyFailure(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// showFormattedFiles tests
// ============================================================================

func TestShowFormattedFilesWithPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic with panel manager
	m.showFormattedFiles([]string{"main.tf", "variables.tf"})
}

func TestShowFormattedFilesNilCommandLogPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.commandLogPanel = nil

	// Should not panic
	m.showFormattedFiles([]string{"main.tf"})
}

// ============================================================================
// addErrorDiagnostic tests
// ============================================================================

func TestAddErrorDiagnosticWithDiagnosticsPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic
	m.addErrorDiagnostic("Test error", errors.New("test error"), "additional context")
}

func TestAddErrorDiagnosticWithCommandLogPanelExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsPanel = nil // Only command log panel

	// Should not panic
	m.addErrorDiagnostic("Test error", errors.New("test"), "output")
}

// ============================================================================
// waitPlanCompleteCmd with executor tests
// ============================================================================

func TestWaitPlanCompleteCmdWithExecutorShowPlan(t *testing.T) {
	planOutput := `Terraform will perform the following actions:

  # aws_instance.example will be created
  + resource "aws_instance" "example" {
      + ami           = "ami-12345"
      + instance_type = "t2.micro"
    }

Plan: 1 to add, 0 to change, 0 to destroy.`

	mock := setupMockExecutor(t)
	showResult := testutil.NewMockResult(planOutput, 0)
	showResult.Output = planOutput // Set Output field that waitPlanCompleteCmd checks
	mock.ShowResult = showResult

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planFilePath = "/tmp/test.tfplan"

	// Result with minimal output - should trigger show plan call
	result := testutil.NewMockResult("", 0)

	cmd := m.waitPlanCompleteCmd(result)
	msg := cmd()
	completeMsg, ok := msg.(PlanCompleteMsg)
	if !ok {
		t.Fatalf("expected PlanCompleteMsg, got %T", msg)
	}

	// Should have called Show to get plan details
	if mock.ShowCalls != 1 {
		t.Errorf("expected 1 show call, got %d", mock.ShowCalls)
	}

	if completeMsg.Error != nil {
		t.Errorf("unexpected error: %v", completeMsg.Error)
	}
}

// ============================================================================
// beginPlan command execution tests
// ============================================================================

func TestBeginPlanExecuteCommandSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.PlanOutput = testutil.NewMockOutputChannel("Planning...", "Done")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil // Disable for simpler test

	cmd := m.beginPlan()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Verify synchronous state changes
	if !m.planRunning {
		t.Error("expected planRunning to be true")
	}

	// Execute the command
	msg := cmd()
	startMsg, ok := msg.(PlanStartMsg)
	if !ok {
		t.Fatalf("expected PlanStartMsg, got %T", msg)
	}

	if startMsg.Error != nil {
		t.Errorf("unexpected error: %v", startMsg.Error)
	}
}

func TestBeginPlanExecuteCommandError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.PlanErr = errors.New("plan failed")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginPlan()
	msg := cmd()
	startMsg, ok := msg.(PlanStartMsg)
	if !ok {
		t.Fatalf("expected PlanStartMsg, got %T", msg)
	}

	if startMsg.Error == nil {
		t.Error("expected error")
	}
}

func TestBeginPlanWithCustomPlanFlags(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planFlags = []string{"-var=foo=bar", "-out=/custom/path.tfplan"}
	m.progressIndicator = nil

	cmd := m.beginPlan()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Verify plan file path was extracted from flags
	if m.planFilePath != "/custom/path.tfplan" {
		t.Errorf("expected planFilePath=/custom/path.tfplan, got %s", m.planFilePath)
	}
}

// ============================================================================
// beginApply command execution tests
// ============================================================================

func TestBeginApplyExecuteCommandSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.ApplyOutput = testutil.NewMockOutputChannel("Applying...", "Done")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginApply()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Verify synchronous state changes
	if !m.applyRunning {
		t.Error("expected applyRunning to be true")
	}

	// Execute the command
	msg := cmd()
	startMsg, ok := msg.(ApplyStartMsg)
	if !ok {
		t.Fatalf("expected ApplyStartMsg, got %T", msg)
	}

	if startMsg.Error != nil {
		t.Errorf("unexpected error: %v", startMsg.Error)
	}
}

func TestBeginApplyExecuteCommandError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.ApplyErr = errors.New("apply failed")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginApply()
	msg := cmd()
	startMsg, ok := msg.(ApplyStartMsg)
	if !ok {
		t.Fatalf("expected ApplyStartMsg, got %T", msg)
	}

	if startMsg.Error == nil {
		t.Error("expected error")
	}
}

func TestBeginApplyFromConfirmView(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewPlanConfirm // Start from confirm view
	m.progressIndicator = nil

	cmd := m.beginApply()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Should transition to main view
	if m.execView != viewMain {
		t.Errorf("expected execView=viewMain, got %d", m.execView)
	}
}

func TestBeginApplyUsesSavedPlanFile(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil
	m.applyFlags = []string{"-parallelism=5"}
	planPath := filepath.Join(t.TempDir(), "plan.tfplan")
	if err := os.WriteFile(planPath, []byte("plan"), 0o600); err != nil {
		t.Fatalf("write plan file: %v", err)
	}
	m.planFilePath = planPath

	cmd := m.beginApply()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	_ = cmd()

	if mock.ApplyCalls != 1 {
		t.Fatalf("expected one apply call, got %d", mock.ApplyCalls)
	}
	if len(mock.LastApplyOpts.Flags) != 2 {
		t.Fatalf("expected apply flags and plan path, got %v", mock.LastApplyOpts.Flags)
	}
	if mock.LastApplyOpts.Flags[1] != planPath {
		t.Fatalf("expected saved plan path in apply args, got %v", mock.LastApplyOpts.Flags)
	}
}

func TestBeginApplyReplacesExistingPlanFileArg(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	oldPlanPath := filepath.Join(t.TempDir(), "old.tfplan")
	if err := os.WriteFile(oldPlanPath, []byte("old"), 0o600); err != nil {
		t.Fatalf("write old plan file: %v", err)
	}
	newPlanPath := filepath.Join(t.TempDir(), "new.tfplan")
	if err := os.WriteFile(newPlanPath, []byte("new"), 0o600); err != nil {
		t.Fatalf("write new plan file: %v", err)
	}

	m.applyFlags = []string{"-parallelism=5", oldPlanPath}
	m.planFilePath = newPlanPath

	cmd := m.beginApply()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	_ = cmd()

	if mock.ApplyCalls != 1 {
		t.Fatalf("expected one apply call, got %d", mock.ApplyCalls)
	}
	if len(mock.LastApplyOpts.Flags) != 2 {
		t.Fatalf("expected apply flags and one plan path, got %v", mock.LastApplyOpts.Flags)
	}
	if mock.LastApplyOpts.Flags[1] != newPlanPath {
		t.Fatalf("expected updated plan path in apply args, got %v", mock.LastApplyOpts.Flags)
	}
}

func TestBeginApplyRemovesMultipleExistingPlanFileArgs(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	oldPlanPathOne := filepath.Join(t.TempDir(), "old-one.tfplan")
	if err := os.WriteFile(oldPlanPathOne, []byte("old-one"), 0o600); err != nil {
		t.Fatalf("write old plan file one: %v", err)
	}
	oldPlanPathTwo := filepath.Join(t.TempDir(), "old-two.tfplan")
	if err := os.WriteFile(oldPlanPathTwo, []byte("old-two"), 0o600); err != nil {
		t.Fatalf("write old plan file two: %v", err)
	}
	newPlanPath := filepath.Join(t.TempDir(), "new.tfplan")
	if err := os.WriteFile(newPlanPath, []byte("new"), 0o600); err != nil {
		t.Fatalf("write new plan file: %v", err)
	}

	m.applyFlags = []string{"-parallelism=5", oldPlanPathOne, oldPlanPathTwo}
	m.planFilePath = newPlanPath

	cmd := m.beginApply()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	_ = cmd()

	if mock.ApplyCalls != 1 {
		t.Fatalf("expected one apply call, got %d", mock.ApplyCalls)
	}

	gotFlags := mock.LastApplyOpts.Flags
	if len(gotFlags) != 2 {
		t.Fatalf("expected apply flags and one plan path, got %v", gotFlags)
	}
	if gotFlags[1] != newPlanPath {
		t.Fatalf("expected updated plan path in apply args, got %v", gotFlags)
	}

	planArgCount := 0
	for _, flag := range gotFlags {
		if strings.HasSuffix(strings.TrimSpace(flag), ".tfplan") {
			planArgCount++
		}
	}
	if planArgCount != 1 {
		t.Fatalf("expected exactly one .tfplan arg, got %d in %v", planArgCount, gotFlags)
	}
}

func TestBeginApplyFailsWhenSavedPlanMissing(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil
	m.planFilePath = filepath.Join(t.TempDir(), "missing.tfplan")

	cmd := m.beginApply()
	if cmd != nil {
		_ = cmd()
	}

	if mock.ApplyCalls != 0 {
		t.Fatalf("expected apply not to run when saved plan is missing")
	}
	if m.err == nil || !strings.Contains(m.err.Error(), "run terraform plan again") {
		t.Fatalf("expected missing saved plan error, got %v", m.err)
	}
}

func TestBeginApplyFailsWhenSavedPlanEnvironmentMismatch(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil
	planPath := filepath.Join(t.TempDir(), "plan.tfplan")
	if err := os.WriteFile(planPath, []byte("plan"), 0o600); err != nil {
		t.Fatalf("write plan file: %v", err)
	}
	m.planFilePath = planPath
	m.planEnvironment = "ws-1"
	m.envCurrent = "ws-2"

	cmd := m.beginApply()
	if cmd != nil {
		_ = cmd()
	}

	if mock.ApplyCalls != 0 {
		t.Fatalf("expected apply not to run when saved plan environment mismatches")
	}
	if m.err == nil || !strings.Contains(m.err.Error(), "saved plan belongs to environment") {
		t.Fatalf("expected environment mismatch error, got %v", m.err)
	}
}

func TestBeginApplyFailsWhenSavedPlanWorkdirMismatch(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.MockWorkDir = t.TempDir()

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil
	planPath := filepath.Join(t.TempDir(), "plan.tfplan")
	if err := os.WriteFile(planPath, []byte("plan"), 0o600); err != nil {
		t.Fatalf("write plan file: %v", err)
	}
	m.planFilePath = planPath
	m.planEnvironment = "dev"
	m.envCurrent = "dev"
	m.planWorkDir = filepath.Join(t.TempDir(), "other")

	cmd := m.beginApply()
	if cmd != nil {
		_ = cmd()
	}

	if mock.ApplyCalls != 0 {
		t.Fatalf("expected apply not to run when saved plan workdir mismatches")
	}
	if m.err == nil || !strings.Contains(m.err.Error(), "saved plan belongs to workdir") {
		t.Fatalf("expected workdir mismatch error, got %v", m.err)
	}
}

// ============================================================================
// beginRefresh command execution tests
// ============================================================================

func TestBeginRefreshExecuteCommandSuccess(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.RefreshOutput = testutil.NewMockOutputChannel("Refreshing...", "Done")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginRefresh()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Verify synchronous state changes
	if !m.refreshRunning {
		t.Error("expected refreshRunning to be true")
	}

	// Execute the command
	msg := cmd()
	startMsg, ok := msg.(RefreshStartMsg)
	if !ok {
		t.Fatalf("expected RefreshStartMsg, got %T", msg)
	}

	if startMsg.Error != nil {
		t.Errorf("unexpected error: %v", startMsg.Error)
	}
}

func TestBeginRefreshExecuteCommandError(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.RefreshErr = errors.New("refresh failed")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	cmd := m.beginRefresh()
	msg := cmd()
	startMsg, ok := msg.(RefreshStartMsg)
	if !ok {
		t.Fatalf("expected RefreshStartMsg, got %T", msg)
	}

	if startMsg.Error == nil {
		t.Error("expected error")
	}
}

// ============================================================================
// addErrorDiagnostic additional tests
// ============================================================================

func TestAddErrorDiagnosticWithOutputText(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Test with non-empty output (triggers output append)
	m.addErrorDiagnostic("Test error", errors.New("base error"), "additional output text")
}

func TestAddErrorDiagnosticNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.panelManager = nil

	// Should not panic with nil panel manager
	m.addErrorDiagnostic("Test error", errors.New("test"), "")
}

func TestAddErrorDiagnosticWithOperationState(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// operationState should be set after updateLayout
	if m.operationState != nil {
		m.addErrorDiagnostic("Test error", errors.New("test"), "output")
	}
}

// ============================================================================
// streamPlanOutputCmd tests
// ============================================================================

func TestStreamPlanOutputCmdWithChannel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	outputChan := make(chan string, 3)
	outputChan <- "line1"
	outputChan <- "line2"
	close(outputChan)
	m.outputChan = outputChan

	cmd := m.streamPlanOutputCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	outputMsg, ok := msg.(PlanOutputMsg)
	if !ok {
		t.Fatalf("expected PlanOutputMsg, got %T", msg)
	}

	if outputMsg.Line != "line1" {
		t.Errorf("expected line1, got %s", outputMsg.Line)
	}
}

func TestStreamPlanOutputCmdNilChannel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.outputChan = nil

	cmd := m.streamPlanOutputCmd()
	msg := cmd()
	if msg != nil {
		t.Errorf("expected nil message for nil channel, got %T", msg)
	}
}

func TestStreamPlanOutputCmdClosedChannel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	outputChan := make(chan string)
	close(outputChan)
	m.outputChan = outputChan

	cmd := m.streamPlanOutputCmd()
	msg := cmd()
	if msg != nil {
		t.Errorf("expected nil message for closed channel, got %T", msg)
	}
}

// ============================================================================
// streamApplyOutputCmd tests
// ============================================================================

func TestStreamApplyOutputCmdWithChannel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	outputChan := make(chan string, 2)
	outputChan <- "applying"
	close(outputChan)
	m.outputChan = outputChan

	cmd := m.streamApplyOutputCmd()
	msg := cmd()
	outputMsg, ok := msg.(ApplyOutputMsg)
	if !ok {
		t.Fatalf("expected ApplyOutputMsg, got %T", msg)
	}

	if outputMsg.Line != "applying" {
		t.Errorf("expected 'applying', got %s", outputMsg.Line)
	}
}

func TestStreamApplyOutputCmdNilChannel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.outputChan = nil

	cmd := m.streamApplyOutputCmd()
	msg := cmd()
	if msg != nil {
		t.Errorf("expected nil message for nil channel, got %T", msg)
	}
}

// ============================================================================
// handlePlanComplete additional tests
// ============================================================================

func TestHandlePlanCompleteWithPlan(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true

	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Change: &terraform.Change{Actions: []string{"create"}}},
		},
	}

	msg := PlanCompleteMsg{
		Plan:   plan,
		Output: "Plan output",
	}

	model, _ := m.handlePlanComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}

	// Verify planRunning is reset
	if m.planRunning {
		t.Error("expected planRunning to be false after completion")
	}
}

func TestHandlePlanCompleteWithError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true

	msg := PlanCompleteMsg{
		Error:  errors.New("plan failed"),
		Output: "Error output",
	}

	model, _ := m.handlePlanComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandlePlanCompleteNoChanges(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true

	// Empty plan with no resources
	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources:     []terraform.ResourceChange{},
	}

	msg := PlanCompleteMsg{
		Plan:   plan,
		Output: "No changes",
	}

	model, _ := m.handlePlanComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandlePlanCompleteClearsStaleDiagnostics(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true

	m.setDiagnostics([]terraform.Diagnostic{{
		Severity: "error",
		Summary:  "stale error",
	}})

	msg := PlanCompleteMsg{Output: "No changes."}
	m.handlePlanComplete(msg)

	if m.commandLogPanel == nil {
		t.Fatal("expected command log panel")
	}
	view := m.commandLogPanel.GetDiagnosticsPanel().View()
	if strings.Contains(view, "stale error") || strings.Contains(view, "Diagnostics") {
		t.Fatalf("expected stale diagnostics to be cleared, got %q", view)
	}
}

func executeBatchMessages(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	result := make([]tea.Msg, 0, len(batch))
	for _, batchCmd := range batch {
		if batchCmd == nil {
			continue
		}
		result = append(result, batchCmd())
	}
	return result
}

func TestHandleApplyCompleteEmitsNotificationOnSuccess(t *testing.T) {
	notifier := &recordingNotifier{}
	m := NewExecutionModel(nil, ExecutionConfig{Notifier: notifier})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.applyRunning = true
	m.applyStartedAt = time.Now().Add(-1 * time.Second)

	result := testutil.NewMockResult("Apply complete", 0)
	_, cmd := m.handleApplyComplete(ApplyCompleteMsg{Success: true, Result: result})
	_ = executeBatchMessages(cmd)

	if len(notifier.events) != 1 {
		t.Fatalf("expected one notification event, got %d", len(notifier.events))
	}
	event := notifier.events[0]
	if event.Action != "apply" {
		t.Fatalf("expected action apply, got %q", event.Action)
	}
	if event.Status != notifications.StatusSuccess {
		t.Fatalf("expected status success, got %q", event.Status)
	}
}

func TestHandleApplyCompleteNotificationFailureIsNonFatal(t *testing.T) {
	notifier := &recordingNotifier{err: errors.New("endpoint down")}
	m := NewExecutionModel(nil, ExecutionConfig{Notifier: notifier})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.applyRunning = true
	m.applyStartedAt = time.Now().Add(-1 * time.Second)

	result := testutil.NewMockErrorResult("apply failed output", errors.New("apply failed"))
	_, cmd := m.handleApplyComplete(ApplyCompleteMsg{Success: false, Error: errors.New("apply failed"), Result: result})
	msgs := executeBatchMessages(cmd)

	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		if _, ok := msg.(NotificationFailedMsg); ok {
			m.Update(msg)
		}
	}

	if len(notifier.events) != 1 {
		t.Fatalf("expected one notification event, got %d", len(notifier.events))
	}
	if notifier.events[0].Status != notifications.StatusFailed {
		t.Fatalf("expected failed notification status, got %q", notifier.events[0].Status)
	}
	if m.err != nil {
		t.Fatalf("expected notification error to be non-fatal, got %v", m.err)
	}
	if m.commandLogPanel == nil {
		t.Fatalf("expected command log panel")
	}
	view := m.commandLogPanel.GetDiagnosticsPanel().View()
	if !strings.Contains(view, "Desktop notification for apply was not sent") {
		t.Fatalf("expected notification failure diagnostics, got %q", view)
	}
}

func TestHandlePlanCompleteEmitsNotificationOnSuccess(t *testing.T) {
	notifier := &recordingNotifier{}
	m := NewExecutionModel(nil, ExecutionConfig{Notifier: notifier})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true
	m.planStartedAt = time.Now().Add(-500 * time.Millisecond)

	plan := &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}}}
	_, cmd := m.handlePlanComplete(PlanCompleteMsg{Plan: plan, Output: "Plan complete"})
	_ = executeBatchMessages(cmd)

	if len(notifier.events) != 1 {
		t.Fatalf("expected one notification event, got %d", len(notifier.events))
	}
	event := notifier.events[0]
	if event.Action != "plan" {
		t.Fatalf("expected action plan, got %q", event.Action)
	}
	if event.Status != notifications.StatusSuccess {
		t.Fatalf("expected success status, got %q", event.Status)
	}
}

func TestHandleRefreshCompleteEmitsFailedNotificationWhenSuccessFalseWithoutError(t *testing.T) {
	notifier := &recordingNotifier{}
	m := NewExecutionModel(nil, ExecutionConfig{Notifier: notifier})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.refreshRunning = true
	m.refreshStartedAt = time.Now().Add(-500 * time.Millisecond)

	result := testutil.NewMockResult("refresh output", 1)
	_, cmd := m.handleRefreshComplete(RefreshCompleteMsg{Success: false, Result: result})
	_ = executeBatchMessages(cmd)

	if len(notifier.events) != 1 {
		t.Fatalf("expected one notification event, got %d", len(notifier.events))
	}
	if notifier.events[0].Action != "refresh" {
		t.Fatalf("expected action refresh, got %q", notifier.events[0].Action)
	}
	if notifier.events[0].Status != notifications.StatusFailed {
		t.Fatalf("expected failed status, got %q", notifier.events[0].Status)
	}
}

func TestHandleApplyCompleteEmitsCanceledNotification(t *testing.T) {
	notifier := &recordingNotifier{}
	m := NewExecutionModel(nil, ExecutionConfig{Notifier: notifier})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.applyRunning = true
	m.applyStartedAt = time.Now().Add(-500 * time.Millisecond)

	result := testutil.NewMockErrorResult("context canceled", context.Canceled)
	_, cmd := m.handleApplyComplete(ApplyCompleteMsg{Success: false, Error: context.Canceled, Result: result})
	_ = executeBatchMessages(cmd)

	if len(notifier.events) != 1 {
		t.Fatalf("expected one notification event, got %d", len(notifier.events))
	}
	if notifier.events[0].Status != notifications.StatusCanceled {
		t.Fatalf("expected canceled status, got %q", notifier.events[0].Status)
	}
}

// ============================================================================
// Cleanup tests
// ============================================================================

func TestCleanupBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic
	m.Cleanup()
}

func TestCleanupWithCancelFunc(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Set up a cancel function
	m.cancelFunc = func() {}

	m.Cleanup()

	// cancelFunc should have been called (via cancelExecution)
	// Note: Cleanup calls cancelExecution which sets cancelFunc to nil after calling
}

func TestCleanupWithHistoryStore(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Create a temp directory for the test store
	tmpDir := t.TempDir()
	store, err := history.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}
	m.historyStore = store

	m.Cleanup()

	if m.historyStore != nil {
		t.Error("expected historyStore to be nil after cleanup")
	}
}

// ============================================================================
// CleanupTempFiles tests
// ============================================================================

func TestCleanupTempFilesBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic
	m.CleanupTempFiles()
}

func TestCleanupTempFilesWithExecutor(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Should not panic
	m.CleanupTempFiles()
}

func TestCleanupTempFilesEmptyWorkDir(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.envWorkDir = ""
	m.executor = nil

	// Should not panic - uses "." as fallback
	m.CleanupTempFiles()
}

// ============================================================================
// viewExecutionOverride tests
// ============================================================================

func TestViewExecutionOverrideNotExecutionMode(t *testing.T) {
	m := NewModel(nil)
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executionMode = false

	view := m.viewExecutionOverride()
	if view != "" {
		t.Errorf("expected empty view when not in execution mode, got %q", view)
	}
}

func TestViewExecutionOverrideExecutionModeMainView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewMain

	view := m.viewExecutionOverride()
	// viewMain returns empty string
	if view != "" {
		t.Errorf("expected empty view for viewMain, got %q", view)
	}
}

func TestViewExecutionOverrideExecutionModePlanOutput(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewPlanOutput

	view := m.viewExecutionOverride()
	// viewPlanOutput returns empty string
	if view != "" {
		t.Errorf("expected empty view for viewPlanOutput, got %q", view)
	}
}

// ============================================================================
// handleRequestApply additional tests
// ============================================================================

func TestHandleRequestApplyRefreshRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.refreshRunning = true

	model, _, handled := m.handleRequestApply()
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
}

func TestHandleRequestApplyEmptyPlan(t *testing.T) {
	// Plan with no changes
	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources:     []terraform.ResourceChange{},
	}
	m := NewExecutionModel(plan, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _, handled := m.handleRequestApply()
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
}

// ============================================================================
// handleToggleFilter tests
// ============================================================================

func TestHandleToggleFilterCreate(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	initialState := m.filterCreate
	m.handleToggleFilter(terraform.ActionCreate)

	if m.filterCreate == initialState {
		t.Error("expected filterCreate to be toggled")
	}
}

func TestHandleToggleFilterUpdate(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	initialState := m.filterUpdate
	m.handleToggleFilter(terraform.ActionUpdate)

	if m.filterUpdate == initialState {
		t.Error("expected filterUpdate to be toggled")
	}
}

func TestHandleToggleFilterDelete(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	initialState := m.filterDelete
	m.handleToggleFilter(terraform.ActionDelete)

	if m.filterDelete == initialState {
		t.Error("expected filterDelete to be toggled")
	}
}

func TestHandleToggleFilterReplace(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	initialState := m.filterReplace
	m.handleToggleFilter(terraform.ActionReplace)

	if m.filterReplace == initialState {
		t.Error("expected filterReplace to be toggled")
	}
}

// ============================================================================
// handleStateTabKey tests
// ============================================================================

func TestHandleStateTabKeyResourcesFocused(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1 // State tab

	// Set up resources so stateListContent has items
	if m.stateListContent != nil {
		m.stateListContent.SetResources([]terraform.StateResource{
			{Address: "aws_instance.web", ResourceType: "aws_instance", Name: "web"},
			{Address: "aws_s3_bucket.logs", ResourceType: "aws_s3_bucket", Name: "logs"},
		})
	}

	// Focus the resources panel
	if m.panelManager != nil {
		m.panelManager.SetFocus(PanelResources)
	}

	// Send a down key
	cmd, handled := m.handleStateTabKey(tea.KeyMsg{Type: tea.KeyDown})
	// Result depends on stateListContent handling
	_ = cmd
	_ = handled
}

func TestHandleStateTabKeyNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1
	m.panelManager = nil

	cmd, handled := m.handleStateTabKey(tea.KeyMsg{Type: tea.KeyDown})
	if handled {
		t.Error("expected handled to be false when panelManager is nil")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleStateTabKeyNilStateListContent(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1
	m.stateListContent = nil

	cmd, handled := m.handleStateTabKey(tea.KeyMsg{Type: tea.KeyDown})
	if handled {
		t.Error("expected handled to be false when stateListContent is nil")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ============================================================================
// handleRequestApply tests
// ============================================================================

func TestHandleRequestApplyWithPlanShowsModal(t *testing.T) {
	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Change: &terraform.Change{Actions: []string{"create"}}},
		},
	}
	m := NewExecutionModel(plan, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _, handled := m.handleRequestApply()
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
	// Should show confirm modal
}

func TestHandleRequestApplyNilToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.toast = nil
	m.plan = nil

	model, _, handled := m.handleRequestApply()
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
}

// ============================================================================
// addErrorDiagnostic additional tests
// ============================================================================

func TestAddErrorDiagnosticNilOperationState(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.operationState = nil

	// Should not panic and should route to diagnostics/command panel
	m.addErrorDiagnostic("Test error", errors.New("test"), "output")
}

func TestAddErrorDiagnosticNilAllPanels(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.operationState = nil
	m.commandLogPanel = nil
	m.diagnosticsPanel = nil

	// Should not panic
	m.addErrorDiagnostic("Test error", errors.New("test"), "output")
}

// ============================================================================
// renderToast tests
// ============================================================================

func TestRenderToastNilToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.toast = nil

	result := m.renderToast("test message", false)
	// When toast is nil, renderToast returns empty string
	_ = result
}

func TestRenderToastWithToastInfo(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	result := m.renderToast("info message", false)
	// Result may be empty or have content depending on toast state
	_ = result
}

func TestRenderToastWithToastError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	result := m.renderToast("error message", true)
	// Result may be empty or have content depending on toast state
	_ = result
}

// ============================================================================
// setPlan tests
// ============================================================================

func TestSetPlanWithResources(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Change: &terraform.Change{Actions: []string{"create"}}},
			{Address: "aws_s3_bucket.logs", Change: &terraform.Change{Actions: []string{"delete"}}},
		},
	}

	m.setPlan(plan)

	if m.plan != plan {
		t.Error("expected plan to be set")
	}
}

func TestSetPlanNil(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.setPlan(nil)

	if m.plan != nil {
		t.Error("expected plan to be nil")
	}
}

func TestSetPlanNilResourceList(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	// Don't call updateLayout to keep resourceList nil

	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Change: &terraform.Change{Actions: []string{"create"}}},
		},
	}

	// Should not panic
	m.setPlan(plan)
}

// ============================================================================
// planSummaryVerbose tests
// ============================================================================

func TestPlanSummaryVerboseWithChanges(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.plan = &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Change: &terraform.Change{Actions: []string{"create"}}},
			{Address: "aws_s3_bucket.logs", Change: &terraform.Change{Actions: []string{"delete"}}},
			{Address: "aws_vpc.main", Change: &terraform.Change{Actions: []string{"update"}}},
		},
	}

	summary := m.planSummaryVerbose()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestPlanSummaryVerboseNilPlan(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.plan = nil

	summary := m.planSummaryVerbose()
	// planSummaryVerbose returns "No changes" for nil plan
	if summary == "" {
		t.Error("expected non-empty summary for nil plan")
	}
}

// ============================================================================
// viewImmediate tests
// ============================================================================

func TestViewImmediateNotReady(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = false

	view := m.viewImmediate()
	if view == "" {
		t.Error("expected non-empty view when not ready")
	}
}

func TestViewImmediateWithError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.err = errors.New("test error")

	view := m.viewImmediate()
	if view == "" {
		t.Error("expected non-empty view when error is set")
	}
}

func TestViewImmediateNoPlan(t *testing.T) {
	m := NewModel(nil)
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.plan = nil
	m.executionMode = false

	view := m.viewImmediate()
	if view == "" {
		t.Error("expected non-empty view when no plan and not execution mode")
	}
}

// ============================================================================
// Update message handling tests
// ============================================================================

func TestUpdateWithPlanStartMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	result := testutil.NewMockResult("plan output", 0)
	outputCh := make(chan string)
	close(outputCh)

	msg := PlanStartMsg{
		Result: result,
		Output: outputCh,
	}

	model, _ := m.Update(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandlePlanStartErrorClearsSavedPlanMetadata(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.planFilePath = "test.tfplan"
	m.planRunFlags = []string{"-out=test.tfplan"}
	m.applyRunFlags = []string{"test.tfplan"}
	m.planEnvironment = "ws-1"
	m.planWorkDir = "/tmp/ws-1"
	m.lastPlanOutput = "old output"
	m.setPlan(&terraform.Plan{FormatVersion: "1.0"})

	_, _ = m.handlePlanStart(PlanStartMsg{Error: errors.New("failed to start")})

	if m.planFilePath != "" {
		t.Fatalf("expected plan file path cleared, got %q", m.planFilePath)
	}
	if m.planRunFlags != nil {
		t.Fatalf("expected plan flags cleared, got %v", m.planRunFlags)
	}
	if m.applyRunFlags != nil {
		t.Fatalf("expected apply flags cleared, got %v", m.applyRunFlags)
	}
	if m.planEnvironment != "" {
		t.Fatalf("expected plan environment cleared, got %q", m.planEnvironment)
	}
	if m.planWorkDir != "" {
		t.Fatalf("expected plan workdir cleared, got %q", m.planWorkDir)
	}
	if m.lastPlanOutput != "" {
		t.Fatalf("expected last plan output cleared, got %q", m.lastPlanOutput)
	}
	if m.plan != nil {
		t.Fatal("expected plan to be cleared")
	}
}

func TestHandlePlanStartSuccessClearsStalePlanResultPreservesPlanMetadata(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	planPath := filepath.Join(t.TempDir(), "plan.tfplan")
	m.planFilePath = planPath
	m.planRunFlags = []string{"-out=" + planPath}
	m.lastPlanOutput = "stale output"
	m.setPlan(&terraform.Plan{FormatVersion: "1.0"})

	output := make(chan string)
	close(output)
	result := testutil.NewMockResult("plan output", 0)

	_, cmd := m.handlePlanStart(PlanStartMsg{Result: result, Output: output})
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	if m.planFilePath != planPath {
		t.Fatalf("expected plan file path preserved, got %q", m.planFilePath)
	}
	if len(m.planRunFlags) != 1 || m.planRunFlags[0] != "-out="+planPath {
		t.Fatalf("expected plan flags preserved, got %v", m.planRunFlags)
	}
	if m.lastPlanOutput != "" {
		t.Fatalf("expected stale plan output to be cleared, got %q", m.lastPlanOutput)
	}
	if m.plan != nil {
		t.Fatal("expected stale plan to be cleared")
	}
}

func TestPlanLifecycleApplyUsesSavedPlanFromCurrentRun(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.PlanOutput = testutil.NewMockOutputChannel("Planning...", "Done")
	mock.ApplyOutput = testutil.NewMockOutputChannel("Applying...", "Done")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil
	m.envCurrent = "dev"

	stalePlanPath := filepath.Join(t.TempDir(), "stale.tfplan")
	m.applyFlags = []string{"-parallelism=5", stalePlanPath}

	planCmd := m.beginPlan()
	if planCmd == nil {
		t.Fatal("expected non-nil plan command")
	}

	planPath := m.planFilePath
	planRunFlags := append([]string{}, m.planRunFlags...)
	if !strings.HasSuffix(planPath, ".tfplan") {
		t.Fatalf("expected .tfplan output path, got %q", planPath)
	}
	if err := os.WriteFile(planPath, []byte("plan"), 0o600); err != nil {
		t.Fatalf("write plan file: %v", err)
	}

	msg := planCmd()
	startMsg, ok := msg.(PlanStartMsg)
	if !ok {
		t.Fatalf("expected PlanStartMsg, got %T", msg)
	}
	if startMsg.Error != nil {
		t.Fatalf("unexpected plan start error: %v", startMsg.Error)
	}

	_, startCmd := m.handlePlanStart(startMsg)
	if startCmd == nil {
		t.Fatal("expected plan-start continuation command")
	}

	if m.planFilePath != planPath {
		t.Fatalf("expected plan path preserved after plan start, got %q", m.planFilePath)
	}
	if strings.Join(m.planRunFlags, "|") != strings.Join(planRunFlags, "|") {
		t.Fatalf("expected plan flags preserved after plan start, got %v", m.planRunFlags)
	}

	_, _ = m.handlePlanComplete(PlanCompleteMsg{
		Plan:   &terraform.Plan{FormatVersion: "1.0"},
		Result: startMsg.Result,
		Output: "Plan: 1 to add, 0 to change, 0 to destroy.",
	})

	if m.planEnvironment != m.envCurrent {
		t.Fatalf("expected plan environment %q, got %q", m.envCurrent, m.planEnvironment)
	}
	if filepath.Clean(m.planWorkDir) != filepath.Clean(mock.MockWorkDir) {
		t.Fatalf("expected plan workdir %q, got %q", mock.MockWorkDir, m.planWorkDir)
	}

	applyCmd := m.beginApply()
	if applyCmd == nil {
		t.Fatal("expected non-nil apply command")
	}
	_ = applyCmd()

	if mock.ApplyCalls != 1 {
		t.Fatalf("expected one apply call, got %d", mock.ApplyCalls)
	}

	gotFlags := mock.LastApplyOpts.Flags
	if len(gotFlags) != 2 {
		t.Fatalf("expected apply flags and one plan path, got %v", gotFlags)
	}
	if gotFlags[1] != planPath {
		t.Fatalf("expected saved plan path in apply args, got %v", gotFlags)
	}
	if strings.Contains(strings.Join(gotFlags, "|"), stalePlanPath) {
		t.Fatalf("expected stale plan path to be removed, got %v", gotFlags)
	}

	planArgCount := 0
	for _, flag := range gotFlags {
		if strings.HasSuffix(strings.TrimSpace(flag), ".tfplan") {
			planArgCount++
		}
	}
	if planArgCount != 1 {
		t.Fatalf("expected exactly one plan file arg, got %d in %v", planArgCount, gotFlags)
	}
}

func TestPlanCompleteErrorClearsSavedPlanMetadataAfterSuccessfulPlanStart(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.PlanOutput = testutil.NewMockOutputChannel("Planning...", "Done")

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil
	m.applyRunFlags = []string{"stale.tfplan"}

	planCmd := m.beginPlan()
	if planCmd == nil {
		t.Fatal("expected non-nil plan command")
	}

	planPath := m.planFilePath
	if err := os.WriteFile(planPath, []byte("plan"), 0o600); err != nil {
		t.Fatalf("write plan file: %v", err)
	}

	msg := planCmd()
	startMsg, ok := msg.(PlanStartMsg)
	if !ok {
		t.Fatalf("expected PlanStartMsg, got %T", msg)
	}

	_, _ = m.handlePlanStart(startMsg)
	if m.planFilePath == "" {
		t.Fatal("expected plan metadata to exist before completion")
	}

	_, _ = m.handlePlanComplete(PlanCompleteMsg{
		Error:  errors.New("plan failed"),
		Result: startMsg.Result,
		Output: "plan failed",
	})

	if m.planFilePath != "" {
		t.Fatalf("expected plan file path cleared, got %q", m.planFilePath)
	}
	if m.planRunFlags != nil {
		t.Fatalf("expected plan run flags cleared, got %v", m.planRunFlags)
	}
	if m.applyRunFlags != nil {
		t.Fatalf("expected apply run flags cleared, got %v", m.applyRunFlags)
	}
	if m.planEnvironment != "" {
		t.Fatalf("expected plan environment cleared, got %q", m.planEnvironment)
	}
	if m.planWorkDir != "" {
		t.Fatalf("expected plan workdir cleared, got %q", m.planWorkDir)
	}
}

func TestHandlePlanCompleteSuccessBindsScopeWhenPlanIsNil(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.envCurrent = "dev"
	m.planRunning = true

	_, _ = m.handlePlanComplete(PlanCompleteMsg{
		Plan:   nil,
		Result: testutil.NewMockResult("plan output", 0),
		Output: "No changes.",
	})

	if m.planEnvironment != m.envCurrent {
		t.Fatalf("expected plan environment %q, got %q", m.envCurrent, m.planEnvironment)
	}
	if filepath.Clean(m.planWorkDir) != filepath.Clean(mock.MockWorkDir) {
		t.Fatalf("expected plan workdir %q, got %q", mock.MockWorkDir, m.planWorkDir)
	}
}

func TestUpdateWithApplyStartMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	result := testutil.NewMockResult("apply output", 0)
	outputCh := make(chan string)
	close(outputCh)

	msg := ApplyStartMsg{
		Result: result,
		Output: outputCh,
	}

	model, _ := m.Update(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithRefreshStartMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	result := testutil.NewMockResult("refresh output", 0)
	outputCh := make(chan string)
	close(outputCh)

	msg := RefreshStartMsg{
		Result: result,
		Output: outputCh,
	}

	model, _ := m.Update(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// handleEnvironmentChanged tests
// ============================================================================

func TestHandleEnvironmentChangedOperationRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true // Operation running should cause error

	msg := components.EnvironmentChangedMsg{
		Environment: environment.Environment{
			Strategy: environment.StrategyFolder,
			Path:     t.TempDir(),
			Name:     "test-env",
		},
	}

	model, _ := m.handleEnvironmentChanged(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// Should have triggered an error toast since operation is running
}

func TestHandleEnvironmentChangedWhenNonStreamingOperationRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.operationRunning = true

	msg := components.EnvironmentChangedMsg{
		Environment: environment.Environment{
			Strategy: environment.StrategyFolder,
			Path:     t.TempDir(),
			Name:     "test-env",
		},
	}

	_, cmd := m.handleEnvironmentChanged(msg)
	if cmd == nil {
		t.Fatal("expected error toast command when operation is running")
	}
}

func TestHandleEnvironmentChangedNilExecutorFolder(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = nil

	msg := components.EnvironmentChangedMsg{
		Environment: environment.Environment{
			Strategy: environment.StrategyFolder,
			Path:     t.TempDir(),
			Name:     "test-env",
		},
	}

	model, _ := m.handleEnvironmentChanged(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// Should have triggered an error toast since executor is nil
}

func TestHandleEnvironmentChangedWithCommandLogPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true // Trigger error

	// Ensure commandLogPanel is set up
	msg := components.EnvironmentChangedMsg{
		Environment: environment.Environment{
			Strategy: environment.StrategyFolder,
			Path:     t.TempDir(),
			Name:     "test-env",
		},
	}

	model, _ := m.handleEnvironmentChanged(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// Should have logged to command panel
}

// ============================================================================
// handleExecutionKey tests
// ============================================================================

func TestHandleExecutionKeyPlanOutputView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewPlanOutput

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEsc})
	// Should be handled by handleLegacyOutputKey
	_ = handled
	_ = cmd
}

func TestHandleExecutionKeyApplyOutputView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewApplyOutput

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEsc})
	// Should be handled by handleLegacyOutputKey
	_ = handled
	_ = cmd
}

func TestHandleExecutionKeyCommandLogView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewCommandLog

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEsc})
	// Should be handled by handleCommandLogKey
	_ = handled
	_ = cmd
}

func TestHandleExecutionKeyStateListView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateList

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEsc})
	// Should be handled by handleStateListKey
	_ = handled
	_ = cmd
}

func TestHandleExecutionKeyStateShowView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateShow

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEsc})
	// Should be handled by handleStateShowKey
	_ = handled
	_ = cmd
}

func TestHandleExecutionKeyMainViewNotHandled(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewMain

	handled, cmd := m.handleExecutionKey(tea.KeyMsg{Type: tea.KeyEsc})
	if handled {
		t.Error("expected handled to be false for viewMain")
	}
	if cmd != nil {
		t.Error("expected nil cmd for viewMain")
	}
}

// ============================================================================
// initHistory additional tests
// ============================================================================

func TestInitHistoryWithNilStore(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cfg := ExecutionConfig{
		HistoryEnabled: true,
		HistoryStore:   nil, // Will try to open default, which may fail
	}

	// Should not panic even if store cannot be opened
	m.initHistory(cfg)
}

func TestInitHistoryWithHistoryPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	tmpDir := t.TempDir()
	store, err := history.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}
	defer store.Close()

	cfg := ExecutionConfig{
		HistoryEnabled: true,
		HistoryStore:   store,
	}

	m.initHistory(cfg)

	if m.historyStore != store {
		t.Error("expected historyStore to be set")
	}
}

// ============================================================================
// beginStateShow tests (showSelectedStateDetail is internal)
// ============================================================================

func TestBeginStateShowWithMockExecutor(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateShowResult = testutil.NewMockResult(`# aws_instance.web:
resource "aws_instance" "web" {
    ami = "ami-12345"
}`, 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateShow("aws_instance.web")
	_ = cmd
}

func TestBeginStateShowNilExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = nil

	cmd := m.beginStateShow("aws_instance.web")
	if cmd != nil {
		t.Error("expected nil cmd when executor is nil")
	}
}

func TestBeginStateShowWithAddress(t *testing.T) {
	mock := setupMockExecutor(t)
	mock.StateShowResult = testutil.NewMockResult("resource details", 0)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateShow("aws_s3_bucket.data")
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}

	// Execute the command
	msg := cmd()
	showMsg, ok := msg.(StateShowCompleteMsg)
	if !ok {
		t.Fatalf("expected StateShowCompleteMsg, got %T", msg)
	}
	if showMsg.Address != "aws_s3_bucket.data" {
		t.Errorf("expected address aws_s3_bucket.data, got %s", showMsg.Address)
	}
}

// ============================================================================
// handleLegacyOutputKey tests
// ============================================================================

func TestHandleLegacyOutputKeyQNotRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = false
	m.applyRunning = false
	m.execView = viewPlanOutput

	handled, cmd := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain")
	}
}

func TestHandleLegacyOutputKeyQWhileRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true
	m.execView = viewPlanOutput

	handled, cmd := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleLegacyOutputKeyCtrlC(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true
	cancelCalled := false
	m.cancelFunc = func() { cancelCalled = true }

	handled, _ := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !handled {
		t.Error("expected handled to be true")
	}
	if !cancelCalled {
		t.Error("expected cancelFunc to be called")
	}
}

func TestHandleLegacyOutputKeyEscNotRunningExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = false
	m.applyRunning = false
	m.execView = viewPlanOutput

	handled, cmd := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain")
	}
}

func TestHandleLegacyOutputKeyEscWhileRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planRunning = true
	m.execView = viewPlanOutput
	originalView := m.execView

	handled, cmd := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	// execView should stay the same when running
	if m.execView != originalView {
		t.Error("expected execView to remain unchanged while running")
	}
}

func TestHandleLegacyOutputKeyOtherKey(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewPlanOutput

	handled, cmd := m.handleLegacyOutputKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected handled to be false")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ============================================================================
// handleKeyMsg tests
// ============================================================================

func TestHandleKeyMsgDiagnosticsFocused(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	model, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

func TestHandleKeyMsgEnvironmentPanelActive(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Focus environment panel to activate selector handling.
	if m.environmentPanel != nil {
		m.environmentPanel.SetFocused(true)
	}

	model, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

func TestHandleKeyMsgNonMainView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateList

	model, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

func TestHandleKeyMsgWithKeybindRegistry(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewMain

	// Registry should be set up
	model, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

func TestHandleKeyMsgNilKeybindRegistry(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewMain
	m.keybindRegistry = nil

	model, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if cmd != nil {
		t.Error("expected nil cmd when registry is nil")
	}
}

// ============================================================================
// handleEnvironmentPanelKey tests
// ============================================================================

func TestHandleEnvironmentPanelKeyNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil

	handled, cmd := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyEsc})
	if handled {
		t.Error("expected handled to be false when panelManager is nil")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleEnvironmentPanelKeyNilEnvironmentPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.environmentPanel = nil

	handled, cmd := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyEsc})
	if handled {
		t.Error("expected handled to be false when environmentPanel is nil")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleEnvironmentPanelKeySelectorNotActive(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Don't activate selector
	handled, cmd := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyEsc})
	if handled {
		t.Error("expected handled to be false when selector not active")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleEnvironmentPanelKeySelectorActive(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.environmentPanel != nil {
		m.environmentPanel.SetFocused(true)
		handled, _ := m.handleEnvironmentPanelKey(tea.KeyMsg{Type: tea.KeyEsc})
		// Result depends on panel handling the key
		_ = handled
	}
}

// ============================================================================
// handleDiagnosticsKey tests (execution manager specific)
// ============================================================================

func TestHandleDiagnosticsKeyNotFocusedExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = false

	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyEsc})
	if handled {
		t.Error("expected handled to be false when not focused")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleDiagnosticsKeyNilPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.diagnosticsPanel = nil

	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyEsc})
	if handled {
		t.Error("expected handled to be false when panel is nil")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleDiagnosticsKeyNotMainView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewStateList

	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyEsc})
	if handled {
		t.Error("expected handled to be false when not in main view")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleDiagnosticsKeyQuitExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleDiagnosticsKeyEscape(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.diagnosticsFocused {
		t.Error("expected diagnosticsFocused to be false")
	}
}

func TestHandleDiagnosticsKeyDExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	handled, cmd := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.diagnosticsFocused {
		t.Error("expected diagnosticsFocused to be false")
	}
}

func TestHandleDiagnosticsKeyOtherExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.diagnosticsFocused = true
	m.execView = viewMain

	handled, _ := m.handleDiagnosticsKey(tea.KeyMsg{Type: tea.KeyUp})
	if !handled {
		t.Error("expected handled to be true")
	}
}

// ============================================================================
// handleModalConfirmApplyKey tests
// ============================================================================

func TestHandleModalConfirmApplyKeyNotActive(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.modalState = ModalNone

	handled, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if handled {
		t.Error("expected handled to be false when modal not active")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleModalConfirmApplyKeyQuit(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.modalState = ModalConfirmApply

	handled, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleModalConfirmApplyKeyYesExec(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalConfirmApply

	handled, _ := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if m.modalState != ModalNone {
		t.Error("expected modalState to be ModalNone")
	}
}

func TestHandleModalConfirmApplyKeyYesUsesPendingCmdWithoutStartingApply(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalConfirmApply

	pendingCalled := false
	m.pendingConfirmCmd = func() tea.Msg {
		pendingCalled = true
		return nil
	}

	handled, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if m.modalState != ModalNone {
		t.Error("expected modalState to be ModalNone")
	}
	if m.applyRunning {
		t.Error("expected applyRunning to remain false")
	}
	if m.pendingConfirmCmd != nil {
		t.Error("expected pendingConfirmCmd to be consumed")
	}
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	_ = cmd()
	if !pendingCalled {
		t.Error("expected pending confirm command to run")
	}
}

func TestHandleModalConfirmApplyKeyNoExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.modalState = ModalConfirmApply

	handled, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.modalState != ModalNone {
		t.Error("expected modalState to be ModalNone")
	}
}

func TestHandleModalConfirmApplyKeyCtrlC(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.modalState = ModalConfirmApply
	cancelCalled := false
	m.cancelFunc = func() { cancelCalled = true }

	handled, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if !cancelCalled {
		t.Error("expected cancelFunc to be called")
	}
	if m.modalState != ModalNone {
		t.Error("expected modalState to be ModalNone")
	}
}

func TestHandleModalConfirmApplyKeyOther(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.modalState = ModalConfirmApply

	handled, cmd := m.handleModalConfirmApplyKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ============================================================================
// handleNonMainViewKey tests
// ============================================================================

func TestHandleNonMainViewKeyCommandLogView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewCommandLog

	model, _ := m.handleNonMainViewKey(tea.KeyMsg{Type: tea.KeyUp})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestHandleNonMainViewKeyCommandLogViewNilPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewCommandLog
	m.commandLogPanel = nil

	model, cmd := m.handleNonMainViewKey(tea.KeyMsg{Type: tea.KeyUp})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if cmd != nil {
		t.Error("expected nil cmd when commandLogPanel is nil")
	}
}

func TestHandleNonMainViewKeyOtherView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateList

	model, cmd := m.handleNonMainViewKey(tea.KeyMsg{Type: tea.KeyUp})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if cmd != nil {
		t.Error("expected nil cmd for non-command log view")
	}
}

// ============================================================================
// handleCommandLogKey tests
// ============================================================================

func TestHandleCommandLogKeyQuitExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	handled, cmd := m.handleCommandLogKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleCommandLogKeyEscape(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewCommandLog

	handled, cmd := m.handleCommandLogKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain")
	}
}

func TestHandleCommandLogKeyOther(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	handled, cmd := m.handleCommandLogKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected handled to be false")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ============================================================================
// handleStateListKey tests
// ============================================================================

func TestHandleStateListKeyQuitExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleStateListKeyEscape(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateList

	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.execView != viewMain {
		t.Error("expected execView to be viewMain")
	}
}

func TestHandleStateListKeyEnterExec(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateList

	// Set up resources
	if m.stateListContent != nil {
		m.stateListContent.SetResources([]terraform.StateResource{
			{Address: "aws_instance.web", ResourceType: "aws_instance", Name: "web"},
		})
	}

	handled, _ := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected handled to be true")
	}
}

func TestHandleStateListKeyOther(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected handled to be false")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ============================================================================
// handleStateShowKey tests
// ============================================================================

func TestHandleStateShowKeyQuitExec(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	handled, cmd := m.handleStateShowKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (tea.Quit)")
	}
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestHandleStateShowKeyEscape(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateShow

	handled, cmd := m.handleStateShowKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.execView != viewStateList {
		t.Error("expected execView to be viewStateList")
	}
}

func TestHandleStateShowKeyOther(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	handled, cmd := m.handleStateShowKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	// handleStateShowKey returns handled=true for all keys (updates stateShowView)
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ============================================================================
// viewExecutionOverride tests
// ============================================================================

func TestViewExecutionOverrideStateListView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateList

	view := m.viewExecutionOverride()
	// Should return state list view content
	_ = view
}

func TestViewExecutionOverrideStateShowView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewStateShow

	view := m.viewExecutionOverride()
	// Should return state show view content
	_ = view
}

func TestViewExecutionOverrideCommandLogView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewCommandLog

	view := m.viewExecutionOverride()
	// Should return command log view content
	_ = view
}

// ============================================================================
// handleStateListKey navigation tests
// ============================================================================

func TestHandleStateListKeyUpK(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Set up state list view with resources
	if m.stateListView != nil {
		m.stateListView.SetResources([]terraform.StateResource{
			{Address: "aws_instance.web", ResourceType: "aws_instance", Name: "web"},
			{Address: "aws_s3_bucket.logs", ResourceType: "aws_s3_bucket", Name: "logs"},
		})
	}

	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleStateListKeyDownJ(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Set up state list view with resources
	if m.stateListView != nil {
		m.stateListView.SetResources([]terraform.StateResource{
			{Address: "aws_instance.web", ResourceType: "aws_instance", Name: "web"},
			{Address: "aws_s3_bucket.logs", ResourceType: "aws_s3_bucket", Name: "logs"},
		})
	}

	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleStateListKeyUpArrow(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyUp})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleStateListKeyDownArrow(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleStateListKeyEnterNoSelected(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Don't set any resources, so nothing is selected
	handled, cmd := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd when nothing selected")
	}
}

func TestHandleStateListKeyNilStateListView(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.stateListView = nil

	// Test navigation keys with nil stateListView
	handled, _ := m.handleStateListKey(tea.KeyMsg{Type: tea.KeyUp})
	if !handled {
		t.Error("expected handled to be true")
	}

	handled, _ = m.handleStateListKey(tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Error("expected handled to be true")
	}

	handled, _ = m.handleStateListKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected handled to be true")
	}
}

// ============================================================================
// applyViewOverlays tests
// ============================================================================

func TestApplyViewOverlaysHelpModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalHelp

	result := m.applyViewOverlays("base view")
	// Should apply help modal overlay
	_ = result
}

func TestApplyViewOverlaysConfirmApplyModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalConfirmApply

	result := m.applyViewOverlays("base view")
	// Should apply confirm apply modal overlay
	_ = result
}

func TestApplyViewOverlaysThemeModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalTheme

	result := m.applyViewOverlays("base view")
	// Should apply theme modal overlay
	_ = result
}

func TestApplyViewOverlaysNoModal(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalNone

	result := m.applyViewOverlays("base view")
	if result != "base view" {
		t.Error("expected unchanged view when no modal is active")
	}
}

func TestApplyViewOverlaysToastVisible(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.modalState = ModalNone

	// Show a toast
	if m.toast != nil {
		m.toast.ShowInfo("Test message")
	}

	result := m.applyViewOverlays("base view")
	_ = result
}

// ============================================================================
// handlePostUpdate tests
// ============================================================================

func TestHandlePostUpdateKeyMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, cmd := m.handlePostUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

func TestHandlePostUpdateUnknownMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Send a custom message type that isn't handled
	type customMsg struct{}
	model, cmd := m.handlePostUpdate(customMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if cmd != nil {
		t.Error("expected nil cmd for unknown message")
	}
}

// ============================================================================
// loadStateListIfNeeded tests
// ============================================================================

func TestLoadStateListIfNeededWithExecutor(t *testing.T) {
	mock := setupMockExecutor(t)

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.resourcesActiveTab = 1 // State tab

	cmd := m.loadStateListIfNeeded()
	// Should return a command to load state list
	_ = cmd
}

// ============================================================================
// showConfirmApplyModal tests
// ============================================================================

func TestShowConfirmApplyModalBasic(t *testing.T) {
	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Change: &terraform.Change{Actions: []string{"create"}}},
		},
	}

	m := NewExecutionModel(plan, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.showConfirmApplyModal()

	if m.modalState != ModalConfirmApply {
		t.Error("expected modalState to be ModalConfirmApply")
	}
}

func TestShowConfirmApplyModalNilHelpModal(t *testing.T) {
	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Change: &terraform.Change{Actions: []string{"create"}}},
		},
	}

	m := NewExecutionModel(plan, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.helpModal = nil

	// When helpModal is nil, the function returns early and modal state should remain ModalNone
	m.showConfirmApplyModal()
	if m.modalState != ModalNone {
		t.Error("expected modalState to remain ModalNone when helpModal is nil")
	}
}

// ============================================================================
// reloadHistoryCmd tests
// ============================================================================

func TestReloadHistoryCmdExecute(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	tmpDir := t.TempDir()
	store, err := history.Open(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}
	defer store.Close()

	m.historyStore = store

	cmd := m.reloadHistoryCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	// Execute the command
	msg := cmd()
	historyMsg, ok := msg.(HistoryLoadedMsg)
	if !ok {
		t.Fatalf("expected HistoryLoadedMsg, got %T", msg)
	}

	if historyMsg.Error != nil {
		t.Errorf("unexpected error: %v", historyMsg.Error)
	}
}

// ============================================================================
// handleEscKey tests
// ============================================================================

func TestHandleEscKeyNilMainArea(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.mainArea = nil

	result := m.handleEscKey()
	if result {
		t.Error("expected false when mainArea is nil")
	}
}

func TestHandleEscKeyHistoryDetailMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea != nil {
		m.mainArea.SetMode(ModeHistoryDetail)
		result := m.handleEscKey()
		if !result {
			t.Error("expected true when in history detail mode")
		}
	}
}

func TestHandleEscKeyStateShowMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea != nil {
		m.mainArea.SetMode(ModeStateShow)
		result := m.handleEscKey()
		if !result {
			t.Error("expected true when in state show mode")
		}
	}
}

func TestHandleEscKeyDiffMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	if m.mainArea != nil {
		m.mainArea.SetMode(ModeDiff)
		result := m.handleEscKey()
		if result {
			t.Error("expected false when in diff mode")
		}
	}
}

func TestPlanFlagsForRunIncludesSelectedTargets(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.planFlags = []string{"-refresh=true"}
	m.targetModeEnabled = true
	m.resourceList.SetTargetModeEnabled(true)
	m.resourceList.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
	})
	_ = m.resourceList.ToggleTargetSelectionAtSelected()

	flags, _ := m.planFlagsForRun()
	joined := strings.Join(flags, " ")
	if !strings.Contains(joined, "-target=module.alpha.aws_instance.web") {
		t.Fatalf("expected target flag in %q", joined)
	}
}

func TestApplyFlagsForRunRejectsUnpinnedTargetSelection(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.targetModeEnabled = true
	m.resourceList.SetTargetModeEnabled(true)
	m.resourceList.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})
	_ = m.resourceList.ToggleTargetSelectionAtSelected()

	_, err := m.applyFlagsForRun()
	if err == nil {
		t.Fatal("expected error when target plan pin is missing")
	}
}

func TestHandleRequestApplyInTargetModeRequiresSelectionAndPin(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.plan = &terraform.Plan{Resources: []terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}}}
	m.targetModeEnabled = true
	m.resourceList.SetTargetModeEnabled(true)
	m.resourceList.SetResources(m.plan.Resources)

	_, _, handled := m.handleRequestApply()
	if !handled {
		t.Fatal("expected apply request to be handled")
	}
	if m.modalState == ModalConfirmApply {
		t.Fatal("did not expect confirm modal without target selection")
	}

	_ = m.resourceList.ToggleTargetSelectionAtSelected()
	_, _, handled = m.handleRequestApply()
	if !handled {
		t.Fatal("expected apply request with target selection to be handled")
	}
	if !m.pendingTargetApply {
		t.Fatal("expected pending target apply intent")
	}
}

// ============================================================================
// focusCommandLog tests
// ============================================================================

func TestFocusCommandLogBasic(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.focusCommandLog()
	_ = cmd
}

func TestFocusCommandLogNilPanelManager(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.panelManager = nil

	cmd := m.focusCommandLog()
	if cmd != nil {
		t.Error("expected nil cmd when panelManager is nil")
	}
}

func TestFocusCommandLogNilCommandLogPanel(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.commandLogPanel = nil

	cmd := m.focusCommandLog()
	_ = cmd
}

// ============================================================================
// Update message handling additional tests
// ============================================================================

func TestUpdateWithClearToastMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(ClearToastMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithErrorMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(ErrorMsg{Err: errors.New("test error")})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithHistoryLoadedMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(HistoryLoadedMsg{
		Entries: []history.Entry{},
	})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithHistoryDetailMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(HistoryDetailMsg{
		Entry: history.Entry{ID: 1, Summary: "test"},
	})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithEnvironmentDetectedMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(EnvironmentDetectedMsg{
		Result: environment.DetectionResult{},
	})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithValidateCompleteMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(ValidateCompleteMsg{
		Result: &terraform.ValidateResult{Valid: true},
	})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithFormatCompleteMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(FormatCompleteMsg{
		ChangedFiles: []string{},
	})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithStateShowCompleteMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(StateShowCompleteMsg{
		Address: "aws_instance.web",
		Output:  "resource details",
	})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// Additional coverage tests for handleTertiaryUpdate
// ============================================================================

func TestUpdateWithSpinnerTickNilProgressIndicator(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.progressIndicator = nil

	model, _ := m.Update(spinner.TickMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithRequestPlanMsg(t *testing.T) {
	executor := testutil.NewMockExecutor()
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = executor
	m.executionMode = true

	model, cmd := m.Update(RequestPlanMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// Should trigger beginPlan
	_ = cmd
}

func TestUpdateWithRequestRefreshMsg(t *testing.T) {
	executor := testutil.NewMockExecutor()
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = executor
	m.executionMode = true

	model, cmd := m.Update(RequestRefreshMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

func TestUpdateWithRequestValidateMsg(t *testing.T) {
	executor := testutil.NewMockExecutor()
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = executor
	m.executionMode = true

	model, cmd := m.Update(RequestValidateMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

func TestUpdateWithRequestFormatMsg(t *testing.T) {
	executor := testutil.NewMockExecutor()
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = executor
	m.executionMode = true

	model, cmd := m.Update(RequestFormatMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

func TestUpdateWithToggleFilterMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(ToggleFilterMsg{Action: terraform.ActionCreate})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithToggleStatusMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	initialStatus := m.resourceList.ShowStatus()
	model, _ := m.Update(ToggleStatusMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	// Status should be toggled
	if m.resourceList.ShowStatus() == initialStatus {
		t.Error("expected ShowStatus to be toggled")
	}
}

func TestUpdateWithToggleAllGroupsMsg(t *testing.T) {
	plan := &terraform.Plan{
		FormatVersion: "1.0",
		Resources: []terraform.ResourceChange{
			{Address: "aws_instance.web", Change: &terraform.Change{Actions: []string{"create"}}},
		},
	}
	m := NewExecutionModel(plan, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(ToggleAllGroupsMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithStateListStartMsg(t *testing.T) {
	executor := testutil.NewMockExecutor()
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = executor
	m.executionMode = true

	model, cmd := m.Update(StateListStartMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

func TestUpdateWithSwitchResourcesTabMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executionMode = true

	// Test forward direction
	model, _ := m.Update(SwitchResourcesTabMsg{Direction: 1})
	if model == nil {
		t.Fatal("expected non-nil model")
	}

	// Test backward direction
	model, _ = m.Update(SwitchResourcesTabMsg{Direction: -1})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// handleRequestApply additional tests (unique tests only)
// ============================================================================

func TestHandleRequestApplyNilToastWhilePlanRunning(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.toast = nil
	m.planRunning = true

	model, cmd, handled := m.handleRequestApply()
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if !handled {
		t.Error("expected handled to be true")
	}
	if cmd != nil {
		t.Error("expected nil cmd when toast is nil and operation running")
	}
}

// ============================================================================
// handleToggleFilter additional tests (unique tests only)
// ============================================================================

func TestHandleToggleFilterNoOpAction(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.handleToggleFilter(terraform.ActionNoOp)
}

func TestHandleToggleFilterReadAction(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	m.handleToggleFilter(terraform.ActionRead)
}

// ============================================================================
// viewImmediate additional tests (unique tests only)
// ============================================================================

func TestViewImmediateExecutionViewOverride(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.execView = viewPlanOutput

	// viewImmediate delegates to viewExecutionOverride when in execution view
	// The view may be empty if planView isn't set up, but it shouldn't panic
	_ = m.viewImmediate()
}

// ============================================================================
// loadStateListIfNeeded additional tests (unique tests only)
// ============================================================================

func TestLoadStateListIfNeededWithNilExecutor(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = nil

	cmd := m.loadStateListIfNeeded()
	if cmd != nil {
		t.Error("expected nil cmd when executor is nil")
	}
}

// ============================================================================
// Update with unhandled default case
// ============================================================================

func TestUpdateWithUnhandledMessage(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Send a custom struct that isn't handled
	type CustomMsg struct{}
	model, _ := m.Update(CustomMsg{})
	if model == nil {
		t.Fatal("expected non-nil model for unhandled message")
	}
}

// ============================================================================
// handlePostUpdate tests
// ============================================================================

func TestHandlePostUpdateWithTickMsg(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// Create a blinking cursor tick
	model, cmd := m.handlePostUpdate(cursor.BlinkMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
}

// ============================================================================
// Additional message handling tests
// ============================================================================

func TestUpdateWithComponentsClearToast(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	model, _ := m.Update(components.ClearToast{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestUpdateWithSpinnerTickWithProgressIndicator(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	// progressIndicator should be set by updateLayout
	if m.progressIndicator == nil {
		t.Skip("progressIndicator not initialized")
	}

	model, _ := m.Update(spinner.TickMsg{})
	if model == nil {
		t.Fatal("expected non-nil model")
	}
}

// ============================================================================
// handleSwitchResourcesTab additional tests (unique only)
// ============================================================================

func TestHandleSwitchResourcesTabNotExecMode(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executionMode = false

	model, cmd, handled := m.handleSwitchResourcesTab(1)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
	_ = handled
}

func TestHandleSwitchResourcesTabToStateListTab(t *testing.T) {
	executor := testutil.NewMockExecutor()
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.executor = executor
	m.executionMode = true
	m.resourcesActiveTab = 0

	model, cmd, handled := m.handleSwitchResourcesTab(1)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	_ = cmd
	_ = handled
}

func TestStateMutationSessionOutputIncludesTerraformOutputOnError(t *testing.T) {
	err := errors.New("exit status 1")
	output := "Error: Invalid target address\n\nDestination address already exists"

	result := stateMutationSessionOutput(output, err, "/tmp/backup.tfstate")
	if !strings.Contains(result, "Destination address already exists") {
		t.Fatalf("expected terraform error output in session log, got: %q", result)
	}
	if !strings.Contains(result, "backup: /tmp/backup.tfstate") {
		t.Fatalf("expected backup path in session log, got: %q", result)
	}
}

func TestBeginStateMvUsesCombinedResultOutputOnError(t *testing.T) {
	mock := setupMockExecutor(t)
	workDir := t.TempDir()
	mock.MockWorkDir = workDir
	mock.StatePullResult = testutil.NewMockResult(`{"version":4}`, 0)

	result := testutil.NewMockErrorResult("Error: destination address already exists", errors.New("exit status 1"))
	result.Output = "Error: destination address already exists"
	mock.StateMvResult = result

	m := NewExecutionModel(nil, ExecutionConfig{})
	m.executor = mock
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	cmd := m.beginStateMv("null_resource.example", "null_resource.example2")
	if cmd == nil {
		t.Fatal("expected non-nil state mv cmd")
	}
	raw := cmd()
	msg, ok := raw.(StateMvCompleteMsg)
	if !ok {
		t.Fatalf("expected StateMvCompleteMsg, got %T", raw)
	}
	if msg.Error == nil {
		t.Fatal("expected state mv error")
	}
	if !strings.Contains(msg.Output, "destination address already exists") {
		t.Fatalf("expected terraform stderr in output, got %q", msg.Output)
	}
}

func TestHandleInitCompleteSuccess(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := InitCompleteMsg{Output: "Terraform has been successfully initialized!"}
	model, cmd := m.handleInitComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if cmd == nil {
		t.Fatal("expected success toast command")
	}

	if m.commandLogPanel == nil {
		t.Fatal("expected command log panel")
	}
	if got := m.commandLogPanel.View(); !strings.Contains(got, "terraform init") {
		t.Fatalf("expected init command log entry, got %q", got)
	}
}

func TestHandleInitCompleteError(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	msg := InitCompleteMsg{Error: errors.New("init failed")}
	model, cmd := m.handleInitComplete(msg)
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if cmd == nil {
		t.Fatal("expected error toast command")
	}
}
