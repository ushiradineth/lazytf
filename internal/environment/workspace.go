package environment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ushiradineth/lazytf/internal/tfbinary"
)

// WorkspaceListOutputFunc returns the raw workspace list output for a directory.
type WorkspaceListOutputFunc func(ctx context.Context, workDir string) (string, error)

// WorkspaceSelectFunc selects a workspace in the given directory.
type WorkspaceSelectFunc func(ctx context.Context, workDir, name string) error

// WorkspaceManager manages Terraform workspaces.
type WorkspaceManager struct {
	workDir         string
	listOutput      WorkspaceListOutputFunc
	selectWorkspace WorkspaceSelectFunc
}

// WorkspaceManagerOption configures a WorkspaceManager.
type WorkspaceManagerOption func(*WorkspaceManager) error

// WithWorkspaceListOutputFunc overrides how workspace list output is fetched.
func WithWorkspaceListOutputFunc(fn WorkspaceListOutputFunc) WorkspaceManagerOption {
	return func(m *WorkspaceManager) error {
		if fn == nil {
			return errors.New("workspace list output function cannot be nil")
		}
		m.listOutput = fn
		return nil
	}
}

// WithWorkspaceSelectFunc overrides how workspace selection is executed.
func WithWorkspaceSelectFunc(fn WorkspaceSelectFunc) WorkspaceManagerOption {
	return func(m *WorkspaceManager) error {
		if fn == nil {
			return errors.New("workspace select function cannot be nil")
		}
		m.selectWorkspace = fn
		return nil
	}
}

// NewWorkspaceManager creates a WorkspaceManager for the provided working directory.
func NewWorkspaceManager(workDir string, opts ...WorkspaceManagerOption) (*WorkspaceManager, error) {
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

	manager := &WorkspaceManager{
		workDir:         absDir,
		listOutput:      terraformWorkspaceListOutput,
		selectWorkspace: terraformWorkspaceSelect,
	}
	for _, opt := range opts {
		if err := opt(manager); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

// List returns all available workspaces.
func (m *WorkspaceManager) List(ctx context.Context) ([]string, error) {
	if m == nil {
		return nil, errors.New("workspace manager is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	output, err := m.listOutput(ctx, m.workDir)
	if err != nil {
		return nil, err
	}
	parsed := parseWorkspaceListOutput(output)
	return parsed.Workspaces, nil
}

// Current returns the currently selected workspace.
func (m *WorkspaceManager) Current(ctx context.Context) (string, error) {
	if m == nil {
		return "", errors.New("workspace manager is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	output, err := m.listOutput(ctx, m.workDir)
	if err != nil {
		return "", err
	}
	parsed := parseWorkspaceListOutput(output)
	if parsed.Current == "" {
		return "", errors.New("current workspace not found")
	}
	return parsed.Current, nil
}

// Validate ensures the named workspace exists.
func (m *WorkspaceManager) Validate(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("workspace name cannot be empty")
	}
	workspaces, err := m.List(ctx)
	if err != nil {
		return err
	}
	for _, workspace := range workspaces {
		if workspace == name {
			return nil
		}
	}
	return fmt.Errorf("workspace not found: %s", name)
}

// Switch changes to the provided workspace.
func (m *WorkspaceManager) Switch(ctx context.Context, name string) error {
	if m == nil {
		return errors.New("workspace manager is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.Validate(ctx, name); err != nil {
		return err
	}
	return m.selectWorkspace(ctx, m.workDir, name)
}

type workspaceListResult struct {
	Workspaces []string
	Current    string
}

func parseWorkspaceListOutput(output string) workspaceListResult {
	lines := strings.Split(output, "\n")
	workspaces := make([]string, 0, len(lines))
	current := ""
	for _, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == "" {
			continue
		}
		isCurrent := strings.HasPrefix(entry, "*")
		entry = strings.TrimSpace(strings.TrimPrefix(entry, "*"))
		if entry == "" {
			continue
		}
		workspaces = append(workspaces, entry)
		if isCurrent {
			current = entry
		}
	}
	return workspaceListResult{Workspaces: workspaces, Current: current}
}

func terraformWorkspaceListOutput(ctx context.Context, workDir string) (string, error) {
	path, err := resolveTerraformBinaryPath()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, path, "workspace", "list", "-no-color")
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return "", fmt.Errorf("terraform/tofu workspace list failed: %w", err)
		}
		return "", fmt.Errorf("terraform/tofu workspace list failed: %w: %s", err, trimmed)
	}

	return string(output), nil
}

func terraformWorkspaceSelect(ctx context.Context, workDir, name string) error {
	path, err := resolveTerraformBinaryPath()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, path, "workspace", "select", "-no-color", name)
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("terraform/tofu workspace select failed: %w", err)
		}
		return fmt.Errorf("terraform/tofu workspace select failed: %w: %s", err, trimmed)
	}
	return nil
}

func resolveTerraformBinaryPath() (string, error) {
	return tfbinary.Resolve()
}
