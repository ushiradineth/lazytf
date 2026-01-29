package environment

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func setupFakeTerraform(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake terraform script not supported on windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "terraform")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"workspace\" ] && [ \"$2\" = \"list\" ]; then\n" +
		"  echo \"  default\"\n" +
		"  echo \"* dev\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"workspace\" ] && [ \"$2\" = \"select\" ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	t.Setenv("PATH", dir)
}

func setupFakeTerraformError(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake terraform script not supported on windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "terraform")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"workspace\" ] && [ \"$2\" = \"list\" ]; then\n" +
		"  echo \"boom\" 1>&2\n" +
		"  exit 1\n" +
		"fi\n" +
		"if [ \"$1\" = \"workspace\" ] && [ \"$2\" = \"select\" ]; then\n" +
		"  echo \"bad\" 1>&2\n" +
		"  exit 1\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatalf("write terraform script: %v", err)
	}
	t.Setenv("PATH", dir)
}
