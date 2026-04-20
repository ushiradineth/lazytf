package components

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ushiradineth/lazytf/internal/consts"
	"github.com/ushiradineth/lazytf/internal/diff"
	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
	"github.com/ushiradineth/lazytf/internal/utils"
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
	if got := formatValue(map[string]any{"a": 1}); got != "{...}" {
		t.Fatalf("expected map placeholder, got %q", got)
	}
	if got := formatValue([]any{"one"}); got != "\"one\"" {
		t.Fatalf("expected single string list to format to string, got %q", got)
	}
	if got := formatValue([]any{"one", "two"}); got != "[...]" {
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

func TestTruncateEndAndStripListMarker(t *testing.T) {
	if got := utils.TruncateEnd("hello", 3); got != "hel" {
		t.Fatalf("unexpected truncation: %q", got)
	}
	if got := utils.TruncateEnd("hello", 5); got != "hello" {
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
	if !utils.IsMap(map[string]any{"a": 1}) {
		t.Fatalf("expected map to be detected")
	}
	if utils.IsMap(nil) {
		t.Fatalf("expected nil not to be a map")
	}
	if !utils.IsList([]int{1, 2}) {
		t.Fatalf("expected list to be detected")
	}
	if utils.IsList(nil) {
		t.Fatalf("expected nil not to be a list")
	}
	if got := utils.InterfaceToList([]string{"a", "b"}); len(got) != 2 {
		t.Fatalf("unexpected list length: %d", len(got))
	}
}

func TestResourceListFilteringAndSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resources := []terraform.ResourceChange{
		{Address: "aws_instance.a", Action: terraform.ActionCreate},
		{Address: "aws_instance.b", Action: terraform.ActionDelete},
	}
	r.SetResources(resources)
	r.SetFilter(terraform.ActionDelete, false)

	r.MoveDown()
	if got := r.GetSelectedResource(); got == nil || got.Address != "aws_instance.a" {
		t.Fatalf("unexpected selected resource: %#v", got)
	}

	r.SetFilter(terraform.ActionCreate, false)
	if got := r.GetSelectedResource(); got != nil {
		t.Fatalf("expected nil selection when filtered out, got %#v", got)
	}
}

func TestResourceListSelectVisibleRow(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(50, 8)
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.a", Action: terraform.ActionCreate},
		{Address: "aws_instance.b", Action: terraform.ActionUpdate},
		{Address: "aws_instance.c", Action: terraform.ActionDelete},
	})

	if !r.SelectVisibleRow(1) {
		t.Fatal("expected visible row selection to succeed")
	}
	selected := r.GetSelectedResource()
	if selected == nil || selected.Address != "aws_instance.b" {
		t.Fatalf("expected second resource selected, got %#v", selected)
	}
}

func TestResourceListSelectVisibleRowWithSummaryHeader(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(50, 8)
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.a", Action: terraform.ActionCreate},
		{Address: "aws_instance.b", Action: terraform.ActionUpdate},
	})
	r.SetSummary(1, 1, 0, 0)

	if r.SelectVisibleRow(0) {
		t.Fatal("expected summary header row to be non-selectable")
	}
	if !r.SelectVisibleRow(1) {
		t.Fatal("expected first list row after summary header to be selectable")
	}
	selected := r.GetSelectedResource()
	if selected == nil || selected.Address != "aws_instance.a" {
		t.Fatalf("expected first resource selected, got %#v", selected)
	}
}

func TestResourceListViewEmpty(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(40, 5)
	r.SetResources(nil)
	got := r.View()
	// Check that the view contains the expected text
	if !strings.Contains(got, "No resources") || !strings.Contains(got, "display") {
		t.Fatalf("unexpected empty view: %q", got)
	}
	// Check that it has a border/title (new panel format)
	if !strings.Contains(got, "Resources") {
		t.Fatalf("expected Resources title in view: %q", got)
	}
}

func TestResourceListShowStatusToggle(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	if r.ShowStatus() {
		t.Fatalf("expected status hidden by default")
	}
	r.SetShowStatus(true)
	if !r.ShowStatus() {
		t.Fatalf("expected status enabled")
	}
}

func TestResourceListRenderGroupAndVisibleItems(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(40, 5)
	group := r.renderGroup(consts.ModuleAlpha, 2, true, true, 2, 38, "")
	if !strings.Contains(group, consts.ModuleAlpha) {
		t.Fatalf("expected group label")
	}

	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionUpdate},
	})
	out := r.renderVisibleItems()
	if out == "" {
		t.Fatalf("expected visible items output")
	}
}

