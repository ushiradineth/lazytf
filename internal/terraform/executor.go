package terraform

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ushiradineth/lazytf/internal/tfbinary"
)

const defaultTimeout = 10 * time.Minute

// Executor runs terraform commands in a working directory.
type Executor struct {
	workDir       string
	terraformPath string
	defaultFlags  []string
	env           []string
	timeout       time.Duration
}

// ExecutionResult captures the result of a terraform command.
type ExecutionResult struct {
	Output   string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
	done     chan struct{}
}

// NewExecutionResult creates a result with a completion channel.
func NewExecutionResult() *ExecutionResult {
	return &ExecutionResult{done: make(chan struct{})}
}

// Done returns a channel that closes when the command finishes.
func (r *ExecutionResult) Done() <-chan struct{} {
	if r == nil {
		return nil
	}
	return r.done
}

// Finish closes the completion channel if it is still open.
func (r *ExecutionResult) Finish() {
	if r == nil {
		return
	}
	if r.done == nil {
		r.done = make(chan struct{})
		close(r.done)
		return
	}
	select {
	case <-r.done:
		return
	default:
		close(r.done)
	}
}

// ExecutorOption configures an Executor.
type ExecutorOption func(*Executor) error

// WithTerraformPath sets the terraform binary path.
func WithTerraformPath(path string) ExecutorOption {
	return func(e *Executor) error {
		if strings.TrimSpace(path) == "" {
			return errors.New("terraform path cannot be empty")
		}
		e.terraformPath = path
		return nil
	}
}

// WithDefaultFlags sets default flags to include with plan/apply.
func WithDefaultFlags(flags []string) ExecutorOption {
	return func(e *Executor) error {
		e.defaultFlags = append([]string{}, flags...)
		return nil
	}
}

// WithEnv adds extra environment variables.
func WithEnv(env []string) ExecutorOption {
	return func(e *Executor) error {
		e.env = append([]string{}, env...)
		return nil
	}
}

// WithTimeout sets the default command timeout.
func WithTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) error {
		e.timeout = timeout
		return nil
	}
}

// PlanOptions controls plan execution.
type PlanOptions struct {
	Flags   []string
	Timeout time.Duration
	Env     []string
}

// ApplyOptions controls apply execution.
type ApplyOptions struct {
	Flags       []string
	Timeout     time.Duration
	Env         []string
	AutoApprove bool
}

// RefreshOptions controls refresh execution.
type RefreshOptions struct {
	Flags   []string
	Timeout time.Duration
	Env     []string
}

// InitOptions controls init execution.
type InitOptions struct {
	Upgrade bool
	Timeout time.Duration
	Env     []string
}

// ValidateOptions controls validate execution.
type ValidateOptions struct {
	Timeout time.Duration
	Env     []string
}

// FormatOptions controls fmt execution.
type FormatOptions struct {
	Recursive bool
	Check     bool
	Timeout   time.Duration
	Env       []string
}

// StateListOptions controls state list execution.
type StateListOptions struct {
	Timeout time.Duration
	Env     []string
}

// StateShowOptions controls state show execution.
type StateShowOptions struct {
	Timeout time.Duration
	Env     []string
}

// StateRmOptions controls state rm execution.
type StateRmOptions struct {
	Timeout time.Duration
	Env     []string
}

// StateMvOptions controls state mv execution.
type StateMvOptions struct {
	Timeout time.Duration
	Env     []string
}

// StatePullOptions controls state pull execution.
type StatePullOptions struct {
	Timeout time.Duration
	Env     []string
}

// ShowOptions controls terraform show execution.
type ShowOptions struct {
	Timeout time.Duration
	Env     []string
}

// NewExecutor creates a terraform executor.
func NewExecutor(workDir string, opts ...ExecutorOption) (*Executor, error) {
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	absDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workdir: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("workdir not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workdir is not a directory: %s", absDir)
	}

	exec := &Executor{
		workDir: absDir,
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		if err := opt(exec); err != nil {
			return nil, err
		}
	}

	if exec.terraformPath == "" {
		path, err := resolveTerraformPath()
		if err != nil {
			return nil, err
		}
		exec.terraformPath = path
	}

	if err := validateTerraformPath(exec.terraformPath); err != nil {
		return nil, err
	}

	return exec, nil
}

