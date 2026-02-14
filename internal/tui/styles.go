package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles holds all the Lipgloss styles for the TUI
type Styles struct {
	// Container styles
	App     lipgloss.Style
	Header  lipgloss.Style
	Footer  lipgloss.Style
	Content lipgloss.Style
	Help    lipgloss.Style

	// Component styles
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Description lipgloss.Style
	Selected    lipgloss.Style
	Normal      lipgloss.Style
	Dimmed      lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style

	// Form styles
	Input       lipgloss.Style
	InputPrompt lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style

	// List styles
	List         lipgloss.Style
	ListItem     lipgloss.Style
	ListSelected lipgloss.Style
	ListActive   lipgloss.Style
	Category     lipgloss.Style

	// Box styles
	Box        lipgloss.Style
	BoxTitle   lipgloss.Style
	BoxContent lipgloss.Style

	// Button styles
	ButtonActive   lipgloss.Style
	ButtonInactive lipgloss.Style

	// Form inactive input (unfocused fields with dim border)
	InputInactive lipgloss.Style

	// Picker box (model picker overlay)
	PickerBox      lipgloss.Style
	PickerBoxTitle lipgloss.Style

	// Header line (compact single-line header)
	HeaderLine lipgloss.Style
	HeaderSep  lipgloss.Style

	// Colors
	PrimaryColor   lipgloss.Color
	SecondaryColor lipgloss.Color
	SuccessColor   lipgloss.Color
	ErrorColor     lipgloss.Color
	WarningColor   lipgloss.Color
	InfoColor      lipgloss.Color
	DimColor       lipgloss.Color
	BgColor        lipgloss.Color
}

// DefaultStyles returns the default styles for the TUI
func DefaultStyles() Styles {
	// Color palette
	primary := lipgloss.Color("#7C3AED")   // Violet
	secondary := lipgloss.Color("#EC4899") // Pink
	success := lipgloss.Color("#10B981")   // Emerald
	error := lipgloss.Color("#EF4444")     // Red
	warning := lipgloss.Color("#F59E0B")   // Amber
	info := lipgloss.Color("#3B82F6")      // Blue
	dim := lipgloss.Color("#6B7280")       // Gray
	bg := lipgloss.Color("#1F2937")        // Dark gray

	s := Styles{
		PrimaryColor:   primary,
		SecondaryColor: secondary,
		SuccessColor:   success,
		ErrorColor:     error,
		WarningColor:   warning,
		InfoColor:      info,
		DimColor:       dim,
		BgColor:        bg,
	}

	// Container styles
	s.App = lipgloss.NewStyle().
		Padding(1, 2)

	s.Header = lipgloss.NewStyle().
		MarginBottom(1).
		BorderBottom(true).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(primary)

	s.Footer = lipgloss.NewStyle().
		MarginTop(1).
		BorderTop(true).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(dim)

	s.Content = lipgloss.NewStyle().
		MarginLeft(1).
		MarginRight(1)

	s.Help = lipgloss.NewStyle().
		Foreground(dim).
		Italic(true)

	// Text styles
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(primary).
		MarginBottom(1).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(primary)

	s.Subtitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(secondary).
		MarginBottom(1)

	s.Description = lipgloss.NewStyle().
		Foreground(dim).
		Italic(true)

	s.Selected = lipgloss.NewStyle().
		Bold(true).
		Foreground(primary).
		PaddingLeft(1).
		PaddingRight(1)

	s.Normal = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB"))

	s.Dimmed = lipgloss.NewStyle().
		Foreground(dim)

	// Status styles
	s.Success = lipgloss.NewStyle().
		Foreground(success).
		Bold(true)

	s.Error = lipgloss.NewStyle().
		Foreground(error).
		Bold(true)

	s.Warning = lipgloss.NewStyle().
		Foreground(warning).
		Bold(true)

	s.Info = lipgloss.NewStyle().
		Foreground(info)

	// Form styles
	s.Input = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(primary).
		Padding(0, 1)

	s.InputPrompt = lipgloss.NewStyle().
		Foreground(primary).
		Bold(true)

	s.Label = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Bold(true)

	s.Value = lipgloss.NewStyle().
		Foreground(info)

	// List styles
	s.List = lipgloss.NewStyle().
		MarginLeft(2)

	s.ListItem = lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		PaddingTop(0).
		PaddingBottom(0)

	s.ListSelected = lipgloss.NewStyle().
		Foreground(primary).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(2).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(primary)

	s.ListActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(2).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(success)

	s.Category = lipgloss.NewStyle().
		Bold(true).
		Foreground(secondary).
		MarginTop(1).
		MarginBottom(0)

	// Box styles
	s.Box = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(1, 2).
		Margin(1, 0)

	s.BoxTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primary).
		MarginBottom(1)

	s.BoxContent = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB"))

	// Button styles
	s.ButtonActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(primary).
		Padding(0, 2)

	s.ButtonInactive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Background(bg).
		Padding(0, 2)

	// Inactive input (unfocused field with dim border)
	s.InputInactive = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(dim).
		Padding(0, 1)

	// Picker box
	s.PickerBox = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(info).
		Padding(0, 1).
		MarginLeft(2)

	s.PickerBoxTitle = lipgloss.NewStyle().
		Foreground(info).
		Bold(true)

	// Header line
	s.HeaderLine = lipgloss.NewStyle().
		Bold(true).
		Foreground(primary)

	s.HeaderSep = lipgloss.NewStyle().
		Foreground(dim)

	return s
}

// CompactStyles returns compact styles for smaller terminals
func CompactStyles() Styles {
	s := DefaultStyles()

	// Reduce margins and padding
	s.App = lipgloss.NewStyle().Padding(0, 1)
	s.Header = lipgloss.NewStyle().MarginBottom(0)
	s.Footer = lipgloss.NewStyle().MarginTop(0)
	s.Box = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(s.PrimaryColor).
		Padding(0, 1).
		Margin(0)
	s.PickerBox = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(s.InfoColor).
		Padding(0, 1).
		MarginLeft(1)

	return s
}