func TestResourceListStatusDisplayAndDuration(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	state := terraform.NewOperationState()
	state.StartResource("aws_instance.web", terraform.ActionCreate)
	state.CompleteResource("aws_instance.web", "id")
	r.SetOperationState(state)
	r.SetShowStatus(true)

	resource := terraform.ResourceChange{Address: "aws_instance.web", Action: terraform.ActionCreate}
	status, opStatus, elapsed := r.getStatusDisplay(resource)
	if status == "" {
		t.Fatalf("expected status badge")
	}
	if opStatus != terraform.StatusComplete {
		t.Fatalf("expected complete status, got %v", opStatus)
	}
	if elapsed == "" {
		t.Fatalf("expected elapsed text")
	}
	if formatShortDuration(500*time.Millisecond) == "" {
		t.Fatalf("expected short duration format")
	}
	if PadLineWithBg("x", 3, r.styles.SelectedLineBackground) == "" {
		t.Fatalf("expected padded line")
	}
}

func TestRenderResourceDoesNotExpandDiffs(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resource := terraform.ResourceChange{
		Address: "aws_vpc.main",
		Action:  terraform.ActionUpdate,
		Change: &terraform.Change{
			Before: map[string]any{"name": "old"},
			After:  map[string]any{"name": "new"},
		},
	}

	out := r.renderResource(&resource, false, 0, 80, "")
	if !strings.Contains(out, "aws_vpc.main") || strings.Contains(out, "~ name") {
		t.Fatalf("unexpected render output: %q", out)
	}
}

func TestRenderResourceTrimsModulePrefix(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resource := terraform.ResourceChange{
		Address: "module.alpha.aws_instance.web",
		Action:  terraform.ActionUpdate,
		Change:  &terraform.Change{Before: map[string]any{"a": 1}, After: map[string]any{"a": 2}},
	}

	out := r.renderResource(&resource, false, 2, 80, "")
	if !strings.Contains(out, "aws_instance.web") || strings.Contains(out, consts.ModuleAlpha) {
		t.Fatalf("expected trimmed module prefix, got %q", out)
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

func selectFirstResource(r *ResourceList) *terraform.ResourceChange {
	for i := 0; i < 20; i++ {
		if res := r.GetSelectedResource(); res != nil {
			return res
		}
		r.MoveDown()
	}
	return nil
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func isSorted(list []string) bool {
	for i := 1; i < len(list); i++ {
		if list[i-1] > list[i] {
			return false
		}
	}
	return true
}

func indexOfGroup(items []listItem, label string) int {
	for i, item := range items {
		if item.kind == itemGroup && item.label == label {
			return i
		}
	}
	return -1
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

func TestFuzzyMatchOrderRequired(t *testing.T) {
	// Test that characters must appear in order
	tests := []struct {
		query       string
		candidate   string
		shouldMatch bool
	}{
		// Should match: e-g-g appears in order
		{"egg", "gselllllglllawdaeaewdwag", true},
		// Should NOT match: only one 'g' after 'e'
		{"egg", "gawdhuaowdhoaweawdbiuawbdiauwbdg", false},
		// Should NOT match: 'e' comes after both 'g's
		{"egg", "gge", false},
		// Additional test cases
		{"abc", "aabbcc", true},
		{"abc", "acb", false}, // 'b' comes after 'c'
		{"xyz", "xaybzc", true},
		{"xyz", "xzy", false}, // 'y' comes after 'z'
		// Real-world terraform examples
		{"edge4", "module.gamma.kubernetes_config_map.settings_4", false},          // no 'd' after 'e' and before 'g'
		{"edge4", "module.zeta.aws_instance.node_4", false},                        // no 'g' in address
		{"edge4", "module.beta.module.db.aws_security_group.legacy_4", true},       // e(modul-e) d(mo-d-ule) g(security_-g-roup) e(l-e-gacy) 4(-4)
		{"edge4", "module.alpha.module.net.module.edge.aws_instance.node_4", true}, // has actual 'edge' substring
		{"edge", "module.alpha.module.net.module.edge.aws_instance.node_0", true},
	}

	for _, tt := range tests {
		result := fuzzyMatch(tt.query, tt.candidate)
		if result != tt.shouldMatch {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v",
				tt.query, tt.candidate, result, tt.shouldMatch)
		}
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("AWS")
	if got := selectFirstResource(r); got == nil {
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
	if got := selectFirstResource(r); got == nil {
		t.Fatalf("expected selection restored to resource, got %#v", got)
	}
}

func TestSearchQueryTrimsWhitespace(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("  web ")
	if got := selectFirstResource(r); got == nil || got.Address != "aws_instance.web" {
		t.Fatalf("expected trimmed query to match, got %#v", got)
	}
}

func TestSearchQueryMatchesDotAndBracketSegments(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.app.aws_instance.web[0]", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("web[0]")
	if got := selectFirstResource(r); got == nil {
		t.Fatalf("expected bracketed query to match")
	}
	r.SetSearchQuery("module.app")
	if got := selectFirstResource(r); got == nil {
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
	r.SetSearchQuery("aws_instance")
	if got := selectFirstResource(r); got == nil {
		t.Fatalf("expected resource type to match")
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
		Change:  &terraform.Change{Actions: []string{"create"}, Before: nil, After: map[string]any{"x": "y"}},
	}
	out := stripANSI(r.renderResource(&resource, false, 0, 18, ""))
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
		Change:  &terraform.Change{Actions: []string{"update"}, Before: map[string]any{"a": 1}, After: map[string]any{"a": 1}},
	}
	out := r.renderResource(&resource, false, 0, 80, "")
	if strings.Count(out, "\n") != 0 {
		t.Fatalf("expected no extra diff lines, got %q", out)
	}
}

func TestFilterPreservesSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.a", Action: terraform.ActionCreate},
		{Address: "aws_instance.b", Action: terraform.ActionDelete},
	})
	r.selectedIndex = 1
	r.SetFilter(terraform.ActionDelete, false)
	if got := r.GetSelectedResource(); got == nil || got.Address != "aws_instance.a" {
		t.Fatalf("expected selection preserved, got %#v", got)
	}
}

func TestToggleAllGroups(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	if len(r.visibleItems) == 0 {
		t.Fatalf("expected visible items")
	}

	r.ToggleAllGroups()
	if r.allExpanded {
		t.Fatalf("expected all groups collapsed")
	}

	r.ToggleAllGroups()
	if !r.allExpanded {
		t.Fatalf("expected all groups expanded")
	}
}

func TestModulePathParsing(t *testing.T) {
	if got := modulePath("aws_instance.web"); len(got) != 0 {
		t.Fatalf("expected empty module path, got %#v", got)
	}
	got := modulePath("module.alpha.module.net.aws_instance.web")
	if len(got) != 2 || got[0] != "alpha" || got[1] != "net" {
		t.Fatalf("unexpected module path: %#v", got)
	}
}

func TestDeepGroupingCreatesNestedItems(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.aws_instance.db", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.security.aws_instance.bastion", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	foundGroup := false
	foundSubGroup := false
	for _, item := range r.visibleItems {
		if item.kind == itemGroup && item.label == consts.ModuleAlpha {
			foundGroup = true
		}
		if item.kind == itemGroup && item.label == "module.net" {
			foundSubGroup = true
		}
	}
	if !foundGroup || !foundSubGroup {
		t.Fatalf("expected nested groups, got %#v", r.visibleItems)
	}
}

func TestGroupLabelLocalNamePerDepth(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	labels := []string{}
	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			labels = append(labels, item.label)
		}
	}
	if !containsString(labels, consts.ModuleAlpha) || !containsString(labels, "module.net") || !containsString(labels, "module.edge") {
		t.Fatalf("unexpected group labels: %#v", labels)
	}
}

