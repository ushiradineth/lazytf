package terraform

import (
	"bytes"
	"encoding/json"
	"time"
)

// ActionType represents the type of action Terraform will take on a resource
type ActionType string

const (
	ActionCreate  ActionType = "create"
	ActionUpdate  ActionType = "update"
	ActionDelete  ActionType = "delete"
	ActionReplace ActionType = "replace"
	ActionNoOp    ActionType = "no-op"
	ActionRead    ActionType = "read"
)

// Plan represents a parsed Terraform plan
type Plan struct {
	FormatVersion string           `json:"format_version"`
	Resources     []ResourceChange `json:"resource_changes"`
	OutputChanges []OutputChange   `json:"output_changes"`
	Variables     map[string]any   `json:"variables"`
	Metadata      PlanMetadata     `json:"metadata"`
}

// ModuleAddress supports Terraform's module_address being a string or array.
type ModuleAddress []string

// UnmarshalJSON accepts module_address as a string or []string.
func (m *ModuleAddress) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*m = nil
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		if asString == "" {
			*m = nil
			return nil
		}
		*m = ModuleAddress{asString}
		return nil
	}

	var asSlice []string
	if err := json.Unmarshal(data, &asSlice); err != nil {
		return err
	}
	*m = ModuleAddress(asSlice)
	return nil
}

// ResourceChange represents a single resource change in the plan
type ResourceChange struct {
	Address      string        `json:"address"`
	ModulePath   ModuleAddress `json:"module_address,omitempty"`
	Mode         string        `json:"mode"`
	ResourceType string        `json:"type"`
	ResourceName string        `json:"name"`
	ProviderName string        `json:"provider_name"`
	Action       ActionType    `json:"-"` // Computed from change.actions
	ActionReason string        `json:"action_reason,omitempty"`
	Change       *Change       `json:"change"`
}

// Change represents the before/after state of a resource
type Change struct {
	Actions           []string            `json:"actions"`
	Before            map[string]any      `json:"before"`
	After             map[string]any      `json:"after"`
	AfterUnknown      map[string]any      `json:"after_unknown,omitempty"`
	BeforeSensitive   map[string]any      `json:"before_sensitive,omitempty"`
	AfterSensitive    map[string]any      `json:"after_sensitive,omitempty"`
	ReplacePaths      [][]string          `json:"replace_paths,omitempty"`
	BeforeOrder       map[string][]string `json:"-"`
	AfterOrder        map[string][]string `json:"-"`
	AfterUnknownOrder map[string][]string `json:"-"`
}

// UnmarshalJSON captures key order for before/after/after_unknown maps.
func (c *Change) UnmarshalJSON(data []byte) error {
	type changeAlias struct {
		Actions         []string        `json:"actions"`
		Before          json.RawMessage `json:"before"`
		After           json.RawMessage `json:"after"`
		AfterUnknown    json.RawMessage `json:"after_unknown"`
		BeforeSensitive map[string]any  `json:"before_sensitive"`
		AfterSensitive  map[string]any  `json:"after_sensitive"`
		ReplacePaths    [][]string      `json:"replace_paths"`
	}

	var aux changeAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.Actions = aux.Actions
	c.BeforeSensitive = aux.BeforeSensitive
	c.AfterSensitive = aux.AfterSensitive
	c.ReplacePaths = aux.ReplacePaths

	if len(aux.Before) > 0 && string(aux.Before) != "null" {
		if err := json.Unmarshal(aux.Before, &c.Before); err != nil {
			return err
		}
		c.BeforeOrder = buildOrderMap(aux.Before)
	}
	if len(aux.After) > 0 && string(aux.After) != "null" {
		if err := json.Unmarshal(aux.After, &c.After); err != nil {
			return err
		}
		c.AfterOrder = buildOrderMap(aux.After)
	}
	if len(aux.AfterUnknown) > 0 && string(aux.AfterUnknown) != "null" {
		if err := json.Unmarshal(aux.AfterUnknown, &c.AfterUnknown); err != nil {
			return err
		}
		c.AfterUnknownOrder = buildOrderMap(aux.AfterUnknown)
	}

	return nil
}

func buildOrderMap(raw json.RawMessage) map[string][]string {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil {
		return nil
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return nil
	}

	order := make(map[string][]string)
	if err := parseObjectOrder(decoder, "", order); err != nil {
		return nil
	}
	return order
}

