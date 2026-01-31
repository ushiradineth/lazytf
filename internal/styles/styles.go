package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Terraform CLI diff colors - fixed colors that match terraform's actual output.
// These are NOT affected by user themes to ensure consistency with terraform CLI.
var (
	// TfDiffAdd is muted green like terraform uses for additions (+).
	TfDiffAdd = lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
	// TfDiffRemove is muted red/salmon like terraform uses for deletions (-).
	TfDiffRemove = lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75"))
	// TfDiffChange is yellow/orange like terraform uses for changes (~).
	TfDiffChange = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B"))
	// TfDimmed is gray for null values and separators.
	TfDimmed = lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370"))
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

func adaptive(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

func newTheme(
	name string,
	create, update, deleteColor, replace, noChange, background, foreground, border, selected, dimmed, highlight lipgloss.AdaptiveColor,
) Theme {
	return Theme{
		Name:            name,
		CreateColor:     create,
		UpdateColor:     update,
		DeleteColor:     deleteColor,
		ReplaceColor:    replace,
		NoChangeColor:   noChange,
		BackgroundColor: background,
		ForegroundColor: foreground,
		BorderColor:     border,
		SelectedColor:   selected,
		DimmedColor:     dimmed,
		HighlightColor:  highlight,
	}
}

// Predefined themes.
var (
	// DefaultTheme is a clean, minimal theme.
	DefaultTheme = newTheme(
		"default",
		adaptive("#00AF00", "#00D700"),
		adaptive("#FF8700", "#FFAF00"),
		adaptive("#D70000", "#FF5F5F"),
		adaptive("#B200B2", "#D75FD7"),
		adaptive("#767676", "#9E9E9E"),
		adaptive("#FFFFFF", "#1C1C1C"),
		adaptive("#262626", "#E4E4E4"),
		adaptive("#BCBCBC", "#4E4E4E"),
		adaptive("#5F87D7", "#87AFFF"),
		adaptive("#9E9E9E", "#767676"),
		adaptive("#D7FF00", "#FFFF87"),
	)

	// TerraformCloudTheme mimics the Terraform Cloud UI.
	TerraformCloudTheme = newTheme(
		"terraform-cloud",
		adaptive("#00CA72", "#00CA72"),
		adaptive("#FFA500", "#FFA500"),
		adaptive("#E03A3E", "#E03A3E"),
		adaptive("#C026D3", "#C026D3"),
		adaptive("#8A8A8A", "#8A8A8A"),
		adaptive("#FAFAFA", "#1A1B26"),
		adaptive("#1A1B26", "#F8F8F2"),
		adaptive("#D1D5DA", "#30313C"),
		adaptive("#5F5FFF", "#7070FF"),
		adaptive("#9E9E9E", "#6E6E6E"),
		adaptive("#FFFACD", "#4A4A2A"),
	)

	// MonokaiTheme is inspired by the Monokai palette.
	MonokaiTheme = newTheme(
		"monokai",
		adaptive("#A6E22E", "#A6E22E"),
		adaptive("#FD971F", "#FD971F"),
		adaptive("#F92672", "#F92672"),
		adaptive("#AE81FF", "#AE81FF"),
		adaptive("#75715E", "#75715E"),
		adaptive("#F2F2F2", "#272822"),
		adaptive("#272822", "#F8F8F2"),
		adaptive("#BDBDBD", "#3E3D32"),
		adaptive("#66D9EF", "#66D9EF"),
		adaptive("#9E9E9E", "#75715E"),
		adaptive("#E6DB74", "#E6DB74"),
	)

	// NordTheme is based on the Nord color palette.
	NordTheme = newTheme(
		"nord",
		adaptive("#5E81AC", "#81A1C1"),
		adaptive("#D08770", "#D08770"),
		adaptive("#BF616A", "#BF616A"),
		adaptive("#B48EAD", "#B48EAD"),
		adaptive("#7A7A7A", "#6E7A88"),
		adaptive("#ECEFF4", "#2E3440"),
		adaptive("#2E3440", "#ECEFF4"),
		adaptive("#D8DEE9", "#4C566A"),
		adaptive("#88C0D0", "#88C0D0"),
		adaptive("#8C92A0", "#7A8699"),
		adaptive("#EBCB8B", "#EBCB8B"),
	)

	// GitHubDarkTheme mimics GitHub's dark UI.
	GitHubDarkTheme = newTheme(
		"github-dark",
		adaptive("#2DA44E", "#2DA44E"),
		adaptive("#D29922", "#D29922"),
		adaptive("#F85149", "#F85149"),
		adaptive("#A371F7", "#A371F7"),
		adaptive("#6E7781", "#6E7781"),
		adaptive("#FFFFFF", "#0D1117"),
		adaptive("#24292F", "#C9D1D9"),
		adaptive("#D0D7DE", "#30363D"),
		adaptive("#0969DA", "#1F6FEB"),
		adaptive("#57606A", "#8B949E"),
		adaptive("#D2A8FF", "#D2A8FF"),
	)
)

// Styles contains all the lipgloss styles used in the TUI.
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

// NewStyles creates a new set of styles based on a theme.
func NewStyles(theme Theme) *Styles {
	s := &Styles{
		Theme: theme,
	}

	applyActionStyles(s, theme)
	applyResourceStyles(s, theme)
	applyFilterStyles(s, theme)
	applyStatusStyles(s, theme)
	applySearchStyles(s, theme)
	applyBorderStyles(s, theme)
	applyDiffStyles(s, theme)
	applyHighlightStyles(s, theme)
	applyHelpStyles(s, theme)
	applyTitleStyles(s, theme)
	applyListStyles(s)
	applyDimmedStyles(s, theme)
	applyPanelStyles(s, theme)

	return s
}

func applyActionStyles(s *Styles, theme Theme) {
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
}

func applyResourceStyles(s *Styles, theme Theme) {
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
}

func applyFilterStyles(s *Styles, theme Theme) {
	s.FilterBarActive = lipgloss.NewStyle().
		Foreground(theme.HighlightColor).
		Background(theme.SelectedColor).
		Padding(0, 1).
		Bold(true)

	s.FilterBarInactive = lipgloss.NewStyle().
		Foreground(theme.DimmedColor).
		Background(lipgloss.AdaptiveColor{Light: "#E0E0E0", Dark: "#3A3A3A"}).
		Padding(0, 1)
}

func applyStatusStyles(s *Styles, theme Theme) {
	s.StatusBar = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor).
		Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#2A2A2A"}).
		Padding(0, 1)
}

func applySearchStyles(s *Styles, theme Theme) {
	s.SearchBar = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor).
		Background(lipgloss.AdaptiveColor{Light: "#F2F2F2", Dark: "#262626"}).
		Padding(0, 1)
}

