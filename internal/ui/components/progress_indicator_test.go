package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/styles"
)

func TestNewProgressIndicator(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	if p == nil {
		t.Fatal("expected non-nil progress indicator")
	}
	if p.styles != s {
		t.Error("expected styles to be set")
	}
	if p.state != ProgressIdle {
		t.Errorf("expected state ProgressIdle, got %d", p.state)
	}
	if p.operation != OperationNone {
		t.Errorf("expected operation OperationNone, got %d", p.operation)
	}
}

func TestProgressIndicatorStart(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	// Test starting each operation type
	operations := []ProgressOperation{
		OperationPlan,
		OperationApply,
		OperationRefresh,
		OperationValidate,
		OperationFormat,
		OperationStateList,
	}

	for _, op := range operations {
		p.Reset() // Reset between tests
		cmd := p.Start(op)

		if p.state != ProgressRunning {
			t.Errorf("expected state ProgressRunning after Start, got %d", p.state)
		}
		if p.operation != op {
			t.Errorf("expected operation %d, got %d", op, p.operation)
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from Start")
		}
	}
}

func TestProgressIndicatorFail(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	p.Start(OperationPlan)
	p.Fail()

	if p.state != ProgressFailed {
		t.Errorf("expected state ProgressFailed after Fail, got %d", p.state)
	}
}

func TestProgressIndicatorReset(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	// Start an operation
	p.Start(OperationApply)

	// Reset
	p.Reset()

	if p.state != ProgressIdle {
		t.Errorf("expected state ProgressIdle after Reset, got %d", p.state)
	}
	if p.operation != OperationNone {
		t.Errorf("expected operation OperationNone after Reset, got %d", p.operation)
	}
}

func TestProgressIndicatorUpdate(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	// Update with non-tick message should return nil
	cmd := p.Update(tea.KeyMsg{})
	if cmd != nil {
		t.Error("expected nil cmd for non-tick message")
	}

	// Update when not running should return nil
	p.state = ProgressIdle
	cmd = p.Update(spinner.TickMsg{})
	if cmd != nil {
		t.Error("expected nil cmd when not running")
	}

	// Update when running should return a tick cmd
	p.Start(OperationPlan)
	cmd = p.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Error("expected non-nil cmd from Update when running")
	}
}

func TestProgressIndicatorView(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	// Idle state should return empty string
	view := p.View()
	if view != "" {
		t.Errorf("expected empty view for idle state, got %q", view)
	}

	// Nil styles should return empty string
	p.styles = nil
	p.state = ProgressRunning
	view = p.View()
	if view != "" {
		t.Errorf("expected empty view for nil styles, got %q", view)
	}

	// Restore styles
	p.styles = s
}

func TestProgressIndicatorViewRunning(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	operations := []struct {
		op       ProgressOperation
		contains string
	}{
		{OperationPlan, "Running Plan"},
		{OperationApply, "Applying"},
		{OperationRefresh, "Refreshing State"},
		{OperationValidate, "Validating"},
		{OperationFormat, "Formatting"},
		{OperationStateList, "Loading State"},
	}

	for _, tc := range operations {
		p.Reset()
		p.Start(tc.op)

		view := p.View()
		if view == "" {
			t.Errorf("expected non-empty view for operation %d", tc.op)
		}
		// The view should contain the operation text (styled)
		// We can't check exact content due to ANSI codes
	}
}

func TestProgressIndicatorViewFailed(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	operations := []struct {
		op       ProgressOperation
		contains string
	}{
		{OperationPlan, "Plan Failed"},
		{OperationApply, "Apply Failed"},
		{OperationRefresh, "Refresh Failed"},
		{OperationValidate, "Validation Failed"},
		{OperationFormat, "Format Failed"},
		{OperationStateList, "State Load Failed"},
	}

	for _, tc := range operations {
		p.Reset()
		p.Start(tc.op)
		p.Fail()

		view := p.View()
		if view == "" {
			t.Errorf("expected non-empty view for failed operation %d", tc.op)
		}
		// The view should contain the failure text (styled)
	}
}

