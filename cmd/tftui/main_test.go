package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRun_NoPlanFile(t *testing.T) {
	oldPlanFile := planFile
	t.Cleanup(func() {
		planFile = oldPlanFile
	})
	planFile = ""

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected error for missing plan file")
	}
	if !strings.Contains(err.Error(), "no plan file specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ParseFileError(t *testing.T) {
	oldPlanFile := planFile
	t.Cleanup(func() {
		planFile = oldPlanFile
	})
	planFile = filepath.Join(t.TempDir(), "missing.json")

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected error for missing plan file")
	}
	if !strings.Contains(err.Error(), "failed to parse plan file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
