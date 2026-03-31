package ui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ushiradineth/lazytf/internal/config"
	"github.com/ushiradineth/lazytf/internal/ui/keybinds"
)

func TestHandleActionSelectThemeModalSavesTheme(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	manager, err := config.NewManager(configPath)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	cfg := config.DefaultConfig()
	m := NewExecutionModel(nil, ExecutionConfig{Config: &cfg, ConfigManager: manager})
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.toggleThemeModal()
	m.themeModal.SetSelectedIndex(themeIndex(t, "monochrome"))

	cmd := m.handleActionSelect(&keybinds.Context{ActiveModal: keybinds.ModalTheme})
	if cmd == nil {
		t.Fatal("expected toast command")
	}
	if m.modalState != ModalNone {
		t.Fatalf("expected theme modal to close, got %v", m.modalState)
	}
	if m.styles.Theme.Name != "monochrome" {
		t.Fatalf("expected monochrome theme, got %q", m.styles.Theme.Name)
	}
	if m.config == nil || m.config.Theme.Name != "monochrome" {
		t.Fatalf("expected model config theme to update, got %#v", m.config)
	}

	saved, err := manager.Load()
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if saved.Theme.Name != "monochrome" {
		t.Fatalf("expected saved theme monochrome, got %q", saved.Theme.Name)
	}
}

func TestHandleActionSelectThemeModalFallsBackWhenSaveFails(t *testing.T) {
	readOnlyDir := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(readOnlyDir, 0o500); err != nil {
		t.Fatalf("mkdir readonly dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(readOnlyDir, 0o700)
	})

	manager, err := config.NewManager(filepath.Join(readOnlyDir, "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	cfg := config.DefaultConfig()
	m := NewExecutionModel(nil, ExecutionConfig{Config: &cfg, ConfigManager: manager})
	m.width = 100
	m.height = 30
	m.updateLayout()
	m.toggleThemeModal()
	m.themeModal.SetSelectedIndex(themeIndex(t, "monochrome"))

	cmd := m.handleActionSelect(&keybinds.Context{ActiveModal: keybinds.ModalTheme})
	if cmd == nil {
		t.Fatal("expected toast command")
	}
	if m.modalState != ModalNone {
		t.Fatalf("expected theme modal to close, got %v", m.modalState)
	}
	if m.styles.Theme.Name != "monochrome" {
		t.Fatalf("expected monochrome theme, got %q", m.styles.Theme.Name)
	}
	if m.config == nil || m.config.Theme.Name != "monochrome" {
		t.Fatalf("expected in-memory config theme to update, got %#v", m.config)
	}
	if _, err := os.Stat(filepath.Join(readOnlyDir, "config.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected config file to remain unsaved, got err=%v", err)
	}
}

func themeIndex(t *testing.T, name string) int {
	t.Helper()
	for i, themeName := range availableThemes {
		if themeName == name {
			return i
		}
	}
	t.Fatalf("theme %q not found", name)
	return -1
}
