package terraform

import "context"

// ExecutorInterface defines the interface for terraform command execution.
// This interface enables dependency injection and testing.
type ExecutorInterface interface {
	// Init runs terraform init.
	Init(ctx context.Context) (*ExecutionResult, error)

	// Plan runs terraform plan with streaming output.
	Plan(ctx context.Context, opts PlanOptions) (*ExecutionResult, <-chan string, error)

	// Apply runs terraform apply with streaming output.
	Apply(ctx context.Context, opts ApplyOptions) (*ExecutionResult, <-chan string, error)

	// Refresh runs terraform refresh with streaming output.
	Refresh(ctx context.Context, opts RefreshOptions) (*ExecutionResult, <-chan string, error)

	// Validate runs terraform validate.
	Validate(ctx context.Context, opts ValidateOptions) (*ExecutionResult, error)

	// Format runs terraform fmt.
	Format(ctx context.Context, opts FormatOptions) (*ExecutionResult, error)

	// StateList runs terraform state list.
	StateList(ctx context.Context, opts StateListOptions) (*ExecutionResult, error)

	// StateShow runs terraform state show for a specific address.
	StateShow(ctx context.Context, address string, opts StateShowOptions) (*ExecutionResult, error)

	// Show runs terraform show on a plan file.
	Show(ctx context.Context, planFile string, opts ShowOptions) (*ExecutionResult, error)

	// Version returns the terraform version.
	Version() (string, error)

	// WorkDir returns the working directory.
	WorkDir() string

	// CloneWithWorkDir creates a new executor with a different working directory.
	CloneWithWorkDir(workDir string) (*Executor, error)
}

// Verify that *Executor implements ExecutorInterface at compile time.
var _ ExecutorInterface = (*Executor)(nil)
