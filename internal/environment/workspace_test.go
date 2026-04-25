package environment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/consts"
)

func TestParseWorkspaceListOutput(t *testing.T) {
	output := "  default\n* dev\n  staging\n\n"
	parsed := parseWorkspaceListOutput(output)
	wantList := []string{consts.DefaultName, consts.EnvDev, "staging"}
	if !reflect.DeepEqual(parsed.Workspaces, wantList) {
		t.Fatalf("expected workspaces %v, got %v", wantList, parsed.Workspaces)
	}
	if parsed.Current != consts.EnvDev {
		t.Fatalf("expected current dev, got %q", parsed.Current)
	}
}

func TestWorkspaceManagerListAndCurrent(t *testing.T) {
	manager, err := NewWorkspaceManager(t.TempDir(), WithWorkspaceListOutputFunc(func(_ context.Context, _ string, _ string) (string, error) {
		return "  default\n* prod\n", nil
	}))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	workspaces, err := manager.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	wantList := []string{"default", "prod"}
	if !reflect.DeepEqual(workspaces, wantList) {
		t.Fatalf("expected %v, got %v", wantList, workspaces)
	}

	current, err := manager.Current(context.Background())
	if err != nil {
		t.Fatalf("current: %v", err)
	}
	if current != "prod" {
		t.Fatalf("expected current prod, got %q", current)
	}
}

func TestWorkspaceManagerCurrentMissing(t *testing.T) {
	manager, err := NewWorkspaceManager(t.TempDir(), WithWorkspaceListOutputFunc(func(_ context.Context, _ string, _ string) (string, error) {
		return "default\n", nil
	}))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	_, err = manager.Current(context.Background())
	if err == nil {
		t.Fatal("expected error for missing current workspace")
	}
}

func TestWorkspaceManagerSwitchValidates(t *testing.T) {
	selected := ""
	manager, err := NewWorkspaceManager(t.TempDir(),
		WithWorkspaceListOutputFunc(func(_ context.Context, _ string, _ string) (string, error) {
			return "* dev\n  prod\n", nil
		}),
		WithWorkspaceSelectFunc(func(_ context.Context, _ string, name string, _ string) error {
			selected = name
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := manager.Switch(context.Background(), "prod"); err != nil {
		t.Fatalf("switch: %v", err)
	}
	if selected != "prod" {
		t.Fatalf("expected selection prod, got %q", selected)
	}

	if err := manager.Switch(context.Background(), "unknown"); err == nil {
		t.Fatal("expected error for unknown workspace")
	}
}

func TestWorkspaceManagerListError(t *testing.T) {
	manager, err := NewWorkspaceManager(t.TempDir(), WithWorkspaceListOutputFunc(func(_ context.Context, _ string, _ string) (string, error) {
		return "", errors.New("boom")
	}))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	_, err = manager.List(context.Background())
	if err == nil {
		t.Fatal("expected list error")
	}
}

func TestWithWorkspaceListOutputFuncNil(t *testing.T) {
	_, err := NewWorkspaceManager(t.TempDir(), WithWorkspaceListOutputFunc(nil))
	if err == nil {
		t.Fatalf("expected error for nil list func")
	}
}

func TestWithWorkspaceSelectFuncNil(t *testing.T) {
	_, err := NewWorkspaceManager(t.TempDir(), WithWorkspaceSelectFunc(nil))
	if err == nil {
		t.Fatalf("expected error for nil select func")
	}
}

func TestTerraformWorkspaceListOutputMissingBinary(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := terraformWorkspaceListOutput(context.Background(), t.TempDir(), "")
	if err == nil {
		t.Fatalf("expected error when terraform binary missing")
	}
}

func TestTerraformWorkspaceSelectMissingBinary(t *testing.T) {
	t.Setenv("PATH", "")
	if err := terraformWorkspaceSelect(context.Background(), t.TempDir(), consts.EnvDev, ""); err == nil {
		t.Fatalf("expected error when terraform binary missing")
	}
}

func TestTerraformWorkspaceListOutputSuccess(t *testing.T) {
	setupFakeTerraform(t)
	out, err := terraformWorkspaceListOutput(context.Background(), t.TempDir(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, consts.DefaultName) || !strings.Contains(out, consts.EnvDev) {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestTerraformWorkspaceSelectSuccess(t *testing.T) {
	setupFakeTerraform(t)
	if err := terraformWorkspaceSelect(context.Background(), t.TempDir(), consts.EnvDev, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTerraformWorkspaceListOutputTofuFallback(t *testing.T) {
	setupFakeTofu(t)
	out, err := terraformWorkspaceListOutput(context.Background(), t.TempDir(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, consts.DefaultName) || !strings.Contains(out, consts.EnvDev) {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestTerraformWorkspaceSelectTofuFallback(t *testing.T) {
	setupFakeTofu(t)
	if err := terraformWorkspaceSelect(context.Background(), t.TempDir(), consts.EnvDev, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTerraformWorkspaceListOutputUsesPreferredBinary(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}

	dir := t.TempDir()
	writeWorkspaceBinaryScript(t, dir, "custom-tofu", "dev")
	preferredPath := filepath.Join(dir, "custom-tofu")
	t.Setenv("PATH", "")

	out, err := terraformWorkspaceListOutput(context.Background(), t.TempDir(), preferredPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parsed := parseWorkspaceListOutput(out)
	if parsed.Current != "dev" {
		t.Fatalf("expected preferred binary workspace, got %q", parsed.Current)
	}
}

func TestTerraformWorkspaceListOutputPrefersTerraformOverTofu(t *testing.T) {
	if runtime.GOOS == consts.OSWindows {
		t.Skip("shell script test not supported on windows")
	}

	dir := t.TempDir()
	writeWorkspaceBinaryScript(t, dir, "terraform", "dev")
	writeWorkspaceBinaryScript(t, dir, "tofu", "tofu-dev")
	t.Setenv("PATH", dir)

	out, err := terraformWorkspaceListOutput(context.Background(), t.TempDir(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parsed := parseWorkspaceListOutput(out)
	if parsed.Current != "dev" {
		t.Fatalf("expected terraform to win, got current workspace %q", parsed.Current)
	}
}

func TestTerraformWorkspaceSelectErrorOutput(t *testing.T) {
	setupFakeTerraformError(t)
	if err := terraformWorkspaceSelect(context.Background(), t.TempDir(), consts.EnvDev, ""); err == nil {
		t.Fatalf("expected error when terraform workspace select fails")
	}
}

func TestWorkspaceManagerNil(t *testing.T) {
	var manager *WorkspaceManager
	if _, err := manager.List(context.Background()); err == nil {
		t.Fatalf("expected list error for nil manager")
	}
	if _, err := manager.Current(context.Background()); err == nil {
		t.Fatalf("expected current error for nil manager")
	}
	if err := manager.Switch(context.Background(), consts.EnvDev); err == nil {
		t.Fatalf("expected switch error for nil manager")
	}
	if err := manager.Validate(context.Background(), consts.EnvDev); err == nil {
		t.Fatalf("expected validate error for nil manager")
	}
}

func writeWorkspaceBinaryScript(t *testing.T, dir, binaryName, currentWorkspace string) {
	t.Helper()
	path := filepath.Join(dir, binaryName)
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"workspace\" ] && [ \"$2\" = \"list\" ]; then\n" +
		"  echo \"  default\"\n" +
		"  echo \"* " + currentWorkspace + "\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"workspace\" ] && [ \"$2\" = \"select\" ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write %s script: %v", binaryName, err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("chmod %s script: %v", binaryName, err)
	}
}
