// Package config manages application configuration with atomic writes,
// file locking, and automatic migration. Configuration files are stored
// in YAML format at ~/.config/lazytf/config.yaml by default.
//
// The Manager type provides thread-safe config operations using file locks
// to prevent concurrent modifications. All writes are atomic using a
// temp-file-and-rename pattern to prevent corruption on crash or power loss.
//
//go:generate go run ../../scripts/gen-config-schema.go
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ushiradineth/lazytf/internal/consts"
)

const currentVersion = 1

const (
	mainSchemaURL           = "https://raw.githubusercontent.com/ushiradineth/lazytf/main/internal/config/config.schema.json"
	releaseSchemaURLPattern = "https://raw.githubusercontent.com/ushiradineth/lazytf/v%s/internal/config/config.schema.json"
	schemaHintPrefix        = "# yaml-language-server: $schema="
	defaultMouseHintComment = "# mouse: true # optional override. By default lazytf enables mouse outside tmux and disables it inside tmux."
)

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// Config defines the user configuration file schema.
type Config struct {
	Version            int                       `yaml:"version" schema:"-"`
	DefaultEnvironment string                    `yaml:"default_environment,omitempty" description:"Default workspace or folder environment to select when lazytf starts."`
	Mouse              *bool                     `yaml:"mouse,omitempty" description:"Enable mouse navigation in the UI. By default lazytf enables mouse outside tmux and disables it inside tmux to respect tmux mouse settings. Set this explicitly to override that behavior."`
	Notification       *bool                     `yaml:"notification,omitempty" description:"Enable user notifications for important UI events. Set this explicitly to override the default behavior."`
	Warnings           WarningConfig             `yaml:"warnings,omitempty" description:"Controls warning visibility in the UI and runtime diagnostics."`
	Theme              ThemeConfig               `yaml:"theme,omitempty" description:"Theme settings for the lazytf UI."`
	Terraform          TerraformConfig           `yaml:"terraform,omitempty" description:"Terraform execution settings used by lazytf."`
	History            HistoryConfig             `yaml:"history,omitempty" description:"History storage and retention settings."`
	Presets            []EnvironmentPreset       `yaml:"presets,omitempty" description:"Named presets that bundle environment selection, workdir, theme, and default Terraform flags."`
	ProjectOverrides   map[string]*ProjectConfig `yaml:"project_overrides,omitempty" description:"Per-project overrides keyed by project path."`
}

// WarningConfig controls non-blocking warning output.
type WarningConfig struct {
	SuppressAll                bool `yaml:"suppress_all,omitempty" description:"Suppress all non-blocking warnings shown by lazytf."`
	SuppressSchemaHintMismatch bool `yaml:"suppress_schema_hint_mismatch,omitempty" description:"Suppress warnings when the config file schema hint does not match the running lazytf version."`
}

// ThemeConfig holds theme selection settings.
type ThemeConfig struct {
	Name string `yaml:"name,omitempty" description:"Built-in UI theme name."`
}

// TerraformConfig holds terraform-specific settings.
type TerraformConfig struct {
	DefaultFlags []string      `yaml:"default_flags,omitempty" description:"Default flags appended to Terraform commands run by lazytf."`
	Binary       string        `yaml:"binary,omitempty" description:"Path to the Terraform or OpenTofu binary to run."`
	WorkingDir   string        `yaml:"working_dir,omitempty" description:"Default working directory used when no folder or preset overrides it."`
	Timeout      time.Duration `yaml:"timeout,omitempty" description:"Maximum time allowed for Terraform commands before lazytf cancels them."`
	Parallelism  int           `yaml:"parallelism,omitempty" description:"Default Terraform parallelism value used when no explicit -parallelism flag is provided."`
}

// HistoryConfig configures history logging.
type HistoryConfig struct {
	Enabled              bool   `yaml:"enabled,omitempty" description:"Enable persistent operation history."`
	Level                string `yaml:"level,omitempty" description:"History detail level. Supported values are minimal, standard, and full."`
	Path                 string `yaml:"path,omitempty" description:"Path to the history database file."`
	CompressionThreshold int    `yaml:"compression_threshold,omitempty" description:"Compress stored output larger than this many bytes."`
}