func TestGroupIndentIncrementsPerDepth(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	indents := []int{}
	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			indents = append(indents, item.indent)
		}
	}
	if len(indents) < 3 || indents[0] != 0 || indents[1] != 2 || indents[2] != 4 {
		t.Fatalf("unexpected indents: %#v", indents)
	}
}

func TestToggleGroupCollapseHidesDescendants(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	if groupIdx < 0 {
		t.Fatalf("expected module.alpha group")
	}
	r.selectedIndex = groupIdx
	r.ToggleGroup()
	for _, item := range r.visibleItems {
		if item.kind == itemGroup && item.label != consts.ModuleAlpha {
			t.Fatalf("expected descendants hidden, got %#v", r.visibleItems)
		}
		if item.kind == itemResource {
			t.Fatalf("expected no resources when collapsed, got %#v", r.visibleItems)
		}
	}
}

func TestToggleGroupExpandRestoresSortedChildren(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.zeta", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.alpha", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	r.selectedIndex = groupIdx
	r.ToggleGroup()
	r.ToggleGroup()

	resourceAddrs := []string{}
	for _, item := range r.visibleItems {
		if item.kind == itemResource {
			resourceAddrs = append(resourceAddrs, item.resource.Address)
		}
	}
	if len(resourceAddrs) < 2 || resourceAddrs[0] > resourceAddrs[1] {
		t.Fatalf("expected sorted children, got %#v", resourceAddrs)
	}
}

func TestGroupsSortedAlphabeticallyPerDepth(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.beta.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.gamma.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupLabels := []string{}
	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			groupLabels = append(groupLabels, item.label)
		}
	}
	if !isSorted(groupLabels) {
		t.Fatalf("expected groups sorted, got %#v", groupLabels)
	}
}

func TestResourcesSortedWithinGroup(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.zeta", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.alpha", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	resourceAddrs := []string{}
	for _, item := range r.visibleItems {
		if item.kind == itemResource {
			resourceAddrs = append(resourceAddrs, item.resource.Address)
		}
	}
	if len(resourceAddrs) < 2 || resourceAddrs[0] > resourceAddrs[1] {
		t.Fatalf("expected sorted resources, got %#v", resourceAddrs)
	}
}

