package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRun_NoPlanFile(t *testing.T) {
	oldPlanFile := planFile
	oldExecute := executeMode
	t.Cleanup(func() {
		planFile = oldPlanFile
		executeMode = oldExecute
	})
	planFile = ""
	executeMode = false

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
	oldExecute := executeMode
	t.Cleanup(func() {
		planFile = oldPlanFile
		executeMode = oldExecute
	})
	planFile = filepath.Join(t.TempDir(), "missing.json")
	executeMode = false

	err := run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatalf("expected error for missing plan file")
	}
	if !strings.Contains(err.Error(), "failed to parse plan file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
