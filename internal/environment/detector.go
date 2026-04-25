// Package environment provides automatic detection and management of
// Terraform environment strategies (workspace-based or folder-based).
//
// The Detector analyzes a Terraform project to determine whether it uses
// workspaces, folder-based environments, or a combination of both. It
// provides confidence scores for each strategy to help guide environment
// selection.
//
// WorkspaceManager handles workspace selection and validation, while
// FolderManager handles folder-based environment switching.
package environment

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ushiradineth/lazytf/internal/tfbinary"
)

// StrategyType represents the detected environment strategy.
type StrategyType string

const (
	StrategyUnknown   StrategyType = "unknown"
	StrategyWorkspace StrategyType = "workspace"
	StrategyFolder    StrategyType = "folder"
	StrategyMixed     StrategyType = "mixed"
)

const defaultMaxDepth = 6

// DetectionResult captures environment detection details.
type DetectionResult struct {
	Strategy        StrategyType
	Confidence      map[StrategyType]float64
	ConfidenceScore float64
	Workspaces      []string
	FolderPaths     []string
	Warnings        []string
	BaseDir         string
	Environments    []Environment
}

// WorkspaceListFunc returns available workspaces for the given directory.
type WorkspaceListFunc func(ctx context.Context, workDir string) ([]string, error)

// Detector identifies environment strategies for a Terraform project.
type Detector struct {
	workDir        string
	binaryPath     string
	listWorkspaces WorkspaceListFunc
	maxDepth       int
}

// DetectorOption configures a Detector.
type DetectorOption func(*Detector) error

// WithWorkspaceListFunc overrides how workspaces are listed.
func WithWorkspaceListFunc(fn WorkspaceListFunc) DetectorOption {
	return func(d *Detector) error {
		if fn == nil {
			return errors.New("workspace list function cannot be nil")
		}
		d.listWorkspaces = fn
		return nil
	}
}

// WithMaxDepth sets the maximum depth to scan for Terraform folders.
func WithMaxDepth(depth int) DetectorOption {
	return func(d *Detector) error {
		if depth < 0 {
			return errors.New("max depth cannot be negative")
		}
		d.maxDepth = depth
		return nil
	}
}

// WithBinaryPath sets a preferred terraform/tofu binary path.
func WithBinaryPath(path string) DetectorOption {
	return func(d *Detector) error {
		d.binaryPath = strings.TrimSpace(path)
		return nil
	}
}

// NewDetector creates a Detector for the provided working directory.
func NewDetector(workDir string, opts ...DetectorOption) (*Detector, error) {
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	absDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve working directory %s: %w", workDir, err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("working directory not found %s: %w", absDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("working directory is not a directory: %s", absDir)
	}

	detector := &Detector{workDir: absDir, maxDepth: defaultMaxDepth}
	detector.listWorkspaces = func(ctx context.Context, dir string) ([]string, error) {
		return terraformWorkspaceList(ctx, dir, detector.binaryPath)
	}
	for _, opt := range opts {
		if err := opt(detector); err != nil {
			return nil, err
		}
	}
	return detector, nil
}

