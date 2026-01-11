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
	detector, err := NewDetector(t.TempDir(), WithWorkspaceListFunc(func(_ context.Context, _ string) ([]string, error) {
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

	detector, err := NewDetector(root, WithWorkspaceListFunc(func(_ context.Context, _ string) ([]string, error) {
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

	detector, err := NewDetector(root, WithWorkspaceListFunc(func(_ context.Context, _ string) ([]string, error) {
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
	detector, err := NewDetector(t.TempDir(), WithWorkspaceListFunc(func(_ context.Context, _ string) ([]string, error) {
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

func TestWithMaxDepthNegative(t *testing.T) {
	_, err := NewDetector(t.TempDir(), WithMaxDepth(-1))
	if err == nil {
		t.Fatalf("expected error for negative max depth")
	}
}

func TestParseWorkspaceList(t *testing.T) {
	output := "  default\n* dev\n  staging\n\n"
	parsed := parseWorkspaceList(output)
	if len(parsed) != 3 || parsed[1] != "dev" {
		t.Fatalf("unexpected parsed list: %#v", parsed)
	}
}

func TestTerraformWorkspaceListMissingBinary(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := terraformWorkspaceList(context.Background(), t.TempDir())
	if err == nil {
		t.Fatalf("expected error when terraform binary missing")
	}
}

func TestTerraformWorkspaceListSuccess(t *testing.T) {
	setupFakeTerraform(t)
	workspaces, err := terraformWorkspaceList(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workspaces) != 2 || workspaces[1] != "dev" {
		t.Fatalf("unexpected workspaces: %#v", workspaces)
	}
}

func TestWithMaxDepthSuccess(t *testing.T) {
	_, err := NewDetector(t.TempDir(), WithMaxDepth(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShouldSkipDirAndIgnoredSegment(t *testing.T) {
	if !shouldSkipDir(".terraform") {
		t.Fatalf("expected terraform dir to be skipped")
	}
	if shouldSkipDir("envs") {
		t.Fatalf("did not expect envs to be skipped")
	}
	if !containsIgnoredSegment("path/to/.git/modules") {
		t.Fatalf("expected ignored segment detection")
	}
	if containsIgnoredSegment("path/to/envs") {
		t.Fatalf("did not expect ignored segment")
	}
}

func TestWorkspaceConfidenceScores(t *testing.T) {
	if got := workspaceConfidence(nil); got != 0 {
		t.Fatalf("expected zero confidence")
	}
	if got := workspaceConfidence([]string{"default"}); got != 0.3 {
		t.Fatalf("expected default-only confidence")
	}
	if got := workspaceConfidence([]string{"dev"}); got != 0.6 {
		t.Fatalf("expected single non-default confidence")
	}
	if got := workspaceConfidence([]string{"default", "dev"}); got != 0.8 {
		t.Fatalf("expected multi workspace confidence")
	}
}

func TestTerraformWorkspaceListErrorOutput(t *testing.T) {
	setupFakeTerraformError(t)
	_, err := terraformWorkspaceList(context.Background(), t.TempDir())
	if err == nil {
		t.Fatalf("expected error for terraform workspace list")
	}
}