func TestUngroupedResourcesAfterGroups(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "aws_instance.root", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	lastGroupIdx := -1
	rootIdx := -1
	for i, item := range r.visibleItems {
		if item.kind == itemGroup {
			lastGroupIdx = i
		}
		if item.kind == itemResource && item.resource.Address == "aws_instance.root" {
			rootIdx = i
		}
	}
	if rootIdx <= lastGroupIdx {
		t.Fatalf("expected ungrouped resource after groups: %#v", r.visibleItems)
	}
}

func TestSelectionOnGroupHeaderHasNoResource(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	r.selectedIndex = groupIdx
	if got := r.GetSelectedResource(); got != nil {
		t.Fatalf("expected nil resource for group selection, got %#v", got)
	}
}

func TestSelectionMovesOverCollapsedGroup(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	r.selectedIndex = groupIdx
	r.ToggleGroup()
	r.MoveDown()
	if r.selectedIndex == groupIdx {
		t.Fatalf("expected selection to move past collapsed group")
	}
}

func TestCollapseSelectedGroupClearsDiffSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	r.selectedIndex = groupIdx
	r.ToggleGroup()
	if r.GetSelectedResource() != nil {
		t.Fatalf("expected no selected resource after collapsing group")
	}
}

func TestSingleResourceLeafNoGroupHeader(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			t.Fatalf("expected no group header for single resource, got %#v", r.visibleItems)
		}
	}
}

func TestSingleResourceLeafShowsGroupHeaderDuringSearch(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("web")
	r.SetSize(80, 10)

	// During search, even single resources should show group headers for context
	foundGroup := false
	for _, item := range r.visibleItems {
		if item.kind == itemGroup && item.label == consts.ModuleAlpha {
			foundGroup = true
			break
		}
	}
	if !foundGroup {
		t.Fatalf("expected group header during search, got %#v", r.visibleItems)
	}
}

func TestNestedSingleResourceShowsAllGroupsDuringSearch(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.module.edge.aws_instance.node_4", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.other", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("node_4")
	r.SetSize(80, 10)

	// Should show all nested groups: alpha, net, and edge
	foundAlpha := false
	foundNet := false
	foundEdge := false
	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			switch item.label {
			case consts.ModuleAlpha:
				foundAlpha = true
			case "module.net":
				foundNet = true
			case "module.edge":
				foundEdge = true
			}
		}
	}
	if !foundAlpha || !foundNet || !foundEdge {
		t.Fatalf("expected all nested groups during search (alpha=%v, net=%v, edge=%v), got %#v",
			foundAlpha, foundNet, foundEdge, r.visibleItems)
	}
}

func TestMixedDepthModulesRenderNestedGroups(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.module.edge.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	if indexOfGroup(r.visibleItems, "module.net") < 0 || indexOfGroup(r.visibleItems, "module.edge") < 0 {
		t.Fatalf("expected nested groups for mixed depth, got %#v", r.visibleItems)
	}
}

func TestNonModuleContextDoesNotGroup(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "example.module_value.resource", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	for _, item := range r.visibleItems {
		if item.kind == itemGroup {
			t.Fatalf("expected no grouping for non-module context, got %#v", r.visibleItems)
		}
	}
}

func TestToggleAllGroupsDeepTree(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.module.net.module.edge.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.beta.module.db.module.read.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	r.ToggleAllGroups()
	if r.allExpanded {
		t.Fatalf("expected all groups collapsed")
	}
	r.ToggleAllGroups()
	if !r.allExpanded {
		t.Fatalf("expected all groups expanded")
	}
}

func TestSearchWithGroupingCountsAndVisibility(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.web2", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.web", Action: terraform.ActionCreate},
	})
	r.SetSearchQuery("web")
	r.SetSize(80, 10)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	if groupIdx < 0 || r.visibleItems[groupIdx].count != 2 {
		t.Fatalf("expected filtered count for module.alpha, got %#v", r.visibleItems)
	}
}

func TestFilterWithGroupingCountsAndVisibility(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionDelete},
	})
	r.SetFilter(terraform.ActionDelete, false)
	r.SetSize(80, 10)

	if indexOfGroup(r.visibleItems, consts.ModuleAlpha) >= 0 {
		t.Fatalf("expected single resource without group header, got %#v", r.visibleItems)
	}
	if got := selectFirstResource(r); got == nil || got.Address != "module.alpha.aws_instance.web" {
		t.Fatalf("expected remaining resource, got %#v", got)
	}
}

func TestResourceListMoveUpAndUpdate(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(40, 5)
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
		{Address: "aws_instance.db", Action: terraform.ActionCreate},
	})

	r.MoveDown()
	if r.selectedIndex != 1 {
		t.Fatalf("expected selection to move down, got %d", r.selectedIndex)
	}
	r.MoveUp()
	if r.selectedIndex != 0 {
		t.Fatalf("expected selection to move up, got %d", r.selectedIndex)
	}

	r.selectedIndex = 1
	r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if r.selectedIndex != 0 {
		t.Fatalf("expected key update to move selection up, got %d", r.selectedIndex)
	}
}

