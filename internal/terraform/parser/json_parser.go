package parser

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

// JSONParser parses Terraform JSON plan output
type JSONParser struct{}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse reads and parses JSON plan data from a reader
func (p *JSONParser) Parse(input io.Reader) (*terraform.Plan, error) {
	var rawPlan struct {
		FormatVersion    string                     `json:"format_version"`
		TerraformVersion string                     `json:"terraform_version"`
		ResourceChanges  []terraform.ResourceChange `json:"resource_changes"`
		OutputChanges    map[string]struct {
			Actions   []string `json:"actions"`
			Before    any      `json:"before"`
			After     any      `json:"after"`
			Sensitive bool     `json:"sensitive"`
		} `json:"output_changes"`
		Variables map[string]any `json:"variables"`
	}

	decoder := json.NewDecoder(input)
	if err := decoder.Decode(&rawPlan); err != nil {
		return nil, fmt.Errorf("failed to decode JSON plan: %w", err)
	}

	plan := &terraform.Plan{
		FormatVersion: rawPlan.FormatVersion,
		Resources:     make([]terraform.ResourceChange, 0, len(rawPlan.ResourceChanges)),
		OutputChanges: make([]terraform.OutputChange, 0, len(rawPlan.OutputChanges)),
		Variables:     rawPlan.Variables,
		Metadata: terraform.PlanMetadata{
			TerraformVersion: rawPlan.TerraformVersion,
		},
	}

	// Process resource changes
	for _, rc := range rawPlan.ResourceChanges {
		// Only include resources with actual changes
		if rc.Change == nil {
			continue
		}

		// Determine action type from actions array
		rc.Action = terraform.GetActionType(rc.Change.Actions)

		// Skip no-op resources unless explicitly needed
		if rc.Action == terraform.ActionNoOp {
			continue
		}

		plan.Resources = append(plan.Resources, rc)
	}

	// Process output changes
	for name, change := range rawPlan.OutputChanges {
		outputChange := terraform.OutputChange{
			Name:   name,
			Action: terraform.GetActionType(change.Actions),
			Change: &terraform.OutputChangeDetail{
				Actions: change.Actions,
				Before:  change.Before,
				After:   change.After,
			},
			Sensitive: change.Sensitive,
		}
		plan.OutputChanges = append(plan.OutputChanges, outputChange)
	}

	return plan, nil
}

// ParseFile is a convenience method to parse a JSON file from a path
func (p *JSONParser) ParseFile(filePath string) (*terraform.Plan, error) {
	file, err := openFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return p.Parse(file)
}

// ParseBytes parses JSON plan data from a byte slice
func (p *JSONParser) ParseBytes(data []byte) (*terraform.Plan, error) {
	var rawPlan struct {
		FormatVersion    string                     `json:"format_version"`
		TerraformVersion string                     `json:"terraform_version"`
		ResourceChanges  []terraform.ResourceChange `json:"resource_changes"`
		OutputChanges    map[string]struct {
			Actions   []string `json:"actions"`
			Before    any      `json:"before"`
			After     any      `json:"after"`
			Sensitive bool     `json:"sensitive"`
		} `json:"output_changes"`
		Variables map[string]any `json:"variables"`
	}

	if err := json.Unmarshal(data, &rawPlan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON plan: %w", err)
	}

	plan := &terraform.Plan{
		FormatVersion: rawPlan.FormatVersion,
		Resources:     make([]terraform.ResourceChange, 0, len(rawPlan.ResourceChanges)),
		OutputChanges: make([]terraform.OutputChange, 0, len(rawPlan.OutputChanges)),
		Variables:     rawPlan.Variables,
		Metadata: terraform.PlanMetadata{
			TerraformVersion: rawPlan.TerraformVersion,
		},
	}

	// Process resource changes
	for _, rc := range rawPlan.ResourceChanges {
		if rc.Change == nil {
			continue
		}

		rc.Action = terraform.GetActionType(rc.Change.Actions)

		if rc.Action == terraform.ActionNoOp {
			continue
		}

		plan.Resources = append(plan.Resources, rc)
	}

	// Process output changes
	for name, change := range rawPlan.OutputChanges {
		outputChange := terraform.OutputChange{
			Name:   name,
			Action: terraform.GetActionType(change.Actions),
			Change: &terraform.OutputChangeDetail{
				Actions: change.Actions,
				Before:  change.Before,
				After:   change.After,
			},
			Sensitive: change.Sensitive,
		}
		plan.OutputChanges = append(plan.OutputChanges, outputChange)
	}

	return plan, nil
}
