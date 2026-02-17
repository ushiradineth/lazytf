package testutil

import (
	"context"
	"sync"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

// MockExecutor implements terraform.ExecutorInterface for testing.
type MockExecutor struct {
	mu sync.Mutex

	// Call tracking
	InitCalls      int
	PlanCalls      int
	ApplyCalls     int
	RefreshCalls   int
	ValidateCalls  int
	FormatCalls    int
	StateListCalls int
	StateShowCalls int
	StateRmCalls   int
	StateMvCalls   int
	StatePullCalls int
	ShowCalls      int
	VersionCalls   int

	// Last call arguments (for verification)
	LastPlanOpts      terraform.PlanOptions
	LastApplyOpts     terraform.ApplyOptions
	LastRefreshOpts   terraform.RefreshOptions
	LastValidateOpts  terraform.ValidateOptions
	LastFormatOpts    terraform.FormatOptions
	LastStateListOpts terraform.StateListOptions
	LastStateShowAddr string
	LastStateShowOpts terraform.StateShowOptions
	LastStateRmAddr   string
	LastStateRmOpts   terraform.StateRmOptions
	LastStateMvSrc    string
	LastStateMvDst    string
	LastStateMvOpts   terraform.StateMvOptions
	LastStatePullOpts terraform.StatePullOptions
	LastShowPlanFile  string
	LastShowOpts      terraform.ShowOptions

	// Configurable return values
	InitResult *terraform.ExecutionResult
	InitErr    error

	PlanResult *terraform.ExecutionResult
	PlanOutput chan string
	PlanErr    error

	ApplyResult *terraform.ExecutionResult
	ApplyOutput chan string
	ApplyErr    error

	RefreshResult *terraform.ExecutionResult
	RefreshOutput chan string
	RefreshErr    error

	ValidateResult *terraform.ExecutionResult
	ValidateErr    error

	FormatResult *terraform.ExecutionResult
	FormatErr    error

	StateListResult *terraform.ExecutionResult
	StateListErr    error

	StateShowResult *terraform.ExecutionResult
	StateShowErr    error

	StateRmResult *terraform.ExecutionResult
	StateRmErr    error

	StateMvResult *terraform.ExecutionResult
	StateMvErr    error

	StatePullResult *terraform.ExecutionResult
	StatePullErr    error

	ShowResult *terraform.ExecutionResult
	ShowErr    error

	VersionResult string
	VersionErr    error

	// WorkDir configuration
	MockWorkDir string
}

// NewMockExecutor creates a mock with sensible defaults.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		MockWorkDir:   "/mock/workdir",
		VersionResult: "1.5.0",
	}
}

// NewMockResult creates a completed ExecutionResult with the given stdout and exit code.
func NewMockResult(stdout string, exitCode int) *terraform.ExecutionResult {
	result := terraform.NewExecutionResult()
	result.Stdout = stdout
	result.ExitCode = exitCode
	// Auto-finish in background so Done() channel closes
	go func() {
		result.Finish()
	}()
	return result
}

// NewMockErrorResult creates a completed ExecutionResult representing an error.
func NewMockErrorResult(stderr string, err error) *terraform.ExecutionResult {
	result := terraform.NewExecutionResult()
	result.Stderr = stderr
	result.ExitCode = 1
	result.Error = err
	go func() {
		result.Finish()
	}()
	return result
}

// NewMockOutputChannel creates a channel with the given lines and closes it.
func NewMockOutputChannel(lines ...string) chan string {
	ch := make(chan string, len(lines))
	for _, line := range lines {
		ch <- line
	}
	close(ch)
	return ch
}

// Init implements ExecutorInterface.
func (m *MockExecutor) Init(_ context.Context) (*terraform.ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InitCalls++

	if m.InitErr != nil {
		return nil, m.InitErr
	}

	result := m.InitResult
	if result == nil {
		result = NewMockResult("Terraform has been successfully initialized!", 0)
	}
	return result, nil
}

