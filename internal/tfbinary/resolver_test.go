package tfbinary

import (
	"errors"
	"testing"
)

func TestResolveWithPrefersTerraformOverTofu(t *testing.T) {
	t.Parallel()

	lookPath := func(name string) (string, error) {
		switch name {
		case "terraform":
			return "/tmp/terraform", nil
		case "tofu":
			return "/tmp/tofu", nil
		default:
			return "", errors.New("unknown binary")
		}
	}

	path, err := resolveWith(lookPath, func(_ string) bool { return false })
	if err != nil {
		t.Fatalf("resolve with terraform+tofu available: %v", err)
	}
	if path != "/tmp/terraform" {
		t.Fatalf("expected terraform path, got %q", path)
	}
}

func TestResolveWithFallsBackToTofu(t *testing.T) {
	t.Parallel()

	lookPath := func(name string) (string, error) {
		if name == "tofu" {
			return "/tmp/tofu", nil
		}
		return "", errors.New("missing")
	}

	path, err := resolveWith(lookPath, func(_ string) bool { return false })
	if err != nil {
		t.Fatalf("resolve with tofu fallback: %v", err)
	}
	if path != "/tmp/tofu" {
		t.Fatalf("expected tofu path, got %q", path)
	}
}

func TestResolveWithFallsBackToCommonAbsolutePath(t *testing.T) {
	t.Parallel()

	lookPath := func(string) (string, error) {
		return "", errors.New("missing")
	}
	path, err := resolveWith(lookPath, func(path string) bool {
		return path == "/opt/homebrew/bin/tofu"
	})
	if err != nil {
		t.Fatalf("resolve with common absolute fallback: %v", err)
	}
	if path != "/opt/homebrew/bin/tofu" {
		t.Fatalf("expected common path fallback, got %q", path)
	}
}

func TestResolveWithCommonPathFallbackPrefersTerraform(t *testing.T) {
	t.Parallel()

	lookPath := func(string) (string, error) {
		return "", errors.New("missing")
	}

	path, err := resolveWith(lookPath, func(path string) bool {
		return path == "/opt/homebrew/bin/terraform" || path == "/usr/local/bin/tofu"
	})
	if err != nil {
		t.Fatalf("resolve with conflicting common paths: %v", err)
	}
	if path != "/opt/homebrew/bin/terraform" {
		t.Fatalf("expected terraform-preferred fallback, got %q", path)
	}
}

func TestResolveWithMissingBinary(t *testing.T) {
	t.Parallel()

	lookPath := func(string) (string, error) {
		return "", errors.New("missing")
	}

	_, err := resolveWith(lookPath, func(_ string) bool { return false })
	if err == nil {
		t.Fatal("expected missing binary error")
	}
	if !errors.Is(err, errBinaryNotFound) {
		t.Fatalf("expected errBinaryNotFound, got %v", err)
	}
}
