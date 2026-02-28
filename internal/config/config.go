// Package config manages application configuration with atomic writes,
// file locking, and automatic migration. Configuration files are stored
// in YAML format at ~/.config/lazytf/config.yaml by default.
//
// The Manager type provides thread-safe config operations using file locks
// to prevent concurrent modifications. All writes are atomic using a
// temp-file-and-rename pattern to prevent corruption on crash or power loss.
//
//go:generate go run ../../scripts/gen-config-schema.go
//go:generate go run ../../scripts/gen-nix-config-options/main.go
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const currentVersion = 1

const defaultSchemaComment = "# yaml-language-server: $schema=https://raw.githubusercontent.com/ushiradineth/lazytf/main/internal/config/config.schema.json\n"

// Config defines the user configuration file schema.
type Config struct {
	Version          int                       `yaml:"version"`
	General          GeneralConfig             `yaml:"general,omitempty"`
	Theme            ThemeConfig               `yaml:"theme,omitempty"`
	Terraform        TerraformConfig           `yaml:"terraform,omitempty"`
	History          HistoryConfig             `yaml:"history,omitempty"`
	Presets          []EnvironmentPreset       `yaml:"presets,omitempty"`
	ProjectOverrides map[string]*ProjectConfig `yaml:"project_overrides,omitempty"`
}

// GeneralConfig groups general preferences.
type GeneralConfig struct {
	DefaultEnvironment string `yaml:"default_environment,omitempty"`
}

// ThemeConfig holds theme selection settings.
type ThemeConfig struct {
	Name string `yaml:"name,omitempty"`
}

// TerraformConfig holds terraform-specific settings.
type TerraformConfig struct {
	DefaultFlags []string      `yaml:"default_flags,omitempty"`
	Binary       string        `yaml:"binary,omitempty"`
	WorkingDir   string        `yaml:"working_dir,omitempty"`
	Timeout      time.Duration `yaml:"timeout,omitempty"`
	Parallelism  int           `yaml:"parallelism,omitempty"`
}

// HistoryConfig configures history logging.
type HistoryConfig struct {
	Enabled              bool   `yaml:"enabled,omitempty"`
	Level                string `yaml:"level,omitempty"`
	Path                 string `yaml:"path,omitempty"`
	CompressionThreshold int    `yaml:"compression_threshold,omitempty"`
}

// Manager loads and saves configuration files with locking.
type Manager struct {
	path string
}

// DefaultPath returns the default config path.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lazytf", "config.yaml"), nil
}

// ResolvePath resolves the config path using priority rules.
func ResolvePath() (string, error) {
	if env := strings.TrimSpace(os.Getenv("LAZYTF_CONFIG")); env != "" {
		return expandPath(env)
	}

	primary, err := DefaultPath()
	if err != nil {
		return "", err
	}
	xdg, err := resolveXDGConfigPath()
	if err != nil {
		return "", err
	}
	candidates := []string{
		primary,
		xdg,
		"/etc/lazytf/config.yaml",
	}

	for _, candidate := range candidates {
		expanded, err := expandPath(candidate)
		if err != nil {
			return "", err
		}
		if _, statErr := os.Stat(expanded); statErr == nil {
			return expanded, nil
		}
	}
	return expandPath(primary)
}

// NewManager creates a Manager for the provided path.
func NewManager(path string) (*Manager, error) {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = ResolvePath()
		if err != nil {
			return nil, err
		}
	}
	expanded, err := expandPath(path)
	if err != nil {
		return nil, err
	}
	return &Manager{path: expanded}, nil
}

// Path returns the resolved config path.
func (m *Manager) Path() string {
	if m == nil {
		return ""
	}
	return m.path
}

// Load reads configuration from disk, applying migrations and validation.
func (m *Manager) Load() (Config, error) {
	if m == nil {
		return Config{}, errors.New("config manager is nil")
	}
	lock, err := lockFile(m.lockPath())
	if err != nil {
		return Config{}, err
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			// Best effort unlock; load errors already take precedence.
			_ = err
		}
	}()

	data, err := os.ReadFile(m.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := DefaultConfig()
			if err := m.writeDefaultLocked(cfg); err != nil {
				return Config{}, fmt.Errorf("write default config to %s: %w", m.path, err)
			}
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config from %s: %w", m.path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config from %s: %w", m.path, err)
	}

	cfg, migrated, err := migrateConfig(cfg)
	if err != nil {
		return Config{}, err
	}

	if err := expandConfigPaths(&cfg); err != nil {
		return Config{}, fmt.Errorf("expand paths in config %s: %w", m.path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config from %s: %w", m.path, err)
	}

	if migrated {
		if err := m.backupLocked(); err != nil {
			return Config{}, fmt.Errorf("backup config %s before migration: %w", m.path, err)
		}
		if err := m.saveLocked(cfg); err != nil {
			return Config{}, fmt.Errorf("save migrated config to %s: %w", m.path, err)
		}
	}

	return cfg, nil
}

