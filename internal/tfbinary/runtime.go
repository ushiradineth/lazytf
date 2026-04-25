package tfbinary

import (
	"context"
	"os/exec"
)

// Runtime encapsulates a resolved terraform-compatible binary path.
type Runtime struct {
	path string
}

// NewRuntimeFromPath creates a runtime from an already validated path.
func NewRuntimeFromPath(path string) Runtime {
	return Runtime{path: path}
}

// NewRuntime resolves a runtime using preferred binary when provided.
func NewRuntime(preferred string) (Runtime, error) {
	path, err := ResolvePreferred(preferred)
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{path: path}, nil
}

// Path returns the resolved binary path.
func (r Runtime) Path() string {
	return r.path
}

// CommandContext creates a command using the resolved runtime path.
func (r Runtime) CommandContext(ctx context.Context, args ...string) *exec.Cmd {
	// #nosec G204 -- terraform/tofu execution is intentional and arguments are controlled by caller.
	return exec.CommandContext(ctx, r.path, args...)
}

// CombinedOutput runs a command with directory and returns combined output.
func (r Runtime) CombinedOutput(ctx context.Context, workDir string, args ...string) ([]byte, error) {
	cmd := r.CommandContext(ctx, args...)
	cmd.Dir = workDir
	return cmd.CombinedOutput()
}
