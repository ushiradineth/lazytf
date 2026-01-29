package styles

import "testing"

func TestResolveTheme(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		themeName   string
	}{
		{"empty string", "", false, DefaultTheme.Name},
		{"default", "default", false, DefaultTheme.Name},
		{"Default uppercase", "Default", false, DefaultTheme.Name},
		{"terraform-cloud", "terraform-cloud", false, TerraformCloudTheme.Name},
		{"terraform cloud space", "terraform cloud", false, TerraformCloudTheme.Name},
		{"terraform_cloud underscore", "terraform_cloud", false, TerraformCloudTheme.Name},
		{"Terraform-Cloud mixed case", "Terraform-Cloud", false, TerraformCloudTheme.Name},
		{"monokai", "monokai", false, MonokaiTheme.Name},
		{"Monokai uppercase", "Monokai", false, MonokaiTheme.Name},
		{"nord", "nord", false, NordTheme.Name},
		{"Nord uppercase", "Nord", false, NordTheme.Name},
		{"github-dark", "github-dark", false, GitHubDarkTheme.Name},
		{"github dark space", "github dark", false, GitHubDarkTheme.Name},
		{"github_dark underscore", "github_dark", false, GitHubDarkTheme.Name},
		{"unknown theme", "unknown-theme", true, ""},
		{"whitespace trimmed", "  nord  ", false, NordTheme.Name},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme, err := ResolveTheme(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if theme.Name != tt.themeName {
				t.Errorf("expected theme name %q, got %q", tt.themeName, theme.Name)
			}
		})
	}
}