func TestResourceListInitReturnsNil(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	if cmd := r.Init(); cmd != nil {
		t.Fatalf("expected nil init command, got %#v", cmd)
	}
}

func TestFirstResourceIndex(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.visibleItems = []listItem{
		{kind: itemGroup, label: consts.ModuleAlpha},
		{kind: itemResource, resource: &terraform.ResourceChange{Address: "aws_instance.web"}},
	}
	if idx := r.firstResourceIndex(); idx != 1 {
		t.Fatalf("expected first resource index 1, got %d", idx)
	}
}

func TestResourceListViewportOffsetOnJump(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resources := make([]terraform.ResourceChange, 0, 30)
	for i := 0; i < 30; i++ {
		resources = append(resources, terraform.ResourceChange{
			Address: fmt.Sprintf("aws_instance.%d", i),
			Action:  terraform.ActionCreate,
		})
	}
	r.SetResources(resources)
	r.SetSize(40, 5)

	for i := 0; i < 15; i++ {
		r.MoveDown()
	}
	if r.viewport.YOffset == 0 {
		t.Fatalf("expected viewport to scroll, got offset %d", r.viewport.YOffset)
	}
}

func TestToggleGroupWhileSearchRetainsSelection(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resources := []terraform.ResourceChange{
		{Address: "module.foo.aws_instance.a", Action: terraform.ActionCreate},
		{Address: "module.foo.aws_instance.b", Action: terraform.ActionCreate},
	}
	r.SetResources(resources)
	r.SetSize(60, 10)
	r.SetSearchQuery("a")

	r.MoveDown()
	selected := r.GetSelectedResource()
	if selected == nil || selected.Address != "module.foo.aws_instance.a" {
		t.Fatalf("unexpected selected resource: %#v", selected)
	}
	r.ToggleGroup()

	selectedAfter := r.GetSelectedResource()
	if selectedAfter == nil || selectedAfter.Address != "module.foo.aws_instance.a" {
		t.Fatalf("selection changed after toggle: %#v", selectedAfter)
	}
}

func TestSearchGroupingAndFilterCounts(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	resources := []terraform.ResourceChange{
		{Address: "module.foo.aws_instance.a", Action: terraform.ActionCreate},
		{Address: "module.foo.aws_instance.b", Action: terraform.ActionUpdate},
		{Address: "module.bar.aws_instance.c", Action: terraform.ActionCreate},
	}
	r.SetResources(resources)
	r.SetSize(80, 10)
	r.SetFilter(terraform.ActionUpdate, false)
	r.SetSearchQuery("module.foo")

	if len(r.visibleItems) == 0 {
		t.Fatalf("expected visible items")
	}
	group := r.visibleItems[0]
	if group.kind != itemGroup || group.count != 1 {
		t.Fatalf("expected group count 1, got %#v", group)
	}
}

func TestResourceListRefresh(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})

	// Refresh should rebuild the visible items
	r.Refresh()

	if len(r.visibleItems) == 0 {
		t.Fatal("expected visible items after refresh")
	}
}

func TestResourceListIsFocused(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())

	if r.IsFocused() {
		t.Error("expected unfocused by default")
	}

	r.SetFocused(true)
	if !r.IsFocused() {
		t.Error("expected focused after SetFocused(true)")
	}

	r.SetFocused(false)
	if r.IsFocused() {
		t.Error("expected unfocused after SetFocused(false)")
	}
}

func TestResourceListSetStyles(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	newStyles := styles.DefaultStyles()

	r.SetStyles(newStyles)

	if r.styles != newStyles {
		t.Error("expected styles to be updated")
	}
}

func TestResourceListHandleKey(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
		{Address: "aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetFocused(true)

	// Test j key (move down)
	handled, _ := r.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !handled {
		t.Error("expected j key to be handled")
	}
	if r.selectedIndex != 1 {
		t.Errorf("expected selectedIndex=1 after j, got %d", r.selectedIndex)
	}

	// Test k key (move up)
	handled, _ = r.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if !handled {
		t.Error("expected k key to be handled")
	}
	if r.selectedIndex != 0 {
		t.Errorf("expected selectedIndex=0 after k, got %d", r.selectedIndex)
	}

	// Test down arrow
	handled, _ = r.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Error("expected down key to be handled")
	}

	// Test up arrow
	handled, _ = r.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if !handled {
		t.Error("expected up key to be handled")
	}
}

func TestResourceListHandleKeyUnknown(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})

	// Unknown keys should not be handled
	handled, _ := r.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("expected unknown key 'x' to not be handled")
	}
}

func TestResourceListSetSummary(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())

	r.SetSummary(5, 3, 1, 2)

	if r.summaryCreate != 5 {
		t.Errorf("expected summaryCreate=5, got %d", r.summaryCreate)
	}
	if r.summaryUpdate != 3 {
		t.Errorf("expected summaryUpdate=3, got %d", r.summaryUpdate)
	}
	if r.summaryDelete != 1 {
		t.Errorf("expected summaryDelete=1, got %d", r.summaryDelete)
	}
	if r.summaryReplace != 2 {
		t.Errorf("expected summaryReplace=2, got %d", r.summaryReplace)
	}
}

