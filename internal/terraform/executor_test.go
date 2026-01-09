package terraform

import (
	"reflect"
	"testing"
)

func TestMergeEnvOverrides(t *testing.T) {
	base := []string{"FOO=1", "BAR=2"}
	set := []string{"FOO=3", "BAZ=4"}
	merged := mergeEnv(base, set)

	got := map[string]string{}
	for _, item := range merged {
		key, val := splitEnv(item)
		got[key] = val
	}

	want := map[string]string{"FOO": "3", "BAR": "2", "BAZ": "4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected env merge: %#v", got)
	}
}

func TestContainsFlag(t *testing.T) {
	flags := []string{"-lock=false", "-auto-approve"}
	if !containsFlag(flags, "-auto-approve") {
		t.Fatalf("expected flag to be found")
	}
	if containsFlag(flags, "-input=false") {
		t.Fatalf("did not expect flag to be found")
	}
}
