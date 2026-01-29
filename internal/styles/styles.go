package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color scheme for the TUI.
type Theme struct {
	Name            string
	CreateColor     lipgloss.AdaptiveColor
	UpdateColor     lipgloss.AdaptiveColor
	DeleteColor     lipgloss.AdaptiveColor
	ReplaceColor    lipgloss.AdaptiveColor
	NoChangeColor   lipgloss.AdaptiveColor
	BackgroundColor lipgloss.AdaptiveColor
	ForegroundColor lipgloss.AdaptiveColor
	BorderColor     lipgloss.AdaptiveColor
	SelectedColor   lipgloss.AdaptiveColor
	DimmedColor     lipgloss.AdaptiveColor
	HighlightColor  lipgloss.AdaptiveColor
}

// Predefined themes
var (
	// DefaultTheme is a clean, minimal theme
	DefaultTheme = Theme{
		Name:            "default",
		CreateColor:     lipgloss.AdaptiveColor{Light: "#00AF00", Dark: "#00D700"},
		UpdateColor:     lipgloss.AdaptiveColor{Light: "#FF8700", Dark: "#FFAF00"},
		DeleteColor:     lipgloss.AdaptiveColor{Light: "#D70000", Dark: "#FF5F5F"},
		ReplaceColor:    lipgloss.AdaptiveColor{Light: "#B200B2", Dark: "#D75FD7"},
		NoChangeColor:   lipgloss.AdaptiveColor{Light: "#767676", Dark: "#9E9E9E"},
		BackgroundColor: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1C1C1C"},
		ForegroundColor: lipgloss.AdaptiveColor{Light: "#262626", Dark: "#E4E4E4"},
		BorderColor:     lipgloss.AdaptiveColor{Light: "#BCBCBC", Dark: "#4E4E4E"},
		SelectedColor:   lipgloss.AdaptiveColor{Light: "#5F87D7", Dark: "#87AFFF"},
		DimmedColor:     lipgloss.AdaptiveColor{Light: "#9E9E9E", Dark: "#767676"},
		HighlightColor:  lipgloss.AdaptiveColor{Light: "#D7FF00", Dark: "#FFFF87"},
	}

	// TerraformCloudTheme mimics the Terraform Cloud UI
	TerraformCloudTheme = Theme{
		Name:            "terraform-cloud",
		CreateColor:     lipgloss.AdaptiveColor{Light: "#00CA72", Dark: "#00CA72"},
		UpdateColor:     lipgloss.AdaptiveColor{Light: "#FFA500", Dark: "#FFA500"},
		DeleteColor:     lipgloss.AdaptiveColor{Light: "#E03A3E", Dark: "#E03A3E"},
		ReplaceColor:    lipgloss.AdaptiveColor{Light: "#C026D3", Dark: "#C026D3"},
		NoChangeColor:   lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#8A8A8A"},
		BackgroundColor: lipgloss.AdaptiveColor{Light: "#FAFAFA", Dark: "#1A1B26"},
		ForegroundColor: lipgloss.AdaptiveColor{Light: "#1A1B26", Dark: "#F8F8F2"},
		BorderColor:     lipgloss.AdaptiveColor{Light: "#D1D5DA", Dark: "#30313C"},
		SelectedColor:   lipgloss.AdaptiveColor{Light: "#5F5FFF", Dark: "#7070FF"},
		DimmedColor:     lipgloss.AdaptiveColor{Light: "#9E9E9E", Dark: "#6E6E6E"},
		HighlightColor:  lipgloss.AdaptiveColor{Light: "#FFFACD", Dark: "#4A4A2A"},
	}

	// MonokaiTheme is inspired by the Monokai palette.
	MonokaiTheme = Theme{
		Name:            "monokai",
		CreateColor:     lipgloss.AdaptiveColor{Light: "#A6E22E", Dark: "#A6E22E"},
		UpdateColor:     lipgloss.AdaptiveColor{Light: "#FD971F", Dark: "#FD971F"},
		DeleteColor:     lipgloss.AdaptiveColor{Light: "#F92672", Dark: "#F92672"},
		ReplaceColor:    lipgloss.AdaptiveColor{Light: "#AE81FF", Dark: "#AE81FF"},
		NoChangeColor:   lipgloss.AdaptiveColor{Light: "#75715E", Dark: "#75715E"},
		BackgroundColor: lipgloss.AdaptiveColor{Light: "#F2F2F2", Dark: "#272822"},
		ForegroundColor: lipgloss.AdaptiveColor{Light: "#272822", Dark: "#F8F8F2"},
		BorderColor:     lipgloss.AdaptiveColor{Light: "#BDBDBD", Dark: "#3E3D32"},
		SelectedColor:   lipgloss.AdaptiveColor{Light: "#66D9EF", Dark: "#66D9EF"},
		DimmedColor:     lipgloss.AdaptiveColor{Light: "#9E9E9E", Dark: "#75715E"},
		HighlightColor:  lipgloss.AdaptiveColor{Light: "#E6DB74", Dark: "#E6DB74"},
	}

	// NordTheme is based on the Nord color palette.
	NordTheme = Theme{
		Name:            "nord",
		CreateColor:     lipgloss.AdaptiveColor{Light: "#5E81AC", Dark: "#81A1C1"},
		UpdateColor:     lipgloss.AdaptiveColor{Light: "#D08770", Dark: "#D08770"},
		DeleteColor:     lipgloss.AdaptiveColor{Light: "#BF616A", Dark: "#BF616A"},
		ReplaceColor:    lipgloss.AdaptiveColor{Light: "#B48EAD", Dark: "#B48EAD"},
		NoChangeColor:   lipgloss.AdaptiveColor{Light: "#7A7A7A", Dark: "#6E7A88"},
		BackgroundColor: lipgloss.AdaptiveColor{Light: "#ECEFF4", Dark: "#2E3440"},
		ForegroundColor: lipgloss.AdaptiveColor{Light: "#2E3440", Dark: "#ECEFF4"},
		BorderColor:     lipgloss.AdaptiveColor{Light: "#D8DEE9", Dark: "#4C566A"},
		SelectedColor:   lipgloss.AdaptiveColor{Light: "#88C0D0", Dark: "#88C0D0"},
		DimmedColor:     lipgloss.AdaptiveColor{Light: "#8C92A0", Dark: "#7A8699"},
		HighlightColor:  lipgloss.AdaptiveColor{Light: "#EBCB8B", Dark: "#EBCB8B"},
	}

	// GitHubDarkTheme mimics GitHub's dark UI.
	GitHubDarkTheme = Theme{
		Name:            "github-dark",
		CreateColor:     lipgloss.AdaptiveColor{Light: "#2DA44E", Dark: "#2DA44E"},
		UpdateColor:     lipgloss.AdaptiveColor{Light: "#D29922", Dark: "#D29922"},
		DeleteColor:     lipgloss.AdaptiveColor{Light: "#F85149", Dark: "#F85149"},
		ReplaceColor:    lipgloss.AdaptiveColor{Light: "#A371F7", Dark: "#A371F7"},
		NoChangeColor:   lipgloss.AdaptiveColor{Light: "#6E7781", Dark: "#6E7781"},
		BackgroundColor: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#0D1117"},
		ForegroundColor: lipgloss.AdaptiveColor{Light: "#24292F", Dark: "#C9D1D9"},
		BorderColor:     lipgloss.AdaptiveColor{Light: "#D0D7DE", Dark: "#30363D"},
		SelectedColor:   lipgloss.AdaptiveColor{Light: "#0969DA", Dark: "#1F6FEB"},
		DimmedColor:     lipgloss.AdaptiveColor{Light: "#57606A", Dark: "#8B949E"},
		HighlightColor:  lipgloss.AdaptiveColor{Light: "#D2A8FF", Dark: "#D2A8FF"},
	}
)