// streamingOpResult returns a result, output channel, and error for streaming operations.
func (m *MockExecutor) streamingOpResult(
	configuredResult *terraform.ExecutionResult,
	configuredOutput chan string,
	configuredErr error,
	defaultMsg string,
) (*terraform.ExecutionResult, <-chan string, error) {
	if configuredErr != nil {
		return nil, nil, configuredErr
	}

	result := configuredResult
	if result == nil {
		result = NewMockResult(defaultMsg, 0)
	}

	output := configuredOutput
	if output == nil {
		output = make(chan string)
		close(output)
	}

	return result, output, nil
}

// Plan implements ExecutorInterface.
func (m *MockExecutor) Plan(_ context.Context, opts terraform.PlanOptions) (*terraform.ExecutionResult, <-chan string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PlanCalls++
	m.LastPlanOpts = opts
	return m.streamingOpResult(m.PlanResult, m.PlanOutput, m.PlanErr, "Plan: 1 to add, 0 to change, 0 to destroy.")
}

// Apply implements ExecutorInterface.
func (m *MockExecutor) Apply(_ context.Context, opts terraform.ApplyOptions) (*terraform.ExecutionResult, <-chan string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ApplyCalls++
	m.LastApplyOpts = opts
	return m.streamingOpResult(m.ApplyResult, m.ApplyOutput, m.ApplyErr, "Apply complete! Resources: 1 added, 0 changed, 0 destroyed.")
}

// Refresh implements ExecutorInterface.
func (m *MockExecutor) Refresh(_ context.Context, opts terraform.RefreshOptions) (*terraform.ExecutionResult, <-chan string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RefreshCalls++
	m.LastRefreshOpts = opts
	return m.streamingOpResult(m.RefreshResult, m.RefreshOutput, m.RefreshErr, "Refresh complete.")
}

// Validate implements ExecutorInterface.
func (m *MockExecutor) Validate(_ context.Context, opts terraform.ValidateOptions) (*terraform.ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ValidateCalls++
	m.LastValidateOpts = opts

	if m.ValidateErr != nil {
		return nil, m.ValidateErr
	}

	result := m.ValidateResult
	if result == nil {
		result = NewMockResult(`{"format_version":"1.0","valid":true,"error_count":0,"warning_count":0,"diagnostics":[]}`, 0)
	}
	return result, nil
}

// Format implements ExecutorInterface.
func (m *MockExecutor) Format(_ context.Context, opts terraform.FormatOptions) (*terraform.ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FormatCalls++
	m.LastFormatOpts = opts

	if m.FormatErr != nil {
		return nil, m.FormatErr
	}

	result := m.FormatResult
	if result == nil {
		result = NewMockResult("", 0) // No files changed by default
	}
	return result, nil
}

// StateList implements ExecutorInterface.
func (m *MockExecutor) StateList(_ context.Context, opts terraform.StateListOptions) (*terraform.ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StateListCalls++
	m.LastStateListOpts = opts

	if m.StateListErr != nil {
		return nil, m.StateListErr
	}

	result := m.StateListResult
	if result == nil {
		result = NewMockResult("aws_instance.example\naws_s3_bucket.data", 0)
	}
	return result, nil
}

// StateShow implements ExecutorInterface.
func (m *MockExecutor) StateShow(_ context.Context, address string, opts terraform.StateShowOptions) (*terraform.ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StateShowCalls++
	m.LastStateShowAddr = address
	m.LastStateShowOpts = opts

	if m.StateShowErr != nil {
		return nil, m.StateShowErr
	}

	result := m.StateShowResult
	if result == nil {
		result = NewMockResult(`# `+address+`:
resource "aws_instance" "example" {
    ami           = "ami-12345"
    instance_type = "t2.micro"
}`, 0)
	}
	return result, nil
}

