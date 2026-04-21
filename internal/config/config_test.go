package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ushiradineth/lazytf/internal/consts"
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

func TestSchemaURLForVersion(t *testing.T) {
	if got, want := schemaURLForVersion("1.2.3"), "https://raw.githubusercontent.com/ushiradineth/lazytf/v1.2.3/internal/config/config.schema.json"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if got := schemaURLForVersion("dev"); got != mainSchemaURL {
		t.Fatalf("expected main schema URL, got %q", got)
	}
	if got := schemaURLForVersion("1.2.3-rc1"); got != mainSchemaURL {
		t.Fatalf("expected prerelease to fall back to main URL, got %q", got)
	}
}

func TestLoadDoesNotRewriteExistingConfigForMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := schemaHintPrefix + "https://raw.githubusercontent.com/ushiradineth/lazytf/v1.1.1/internal/config/config.schema.json\nversion: 0\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
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
		t.Fatalf("expected migrated in-memory version %d, got %d", currentVersion, cfg.Version)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after load: %v", err)
	}
	if string(after) != original {
		t.Fatalf("expected existing config to remain unchanged on load\nwant:\n%s\ngot:\n%s", original, string(after))
	}
}

func TestSaveWritesSingleSchemaHintComment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if err := manager.Save(DefaultConfig()); err != nil {
		t.Fatalf("save config: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, schemaHintPrefix) {
		t.Fatalf("expected config to start with schema hint comment, got %q", content)
	}
	if got := strings.Count(content, schemaHintPrefix); got != 1 {
		t.Fatalf("expected single schema hint comment, got %d", got)
	}
}

func TestWriteDefaultLockedAddsExplicitNotificationAndMouseHint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if err := manager.writeDefaultLocked(DefaultConfig()); err != nil {
		t.Fatalf("write default config: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, schemaHintPrefix) {
		t.Fatalf("expected schema hint prefix in bootstrap config")
	}
	if !strings.Contains(content, "notification: false") {
		t.Fatalf("expected explicit notification: false in bootstrap config")
	}
	if !strings.Contains(content, defaultMouseHintComment) {
		t.Fatalf("expected mouse guidance comment in bootstrap config")
	}
}

func TestSchemaHintStatusDetectsMismatch(t *testing.T) {
	oldVersion := consts.Version
	consts.Version = "1.2.3"
	t.Cleanup(func() {
		consts.Version = oldVersion
	})

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := schemaHintPrefix + "https://raw.githubusercontent.com/ushiradineth/lazytf/v1.1.1/internal/config/config.schema.json\nversion: 1\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	status, err := manager.SchemaHintStatus()
	if err != nil {
		t.Fatalf("schema hint status: %v", err)
	}
	if !status.HasHint {
		t.Fatalf("expected schema hint to be detected")
	}
	if !status.Mismatch {
		t.Fatalf("expected mismatch status")
	}
	if status.ExpectedURL != schemaURLForVersion(consts.Version) {
		t.Fatalf("unexpected expected URL %q", status.ExpectedURL)
	}
}

func TestSchemaHintStatusWithoutHint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("version: 1\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	status, err := manager.SchemaHintStatus()
	if err != nil {
		t.Fatalf("schema hint status: %v", err)
	}
	if status.HasHint {
		t.Fatalf("expected no schema hint")
	}
	if status.Mismatch {
		t.Fatalf("expected no mismatch without hint")
	}
}

