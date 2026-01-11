package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

var testMu sync.Mutex

func TestRun_NoPlanFile(t *testing.T) {
	oldPlanFile := planFile
	oldReadOnly := readOnlyMode
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnlyMode = oldReadOnly
	})
	useTempConfig(t)
	planFile = ""
	readOnlyMode = true

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
	oldReadOnly := readOnlyMode
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnlyMode = oldReadOnly
	})
	useTempConfig(t)
	planFile = filepath.Join(t.TempDir(), "missing.json")
	readOnlyMode = true

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
	oldReadOnly := readOnlyMode
	oldAuto := autoPlan
	oldFlags := tfFlags
	oldWorkDir := workDir
	oldRunner := programRunner
	t.Cleanup(func() {
		planFile = oldPlanFile
		readOnlyMode = oldReadOnly
		autoPlan = oldAuto
		tfFlags = oldFlags
		workDir = oldWorkDir
		programRunner = oldRunner
	})
	useTempConfig(t)

	workDir = t.TempDir()
	tfDir := t.TempDir()
	tfPath := filepath.Join(tfDir, "terraform")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(tfPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	t.Setenv("PATH", tfDir)

	readOnlyMode = false
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
	oldReadOnly := readOnlyMode
	oldWorkDir := workDir
	oldRunner := programRunner
	t.Cleanup(func() {
		readOnlyMode = oldReadOnly
		workDir = oldWorkDir
		programRunner = oldRunner
	})
	useTempConfig(t)

	readOnlyMode = false
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

	oldReadOnly := readOnlyMode
	oldWorkDir := workDir
	oldRunner := programRunner
	oldFactory := executorFactory
	t.Cleanup(func() {
		readOnlyMode = oldReadOnly
		workDir = oldWorkDir
		programRunner = oldRunner
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

	readOnlyMode = false
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
	t.Cleanup(func() {
		configPath = oldConfigPath
		themeName = oldThemeName
		noHistory = oldNoHistory
		testMu.Unlock()
	})
	configPath = filepath.Join(t.TempDir(), "config.yaml")
	themeName = ""
	noHistory = true
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
	folder := "envs/dev"
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
	oldReadOnly := readOnlyMode
	oldWorkspace := workspaceName
	oldFolder := folderPath
	t.Cleanup(func() {
		readOnlyMode = oldReadOnly
		workspaceName = oldWorkspace
		folderPath = oldFolder
	})
	useTempConfig(t)

	readOnlyMode = false
	workspaceName = "dev"
	folderPath = "envs/dev"

	err := run(&cobra.Command{}, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot use --workspace and --folder together") {
		t.Fatalf("expected workspace/folder conflict error, got %v", err)
	}
}

func TestRun_WorkspaceSwitchError(t *testing.T) {
	oldReadOnly := readOnlyMode
	oldWorkspace := workspaceName
	oldWorkDir := workDir
	oldRunner := programRunner
	oldNewWorkspace := newWorkspaceManager
	t.Cleanup(func() {
		readOnlyMode = oldReadOnly
		workspaceName = oldWorkspace
		workDir = oldWorkDir
		programRunner = oldRunner
		newWorkspaceManager = oldNewWorkspace
	})
	useTempConfig(t)

	readOnlyMode = false
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
	oldReadOnly := readOnlyMode
	oldFolder := folderPath
	oldWorkDir := workDir
	oldRunner := programRunner
	oldNewFolder := newFolderManager
	t.Cleanup(func() {
		readOnlyMode = oldReadOnly
		folderPath = oldFolder
		workDir = oldWorkDir
		programRunner = oldRunner
		newFolderManager = oldNewFolder
	})
	useTempConfig(t)

	readOnlyMode = false
	folderPath = "envs/dev"
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
	oldReadOnly := readOnlyMode
	oldPlanFile := planFile
	oldTheme := themeName
	t.Cleanup(func() {
		readOnlyMode = oldReadOnly
		planFile = oldPlanFile
		themeName = oldTheme
	})
	useTempConfig(t)

	readOnlyMode = true
	planFile = filepath.Join("..", "..", "testdata", "plans", "sample.json")
	themeName = "missing-theme"

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected theme error")
	}
}

func TestRun_PlanArgDisablesExecute(t *testing.T) {
	oldReadOnly := readOnlyMode
	oldPlanFile := planFile
	oldRunner := programRunner
	t.Cleanup(func() {
		readOnlyMode = oldReadOnly
		planFile = oldPlanFile
		programRunner = oldRunner
	})
	useTempConfig(t)

	readOnlyMode = false
	planFile = ""
	called := false
	programRunner = func(_ tea.Model) error {
		called = true
		return nil
	}

	err := run(&cobra.Command{}, []string{filepath.Join("..", "..", "testdata", "plans", "sample.json")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected program runner to be called")
	}
	if !readOnlyMode {
		t.Fatalf("expected read-only mode to be enabled when plan arg provided")
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
	oldArgs := os.Args
	oldPlanFile := planFile
	oldReadOnly := readOnlyMode
	oldRunner := programRunner
	t.Cleanup(func() {
		os.Args = oldArgs
		planFile = oldPlanFile
		readOnlyMode = oldReadOnly
		programRunner = oldRunner
	})
	useTempConfig(t)

	sample := filepath.Join("..", "..", "testdata", "plans", "sample.json")
	os.Args = []string{"lazytf", "--read-only", "--file", sample}
	planFile = ""
	readOnlyMode = true
	called := false
	programRunner = func(_ tea.Model) error {
		called = true
		return nil
	}

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
	oldReadOnly := readOnlyMode
	oldRunner := programRunner
	t.Cleanup(func() {
		os.Args = oldArgs
		planFile = oldPlanFile
		readOnlyMode = oldReadOnly
		programRunner = oldRunner
	})
	useTempConfig(t)

	os.Args = []string{"lazytf", "--read-only"}
	planFile = ""
	readOnlyMode = true
	programRunner = func(_ tea.Model) error {
		t.Fatalf("program runner should not be called on error")
		return nil
	}

	if err := runMain(); err == nil {
		t.Fatalf("expected runMain error")
	}
}

type fakeWorkspaceManager struct {
	switchErr error
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

type fakeModel struct{}

func (f *fakeModel) Init() tea.Cmd {
	return nil
}

func (f *fakeModel) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return f, nil
}

func (f *fakeModel) View() string {
	return ""
}