func TestResourceListRenderSummaryHeader(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	r.summaryCreate = 2
	r.summaryUpdate = 1
	r.summaryDelete = 0
	r.summaryReplace = 1

	header := r.renderSummaryHeader()
	if header == "" {
		t.Error("expected non-empty summary header")
	}
}

func TestResourceListGetSelectedIndex(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
		{Address: "aws_instance.db", Action: terraform.ActionCreate},
	})

	if r.GetSelectedIndex() != 0 {
		t.Errorf("expected initial index=0, got %d", r.GetSelectedIndex())
	}

	r.MoveDown()
	if r.GetSelectedIndex() != 1 {
		t.Errorf("expected index=1 after MoveDown, got %d", r.GetSelectedIndex())
	}
}

func TestResourceListItemCount(t *testing.T) {
	s := styles.DefaultStyles()
	r := NewResourceList(s)
	r.SetSize(80, 20)

	// Empty list
	if r.ItemCount() != 0 {
		t.Errorf("expected 0 items, got %d", r.ItemCount())
	}

	// With resources
	resources := []terraform.ResourceChange{
		{Address: "a", Action: terraform.ActionCreate},
		{Address: "b", Action: terraform.ActionUpdate},
		{Address: "c", Action: terraform.ActionDelete},
	}
	r.SetResources(resources)

	if r.ItemCount() != 3 {
		t.Errorf("expected 3 items, got %d", r.ItemCount())
	}
}

func TestFormatShortDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, ""},
		{"negative", -1 * time.Second, ""},
		{"seconds", 30 * time.Second, "30s"},
		{"one second", 1 * time.Second, "1s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"one minute", 60 * time.Second, "1m"},
		{"minutes", 5 * time.Minute, "5m"},
		{"59 minutes", 59 * time.Minute, "59m"},
		{"one hour", 60 * time.Minute, "1h"},
		{"hours", 3 * time.Hour, "3h"},
		{"mixed", 90 * time.Second, "1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatShortDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatShortDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestGetStatusStyle(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())

	tests := []struct {
		name   string
		status terraform.OperationStatus
	}{
		{"pending", terraform.StatusPending},
		{"in progress", terraform.StatusInProgress},
		{"complete", terraform.StatusComplete},
		{"errored", terraform.StatusErrored},
		{"unknown", terraform.OperationStatus("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic and returns a style
			style := r.getStatusStyle(tt.status)
			_ = style.Render("test")
		})
	}
}

func TestResourceMatchesQuery(t *testing.T) {
	tests := []struct {
		name     string
		resource terraform.ResourceChange
		query    string
		expected bool
	}{
		{
			name:     "empty query matches all",
			resource: terraform.ResourceChange{Address: "anything"},
			query:    "",
			expected: true,
		},
		{
			name:     "match by address",
			resource: terraform.ResourceChange{Address: "aws_instance.web"},
			query:    "web",
			expected: true,
		},
		{
			name:     "match by resource type",
			resource: terraform.ResourceChange{Address: "aws_instance.web", ResourceType: "aws_instance"},
			query:    "aws_instance",
			expected: true,
		},
		{
			name:     "match by resource name",
			resource: terraform.ResourceChange{Address: "aws_instance.web", ResourceName: "web"},
			query:    "web",
			expected: true,
		},
		{
			name:     "no match",
			resource: terraform.ResourceChange{Address: "aws_instance.web"},
			query:    "xyz",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resourceMatchesQuery(tc.resource, tc.query)
			if result != tc.expected {
				t.Errorf("resourceMatchesQuery(%v, %q) = %v, want %v",
					tc.resource.Address, tc.query, result, tc.expected)
			}
		})
	}
}

func TestFuzzyMatchEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		candidate   string
		shouldMatch bool
	}{
		{"empty query", "", "anything", true},
		{"query longer than candidate", "abcdef", "abc", false},
		{"exact match", "abc", "abc", true},
		{"partial match at end", "bc", "abc", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := fuzzyMatch(tc.query, tc.candidate)
			if result != tc.shouldMatch {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v",
					tc.query, tc.candidate, result, tc.shouldMatch)
			}
		})
	}
}

func TestFirstResourceIndexEmpty(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	// Empty visible items
	r.visibleItems = []listItem{}
	if idx := r.firstResourceIndex(); idx != -1 {
		t.Errorf("expected -1 for empty items, got %d", idx)
	}
}

func TestFirstResourceIndexOnlyGroups(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.visibleItems = []listItem{
		{kind: itemGroup, label: "group1"},
		{kind: itemGroup, label: "group2"},
	}
	if idx := r.firstResourceIndex(); idx != -1 {
		t.Errorf("expected -1 when no resources, got %d", idx)
	}
}

