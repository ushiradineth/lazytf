package terraform

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ushiradineth/lazytf/internal/consts"
)

func TestMergeEnvOverrides(t *testing.T) {
	base := []string{"FOO=1", "BAR=2"}
	set := []string{"FOO=3", "BAZ=4"}
	merged := mergeEnv(base, set)

	got := map[string]string{}
	for _, item := range merged {
		key, val := splitEnv(item)
		got[key] = val
	}

	want := map[string]string{"FOO": "3", "BAR": "2", "BAZ": "4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected env merge: %#v", got)
	}
}

func TestContainsFlag(t *testing.T) {
	flags := []string{"-lock=false", "-auto-approve"}
	if !containsFlag(flags, "-auto-approve") {
		t.Fatalf("expected flag to be found")
	}
	if containsFlag(flags, "-input=false") {
		t.Fatalf("did not expect flag to be found")
	}
}

func TestNewExecutorWorkdirValidation(t *testing.T) {
	dir := t.TempDir()
	if _, err := NewExecutor(filepath.Join(dir, "missing")); err == nil {
		t.Fatalf("expected error for missing workdir")
	}

	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := NewExecutor(file); err == nil {
		t.Fatalf("expected error for non-directory workdir")
	}
}

func TestNewExecutorResolvesPathFromEnv(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformArgsOnly(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	if exec.terraformPath != tfPath {
		t.Fatalf("expected terraform path %q, got %q", tfPath, exec.terraformPath)
	}
}

func TestNewExecutorInvalidTerraformPath(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	if _, err := NewExecutor(dir, WithTerraformPath(filepath.Join(dir, "missing"))); err == nil {
		t.Fatalf("expected error for missing terraform path")
	}

	subdir := filepath.Join(dir, "dir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := NewExecutor(dir, WithTerraformPath(subdir)); err == nil {
		t.Fatalf("expected error for directory terraform path")
	}
}

func TestExecutorPlanApplyFlagsAndAutoApprove(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraform(t, dir)
	exec, err := NewExecutor(dir,
		WithTerraformPath(tfPath),
		WithDefaultFlags([]string{"-foo", "-bar=1"}),
	)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	ctx := context.Background()
	result, output, err := exec.Plan(ctx, PlanOptions{Flags: []string{"-baz"}})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	planOutput := collectOutput(output)
	<-result.Done()
	if (planOutput != "" || result.Stdout != "") &&
		!strings.Contains(planOutput, "ARGS:plan -foo -bar=1 -baz") &&
		!strings.Contains(result.Stdout, "ARGS:plan -foo -bar=1 -baz") {
		t.Fatalf("unexpected plan args: %q", result.Stdout)
	}

	applyResult, applyOutputChan, err := exec.Apply(ctx, ApplyOptions{Flags: []string{"-baz"}, AutoApprove: true})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	applyOutput := collectOutput(applyOutputChan)
	<-applyResult.Done()
	if (applyOutput != "" || applyResult.Stdout != "") &&
		!strings.Contains(applyOutput, "ARGS:apply -foo -bar=1 -baz -auto-approve") &&
		!strings.Contains(applyResult.Stdout, "ARGS:apply -foo -bar=1 -baz -auto-approve") {
		t.Fatalf("expected auto-approve flag, got %q", applyResult.Stdout)
	}
}

func TestExecutorVersion(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraform(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	version, err := exec.Version()
	if err != nil {
		t.Fatalf("version error: %v", err)
	}
	if version == "" {
		_, err = exec.Version()
		if err != nil {
			t.Fatalf("version retry error: %v", err)
		}
	}
}

func TestExecutorStreamingStdoutStderr(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraform(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, output, err := exec.Plan(context.Background(), PlanOptions{})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	lines := collectLines(output, result)
	if len(lines) == 0 {
		lines = appendResultLines(lines, result.Stdout)
		lines = appendResultLines(lines, result.Stderr)
	}
	if !containsLine(lines, "stdout plan") {
		t.Fatalf("expected stdout line, got %#v", lines)
	}
	if !containsLine(lines, "stderr plan") {
		t.Fatalf("expected stderr line, got %#v", lines)
	}
}

func TestExecutorExitCodeMapping(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraform(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, _, err := exec.Apply(context.Background(), ApplyOptions{Flags: []string{"exit7"}})
	if err != nil {
		t.Fatalf("apply start error: %v", err)
	}
	<-result.Done()
	if result.ExitCode != 7 {
		t.Fatalf("expected exit code 7, got %d", result.ExitCode)
	}
}

func TestExecutorTimeoutAndCancel(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraform(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, _, err := exec.Plan(context.Background(), PlanOptions{Timeout: 30 * time.Millisecond, Flags: []string{"sleep"}})
	if err != nil {
		t.Fatalf("plan start error: %v", err)
	}
	<-result.Done()
	if !errors.Is(result.Error, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", result.Error)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelResult, _, err := exec.Plan(ctx, PlanOptions{Flags: []string{"sleep"}})
	if err != nil {
		t.Fatalf("plan start error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-cancelResult.Done()
	if !errors.Is(cancelResult.Error, context.Canceled) {
		t.Fatalf("expected canceled, got %v", cancelResult.Error)
	}
}

func TestExecutorEnvPrecedence(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformEnvTest(t, dir)
	t.Setenv("FOO", "base")
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath), WithEnv([]string{"FOO=exec", "BAR=exec"}))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	const expected = "ENV:FOO=opts BAR=exec"
	var stdout string
	for range 3 {
		result, _, runErr := exec.Plan(context.Background(), PlanOptions{Flags: []string{"envtest"}, Env: []string{"FOO=opts"}})
		if runErr != nil {
			t.Fatalf("plan start error: %v", runErr)
		}
		<-result.Done()
		stdout = result.Stdout
		if strings.Contains(stdout, expected) {
			return
		}
	}
	t.Fatalf("unexpected env output after retries: %q", stdout)
}

func TestExecutorSymlinkedTerraformPath(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("symlink test not supported on windows")
	}
	dir := t.TempDir()
	target := writeFakeTerraform(t, dir)
	link := filepath.Join(dir, "terraform-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	exec, err := NewExecutor(dir, WithTerraformPath(link))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, _, err := exec.Plan(context.Background(), PlanOptions{})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	<-result.Done()
	if result.ExitCode != 0 {
		t.Fatalf("expected plan to succeed")
	}
}

func TestExitCodeMissingCommand(t *testing.T) {
	if got := exitCode(errors.New("missing")); got != -1 {
		t.Fatalf("expected exit code -1, got %d", got)
	}
}

func TestExecutorInterleavedOutputOrder(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeInterleaveTerraform(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, output, err := exec.Plan(context.Background(), PlanOptions{})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	lines := collectLines(output, result)
	if !inOrder(lines, []string{"out-1", "out-2"}) {
		t.Fatalf("stdout lines out of order: %#v", lines)
	}
	if !inOrder(lines, []string{"err-1", "err-2"}) {
		t.Fatalf("stderr lines out of order: %#v", lines)
	}
}

func TestExecutionResultFinish(t *testing.T) {
	result := NewExecutionResult()
	select {
	case <-result.Done():
		t.Fatalf("expected open result channel")
	default:
	}

	result.Finish()
	select {
	case <-result.Done():
	default:
		t.Fatalf("expected closed result channel")
	}

	result.Finish()
}

func TestExecutionResultFinishNilChannel(t *testing.T) {
	result := &ExecutionResult{}
	result.Finish()
	if result.Done() == nil {
		t.Fatalf("expected Done channel to be initialized")
	}
	select {
	case <-result.Done():
	default:
		t.Fatalf("expected Done channel to be closed")
	}
}

func TestExecutionResultDoneNil(t *testing.T) {
	var result *ExecutionResult
	if result.Done() != nil {
		t.Fatalf("expected nil done channel")
	}
}

func TestExecutorInitAndShow(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraform(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.Init(context.Background())
	if err != nil {
		t.Fatalf("init error: %v", err)
	}
	<-result.Done()

	planFile := filepath.Join(dir, "plan.tfplan")
	if err := os.WriteFile(planFile, []byte("plan"), 0o600); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	showResult, err := exec.Show(context.Background(), planFile, ShowOptions{})
	if err != nil {
		t.Fatalf("show error: %v", err)
	}
	if showResult.ExitCode != 0 {
		t.Fatalf("expected successful show, got exit code %d", showResult.ExitCode)
	}
	if showResult.Output != "" && !strings.Contains(showResult.Output, "show output") {
		t.Fatalf("unexpected show output: %q", showResult.Output)
	}
}

func TestExecutorWorkDirAndClone(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraform(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	if exec.WorkDir() == "" {
		t.Fatalf("expected workdir to be set")
	}

	cloneDir := t.TempDir()
	clone, err := exec.CloneWithWorkDir(cloneDir)
	if err != nil {
		t.Fatalf("clone error: %v", err)
	}
	if clone.WorkDir() == exec.WorkDir() {
		t.Fatalf("expected clone workdir to differ")
	}

	var nilExec *Executor
	if _, err := nilExec.CloneWithWorkDir(cloneDir); err == nil {
		t.Fatalf("expected error for nil executor")
	}
}

func TestWithTimeoutOption(t *testing.T) {
	exec := &Executor{}
	if err := WithTimeout(5 * time.Second)(exec); err != nil {
		t.Fatalf("timeout option error: %v", err)
	}
	if exec.timeout != 5*time.Second {
		t.Fatalf("expected timeout to be set")
	}
}

func TestResolveTerraformPath(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	writeFakeTerraform(t, dir)
	t.Setenv("PATH", dir)

	path, err := resolveTerraformPath()
	if err != nil {
		t.Fatalf("resolve terraform path: %v", err)
	}
	if path == "" {
		t.Fatalf("expected terraform path")
	}
}

func TestResolveTerraformPathMissing(t *testing.T) {
	t.Setenv("PATH", "")
	if _, err := resolveTerraformPath(); err == nil {
		t.Fatalf("expected resolve terraform path error")
	}
}

func TestExecutorWorkDirNil(t *testing.T) {
	var exec *Executor
	if exec.WorkDir() != "" {
		t.Fatalf("expected empty workdir for nil executor")
	}
}

func writeFakeTerraform(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "terraform")
	script := `#!/bin/sh
cmd="$1"
shift
if [ "$cmd" = "version" ]; then
  echo "Terraform v1.0.0"
  exit 0
fi
if [ "$cmd" = "show" ]; then
  echo "show output"
  exit 0
fi
if [ "$cmd" = "plan" ] || [ "$cmd" = "apply" ]; then
  echo "stdout $cmd"
  echo "stderr $cmd" 1>&2
  sleep 0.05
fi
if [ "$cmd" = "init" ]; then
  echo "stdout init"
fi
if [ "$1" = "sleep" ]; then
  sleep 1
fi
for arg in "$@"; do
  if [ "$arg" = "envtest" ]; then
    echo "ENV:FOO=$FOO BAR=$BAR"
    break
  fi
done
if [ "$1" = "exit7" ]; then
  exit 7
fi
echo "ARGS:$cmd $*"
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func writeFakeTerraformArgsOnly(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "terraform")
	script := `#!/bin/sh
cmd="$1"
shift
if [ "$cmd" = "version" ]; then
  echo "Terraform v1.0.0"
  exit 0
fi
echo "ARGS:$cmd $*"
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func writeFakeTerraformEnvTest(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "terraform")
	script := `#!/bin/sh
cmd="$1"
shift
if [ "$cmd" = "plan" ]; then
  echo "ENV:FOO=$FOO BAR=$BAR"
fi
echo "ARGS:$cmd $*"
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func writeInterleaveTerraform(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "terraform")
	script := `#!/bin/sh
cmd="$1"
if [ "$cmd" = "plan" ]; then
  echo "out-1"
  sleep 0.01
  echo "err-1" 1>&2
  sleep 0.01
  echo "out-2"
  sleep 0.01
  echo "err-2" 1>&2
fi
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func collectLines(ch <-chan string, result *ExecutionResult) []string {
	lines := []string{}
	for line := range ch {
		lines = append(lines, line)
	}
	if result != nil {
		<-result.Done()
	}
	return lines
}

func collectOutput(ch <-chan string) string {
	var b strings.Builder
	for line := range ch {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func containsLine(lines []string, target string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}

func inOrder(lines []string, seq []string) bool {
	if len(seq) == 0 {
		return true
	}
	index := 0
	for _, line := range lines {
		if index >= len(seq) {
			return true
		}
		if strings.TrimSpace(line) == seq[index] {
			index++
			if index == len(seq) {
				return true
			}
		}
	}
	return false
}

func appendResultLines(lines []string, content string) []string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}

func TestExecutorRefresh(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, _, err := exec.Refresh(context.Background(), RefreshOptions{})
	if err != nil {
		t.Fatalf("refresh error: %v", err)
	}
	<-result.Done()
	if result.ExitCode != 0 {
		t.Fatalf("expected refresh to succeed, got exit code %d", result.ExitCode)
	}
}

func TestExecutorValidate(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.Validate(context.Background(), ValidateOptions{})
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	<-result.Done()
	if result.ExitCode != 0 {
		t.Fatalf("expected validate to succeed")
	}
}

func TestExecutorFormat(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.Format(context.Background(), FormatOptions{Check: true})
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	<-result.Done()
}

func TestExecutorStateList(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.StateList(context.Background(), StateListOptions{})
	if err != nil {
		t.Fatalf("state list error: %v", err)
	}
	<-result.Done()
	if result.ExitCode != 0 {
		t.Errorf("expected state list to succeed, got exit code %d", result.ExitCode)
	}
}

func TestExecutorStateShow(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, err := exec.StateShow(context.Background(), "aws_instance.web", StateShowOptions{})
	if err != nil {
		t.Fatalf("state show error: %v", err)
	}
	<-result.Done()
	if result.ExitCode != 0 {
		t.Error("expected state show to succeed")
	}
}

func TestCloneWithWorkDirNilExecutor(t *testing.T) {
	var exec *Executor
	_, err := exec.CloneWithWorkDir("/tmp")
	if err == nil {
		t.Error("expected error for nil executor")
	}
}

func TestCloneWithWorkDirEmptyPath(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	writeFakeTerraform(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	// Empty workdir should default to "."
	clone, err := exec.CloneWithWorkDir("")
	if err != nil {
		t.Fatalf("clone with empty workdir: %v", err)
	}
	if clone.workDir == "" {
		t.Error("expected workdir to be set")
	}
}

func TestCloneWithWorkDirMissingPath(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	writeFakeTerraform(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	// Clone with missing workdir should fail
	_, err = exec.CloneWithWorkDir("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for missing workdir")
	}
}

func TestCloneWithWorkDirFilePath(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	writeFakeTerraform(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	// Create a file
	filePath := filepath.Join(dir, "not-a-dir.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Clone with file path should fail
	_, err = exec.CloneWithWorkDir(filePath)
	if err == nil {
		t.Error("expected error when workdir is a file")
	}
}

func TestWorkDir(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	writeFakeTerraform(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	workDir := exec.WorkDir()
	if workDir == "" {
		t.Error("expected non-empty workdir")
	}
}

func TestWorkDirNilExecutor(t *testing.T) {
	var exec *Executor
	workDir := exec.WorkDir()
	if workDir != "" {
		t.Error("expected empty workdir for nil executor")
	}
}

func TestSplitEnvVariants(t *testing.T) {
	tests := []struct {
		input     string
		wantKey   string
		wantValue string
	}{
		{"FOO=bar", "FOO", "bar"},
		{"KEY=value=with=equals", "KEY", "value=with=equals"},
		{"EMPTY=", "EMPTY", ""},
		{"NOEQUALS", "NOEQUALS", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, value := splitEnv(tt.input)
			if key != tt.wantKey {
				t.Errorf("splitEnv(%q) key = %q, want %q", tt.input, key, tt.wantKey)
			}
			if value != tt.wantValue {
				t.Errorf("splitEnv(%q) value = %q, want %q", tt.input, value, tt.wantValue)
			}
		})
	}
}

func TestRunDisablesOutputStreamForSyncCommands(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}

	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)

	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, output, err := exec.run(context.Background(), []string{"validate", "-json"}, execOptions{streamOutput: false})
	if err != nil {
		t.Fatalf("run validate: %v", err)
	}
	if output != nil {
		t.Fatalf("expected nil output channel for non-streaming command")
	}

	<-result.Done()
	if result.Error != nil {
		t.Fatalf("unexpected result error: %v", result.Error)
	}
}

func TestStreamLinesReportsScannerError(t *testing.T) {
	var (
		wg         sync.WaitGroup
		buffer     strings.Builder
		output     = make(chan string, 1)
		streamErrs = make(chan error, 1)
	)

	wg.Add(1)
	go streamLines(&failingScannerReader{}, output, &buffer, streamErrs, &wg)
	wg.Wait()
	close(streamErrs)

	if got := buffer.String(); !strings.Contains(got, "line") {
		t.Fatalf("expected buffer to include streamed line, got %q", got)
	}

	select {
	case line := <-output:
		if line != "line" {
			t.Fatalf("unexpected streamed line %q", line)
		}
	default:
		t.Fatalf("expected one streamed line")
	}

	err := collectStreamError(streamErrs)
	if err == nil {
		t.Fatalf("expected scanner error")
	}
	if !strings.Contains(err.Error(), "forced read error") {
		t.Fatalf("unexpected scanner error: %v", err)
	}
}

type failingScannerReader struct {
	readOnce bool
}

func (r *failingScannerReader) Read(p []byte) (int, error) {
	if r.readOnce {
		return 0, errors.New("forced read error")
	}
	r.readOnce = true
	return strings.NewReader("line\n").Read(p)
}

var _ io.Reader = (*failingScannerReader)(nil)

func TestExecutorShowWithOptions(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.Show(context.Background(), "plan.tfplan", ShowOptions{})
	if err != nil {
		t.Fatalf("show error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result from show")
	}
}

func TestExecutorShowNilExecutor(t *testing.T) {
	var exec *Executor
	_, err := exec.Show(context.Background(), "plan.tfplan", ShowOptions{})
	if err == nil {
		t.Error("expected error for nil executor")
	}
}

func TestExecutorShowEmptyPlanFile(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = exec.Show(context.Background(), "", ShowOptions{})
	if err == nil {
		t.Error("expected error for empty plan file")
	}
}

func TestExecutorStateShowWithOptions(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := exec.StateShow(context.Background(), "aws_instance.web", StateShowOptions{})
	if err != nil {
		t.Fatalf("state show error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result from state show")
	}
}

func TestExecutorStateShowNilExecutor(t *testing.T) {
	var exec *Executor
	_, err := exec.StateShow(context.Background(), "resource", StateShowOptions{})
	if err == nil {
		t.Error("expected error for nil executor")
	}
}

func TestExecutorStateShowEmptyAddress(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = exec.StateShow(context.Background(), "", StateShowOptions{})
	if err == nil {
		t.Error("expected error for empty address")
	}
}

func TestExecutorFormatSuccess(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraformExtended(t, dir)
	t.Setenv("PATH", dir)

	exec, err := NewExecutor(dir, WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	output, err := exec.Format(context.Background(), FormatOptions{})
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if output == nil {
		t.Error("expected non-nil output from format")
	}
}

func TestExecutorFormatNilExecutor(t *testing.T) {
	var exec *Executor
	_, err := exec.Format(context.Background(), FormatOptions{})
	if err == nil {
		t.Error("expected error for nil executor")
	}
}

//nolint:dupword // shell script has repeated 'fi' keywords
func writeFakeTerraformExtended(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "terraform")
	script := `#!/bin/sh
cmd="$1"
shift
if [ "$cmd" = "version" ]; then
  echo "Terraform v1.0.0"
  exit 0
fi
if [ "$cmd" = "show" ]; then
  echo "show output"
  exit 0
fi
if [ "$cmd" = "refresh" ]; then
  echo "refresh complete"
  exit 0
fi
if [ "$cmd" = "validate" ]; then
  echo "Success! The configuration is valid."
  exit 0
fi
if [ "$cmd" = "fmt" ]; then
  echo "formatted"
  exit 0
fi
if [ "$cmd" = "state" ]; then
  if [ "$1" = "list" ]; then
    echo "aws_instance.web"
    echo "aws_s3_bucket.data"
    exit 0
  fi
  if [ "$1" = "show" ]; then
    echo "# aws_instance.web:"
    echo "resource \"aws_instance\" \"web\" {"
    echo "  ami = \"ami-12345\""
    echo "}"
    exit 0
  fi
fi
if [ "$cmd" = "plan" ] || [ "$cmd" = "apply" ]; then
  echo "stdout $cmd"
  echo "stderr $cmd" 1>&2
fi
if [ "$cmd" = "init" ]; then
  echo "stdout init"
fi
echo "ARGS:$cmd $*"
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}
