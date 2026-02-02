package environment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMetadataFromState(t *testing.T) {
	// Create a temp directory with a state file
	tmpDir, err := os.MkdirTemp("", "metadata-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid terraform state file
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{
		"terraform_version": "1.5.0",
		"resources": [
			{"type": "aws_instance", "name": "web"},
			{"type": "aws_vpc", "name": "main"}
		]
	}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0o600); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Test with valid state file
	meta := metadataFromState(stateFile)
	if !meta.HasState {
		t.Error("expected HasState to be true")
	}
	if meta.ResourceCount != 2 {
		t.Errorf("expected ResourceCount=2, got %d", meta.ResourceCount)
	}
	if meta.TerraformVersion != "1.5.0" {
		t.Errorf("expected TerraformVersion='1.5.0', got %q", meta.TerraformVersion)
	}
	if meta.LastModified.IsZero() {
		t.Error("expected non-zero LastModified")
	}
}

func TestMetadataFromStateNonExistent(t *testing.T) {
	meta := metadataFromState("/nonexistent/path/terraform.tfstate")
	if meta.HasState {
		t.Error("expected HasState to be false for nonexistent file")
	}
}

func TestMetadataFromStateInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "metadata-test-invalid")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file with invalid JSON
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	if err := os.WriteFile(stateFile, []byte("not valid json"), 0o600); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	meta := metadataFromState(stateFile)
	// Should still have HasState true (file exists) but no parsed data
	if !meta.HasState {
		t.Error("expected HasState to be true even with invalid JSON")
	}
	if meta.ResourceCount != 0 {
		t.Errorf("expected ResourceCount=0 for invalid JSON, got %d", meta.ResourceCount)
	}
	if meta.TerraformVersion != "" {
		t.Errorf("expected empty TerraformVersion for invalid JSON, got %q", meta.TerraformVersion)
	}
}

func TestMetadataForWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "metadata-workspace-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create workspace state directory
	wsDir := filepath.Join(tmpDir, "terraform.tfstate.d", "dev")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Create state file in workspace
	stateFile := filepath.Join(wsDir, "terraform.tfstate")
	stateContent := `{
		"terraform_version": "1.4.0",
		"resources": [{"type": "aws_s3_bucket", "name": "data"}]
	}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0o600); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	meta := metadataForWorkspace(tmpDir, "dev")
	if !meta.HasState {
		t.Error("expected HasState to be true")
	}
	if meta.ResourceCount != 1 {
		t.Errorf("expected ResourceCount=1, got %d", meta.ResourceCount)
	}
}

func TestMetadataForWorkspaceDefault(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "metadata-workspace-default-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create default state file
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{"terraform_version": "1.3.0", "resources": []}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0o600); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	meta := metadataForWorkspace(tmpDir, "default")
	if !meta.HasState {
		t.Error("expected HasState to be true for default workspace")
	}
	if meta.TerraformVersion != "1.3.0" {
		t.Errorf("expected TerraformVersion='1.3.0', got %q", meta.TerraformVersion)
	}
}

func TestMetadataForWorkspaceEmptyInputs(t *testing.T) {
	// Empty baseDir
	meta := metadataForWorkspace("", "dev")
	if meta.HasState {
		t.Error("expected HasState to be false for empty baseDir")
	}

	// Empty workspace
	meta = metadataForWorkspace("/some/path", "")
	if meta.HasState {
		t.Error("expected HasState to be false for empty workspace")
	}

	// Whitespace only
	meta = metadataForWorkspace("   ", "dev")
	if meta.HasState {
		t.Error("expected HasState to be false for whitespace baseDir")
	}
}

func TestMetadataForFolder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "metadata-folder-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create state file in folder
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{
		"terraform_version": "1.6.0",
		"resources": [
			{"type": "null_resource", "name": "test"}
		]
	}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0o600); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	meta := metadataForFolder(tmpDir)
	if !meta.HasState {
		t.Error("expected HasState to be true")
	}
	if meta.ResourceCount != 1 {
		t.Errorf("expected ResourceCount=1, got %d", meta.ResourceCount)
	}
	if meta.TerraformVersion != "1.6.0" {
		t.Errorf("expected TerraformVersion='1.6.0', got %q", meta.TerraformVersion)
	}
}

func TestMetadataForFolderNoState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "metadata-folder-nostate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Don't create state file
	meta := metadataForFolder(tmpDir)
	if meta.HasState {
		t.Error("expected HasState to be false when no state file exists")
	}
}

func TestMetadataForFolderEmptyPath(t *testing.T) {
	meta := metadataForFolder("")
	if meta.HasState {
		t.Error("expected HasState to be false for empty path")
	}

	meta = metadataForFolder("   ")
	if meta.HasState {
		t.Error("expected HasState to be false for whitespace path")
	}
}

func TestWorkspaceStateFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wsstate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test empty inputs
	_, ok := workspaceStateFile("", "dev")
	if ok {
		t.Error("expected false for empty baseDir")
	}

	_, ok = workspaceStateFile(tmpDir, "")
	if ok {
		t.Error("expected false for empty workspace")
	}

	// Test whitespace inputs
	_, ok = workspaceStateFile("   ", "dev")
	if ok {
		t.Error("expected false for whitespace baseDir")
	}

	_, ok = workspaceStateFile(tmpDir, "   ")
	if ok {
		t.Error("expected false for whitespace workspace")
	}
}

func TestFolderStateFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "folderstate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test empty input
	_, ok := folderStateFile("")
	if ok {
		t.Error("expected false for empty path")
	}

	// Test whitespace input
	_, ok = folderStateFile("   ")
	if ok {
		t.Error("expected false for whitespace path")
	}

	// Test with no state file
	_, ok = folderStateFile(tmpDir)
	if ok {
		t.Error("expected false when no state file exists")
	}

	// Create state file and test again
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	if err := os.WriteFile(stateFile, []byte("{}"), 0o600); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	path, ok := folderStateFile(tmpDir)
	if !ok {
		t.Error("expected true when state file exists")
	}
	if path != stateFile {
		t.Errorf("expected path %q, got %q", stateFile, path)
	}
}
