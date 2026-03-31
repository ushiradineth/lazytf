package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Terraform CLI diff colors - fixed colors that match terraform's actual output.
// These are unexported to ensure all styling goes through the Styles struct.
const (
	colorAdd    = lipgloss.Color("#98C379") // muted green for additions (+)
	colorRemove = lipgloss.Color("#E06C75") // muted red/salmon for deletions (-)
	colorChange = lipgloss.Color("#E5C07B") // yellow/orange for changes (~)
	colorDimmed = lipgloss.Color("#5C6370") // gray for null values and separators
)

// Theme defines the color scheme for the TUI.
// Note: Action colors (create/update/delete) are fixed to Terraform CLI colors
// and not configurable via themes.
type Theme struct {
	Name            string
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
	background, foreground, border, selected, dimmed, highlight lipgloss.AdaptiveColor,
) Theme {
	return Theme{
		Name:            name,
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
		adaptive("#FFFFFF", "#1C1C1C"), // background
		adaptive("#262626", "#E4E4E4"), // foreground
		adaptive("#BCBCBC", "#4E4E4E"), // border
		adaptive("#5F87D7", "#87AFFF"), // selected
		adaptive("#9E9E9E", "#767676"), // dimmed
		adaptive("#D7FF00", "#FFFF87"), // highlight
	)

	// TerraformCloudTheme mimics the Terraform Cloud UI.
	TerraformCloudTheme = newTheme(
		"terraform-cloud",
		adaptive("#FAFAFA", "#1A1B26"), // background
		adaptive("#1A1B26", "#F8F8F2"), // foreground
		adaptive("#D1D5DA", "#30313C"), // border
		adaptive("#5F5FFF", "#7070FF"), // selected
		adaptive("#9E9E9E", "#6E6E6E"), // dimmed
		adaptive("#FFFACD", "#4A4A2A"), // highlight
	)

	// MonokaiTheme is inspired by the Monokai palette.
	MonokaiTheme = newTheme(
		"monokai",
		adaptive("#F2F2F2", "#272822"), // background
		adaptive("#272822", "#F8F8F2"), // foreground
		adaptive("#BDBDBD", "#3E3D32"), // border
		adaptive("#66D9EF", "#66D9EF"), // selected
		adaptive("#9E9E9E", "#75715E"), // dimmed
		adaptive("#E6DB74", "#E6DB74"), // highlight
	)

	// MonochromeTheme is a grayscale theme with strong contrast.
	MonochromeTheme = newTheme(
		"monochrome",
		adaptive("#FFFFFF", "#121212"), // background
		adaptive("#111111", "#F5F5F5"), // foreground
		adaptive("#B0B0B0", "#5A5A5A"), // border
		adaptive("#4A4A4A", "#D0D0D0"), // selected
		adaptive("#7A7A7A", "#8A8A8A"), // dimmed
		adaptive("#000000", "#FFFFFF"), // highlight
	)

	// NordTheme is based on the Nord color palette.
	NordTheme = newTheme(
		"nord",
		adaptive("#ECEFF4", "#2E3440"), // background
		adaptive("#2E3440", "#ECEFF4"), // foreground
		adaptive("#D8DEE9", "#4C566A"), // border
		adaptive("#88C0D0", "#88C0D0"), // selected
		adaptive("#8C92A0", "#7A8699"), // dimmed
		adaptive("#EBCB8B", "#EBCB8B"), // highlight
	)

	// GitHubDarkTheme mimics GitHub's dark UI.
	GitHubDarkTheme = newTheme(
		"github-dark",
		adaptive("#FFFFFF", "#0D1117"), // background
		adaptive("#24292F", "#C9D1D9"), // foreground
		adaptive("#D0D7DE", "#30363D"), // border
		adaptive("#0969DA", "#1F6FEB"), // selected
		adaptive("#57606A", "#8B949E"), // dimmed
		adaptive("#D2A8FF", "#D2A8FF"), // highlight
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

	// Terraform CLI colors (for borders and other non-style uses)
	AddColor    lipgloss.Color
	RemoveColor lipgloss.Color
	ChangeColor lipgloss.Color
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

func applyActionStyles(s *Styles, _ Theme) {
	s.Create = lipgloss.NewStyle().Foreground(colorAdd).Bold(true)
	s.Update = lipgloss.NewStyle().Foreground(colorChange).Bold(true)
	s.Delete = lipgloss.NewStyle().Foreground(colorRemove).Bold(true)
	s.Replace = lipgloss.NewStyle().Foreground(colorChange).Bold(true)
	s.NoChange = lipgloss.NewStyle().Foreground(colorDimmed)

	// Expose colors for non-style uses (e.g., borders)
	s.AddColor = colorAdd
	s.RemoveColor = colorRemove
	s.ChangeColor = colorChange
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
		Foreground(theme.ForegroundColor)
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

func applyDiffStyles(s *Styles, _ Theme) {
	s.DiffAdd = lipgloss.NewStyle().Foreground(colorAdd)
	s.DiffRemove = lipgloss.NewStyle().Foreground(colorRemove)
	s.DiffChange = lipgloss.NewStyle().Foreground(colorChange)
	s.Comment = lipgloss.NewStyle().Foreground(colorRemove)
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

func applyDimmedStyles(s *Styles, _ Theme) {
	s.Dimmed = lipgloss.NewStyle().Foreground(colorDimmed)
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
