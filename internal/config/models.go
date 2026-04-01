package config

import (
	"path/filepath"
	"strings"
)

// ProjectConfig overrides config for a specific project path.
type ProjectConfig struct {
	Theme      string   `yaml:"theme,omitempty" description:"Built-in UI theme override for this project."`
	Flags      []string `yaml:"flags,omitempty" description:"Additional Terraform flags for this project."`
	PresetName string   `yaml:"preset_name,omitempty" description:"Preset name to apply for this project before project-specific overrides."`
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
	absPath = filepath.Clean(absPath)

	var bestMatch *ProjectConfig
	bestLen := -1
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
		absCandidate = filepath.Clean(absCandidate)
		rel, err := filepath.Rel(absCandidate, absPath)
		if err != nil {
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if l := len(absCandidate); l > bestLen {
			bestLen = l
			bestMatch = override
		}
	}
	return bestMatch
}
