package tfbinary

import (
	"errors"
	"os"
	"os/exec"
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

var errBinaryNotFound = errors.New("terraform/tofu binary not found in PATH")

// Resolve returns the first available terraform-compatible binary path.
func Resolve() (string, error) {
	return resolveWith(exec.LookPath, pathExists)
}

func resolveWith(lookPath func(string) (string, error), exists func(string) bool) (string, error) {
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