// StateRm implements ExecutorInterface.
func (m *MockExecutor) StateRm(_ context.Context, address string, opts terraform.StateRmOptions) (*terraform.ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StateRmCalls++
	m.LastStateRmAddr = address
	m.LastStateRmOpts = opts

	if m.StateRmErr != nil {
		return nil, m.StateRmErr
	}

	result := m.StateRmResult
	if result == nil {
		result = NewMockResult("Removed "+address, 0)
	}
	return result, nil
}

// StateMv implements ExecutorInterface.
func (m *MockExecutor) StateMv(
	_ context.Context,
	srcAddress, dstAddress string,
	opts terraform.StateMvOptions,
) (*terraform.ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StateMvCalls++
	m.LastStateMvSrc = srcAddress
	m.LastStateMvDst = dstAddress
	m.LastStateMvOpts = opts

	if m.StateMvErr != nil {
		return nil, m.StateMvErr
	}

	result := m.StateMvResult
	if result == nil {
		result = NewMockResult("Moved "+srcAddress+" to "+dstAddress, 0)
	}
	return result, nil
}

// StatePull implements ExecutorInterface.
func (m *MockExecutor) StatePull(_ context.Context, opts terraform.StatePullOptions) (*terraform.ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatePullCalls++
	m.LastStatePullOpts = opts

	if m.StatePullErr != nil {
		return nil, m.StatePullErr
	}

	result := m.StatePullResult
	if result == nil {
		result = NewMockResult(`{"version":4}`, 0)
	}
	return result, nil
}

// Show implements ExecutorInterface.
func (m *MockExecutor) Show(_ context.Context, planFile string, opts terraform.ShowOptions) (*terraform.ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShowCalls++
	m.LastShowPlanFile = planFile
	m.LastShowOpts = opts

	if m.ShowErr != nil {
		return nil, m.ShowErr
	}

	result := m.ShowResult
	if result == nil {
		result = NewMockResult("{}", 0)
	}
	return result, nil
}

// Version implements ExecutorInterface.
func (m *MockExecutor) Version() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.VersionCalls++

	if m.VersionErr != nil {
		return "", m.VersionErr
	}
	return m.VersionResult, nil
}

// WorkDir implements ExecutorInterface.
func (m *MockExecutor) WorkDir() string {
	return m.MockWorkDir
}

// CloneWithWorkDir implements ExecutorInterface.
// Returns nil for mock - tests typically don't need the actual clone behavior.
func (m *MockExecutor) CloneWithWorkDir(_ string) (*terraform.Executor, error) {
	return nil, nil //nolint:nilnil // Intentional for mock - callers check if result is nil
}

// Reset clears all call counts and last arguments.
func (m *MockExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.InitCalls = 0
	m.PlanCalls = 0
	m.ApplyCalls = 0
	m.RefreshCalls = 0
	m.ValidateCalls = 0
	m.FormatCalls = 0
	m.StateListCalls = 0
	m.StateShowCalls = 0
	m.StateRmCalls = 0
	m.StateMvCalls = 0
	m.StatePullCalls = 0
	m.ShowCalls = 0
	m.VersionCalls = 0

	m.LastPlanOpts = terraform.PlanOptions{}
	m.LastApplyOpts = terraform.ApplyOptions{}
	m.LastRefreshOpts = terraform.RefreshOptions{}
	m.LastValidateOpts = terraform.ValidateOptions{}
	m.LastFormatOpts = terraform.FormatOptions{}
	m.LastStateListOpts = terraform.StateListOptions{}
	m.LastStateShowAddr = ""
	m.LastStateShowOpts = terraform.StateShowOptions{}
	m.LastStateRmAddr = ""
	m.LastStateRmOpts = terraform.StateRmOptions{}
	m.LastStateMvSrc = ""
	m.LastStateMvDst = ""
	m.LastStateMvOpts = terraform.StateMvOptions{}
	m.LastStatePullOpts = terraform.StatePullOptions{}
	m.LastShowPlanFile = ""
	m.LastShowOpts = terraform.ShowOptions{}
}

// Verify MockExecutor implements ExecutorInterface at compile time.
var _ terraform.ExecutorInterface = (*MockExecutor)(nil)
