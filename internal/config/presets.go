package config

// EnvironmentPreset stores per-environment defaults and flags.
type EnvironmentPreset struct {
	Name        string   `yaml:"name" description:"Preset name used with --preset."`
	Environment string   `yaml:"environment,omitempty" description:"Workspace or folder environment selected when this preset is used."`
	WorkDir     string   `yaml:"workdir,omitempty" description:"Working directory to use when this preset is selected."`
	Theme       string   `yaml:"theme,omitempty" description:"Built-in UI theme to apply when this preset is selected."`
	Flags       []string `yaml:"flags,omitempty" description:"Additional Terraform flags appended when this preset is selected."`
}

// PresetByName returns the first preset matching the name.
func (c Config) PresetByName(name string) (*EnvironmentPreset, bool) {
	if name == "" {
		return nil, false
	}
	for i := range c.Presets {
		if c.Presets[i].Name == name {
			return &c.Presets[i], true
		}
	}
	return nil, false
}