func TestResourceListUpdateWithEnterKey(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
	})

	// Find and select a group
	for i, item := range r.visibleItems {
		if item.kind == itemGroup {
			r.selectedIndex = i
			break
		}
	}

	// Press enter to toggle group
	r.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Should have toggled the group expansion
}

func TestResourceListUpdateWithSpaceKey(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
	})

	// Find and select a group
	for i, item := range r.visibleItems {
		if item.kind == itemGroup {
			r.selectedIndex = i
			break
		}
	}

	// Press space to toggle group
	r.Update(tea.KeyMsg{Type: tea.KeySpace})
}

func TestResourceListUpdateViewportMessage(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
	})

	// Send a non-key message that should be passed to viewport
	model, cmd := r.Update(tea.MouseMsg{})
	if model == nil {
		t.Error("expected non-nil model")
	}
	_ = cmd
}

func TestRenderMultilineDiff(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)

	block := "~ change\n- remove\n+ add\n  neutral"
	result := r.renderMultilineDiff(block)
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderMultilineDiffZeroWidth(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.width = 0

	block := "~ change\n- remove"
	result := r.renderMultilineDiff(block)
	if result == "" {
		t.Error("expected non-empty result even with zero width")
	}
}

func TestToggleAllGroupsEmptyGroups(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	// No resources - no groups
	r.SetResources(nil)

	// Should not panic
	r.ToggleAllGroups()
}

func TestComputeAllExpandedEmpty(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	// Empty groupExpanded map
	r.groupExpanded = map[string]bool{}
	result := r.computeAllExpanded()
	if result {
		t.Error("expected false for empty groups")
	}
}

func TestComputeAllExpandedPartial(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.groupExpanded = map[string]bool{
		"group1": true,
		"group2": false,
	}
	result := r.computeAllExpanded()
	if result {
		t.Error("expected false when some groups collapsed")
	}
}

func TestComputeAllExpandedAllExpanded(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.groupExpanded = map[string]bool{
		"group1": true,
		"group2": true,
	}
	result := r.computeAllExpanded()
	if !result {
		t.Error("expected true when all groups expanded")
	}
}

func TestResourceMatchesQueryWithResourceType(t *testing.T) {
	resource := terraform.ResourceChange{
		Address:      "aws_instance.web",
		ResourceType: "aws_instance",
		ResourceName: "web",
	}

	// Test matching on ResourceType
	if !resourceMatchesQuery(resource, "instance") {
		t.Error("expected query to match ResourceType")
	}
}

func TestResourceMatchesQueryWithResourceName(t *testing.T) {
	resource := terraform.ResourceChange{
		Address:      "aws_instance.my_web_server",
		ResourceType: "aws_instance",
		ResourceName: "my_web_server",
	}

	// Test matching on ResourceName
	if !resourceMatchesQuery(resource, "webserver") {
		t.Error("expected query to match ResourceName via fuzzy match")
	}
}

func TestResourceMatchesQueryEmptyResourceTypeAndName(t *testing.T) {
	resource := terraform.ResourceChange{
		Address:      "aws_instance.web",
		ResourceType: "",
		ResourceName: "",
	}

	// Should still match on address
	if !resourceMatchesQuery(resource, "aws") {
		t.Error("expected query to match address when ResourceType and ResourceName are empty")
	}
	// Should not match non-matching query
	if resourceMatchesQuery(resource, "xyz") {
		t.Error("expected non-matching query to return false")
	}
}

func TestHandleKeySpace(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)

	// Select the group
	r.selectedIndex = 0

	// Press space to toggle
	msg := tea.KeyMsg{Type: tea.KeySpace}
	handled, _ := r.HandleKey(msg)
	if !handled {
		t.Error("expected space key to be handled")
	}
}

func TestHandleKeyUnhandled(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)

	// Press an unhandled key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	handled, _ := r.HandleKey(msg)
	if handled {
		t.Error("expected unhandled key to return false")
	}
}

func TestToggleTargetSelectionSingleResource(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})
	r.SetSize(80, 10)
	r.SetTargetModeEnabled(true)

	if ok := r.ToggleTargetSelectionAtSelected(); !ok {
		t.Fatal("expected toggle to succeed")
	}
	targets := r.SelectedTargets()
	if len(targets) != 1 || targets[0] != "aws_instance.web" {
		t.Fatalf("unexpected targets: %#v", targets)
	}

	if ok := r.ToggleTargetSelectionAtSelected(); !ok {
		t.Fatal("expected second toggle to succeed")
	}
	if got := r.SelectedTargets(); len(got) != 0 {
		t.Fatalf("expected target list to be empty, got %#v", got)
	}
}