// Detect evaluates the working directory and returns a detection result.
func (d *Detector) Detect(ctx context.Context) (DetectionResult, error) {
	if d == nil {
		return DetectionResult{}, errors.New("detector is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	result := DetectionResult{
		Strategy: StrategyUnknown,
		BaseDir:  d.workDir,
		Confidence: map[StrategyType]float64{
			StrategyWorkspace: 0,
			StrategyFolder:    0,
			StrategyMixed:     0,
		},
	}

	workspaces, err := d.listWorkspaces(ctx, d.workDir)
	if err != nil {
		result.Warnings = append(result.Warnings, err.Error())
	} else {
		result.Workspaces = workspaces
	}

	folders, err := scanTerraformFolders(ctx, d.workDir, d.maxDepth)
	if err != nil {
		return result, err
	}
	result.FolderPaths = folders

	workspaceScore := workspaceConfidence(result.Workspaces)
	folderScore := folderConfidence(result.FolderPaths)
	result.Confidence[StrategyWorkspace] = workspaceScore
	result.Confidence[StrategyFolder] = folderScore

	switch {
	case workspaceScore > 0 && folderScore > 0:
		mixedScore := (workspaceScore+folderScore)/2 + 0.1
		if mixedScore > 1 {
			mixedScore = 1
		}
		result.Confidence[StrategyMixed] = mixedScore
		result.Strategy = StrategyMixed
	case workspaceScore >= folderScore && workspaceScore > 0:
		result.Strategy = StrategyWorkspace
	case folderScore > 0:
		result.Strategy = StrategyFolder
	}

	result.ConfidenceScore = result.Confidence[result.Strategy]
	result.Environments = BuildEnvironments(result, "")

	return result, nil
}

func terraformWorkspaceList(ctx context.Context, workDir, preferredBinary string) ([]string, error) {
	runtime, err := tfbinary.NewRuntime(preferredBinary)
	if err != nil {
		return nil, err
	}

	output, err := runtime.CombinedOutput(ctx, workDir, "workspace", "list", "-no-color")
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return nil, fmt.Errorf("terraform/tofu workspace list failed: %w", err)
		}
		return nil, fmt.Errorf("terraform/tofu workspace list failed: %w: %s", err, trimmed)
	}

	return parseWorkspaceList(string(output)), nil
}

func parseWorkspaceList(output string) []string {
	lines := strings.Split(output, "\n")
	workspaces := make([]string, 0, len(lines))
	for _, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == "" {
			continue
		}
		entry = strings.TrimPrefix(entry, "*")
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		workspaces = append(workspaces, entry)
	}
	return workspaces
}

func scanTerraformFolders(ctx context.Context, root string, maxDepth int) ([]string, error) {
	folders := make(map[string]struct{})
	baseDepth := pathDepth(root)

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}

		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return fs.SkipDir
			}
			if maxDepth > 0 && pathDepth(path)-baseDepth > maxDepth {
				return fs.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".tf") {
			return nil
		}

		dir := filepath.Dir(path)
		if containsIgnoredSegment(dir) {
			return nil
		}
		folders[dir] = struct{}{}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	paths := make([]string, 0, len(folders))
	for path := range folders {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func pathDepth(path string) int {
	clean := filepath.Clean(path)
	if clean == string(filepath.Separator) {
		return 0
	}
	return strings.Count(clean, string(filepath.Separator))
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".terraform", ".git", "vendor":
		return true
	default:
		return false
	}
}

func containsIgnoredSegment(path string) bool {
	segments := strings.Split(filepath.Clean(path), string(filepath.Separator))
	for _, segment := range segments {
		switch segment {
		case ".terraform", ".git", "vendor", "modules", "module":
			return true
		default:
		}
	}
	return false
}

func workspaceConfidence(workspaces []string) float64 {
	if len(workspaces) == 0 {
		return 0
	}

	nonDefault := 0
	for _, name := range workspaces {
		if name != "default" {
			nonDefault++
		}
	}

	if len(workspaces) == 1 && nonDefault == 0 {
		return 0.3
	}
	if nonDefault > 0 && len(workspaces) > 1 {
		return 0.8
	}
	if nonDefault > 0 {
		return 0.6
	}
	return 0.4
}

func folderConfidence(folders []string) float64 {
	if len(folders) == 0 {
		return 0
	}
	score := 0.4
	if len(folders) > 1 {
		score = 0.7
	}

	for _, path := range folders {
		lower := strings.ToLower(path)
		if strings.Contains(lower, string(filepath.Separator)+"envs"+string(filepath.Separator)) ||
			strings.Contains(lower, string(filepath.Separator)+"environments"+string(filepath.Separator)) {
			score += 0.2
			break
		}
	}
	if score > 1 {
		return 1
	}
	return score
}
