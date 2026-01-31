package utils

import (
	"regexp"
	"strings"
	"time"

	"github.com/ushiradineth/lazytf/internal/history"
)

// SectionType represents the type of terraform output section.
type SectionType int

const (
	SectionProviderInfo    SectionType = iota // "Terraform used the selected providers..."
	SectionLegend                             // "+ create", "- destroy" symbol explanations
	SectionResourceChanges                    // "# resource.name will be created" + HCL block
	SectionPlanSummary                        // "Plan: X to add, Y to change, Z to destroy"
	SectionApplyProgress                      // "Creating...", "Still creating..."
	SectionApplyComplete                      // "Apply complete! Resources: ..."
	SectionSavedPlan                          // "Saved the plan to: ..."
	SectionRefreshInfo                        // "Refreshing state..." lines
	SectionOther                              // Unclassified content
)

// Section represents a parsed section of terraform output.
type Section struct {
	Type    SectionType
	Content []string
}

// LogMetadata contains metadata for a history log entry.
type LogMetadata struct {
	Status      history.Status
	StartedAt   time.Time
	FinishedAt  time.Time
	Duration    time.Duration
	Environment string
	WorkDir     string
}

// Regular expressions for detecting section types.
var (
	providerInfoPattern          = regexp.MustCompile(`(?i)^Terraform (used|has created|performed)`)
	refreshPattern               = regexp.MustCompile(`(?i)^(Refreshing|Acquiring) (state|lock)`)
	legendPattern                = regexp.MustCompile(`^\s*[+~-]\s+(create|destroy|update|replace|read)`)
	resourceChangePattern        = regexp.MustCompile(`^\s*#\s+\S+`)
	planSummaryPattern           = regexp.MustCompile(`^Plan:\s+\d+\s+to\s+(add|create)`)
	noChangesPattern             = regexp.MustCompile(`^No changes`)
	applyProgressPattern         = regexp.MustCompile(`(?i)^\S+:\s+(Creating|Modifying|Destroying|Still (creating|modifying|destroying))`)
	applyCompletePattern         = regexp.MustCompile(`^Apply complete!`)
	savedPlanPattern             = regexp.MustCompile(`^Saved the plan to:`)
	creationCompletePattern      = regexp.MustCompile(`(?i):\s+Creation complete`)
	destructionCompletePattern   = regexp.MustCompile(`(?i):\s+Destruction(s)?\s+complete`)
	modificationsCompletePattern = regexp.MustCompile(`(?i):\s+Modifications?\s+complete`)
)

// ParseTerraformOutput parses terraform output into logical sections.
func ParseTerraformOutput(output string) []Section {
	lines := strings.Split(output, "\n")
	var sections []Section
	var currentSection *Section

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines at the start of a section
		if trimmed == "" {
			if currentSection != nil && len(currentSection.Content) > 0 {
				currentSection.Content = append(currentSection.Content, line)
			}
			continue
		}

		sectionType := detectSectionType(trimmed)

		// If we detect a new section type (or first line), finalize current section
		if currentSection == nil || shouldStartNewSection(currentSection.Type, sectionType, trimmed) {
			if currentSection != nil {
				sections = append(sections, *currentSection)
			}
			currentSection = &Section{
				Type:    sectionType,
				Content: []string{line},
			}
		} else {
			currentSection.Content = append(currentSection.Content, line)
		}
	}

	// Append final section
	if currentSection != nil {
		sections = append(sections, *currentSection)
	}

	return sections
}

// detectSectionType determines the section type based on line content.
func detectSectionType(line string) SectionType {
	switch {
	case providerInfoPattern.MatchString(line):
		return SectionProviderInfo
	case refreshPattern.MatchString(line):
		return SectionRefreshInfo
	case legendPattern.MatchString(line):
		return SectionLegend
	case resourceChangePattern.MatchString(line):
		return SectionResourceChanges
	case planSummaryPattern.MatchString(line) || noChangesPattern.MatchString(line):
		return SectionPlanSummary
	case applyProgressPattern.MatchString(line):
		return SectionApplyProgress
	case creationCompletePattern.MatchString(line):
		return SectionApplyProgress
	case destructionCompletePattern.MatchString(line):
		return SectionApplyProgress
	case modificationsCompletePattern.MatchString(line):
		return SectionApplyProgress
	case applyCompletePattern.MatchString(line):
		return SectionApplyComplete
	case savedPlanPattern.MatchString(line):
		return SectionSavedPlan
	default:
		return SectionOther
	}
}