func TestToggleTargetSelectionOnGroupTogglesVisibleDescendants(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.module.net.aws_instance.db", Action: terraform.ActionCreate},
		{Address: "module.beta.aws_instance.cache", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 12)
	r.SetTargetModeEnabled(true)

	groupIdx := indexOfGroup(r.visibleItems, consts.ModuleAlpha)
	if groupIdx < 0 {
		t.Fatal("expected module.alpha group")
	}
	r.selectedIndex = groupIdx

	if ok := r.ToggleTargetSelectionAtSelected(); !ok {
		t.Fatal("expected group toggle to succeed")
	}
	targets := r.SelectedTargets()
	if len(targets) != 2 {
		t.Fatalf("expected two descendants selected, got %#v", targets)
	}
	if targets[0] != "module.alpha.aws_instance.web" && targets[1] != "module.alpha.aws_instance.web" {
		t.Fatalf("expected alpha web target, got %#v", targets)
	}
	if targets[0] != "module.alpha.module.net.aws_instance.db" && targets[1] != "module.alpha.module.net.aws_instance.db" {
		t.Fatalf("expected nested target, got %#v", targets)
	}
}

func TestTargetSelectionMarkersRenderInTargetMode(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.web", Action: terraform.ActionCreate},
		{Address: "module.alpha.aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)
	r.SetTargetModeEnabled(true)

	r.selectedIndex = 0
	if ok := r.ToggleTargetSelectionAtSelected(); !ok {
		t.Fatal("expected target toggle")
	}

	view := r.View()
	if strings.Contains(view, "[x]") || strings.Contains(view, "[-]") || strings.Contains(view, "[ ]") {
		t.Fatalf("expected no checkbox markers in target mode view, got %q", view)
	}
}

func TestToggleAllTargetSelectionVisibleResources(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetResources([]terraform.ResourceChange{
		{Address: "aws_instance.web", Action: terraform.ActionCreate},
		{Address: "aws_instance.db", Action: terraform.ActionCreate},
	})
	r.SetSize(80, 10)
	r.SetTargetModeEnabled(true)

	if ok := r.ToggleAllTargetSelection(); !ok {
		t.Fatal("expected toggle all to succeed")
	}
	if got := r.SelectedTargets(); len(got) != 2 {
		t.Fatalf("expected two selected targets, got %#v", got)
	}

	if ok := r.ToggleAllTargetSelection(); !ok {
		t.Fatal("expected second toggle all to succeed")
	}
	if got := r.SelectedTargets(); len(got) != 0 {
		t.Fatalf("expected no selected targets after second toggle, got %#v", got)
	}
}

func TestUpdateWithViewportMessage(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)

	// Send a message that's not a KeyMsg
	_, _ = r.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	// Should not panic
}

func TestTargetPlanPreviewRendersDuringTargetedPlan(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(60, 10)
	r.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})

	r.SetTargetPlanPreview([]string{"module.alpha.aws_instance.web", "module.alpha.aws_instance.db"}, true)
	view := r.View()

	if !strings.Contains(view, "Targeted plan running for:") {
		t.Fatalf("expected targeted plan preview header, got %q", view)
	}
	if !strings.Contains(view, "module.alpha.aws_instance.web") {
		t.Fatalf("expected first target in preview, got %q", view)
	}
	if !strings.Contains(view, "module.alpha.aws_instance.db") {
		t.Fatalf("expected second target in preview, got %q", view)
	}
}

func TestTargetPlanPreviewSuppressesNoResourcesText(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 10)
	r.SetResources(nil)

	r.SetTargetPlanPreview([]string{"module.foundation.module.ingress.helm_release.release"}, true)

	view := r.View()
	if strings.Contains(view, "No resources to display") {
		t.Fatalf("expected no empty-state line while target preview is active, got %q", view)
	}
}

func TestWrapTargetPreviewLineWrapsLongTargets(t *testing.T) {
	lines := wrapTargetPreviewLine("module.foundation.module.ingress.helm_release.release", 24)
	if len(lines) < 2 {
		t.Fatalf("expected wrapped lines, got %#v", lines)
	}
	if !strings.HasPrefix(lines[0], "• ") {
		t.Fatalf("expected first wrapped line to have bullet prefix, got %#v", lines[0])
	}
	if !strings.HasPrefix(lines[1], "  ") {
		t.Fatalf("expected continuation line to be indented, got %#v", lines[1])
	}
}

func TestTargetPlanPreviewClearsWhenInactive(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(100, 10)
	r.SetResources([]terraform.ResourceChange{{Address: "aws_instance.web", Action: terraform.ActionCreate}})

	r.SetTargetPlanPreview([]string{"module.alpha.aws_instance.web"}, true)
	withPreview := r.View()
	if !strings.Contains(withPreview, "Targeted plan running for:") {
		t.Fatalf("expected preview header before clearing, got %q", withPreview)
	}

	r.SetTargetPlanPreview(nil, false)
	withoutPreview := r.View()
	if strings.Contains(withoutPreview, "Targeted plan running for:") {
		t.Fatalf("expected preview header to be cleared, got %q", withoutPreview)
	}
}
