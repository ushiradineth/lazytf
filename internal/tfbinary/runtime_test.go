package tfbinary

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewRuntimeUsesPreferredBinary(t *testing.T) {
	t.Parallel()

	path, err := resolveWith("/tmp/custom-tf", func(string) (string, error) {
		return "", errors.New("not used")
	}, func(path string) bool {
		return path == "/tmp/custom-tf"
	})
	if err != nil {
		t.Fatalf("resolve with preferred binary: %v", err)
	}
	if path != "/tmp/custom-tf" {
		t.Fatalf("expected preferred path, got %q", path)
	}
}

func TestRuntimeCombinedOutputUsesResolvedPath(t *testing.T) {
	t.Parallel()

	runtime := Runtime{path: "/bin/sh"}
	output, err := runtime.CombinedOutput(context.Background(), ".", "-c", "echo hi")
	if err != nil {
		t.Fatalf("combined output: %v", err)
	}
	if strings.TrimSpace(string(output)) != "hi" {
		t.Fatalf("expected shell output hi, got %q", string(output))
	}
}
