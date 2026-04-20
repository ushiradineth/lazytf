package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/consts"
)

func TestFooterVersionLabel(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "semver", raw: "0.6.1", want: "v0.6.1"},
		{name: "prefixed semver", raw: "v0.6.1", want: "v0.6.1"},
		{name: "dev", raw: "dev", want: "dev"},
		{name: "commit", raw: "abc1234", want: "abc1234"},
		{name: "empty", raw: "", want: "dev"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := footerVersionLabel(tc.raw); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestShouldCheckLatestRelease(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "release version", raw: "0.6.1", want: true},
		{name: "prefixed release version", raw: "v0.6.1", want: true},
		{name: "dev", raw: "dev", want: false},
		{name: "short commit hash", raw: "abc1234", want: false},
		{name: "non-semver", raw: "next", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldCheckLatestRelease(tc.raw); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestIsRemoteVersionNewer(t *testing.T) {
	if !isRemoteVersionNewer("0.6.1", "v0.6.2") {
		t.Fatal("expected remote patch version to be newer")
	}
	if isRemoteVersionNewer("0.6.2", "v0.6.1") {
		t.Fatal("did not expect older remote version to be newer")
	}
	if isRemoteVersionNewer("dev", "v0.6.2") {
		t.Fatal("did not expect non-semver local version to be checked")
	}
}

func TestCheckLatestReleaseCmdSkipsDevAndCommitVersions(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	oldVersion := consts.Version
	t.Cleanup(func() {
		consts.Version = oldVersion
	})

	consts.Version = "dev"
	if cmd := m.checkLatestReleaseCmd(); cmd != nil {
		t.Fatal("expected nil command for dev version")
	}

	consts.Version = "abc1234"
	if cmd := m.checkLatestReleaseCmd(); cmd != nil {
		t.Fatal("expected nil command for commit-hash version")
	}
}

func TestCheckLatestReleaseCmdCallsFetcherForReleaseVersions(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})

	oldVersion := consts.Version
	oldFetcher := fetchLatestReleaseVersion
	t.Cleanup(func() {
		consts.Version = oldVersion
		fetchLatestReleaseVersion = oldFetcher
	})

	consts.Version = "0.6.1"
	fetchLatestReleaseVersion = func(context.Context) (string, error) {
		return "v0.7.0", nil
	}

	cmd := m.checkLatestReleaseCmd()
	if cmd == nil {
		t.Fatal("expected version-check command")
	}

	msg := cmd()
	typed, ok := msg.(VersionCheckMsg)
	if !ok {
		t.Fatalf("expected VersionCheckMsg, got %T", msg)
	}
	if typed.Error != nil {
		t.Fatalf("expected nil error, got %v", typed.Error)
	}
	if typed.Latest != "v0.7.0" {
		t.Fatalf("expected latest version v0.7.0, got %q", typed.Latest)
	}
}

func TestHandleVersionCheckShowsToastWhenUpdateAvailable(t *testing.T) {
	m := NewExecutionModel(nil, ExecutionConfig{})
	m.ready = true
	m.width = 100
	m.height = 30
	m.updateLayout()

	oldVersion := consts.Version
	t.Cleanup(func() {
		consts.Version = oldVersion
	})
	consts.Version = "0.6.1"

	_, cmd := m.handleVersionCheck(VersionCheckMsg{Latest: "v0.7.0"})
	if cmd == nil {
		t.Fatal("expected toast command when update is available")
	}
	if m.toast == nil || !m.toast.IsVisible() {
		t.Fatal("expected visible toast for version update")
	}

	view := m.View()
	if !strings.Contains(view, "Update available") {
		t.Fatalf("expected update toast message in view, got %q", view)
	}
}
