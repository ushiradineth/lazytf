package components

import (
	"strings"
	"testing"

	"github.com/ushiradineth/tftui/internal/diff"
	"github.com/ushiradineth/tftui/internal/styles"
	"github.com/ushiradineth/tftui/internal/terraform"
)

func TestFormatValue(t *testing.T) {
	if got := formatValue(nil); got != "(null)" {
		t.Fatalf("expected null, got %q", got)
	}
	if got := formatValue("hi"); got != "\"hi\"" {
		t.Fatalf("expected quoted string, got %q", got)
	}
	if got := formatValue(diff.UnknownValue{}); got != "(known after apply)" {
		t.Fatalf("expected known after apply, got %q", got)
	}
	if got := formatValue(map[string]interface{}{"a": 1}); got != "{...}" {
		t.Fatalf("expected map placeholder, got %q", got)
	}
	if got := formatValue([]interface{}{"one"}); got != "\"one\"" {
		t.Fatalf("expected single string list to format to string, got %q", got)
	}
	if got := formatValue([]interface{}{"one", "two"}); got != "[...]" {
		t.Fatalf("expected list placeholder, got %q", got)
	}
}

func TestFormatMultilineStringDiff(t *testing.T) {
	before := "- a\n- b\n- c"
	after := "- a\n- x\n- c"
	got := formatMultilineStringDiff("list", before, after)
	if got == "" {
		t.Fatalf("expected multiline diff")
	}
	if !strings.Contains(got, "~ list:") || !strings.Contains(got, "- b") || !strings.Contains(got, "+ x") {
		t.Fatalf("unexpected multiline diff: %q", got)
	}
}

func TestFormatMultilineStringDiff_MismatchOrNoDiff(t *testing.T) {
	if got := formatMultilineStringDiff("list", "a\nb", "a"); got != "" {
		t.Fatalf("expected empty diff for mismatched lines, got %q", got)
	}
	if got := formatMultilineStringDiff("list", "a\nb", "a\nb"); got != "" {
		t.Fatalf("expected empty diff for identical lines, got %q", got)
	}
}

func TestTruncateLineAndStripListMarker(t *testing.T) {
	if got := truncateLine("hello", 3); got != "hel" {
		t.Fatalf("unexpected truncation: %q", got)
	}
	if got := truncateLine("hello", 5); got != "hello" {
		t.Fatalf("unexpected truncation: %q", got)
	}
	if got := stripListMarker("- item"); got != "item" {
		t.Fatalf("unexpected strip result: %q", got)
	}
	if got := stripListMarker("item"); got != "item" {
		t.Fatalf("unexpected strip result: %q", got)
	}
}

func TestInterfaceHelpers(t *testing.T) {
	if !isMap(map[string]interface{}{"a": 1}) {
		t.Fatalf("expected map to be detected")
	}
	if isMap(nil) {
		t.Fatalf("expected nil not to be a map")
	}
	if !isList([]int{1, 2}) {
		t.Fatalf("expected list to be detected")
	}
	if isList(nil) {
		t.Fatalf("expected nil not to be a list")
	}
	if got := interfaceToList([]string{"a", "b"}); len(got) != 2 {
		t.Fatalf("unexpected list length: %d", len(got))
	}
}

func TestResourceListFilteringAndSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resources := []terraform.ResourceChange{
		{Address: "a", Action: terraform.ActionCreate},
		{Address: "b", Action: terraform.ActionDelete},
	}
	r.SetResources(resources)
	r.SetFilter(terraform.ActionDelete, false)

	if got := r.GetSelectedResource(); got == nil || got.Address != "a" {
		t.Fatalf("unexpected selected resource: %#v", got)
	}

	r.MoveDown()
	if r.selectedIndex != 0 {
		t.Fatalf("expected selection to stay at 0, got %d", r.selectedIndex)
	}

	r.ToggleSelected()
	if !r.expandedMap["a"] {
		t.Fatalf("expected resource to be expanded")
	}

	r.SetFilter(terraform.ActionCreate, false)
	if got := r.GetSelectedResource(); got != nil {
		t.Fatalf("expected nil selection when filtered out, got %#v", got)
	}
}

func TestResourceListViewEmpty(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(20, 5)
	r.SetResources(nil)
	got := r.View()
	normalized := strings.Join(strings.Fields(got), "")
	if !strings.Contains(normalized, "Noresourcestodisplay") {
		t.Fatalf("unexpected empty view: %q", got)
	}
}

func TestRenderResourceExpandedIncludesDiff(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resource := terraform.ResourceChange{
		Address: "aws_vpc.main",
		Action:  terraform.ActionUpdate,
		Change: &terraform.Change{
			Before: map[string]interface{}{"name": "old"},
			After:  map[string]interface{}{"name": "new"},
		},
	}

	out := r.renderResource(resource, false, true)
	if !strings.Contains(out, "aws_vpc.main") || !strings.Contains(out, "~ name") {
		t.Fatalf("unexpected render output: %q", out)
	}
}

func TestRenderDiffUnknownAction(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	line := r.renderDiff(diff.MinimalDiff{Path: []string{"a"}, Action: diff.DiffAction("weird")})
	if !strings.Contains(line, "? a") {
		t.Fatalf("unexpected diff line: %q", line)
	}
}