func applyBorderStyles(s *Styles, theme Theme) {
	s.Border = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.BorderColor)
}

func applyDiffStyles(s *Styles, theme Theme) {
	s.DiffAdd = lipgloss.NewStyle().
		Foreground(theme.CreateColor)

	s.DiffRemove = lipgloss.NewStyle().
		Foreground(theme.DeleteColor)

	s.DiffChange = lipgloss.NewStyle().
		Foreground(theme.UpdateColor)

	s.Comment = lipgloss.NewStyle().
		Foreground(theme.DeleteColor)
}

func applyHighlightStyles(s *Styles, theme Theme) {
	s.Highlight = lipgloss.NewStyle().
		Foreground(theme.HighlightColor).
		Bold(true)
}

func applyHelpStyles(s *Styles, theme Theme) {
	s.HelpKey = lipgloss.NewStyle().
		Foreground(theme.SelectedColor).
		Bold(true)

	s.HelpValue = lipgloss.NewStyle().
		Foreground(theme.DimmedColor)
}

func applyTitleStyles(s *Styles, theme Theme) {
	s.Title = lipgloss.NewStyle().
		Foreground(theme.ForegroundColor).
		Background(theme.BorderColor).
		Padding(0, 1).
		Bold(true)
}

func applyListStyles(s *Styles) {
	s.LineItemText = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"})

	s.SelectedLine = lipgloss.NewStyle().
		Background(lipgloss.AdaptiveColor{Light: "#B3D9FF", Dark: "#2F5D8A"}).
		Bold(true)

	s.SelectedLineBackground = lipgloss.AdaptiveColor{Light: "#B3D9FF", Dark: "#2F5D8A"}
}

func applyDimmedStyles(s *Styles, theme Theme) {
	s.Dimmed = lipgloss.NewStyle().
		Foreground(theme.DimmedColor)
}

func applyPanelStyles(s *Styles, theme Theme) {
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
}

// DefaultStyles returns styles with the default theme.
func DefaultStyles() *Styles {
	return NewStyles(DefaultTheme)
}
