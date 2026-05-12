package config

import (
	"path/filepath"
	"strings"
)

// ProjectConfig overrides config for a specific project path.
type ProjectConfig struct {
	Theme      string               `yaml:"theme,omitempty" description:"Built-in UI theme override for this project."`
	Flags      []string             `yaml:"flags,omitempty" description:"Additional Terraform flags for this project."`
	PresetName string               `yaml:"preset_name,omitempty" description:"Preset name to apply for this project."`
	Binary     string               `yaml:"binary,omitempty" description:"Path to the Terraform or OpenTofu binary to run for this project."`
	History    ProjectHistoryConfig `yaml:"history,omitempty" description:"History settings override for this project."`
}

// ProjectHistoryConfig overrides history settings for a specific project path.
type ProjectHistoryConfig struct {
	Enabled              *bool  `yaml:"enabled,omitempty" description:"Enable persistent operation history for this project. Omit to inherit the global history setting."`
	Level                string `yaml:"level,omitempty" description:"History detail level for this project. Supported values are minimal, standard, and full. verbose is accepted as a legacy alias for full."`
	Path                 string `yaml:"path,omitempty" description:"Path to the history database file for this project."`
	CompressionThreshold int    `yaml:"compression_threshold,omitempty" description:"Compress stored output larger than this many bytes for this project."`
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
