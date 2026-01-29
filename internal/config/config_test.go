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

	expanded, err := expandPath("~/.config/lazytf/config.yaml")
	if err != nil {
		t.Fatalf("expand path: %v", err)
	}
	expected := filepath.Join(home, ".config", "lazytf", "config.yaml")
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
	expected := filepath.Join(home, ".config", "lazytf", "config.yaml")
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
	expected := filepath.Join(home, ".config", "lazytf", "config.yaml")
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
		History:   HistoryConfig{Path: "~/.config/lazytf/history.db"},
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

func TestProjectOverrideFor(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		cfg := Config{
			ProjectOverrides: map[string]*ProjectConfig{
				"/some/path": {Theme: "dark"},
			},
		}
		if cfg.ProjectOverrideFor("") != nil {
			t.Error("expected nil for empty path")
		}
	})

	t.Run("no overrides", func(t *testing.T) {
		cfg := Config{}
		if cfg.ProjectOverrideFor("/some/path") != nil {
			t.Error("expected nil when no overrides configured")
		}
	})

	t.Run("matching path", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := Config{
			ProjectOverrides: map[string]*ProjectConfig{
				tmpDir: {Theme: "nord", Flags: []string{"-no-color"}},
			},
		}
		override := cfg.ProjectOverrideFor(tmpDir)
		if override == nil {
			t.Fatal("expected override for matching path")
		}
		if override.Theme != "nord" {
			t.Errorf("expected theme 'nord', got %q", override.Theme)
		}
	})

	t.Run("non-matching path", func(t *testing.T) {
		cfg := Config{
			ProjectOverrides: map[string]*ProjectConfig{
				"/project/a": {Theme: "dark"},
			},
		}
		if cfg.ProjectOverrideFor("/project/b") != nil {
			t.Error("expected nil for non-matching path")
		}
	})

	t.Run("nil override value", func(t *testing.T) {
		cfg := Config{
			ProjectOverrides: map[string]*ProjectConfig{
				"/some/path": nil,
			},
		}
		if cfg.ProjectOverrideFor("/some/path") != nil {
			t.Error("expected nil for nil override")
		}
	})

	t.Run("empty key", func(t *testing.T) {
		cfg := Config{
			ProjectOverrides: map[string]*ProjectConfig{
				"": {Theme: "dark"},
			},
		}
		if cfg.ProjectOverrideFor("/any/path") != nil {
			t.Error("expected nil when key is empty")
		}
	})

	t.Run("tilde expansion", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		projectDir := filepath.Join(home, "myproject")
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatal(err)
		}

		cfg := Config{
			ProjectOverrides: map[string]*ProjectConfig{
				"~/myproject": {Theme: "mocha"},
			},
		}
		override := cfg.ProjectOverrideFor(projectDir)
		if override == nil {
			t.Fatal("expected override for tilde-expanded path")
		}
		if override.Theme != "mocha" {
			t.Errorf("expected theme 'mocha', got %q", override.Theme)
		}
	})
}

func TestPresetByName(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		cfg := Config{
			Presets: []EnvironmentPreset{
				{Name: "dev"},
			},
		}
		preset, found := cfg.PresetByName("")
		if found || preset != nil {
			t.Error("expected no match for empty name")
		}
	})

	t.Run("no presets", func(t *testing.T) {
		cfg := Config{}
		preset, found := cfg.PresetByName("dev")
		if found || preset != nil {
			t.Error("expected no match when no presets configured")
		}
	})

	t.Run("matching preset", func(t *testing.T) {
		cfg := Config{
			Presets: []EnvironmentPreset{
				{Name: "dev", Environment: "development", Theme: "dark"},
				{Name: "prod", Environment: "production", Theme: "light"},
			},
		}
		preset, found := cfg.PresetByName("prod")
		if !found {
			t.Fatal("expected to find preset")
		}
		if preset.Environment != "production" {
			t.Errorf("expected environment 'production', got %q", preset.Environment)
		}
		if preset.Theme != "light" {
			t.Errorf("expected theme 'light', got %q", preset.Theme)
		}
	})

	t.Run("non-matching preset", func(t *testing.T) {
		cfg := Config{
			Presets: []EnvironmentPreset{
				{Name: "dev"},
				{Name: "staging"},
			},
		}
		preset, found := cfg.PresetByName("prod")
		if found || preset != nil {
			t.Error("expected no match for non-existent preset")
		}
	})

	t.Run("first match wins", func(t *testing.T) {
		cfg := Config{
			Presets: []EnvironmentPreset{
				{Name: "test", Environment: "first"},
				{Name: "test", Environment: "second"},
			},
		}
		preset, found := cfg.PresetByName("test")
		if !found {
			t.Fatal("expected to find preset")
		}
		if preset.Environment != "first" {
			t.Errorf("expected first match, got environment %q", preset.Environment)
		}
	})
}

