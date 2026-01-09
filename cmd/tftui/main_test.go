package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ushiradineth/tftui/internal/terraform"
)

func TestRun_NoPlanFile(t *testing.T) {
	oldPlanFile := planFile
	oldExecute := executeMode
	t.Cleanup(func() {
		planFile = oldPlanFile
		executeMode = oldExecute
	})
	planFile = ""
	executeMode = false

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected error for missing plan file")
	}
	if !strings.Contains(err.Error(), "no plan file specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ParseFileError(t *testing.T) {
	oldPlanFile := planFile
	oldExecute := executeMode
	t.Cleanup(func() {
		planFile = oldPlanFile
		executeMode = oldExecute
	})
	planFile = filepath.Join(t.TempDir(), "missing.json")
	executeMode = false

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected error for missing plan file")
	}
	if !strings.Contains(err.Error(), "failed to parse plan file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSplitFlags(t *testing.T) {
	if got := splitFlags(""); got != nil {
		t.Fatalf("expected nil for empty flags")
	}
	got := splitFlags("-var-file=dev.tfvars -lock=false")
	if len(got) != 2 || got[0] != "-var-file=dev.tfvars" || got[1] != "-lock=false" {
		t.Fatalf("unexpected flags: %#v", got)
	}

	quoted := splitFlags("-var 'a=b' -lock=false")
	if len(quoted) != 3 || quoted[1] != "a=b" {
		t.Fatalf("unexpected quoted flags: %#v", quoted)
	}
}

func TestRun_ExecuteModeNoPlanFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	oldPlanFile := planFile
	oldExecute := executeMode
	oldAuto := autoPlan
	oldFlags := tfFlags
	oldWorkDir := workDir
	oldRunner := programRunner
	t.Cleanup(func() {
		planFile = oldPlanFile
		executeMode = oldExecute
		autoPlan = oldAuto
		tfFlags = oldFlags
		workDir = oldWorkDir
		programRunner = oldRunner
	})

	workDir = t.TempDir()
	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(tfPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	t.Setenv("PATH", tfDir)

	executeMode = true
	autoPlan = false
	tfFlags = ""

	programRunner = func(_ tea.Model) error {
		return nil
	}

	if err := run(&cobra.Command{}, nil); err != nil {
		t.Fatalf("expected execute mode to run without plan file, got %v", err)
	}
}

func TestRun_ExecuteModeTerraformMissing(t *testing.T) {
	oldExecute := executeMode
	oldWorkDir := workDir
	oldRunner := programRunner
	t.Cleanup(func() {
		executeMode = oldExecute
		workDir = oldWorkDir
		programRunner = oldRunner
	})

	executeMode = true
	workDir = t.TempDir()
	programRunner = func(_ tea.Model) error {
		t.Fatalf("program runner should not be called when executor init fails")
		return nil
	}
	t.Setenv("PATH", "")

	err := run(&cobra.Command{}, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to initialize terraform") {
		t.Fatalf("expected executor init error, got %v", err)
	}
}

func TestRun_ExecuteModeWorkdirResolution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	oldExecute := executeMode
	oldWorkDir := workDir
	oldRunner := programRunner
	oldFactory := executorFactory
	t.Cleanup(func() {
		executeMode = oldExecute
		workDir = oldWorkDir
		programRunner = oldRunner
		executorFactory = oldFactory
	})

	cwd := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldCwd)
	})
	relDir := "relative"
	if err := os.MkdirAll(filepath.Join(cwd, relDir), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(tfPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}

	var captured string
	executorFactory = func(dir string, _ ...terraform.ExecutorOption) (*terraform.Executor, error) {
		exec, err := terraform.NewExecutor(dir, terraform.WithTerraformPath(tfPath))
		if err != nil {
			return nil, err
		}
		captured = exec.WorkDir()
		return exec, nil
	}
	programRunner = func(_ tea.Model) error {
		return nil
	}

	executeMode = true
	workDir = relDir
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := run(&cobra.Command{}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	absDir := filepath.Join(cwd, relDir)
	expected, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	actual, err := filepath.EvalSymlinks(captured)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	if actual != expected {
		t.Fatalf("expected workdir %q, got %q", expected, actual)
	}
}