// Styles contains all the lipgloss styles used in the TUI
type Styles struct {
	Theme                  Theme
	Create                 lipgloss.Style
	Update                 lipgloss.Style
	Delete                 lipgloss.Style
	Replace                lipgloss.Style
	NoChange               lipgloss.Style
	ResourceAddress        lipgloss.Style
	ResourceCollapsed      lipgloss.Style
	ResourceExpanded       lipgloss.Style
	Selected               lipgloss.Style
	FilterBarActive        lipgloss.Style
	FilterBarInactive      lipgloss.Style
	SearchBar              lipgloss.Style
	StatusBar              lipgloss.Style
	Border                 lipgloss.Style
	DiffAdd                lipgloss.Style
	DiffRemove             lipgloss.Style
	DiffChange             lipgloss.Style
	Comment                lipgloss.Style
	Highlight              lipgloss.Style
	HelpKey                lipgloss.Style
	HelpValue              lipgloss.Style
	Title                  lipgloss.Style
	LineItemText           lipgloss.Style
	SelectedLine           lipgloss.Style
	SelectedLineBackground lipgloss.AdaptiveColor
	Dimmed                 lipgloss.Style

	// Panel styles
	FocusedBorder     lipgloss.Style
	PanelTitle        lipgloss.Style
	FocusedPanelTitle lipgloss.Style
	ListItem          lipgloss.Style
	Bold              lipgloss.Style
	Help              lipgloss.Style
}

