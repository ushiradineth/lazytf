package testutil

import (
	"time"

	"github.com/ushiradineth/lazytf/internal/history"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

// Sample resources for testing.
var (
	// SampleResources contains a mix of 5 resources with different actions.
	SampleResources = []terraform.ResourceChange{
		ResourceWithAction(terraform.ActionCreate),
		ResourceWithAction(terraform.ActionUpdate),
		ResourceWithAction(terraform.ActionDelete),
		ResourceWithAction(terraform.ActionReplace),
		ResourceWithAction(terraform.ActionCreate),
	}

	// FewResources contains 3 resources for testing without scrollbar.
	FewResources = []terraform.ResourceChange{
		{
			Address:      "aws_instance.web_1",
			ResourceType: "aws_instance",
			ResourceName: "web_1",
			Action:       terraform.ActionCreate,
		},
		{
			Address:      "aws_instance.web_2",
			ResourceType: "aws_instance",
			ResourceName: "web_2",
			Action:       terraform.ActionUpdate,
		},
		{
			Address:      "aws_instance.web_3",
			ResourceType: "aws_instance",
			ResourceName: "web_3",
			Action:       terraform.ActionDelete,
		},
	}

	// ManyResources contains 50+ resources for testing scrollbar behavior.
	ManyResources = generateManyResources(50)

	// ModuleResources contains resources in nested modules.
	ModuleResources = []terraform.ResourceChange{
		{
			Address:      "module.vpc.aws_vpc.main",
			ResourceType: "aws_vpc",
			ResourceName: "main",
			Action:       terraform.ActionCreate,
		},
		{
			Address:      "module.vpc.aws_subnet.public",
			ResourceType: "aws_subnet",
			ResourceName: "public",
			Action:       terraform.ActionCreate,
		},
		{
			Address:      "module.vpc.module.nat.aws_nat_gateway.main",
			ResourceType: "aws_nat_gateway",
			ResourceName: "main",
			Action:       terraform.ActionCreate,
		},
		{
			Address:      "module.app.module.web.aws_instance.server",
			ResourceType: "aws_instance",
			ResourceName: "server",
			Action:       terraform.ActionUpdate,
		},
		{
			Address:      "module.app.module.db.aws_rds_instance.main",
			ResourceType: "aws_rds_instance",
			ResourceName: "main",
			Action:       terraform.ActionDelete,
		},
	}

	// SampleHistory contains sample history entries for testing.
	SampleHistory = []history.Entry{
		{
			ID:          1,
			StartedAt:   time.Now().Add(-1 * time.Hour),
			FinishedAt:  time.Now().Add(-55 * time.Minute),
			Duration:    5 * time.Minute,
			Status:      history.StatusSuccess,
			Summary:     "Created 3 resources",
			WorkDir:     "/path/to/project",
			Environment: "dev",
		},
		{
			ID:          2,
			StartedAt:   time.Now().Add(-2 * time.Hour),
			FinishedAt:  time.Now().Add(-1*time.Hour - 50*time.Minute),
			Duration:    10 * time.Minute,
			Status:      history.StatusFailed,
			Summary:     "Failed to create aws_instance.web",
			Error:       "Error: rate limit exceeded",
			WorkDir:     "/path/to/project",
			Environment: "prod",
		},
		{
			ID:          3,
			StartedAt:   time.Now().Add(-3 * time.Hour),
			FinishedAt:  time.Now().Add(-2*time.Hour - 58*time.Minute),
			Duration:    2 * time.Minute,
			Status:      history.StatusCanceled,
			Summary:     "User canceled apply",
			WorkDir:     "/path/to/project",
			Environment: "staging",
		},
	}
)

// ResourceWithAction creates a sample resource with the given action.
func ResourceWithAction(action terraform.ActionType) terraform.ResourceChange {
	name := resourceNameForAction(action)
	return terraform.ResourceChange{
		Address:      "aws_instance." + name,
		ResourceType: "aws_instance",
		ResourceName: name,
		Action:       action,
		Change:       changeForAction(action),
	}
}

// ModuleResource creates a resource within a module path.
func ModuleResource(modulePath, resourceType, resourceName string, action terraform.ActionType) terraform.ResourceChange {
	address := modulePath + "." + resourceType + "." + resourceName
	return terraform.ResourceChange{
		Address:      address,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Action:       action,
		Change:       changeForAction(action),
	}
}

// ResourceWithChange creates a resource with specific before/after values.
func ResourceWithChange(address string, action terraform.ActionType, before, after map[string]any) terraform.ResourceChange {
	return terraform.ResourceChange{
		Address:      address,
		ResourceType: "aws_instance",
		ResourceName: "test",
		Action:       action,
		Change: &terraform.Change{
			Before: before,
			After:  after,
		},
	}
}

// resourceNameForAction returns a descriptive resource name for an action.
func resourceNameForAction(action terraform.ActionType) string {
	switch action {
	case terraform.ActionCreate:
		return "new_server"
	case terraform.ActionUpdate:
		return "updated_server"
	case terraform.ActionDelete:
		return "old_server"
	case terraform.ActionReplace:
		return "replaced_server"
	default:
		return "server"
	}
}

// changeForAction creates a sample Change struct for an action.
func changeForAction(action terraform.ActionType) *terraform.Change {
	switch action {
	case terraform.ActionCreate:
		return &terraform.Change{
			Before: nil,
			After: map[string]any{
				"instance_type": "t3.micro",
				"ami":           "ami-12345678",
				"tags": map[string]any{
					"Name": "web-server",
				},
			},
		}
	case terraform.ActionUpdate:
		return &terraform.Change{
			Before: map[string]any{
				"instance_type": "t3.micro",
				"ami":           "ami-12345678",
			},
			After: map[string]any{
				"instance_type": "t3.small",
				"ami":           "ami-12345678",
			},
		}
	case terraform.ActionDelete:
		return &terraform.Change{
			Before: map[string]any{
				"instance_type": "t3.micro",
				"ami":           "ami-12345678",
			},
			After: nil,
		}
	case terraform.ActionReplace:
		return &terraform.Change{
			Before: map[string]any{
				"ami": "ami-12345678",
			},
			After: map[string]any{
				"ami": "ami-87654321",
			},
		}
	default:
		return nil
	}
}

// generateManyResources creates n resources for testing.
func generateManyResources(n int) []terraform.ResourceChange {
	resources := make([]terraform.ResourceChange, n)
	actions := []terraform.ActionType{
		terraform.ActionCreate,
		terraform.ActionUpdate,
		terraform.ActionDelete,
		terraform.ActionReplace,
	}
	for i := range n {
		action := actions[i%len(actions)]
		resources[i] = terraform.ResourceChange{
			Address:      "aws_instance.server_" + intToString(i),
			ResourceType: "aws_instance",
			ResourceName: "server_" + intToString(i),
			Action:       action,
		}
	}
	return resources
}

// intToString converts an int to a string.
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// HistoryEntry creates a sample history entry.
func HistoryEntry(id int64, status history.Status, summary string) history.Entry {
	return history.Entry{
		ID:          id,
		StartedAt:   time.Now().Add(-time.Duration(id) * time.Hour),
		FinishedAt:  time.Now().Add(-time.Duration(id)*time.Hour + 5*time.Minute),
		Duration:    5 * time.Minute,
		Status:      status,
		Summary:     summary,
		WorkDir:     "/path/to/project",
		Environment: "test",
	}
}