// Save validates and writes configuration to disk.
func (m *Manager) Save(cfg Config) error {
	if m == nil {
		return errors.New("config manager is nil")
	}
	lock, err := lockFile(m.lockPath())
	if err != nil {
		return err
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			// Best effort unlock; save errors already take precedence.
			_ = err
		}
	}()
	return m.saveLocked(cfg)
}

func (m *Manager) saveLocked(cfg Config) error {
	cfg = cfg.WithDefaults()
	cfg.Version = currentVersion
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := expandConfigPaths(&cfg); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return writeFileAtomic(m.path, data)
}

func (m *Manager) writeDefaultLocked(cfg Config) error {
	cfg = cfg.WithDefaults()
	cfg.Version = currentVersion
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := expandConfigPaths(&cfg); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	payload := append([]byte(defaultSchemaComment), data...)
	return writeFileAtomic(m.path, payload)
}

func (m *Manager) lockPath() string {
	return m.path + ".lock"
}

func (m *Manager) backupLocked() error {
	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		return nil
	}
	backupPath := m.path + ".bak"
	data, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("read config for backup: %w", err)
	}
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	return nil
}

// DefaultConfig returns a config populated with defaults.
func DefaultConfig() Config {
	cfg := Config{Version: currentVersion}
	cfg.History.Enabled = true
	return cfg.WithDefaults()
}

// WithDefaults ensures default values are set.
func (c Config) WithDefaults() Config {
	if c.Version == 0 {
		c.Version = currentVersion
	}
	if c.Theme.Name == "" {
		c.Theme.Name = "default"
	}
	if len(c.Terraform.DefaultFlags) == 0 {
		c.Terraform.DefaultFlags = []string{"-compact-warnings"}
	}
	if c.Terraform.Timeout == 0 {
		c.Terraform.Timeout = 10 * time.Minute
	}
	if c.Terraform.Parallelism == 0 {
		c.Terraform.Parallelism = 10
	}
	if c.History.Level == "" {
		c.History.Level = "standard"
	}
	if c.History.CompressionThreshold == 0 {
		c.History.CompressionThreshold = 64 * 1024 // 64KB default
	}
	return c
}

// Validate ensures the config is usable.
func (c Config) Validate() error {
	if c.Version > currentVersion {
		return fmt.Errorf("config version %d is newer than supported %d", c.Version, currentVersion)
	}
	if c.Version < 0 {
		return errors.New("config version cannot be negative")
	}
	if c.Terraform.Timeout < 0 {
		return errors.New("terraform timeout cannot be negative")
	}
	if c.Terraform.Parallelism < 0 {
		return errors.New("terraform parallelism cannot be negative")
	}
	return validateHistoryLevel(c.History.Level)
}

func validateHistoryLevel(level string) error {
	if level == "" {
		return nil
	}
	switch level {
	case "minimal", "standard", "verbose":
		return nil
	default:
		return fmt.Errorf("invalid history level: %s", level)
	}
}

func migrateConfig(cfg Config) (Config, bool, error) {
	cfg = cfg.WithDefaults()
	if cfg.Version == 0 {
		cfg.Version = currentVersion
		return cfg, true, nil
	}
	if cfg.Version < currentVersion {
		cfg.Version = currentVersion
		return cfg, true, nil
	}
	if cfg.Version > currentVersion {
		return cfg, false, fmt.Errorf("unsupported config version: %d", cfg.Version)
	}
	return cfg, false, nil
}

func expandConfigPaths(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	if cfg.Terraform.WorkingDir != "" {
		path, err := expandPath(cfg.Terraform.WorkingDir)
		if err != nil {
			return err
		}
		cfg.Terraform.WorkingDir = path
	}
	if cfg.History.Path != "" {
		path, err := expandPath(cfg.History.Path)
		if err != nil {
			return err
		}
		cfg.History.Path = path
	}
	for i := range cfg.Presets {
		if strings.TrimSpace(cfg.Presets[i].WorkDir) == "" {
			continue
		}
		path, err := expandPath(cfg.Presets[i].WorkDir)
		if err != nil {
			return err
		}
		cfg.Presets[i].WorkDir = path
	}
	for key, project := range cfg.ProjectOverrides {
		if project == nil {
			continue
		}
		cfg.ProjectOverrides[key] = project
	}
	return nil
}

func expandPath(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value, nil
	}
	expanded := os.ExpandEnv(trimmed)
	if strings.HasPrefix(expanded, "~") {
		if len(expanded) == 1 || expanded[1] == '/' || expanded[1] == '\\' {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~"))
		}
	}
	return expanded, nil
}

func resolveXDGConfigPath() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "lazytf", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lazytf", "config.yaml"), nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		if err := os.Remove(tmpPath); err != nil {
			// Best effort cleanup of temp file.
			_ = err
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp config: %w", err)
	}
	return nil
}
