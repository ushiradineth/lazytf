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
	UseJSON bool
}

// ApplyOptions controls apply execution.
type ApplyOptions struct {
	Flags       []string
	Timeout     time.Duration
	Env         []string
	AutoApprove bool
	UseJSON     bool
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
func (e *Executor) Init(ctx context.Context) (*ExecutionResult, error) {
	result, _, err := e.run(ctx, []string{"init"}, execOptions{})
	return result, err
}

// Plan runs terraform plan and streams output.
func (e *Executor) Plan(ctx context.Context, opts PlanOptions) (*ExecutionResult, <-chan string, error) {
	args := []string{"plan"}
	args = append(args, e.defaultFlags...)
	args = append(args, opts.Flags...)
	if opts.UseJSON && !containsFlag(args, "-json") {
		args = append(args, "-json")
	}
	return e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env})
}

// Apply runs terraform apply and streams output.
func (e *Executor) Apply(ctx context.Context, opts ApplyOptions) (*ExecutionResult, <-chan string, error) {
	args := []string{"apply"}
	args = append(args, e.defaultFlags...)
	args = append(args, opts.Flags...)
	if opts.UseJSON && !containsFlag(args, "-json") {
		args = append(args, "-json")
	}
	if opts.AutoApprove && !containsFlag(args, "-auto-approve") {
		args = append(args, "-auto-approve")
	}
	return e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env})
}

// ShowJSON runs terraform show -json on a plan file.
func (e *Executor) ShowJSON(ctx context.Context, planFile string, opts ShowOptions) (*ExecutionResult, error) {
	if strings.TrimSpace(planFile) == "" {
		return nil, errors.New("plan file path is required")
	}
	args := []string{"show", "-json", planFile}
	result, _, err := e.run(ctx, args, execOptions{timeout: opts.Timeout, env: opts.Env})
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
	result, _, err := e.run(ctx, []string{"version"}, execOptions{})
	if err != nil {
		return "", err
	}
	<-result.Done()
	if result.Error != nil {
		return "", result.Error
	}
	return strings.TrimSpace(result.Stdout), nil
}

// SupportsJSON checks if the terraform version supports streaming JSON output.
func (e *Executor) SupportsJSON() (bool, error) {
	versionOutput, err := e.Version()
	if err != nil {
		return false, err
	}
	parsed, err := parseTerraformVersion(versionOutput)
	if err != nil {
		return false, err
	}
	return versionAtLeast(parsed, semVersion{major: 0, minor: 15, patch: 3}), nil
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
	timeout time.Duration
	env     []string
}

func (e *Executor) run(ctx context.Context, args []string, opts execOptions) (*ExecutionResult, <-chan string, error) {
	if e == nil {
		return nil, nil, errors.New("executor is nil")
	}
	if strings.TrimSpace(e.terraformPath) == "" {
		return nil, nil, errors.New("terraform path is not set")
	}

	timeout := opts.timeout
	if timeout <= 0 {
		timeout = e.timeout
	}
	cmdCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		cmdCtx, cancel = context.WithCancel(ctx)
	}

	cmd := exec.CommandContext(cmdCtx, e.terraformPath, args...)
	cmd.Dir = e.workDir
	cmd.Env = mergeEnv(os.Environ(), e.env, opts.env)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("stderr pipe: %w", err)
	}

	result := &ExecutionResult{
		done: make(chan struct{}),
	}
	outputChan := make(chan string, 100)
	start := time.Now()

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("start terraform: %w", err)
	}

	var stdoutBuf strings.Builder
	var stderrBuf strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)
	go streamLines(stdoutPipe, outputChan, &stdoutBuf, &wg)
	go streamLines(stderrPipe, outputChan, &stderrBuf, &wg)

	go func() {
		defer cancel()
		defer close(result.done)

		waitErr := cmd.Wait()
		wg.Wait()

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
		close(outputChan)
	}()

	return result, outputChan, nil
}

func streamLines(reader io.Reader, output chan<- string, buffer *strings.Builder, wg *sync.WaitGroup) {
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
	if path, err := exec.LookPath("terraform"); err == nil {
		return path, nil
	}

	commonPaths := []string{
		"/usr/local/bin/terraform",
		"/opt/homebrew/bin/terraform",
		"/usr/bin/terraform",
	}
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", errors.New("terraform binary not found in PATH")
}

func validateTerraformPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("terraform binary not found: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("terraform path is a directory: %s", path)
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

type semVersion struct {
	major int
	minor int
	patch int
}

func parseTerraformVersion(output string) (semVersion, error) {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return semVersion{}, errors.New("empty version output")
	}
	first := strings.TrimSpace(lines[0])
	first = strings.TrimPrefix(first, "Terraform ")
	first = strings.TrimPrefix(first, "v")
	first = strings.TrimPrefix(first, "Terraform v")
	first = strings.TrimSpace(first)
	if idx := strings.IndexFunc(first, func(r rune) bool { return (r < '0' || r > '9') && r != '.' }); idx >= 0 {
		first = first[:idx]
	}

	var v semVersion
	_, err := fmt.Sscanf(first, "%d.%d.%d", &v.major, &v.minor, &v.patch)
	if err != nil {
		return semVersion{}, fmt.Errorf("parse version: %w", err)
	}
	return v, nil
}

func versionAtLeast(version semVersion, minimum semVersion) bool {
	if version.major != minimum.major {
		return version.major > minimum.major
	}
	if version.minor != minimum.minor {
		return version.minor > minimum.minor
	}
	return version.patch >= minimum.patch
}
