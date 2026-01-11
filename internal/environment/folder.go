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
)

// FolderInfo describes a detected environment folder.
type FolderInfo struct {
	Path     string
	Score    float64
	HasState bool
}

// FolderScanFunc returns environment folders for a root directory.
type FolderScanFunc func(ctx context.Context, root string, maxDepth int) ([]FolderInfo, error)

// ChangeDirFunc changes the process working directory.
type ChangeDirFunc func(path string) error

// FolderManager manages folder-based environment selection.
type FolderManager struct {
	workDir   string
	maxDepth  int
	scan      FolderScanFunc
	changeDir ChangeDirFunc
}

// FolderManagerOption configures a FolderManager.
type FolderManagerOption func(*FolderManager) error

// WithFolderScanFunc overrides how environment folders are discovered.
func WithFolderScanFunc(fn FolderScanFunc) FolderManagerOption {
	return func(m *FolderManager) error {
		if fn == nil {
			return errors.New("folder scan function cannot be nil")
		}
		m.scan = fn
		return nil
	}
}

// WithFolderChangeDirFunc overrides how directory changes are applied.
func WithFolderChangeDirFunc(fn ChangeDirFunc) FolderManagerOption {
	return func(m *FolderManager) error {
		if fn == nil {
			return errors.New("change dir function cannot be nil")
		}
		m.changeDir = fn
		return nil
	}
}

// WithFolderMaxDepth sets the maximum depth to scan for environment folders.
func WithFolderMaxDepth(depth int) FolderManagerOption {
	return func(m *FolderManager) error {
		if depth < 0 {
			return errors.New("max depth cannot be negative")
		}
		m.maxDepth = depth
		return nil
	}
}

// NewFolderManager creates a FolderManager for the provided working directory.
func NewFolderManager(workDir string, opts ...FolderManagerOption) (*FolderManager, error) {
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

	manager := &FolderManager{
		workDir:   absDir,
		maxDepth:  defaultMaxDepth,
		scan:      scanEnvironmentFolders,
		changeDir: os.Chdir,
	}
	for _, opt := range opts {
		if err := opt(manager); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

// List returns environment folders discovered under the working directory.
func (m *FolderManager) List(ctx context.Context) ([]FolderInfo, error) {
	if m == nil {
		return nil, errors.New("folder manager is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	folders, err := m.scan(ctx, m.workDir, m.maxDepth)
	if err != nil {
		return nil, err
	}
	return folders, nil
}

// Validate ensures a folder exists and contains Terraform files.
func (m *FolderManager) Validate(_ context.Context, path string) error {
	if m == nil {
		return errors.New("folder manager is nil")
	}
	if strings.TrimSpace(path) == "" {
		return errors.New("folder path cannot be empty")
	}
	resolved, err := m.resolveFolderPath(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("folder not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("folder is not a directory: %s", resolved)
	}
	hasTF, err := folderHasTerraformFiles(resolved)
	if err != nil {
		return err
	}
	if !hasTF {
		return fmt.Errorf("no terraform files in folder: %s", resolved)
	}
	return nil
}

// Switch changes the working directory to the named folder.
func (m *FolderManager) Switch(ctx context.Context, path string) error {
	if m == nil {
		return errors.New("folder manager is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.Validate(ctx, path); err != nil {
		return err
	}
	resolved, err := m.resolveFolderPath(path)
	if err != nil {
		return err
	}
	return m.changeDir(resolved)
}

func (m *FolderManager) resolveFolderPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Abs(path)
	}
	return filepath.Abs(filepath.Join(m.workDir, path))
}

func scanEnvironmentFolders(ctx context.Context, root string, maxDepth int) ([]FolderInfo, error) {
	folders := make(map[string]FolderInfo)
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
		if !containsEnvSegment(dir) {
			return nil
		}

		if _, exists := folders[dir]; !exists {
			folders[dir] = FolderInfo{Path: dir}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	paths := make([]FolderInfo, 0, len(folders))
	for _, info := range folders {
		score, hasState, err := scoreFolder(info.Path)
		if err != nil {
			return nil, err
		}
		info.Score = score
		info.HasState = hasState
		paths = append(paths, info)
	}

	sort.Slice(paths, func(i, j int) bool {
		if paths[i].Score == paths[j].Score {
			return paths[i].Path < paths[j].Path
		}
		return paths[i].Score > paths[j].Score
	})
	return paths, nil
}

func containsEnvSegment(path string) bool {
	segments := strings.Split(filepath.Clean(path), string(filepath.Separator))
	for _, segment := range segments {
		switch segment {
		case "envs", "environments":
			return true
		default:
		}
	}
	return false
}

func folderHasTerraformFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("read folder: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".tf") {
			return true, nil
		}
	}
	return false, nil
}

func folderHasState(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("read folder: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			if name == "terraform.tfstate.d" || strings.HasSuffix(name, ".tfstate.d") {
				return true, nil
			}
			continue
		}
		if name == "terraform.tfstate" || strings.HasSuffix(name, ".tfstate") {
			return true, nil
		}
	}
	return false, nil
}

func scoreFolder(path string) (float64, bool, error) {
	score := 0.4
	hasState, err := folderHasState(path)
	if err != nil {
		return 0, false, err
	}
	if hasState {
		score += 0.3
	}
	if hasNamingHint(filepath.Base(path)) {
		score += 0.2
	}
	if score > 1 {
		score = 1
	}
	return score, hasState, nil
}

func hasNamingHint(name string) bool {
	lower := strings.ToLower(name)
	candidates := []string{
		"prod", "production", "stage", "staging",
		"dev", "development", "test", "qa", "uat", "sandbox",
	}
	for _, candidate := range candidates {
		if lower == candidate {
			return true
		}
		if strings.Contains(lower, candidate) {
			return true
		}
	}
	return false
}
