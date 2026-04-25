package tfbinary

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var lookupOrder = []string{"terraform", "tofu"}

var commonPaths = []string{
	"/usr/local/bin/terraform",
	"/opt/homebrew/bin/terraform",
	"/usr/bin/terraform",
	"/usr/local/bin/tofu",
	"/opt/homebrew/bin/tofu",
	"/usr/bin/tofu",
}

var errBinaryNotFound = errors.New("terraform/tofu binary not found")

// Resolve returns the first available terraform-compatible binary path.
func Resolve() (string, error) {
	return ResolvePreferred("")
}

// ResolvePreferred returns the configured binary when provided, otherwise
// falls back to terraform first then tofu.
func ResolvePreferred(preferred string) (string, error) {
	return resolveWith(preferred, exec.LookPath, pathExists)
}

func resolveWith(preferred string, lookPath func(string) (string, error), exists func(string) bool) (string, error) {
	if trimmed := strings.TrimSpace(preferred); trimmed != "" {
		if exists(trimmed) {
			return trimmed, nil
		}
		return "", fmt.Errorf("configured terraform/tofu binary not found: %s", trimmed)
	}

	for _, name := range lookupOrder {
		if path, err := lookPath(name); err == nil {
			return path, nil
		}
	}

	for _, path := range commonPaths {
		if exists(path) {
			return path, nil
		}
	}

	return "", errBinaryNotFound
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
