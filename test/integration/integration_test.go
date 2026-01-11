//go:build integration

package integration_test

import (
	"context"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/terraform"
	tfparser "github.com/ushiradineth/lazytf/internal/terraform/parser"
)

func TestIntegrationPlanApplyWorkflow(t *testing.T) {
	workdir := copyFixture(t, "basic")
	executor := newTerraformExecutor(t, workdir)

	ctx := context.Background()
	initResult, err := executor.Init(ctx)
	if err != nil {
		t.Fatalf("init error: %v", err)
	}
	<-initResult.Done()
	if initResult.ExitCode != 0 {
		t.Fatalf("init failed: %s", initResult.Output)
	}

	planResult, _, err := executor.Plan(ctx, terraform.PlanOptions{})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	<-planResult.Done()
	if planResult.ExitCode != 0 {
		t.Fatalf("plan failed: %s", planResult.Output)
	}
	if !strings.Contains(planResult.Output, "Plan:") && !strings.Contains(planResult.Output, "Terraform will perform") {
		t.Fatalf("unexpected plan output: %s", planResult.Output)
	}

	applyResult, _, err := executor.Apply(ctx, terraform.ApplyOptions{AutoApprove: true})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	<-applyResult.Done()
	if applyResult.ExitCode != 0 {
		t.Fatalf("apply failed: %s", applyResult.Output)
	}
	if !strings.Contains(applyResult.Output, "Apply complete") {
		t.Fatalf("unexpected apply output: %s", applyResult.Output)
	}

	noChangeResult, _, err := executor.Plan(ctx, terraform.PlanOptions{})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	<-noChangeResult.Done()
	if noChangeResult.ExitCode != 0 {
		t.Fatalf("plan failed: %s", noChangeResult.Output)
	}
	if !strings.Contains(noChangeResult.Output, "No changes.") {
		t.Fatalf("expected no-changes output, got: %s", noChangeResult.Output)
	}
}

func TestIntegrationJSONPlanParsingModuleFixture(t *testing.T) {
	workdir := copyFixture(t, "module")
	executor := newTerraformExecutor(t, workdir)

	supports, err := executor.SupportsJSON()
	if err != nil {
		t.Skipf("terraform version unavailable: %v", err)
	}
	if !supports {
		t.Skip("terraform does not support -json streaming output")
	}

	ctx := context.Background()
	initResult, err := executor.Init(ctx)
	if err != nil {
		t.Fatalf("init error: %v", err)
	}
	<-initResult.Done()
	if initResult.ExitCode != 0 {
		t.Fatalf("init failed: %s", initResult.Output)
	}

	plan := planWithJSON(t, executor, ctx)
	if plan == nil {
		t.Fatal("expected plan from JSON stream")
	}
	if len(plan.Resources) == 0 {
		t.Fatal("expected planned resources")
	}

	found := false
	for _, resource := range plan.Resources {
		if resource.Address == "module.example.null_resource.child" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected module resource address in plan, got %d resources", len(plan.Resources))
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

func planWithJSON(t *testing.T, executor *terraform.Executor, ctx context.Context) *terraform.Plan {
	t.Helper()
	result, output, err := executor.Plan(ctx, terraform.PlanOptions{UseJSON: true})
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}

	reader, writer := io.Pipe()
	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		for line := range output {
			if _, writeErr := io.WriteString(writer, line+"\n"); writeErr != nil {
				_ = writer.CloseWithError(writeErr)
				return
			}
		}
		_ = writer.Close()
	}()

	parser := tfparser.NewStreamParser()
	if err := parser.Parse(reader, nil); err != nil {
		t.Fatalf("parse JSON plan: %v", err)
	}
	<-result.Done()
	<-writeDone

	if result.ExitCode != 0 {
		t.Fatalf("plan failed: %s", result.Output)
	}
	if result.Error != nil {
		t.Fatalf("plan error: %v", result.Error)
	}

	return parser.GetAccumulatedPlan()
}