func parseObjectOrder(decoder *json.Decoder, path string, order map[string][]string) error {
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, _ := keyToken.(string)
		order[path] = append(order[path], key)

		valueToken, err := decoder.Token()
		if err != nil {
			return err
		}
		if delim, ok := valueToken.(json.Delim); ok {
			switch delim {
			case '{':
				childPath := path + "/" + escapeJSONPointer(key)
				if err := parseObjectOrder(decoder, childPath, order); err != nil {
					return err
				}
			case '[':
				if err := skipArray(decoder); err != nil {
					return err
				}
			}
		}
	}
	_, err := decoder.Token()
	return err
}

func skipArray(decoder *json.Decoder) error {
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if err := parseObjectOrder(decoder, "", make(map[string][]string)); err != nil {
					return err
				}
			case '[':
				if err := skipArray(decoder); err != nil {
					return err
				}
			}
		}
	}
	_, err := decoder.Token()
	return err
}

func escapeJSONPointer(segment string) string {
	escaped := bytes.ReplaceAll([]byte(segment), []byte("~"), []byte("~0"))
	escaped = bytes.ReplaceAll(escaped, []byte("/"), []byte("~1"))
	return string(escaped)
}

// OutputChange represents a change to a Terraform output
type OutputChange struct {
	Name      string              `json:"name"`
	Action    ActionType          `json:"-"`
	Change    *OutputChangeDetail `json:"change"`
	Sensitive bool                `json:"sensitive"`
}

// OutputChangeDetail contains the before/after values of an output
type OutputChangeDetail struct {
	Actions []string `json:"actions"`
	Before  any      `json:"before"`
	After   any      `json:"after"`
}

// PlanMetadata contains metadata about the plan execution
type PlanMetadata struct {
	TerraformVersion string    `json:"terraform_version"`
	Timestamp        time.Time `json:"timestamp"`
}

// GetActionType determines the action type from a list of actions
func GetActionType(actions []string) ActionType {
	if len(actions) == 0 {
		return ActionNoOp
	}

	// Terraform uses array of actions to represent the operation
	// Common patterns:
	// ["create"] -> create
	// ["delete"] -> delete
	// ["update"] -> update
	// ["delete", "create"] -> replace
	// ["no-op"] -> no-op
	// ["read"] -> read

	if len(actions) == 2 && contains(actions, "delete") && contains(actions, "create") {
		return ActionReplace
	}

	if len(actions) == 1 {
		switch actions[0] {
		case "create":
			return ActionCreate
		case "delete":
			return ActionDelete
		case "update":
			return ActionUpdate
		case "no-op":
			return ActionNoOp
		case "read":
			return ActionRead
		}
	}

	// Default to update if we have multiple actions
	if len(actions) > 0 {
		return ActionUpdate
	}

	return ActionNoOp
}

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// GetActionIcon returns the display icon for an action type
func (a ActionType) GetActionIcon() string {
	switch a {
	case ActionCreate:
		return "[+]"
	case ActionDelete:
		return "[-]"
	case ActionUpdate:
		return "[~]"
	case ActionReplace:
		return "[±]"
	case ActionRead:
		return "[→]"
	case ActionNoOp:
		return "[ ]"
	default:
		return "[?]"
	}
}

// GetActionVerb returns a human-readable verb for the action
func (a ActionType) GetActionVerb() string {
	switch a {
	case ActionCreate:
		return "will be created"
	case ActionDelete:
		return "will be destroyed"
	case ActionUpdate:
		return "will be updated"
	case ActionReplace:
		return "will be replaced"
	case ActionRead:
		return "will be read"
	case ActionNoOp:
		return "no changes"
	default:
		return "unknown action"
	}
}

// ValidateResult represents the output of terraform validate -json.
type ValidateResult struct {
	FormatVersion string       `json:"format_version"`
	Valid         bool         `json:"valid"`
	ErrorCount    int          `json:"error_count"`
	WarningCount  int          `json:"warning_count"`
	Diagnostics   []Diagnostic `json:"diagnostics"`
}

// FormatResult represents the output of terraform fmt.
type FormatResult struct {
	ChangedFiles []string
	Success      bool
}

// StateResource represents a resource in terraform state.
type StateResource struct {
	Address      string
	ResourceType string
	Name         string
	Provider     string
	Module       string
}
