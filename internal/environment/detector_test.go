package environment

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestDetectWorkspaceStrategy(t *testing.T) {
	detector, err := NewDetector(t.TempDir(), WithWorkspaceListFunc(func(ctx context.Context, workDir string) ([]string, error) {
		return []string{"default", "dev"}, nil
	}))
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	result, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if result.Strategy != StrategyWorkspace {
		t.Fatalf("expected workspace strategy, got %q", result.Strategy)
	}
	if result.Confidence[StrategyWorkspace] <= 0 {
		t.Fatalf("expected workspace confidence > 0")
	}
}

func TestDetectFolderStrategyNested(t *testing.T) {
	root := t.TempDir()
	dirs := []string{
		filepath.Join(root, "envs", "dev"),
		filepath.Join(root, "envs", "prod"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		file := filepath.Join(dir, "main.tf")
		if err := os.WriteFile(file, []byte(""), 0o644); err != nil {
			t.Fatalf("write tf: %v", err)
		}
	}

	detector, err := NewDetector(root, WithWorkspaceListFunc(func(ctx context.Context, workDir string) ([]string, error) {
		return nil, nil
	}))
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	result, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if result.Strategy != StrategyFolder {
		t.Fatalf("expected folder strategy, got %q", result.Strategy)
	}
	paths := append([]string{}, result.FolderPaths...)
	sort.Strings(paths)
	sort.Strings(dirs)
	if !reflect.DeepEqual(paths, dirs) {
		t.Fatalf("expected folders %v, got %v", dirs, paths)
	}
}

func TestDetectMixedStrategy(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "envs", "dev")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(""), 0o644); err != nil {
		t.Fatalf("write tf: %v", err)
	}

	detector, err := NewDetector(root, WithWorkspaceListFunc(func(ctx context.Context, workDir string) ([]string, error) {
		return []string{"default", "prod"}, nil
	}))
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	result, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if result.Strategy != StrategyMixed {
		t.Fatalf("expected mixed strategy, got %q", result.Strategy)
	}
	if result.Confidence[StrategyMixed] <= 0 {
		t.Fatalf("expected mixed confidence > 0")
	}
}

func TestEmptyWorkspaceList(t *testing.T) {
	detector, err := NewDetector(t.TempDir(), WithWorkspaceListFunc(func(ctx context.Context, workDir string) ([]string, error) {
		return []string{}, nil
	}))
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	result, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if result.Strategy != StrategyUnknown {
		t.Fatalf("expected unknown strategy, got %q", result.Strategy)
	}
	if result.Confidence[StrategyWorkspace] != 0 {
		t.Fatalf("expected workspace confidence 0, got %v", result.Confidence[StrategyWorkspace])
	}
}
