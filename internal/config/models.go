package config

import "path/filepath"

// ThemeModel defines a customizable color theme.
type ThemeModel struct {
	Name            string `yaml:"name,omitempty"`
	CreateColor     string `yaml:"create_color,omitempty"`
	UpdateColor     string `yaml:"update_color,omitempty"`
	DeleteColor     string `yaml:"delete_color,omitempty"`
	ReplaceColor    string `yaml:"replace_color,omitempty"`
	NoChangeColor   string `yaml:"no_change_color,omitempty"`
	BackgroundColor string `yaml:"background_color,omitempty"`
	ForegroundColor string `yaml:"foreground_color,omitempty"`
	BorderColor     string `yaml:"border_color,omitempty"`
	SelectedColor   string `yaml:"selected_color,omitempty"`
	DimmedColor     string `yaml:"dimmed_color,omitempty"`
	HighlightColor  string `yaml:"highlight_color,omitempty"`
}

// ProjectConfig overrides config for a specific project path.
type ProjectConfig struct {
	Theme      string   `yaml:"theme,omitempty"`
	Flags      []string `yaml:"flags,omitempty"`
	PresetName string   `yaml:"preset_name,omitempty"`
}

// ProjectOverrideFor returns the project override matching a path.
func (c Config) ProjectOverrideFor(path string) *ProjectConfig {
	if path == "" {
		return nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil
	}
	for key, override := range c.ProjectOverrides {
		if override == nil || key == "" {
			continue
		}
		expanded, err := expandPath(key)
		if err != nil {
			continue
		}
		absCandidate, err := filepath.Abs(expanded)
		if err != nil {
			continue
		}
		if filepath.Clean(absCandidate) == filepath.Clean(absPath) {
			return override
		}
	}
	return nil
}
