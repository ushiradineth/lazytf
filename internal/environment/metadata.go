package environment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const maxStateSize = 5 * 1024 * 1024

type terraformState struct {
	TerraformVersion string            `json:"terraform_version"`
	Resources        []json.RawMessage `json:"resources"`
}

func metadataForWorkspace(baseDir, workspace string) EnvironmentMetadata {
	statePath, ok := workspaceStateFile(baseDir, workspace)
	if !ok {
		return EnvironmentMetadata{}
	}
	return metadataFromState(statePath)
}

func metadataForFolder(path string) EnvironmentMetadata {
	statePath, ok := folderStateFile(path)
	if !ok {
		return EnvironmentMetadata{}
	}
	return metadataFromState(statePath)
}

func workspaceStateFile(baseDir, workspace string) (string, bool) {
	if strings.TrimSpace(baseDir) == "" {
		return "", false
	}
	if strings.TrimSpace(workspace) == "" {
		return "", false
	}

	candidates := []string{}
	if workspace == "default" {
		candidates = append(candidates, filepath.Join(baseDir, "terraform.tfstate"))
	}
	candidates = append(candidates,
		filepath.Join(baseDir, "terraform.tfstate.d", workspace, "terraform.tfstate"),
		filepath.Join(baseDir, ".terraform", "terraform.tfstate.d", workspace, "terraform.tfstate"),
	)

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func folderStateFile(path string) (string, bool) {
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	candidates := []string{
		filepath.Join(path, "terraform.tfstate"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func metadataFromState(path string) EnvironmentMetadata {
	info, err := os.Stat(path)
	if err != nil {
		return EnvironmentMetadata{}
	}
	meta := EnvironmentMetadata{
		HasState:     true,
		LastModified: info.ModTime(),
	}
	if info.Size() > maxStateSize {
		return meta
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return meta
	}
	var state terraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return meta
	}
	meta.ResourceCount = len(state.Resources)
	meta.TerraformVersion = state.TerraformVersion
	return meta
}
