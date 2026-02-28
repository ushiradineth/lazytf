package config

import "path/filepath"

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