// Init runs terraform init.
func (e *Executor) Init(ctx context.Context, opts InitOptions) (*ExecutionResult, error) {
	args := []string{"init"}
	if opts.Upgrade {
		args = append(args, "-upgrade")
	}

	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: false})
	if err != nil {
		return nil, err
	}
	<-result.Done()
	return result, nil
}

// Plan runs terraform plan and streams output.
func (e *Executor) Plan(ctx context.Context, opts PlanOptions) (*ExecutionResult, <-chan string, error) {
	args := make([]string, 0, 1+len(e.defaultFlags)+len(opts.Flags))
	args = append(args, "plan")
	args = append(args, e.defaultFlags...)
	args = append(args, opts.Flags...)
	return e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: true})
}

// Apply runs terraform apply and streams output.
func (e *Executor) Apply(ctx context.Context, opts ApplyOptions) (*ExecutionResult, <-chan string, error) {
	args := []string{"apply"}
	args = append(args, e.defaultFlags...)
	args = append(args, opts.Flags...)
	if opts.AutoApprove && !containsFlag(args, "-auto-approve") {
		args = insertAutoApproveBeforePlanFile(args)
	}
	return e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: true})
}

func insertAutoApproveBeforePlanFile(args []string) []string {
	for i, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if strings.HasPrefix(trimmed, "-") || !strings.HasSuffix(trimmed, ".tfplan") {
			continue
		}
		updated := make([]string, 0, len(args)+1)
		updated = append(updated, args[:i]...)
		updated = append(updated, "-auto-approve")
		updated = append(updated, args[i:]...)
		return updated
	}
	return append(args, "-auto-approve")
}

// Refresh runs terraform apply -refresh-only and streams output.
func (e *Executor) Refresh(ctx context.Context, opts RefreshOptions) (*ExecutionResult, <-chan string, error) {
	args := make([]string, 0, 3+len(e.defaultFlags)+len(opts.Flags))
	args = append(args, "apply", "-refresh-only", "-auto-approve")
	args = append(args, e.defaultFlags...)
	args = append(args, opts.Flags...)
	return e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: true})
}

// Validate runs terraform validate -json and returns the result.
func (e *Executor) Validate(ctx context.Context, opts ValidateOptions) (*ExecutionResult, error) {
	args := []string{"validate", "-json"}
	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: false})
	if err != nil {
		return nil, err
	}
	<-result.Done()
	return result, nil
}

// Format runs terraform fmt and returns the list of changed files.
func (e *Executor) Format(ctx context.Context, opts FormatOptions) (*ExecutionResult, error) {
	args := []string{"fmt"}
	if opts.Recursive {
		args = append(args, "-recursive")
	}
	if opts.Check {
		args = append(args, "-check")
	}
	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: false})
	if err != nil {
		return nil, err
	}
	<-result.Done()
	return result, nil
}

// StateList runs terraform state list and returns the resource addresses.
func (e *Executor) StateList(ctx context.Context, opts StateListOptions) (*ExecutionResult, error) {
	args := []string{"state", "list"}
	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: false})
	if err != nil {
		return nil, err
	}
	<-result.Done()
	return result, nil
}

// StateShow runs terraform state show for a specific resource address.
func (e *Executor) StateShow(ctx context.Context, address string, opts StateShowOptions) (*ExecutionResult, error) {
	if strings.TrimSpace(address) == "" {
		return nil, errors.New("resource address is required")
	}
	args := []string{"state", "show", address}
	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: false})
	if err != nil {
		return nil, err
	}
	<-result.Done()
	return result, nil
}

// StateRm runs terraform state rm for a specific resource address.
func (e *Executor) StateRm(ctx context.Context, address string, opts StateRmOptions) (*ExecutionResult, error) {
	if strings.TrimSpace(address) == "" {
		return nil, errors.New("resource address is required")
	}
	args := []string{"state", "rm", address}
	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: false})
	if err != nil {
		return nil, err
	}
	<-result.Done()
	return result, nil
}