func TestSchemaHintStatusDetectsHintAfterLeadingComment(t *testing.T) {
	oldVersion := consts.Version
	consts.Version = "1.2.3"
	t.Cleanup(func() {
		consts.Version = oldVersion
	})

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "# managed manually\n# yaml-language-server: $schema=https://raw.githubusercontent.com/ushiradineth/lazytf/v1.1.1/internal/config/config.schema.json\nversion: 1\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	status, err := manager.SchemaHintStatus()
	if err != nil {
		t.Fatalf("schema hint status: %v", err)
	}
	if !status.HasHint || !status.Mismatch {
		t.Fatalf("expected mismatch status for leading-comment schema hint")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	notificationEnabled := true
	cfg := Config{
		Theme: ThemeConfig{Name: "Nord"},
		History: HistoryConfig{
			Enabled: true,
			Level:   "minimal",
		},
		Notification: &notificationEnabled,
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
	if loaded.Notification == nil || !*loaded.Notification {
		t.Fatalf("expected notifications to round-trip")
	}
}

func TestSaveAndLoadConfigMouseEnabledRoundTrip(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	mouseEnabled := false
	cfg := Config{
		Mouse: &mouseEnabled,
	}
	if err := manager.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.Mouse == nil {
		t.Fatal("expected mouse_enabled to be present after round-trip")
	}
	if *loaded.Mouse {
		t.Fatal("expected mouse_enabled=false to round-trip")
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
	expected, err := ResolvePath()
	if err != nil {
		t.Fatalf("resolve path: %v", err)
	}
	if manager.Path() != expected {
		t.Fatalf("expected %s, got %s", expected, manager.Path())
	}
}

func TestGeneratedSchemaIncludesMouseDescription(t *testing.T) {
	type schemaNode struct {
		Description string                 `json:"description"`
		Properties  map[string]*schemaNode `json:"properties"`
	}

	data, err := os.ReadFile("config.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	var root schemaNode
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	mouse := root.Properties["mouse"]
	if mouse == nil {
		t.Fatal("expected mouse property in schema")
	}
	if !strings.Contains(mouse.Description, "tmux") {
		t.Fatalf("expected mouse description to mention tmux, got %q", mouse.Description)
	}
}

func TestGeneratedSchemaIncludesPresetThemeEnum(t *testing.T) {
	type schemaNode struct {
		Enum       []string               `json:"enum"`
		Properties map[string]*schemaNode `json:"properties"`
		Items      *schemaNode            `json:"items"`
	}

	data, err := os.ReadFile("config.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	var root schemaNode
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	presets := root.Properties["presets"]
	if presets == nil || presets.Items == nil {
		t.Fatal("expected presets items schema")
	}
	theme := presets.Items.Properties["theme"]
	if theme == nil {
		t.Fatal("expected preset theme schema")
	}
	if len(theme.Enum) == 0 {
		t.Fatal("expected preset theme enum values")
	}
	if !contains(theme.Enum, "monochrome") {
		t.Fatalf("expected monochrome preset theme suggestion, got %v", theme.Enum)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
	notificationEnabled := true
	if err := (Config{
		Notification: &notificationEnabled,
	}).Validate(); err != nil {
		t.Fatalf("expected enabled desktop notifications to validate, got %v", err)
	}
}

func TestDefaultConfigNotificationsDisabled(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Notification != nil && *cfg.Notification {
		t.Fatalf("expected notifications to be disabled by default")
	}
}

func TestValidateNotificationsAllowsDisabled(t *testing.T) {
	notificationEnabled := false
	cfg := Config{
		Notification: &notificationEnabled,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected disabled desktop notifications to validate, got %v", err)
	}
}

func TestValidateNotificationsAllowsEnabled(t *testing.T) {
	notificationEnabled := true
	cfg := Config{
		Notification: &notificationEnabled,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected enabled desktop notifications to validate, got %v", err)
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

	t.Run("parent path override applies", func(t *testing.T) {
		base := t.TempDir()
		child := filepath.Join(base, "apps", "service")
		if err := os.MkdirAll(child, 0o755); err != nil {
			t.Fatal(err)
		}

		cfg := Config{
			ProjectOverrides: map[string]*ProjectConfig{
				base: {Theme: "parent"},
			},
		}

		override := cfg.ProjectOverrideFor(child)
		if override == nil {
			t.Fatal("expected parent override for nested path")
		}
		if override.Theme != "parent" {
			t.Fatalf("expected parent theme, got %q", override.Theme)
		}
	})

	t.Run("most specific parent wins", func(t *testing.T) {
		base := t.TempDir()
		level1 := filepath.Join(base, "apps")
		level2 := filepath.Join(level1, "service")
		if err := os.MkdirAll(level2, 0o755); err != nil {
			t.Fatal(err)
		}

		cfg := Config{
			ProjectOverrides: map[string]*ProjectConfig{
				base:   {Theme: "base"},
				level1: {Theme: "apps"},
				level2: {Theme: "service"},
			},
		}

		override := cfg.ProjectOverrideFor(filepath.Join(level2, "worker"))
		if override == nil {
			t.Fatal("expected override for nested path")
		}
		if override.Theme != "service" {
			t.Fatalf("expected most specific theme, got %q", override.Theme)
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

	t.Run("prefers XDG path when both exist", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)
		t.Setenv("LAZYTF_CONFIG", "")

		defaultDir := filepath.Join(home, ".config", "lazytf")
		if err := os.MkdirAll(defaultDir, 0o755); err != nil {
			t.Fatal(err)
		}
		defaultPath := filepath.Join(defaultDir, "config.yaml")
		if err := os.WriteFile(defaultPath, []byte("version: 1\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		xdgDir := filepath.Join(xdg, "lazytf")
		if err := os.MkdirAll(xdgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		xdgPath := filepath.Join(xdgDir, "config.yaml")
		if err := os.WriteFile(xdgPath, []byte("version: 1\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		result, err := ResolvePath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != xdgPath {
			t.Fatalf("expected XDG path %q, got %q", xdgPath, result)
		}
	})
}

func TestValidateMoreCases(t *testing.T) {
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

func TestWriteFileAtomicSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-atomic.yaml")

	data := []byte("test: data\n")
	if err := writeFileAtomic(path, data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file was written
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !bytes.Equal(content, data) {
		t.Errorf("expected %q, got %q", string(data), string(content))
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected mode 0600, got %o", info.Mode().Perm())
	}
}

func TestDefaultConfigHasDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != currentVersion {
		t.Errorf("expected version %d, got %d", currentVersion, cfg.Version)
	}
	if !cfg.History.Enabled {
		t.Error("expected history enabled by default")
	}
	if cfg.Theme.Name == "" {
		t.Error("expected theme name to have default")
	}
}

func TestLoadCorruptedConfigFails(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	// Write corrupted YAML
	if err := os.WriteFile(path, []byte("!!!invalid yaml: ["), 0o600); err != nil {
		t.Fatalf("failed to write corrupted config: %v", err)
	}

	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	_, err = manager.Load()
	if err == nil {
		t.Error("expected error for corrupted config")
	}
}

func TestValidateVersionEdgeCases(t *testing.T) {
	t.Run("negative version", func(t *testing.T) {
		cfg := &Config{Version: -1}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for negative version")
		}
	})

	t.Run("future version", func(t *testing.T) {
		cfg := &Config{Version: currentVersion + 1}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for future version")
		}
	})

	t.Run("current version", func(t *testing.T) {
		cfg := &Config{Version: currentVersion}
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestNewManagerEmptyPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LAZYTF_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	manager, err := NewManager("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manager.Path() == "" {
		t.Error("expected non-empty path")
	}
}

func TestWriteDefaultLockedSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	cfg := DefaultConfig()
	if err := manager.writeDefaultLocked(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify file was written
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestExpandConfigPathsWithHistoryPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get user home dir")
	}

	cfg := &Config{
		History: HistoryConfig{
			Path: "~/.lazytf/history.db",
		},
	}

	if err := expandConfigPaths(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedPath := filepath.Join(homeDir, ".lazytf", "history.db")
	if cfg.History.Path != expectedPath {
		t.Errorf("expected %q, got %q", expectedPath, cfg.History.Path)
	}
}

func TestExpandPathWithAbsolutePathReturnsValue(t *testing.T) {
	absPath := "/absolute/path/config.yaml"
	expanded, err := expandPath(absPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expanded != absPath {
		t.Errorf("expected %q, got %q", absPath, expanded)
	}
}

func TestExpandPathWithTildeExpands(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get user home dir")
	}

	tildeDir := "~/config/file.yaml"
	expanded, err := expandPath(tildeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(homeDir, "config", "file.yaml")
	if expanded != expected {
		t.Errorf("expected %q, got %q", expected, expanded)
	}
}

func TestMigrateConfigFromVersion0(t *testing.T) {
	cfg := Config{Version: 0}
	migrated, _, err := migrateConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Version 0 gets set to currentVersion by WithDefaults first
	if migrated.Version != currentVersion {
		t.Errorf("expected version %d, got %d", currentVersion, migrated.Version)
	}
}

func TestMigrateConfigAlreadyCurrentNoChange(t *testing.T) {
	cfg := Config{Version: currentVersion}
	_, changed, err := migrateConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Error("expected no migration for current version")
	}
}

func TestValidateHistoryLevelVariants(t *testing.T) {
	tests := []struct {
		level   string
		wantErr bool
	}{
		{"", false},
		{"minimal", false},
		{"standard", false},
		{"verbose", false},
		{"invalid", true},
		{"MINIMAL", true}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			err := validateHistoryLevel(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHistoryLevel(%q) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}
		})
	}
}

func TestLockFilePathValue(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	lockPath := manager.lockPath()
	expected := configPath + ".lock"
	if lockPath != expected {
		t.Errorf("expected %q, got %q", expected, lockPath)
	}
}

func TestWithDefaultsSetsExactDefaults(t *testing.T) {
	cfg := Config{}
	got := cfg.WithDefaults()

	if got.Theme.Name != "default" {
		t.Fatalf("expected default theme, got %q", got.Theme.Name)
	}
	if len(got.Terraform.DefaultFlags) != 1 || got.Terraform.DefaultFlags[0] != "-compact-warnings" {
		t.Fatalf("expected default terraform flags, got %#v", got.Terraform.DefaultFlags)
	}
	if got.Terraform.Timeout != 10*time.Minute {
		t.Fatalf("expected default timeout %v, got %v", 10*time.Minute, got.Terraform.Timeout)
	}
	if got.Terraform.Parallelism != 10 {
		t.Fatalf("expected default parallelism 10, got %d", got.Terraform.Parallelism)
	}
	if got.History.Level != "standard" {
		t.Fatalf("expected default history level standard, got %q", got.History.Level)
	}
	if got.History.CompressionThreshold != 64*1024 {
		t.Fatalf("expected default compression threshold 65536, got %d", got.History.CompressionThreshold)
	}
}

func TestExpandConfigPathsContinuesAfterEmptyPreset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		Presets: []EnvironmentPreset{
			{Name: "empty", WorkDir: ""},
			{Name: "target", WorkDir: "~/projects/dev"},
		},
	}

	if err := expandConfigPaths(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(home, "projects", "dev")
	if cfg.Presets[1].WorkDir != expected {
		t.Fatalf("expected expanded second preset workdir %q, got %q", expected, cfg.Presets[1].WorkDir)
	}
}

func TestExpandPathWithTildeBackslashExpands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	input := `~\projects\dev`
	output, err := expandPath(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == input {
		t.Fatalf("expected backslash tilde path to expand, got %q", output)
	}
	if !strings.HasPrefix(output, home) {
		t.Fatalf("expected expanded path to start with home %q, got %q", home, output)
	}
}