// shouldStartNewSection determines if a new section should be started.
func shouldStartNewSection(currentType, newType SectionType, line string) bool {
	// Always start new section for major type changes
	if newType != currentType && newType != SectionOther {
		// Keep consecutive apply progress lines together
		if currentType == SectionApplyProgress && newType == SectionApplyProgress {
			return false
		}
		return true
	}

	// Start new section for each resource change block (# comment)
	if resourceChangePattern.MatchString(line) {
		return true
	}

	return false
}

// FormatWithSpacing adds visual spacing between sections.
func FormatWithSpacing(sections []Section) string {
	var builder strings.Builder
	var lastType SectionType = -1

	for _, section := range sections {
		// Add spacing between different section types
		if lastType >= 0 && shouldAddSpacing(lastType, section.Type) {
			builder.WriteString("\n")
		}

		// Write section content
		for _, line := range section.Content {
			builder.WriteString(line)
			builder.WriteString("\n")
		}

		lastType = section.Type
	}

	return strings.TrimRight(builder.String(), "\n")
}

// shouldAddSpacing determines if spacing should be added between section types.
func shouldAddSpacing(prevType, currType SectionType) bool {
	// Add spacing before major sections
	switch currType {
	case SectionResourceChanges:
		return true
	case SectionPlanSummary:
		return true
	case SectionApplyProgress:
		return prevType != SectionApplyProgress
	case SectionApplyComplete:
		return true
	case SectionProviderInfo, SectionLegend, SectionSavedPlan, SectionRefreshInfo, SectionOther:
		return false
	}
	return false
}

// FormatMetadataHeader creates a formatted metadata header.
func FormatMetadataHeader(metadata *LogMetadata) string {
	if metadata == nil {
		return ""
	}

	var builder strings.Builder

	// Status line with icon
	statusIcon := getStatusIcon(metadata.Status)
	statusText := getStatusText(metadata.Status)
	builder.WriteString(statusIcon + " " + statusText + "\n")

	// Time and duration
	timeStr := metadata.StartedAt.Format("2006-01-02 15:04:05")
	durationStr := formatDuration(metadata.Duration)
	builder.WriteString("Time: " + timeStr + " (" + durationStr + ")\n")

	// Environment
	if metadata.Environment != "" {
		builder.WriteString("Environment: " + metadata.Environment + "\n")
	}

	// Working directory
	if metadata.WorkDir != "" {
		builder.WriteString("Directory: " + metadata.WorkDir + "\n")
	}

	return builder.String()
}

// FormatSectionSeparator creates a visual separator with title.
func FormatSectionSeparator(title string, width int) string {
	if width <= 0 {
		width = 60
	}

	// Calculate padding for centered title
	titleLen := len(title) + 2 // +2 for spaces around title
	if titleLen >= width {
		return "── " + title + " ──"
	}

	sideLen := (width - titleLen) / 2
	leftSide := strings.Repeat("─", sideLen)
	rightSide := strings.Repeat("─", width-sideLen-titleLen)

	return leftSide + " " + title + " " + rightSide
}

// FormatCombinedOutput combines plan and apply outputs with separators.
func FormatCombinedOutput(metadata *LogMetadata, planOutput, applyOutput string, width int) string {
	var builder strings.Builder

	// Metadata header
	if metadata != nil {
		builder.WriteString(FormatMetadataHeader(metadata))
		builder.WriteString("\n")
	}

	// Plan output section
	if strings.TrimSpace(planOutput) != "" {
		builder.WriteString(FormatSectionSeparator("Plan Output", width))
		builder.WriteString("\n\n")

		planSections := ParseTerraformOutput(planOutput)
		builder.WriteString(FormatWithSpacing(planSections))
		builder.WriteString("\n\n")
	}

	// Apply output section
	if strings.TrimSpace(applyOutput) != "" {
		builder.WriteString(FormatSectionSeparator("Apply Output", width))
		builder.WriteString("\n\n")

		applySections := ParseTerraformOutput(applyOutput)
		builder.WriteString(FormatWithSpacing(applySections))
	}

	return strings.TrimRight(builder.String(), "\n")
}

// getStatusIcon returns an icon for the status.
func getStatusIcon(status history.Status) string {
	switch status {
	case history.StatusSuccess:
		return "●"
	case history.StatusFailed:
		return "✗"
	case history.StatusCanceled:
		return "○"
	default:
		return "?"
	}
}

// getStatusText returns the text representation of status.
func getStatusText(status history.Status) string {
	switch status {
	case history.StatusSuccess:
		return "Success"
	case history.StatusFailed:
		return "Failed"
	case history.StatusCanceled:
		return "Canceled"
	default:
		return "Unknown"
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	return d.Round(time.Second).String()
}
