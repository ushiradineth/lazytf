package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

var testMu sync.Mutex

func TestRun_EmptyPlanFilePath(t *testing.T) {
	oldPlanFile := planFile
	oldReadOnly := readOnly
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnly = oldReadOnly
	})
	useTempConfig(t)
	planFile = ""

	// Positional args are no longer supported.
	err := run(&cobra.Command{}, []string{""})
	if err == nil {
		t.Fatalf("expected positional-argument error")
	}
	if !strings.Contains(err.Error(), "positional arguments are not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ReadOnlyRequiresPlan(t *testing.T) {
	oldPlanFile := planFile
	oldReadOnly := readOnly
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnly = oldReadOnly
	})
	useTempConfig(t)
	planFile = ""
	readOnly = true

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected readonly/plan validation error")
	}
	if !strings.Contains(err.Error(), "--readonly requires --plan") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_PlanStdinRequiresPipe(t *testing.T) {
	oldPlanFile := planFile
	oldReadOnly := readOnly
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnly = oldReadOnly
	})
	useTempConfig(t)

	planFile = "-"
	readOnly = true

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected stdin pipe requirement error")
	}
	if !strings.Contains(err.Error(), "requires piped input") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_PlanStdinEmptyInput(t *testing.T) {
	oldPlanFile := planFile
	oldReadOnly := readOnly
	oldStdin := os.Stdin
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnly = oldReadOnly
		os.Stdin = oldStdin
	})
	useTempConfig(t)

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	_ = writePipe.Close()
	os.Stdin = readPipe

	planFile = "-"
	readOnly = true

	err = run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected empty stdin error")
	}
	if !strings.Contains(err.Error(), "stdin plan input is empty") {
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

	doubleQuoted := splitFlags(`-var "a=b c" -lock=false`)
	if len(doubleQuoted) != 3 || doubleQuoted[1] != "a=b c" {
		t.Fatalf("unexpected double-quoted flags: %#v", doubleQuoted)
	}
}

func TestRun_ExecuteModeNoPlanFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	oldPlanFile := planFile
	oldFlags := tfFlags
	oldWorkDir := workDir
	oldRunner := programRunner
	oldExecRunner := executionModeRunner
	t.Cleanup(func() {
		planFile = oldPlanFile
		tfFlags = oldFlags
		workDir = oldWorkDir
		programRunner = oldRunner
		executionModeRunner = oldExecRunner
	})
	useTempConfig(t)

	workDir = t.TempDir()
	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(tfPath, []byte(script), 0o600); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	if err := os.Chmod(tfPath, 0o700); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	t.Setenv("PATH", tfDir)

	planFile = "" // Ensure execution mode (no plan file)
	tfFlags = ""

	executionModeRunner = func(_ tea.Model, _ *history.Store) error {
		return nil
	}

	if err := run(&cobra.Command{}, nil); err != nil {
		t.Fatalf("expected execute mode to run without plan file, got %v", err)
	}
}

