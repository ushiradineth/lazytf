package styles

import (
	"fmt"
	"strings"
)

// ResolveTheme returns a predefined theme by name.
func ResolveTheme(name string) (Theme, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "", "default":
		return DefaultTheme, nil
	case "terraform-cloud", "terraform cloud", "terraform_cloud":
		return TerraformCloudTheme, nil
	case "monokai":
		return MonokaiTheme, nil
	case "nord":
		return NordTheme, nil
	case "github-dark", "github dark", "github_dark":
		return GitHubDarkTheme, nil
	default:
		return DefaultTheme, fmt.Errorf("unknown theme: %s", name)
	}
}
