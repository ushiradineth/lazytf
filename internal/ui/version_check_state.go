package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ushiradineth/lazytf/internal/config"
)

const updateCheckStateFileName = "update-check-state.json"

type versionCheckState struct {
	LastNotifiedRelease string `json:"last_notified_release,omitempty"`
}

func (m *Model) wasReleaseVersionNotified(release string) (bool, error) {
	release = strings.TrimSpace(release)
	if release == "" {
		return false, nil
	}
	last, err := m.lastNotifiedReleaseVersion()
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(last) == "" {
		return false, nil
	}
	return sameReleaseVersion(last, release), nil
}

func (m *Model) lastNotifiedReleaseVersion() (string, error) {
	state, err := m.loadVersionCheckState()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(state.LastNotifiedRelease), nil
}

func (m *Model) markReleaseVersionNotified(release string) error {
	release = strings.TrimSpace(release)
	if release == "" {
		return nil
	}
	path, err := m.versionCheckStatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create update-check state dir: %w", err)
	}

	payload, err := json.Marshal(versionCheckState{LastNotifiedRelease: release})
	if err != nil {
		return fmt.Errorf("marshal update-check state: %w", err)
	}
	if err := writeVersionCheckStateAtomic(path, payload); err != nil {
		return fmt.Errorf("write update-check state: %w", err)
	}
	return nil
}

func (m *Model) loadVersionCheckState() (versionCheckState, error) {
	path, err := m.versionCheckStatePath()
	if err != nil {
		return versionCheckState{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return versionCheckState{}, nil
		}
		return versionCheckState{}, fmt.Errorf("read update-check state: %w", err)
	}
	if len(data) == 0 {
		return versionCheckState{}, nil
	}

	var state versionCheckState
	if err := json.Unmarshal(data, &state); err != nil {
		return versionCheckState{}, fmt.Errorf("decode update-check state: %w", err)
	}
	return state, nil
}

func (m *Model) versionCheckStatePath() (string, error) {
	if m != nil && m.configManager != nil && strings.TrimSpace(m.configManager.Path()) != "" {
		return filepath.Join(filepath.Dir(m.configManager.Path()), updateCheckStateFileName), nil
	}
	cfgPath, err := config.ResolvePath()
	if err != nil {
		return "", fmt.Errorf("resolve config path for update-check state: %w", err)
	}
	return filepath.Join(filepath.Dir(cfgPath), updateCheckStateFileName), nil
}

func sameReleaseVersion(left, right string) bool {
	leftSemver, leftOK := parseSemVersion(left)
	rightSemver, rightOK := parseSemVersion(right)
	if leftOK && rightOK {
		return compareSemVersion(leftSemver, rightSemver) == 0
	}
	left = strings.TrimSpace(strings.TrimPrefix(left, "v"))
	right = strings.TrimSpace(strings.TrimPrefix(right, "v"))
	return strings.EqualFold(left, right)
}

func writeVersionCheckStateAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".update-check-state-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}
