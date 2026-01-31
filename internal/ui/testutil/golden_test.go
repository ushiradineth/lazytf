package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewGolden(t *testing.T) {
	g := NewGolden("testdata/golden/test")
	if g.basePath != "testdata/golden/test" {
		t.Errorf("expected basePath 'testdata/golden/test', got %q", g.basePath)
	}
}

func TestGoldenPath(t *testing.T) {
	g := NewGolden("testdata/golden/component")
	path := g.path("test_case")
	expected := filepath.Join("testdata/golden/component", "test_case.txt")
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}

func TestGoldenExists(t *testing.T) {
	// Create a temporary golden file
	tmpDir := t.TempDir()
	g := NewGolden(tmpDir)

	// Should not exist initially
	if g.Exists("nonexistent") {
		t.Error("expected nonexistent file to not exist")
	}

	// Create the file
	path := g.path("test")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Should exist now
	if !g.Exists("test") {
		t.Error("expected test file to exist")
	}
}

func TestNormalizeGolden(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims trailing spaces",
			input:    "line1   \nline2  ",
			expected: "line1\nline2",
		},
		{
			name:     "trims trailing newlines",
			input:    "content\n\n\n",
			expected: "content",
		},
		{
			name:     "handles Windows line endings",
			input:    "line1\r\nline2\r\n",
			expected: "line1\nline2",
		},
		{
			name:     "preserves internal whitespace",
			input:    "  indented\n    more indented",
			expected: "  indented\n    more indented",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeGolden(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeGolden(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGoldenDir(t *testing.T) {
	dir := GoldenDir("resource_list")
	expected := filepath.Join("testdata", "golden", "resource_list")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}
}

func TestComponentGolden(t *testing.T) {
	g := ComponentGolden("diff_viewer")
	expectedPath := filepath.Join("testdata", "golden", "diff_viewer")
	if g.basePath != expectedPath {
		t.Errorf("expected basePath %q, got %q", expectedPath, g.basePath)
	}
}

func TestGoldenAssertWithUpdate(t *testing.T) {
	// Skip if not in update mode
	if !shouldUpdate() {
		t.Skip("Run with UPDATE_GOLDEN=1 to test update mode")
	}

	tmpDir := t.TempDir()
	g := NewGolden(tmpDir)

	// Should create the file
	g.Assert(t, "new_test", "test content")

	// Verify file was created
	path := g.path("new_test")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("expected 'test content', got %q", string(content))
	}
}

func TestGoldenAssertMatch(t *testing.T) {
	tmpDir := t.TempDir()
	g := NewGolden(tmpDir)

	// Create a golden file
	path := g.path("matching")
	if err := os.WriteFile(path, []byte("expected content"), 0o644); err != nil {
		t.Fatalf("failed to create golden file: %v", err)
	}

	// Should pass when content matches
	g.Assert(t, "matching", "expected content")
}

func TestShouldUpdate(t *testing.T) {
	// Save and restore env
	original := os.Getenv(UpdateEnv)
	defer func() {
		if original == "" {
			os.Unsetenv(UpdateEnv)
		} else {
			os.Setenv(UpdateEnv, original)
		}
	}()

	os.Unsetenv(UpdateEnv)
	if shouldUpdate() {
		t.Error("expected shouldUpdate() = false when env not set")
	}

	os.Setenv(UpdateEnv, "0")
	if shouldUpdate() {
		t.Error("expected shouldUpdate() = false when env = '0'")
	}

	os.Setenv(UpdateEnv, "1")
	if !shouldUpdate() {
		t.Error("expected shouldUpdate() = true when env = '1'")
	}
}
