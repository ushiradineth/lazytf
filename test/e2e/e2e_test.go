//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
	tfparser "github.com/ushiradineth/lazytf/internal/terraform/parser"
)

func TestE2EPlanApplyHistory(t *testing.T) {
	workdir := copyFixture(t, "basic")
	executor := newTerraformExecutor(t, workdir)

	ctx := context.Background()
	initResult, err := executor.Init(ctx, terraform.InitOptions{})
	if err != nil {
		t.Fatalf("init error: %v", err)
	}
	<-initResult.Done()
	if initResult.ExitCode != 0 {
		if shouldSkipProviderDownload(initResult.Output) {
			t.Skip("terraform init failed while downloading providers; network access required")
		}
		t.Fatalf("init failed: %s", initResult.Output)
	}

	plan, planOutput, planResult := planWithText(t, executor, ctx)
	if plan == nil || len(plan.Resources) == 0 {
		t.Fatal("expected planned resources from text output")
	}

	applyResult, applyOutput := applyAutoApprove(t, executor, ctx)
	if !strings.Contains(applyResult.Output, "Apply complete") {
		t.Fatalf("unexpected apply output: %s", applyResult.Output)
	}

	historyPath := filepath.Join(t.TempDir(), "history.db")
	store, err := history.Open(historyPath)
	if err != nil {
		t.Fatalf("open history store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	logger := history.NewLogger(store, history.LevelStandard)

	recordOperation(t, logger, "plan", workdir, planResult, planOutput)
	recordOperation(t, logger, "apply", workdir, applyResult, applyOutput)

	entries, err := store.QueryOperations(history.OperationFilter{Limit: 10})
	if err != nil {
		t.Fatalf("query operations: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 history entries, got %d", len(entries))
	}
	foundApply := false
	for _, entry := range entries {
		if entry.Action == "apply" {
			foundApply = true
			if entry.Environment != filepath.Base(workdir) {
				t.Fatalf("expected environment %q, got %q", filepath.Base(workdir), entry.Environment)
			}
			break
		}
	}
	if !foundApply {
		t.Fatal("expected apply operation in history")
	}
}

func TestE2EPlanModuleFixture(t *testing.T) {
	workdir := copyFixture(t, "module")
	executor := newTerraformExecutor(t, workdir)

	ctx := context.Background()
	initResult, err := executor.Init(ctx, terraform.InitOptions{})
	if err != nil {
		t.Fatalf("init error: %v", err)
	}
	<-initResult.Done()
	if initResult.ExitCode != 0 {
		if shouldSkipProviderDownload(initResult.Output) {
			t.Skip("terraform init failed while downloading providers; network access required")
		}
		t.Fatalf("init failed: %s", initResult.Output)
	}

	plan, _, _ := planWithText(t, executor, ctx)
	if plan == nil {
		t.Fatal("expected plan from text output")
	}
	found := false
	for _, resource := range plan.Resources {
		if resource.Address == "module.example.null_resource.child" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected module resource address, got %d resources", len(plan.Resources))
	}
}

func terraformPathOrSkip(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform binary not found in PATH")
	}
	return path
}

func shouldSkipProviderDownload(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "registry.terraform.io") ||
		strings.Contains(lower, "failed to query available provider packages") ||
		strings.Contains(lower, "could not connect") ||
		strings.Contains(lower, "no such host")
}

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "terraform", "fixtures", name)
	dst := t.TempDir()
	copyDir(t, src, dst)
	return dst
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		return os.WriteFile(target, data, mode)
	})
	if err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
}

func newTerraformExecutor(t *testing.T, workdir string) *terraform.Executor {
	t.Helper()
	tfPath := terraformPathOrSkip(t)

	dataDir := filepath.Join(workdir, ".terraform-data")
	cacheDir := filepath.Join(workdir, ".terraform-cache")
	for _, dir := range []string{dataDir, cacheDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create terraform dir: %v", err)
		}
	}

	executor, err := terraform.NewExecutor(
		workdir,
		terraform.WithTerraformPath(tfPath),
		terraform.WithEnv([]string{
			"TF_IN_AUTOMATION=1",
			"TF_INPUT=0",
			"TF_DATA_DIR=" + dataDir,
			"TF_PLUGIN_CACHE_DIR=" + cacheDir,
		}),
	)
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}
	return executor
}

func planWithText(t *testing.T, executor *terraform.Executor, ctx context.Context) (*terraform.Plan, string, *terraform.ExecutionResult) {
	t.Helper()
	result, output, err := executor.Plan(ctx, terraform.PlanOptions{})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}

	outputText := collectOutput(output)
	<-result.Done()

	if result.ExitCode != 0 {
		t.Fatalf("plan failed: %s", result.Output)
	}
	if result.Error != nil {
		t.Fatalf("plan error: %v", result.Error)
	}

	parser := tfparser.NewTextParser()
	plan, parseErr := parser.Parse(strings.NewReader(outputText))
	if parseErr != nil {
		t.Fatalf("parse text plan: %v", parseErr)
	}

	return plan, outputText, result
}

func applyAutoApprove(t *testing.T, executor *terraform.Executor, ctx context.Context) (*terraform.ExecutionResult, string) {
	t.Helper()
	result, output, err := executor.Apply(ctx, terraform.ApplyOptions{AutoApprove: true})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	outputText := collectOutput(output)
	<-result.Done()
	if result.ExitCode != 0 {
		t.Fatalf("apply failed: %s", result.Output)
	}
	return result, outputText
}

func collectOutput(output <-chan string) string {
	var buf strings.Builder
	for line := range output {
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	return strings.TrimRight(buf.String(), "\n")
}

func recordOperation(t *testing.T, logger *history.Logger, action, workdir string, result *terraform.ExecutionResult, output string) {
	t.Helper()
	if result == nil {
		t.Fatal("missing execution result")
	}
	status := history.StatusFailed
	if result.ExitCode == 0 && result.Error == nil {
		status = history.StatusSuccess
	}
	startedAt := time.Now().Add(-result.Duration)
	entry := history.OperationEntry{
		StartedAt:   startedAt,
		FinishedAt:  time.Now(),
		Duration:    result.Duration,
		Action:      action,
		Command:     fmt.Sprintf("terraform %s", action),
		ExitCode:    result.ExitCode,
		Status:      status,
		Summary:     fmt.Sprintf("%s complete", action),
		User:        os.Getenv("USER"),
		Environment: filepath.Base(workdir),
		Output:      output,
	}
	if err := logger.RecordOperation(entry); err != nil {
		t.Fatalf("record %s operation: %v", action, err)
	}
}