func TestRun_ExecuteModeTerraformMissing(t *testing.T) {
	oldPlanFile := planFile
	oldWorkDir := workDir
	oldRunner := programRunner
	t.Cleanup(func() {
		planFile = oldPlanFile
		workDir = oldWorkDir
		programRunner = oldRunner
	})
	useTempConfig(t)

	planFile = "" // Ensure execution mode
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

	oldPlanFile := planFile
	oldWorkDir := workDir
	oldRunner := programRunner
	oldExecRunner := executionModeRunner
	oldFactory := executorFactory
	t.Cleanup(func() {
		planFile = oldPlanFile
		workDir = oldWorkDir
		programRunner = oldRunner
		executionModeRunner = oldExecRunner
		executorFactory = oldFactory
	})
	useTempConfig(t)

	cwd := t.TempDir()
	relDir := "relative"
	if err := os.MkdirAll(filepath.Join(cwd, relDir), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(tfPath, []byte(script), 0o600); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	if err := os.Chmod(tfPath, 0o700); err != nil {
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
	executionModeRunner = func(_ tea.Model, _ *history.Store) error {
		return nil
	}

	planFile = "" // Ensure execution mode
	workDir = relDir
	t.Chdir(cwd)

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

func useTempConfig(t *testing.T) {
	t.Helper()
	testMu.Lock()
	oldConfigPath := configPath
	oldThemeName := themeName
	oldNoHistory := noHistory
	oldReadOnly := readOnly
	t.Cleanup(func() {
		configPath = oldConfigPath
		themeName = oldThemeName
		noHistory = oldNoHistory
		readOnly = oldReadOnly
		testMu.Unlock()
	})
	configPath = filepath.Join(t.TempDir(), "config.yaml")
	themeName = ""
	noHistory = true
	readOnly = false
}

func TestResolveFolderSelectionEmpty(t *testing.T) {
	baseDir := t.TempDir()
	resolved, err := resolveFolderSelection(baseDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != baseDir {
		t.Fatalf("expected base dir, got %q", resolved)
	}
}

func TestResolveFolderSelectionTraversal(t *testing.T) {
	_, err := resolveFolderSelection(t.TempDir(), "../evil")
	if err == nil || !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func TestResolveFolderSelectionAbsolute(t *testing.T) {
	oldNewFolder := newFolderManager
	t.Cleanup(func() {
		newFolderManager = oldNewFolder
	})

	manager := &fakeFolderManager{}
	newFolderManager = func(_ string) (folderManager, error) {
		return manager, nil
	}

	baseDir := t.TempDir()
	absFolder := t.TempDir()
	resolved, err := resolveFolderSelection(baseDir, absFolder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != absFolder {
		t.Fatalf("expected abs folder, got %q", resolved)
	}
	if manager.validated != absFolder {
		t.Fatalf("expected validate to be called with abs folder")
	}
}

func TestDetectCurrentWorkspaceForPlanInput(t *testing.T) {
	oldManager := newWorkspaceManager
	t.Cleanup(func() {
		newWorkspaceManager = oldManager
	})

	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return &fakeWorkspaceManager{current: "prod"}, nil
	}
	if got := detectCurrentWorkspaceForPlanInput(t.TempDir()); got != "prod" {
		t.Fatalf("expected workspace prod, got %q", got)
	}
}

func TestDetectCurrentWorkspaceForPlanInputError(t *testing.T) {
	oldManager := newWorkspaceManager
	t.Cleanup(func() {
		newWorkspaceManager = oldManager
	})

	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return &fakeWorkspaceManager{currentErr: errors.New("boom")}, nil
	}
	if got := detectCurrentWorkspaceForPlanInput(t.TempDir()); got != "" {
		t.Fatalf("expected empty workspace on error, got %q", got)
	}
}

func TestReadPlanWorkdirHintRelative(t *testing.T) {
	plansDir := t.TempDir()
	planPath := filepath.Join(plansDir, "sample.tfplan")
	if err := os.WriteFile(planPath, []byte("plan"), 0o600); err != nil {
		t.Fatalf("write plan file: %v", err)
	}
	if err := os.WriteFile(planPath+".workdir", []byte("../terraform/dummy\n"), 0o600); err != nil {
		t.Fatalf("write workdir hint: %v", err)
	}

	got, err := readPlanWorkdirHint(planPath)
	if err != nil {
		t.Fatalf("readPlanWorkdirHint error: %v", err)
	}
	expected := filepath.Clean(filepath.Join(plansDir, "..", "terraform", "dummy"))
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestReadPlanWorkdirHintMissing(t *testing.T) {
	_, err := readPlanWorkdirHint(filepath.Join(t.TempDir(), "missing.tfplan"))
	if err == nil {
		t.Fatal("expected error for missing workdir hint")
	}
}

func TestReadPlanWorkdirHintAbsoluteRejected(t *testing.T) {
	plansDir := t.TempDir()
	planPath := filepath.Join(plansDir, "sample.tfplan")
	if err := os.WriteFile(planPath, []byte("plan"), 0o600); err != nil {
		t.Fatalf("write plan file: %v", err)
	}
	if err := os.WriteFile(planPath+".workdir", []byte("/tmp/anywhere\n"), 0o600); err != nil {
		t.Fatalf("write workdir hint: %v", err)
	}

	_, err := readPlanWorkdirHint(planPath)
	if err == nil || !strings.Contains(err.Error(), "absolute plan workdir hints are not allowed") {
		t.Fatalf("expected absolute hint rejection, got %v", err)
	}
}

func TestResolveFolderSelectionRelative(t *testing.T) {
	oldNewFolder := newFolderManager
	t.Cleanup(func() {
		newFolderManager = oldNewFolder
	})

	manager := &fakeFolderManager{}
	newFolderManager = func(_ string) (folderManager, error) {
		return manager, nil
	}

	baseDir := t.TempDir()
	folder := consts.EnvDevPath
	resolved, err := resolveFolderSelection(baseDir, folder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(baseDir, folder)
	if resolved != expected {
		t.Fatalf("expected resolved path %q, got %q", expected, resolved)
	}
	if manager.validated != folder {
		t.Fatalf("expected validate to be called with relative folder")
	}
}

func TestRun_WorkspaceAndFolderConflict(t *testing.T) {
	oldPlanFile := planFile
	oldWorkspace := workspaceName
	oldFolder := folderPath
	t.Cleanup(func() {
		planFile = oldPlanFile
		workspaceName = oldWorkspace
		folderPath = oldFolder
	})
	useTempConfig(t)

	planFile = "" // Ensure execution mode
	workspaceName = "dev"
	folderPath = consts.EnvDevPath

	err := run(&cobra.Command{}, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot use --workspace and --folder together") {
		t.Fatalf("expected workspace/folder conflict error, got %v", err)
	}
}

func TestRun_WorkspaceSwitchError(t *testing.T) {
	oldPlanFile := planFile
	oldWorkspace := workspaceName
	oldWorkDir := workDir
	oldRunner := programRunner
	oldNewWorkspace := newWorkspaceManager
	t.Cleanup(func() {
		planFile = oldPlanFile
		workspaceName = oldWorkspace
		workDir = oldWorkDir
		programRunner = oldRunner
		newWorkspaceManager = oldNewWorkspace
	})
	useTempConfig(t)

	planFile = "" // Ensure execution mode
	workspaceName = "dev"
	workDir = t.TempDir()
	programRunner = func(_ tea.Model) error {
		t.Fatalf("program runner should not be called on workspace error")
		return nil
	}
	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return &fakeWorkspaceManager{switchErr: errors.New("switch failed")}, nil
	}

	err := run(&cobra.Command{}, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to select workspace") {
		t.Fatalf("expected workspace switch error, got %v", err)
	}
}

func TestRun_FolderSelectionError(t *testing.T) {
	oldPlanFile := planFile
	oldFolder := folderPath
	oldWorkDir := workDir
	oldRunner := programRunner
	oldNewFolder := newFolderManager
	t.Cleanup(func() {
		planFile = oldPlanFile
		folderPath = oldFolder
		workDir = oldWorkDir
		programRunner = oldRunner
		newFolderManager = oldNewFolder
	})
	useTempConfig(t)

	planFile = "" // Ensure execution mode
	folderPath = consts.EnvDevPath
	workDir = t.TempDir()
	programRunner = func(_ tea.Model) error {
		t.Fatalf("program runner should not be called on folder error")
		return nil
	}
	newFolderManager = func(_ string) (folderManager, error) {
		return &fakeFolderManager{validateErr: errors.New("invalid")}, nil
	}

	err := run(&cobra.Command{}, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid folder") {
		t.Fatalf("expected invalid folder error, got %v", err)
	}
}

func TestRun_InvalidTheme(t *testing.T) {
	oldPlanFile := planFile
	oldReadOnly := readOnly
	oldStdin := os.Stdin
	oldTheme := themeName
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnly = oldReadOnly
		os.Stdin = oldStdin
		themeName = oldTheme
	})
	useTempConfig(t)

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := writePipe.WriteString("Terraform will perform the following actions:\n\n  # aws_instance.web will be created\n  + resource \"aws_instance\" \"web\" {\n      + id = (known after apply)\n    }\n\nPlan: 1 to add, 0 to change, 0 to destroy.\n"); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	_ = writePipe.Close()
	os.Stdin = readPipe

	planFile = "-"
	readOnly = true
	themeName = "missing-theme"

	err = run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected theme error")
	}
}

func TestRun_PlanInputRunsExecutionModeByDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	oldPlanFile := planFile
	oldReadOnly := readOnly
	oldRunner := programRunner
	oldExecRunner := executionModeRunner
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnly = oldReadOnly
		programRunner = oldRunner
		executionModeRunner = oldExecRunner
	})
	useTempConfig(t)

	planFile = filepath.Join(t.TempDir(), "plan.tfplan")
	readOnly = false
	called := false
	executionModeRunner = func(_ tea.Model, _ *history.Store) error {
		called = true
		return nil
	}
	programRunner = func(_ tea.Model) error {
		t.Fatalf("program runner should not be called in execution mode")
		return nil
	}

	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\ncmd=\"$1\"\nshift\nif [ \"$cmd\" = \"show\" ]; then\necho \"No changes. Infrastructure is up-to-date.\"\nexit 0\nfi\nexit 0\n"
	if err := os.WriteFile(tfPath, []byte(script), 0o600); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	if err := os.Chmod(tfPath, 0o700); err != nil {
		t.Fatalf("chmod terraform script: %v", err)
	}
	t.Setenv("PATH", tfDir)

	err := run(&cobra.Command{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected execution mode runner to be called")
	}
}

