package ui

import (
	"testing"

	"github.com/ushiradineth/lazytf/internal/environment"
)

func TestEnvMatchesCurrent(t *testing.T) {
	tests := []struct {
		name    string
		env     environment.Environment
		current string
		want    bool
	}{
		{
			name:    "empty current uses IsCurrent flag - true",
			env:     environment.Environment{IsCurrent: true},
			current: "",
			want:    true,
		},
		{
			name:    "empty current uses IsCurrent flag - false",
			env:     environment.Environment{IsCurrent: false},
			current: "",
			want:    false,
		},
		{
			name: "workspace strategy - name matches",
			env: environment.Environment{
				Name:     "production",
				Strategy: environment.StrategyWorkspace,
			},
			current: "production",
			want:    true,
		},
		{
			name: "workspace strategy - name doesn't match",
			env: environment.Environment{
				Name:     "staging",
				Strategy: environment.StrategyWorkspace,
			},
			current: "production",
			want:    false,
		},
		{
			name: "folder strategy - path matches exactly",
			env: environment.Environment{
				Path:     "/path/to/env",
				Strategy: environment.StrategyFolder,
			},
			current: "/path/to/env",
			want:    true,
		},
		{
			name: "folder strategy - base name matches",
			env: environment.Environment{
				Path:     "/path/to/production",
				Strategy: environment.StrategyFolder,
			},
			current: "production",
			want:    true,
		},
		{
			name: "folder strategy - no match",
			env: environment.Environment{
				Path:     "/path/to/staging",
				Strategy: environment.StrategyFolder,
			},
			current: "production",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := envMatchesCurrent(tt.env, tt.current)
			if got != tt.want {
				t.Errorf("envMatchesCurrent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvSelectionValue(t *testing.T) {
	tests := []struct {
		name string
		env  environment.Environment
		want string
	}{
		{
			name: "folder strategy returns path",
			env: environment.Environment{
				Path:     "/path/to/env",
				Name:     "env",
				Strategy: environment.StrategyFolder,
			},
			want: "/path/to/env",
		},
		{
			name: "workspace strategy returns name",
			env: environment.Environment{
				Path:     "/some/path",
				Name:     "production",
				Strategy: environment.StrategyWorkspace,
			},
			want: "production",
		},
		{
			name: "unknown strategy returns name",
			env: environment.Environment{
				Path:     "/some/path",
				Name:     "myenv",
				Strategy: environment.StrategyUnknown,
			},
			want: "myenv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := envSelectionValue(tt.env)
			if got != tt.want {
				t.Errorf("envSelectionValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultWorkDir(t *testing.T) {
	tests := []struct {
		name    string
		workDir string
		want    string
	}{
		{"empty returns dot", "", "."},
		{"whitespace returns dot", "   ", "."},
		{"valid path returns path", "/path/to/dir", "/path/to/dir"},
		{"relative path returns path", "relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultWorkDir(tt.workDir)
			if got != tt.want {
				t.Errorf("defaultWorkDir(%q) = %v, want %v", tt.workDir, got, tt.want)
			}
		})
	}
}

func TestStrategyMatches(t *testing.T) {
	tests := []struct {
		name      string
		selected  environment.StrategyType
		candidate environment.StrategyType
		want      bool
	}{
		{"unknown matches anything", environment.StrategyUnknown, environment.StrategyWorkspace, true},
		{"mixed matches anything", environment.StrategyMixed, environment.StrategyFolder, true},
		{"workspace matches workspace", environment.StrategyWorkspace, environment.StrategyWorkspace, true},
		{"workspace doesn't match folder", environment.StrategyWorkspace, environment.StrategyFolder, false},
		{"folder matches folder", environment.StrategyFolder, environment.StrategyFolder, true},
		{"folder doesn't match workspace", environment.StrategyFolder, environment.StrategyWorkspace, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategyMatches(tt.selected, tt.candidate)
			if got != tt.want {
				t.Errorf("strategyMatches(%v, %v) = %v, want %v", tt.selected, tt.candidate, got, tt.want)
			}
		})
	}
}

func TestMatchCurrentFolder(t *testing.T) {
	tests := []struct {
		name        string
		folderPaths []string
		absWorkDir  string
		want        string
	}{
		{
			name:        "match found",
			folderPaths: []string{"/path/a", "/path/b", "/path/c"},
			absWorkDir:  "/path/b",
			want:        "/path/b",
		},
		{
			name:        "no match",
			folderPaths: []string{"/path/a", "/path/b"},
			absWorkDir:  "/path/c",
			want:        "",
		},
		{
			name:        "empty list",
			folderPaths: []string{},
			absWorkDir:  "/path/a",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchCurrentFolder(tt.folderPaths, tt.absWorkDir)
			if got != tt.want {
				t.Errorf("matchCurrentFolder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindEnvironmentOption(t *testing.T) {
	m := &Model{
		envOptions: []environment.Environment{
			{Name: "default", Strategy: environment.StrategyWorkspace},
			{Name: "production", Strategy: environment.StrategyWorkspace},
			{Path: "/path/to/staging", Strategy: environment.StrategyFolder},
		},
	}

	tests := []struct {
		name  string
		value string
		found bool
	}{
		{"find workspace by name", "production", true},
		{"find folder by path", "/path/to/staging", true},
		{"find folder by basename", "staging", true},
		{"not found", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, found := m.findEnvironmentOption(tt.value)
			if found != tt.found {
				t.Errorf("findEnvironmentOption(%q) found = %v, want %v", tt.value, found, tt.found)
			}
		})
	}
}
