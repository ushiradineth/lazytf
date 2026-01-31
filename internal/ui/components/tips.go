package components

import "math/rand/v2"

// Tips contains helpful hints shown when the command log is empty.
//
//nolint:gochecknoglobals // tips is a read-only collection of strings
var tips = []string{
	// Navigation & UI
	"Press '?' to see all keybindings.",
	"Press 'Tab' to cycle through panels, 'Shift+Tab' to go back.",
	"Use number keys '1-4' to jump directly to panels.",
	"Press 'L' to toggle the command log panel.",
	"Press 'T' to change the theme.",
	"Press ',' to open settings.",
	"Use 'j/k' or arrow keys to navigate lists.",
	"Press 'g' to jump to top, 'G' to jump to bottom.",
	"Press 'Esc' to return to the resource list.",

	// Resource Filtering
	"Press 'c' to filter only create operations.",
	"Press 'u' to filter only update operations.",
	"Press 'd' to filter only delete operations.",
	"Press 'r' to filter only replace operations.",
	"Press 't' to expand/collapse all resource groups.",
	"Use '[' and ']' to switch between tabs in the Resources panel.",

	// Terraform Workflow
	"Press 'p' to run terraform plan.",
	"Press 'a' to run terraform apply (requires confirmation).",
	"Press 'v' to validate your configuration.",
	"Press 'f' to format your Terraform files.",
	"Press 'h' to toggle the history panel (when enabled).",
	"Use 'Ctrl+C' to cancel a running operation.",

	// General Terraform Tips
	"Run 'terraform plan' before 'apply' to preview changes.",
	"Resources marked with '+' will be created, '-' deleted, '~' updated.",
	"Replace operations ('-/+') destroy and recreate the resource.",
	"Review the diff viewer to understand exactly what will change.",
	"State drift occurs when infrastructure changes outside Terraform.",
	"Use '-target' flags carefully - they can leave state inconsistent.",

	// Best Practices
	"Always review the plan output before applying changes.",
	"Check for sensitive values in outputs before sharing logs.",
	"Keep your Terraform state file secure and backed up.",
	"Use workspaces to manage multiple environments.",
	"Consider using 'terraform plan -out=plan.tfplan' to save plans.",
	"Use 'terraform state list' to see all resources in state.",
	"The 'terraform refresh' command updates state without changes.",
	"Lock files (.terraform.lock.hcl) ensure consistent provider versions.",
}

// GetRandomTip returns a random tip from the tips collection.
func GetRandomTip() string {
	//nolint:gosec // math/rand is fine for non-security tip selection
	return tips[rand.IntN(len(tips))]
}