func TestRun_PlanStdinRunsExecutionModeByDefault(t *testing.T) {
	oldPlanFile := planFile
	oldReadOnly := readOnly
	oldStdin := os.Stdin
	oldRunner := programRunner
	oldExecRunner := executionModeRunner
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnly = oldReadOnly
		os.Stdin = oldStdin
		programRunner = oldRunner
		executionModeRunner = oldExecRunner
	})
	useTempConfig(t)

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := writePipe.WriteString("Terraform will perform the following actions:\n\n  # aws_instance.web will be created\n  + resource \"aws_instance\" \"web\" {\n      + id = (known after apply)\n    }\n\nPlan: 1 to add, 0 to change, 0 to destroy.\n"); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	_ = writePipe.Close()
	os.Stdin = readPipe

	called := false
	executionModeRunner = func(_ tea.Model, _ *history.Store) error {
		called = true
		return nil
	}
	programRunner = func(_ tea.Model) error {
		t.Fatalf("program runner should not be called in execution mode")
		return nil
	}

	planFile = "-"
	readOnly = false

	err = run(&cobra.Command{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected execution mode runner to be called")
	}
}

func TestRunProgramSuccess(t *testing.T) {
	oldNewProgram := newProgram
	t.Cleanup(func() {
		newProgram = oldNewProgram
	})

	newProgram = func(model tea.Model, _ ...tea.ProgramOption) teaProgram {
		return fakeTeaProgram{model: model}
	}

	if err := runProgram(&fakeModel{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunProgramError(t *testing.T) {
	oldNewProgram := newProgram
	t.Cleanup(func() {
		newProgram = oldNewProgram
	})

	newProgram = func(model tea.Model, _ ...tea.ProgramOption) teaProgram {
		return fakeTeaProgram{model: model, err: errors.New("boom")}
	}

	if err := runProgram(&fakeModel{}); err == nil {
		t.Fatalf("expected run program error")
	}
}

func TestRunMainSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	oldArgs := os.Args
	oldPlanFile := planFile
	oldReadOnly := readOnly
	oldRunner := programRunner
	oldExecRunner := executionModeRunner
	t.Cleanup(func() {
		os.Args = oldArgs
		planFile = oldPlanFile
		readOnly = oldReadOnly
		programRunner = oldRunner
		executionModeRunner = oldExecRunner
	})
	useTempConfig(t)

	planPath := filepath.Join(t.TempDir(), "plan.tfplan")
	os.Args = []string{"lazytf", "--plan", planPath, "--readonly"}
	planFile = ""
	readOnly = false
	called := false
	programRunner = func(_ tea.Model) error {
		called = true
		return nil
	}
	executionModeRunner = func(_ tea.Model, _ *history.Store) error {
		t.Fatalf("execution mode runner should not be called in readonly mode")
		return nil
	}

	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\ncmd=\"$1\"\nshift\nif [ \"$cmd\" = \"show\" ]; then\necho \"No changes. Infrastructure is up-to-date.\"\nexit 0\nfi\nexit 0\n"
	if err := os.WriteFile(tfPath, []byte(script), 0o600); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	if err := os.Chmod(tfPath, 0o700); err != nil {
		t.Fatalf("chmod terraform script: %v", err)
	}
	t.Setenv("PATH", tfDir)

	if err := runMain(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected program runner to be called")
	}
}

func TestRunMainError(t *testing.T) {
	oldArgs := os.Args
	oldPlanFile := planFile
	oldReadOnly := readOnly
	oldRunner := programRunner
	t.Cleanup(func() {
		os.Args = oldArgs
		planFile = oldPlanFile
		readOnly = oldReadOnly
		programRunner = oldRunner
	})
	useTempConfig(t)

	// Pass a non-existent plan file to trigger an error
	os.Args = []string{"lazytf", "--plan", "/nonexistent/plan.tfplan"}
	planFile = ""
	readOnly = false
	programRunner = func(_ tea.Model) error {
		t.Fatalf("program runner should not be called on error")
		return nil
	}

	if err := runMain(); err == nil {
		t.Fatalf("expected runMain error")
	}
}

type fakeWorkspaceManager struct {
	switchErr  error
	current    string
	currentErr error
}

func (f *fakeWorkspaceManager) Current(ctx context.Context) (string, error) {
	_ = ctx
	if f.currentErr != nil {
		return "", f.currentErr
	}
	return f.current, nil
}

func (f *fakeWorkspaceManager) Switch(ctx context.Context, name string) error {
	_ = ctx
	_ = name
	return f.switchErr
}

type fakeFolderManager struct {
	validateErr error
	validated   string
}

func (f *fakeFolderManager) Validate(ctx context.Context, path string) error {
	_ = ctx
	f.validated = path
	return f.validateErr
}

type fakeTeaProgram struct {
	model tea.Model
	err   error
}

func (f fakeTeaProgram) Run() (tea.Model, error) {
	return f.model, f.err
}

func (fakeTeaProgram) Send(_ tea.Msg) {}

type fakeModel struct{}

func (*fakeModel) Init() tea.Cmd {
	return nil
}

func (f *fakeModel) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return f, nil
}

func (*fakeModel) View() string {
	return ""
}

func TestOpenHistoryDisabled(t *testing.T) {
	cfg := testConfig()
	cfg.History.Enabled = false

	store, logger, err := openHistory(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store != nil {
		t.Fatalf("expected nil store when disabled")
	}
	if logger != nil {
		t.Fatalf("expected nil logger when disabled")
	}
}

func TestOpenHistoryWithCustomPath(t *testing.T) {
	cfg := testConfig()
	cfg.History.Enabled = true
	cfg.History.Path = filepath.Join(t.TempDir(), "history.db")
	cfg.History.CompressionThreshold = 1024

	store, logger, err := openHistory(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatalf("expected non-nil store")
	}
	if logger == nil {
		t.Fatalf("expected non-nil logger")
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
}

func TestOpenHistoryWithDefaultPath(t *testing.T) {
	cfg := testConfig()
	cfg.History.Enabled = true
	cfg.History.Path = ""

	// Set a temp XDG data home to avoid polluting the user's actual data directory
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	store, logger, err := openHistory(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatalf("expected non-nil store")
	}
	if logger == nil {
		t.Fatalf("expected non-nil logger")
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
}

func TestOpenHistoryInvalidPath(t *testing.T) {
	cfg := testConfig()
	cfg.History.Enabled = true
	cfg.History.Path = "/nonexistent/path/that/cannot/be/created/history.db"

	_, _, err := openHistory(&cfg)
	if err == nil {
		t.Fatalf("expected error for invalid path")
	}
	if !strings.Contains(err.Error(), "history store") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShouldDisableHistoryForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "sqlite cgo stub error",
			err:  errors.New("Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub"),
			want: true,
		},
		{
			name: "wrapped sqlite cgo stub error",
			err:  fmt.Errorf("history store: %w", errors.New("go-sqlite3 requires cgo")),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("permission denied"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldDisableHistoryForError(tt.err); got != tt.want {
				t.Fatalf("expected %t, got %t", tt.want, got)
			}
		})
	}
}

func TestApplyPresetEmpty(t *testing.T) {
	oldPreset := presetName
	t.Cleanup(func() {
		presetName = oldPreset
	})

	presetName = ""
	cfg := testConfig()
	flags := []string{"-var", "a=b"}

	result, err := applyPreset(&cfg, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(flags) {
		t.Fatalf("expected flags unchanged, got %v", result)
	}
}

func TestApplyPresetNotFound(t *testing.T) {
	oldPreset := presetName
	t.Cleanup(func() {
		presetName = oldPreset
	})

	presetName = "missing-preset"
	cfg := testConfig()
	flags := []string{}

	_, err := applyPreset(&cfg, flags)
	if err == nil {
		t.Fatalf("expected error for missing preset")
	}
	if !strings.Contains(err.Error(), "preset not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyPresetWithAllOptions(t *testing.T) {
	oldPreset := presetName
	oldWorkDir := workDir
	oldTheme := themeName
	oldEnv := envName
	t.Cleanup(func() {
		presetName = oldPreset
		workDir = oldWorkDir
		themeName = oldTheme
		envName = oldEnv
	})

	presetName = "dev"
	themeName = ""
	envName = ""
	workDir = "original"

	cfg := testConfig()
	cfg.Presets = []config.EnvironmentPreset{
		{
			Name:        "dev",
			WorkDir:     "/custom/workdir",
			Flags:       []string{"-var-file=dev.tfvars"},
			Theme:       "dark",
			Environment: "development",
		},
	}
	flags := []string{"-lock=false"}

	result, err := applyPreset(&cfg, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workDir != "/custom/workdir" {
		t.Fatalf("expected workdir to be set to /custom/workdir, got %s", workDir)
	}
	if cfg.Theme.Name != "dark" {
		t.Fatalf("expected theme to be set to dark, got %s", cfg.Theme.Name)
	}
	if envName != "development" {
		t.Fatalf("expected envName to be set to development, got %s", envName)
	}
	if len(result) != 2 || result[1] != "-var-file=dev.tfvars" {
		t.Fatalf("expected flags to include preset flags, got %v", result)
	}
}

func TestApplyPresetThemeNotOverridden(t *testing.T) {
	oldPreset := presetName
	oldTheme := themeName
	t.Cleanup(func() {
		presetName = oldPreset
		themeName = oldTheme
	})

	presetName = "dev"
	themeName = "existing-theme" // Already set

	cfg := testConfig()
	cfg.Presets = []config.EnvironmentPreset{
		{
			Name:  "dev",
			Theme: "new-theme",
		},
	}

	_, err := applyPreset(&cfg, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Theme should not be changed because themeName is already set
	if cfg.Theme.Name != "" {
		t.Fatalf("theme should not be overridden when themeName is already set")
	}
}

func TestResolveSelectedEnvWorkspace(t *testing.T) {
	oldWorkspace := workspaceName
	oldFolder := folderPath
	oldEnv := envName
	t.Cleanup(func() {
		workspaceName = oldWorkspace
		folderPath = oldFolder
		envName = oldEnv
	})

	workspaceName = "prod"
	folderPath = ""
	envName = "dev"

	result := resolveSelectedEnv()
	if result != "prod" {
		t.Fatalf("expected workspace name 'prod', got %q", result)
	}
}

func TestResolveSelectedEnvFolder(t *testing.T) {
	oldWorkspace := workspaceName
	oldFolder := folderPath
	oldWorkDir := workDir
	oldEnv := envName
	t.Cleanup(func() {
		workspaceName = oldWorkspace
		folderPath = oldFolder
		workDir = oldWorkDir
		envName = oldEnv
	})

	workspaceName = ""
	folderPath = "environments/dev"
	workDir = "/project/environments/dev"
	envName = "other"

	result := resolveSelectedEnv()
	if result != "dev" {
		t.Fatalf("expected folder base name 'dev', got %q", result)
	}
}

func TestResolveSelectedEnvEnvName(t *testing.T) {
	oldWorkspace := workspaceName
	oldFolder := folderPath
	oldEnv := envName
	t.Cleanup(func() {
		workspaceName = oldWorkspace
		folderPath = oldFolder
		envName = oldEnv
	})

	workspaceName = ""
	folderPath = ""
	envName = "staging"

	result := resolveSelectedEnv()
	if result != "staging" {
		t.Fatalf("expected env name 'staging', got %q", result)
	}
}

func TestStripFlagEmpty(t *testing.T) {
	result := stripFlag(nil, "-json")
	if result != nil {
		t.Fatalf("expected nil for nil input")
	}

	result = stripFlag([]string{}, "-json")
	if len(result) != 0 {
		t.Fatalf("expected empty slice for empty input")
	}
}

func TestStripFlagWithTarget(t *testing.T) {
	flags := []string{"-var-file=dev.tfvars", "-json", "-lock=false"}
	result := stripFlag(flags, "-json")
	if len(result) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(result))
	}
	for _, f := range result {
		if f == "-json" {
			t.Fatalf("expected -json to be stripped")
		}
	}
}

func TestPrepareExecutionFlagsWithOverrides(t *testing.T) {
	oldPreset := presetName
	oldTfFlags := tfFlags
	t.Cleanup(func() {
		presetName = oldPreset
		tfFlags = oldTfFlags
	})

	presetName = ""
	tfFlags = "-lock=false"
	cfg := testConfig()
	cfg.Terraform.DefaultFlags = []string{"-var-file=default.tfvars"}
	overrideFlags := []string{"-var-file=override.tfvars"}

	result, err := prepareExecutionFlags(&cfg, overrideFlags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 flags, got %v", result)
	}
	if result[0] != "-var-file=default.tfvars" {
		t.Fatalf("expected default flag first, got %s", result[0])
	}
	if result[1] != "-lock=false" {
		t.Fatalf("expected tf-flags second, got %s", result[1])
	}
	if result[2] != "-var-file=override.tfvars" {
		t.Fatalf("expected override flag third, got %s", result[2])
	}
}

func TestPrepareExecutionFlagsPresetError(t *testing.T) {
	oldPreset := presetName
	t.Cleanup(func() {
		presetName = oldPreset
	})

	presetName = "nonexistent"
	cfg := testConfig()

	_, err := prepareExecutionFlags(&cfg, nil)
	if err == nil {
		t.Fatalf("expected error for missing preset")
	}
}

func TestResolveAppStylesSuccess(t *testing.T) {
	cfg := testConfig()
	cfg.Theme.Name = "default"

	styles, err := resolveAppStyles(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if styles == nil {
		t.Fatalf("expected non-nil styles")
	}
}

func TestResolveAppStylesInvalidTheme(t *testing.T) {
	cfg := testConfig()
	cfg.Theme.Name = "invalid-theme-name"

	_, err := resolveAppStyles(&cfg)
	if err == nil {
		t.Fatalf("expected error for invalid theme")
	}
}

func TestConfigureWorkDirAndWorkspaceInitError(t *testing.T) {
	oldWorkspace := workspaceName
	oldFolder := folderPath
	oldWorkDir := workDir
	oldNewWs := newWorkspaceManager
	t.Cleanup(func() {
		workspaceName = oldWorkspace
		folderPath = oldFolder
		workDir = oldWorkDir
		newWorkspaceManager = oldNewWs
	})

	workspaceName = "dev"
	folderPath = ""
	workDir = t.TempDir()
	newWorkspaceManager = func(_ string) (workspaceManager, error) {
		return nil, errors.New("init failed")
	}

	err := configureWorkDirAndWorkspace()
	if err == nil {
		t.Fatalf("expected error for workspace init failure")
	}
	if !strings.Contains(err.Error(), "failed to initialize workspace manager") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigureWorkDirAndWorkspaceNoWorkspace(t *testing.T) {
	oldWorkspace := workspaceName
	oldFolder := folderPath
	t.Cleanup(func() {
		workspaceName = oldWorkspace
		folderPath = oldFolder
	})

	workspaceName = ""
	folderPath = ""

	err := configureWorkDirAndWorkspace()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildExecutorWithOptions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	oldWorkDir := workDir
	oldFactory := executorFactory
	t.Cleanup(func() {
		workDir = oldWorkDir
		executorFactory = oldFactory
	})

	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\nexit 0\n"
	//nolint:gosec // test executable needs execute permission
	if err := os.WriteFile(tfPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	t.Setenv("PATH", tfDir)

	workDir = t.TempDir()
	cfg := testConfig()
	cfg.Terraform.Binary = tfPath
	cfg.Terraform.Timeout = 60

	exec, err := buildExecutor(&cfg, []string{"-lock=false"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec == nil {
		t.Fatalf("expected non-nil executor")
	}
}

func TestResolveFolderSelectionManagerError(t *testing.T) {
	oldNewFolder := newFolderManager
	t.Cleanup(func() {
		newFolderManager = oldNewFolder
	})

	newFolderManager = func(_ string) (folderManager, error) {
		return nil, errors.New("init failed")
	}

	_, err := resolveFolderSelection(t.TempDir(), "envs/dev")
	if err == nil {
		t.Fatalf("expected error for manager init failure")
	}
	if !strings.Contains(err.Error(), "failed to initialize folder manager") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunWithProjectOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	oldPlanFile := planFile
	oldWorkDir := workDir
	oldRunner := programRunner
	oldExecRunner := executionModeRunner
	oldTheme := themeName
	oldPreset := presetName
	t.Cleanup(func() {
		planFile = oldPlanFile
		workDir = oldWorkDir
		programRunner = oldRunner
		executionModeRunner = oldExecRunner
		themeName = oldTheme
		presetName = oldPreset
	})
	useTempConfig(t)

	tempDir := t.TempDir()
	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\nexit 0\n"
	//nolint:gosec // test executable needs execute permission
	if err := os.WriteFile(tfPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	t.Setenv("PATH", tfDir)

	// Create config with project override
	configContent := `
projects:
  - path: "` + tempDir + `"
    theme: "dark"
    preset: "dev"
    flags:
      - "-lock=false"
presets:
  - name: "dev"
    flags:
      - "-var-file=dev.tfvars"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	workDir = tempDir
	planFile = "" // Ensure execution mode
	themeName = ""
	presetName = ""
	executionModeRunner = func(_ tea.Model, _ *history.Store) error {
		return nil
	}

	if err := run(&cobra.Command{}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunProgramWithMouse(t *testing.T) {
	oldNewProgram := newProgram
	oldMouse := mouseEnabled
	t.Cleanup(func() {
		newProgram = oldNewProgram
		mouseEnabled = oldMouse
	})

	mouseEnabled = true
	var capturedOpts int
	newProgram = func(model tea.Model, opts ...tea.ProgramOption) teaProgram {
		capturedOpts = len(opts)
		return fakeTeaProgram{model: model}
	}

	if err := runProgram(&fakeModel{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With mouse enabled, we should have 2 options: WithAltScreen and WithMouseCellMotion
	if capturedOpts != 2 {
		t.Fatalf("expected 2 program options with mouse, got %d", capturedOpts)
	}
}

func TestRun_UsesConfigMouseWhenFlagNotSet(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	oldPlanFile := planFile
	oldWorkDir := workDir
	oldRunner := programRunner
	oldExecRunner := executionModeRunner
	oldTheme := themeName
	oldMouse := mouseEnabled
	t.Cleanup(func() {
		planFile = oldPlanFile
		workDir = oldWorkDir
		programRunner = oldRunner
		executionModeRunner = oldExecRunner
		themeName = oldTheme
		mouseEnabled = oldMouse
	})
	useTempConfig(t)

	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\nexit 0\n"
	//nolint:gosec // test executable needs execute permission
	if err := os.WriteFile(tfPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	t.Setenv("PATH", tfDir)

	configContent := "mouse: false\n"
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	planFile = ""
	workDir = "."
	themeName = ""
	mouseEnabled = true
	executionModeRunner = func(_ tea.Model, _ *history.Store) error {
		return nil
	}

	cmd := newRootCommand()
	if err := run(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mouseEnabled {
		t.Fatalf("expected mouse to be disabled from config")
	}
}

func testConfig() config.Config {
	return config.Config{
		Theme: config.ThemeConfig{
			Name: "",
		},
		Terraform: config.TerraformConfig{
			DefaultFlags: []string{},
		},
		History: config.HistoryConfig{
			Enabled: false,
		},
		Presets: []config.EnvironmentPreset{},
	}
}

func TestRunWithConfigWorkDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test not supported on windows")
	}

	oldPlanFile := planFile
	oldWorkDir := workDir
	oldRunner := programRunner
	oldExecRunner := executionModeRunner
	oldConfigPath := configPath
	t.Cleanup(func() {
		planFile = oldPlanFile
		workDir = oldWorkDir
		programRunner = oldRunner
		executionModeRunner = oldExecRunner
		configPath = oldConfigPath
	})

	testMu.Lock()
	defer testMu.Unlock()

	// Create temp dirs
	tempDir := t.TempDir()
	customWorkDir := filepath.Join(tempDir, "custom")
	if err := os.MkdirAll(customWorkDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\nexit 0\n"
	//nolint:gosec // test executable needs execute permission
	if err := os.WriteFile(tfPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	t.Setenv("PATH", tfDir)

	// Create config with working_dir
	configPath = filepath.Join(tempDir, "config.yaml")
	configContent := `
terraform:
  working_dir: "` + customWorkDir + `"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	planFile = ""
	workDir = "." // Will be overridden by config
	noHistory = true
	themeName = ""

	executionModeRunner = func(_ tea.Model, _ *history.Store) error {
		return nil
	}

	if err := run(&cobra.Command{}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunConfigLoadError(t *testing.T) {
	oldConfigPath := configPath
	t.Cleanup(func() {
		configPath = oldConfigPath
	})

	testMu.Lock()
	defer testMu.Unlock()

	// Create an invalid config file
	tempDir := t.TempDir()
	configPath = filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("invalid: [yaml: content"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExecutionModeStylesError(t *testing.T) {
	oldPlanFile := planFile
	oldTheme := themeName
	t.Cleanup(func() {
		planFile = oldPlanFile
		themeName = oldTheme
	})
	useTempConfig(t)

	planFile = ""
	themeName = "nonexistent-theme"

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatal("expected error for invalid theme in execution mode")
	}
}
