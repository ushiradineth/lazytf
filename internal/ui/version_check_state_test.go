package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ushiradineth/lazytf/internal/config"
)

func TestVersionCheckStateRoundTrip(t *testing.T) {
	manager, err := config.NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	m := NewExecutionModel(nil, ExecutionConfig{ConfigManager: manager})

	if err := m.markReleaseVersionNotified("v1.2.4"); err != nil {
		t.Fatalf("mark release notified: %v", err)
	}

	got, err := m.lastNotifiedReleaseVersion()
	if err != nil {
		t.Fatalf("read release state: %v", err)
	}
	if got != "v1.2.4" {
		t.Fatalf("expected v1.2.4, got %q", got)
	}
}

func TestVersionCheckStatePathUsesConfigManagerPath(t *testing.T) {
	manager, err := config.NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	m := NewExecutionModel(nil, ExecutionConfig{ConfigManager: manager})

	path, err := m.versionCheckStatePath()
	if err != nil {
		t.Fatalf("resolve state path: %v", err)
	}
	want := filepath.Join(filepath.Dir(manager.Path()), updateCheckStateFileName)
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestVersionCheckStateMissingFileReturnsEmpty(t *testing.T) {
	manager, err := config.NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	m := NewExecutionModel(nil, ExecutionConfig{ConfigManager: manager})

	got, err := m.lastNotifiedReleaseVersion()
	if err != nil {
		t.Fatalf("expected nil error for missing state file, got %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty state, got %q", got)
	}
}

func TestWasReleaseVersionNotifiedSemverEquivalent(t *testing.T) {
	manager, err := config.NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	m := NewExecutionModel(nil, ExecutionConfig{ConfigManager: manager})

	if err := m.markReleaseVersionNotified("v1.2.4"); err != nil {
		t.Fatalf("mark release notified: %v", err)
	}
	alreadyNotified, err := m.wasReleaseVersionNotified("1.2.4")
	if err != nil {
		t.Fatalf("check notified release: %v", err)
	}
	if !alreadyNotified {
		t.Fatal("expected equivalent semver release to be treated as already notified")
	}
	alreadyNotified, err = m.wasReleaseVersionNotified("v1.2.5")
	if err != nil {
		t.Fatalf("check newer release: %v", err)
	}
	if alreadyNotified {
		t.Fatal("did not expect newer release to be treated as already notified")
	}
}

func TestLoadVersionCheckStateInvalidJSON(t *testing.T) {
	manager, err := config.NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	m := NewExecutionModel(nil, ExecutionConfig{ConfigManager: manager})

	statePath, err := m.versionCheckStatePath()
	if err != nil {
		t.Fatalf("resolve state path: %v", err)
	}
	if err := os.WriteFile(statePath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid state file: %v", err)
	}

	if _, err := m.lastNotifiedReleaseVersion(); err == nil {
		t.Fatal("expected decode error for invalid state JSON")
	}
}
