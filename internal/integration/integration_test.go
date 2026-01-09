package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ushiradineth/tftui/internal/terraform"
)

func TestTerraformWorkflowIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	workdir := t.TempDir()
	if err := copyDummyConfig(workdir); err != nil {
		t.Fatalf("copy dummy config: %v", err)
	}

	tfPath := writeFakeTerraform(t, t.TempDir())
	exec, err := terraform.NewExecutor(workdir, terraform.WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	ctx := context.Background()
	initResult, err := exec.Init(ctx)
	if err != nil {
		t.Fatalf("init error: %v", err)
	}
	<-initResult.Done()
	if initResult.ExitCode != 0 {
		t.Fatalf("init failed: %v", initResult.Error)
	}

	planResult, _, err := exec.Plan(ctx, terraform.PlanOptions{})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	<-planResult.Done()
	if planResult.ExitCode != 0 {
		t.Fatalf("plan failed: %v", planResult.Error)
	}
	if !strings.Contains(planResult.Output, "Terraform will perform") {
		t.Fatalf("expected plan output")
	}

	applyResult, _, err := exec.Apply(ctx, terraform.ApplyOptions{AutoApprove: true})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	<-applyResult.Done()
	if applyResult.ExitCode != 0 {
		t.Fatalf("apply failed: %v", applyResult.Error)
	}
	if !strings.Contains(applyResult.Output, "Apply complete") {
		t.Fatalf("expected apply output")
	}
}

func TestTerraformWorkflowNoChanges(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}
	workdir := t.TempDir()
	if err := copyDummyConfig(workdir); err != nil {
		t.Fatalf("copy dummy config: %v", err)
	}
	tfPath := writeFakeTerraform(t, t.TempDir())
	exec, err := terraform.NewExecutor(workdir, terraform.WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, _, err := exec.Plan(context.Background(), terraform.PlanOptions{Flags: []string{"nochanges"}})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	<-result.Done()
	if !strings.Contains(result.Output, "No changes.") {
		t.Fatalf("expected no changes output")
	}
}

func TestTerraformWorkflowFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}
	workdir := t.TempDir()
	if err := copyDummyConfig(workdir); err != nil {
		t.Fatalf("copy dummy config: %v", err)
	}
	tfPath := writeFakeTerraform(t, t.TempDir())
	exec, err := terraform.NewExecutor(workdir, terraform.WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, _, err := exec.Apply(context.Background(), terraform.ApplyOptions{Flags: []string{"fail"}})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	<-result.Done()
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit")
	}
	if result.Error == nil {
		t.Fatalf("expected error on failure")
	}
}

func TestTerraformWorkflowCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}
	workdir := t.TempDir()
	if err := copyDummyConfig(workdir); err != nil {
		t.Fatalf("copy dummy config: %v", err)
	}
	tfPath := writeFakeTerraform(t, t.TempDir())
	exec, err := terraform.NewExecutor(workdir, terraform.WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result, _, err := exec.Plan(ctx, terraform.PlanOptions{Flags: []string{"sleep"}})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-result.Done()
	if !errors.Is(result.Error, context.Canceled) {
		t.Fatalf("expected canceled, got %v", result.Error)
	}
}

func TestTerraformWorkflowLargeOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}
	workdir := t.TempDir()
	if err := copyDummyConfig(workdir); err != nil {
		t.Fatalf("copy dummy config: %v", err)
	}
	tfPath := writeFakeTerraform(t, t.TempDir())
	exec, err := terraform.NewExecutor(workdir, terraform.WithTerraformPath(tfPath))
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, output, err := exec.Plan(context.Background(), terraform.PlanOptions{Flags: []string{"large"}})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	lines := 0
	for range output {
		lines++
	}
	<-result.Done()
	if lines < 12000 {
		t.Fatalf("expected large output, got %d lines", lines)
	}
}

func copyDummyConfig(dir string) error {
	src := filepath.Join("..", "..", "testdata", "terraform", "dummy", "main.tf")
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.tf"), data, 0o644)
}

func writeFakeTerraform(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "terraform")
	script := `#!/bin/sh
cmd="$1"
shift
if [ "$cmd" = "init" ]; then
  echo "Initializing..."
  exit 0
fi
if [ "$cmd" = "plan" ]; then
  if [ "$1" = "sleep" ]; then
    sleep 1
  fi
  if [ "$1" = "nochanges" ]; then
    echo "No changes. Your infrastructure matches the configuration."
    exit 0
  fi
  if [ "$1" = "large" ]; then
    i=0
    while [ $i -lt 12050 ]; do
      echo "line $i"
      i=$((i+1))
    done
    exit 0
  fi
  echo "Terraform will perform the following actions:"
  echo ""
  echo "  # null_resource.example will be created"
  echo "  + resource \"null_resource\" \"example\" {"
  echo "      + id = (known after apply)"
  echo "    }"
  echo ""
  echo "Plan: 1 to add, 0 to change, 0 to destroy."
  exit 0
fi
if [ "$cmd" = "apply" ]; then
  if [ "$1" = "fail" ]; then
    echo "Error: invalid configuration" 1>&2
    exit 1
  fi
  echo "Apply complete! Resources: 1 added, 0 changed, 0 destroyed."
  exit 0
fi
echo "unknown command" 1>&2
exit 1
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake terraform: %v", err)
	}
	return path
}
