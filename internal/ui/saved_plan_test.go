package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRelativePlanOutputUsesExecutorWorkDir(t *testing.T) {
	for _, flags := range [][]string{{"-out=review.tfplan"}, {"-out", "review.tfplan"}} {
		t.Run(flags[0], func(t *testing.T) {
			mock := setupMockExecutor(t)
			m := NewExecutionModel(nil, ExecutionConfig{Flags: flags})
			m.executor = mock
			_, path := m.planFlagsForRun()
			want := filepath.Join(mock.WorkDir(), "review.tfplan")
			if path != want {
				t.Fatalf("saved plan path = %q, want %q", path, want)
			}
			if err := os.WriteFile(want, []byte("plan"), 0o600); err != nil {
				t.Fatal(err)
			}
			m.planFilePath = path
			got, err := m.applyFlagsForRun()
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, []string{want}) {
				t.Fatalf("apply flags = %v, want only %q", got, want)
			}
		})
	}
}

func TestSavedPlanApplyFiltersPlanningOptions(t *testing.T) {
	for _, planning := range [][]string{
		{"-out=custom.tfplan", "-detailed-exitcode", "-destroy", "-refresh=false", "-refresh-only"},
		{"-out", "custom.tfplan"},
		{"-target=aws_instance.web", "-replace=aws_instance.web"},
		{"-target", "aws_instance.web", "-replace", "aws_instance.web", "-generate-config-out", "generated.tf"},
		{"-generate-config-out=generated.tf"},
	} {
		t.Run(planning[0], func(t *testing.T) {
			mock := setupMockExecutor(t)
			path := filepath.Join(mock.WorkDir(), "saved.tfplan")
			if err := os.WriteFile(path, []byte("plan"), 0o600); err != nil {
				t.Fatal(err)
			}
			runtimeFlags := []string{
				"-parallelism=5", "-lock-timeout=60s", "-no-color",
				"-var=token=example", "-var-file=production.tfvars",
				"-var", "image=example.tfplan", "-var-file", "ephemeral.tfplan",
			}
			flags := append(slices.Clone(runtimeFlags), planning...)
			m := NewExecutionModel(nil, ExecutionConfig{Flags: flags})
			m.executor = mock
			m.planFilePath = path
			got, err := m.applyFlagsForRun()
			if err != nil {
				t.Fatal(err)
			}
			want := append(slices.Clone(runtimeFlags), path)
			if !slices.Equal(got, want) {
				t.Fatalf("apply flags = %v, want %v", got, want)
			}
			if !slices.Equal(m.planFlags, flags) {
				t.Fatalf("planning options changed: %v", m.planFlags)
			}
		})
	}
}

func TestApplyWithoutSavedPlanPreservesPlanningOptions(t *testing.T) {
	flags := []string{"-var", "image=example", "-target=aws_instance.web", "-refresh=false"}
	m := NewExecutionModel(nil, ExecutionConfig{Flags: flags})
	got, err := m.applyFlagsForRun()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, flags) {
		t.Fatalf("apply flags = %v, want %v", got, flags)
	}
}
