package diff

// DiffAction represents the type of change in a diff
type DiffAction string

const (
	DiffAdd    DiffAction = "add"
	DiffRemove DiffAction = "remove"
	DiffChange DiffAction = "change"
)

// MinimalDiff represents a single attribute change in a resource
type MinimalDiff struct {
	Path     []string   // Path to the attribute (e.g., ["tags", "Environment"])
	OldValue any        // nil if adding
	NewValue any        // nil if removing
	Action   DiffAction // add/remove/change
}

// UnknownValue represents a Terraform "known after apply" value.
type UnknownValue struct{}

// GetActionSymbol returns the symbol to display for this diff action
func (d DiffAction) GetActionSymbol() string {
	switch d {
	case DiffAdd:
		return "+"
	case DiffRemove:
		return "-"
	case DiffChange:
		return "~"
	default:
		return "?"
	}
}

// AttributeChange is an alias for MinimalDiff for clarity in different contexts
type AttributeChange = MinimalDiff
