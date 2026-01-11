package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadMissingConfigReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	cfg, err := manager.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config to be written: %v", err)
	}
	if cfg.Version != currentVersion {
		t.Fatalf("expected version %d, got %d", currentVersion, cfg.Version)
	}
	if cfg.History.Level == "" {
		t.Fatalf("expected default history level")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	cfg := Config{
		Theme: ThemeConfig{Name: "Nord"},
		History: HistoryConfig{
			Enabled: true,
			Level:   "minimal",
		},
	}
	if err := manager.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.Theme.Name != "Nord" {
		t.Fatalf("expected theme to round-trip")
	}
	if loaded.History.Level != "minimal" {
		t.Fatalf("expected history level to round-trip")
	}
}

func TestMigrateConfigVersion(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	data, err := yaml.Marshal(Config{Version: 0})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	cfg, err := manager.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Version != currentVersion {
		t.Fatalf("expected migrated version %d, got %d", currentVersion, cfg.Version)
	}
}

func TestExpandPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	expanded, err := expandPath("~/.config/tftui/config.yaml")
	if err != nil {
		t.Fatalf("expand path: %v", err)
	}
	expected := filepath.Join(home, ".config", "tftui", "config.yaml")
	if expanded != expected {
		t.Fatalf("expected %s, got %s", expected, expanded)
	}
}

func TestDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	expected := filepath.Join(home, ".config", "tftui", "config.yaml")
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestNewManagerEmptyPathUsesDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	manager, err := NewManager("")
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if manager.Path() == "" {
		t.Fatalf("expected manager path")
	}
	expected := filepath.Join(home, ".config", "tftui", "config.yaml")
	if manager.Path() != expected {
		t.Fatalf("expected %s, got %s", expected, manager.Path())
	}
}

func TestManagerPathNil(t *testing.T) {
	var manager *Manager
	if manager.Path() != "" {
		t.Fatalf("expected empty path for nil manager")
	}
}

func TestValidateConfigErrors(t *testing.T) {
	if err := (Config{Version: -1}).Validate(); err == nil {
		t.Fatalf("expected error for negative version")
	}
	if err := (Config{Version: currentVersion + 1}).Validate(); err == nil {
		t.Fatalf("expected error for newer version")
	}
	if err := (Config{History: HistoryConfig{Level: "bogus"}}).Validate(); err == nil {
		t.Fatalf("expected error for invalid history level")
	}
}

func TestMigrateConfigUnsupportedVersion(t *testing.T) {
	_, _, err := migrateConfig(Config{Version: currentVersion + 1})
	if err == nil {
		t.Fatalf("expected error for unsupported version")
	}
}

func TestExpandConfigPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CONFIG_TEST_DIR", filepath.Join(home, "workdir"))

	cfg := Config{
		Terraform: TerraformConfig{WorkingDir: "$CONFIG_TEST_DIR"},
		History:   HistoryConfig{Path: "~/.config/tftui/history.db"},
	}
	if err := expandConfigPaths(&cfg); err != nil {
		t.Fatalf("expand config paths: %v", err)
	}
	if !strings.Contains(cfg.Terraform.WorkingDir, home) {
		t.Fatalf("expected expanded working dir, got %s", cfg.Terraform.WorkingDir)
	}
	if !strings.Contains(cfg.History.Path, home) {
		t.Fatalf("expected expanded history path, got %s", cfg.History.Path)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := []byte("hello")
	if err := writeFileAtomic(path, data); err != nil {
		t.Fatalf("write file atomic: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("unexpected data: %s", got)
	}
}

func TestBackupLocked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if err := manager.backupLocked(); err != nil {
		t.Fatalf("backup: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
}

func TestSaveInvalidHistoryLevel(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	cfg := Config{
		History: HistoryConfig{Level: "nope"},
	}
	if err := manager.Save(cfg); err == nil {
		t.Fatalf("expected save error for invalid history level")
	}
}

func TestLockFileAndUnlock(t *testing.T) {
	lock, err := lockFile(filepath.Join(t.TempDir(), "config.lock"))
	if err != nil {
		t.Fatalf("lock file: %v", err)
	}
	if err := lock.Unlock(); err != nil {
		t.Fatalf("unlock: %v", err)
	}
}

func TestLoadInvalidYaml(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(":"), 0o600); err != nil {
		t.Fatalf("write invalid yaml: %v", err)
	}
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if _, err := manager.Load(); err == nil {
		t.Fatalf("expected load error")
	}
}

func TestLoadInvalidHistoryLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data, err := yaml.Marshal(Config{History: HistoryConfig{Level: "nope"}})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if _, err := manager.Load(); err == nil {
		t.Fatalf("expected history level validation error")
	}
}