func TestExpandConfigPathsWithPresets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		Presets: []EnvironmentPreset{
			{Name: "dev", WorkDir: "~/projects/dev"},
			{Name: "empty", WorkDir: ""},
			{Name: "spaces", WorkDir: "   "},
		},
	}

	err := expandConfigPaths(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(home, "projects", "dev")
	if cfg.Presets[0].WorkDir != expected {
		t.Errorf("expected WorkDir %q, got %q", expected, cfg.Presets[0].WorkDir)
	}

	// Empty and whitespace-only should be unchanged
	if cfg.Presets[1].WorkDir != "" {
		t.Errorf("expected empty WorkDir unchanged")
	}
	if cfg.Presets[2].WorkDir != "   " {
		t.Errorf("expected whitespace WorkDir unchanged")
	}
}

func TestExpandConfigPathsWithProjectOverrides(t *testing.T) {
	cfg := &Config{
		ProjectOverrides: map[string]*ProjectConfig{
			"/some/path": {Theme: "dark"},
			"/other":     nil,
		},
	}

	err := expandConfigPaths(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Just make sure it doesn't panic with nil project
	if cfg.ProjectOverrides["/some/path"].Theme != "dark" {
		t.Errorf("expected theme unchanged")
	}
}

func TestResolveXDGConfigPath(t *testing.T) {
	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		xdgPath := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdgPath)

		path, err := resolveXDGConfigPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(xdgPath, "lazytf", "config.yaml")
		if path != expected {
			t.Errorf("expected %q, got %q", expected, path)
		}
	})

	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_CONFIG_HOME", "")

		path, err := resolveXDGConfigPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(home, ".config", "lazytf", "config.yaml")
		if path != expected {
			t.Errorf("expected %q, got %q", expected, path)
		}
	})
}

func TestResolvePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LAZYTF_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	t.Run("default resolution", func(t *testing.T) {
		result, err := ResolvePath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty path")
		}
	})

	t.Run("with LAZYTF_CONFIG env", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "custom.yaml")
		t.Setenv("LAZYTF_CONFIG", configPath)
		result, err := ResolvePath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != configPath {
			t.Errorf("expected %q, got %q", configPath, result)
		}
	})

	t.Run("finds existing config", func(t *testing.T) {
		configDir := filepath.Join(home, ".config", "lazytf")
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			t.Fatal(err)
		}
		configPath := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte("version: 1"), 0o600); err != nil {
			t.Fatal(err)
		}

		t.Setenv("LAZYTF_CONFIG", "")
		result, err := ResolvePath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == "" {
			t.Error("expected to find existing config")
		}
	})
}

func TestValidateMoreCases(t *testing.T) {
	t.Run("valid split ratio", func(t *testing.T) {
		cfg := &Config{
			UI: UIConfig{SplitRatio: 0.5},
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("expected valid config, got error: %v", err)
		}
	})

	t.Run("invalid split ratio negative", func(t *testing.T) {
		cfg := &Config{
			UI: UIConfig{SplitRatio: -0.1},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for negative split ratio")
		}
	})

	t.Run("invalid split ratio too high", func(t *testing.T) {
		cfg := &Config{
			UI: UIConfig{SplitRatio: 1.5},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for split ratio above 1")
		}
	})

	t.Run("negative terraform timeout", func(t *testing.T) {
		cfg := &Config{
			Terraform: TerraformConfig{Timeout: -1},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for negative timeout")
		}
	})

	t.Run("negative parallelism", func(t *testing.T) {
		cfg := &Config{
			Terraform: TerraformConfig{Parallelism: -1},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for negative parallelism")
		}
	})
}
