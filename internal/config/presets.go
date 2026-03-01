package config

// EnvironmentPreset stores per-environment defaults and flags.
type EnvironmentPreset struct {
	Name        string   `yaml:"name"`
	Environment string   `yaml:"environment,omitempty"`
	WorkDir     string   `yaml:"workdir,omitempty"`
	Theme       string   `yaml:"theme,omitempty"`
	Flags       []string `yaml:"flags,omitempty"`
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