// StateMv runs terraform state mv for a source and destination resource address.
func (e *Executor) StateMv(ctx context.Context, srcAddress, dstAddress string, opts StateMvOptions) (*ExecutionResult, error) {
	if strings.TrimSpace(srcAddress) == "" {
		return nil, errors.New("source resource address is required")
	}
	if strings.TrimSpace(dstAddress) == "" {
		return nil, errors.New("destination resource address is required")
	}
	args := []string{"state", "mv", srcAddress, dstAddress}
	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: false})
	if err != nil {
		return nil, err
	}
	<-result.Done()
	return result, nil
}

// StatePull runs terraform state pull to get raw state JSON.
func (e *Executor) StatePull(ctx context.Context, opts StatePullOptions) (*ExecutionResult, error) {
	args := []string{"state", "pull"}
	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: false})
	if err != nil {
		return nil, err
	}
	<-result.Done()
	return result, nil
}

// Show runs terraform show on a plan file.
func (e *Executor) Show(ctx context.Context, planFile string, opts ShowOptions) (*ExecutionResult, error) {
	if strings.TrimSpace(planFile) == "" {
		return nil, errors.New("plan file path is required")
	}
	args := []string{"show", planFile}
	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env, streamOutput: false})
	if err != nil {
		return nil, err
	}
	<-result.Done()
	if result.Error != nil {
		return result, result.Error
	}
	return result, nil
}

// Version returns terraform version output.
func (e *Executor) Version() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, _, err := e.run(ctx, []string{"version"}, execOptions{streamOutput: false})
	if err != nil {
		return "", err
	}
	<-result.Done()
	if result.Error != nil {
		return "", result.Error
	}
	return strings.TrimSpace(result.Stdout), nil
}

// WorkDir returns the executor working directory.
func (e *Executor) WorkDir() string {
	if e == nil {
		return ""
	}
	return e.workDir
}

// CloneWithWorkDir returns a new executor with the same settings but a new workdir.
func (e *Executor) CloneWithWorkDir(workDir string) (*Executor, error) {
	if e == nil {
		return nil, errors.New("executor is nil")
	}
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	absDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workdir: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("workdir not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workdir is not a directory: %s", absDir)
	}

	clone := &Executor{
		workDir:       absDir,
		terraformPath: e.terraformPath,
		defaultFlags:  append([]string{}, e.defaultFlags...),
		env:           append([]string{}, e.env...),
		timeout:       e.timeout,
	}
	if clone.terraformPath == "" {
		path, err := resolveTerraformPath()
		if err != nil {
			return nil, err
		}
		clone.terraformPath = path
	}
	if err := validateTerraformPath(clone.terraformPath); err != nil {
		return nil, err
	}
	return clone, nil
}

type execOptions struct {
	timeout      time.Duration
	env          []string
	streamOutput bool
}

func (e *Executor) run(ctx context.Context, args []string, opts execOptions) (*ExecutionResult, <-chan string, error) {
	if err := e.validateRun(); err != nil {
		return nil, nil, err
	}

	cmdCtx, cancel := e.commandContext(ctx, opts)
	cmd := e.buildCommand(cmdCtx, args, opts.env)

	stdoutPipe, stderrPipe, err := setupCommandPipes(cmd)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	result, outputChan, start := newExecutionResult(opts.streamOutput)

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("start terraform: %w", err)
	}

	var stdoutBuf strings.Builder
	var stderrBuf strings.Builder
	var wg sync.WaitGroup
	streamErrs := make(chan error, 2)
	wg.Add(2)
	go streamLines(stdoutPipe, outputChan, &stdoutBuf, streamErrs, &wg)
	go streamLines(stderrPipe, outputChan, &stderrBuf, streamErrs, &wg)

	go finalizeExecution(cmdCtx, cmd, cancel, &wg, &stdoutBuf, &stderrBuf, result, outputChan, streamErrs, start)

	return result, outputChan, nil
}

func (e *Executor) validateRun() error {
	if e == nil {
		return errors.New("executor is nil")
	}
	if strings.TrimSpace(e.terraformPath) == "" {
		return errors.New("terraform path is not set")
	}
	return nil
}