// Manager loads and saves configuration files with locking.
type Manager struct {
	path string
}

// SchemaHintStatus reports schema hint comment state for a config file.
type SchemaHintStatus struct {
	ExpectedURL string
	ActualURL   string
	HasHint     bool
	Mismatch    bool
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
		xdg,
		primary,
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

	cfg, _, err = migrateConfig(cfg)
	if err != nil {
		return Config{}, err
	}

	if err := expandConfigPaths(&cfg); err != nil {
		return Config{}, fmt.Errorf("expand paths in config %s: %w", m.path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config from %s: %w", m.path, err)
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
	return writeFileAtomic(m.path, withSchemaHintComment(data))
}

func (m *Manager) writeDefaultLocked(cfg Config) error {
	cfg = cfg.WithDefaults()
	cfg.Version = currentVersion
	cfg = withBootstrapDefaults(cfg)
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
	return writeFileAtomic(m.path, withSchemaHintComment(withDefaultMouseHint(data)))
}

// SchemaHintStatus inspects the config file schema hint without mutating the file.
func (m *Manager) SchemaHintStatus() (SchemaHintStatus, error) {
	if m == nil {
		return SchemaHintStatus{}, errors.New("config manager is nil")
	}
	data, err := os.ReadFile(m.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SchemaHintStatus{}, nil
		}
		return SchemaHintStatus{}, fmt.Errorf("read config from %s: %w", m.path, err)
	}
	actual, hasHint := parseSchemaHintURL(data)
	expected := schemaURLForVersion(consts.Version)
	return SchemaHintStatus{
		ExpectedURL: expected,
		ActualURL:   actual,
		HasHint:     hasHint,
		Mismatch:    hasHint && actual != expected,
	}, nil
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
	if err := writeFileAtomic(backupPath, data); err != nil {
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

// DefaultBootstrapConfig returns defaults used when creating a new config file.
func DefaultBootstrapConfig() Config {
	return withBootstrapDefaults(DefaultConfig())
}

func withBootstrapDefaults(cfg Config) Config {
	if cfg.Notification == nil {
		notificationDisabled := false
		cfg.Notification = &notificationDisabled
	}
	return cfg
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

func schemaURLForVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	if semverPattern.MatchString(trimmed) {
		return fmt.Sprintf(releaseSchemaURLPattern, trimmed)
	}
	return mainSchemaURL
}

func schemaHintComment() string {
	return schemaHintPrefix + schemaURLForVersion(consts.Version) + "\n"
}

func withSchemaHintComment(data []byte) []byte {
	trimmed := bytes.TrimLeft(data, "\n")
	lines := strings.Split(string(trimmed), "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if strings.HasPrefix(firstLine, schemaHintPrefix) {
			trimmed = []byte(strings.Join(lines[1:], "\n"))
		}
	}
	return append([]byte(schemaHintComment()), trimmed...)
}

func withDefaultMouseHint(data []byte) []byte {
	content := strings.TrimRight(string(data), "\n")
	if strings.Contains(content, defaultMouseHintComment) {
		return []byte(content + "\n")
	}
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines)+1)
	inserted := false
	for _, line := range lines {
		out = append(out, line)
		if strings.HasPrefix(strings.TrimSpace(line), "notification:") {
			out = append(out, defaultMouseHintComment)
			inserted = true
		}
	}
	if !inserted {
		out = append(out, defaultMouseHintComment)
	}
	return []byte(strings.Join(out, "\n") + "\n")
}

func parseSchemaHintURL(data []byte) (string, bool) {
	content := strings.TrimPrefix(string(data), "\uFEFF")
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, schemaHintPrefix) {
			url := strings.TrimSpace(strings.TrimPrefix(trimmed, schemaHintPrefix))
			if url == "" {
				return "", false
			}
			return url, true
		}
		if !strings.HasPrefix(trimmed, "#") {
			break
		}
	}
	return "", false
}