func TestLockFileOpenError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o700)
	})
	_, err := lockFile(filepath.Join(dir, "config.lock"))
	if err == nil {
		t.Fatalf("expected open lock file error")
	}
}

func TestUnlockNilFile(t *testing.T) {
	lock := &fileLock{}
	if err := lock.Unlock(); err != nil {
		t.Fatalf("expected nil unlock error")
	}
}

func TestLockFilePlatformError(t *testing.T) {
	oldLock := lockFilePlatformFunc
	lockFilePlatformFunc = func(_ *os.File) error {
		return errors.New("lock failed")
	}
	t.Cleanup(func() {
		lockFilePlatformFunc = oldLock
	})
	if _, err := lockFile(filepath.Join(t.TempDir(), "config.lock")); err == nil {
		t.Fatalf("expected lock failure")
	}
}

func TestUnlockFilePlatformError(t *testing.T) {
	oldUnlock := unlockFilePlatformFunc
	unlockFilePlatformFunc = func(_ *os.File) error {
		return errors.New("unlock failed")
	}
	t.Cleanup(func() {
		unlockFilePlatformFunc = oldUnlock
	})

	lock, err := lockFile(filepath.Join(t.TempDir(), "config.lock"))
	if err != nil {
		t.Fatalf("lock file: %v", err)
	}
	if err := lock.Unlock(); err == nil {
		t.Fatalf("expected unlock error")
	}
}

func TestLoadUnsupportedVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data, err := yaml.Marshal(Config{Version: currentVersion + 1})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if _, err := manager.Load(); err == nil {
		t.Fatalf("expected load error for unsupported version")
	}
}

func TestSaveNilManager(t *testing.T) {
	var manager *Manager
	if err := manager.Save(Config{}); err == nil {
		t.Fatalf("expected error for nil manager")
	}
}

func TestLoadNilManager(t *testing.T) {
	var manager *Manager
	if _, err := manager.Load(); err == nil {
		t.Fatalf("expected error for nil manager")
	}
}

func TestExpandPathEmpty(t *testing.T) {
	if out, err := expandPath(" "); err != nil || out != " " {
		t.Fatalf("expected trimmed empty path")
	}
}

func TestExpandConfigPathsNil(t *testing.T) {
	if err := expandConfigPaths(nil); err != nil {
		t.Fatalf("expected nil error for nil config")
	}
}

func TestMigrateConfigCurrentVersion(t *testing.T) {
	_, migrated, err := migrateConfig(Config{Version: currentVersion})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if migrated {
		t.Fatalf("expected no migration")
	}
}

func TestMigrateConfigLegacyVersion(t *testing.T) {
	legacyVersion := currentVersion - 1
	if legacyVersion <= 0 {
		legacyVersion = -1
	}
	cfg, migrated, err := migrateConfig(Config{Version: legacyVersion})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !migrated || cfg.Version != currentVersion {
		t.Fatalf("expected migration to current version")
	}
}

func TestValidateHistoryLevelEmpty(t *testing.T) {
	if err := validateHistoryLevel(""); err != nil {
		t.Fatalf("expected empty history level to be valid")
	}
}

func TestWriteFileAtomicMissingDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "config.yaml")
	if err := writeFileAtomic(path, []byte("data")); err == nil {
		t.Fatalf("expected error for missing dir")
	}
}

func TestLockFilePlatformNil(t *testing.T) {
	if err := lockFilePlatform(nil); err == nil {
		t.Fatalf("expected error for nil lock file")
	}
	if err := unlockFilePlatform(nil); err != nil {
		t.Fatalf("expected nil error for nil unlock")
	}
}

func TestWriteFileAtomicRenameError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := writeFileAtomic(target, []byte("data")); err == nil {
		t.Fatalf("expected rename error for directory target")
	}
}

func TestLockFileCreateDirError(t *testing.T) {
	base := t.TempDir()
	if err := os.Chmod(base, 0o500); err != nil {
		t.Fatalf("chmod base: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(base, 0o700)
	})
	_, err := lockFile(filepath.Join(base, "subdir", "config.lock"))
	if err == nil {
		t.Fatalf("expected lock file error")
	}
}

func TestUnlockClosedFileError(t *testing.T) {
	lock, err := lockFile(filepath.Join(t.TempDir(), "config.lock"))
	if err != nil {
		t.Fatalf("lock file: %v", err)
	}
	if err := lock.file.Close(); err != nil {
		t.Fatalf("close lock file: %v", err)
	}
	if err := lock.Unlock(); err == nil {
		t.Fatalf("expected unlock error on closed file")
	}
}
