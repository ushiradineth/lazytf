package environment

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	envCacheFileName  = "env-cache.json"
	envConfigFileName = "env-config.json"
)

// Preference stores the user's preferred environment selection.
type Preference struct {
	Strategy    StrategyType `json:"strategy"`
	Environment string       `json:"environment,omitempty"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type cachedDetection struct {
	DetectedAt      time.Time                `json:"detected_at"`
	Strategy        StrategyType             `json:"strategy"`
	Confidence      map[StrategyType]float64 `json:"confidence,omitempty"`
	ConfidenceScore float64                  `json:"confidence_score,omitempty"`
	Workspaces      []string                 `json:"workspaces,omitempty"`
	FolderPaths     []string                 `json:"folder_paths,omitempty"`
	Warnings        []string                 `json:"warnings,omitempty"`
	BaseDir         string                   `json:"base_dir,omitempty"`
}

func cacheDir(baseDir string) string {
	return filepath.Join(baseDir, ".lazytf")
}

func cacheFilePath(baseDir string) string {
	return filepath.Join(cacheDir(baseDir), envCacheFileName)
}

func preferenceFilePath(baseDir string) string {
	return filepath.Join(cacheDir(baseDir), envConfigFileName)
}

func loadDetectionCache(baseDir string, ttl time.Duration) (DetectionResult, bool, error) {
	if strings.TrimSpace(baseDir) == "" {
		return DetectionResult{}, false, errors.New("base dir required for cache")
	}
	if ttl <= 0 {
		return DetectionResult{}, false, nil
	}
	path := cacheFilePath(baseDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DetectionResult{}, false, nil
		}
		return DetectionResult{}, false, fmt.Errorf("read environment cache: %w", err)
	}

	var cached cachedDetection
	if err := json.Unmarshal(data, &cached); err != nil {
		return DetectionResult{}, false, fmt.Errorf("decode environment cache: %w", err)
	}
	if cached.DetectedAt.IsZero() || time.Since(cached.DetectedAt) > ttl {
		return DetectionResult{}, false, nil
	}
	result := DetectionResult{
		Strategy:        cached.Strategy,
		Confidence:      cached.Confidence,
		ConfidenceScore: cached.ConfidenceScore,
		Workspaces:      cached.Workspaces,
		FolderPaths:     cached.FolderPaths,
		Warnings:        cached.Warnings,
		BaseDir:         cached.BaseDir,
	}
	if result.BaseDir == "" {
		result.BaseDir = baseDir
	}
	result.Environments = BuildEnvironments(result, "")
	return result, true, nil
}

func saveDetectionCache(baseDir string, result DetectionResult) error {
	if strings.TrimSpace(baseDir) == "" {
		return errors.New("base dir required for cache")
	}
	cache := cachedDetection{
		DetectedAt:      time.Now(),
		Strategy:        result.Strategy,
		Confidence:      result.Confidence,
		ConfidenceScore: result.ConfidenceScore,
		Workspaces:      result.Workspaces,
		FolderPaths:     result.FolderPaths,
		Warnings:        result.Warnings,
		BaseDir:         result.BaseDir,
	}
	return writeJSONAtomic(cacheFilePath(baseDir), cache)
}

// LoadPreference reads the user's environment preference.
func LoadPreference(baseDir string) (*Preference, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, errors.New("base dir required for preference")
	}
	path := preferenceFilePath(baseDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read environment preference: %w", err)
	}
	var pref Preference
	if err := json.Unmarshal(data, &pref); err != nil {
		return nil, fmt.Errorf("decode environment preference: %w", err)
	}
	if pref.Strategy == "" && pref.Environment == "" {
		return nil, nil
	}
	return &pref, nil
}

// SavePreference persists the user's environment preference.
func SavePreference(baseDir string, pref Preference) error {
	if strings.TrimSpace(baseDir) == "" {
		return errors.New("base dir required for preference")
	}
	if pref.UpdatedAt.IsZero() {
		pref.UpdatedAt = time.Now()
	}
	return writeJSONAtomic(preferenceFilePath(baseDir), pref)
}

func writeJSONAtomic(path string, payload any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create env config dir: %w", err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode env json: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".env-*.tmp")
	if err != nil {
		return fmt.Errorf("create env temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		if err := os.Remove(tmpPath); err != nil {
			_ = err
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write env temp file: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod env temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync env temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close env temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename env temp file: %w", err)
	}
	return nil
}
