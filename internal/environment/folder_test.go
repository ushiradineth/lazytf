package environment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFolderManagerListOrdersByScore(t *testing.T) {
	root := t.TempDir()
	prodDir := filepath.Join(root, "envs", "prod")
	devDir := filepath.Join(root, "envs", "dev")
	for _, dir := range []string{prodDir, devDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(""), 0o600); err != nil {
			t.Fatalf("write tf: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(prodDir, "terraform.tfstate"), []byte(""), 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}

	manager, err := NewFolderManager(root)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	folders, err := manager.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(folders))
	}
	if folders[0].Path != prodDir {
		t.Fatalf("expected %s first, got %s", prodDir, folders[0].Path)
	}
	if !folders[0].HasState {
		t.Fatalf("expected prod to have state")
	}
	if folders[0].Score <= folders[1].Score {
		t.Fatalf("expected prod score higher than dev")
	}
}

func TestFolderManagerValidate(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "envs", "dev")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	manager, err := NewFolderManager(root)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := manager.Validate(context.Background(), target); err == nil {
		t.Fatal("expected error for missing terraform files")
	}
	if err := os.WriteFile(filepath.Join(target, "main.tf"), []byte(""), 0o600); err != nil {
		t.Fatalf("write tf: %v", err)
	}
	if err := manager.Validate(context.Background(), target); err != nil {
		t.Fatalf("expected validation success, got %v", err)
	}
}

func TestFolderManagerSwitchUsesChangeDir(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "envs", "prod")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "main.tf"), []byte(""), 0o600); err != nil {
		t.Fatalf("write tf: %v", err)
	}

	changed := ""
	manager, err := NewFolderManager(root, WithFolderChangeDirFunc(func(path string) error {
		changed = path
		return nil
	}))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := manager.Switch(context.Background(), filepath.Join("envs", "prod")); err != nil {
		t.Fatalf("switch: %v", err)
	}
	if changed != target {
		t.Fatalf("expected chdir to %s, got %s", target, changed)
	}
}

func TestFolderHasState(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "terraform.tfstate"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}
	hasState, err := folderHasState(dir)
	if err != nil {
		t.Fatalf("folderHasState error: %v", err)
	}
	if !hasState {
		t.Fatalf("expected state file detection")
	}
}

func TestFolderHasStateDir(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "terraform.tfstate.d")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	hasState, err := folderHasState(dir)
	if err != nil {
		t.Fatalf("folderHasState error: %v", err)
	}
	if !hasState {
		t.Fatalf("expected state dir detection")
	}
}

func TestScanEnvironmentFoldersNested(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "teams", "alpha", "envs", "staging")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "main.tf"), []byte(""), 0o600); err != nil {
		t.Fatalf("write tf: %v", err)
	}

	folders, err := scanEnvironmentFolders(context.Background(), root, defaultMaxDepth)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(folders) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(folders))
	}
	if folders[0].Path != target {
		t.Fatalf("expected %s, got %s", target, folders[0].Path)
	}
}

func TestWithFolderScanFuncNil(t *testing.T) {
	_, err := NewFolderManager(t.TempDir(), WithFolderScanFunc(nil))
	if err == nil {
		t.Fatalf("expected error for nil scan func")
	}
}

func TestWithFolderMaxDepthNegative(t *testing.T) {
	_, err := NewFolderManager(t.TempDir(), WithFolderMaxDepth(-1))
	if err == nil {
		t.Fatalf("expected error for negative max depth")
	}
}

func TestWithFolderMaxDepthSuccess(t *testing.T) {
	_, err := NewFolderManager(t.TempDir(), WithFolderMaxDepth(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFolderManagerListError(t *testing.T) {
	manager, err := NewFolderManager(t.TempDir(), WithFolderScanFunc(func(_ context.Context, _ string, _ int) ([]FolderInfo, error) {
		return nil, errors.New("boom")
	}))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if _, err := manager.List(context.Background()); err == nil {
		t.Fatalf("expected list error")
	}
}

func TestFolderManagerValidateNil(t *testing.T) {
	var manager *FolderManager
	if err := manager.Validate(context.Background(), "path"); err == nil {
		t.Fatalf("expected error for nil manager")
	}
}

func TestFolderManagerSwitchNil(t *testing.T) {
	var manager *FolderManager
	if err := manager.Switch(context.Background(), "path"); err == nil {
		t.Fatalf("expected error for nil manager")
	}
}

func TestFolderManagerListNilManager(t *testing.T) {
	var manager *FolderManager
	if _, err := manager.List(context.Background()); err == nil {
		t.Fatal("expected error for nil manager")
	}
}

func TestFolderManagerValidateEmptyPath(t *testing.T) {
	manager, err := NewFolderManager(t.TempDir())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := manager.Validate(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty path")
	}
	if err := manager.Validate(context.Background(), "   "); err == nil {
		t.Fatal("expected error for whitespace path")
	}
}

func TestFolderManagerValidateNotDirectory(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	manager, err := NewFolderManager(root)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := manager.Validate(context.Background(), filePath); err == nil {
		t.Fatal("expected error for non-directory")
	}
}

func TestContainsEnvSegmentVariations(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/projects/envs/dev", true},
		{"/projects/environments/prod", true},
		{"/projects/env/dev", false},
		{"/projects/dev", false},
		{"/envs", true},
		{"/environments", true},
		{"", false},
	}

	for _, tt := range tests {
		got := containsEnvSegment(tt.path)
		if got != tt.want {
			t.Errorf("containsEnvSegment(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestHandleFolderFileNonTfFile(t *testing.T) {
	folders := make(map[string]FolderInfo)
	// Create a mock DirEntry for a non-.tf file
	entry := mockDirEntry{name: "readme.md"}
	err := handleFolderFile(entry, "/path/envs/dev/readme.md", folders)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(folders) != 0 {
		t.Fatalf("expected no folders, got %d", len(folders))
	}
}

func TestSortFolderInfosByScoreAndPath(t *testing.T) {
	folders := []FolderInfo{
		{Path: "/a/envs/dev", Score: 10},
		{Path: "/a/envs/prod", Score: 20},
		{Path: "/a/envs/staging", Score: 10},
	}
	sortFolderInfos(folders)

	// prod (score 20) should be first
	if folders[0].Path != "/a/envs/prod" {
		t.Errorf("expected prod first, got %s", folders[0].Path)
	}
	// dev and staging both have score 10, should be alphabetical
	if folders[1].Path != "/a/envs/dev" {
		t.Errorf("expected dev second, got %s", folders[1].Path)
	}
	if folders[2].Path != "/a/envs/staging" {
		t.Errorf("expected staging third, got %s", folders[2].Path)
	}
}

func TestCheckWalkContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := checkWalkContext(ctx, nil)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestCheckWalkContextWithError(t *testing.T) {
	ctx := context.Background()
	walkErr := errors.New("walk error")

	err := checkWalkContext(ctx, walkErr)
	if err != walkErr {
		t.Errorf("expected walk error, got %v", err)
	}
}

// mockDirEntry is a mock implementation of fs.DirEntry for testing
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() os.FileMode          { return 0 }
func (m mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }
