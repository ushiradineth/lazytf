package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/ushiradineth/lazytf/internal/history"
)

func TestParseTerraformOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // Number of sections
	}{
		{
			name:     "empty input",
			input:    "",
			expected: 0,
		},
		{
			name:     "single plan summary",
			input:    "Plan: 1 to add, 0 to change, 0 to destroy",
			expected: 1,
		},
		{
			name: "resource changes with summary",
			input: `# null_resource.example will be created
  + resource "null_resource" "example" {
      + id = (known after apply)
    }

Plan: 1 to add, 0 to change, 0 to destroy`,
			expected: 2,
		},
		{
			name: "apply progress",
			input: `null_resource.example: Creating...
null_resource.example: Creation complete after 1s [id=123]`,
			expected: 1, // Consecutive progress lines stay together
		},
		{
			name:     "apply complete",
			input:    `Apply complete! Resources: 1 added, 0 changed, 0 destroyed.`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sections := ParseTerraformOutput(tt.input)
			if len(sections) != tt.expected {
				t.Errorf("expected %d sections, got %d", tt.expected, len(sections))
			}
		})
	}
}

func TestDetectSectionType(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected SectionType
	}{
		{"provider info", "Terraform used the selected providers", SectionProviderInfo},
		{"refresh state", "Refreshing state...", SectionRefreshInfo},
		{"legend create", "+ create", SectionLegend},
		{"legend destroy", "- destroy", SectionLegend},
		{"resource comment", "# null_resource.example will be created", SectionResourceChanges},
		{"plan summary", "Plan: 1 to add, 0 to change, 0 to destroy", SectionPlanSummary},
		{"no changes", "No changes. Infrastructure is up-to-date.", SectionPlanSummary},
		{"creating", "null_resource.example: Creating...", SectionApplyProgress},
		{"creation complete", "null_resource.example: Creation complete [id=123]", SectionApplyProgress},
		{"apply complete", "Apply complete! Resources: 1 added, 0 changed, 0 destroyed.", SectionApplyComplete},
		{"saved plan", "Saved the plan to: plan.out", SectionSavedPlan},
		{"other", "Some random line", SectionOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectSectionType(tt.line)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatWithSpacing(t *testing.T) {
	sections := []Section{
		{Type: SectionProviderInfo, Content: []string{"Terraform used providers"}},
		{Type: SectionResourceChanges, Content: []string{"# resource.name will be created"}},
		{Type: SectionPlanSummary, Content: []string{"Plan: 1 to add"}},
	}

	result := FormatWithSpacing(sections)

	// Should have newlines between sections
	if !strings.Contains(result, "\n\n") {
		t.Error("expected spacing between sections")
	}
}

func TestFormatMetadataHeader(t *testing.T) {
	metadata := &LogMetadata{
		Status:      history.StatusSuccess,
		StartedAt:   time.Date(2024, 1, 30, 10, 15, 0, 0, time.Local),
		FinishedAt:  time.Date(2024, 1, 30, 10, 15, 45, 0, time.Local),
		Duration:    45 * time.Second,
		Environment: "production",
		WorkDir:     "/path/to/terraform",
	}

	result := FormatMetadataHeader(metadata)

	if !strings.Contains(result, "● Success") {
		t.Error("expected success status icon")
	}
	if !strings.Contains(result, "2024-01-30") {
		t.Error("expected date in header")
	}
	if !strings.Contains(result, "45s") {
		t.Error("expected duration in header")
	}
	if !strings.Contains(result, "Environment: production") {
		t.Error("expected environment in header")
	}
	if !strings.Contains(result, "Directory: /path/to/terraform") {
		t.Error("expected directory in header")
	}
}

func TestFormatMetadataHeaderNil(t *testing.T) {
	result := FormatMetadataHeader(nil)
	if result != "" {
		t.Error("expected empty string for nil metadata")
	}
}

func TestFormatSectionSeparator(t *testing.T) {
	result := FormatSectionSeparator("Plan Output", 40)

	if !strings.Contains(result, "Plan Output") {
		t.Error("expected title in separator")
	}
	if !strings.Contains(result, "─") {
		t.Error("expected separator character")
	}
}

func TestFormatCombinedOutput(t *testing.T) {
	metadata := &LogMetadata{
		Status:      history.StatusSuccess,
		StartedAt:   time.Now().Add(-1 * time.Minute),
		FinishedAt:  time.Now(),
		Duration:    1 * time.Minute,
		Environment: "dev",
	}

	planOutput := `# null_resource.example will be created
Plan: 1 to add, 0 to change, 0 to destroy`

	applyOutput := `null_resource.example: Creating...
null_resource.example: Creation complete [id=123]
Apply complete! Resources: 1 added, 0 changed, 0 destroyed.`

	result := FormatCombinedOutput(metadata, planOutput, applyOutput, 60)

	if !strings.Contains(result, "● Success") {
		t.Error("expected status header")
	}
	if !strings.Contains(result, "Plan Output") {
		t.Error("expected plan output section")
	}
	if !strings.Contains(result, "Apply Output") {
		t.Error("expected apply output section")
	}
	if !strings.Contains(result, "null_resource.example") {
		t.Error("expected resource name in output")
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   history.Status
		expected string
	}{
		{history.StatusSuccess, "●"},
		{history.StatusFailed, "✗"},
		{history.StatusCanceled, "○"},
		{history.Status("unknown"), "?"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := getStatusIcon(tt.status)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		contains string
	}{
		{500 * time.Millisecond, "ms"},
		{45 * time.Second, "45s"},
		{2 * time.Minute, "2m"},
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			result := formatDuration(tt.duration)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected %s to contain %s", result, tt.contains)
			}
		})
	}
}