// NewStyles creates a new set of styles based on a theme
func NewStyles(theme Theme) *Styles {
	s := &Styles{
		Theme: theme,
	}

	// Action styles
	s.Create = lipgloss.NewStyle().
		Foreground(theme.CreateColor).
		Bold(true)

	s.Update = lipgloss.NewStyle().
		Foreground(theme.UpdateColor).
		Bold(true)

	s.Delete = lipgloss.NewStyle().
		Foreground(theme.DeleteColor).
		Bold(true)

	s.Replace = lipgloss.NewStyle().
		Foreground(theme.ReplaceColor).
		Bold(true)

	s.NoChange = lipgloss.NewStyle().
		Foreground(theme.NoChangeColor)

	// Resource styles
	s.ResourceAddress = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor)

	s.ResourceCollapsed = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor).
		MarginLeft(1)

	s.ResourceExpanded = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor).
		MarginLeft(1).
		MarginTop(0).
		MarginBottom(1)

	s.Selected = lipgloss.NewStyle().
		Foreground(theme.SelectedColor).
		Background(lipgloss.AdaptiveColor{Light: "#F0F0F0", Dark: "#2E2E2E"}).
		Bold(true)

	// Filter bar styles
	s.FilterBarActive = lipgloss.NewStyle().
		Foreground(theme.HighlightColor).
		Background(theme.SelectedColor).
		Padding(0, 1).
		Bold(true)

	s.FilterBarInactive = lipgloss.NewStyle().
		Foreground(theme.DimmedColor).
		Background(lipgloss.AdaptiveColor{Light: "#E0E0E0", Dark: "#3A3A3A"}).
		Padding(0, 1)

	// Status bar
	s.StatusBar = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor).
		Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#2A2A2A"}).
		Padding(0, 1)

	// Search bar
	s.SearchBar = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor).
		Background(lipgloss.AdaptiveColor{Light: "#F2F2F2", Dark: "#262626"}).
		Padding(0, 1)

	// Border
	s.Border = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.BorderColor)

	// Diff styles
	s.DiffAdd = lipgloss.NewStyle().
		Foreground(theme.CreateColor)

	s.DiffRemove = lipgloss.NewStyle().
		Foreground(theme.DeleteColor)

	s.DiffChange = lipgloss.NewStyle().
		Foreground(theme.UpdateColor)

	s.Comment = lipgloss.NewStyle().
		Foreground(theme.DeleteColor)

	// Highlight
	s.Highlight = lipgloss.NewStyle().
		Foreground(theme.HighlightColor).
		Bold(true)

	// Help styles
	s.HelpKey = lipgloss.NewStyle().
		Foreground(theme.SelectedColor).
		Bold(true)

	s.HelpValue = lipgloss.NewStyle().
		Foreground(theme.DimmedColor)

	// Title
	s.Title = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor).
		Background(theme.BorderColor).
		Padding(0, 1).
		Bold(true)

	// List line items
	s.LineItemText = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"})

	s.SelectedLine = lipgloss.NewStyle().
		Background(lipgloss.AdaptiveColor{Light: "#B3D9FF", Dark: "#2F5D8A"}).
		Bold(true)

	s.SelectedLineBackground = lipgloss.AdaptiveColor{Light: "#B3D9FF", Dark: "#2F5D8A"}

	// Dimmed text
	s.Dimmed = lipgloss.NewStyle().
		Foreground(theme.DimmedColor)

	// Panel styles
	s.FocusedBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.SelectedColor)

	s.PanelTitle = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor).
		Bold(true)

	s.FocusedPanelTitle = lipgloss.NewStyle().
		Foreground(theme.SelectedColor).
		Bold(true)

	s.ListItem = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor)

	s.Bold = lipgloss.NewStyle().
		Bold(true)

	s.Help = lipgloss.NewStyle().
		Foreground(theme.DimmedColor)

	return s
}

// DefaultStyles returns styles with the default theme
func DefaultStyles() *Styles {
	return NewStyles(DefaultTheme)
}
