package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/consts"
)

const latestReleaseAPIURL = "https://api.github.com/repos/ushiradineth/lazytf/releases/latest"

const versionDev = "dev"

var (
	commitHashVersionPattern  = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	fetchLatestReleaseVersion = defaultFetchLatestReleaseVersion
)

type semVersion struct {
	major int
	minor int
	patch int
}

type githubLatestReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func (m *Model) checkLatestReleaseCmd() tea.Cmd {
	if m.shouldSuppressUpdateAvailableWarning() {
		return nil
	}

	local := strings.TrimSpace(consts.Version)
	if !shouldCheckLatestRelease(local) {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		latest, err := fetchLatestReleaseVersion(ctx)
		return VersionCheckMsg{Latest: latest, Error: err}
	}
}

func (m *Model) handleVersionCheck(msg VersionCheckMsg) (tea.Model, tea.Cmd) {
	if m.shouldSuppressUpdateAvailableWarning() {
		return m, nil
	}
	if msg.Error != nil {
		return m, nil
	}

	latest := strings.TrimSpace(msg.Latest)
	local := strings.TrimSpace(consts.Version)
	if latest == "" || !shouldCheckLatestRelease(local) {
		return m, nil
	}
	if !isRemoteVersionNewer(local, latest) {
		return m, nil
	}
	alreadyNotified, err := m.wasReleaseVersionNotified(latest)
	if err != nil {
		m.appendSessionLog("Version check state", "update-check-state", err.Error())
	}
	if alreadyNotified {
		return m, nil
	}
	if m.toast == nil {
		return m, nil
	}

	currentLabel := footerVersionLabel(local)
	latestLabel := footerVersionLabel(latest)
	message := fmt.Sprintf("Update available: %s (current %s)", latestLabel, currentLabel)
	if err := m.markReleaseVersionNotified(latest); err != nil {
		m.appendSessionLog("Version check state", "update-check-state", err.Error())
	}
	return m, m.toast.ShowInfo(message)
}

func (m *Model) shouldSuppressUpdateAvailableWarning() bool {
	if m == nil || m.config == nil {
		return false
	}
	return m.config.Warnings.SuppressAll || m.config.Warnings.SuppressUpdateAvailable
}

func (m *Model) footerVersionTag() string {
	label := footerVersionLabel(consts.Version)
	if m.styles == nil {
		return label
	}
	return m.styles.Dimmed.Render(label)
}

func footerVersionLabel(raw string) string {
	version := strings.TrimSpace(raw)
	if version == "" {
		return versionDev
	}
	if strings.EqualFold(version, versionDev) || commitHashVersionPattern.MatchString(version) {
		return version
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func shouldCheckLatestRelease(local string) bool {
	version := strings.TrimSpace(local)
	if version == "" || strings.EqualFold(version, versionDev) {
		return false
	}
	if commitHashVersionPattern.MatchString(version) {
		return false
	}
	_, ok := parseSemVersion(version)
	return ok
}

func isRemoteVersionNewer(local, remote string) bool {
	localSemver, ok := parseSemVersion(local)
	if !ok {
		return false
	}
	remoteSemver, ok := parseSemVersion(remote)
	if !ok {
		return false
	}

	return compareSemVersion(remoteSemver, localSemver) > 0
}

func parseSemVersion(raw string) (semVersion, bool) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "v")
	if idx := strings.IndexAny(value, "-+"); idx >= 0 {
		value = value[:idx]
	}

	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return semVersion{}, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return semVersion{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 {
		return semVersion{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil || patch < 0 {
		return semVersion{}, false
	}

	return semVersion{major: major, minor: minor, patch: patch}, true
}

func compareSemVersion(left, right semVersion) int {
	if left.major != right.major {
		if left.major > right.major {
			return 1
		}
		return -1
	}
	if left.minor != right.minor {
		if left.minor > right.minor {
			return 1
		}
		return -1
	}
	if left.patch != right.patch {
		if left.patch > right.patch {
			return 1
		}
		return -1
	}
	return 0
}

func defaultFetchLatestReleaseVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseAPIURL, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("build latest release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "lazytf-version-check")

	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- request URL is fixed latestReleaseAPIURL constant.
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch latest release: unexpected status %s", resp.Status)
	}

	var payload githubLatestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode latest release response: %w", err)
	}

	return strings.TrimSpace(payload.TagName), nil
}
