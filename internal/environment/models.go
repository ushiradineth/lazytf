package environment

import (
	"path/filepath"
	"time"
)

// EnvironmentStrategy describes how an environment is organized.
type EnvironmentStrategy = StrategyType

// Environment describes a detected workspace or folder environment.
type Environment struct {
	Name      string
	Path      string
	Strategy  EnvironmentStrategy
	IsCurrent bool
	Metadata  EnvironmentMetadata
}

// EnvironmentMetadata captures optional environment details.
type EnvironmentMetadata struct {
	ResourceCount    int
	LastModified     time.Time
	TerraformVersion string
	HasState         bool
}

// BuildEnvironments converts detection output into environment entries.
func BuildEnvironments(result DetectionResult, current string) []Environment {
	envs := make([]Environment, 0, len(result.Workspaces)+len(result.FolderPaths))
	for _, workspace := range result.Workspaces {
		meta := metadataForWorkspace(result.BaseDir, workspace)
		envs = append(envs, Environment{
			Name:      workspace,
			Path:      result.BaseDir,
			Strategy:  StrategyWorkspace,
			IsCurrent: workspace == current,
			Metadata:  meta,
		})
	}
	for _, folder := range result.FolderPaths {
		name := filepath.Base(folder)
		meta := metadataForFolder(folder)
		envs = append(envs, Environment{
			Name:      name,
			Path:      folder,
			Strategy:  StrategyFolder,
			IsCurrent: folder == current || name == current,
			Metadata:  meta,
		})
	}
	return envs
}
