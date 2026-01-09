package terraform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
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
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := NewExecutor(file); err == nil {
		t.Fatalf("expected error for non-directory workdir")
	}
}

func TestNewExecutorResolvesPathFromEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraform(t, dir)
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
	if runtime.GOOS == "windows" {
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
	if runtime.GOOS == "windows" {
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
	result, _, err := exec.Plan(ctx, PlanOptions{Flags: []string{"-baz"}})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	<-result.Done()
	if !strings.Contains(result.Stdout, "ARGS:plan -foo -bar=1 -baz") {
		t.Fatalf("unexpected plan args: %q", result.Stdout)
	}

	applyResult, _, err := exec.Apply(ctx, ApplyOptions{Flags: []string{"-baz"}, AutoApprove: true})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	<-applyResult.Done()
	if !strings.Contains(applyResult.Stdout, "ARGS:apply -foo -bar=1 -baz -auto-approve") {
		t.Fatalf("expected auto-approve flag, got %q", applyResult.Stdout)
	}
}

func TestExecutorVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
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
		t.Fatalf("expected non-empty version")
	}
}

func TestExecutorStreamingStdoutStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
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
	if !containsLine(lines, "stdout plan") {
		t.Fatalf("expected stdout line, got %#v", lines)
	}
	if !containsLine(lines, "stderr plan") {
		t.Fatalf("expected stderr line, got %#v", lines)
	}
}

func TestExecutorExitCodeMapping(t *testing.T) {
	if runtime.GOOS == "windows" {
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
	if runtime.GOOS == "windows" {
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
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}
	dir := t.TempDir()
	tfPath := writeFakeTerraform(t, dir)
	t.Setenv("FOO", "base")
	exec, err := NewExecutor(dir, WithTerraformPath(tfPath), WithEnv([]string{"FOO=exec", "BAR=exec"}))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	result, _, err := exec.Plan(context.Background(), PlanOptions{Flags: []string{"envtest"}, Env: []string{"FOO=opts"}})
	if err != nil {
		t.Fatalf("plan start error: %v", err)
	}
	<-result.Done()
	if !strings.Contains(result.Stdout, "ENV:FOO=opts BAR=exec") {
		t.Fatalf("unexpected env output: %q", result.Stdout)
	}
}

func TestExecutorSymlinkedTerraformPath(t *testing.T) {
	if runtime.GOOS == "windows" {
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
	if runtime.GOOS == "windows" {
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
if [ "$cmd" = "plan" ] || [ "$cmd" = "apply" ]; then
  echo "stdout $cmd"
  echo "stderr $cmd" 1>&2
fi
if [ "$1" = "sleep" ]; then
  sleep 1
fi
if [ "$1" = "envtest" ]; then
  echo "ENV:FOO=$FOO BAR=$BAR"
fi
if [ "$1" = "exit7" ]; then
  exit 7
fi
echo "ARGS:$cmd $*"
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
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
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
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
