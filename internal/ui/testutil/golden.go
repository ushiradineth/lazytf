package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// UpdateEnv is the environment variable name to enable golden file updates.
const UpdateEnv = "UPDATE_GOLDEN"

// Golden manages golden file comparisons for render testing.
type Golden struct {
	basePath string
}

// NewGolden creates a new Golden instance with the given base path.
// The base path should be relative to the testdata directory.
func NewGolden(basePath string) *Golden {
	return &Golden{
		basePath: basePath,
	}
}

// Assert compares the given output against a golden file.
// If UPDATE_GOLDEN=1 is set, updates the golden file instead.
func (g *Golden) Assert(t *testing.T, name string, got string) {
	t.Helper()

	goldenPath := g.path(name)

	if shouldUpdate() {
		g.update(t, goldenPath, got)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("golden file not found: %s\nRun with UPDATE_GOLDEN=1 to create it", goldenPath)
			t.Logf("Got:\n%s", got)
			return
		}
		t.Fatalf("failed to read golden file: %v", err)
	}

	expectedStr := normalizeGolden(string(expected))
	gotStr := normalizeGolden(got)

	if gotStr != expectedStr {
		t.Errorf("output mismatch with golden file: %s", goldenPath)
		t.Logf("Expected:\n%s", expectedStr)
		t.Logf("Got:\n%s", gotStr)
		t.Logf("Run with UPDATE_GOLDEN=1 to update the golden file")
	}
}

// Exists returns true if the golden file exists.
func (g *Golden) Exists(name string) bool {
	_, err := os.Stat(g.path(name))
	return err == nil
}

// path returns the full path to a golden file.
func (g *Golden) path(name string) string {
	filename := name + ".txt"
	return filepath.Join(g.basePath, filename)
}

// update writes the given content to the golden file.
func (g *Golden) update(t *testing.T, path string, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create golden directory: %v", err)
	}

	normalized := normalizeGolden(content)
	if err := os.WriteFile(path, []byte(normalized), 0o600); err != nil {
		t.Fatalf("failed to write golden file: %v", err)
	}

	t.Logf("Updated golden file: %s", path)
}

// shouldUpdate returns true if golden files should be updated.
func shouldUpdate() bool {
	return os.Getenv(UpdateEnv) == "1"
}

// normalizeGolden normalizes content for golden file comparison.
// - Trims trailing whitespace from each line.
// - Ensures consistent line endings.
// - Trims trailing newlines.
func normalizeGolden(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	result := strings.Join(lines, "\n")
	return strings.TrimRight(result, "\n")
}

// GoldenDir returns the standard golden file directory for a component.
func GoldenDir(component string) string {
	return filepath.Join("testdata", "golden", component)
}

// ComponentGolden creates a Golden instance for a specific component.
// The golden files are stored in testdata/golden/<component>/.
func ComponentGolden(component string) *Golden {
	return NewGolden(GoldenDir(component))
}