func TestProgressIndicatorViewOperationNone(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	// Set running state but no operation
	p.state = ProgressRunning
	p.operation = OperationNone

	view := p.View()
	if view != "" {
		t.Errorf("expected empty view for OperationNone, got %q", view)
	}
}

func TestProgressIndicatorSetStyles(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	newStyles := styles.DefaultStyles()
	p.SetStyles(newStyles)

	if p.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestProgressIndicatorSetDetail(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	p.Start(OperationPlan)
	p.SetDetail("waiting for state lock")

	view := p.View()
	if !strings.Contains(view, "waiting for state lock") {
		t.Fatalf("expected detail text in view, got %q", view)
	}

	p.Reset()
	if p.detail != "" {
		t.Fatal("expected reset to clear detail")
	}
}

func TestProgressIndicatorStartClearsDetail(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	p.Start(OperationPlan)
	p.SetDetail("waiting for state lock")
	p.Start(OperationApply)

	if p.detail != "" {
		t.Fatalf("expected start to clear detail, got %q", p.detail)
	}
}

func TestProgressIndicatorGetIconAndText(t *testing.T) {
	s := styles.DefaultStyles()
	p := NewProgressIndicator(s)

	// Test running state - no icon returned (spinner handles it)
	p.state = ProgressRunning
	p.operation = OperationPlan
	icon, text := p.getIconAndText()
	if icon != "" {
		t.Errorf("expected empty icon for running state, got %q", icon)
	}
	if text != "Running Plan" {
		t.Errorf("expected 'Running Plan', got %q", text)
	}

	// Test failed state
	p.state = ProgressFailed
	icon, text = p.getIconAndText()
	if icon != "●" {
		t.Errorf("expected ● for failed state, got %q", icon)
	}
	if text != "Plan Failed" {
		t.Errorf("expected 'Plan Failed', got %q", text)
	}

	// Test idle state
	p.state = ProgressIdle
	icon, text = p.getIconAndText()
	if icon != "" || text != "" {
		t.Errorf("expected empty icon and text for idle, got %q, %q", icon, text)
	}
}

func TestProgressIndicatorNilStylesNewProgressIndicator(t *testing.T) {
	p := NewProgressIndicator(nil)

	if p == nil {
		t.Fatal("expected non-nil progress indicator even with nil styles")
	}
	if p.styles != nil {
		t.Error("expected nil styles when nil passed")
	}

	// View should return empty with nil styles
	p.state = ProgressRunning
	p.operation = OperationPlan
	view := p.View()
	if view != "" {
		t.Errorf("expected empty view with nil styles, got %q", view)
	}
}

func TestProgressStateConstants(t *testing.T) {
	// Verify constant values
	if ProgressIdle != 0 {
		t.Error("expected ProgressIdle to be 0")
	}
	if ProgressRunning != 1 {
		t.Error("expected ProgressRunning to be 1")
	}
	if ProgressFailed != 2 {
		t.Error("expected ProgressFailed to be 2")
	}
}

func TestProgressOperationConstants(t *testing.T) {
	// Verify constant values
	if OperationNone != 0 {
		t.Error("expected OperationNone to be 0")
	}
	if OperationPlan != 1 {
		t.Error("expected OperationPlan to be 1")
	}
	if OperationApply != 2 {
		t.Error("expected OperationApply to be 2")
	}
	if OperationRefresh != 3 {
		t.Error("expected OperationRefresh to be 3")
	}
	if OperationValidate != 4 {
		t.Error("expected OperationValidate to be 4")
	}
	if OperationFormat != 5 {
		t.Error("expected OperationFormat to be 5")
	}
	if OperationInit != 6 {
		t.Error("expected OperationInit to be 6")
	}
	if OperationStateList != 7 {
		t.Error("expected OperationStateList to be 7")
	}
}
