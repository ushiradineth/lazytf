package components

import (
	"regexp"
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

func TestRenderResourceDoesNotExpandDiffs(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resource := terraform.ResourceChange{
		Address: "aws_vpc.main",
		Action:  terraform.ActionUpdate,
		Change: &terraform.Change{
			Before: map[string]interface{}{"name": "old"},
			After:  map[string]interface{}{"name": "new"},
		},
	}

	out := r.renderResource(resource, false)
	if !strings.Contains(out, "aws_vpc.main") || strings.Contains(out, "~ name") {
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

func TestRenderDiffMultilineColorsAndPrefixes(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	multi := diff.MinimalDiff{
		Path:     []string{"values[0]"},
		Action:   diff.DiffChange,
		OldValue: "retry: true\nlog: info\n",
		NewValue: "retry: false\nlog: debug\n",
	}

	out := r.renderDiff(multi)
	plain := stripANSI(out)
	if !strings.Contains(plain, "~ values[0]:") {
		t.Fatalf("expected header, got %q", plain)
	}
	if !strings.Contains(plain, "- retry: true") || !strings.Contains(plain, "+ retry: false") {
		t.Fatalf("expected +/- lines, got %q", plain)
	}
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

func TestFuzzyMatchNonContiguous(t *testing.T) {
	if !fuzzyMatch("awsins", "aws_instance") {
		t.Fatalf("expected non-contiguous match to pass")
	}
	if fuzzyMatch("azs", "aws_instance") {
		t.Fatalf("expected non-match to fail")
	}
}

func TestFuzzyMatchCaseInsensitive(t *testing.T) {
	if !fuzzyMatch("aws", strings.ToLower("AWS_INSTANCE")) {
		t.Fatalf("expected case-insensitive match to pass")
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("AWS")
	if got := r.GetSelectedResource(); got == nil {
		t.Fatalf("expected case-insensitive search to match")
	}
}

func TestSearchClearsSelectionWhenEmpty(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("nomatch")

	if got := r.GetSelectedResource(); got != nil {
		t.Fatalf("expected nil selection, got %#v", got)
	}
}

func TestSearchEmptyQueryRestoresSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
		{Address: "aws_instance.db", Action: terraform.ActionCreate},
	})
	r.selectedIndex = 1
	r.SetSearchQuery("nomatch")
	if r.selectedIndex != 0 {
		t.Fatalf("expected selection reset on empty list, got %d", r.selectedIndex)
	}
	r.SetSearchQuery("")
	if got := r.GetSelectedResource(); got == nil || got.Address != "aws_instance.web" {
		t.Fatalf("expected selection restored to first item, got %#v", got)
	}
}

func TestSearchQueryTrimsWhitespace(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("  web ")
	if got := r.GetSelectedResource(); got == nil || got.Address != "aws_instance.web" {
		t.Fatalf("expected trimmed query to match, got %#v", got)
	}
}

func TestSearchQueryMatchesDotAndBracketSegments(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.app.aws_instance.web[0]", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("web[0]")
	if got := r.GetSelectedResource(); got == nil {
		t.Fatalf("expected bracketed query to match")
	}
	r.SetSearchQuery("module.app")
	if got := r.GetSelectedResource(); got == nil {
		t.Fatalf("expected dot query to match")
	}
}

func TestSearchMatchesResourceTypeAndName(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{
			Address:      "aws_instance.web",
			ResourceType: "aws_instance",
			ResourceName: "web",
			Action:       terraform.ActionCreate,
		},
	})
	r.SetSearchQuery("aws_instance web")
	if got := r.GetSelectedResource(); got == nil {
		t.Fatalf("expected resource type/name to match")
	}
}

func TestFilterAndSearchCombined(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionUpdate},
	})
	r.SetFilter(terraform.ActionUpdate, false)
	r.SetSearchQuery("web")
	if got := r.GetSelectedResource(); got != nil {
		t.Fatalf("expected filtered resource to be hidden, got %#v", got)
	}
}

func TestRenderDiffQuotesPathSegments(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	item := diff.MinimalDiff{
		Path:     []string{"data", "app.yaml"},
		OldValue: "a",
		NewValue: "b",
		Action:   diff.DiffChange,
	}

	out := stripANSI(r.renderDiff(item))
	if !strings.Contains(out, `data."app.yaml"`) {
		t.Fatalf("expected quoted path, got %q", out)
	}
}

func TestRenderLongResourceAddressTruncates(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(20, 5)
	resource := terraform.ResourceChange{
		Address: strings.Repeat("a", 80),
		Action:  terraform.ActionCreate,
		Change:  &terraform.Change{Actions: []string{"create"}, Before: nil, After: map[string]interface{}{"x": "y"}},
	}
	out := stripANSI(r.renderResource(resource, false))
	lines := strings.Split(out, "\n")
	if len(lines) == 0 || len(lines[0]) > 20 {
		t.Fatalf("expected truncated header line, got %q", out)
	}
}

func TestExpandedResourceZeroDiffsNoExtraLine(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resource := terraform.ResourceChange{
		Address: "aws_instance.web",
		Action:  terraform.ActionUpdate,
		Change:  &terraform.Change{Actions: []string{"update"}, Before: map[string]interface{}{"a": 1}, After: map[string]interface{}{"a": 1}},
	}
	out := r.renderResource(resource, false)
	if strings.Count(out, "\n") != 0 {
		t.Fatalf("expected no extra diff lines, got %q", out)
	}
}

func TestFilterPreservesSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "a", Action: terraform.ActionCreate},
		{Address: "b", Action: terraform.ActionDelete},
	})
	r.selectedIndex = 0
	r.SetFilter(terraform.ActionDelete, false)
	if got := r.GetSelectedResource(); got == nil || got.Address != "a" {
		t.Fatalf("expected selection preserved, got %#v", got)
	}
}