func (e *Executor) commandContext(ctx context.Context, opts execOptions) (context.Context, context.CancelFunc) {
	timeout := opts.timeout
	if timeout <= 0 {
		timeout = e.timeout
	}
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

func (e *Executor) buildCommand(ctx context.Context, args []string, extraEnv []string) *exec.Cmd {
	// #nosec G204 -- terraform execution is intentional and arguments come from configured inputs.
	cmd := exec.CommandContext(ctx, e.terraformPath, args...)
	cmd.Dir = e.workDir
	cmd.Env = mergeEnv(os.Environ(), e.env, extraEnv)
	return cmd
}

func setupCommandPipes(cmd *exec.Cmd) (io.ReadCloser, io.ReadCloser, error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stderr pipe: %w", err)
	}
	return stdoutPipe, stderrPipe, nil
}

func newExecutionResult(streamOutput bool) (*ExecutionResult, chan string, time.Time) {
	result := &ExecutionResult{
		done: make(chan struct{}),
	}
	var outputChan chan string
	if streamOutput {
		// Large buffer to prevent blocking when terraform outputs many lines quickly.
		outputChan = make(chan string, 1000)
	}
	return result, outputChan, time.Now()
}

func finalizeExecution(
	cmdCtx context.Context,
	cmd *exec.Cmd,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
	stdoutBuf *strings.Builder,
	stderrBuf *strings.Builder,
	result *ExecutionResult,
	outputChan chan string,
	streamErrs chan error,
	start time.Time,
) {
	defer cancel()
	defer close(result.done)

	wg.Wait()
	waitErr := cmd.Wait()
	close(streamErrs)

	streamErr := collectStreamError(streamErrs)

	duration := time.Since(start)
	stdout := strings.TrimRight(stdoutBuf.String(), "\n")
	stderr := strings.TrimRight(stderrBuf.String(), "\n")

	result.Stdout = stdout
	result.Stderr = stderr
	result.Output = stdout
	if stderr != "" {
		if result.Output != "" {
			result.Output += "\n"
		}
		result.Output += stderr
	}
	result.Duration = duration
	result.ExitCode = exitCode(waitErr)
	if waitErr != nil {
		switch {
		case errors.Is(cmdCtx.Err(), context.DeadlineExceeded):
			result.Error = cmdCtx.Err()
		case errors.Is(cmdCtx.Err(), context.Canceled):
			result.Error = cmdCtx.Err()
		default:
			result.Error = waitErr
		}
	}
	if streamErr != nil {
		if result.Error != nil {
			result.Error = errors.Join(result.Error, streamErr)
		} else {
			result.Error = streamErr
		}
	}
	if outputChan != nil {
		close(outputChan)
	}
}

func streamLines(reader io.Reader, output chan<- string, buffer *strings.Builder, streamErrs chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if buffer != nil {
			buffer.WriteString(line)
			buffer.WriteString("\n")
		}
		if output != nil {
			output <- line
		}
	}

	if err := scanner.Err(); err != nil && !isIgnorableScannerError(err) {
		streamErrs <- err
	}
}

func isIgnorableScannerError(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrClosed) || errors.Is(err, io.EOF) {
		return true
	}
	return strings.Contains(err.Error(), "file already closed")
}

func collectStreamError(streamErrs <-chan error) error {
	var joined error
	for err := range streamErrs {
		if err == nil {
			continue
		}
		joined = errors.Join(joined, err)
	}
	return joined
}

func mergeEnv(base []string, sets ...[]string) []string {
	envMap := make(map[string]string)
	for _, item := range base {
		key, val := splitEnv(item)
		envMap[key] = val
	}
	for _, set := range sets {
		for _, item := range set {
			key, val := splitEnv(item)
			envMap[key] = val
		}
	}

	merged := make([]string, 0, len(envMap))
	for key, val := range envMap {
		merged = append(merged, key+"="+val)
	}
	return merged
}

func splitEnv(item string) (string, string) {
	parts := strings.SplitN(item, "=", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func resolveTerraformPath() (string, error) {
	return tfbinary.Resolve()
}

func validateTerraformPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("terraform/tofu binary not found: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("terraform/tofu path is a directory: %s", path)
	}
	return nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func containsFlag(flags []string, target string) bool {
	for _, flag := range flags {
		if flag == target {
			return true
		}
	}
	return false
}
