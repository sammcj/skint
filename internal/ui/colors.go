// Package ui provides simple non-interactive CLI components (colours, menus, prompts).
// It uses fatih/color for terminal colour output, which is separate from lipgloss
// used in the TUI package. fatih/color is suited for streaming CLI output (Printf-style),
// while lipgloss is designed for Bubble Tea's immediate-mode rendering.
package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/sammcj/skint/internal/config"
)

// ColorScheme holds the color configuration
type ColorScheme struct {
	Enabled bool

	// Colors
	Red     *color.Color
	Green   *color.Color
	Yellow  *color.Color
	Blue    *color.Color
	Cyan    *color.Color
	Magenta *color.Color
	White   *color.Color
	Black   *color.Color

	// Styles
	Bold *color.Color
	Dim  *color.Color
}

// Symbols holds the symbol characters
type Symbols struct {
	OK       string
	Error    string
	Warning  string
	Info     string
	Arrow    string
	Check    string
	Uncheck  string
	Bullet   string
	Ellipsis string
	Spinner  []string
	BoxTL    string
	BoxTR    string
	BoxBL    string
	BoxBR    string
	BoxH     string
	BoxV     string
}

var (
	// Colors is the global color scheme
	Colors *ColorScheme
	// Sym is the global symbols
	Sym *Symbols
)

// Init initializes the UI with the given config
func Init(cfg *config.Config) {
	// Check if colors should be enabled
	enabled := cfg.ColorEnabled && os.Getenv("NO_COLOR") == "" && cfg.OutputFormat == config.FormatHuman

	Colors = &ColorScheme{
		Enabled: enabled,
	}

	if enabled {
		Colors.Red = color.New(color.FgRed)
		Colors.Green = color.New(color.FgGreen)
		Colors.Yellow = color.New(color.FgYellow)
		Colors.Blue = color.New(color.FgBlue)
		Colors.Cyan = color.New(color.FgCyan)
		Colors.Magenta = color.New(color.FgMagenta)
		Colors.White = color.New(color.FgWhite)
		Colors.Black = color.New(color.FgBlack)
		Colors.Bold = color.New(color.Bold)
		Colors.Dim = color.New(color.Faint)
	} else {
		// No-op colors
		Colors.Red = color.New()
		Colors.Green = color.New()
		Colors.Yellow = color.New()
		Colors.Blue = color.New()
		Colors.Cyan = color.New()
		Colors.Magenta = color.New()
		Colors.White = color.New()
		Colors.Black = color.New()
		Colors.Bold = color.New()
		Colors.Dim = color.New()
	}

	// Determine symbols based on terminal capabilities
	useUnicode := enabled && os.Getenv("TERM") != "dumb"

	if useUnicode {
		Sym = &Symbols{
			OK:       "✓",
			Error:    "✗",
			Warning:  "⚠",
			Info:     "→",
			Arrow:    "→",
			Check:    "✓",
			Uncheck:  "○",
			Bullet:   "•",
			Ellipsis: "…",
			Spinner:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
			BoxTL:    "╭",
			BoxTR:    "╮",
			BoxBL:    "╰",
			BoxBR:    "╯",
			BoxH:     "─",
			BoxV:     "│",
		}
	} else {
		Sym = &Symbols{
			OK:       "[OK]",
			Error:    "[X]",
			Warning:  "[!]",
			Info:     ">",
			Arrow:    "->",
			Check:    "[x]",
			Uncheck:  "[ ]",
			Bullet:   "*",
			Ellipsis: "...",
			Spinner:  []string{"-", "\\", "|", "/"},
			BoxTL:    "+",
			BoxTR:    "+",
			BoxBL:    "+",
			BoxBR:    "+",
			BoxH:     "-",
			BoxV:     "|",
		}
	}
}

// Print functions for common use cases

// Success prints a success message
func Success(format string, a ...interface{}) {
	if Colors.Enabled {
		Colors.Green.Printf("%s ", Sym.OK)
	}
	color.White(format, a...)
	println()
}

// Error prints an error message
func Error(format string, a ...interface{}) {
	if Colors.Enabled {
		Colors.Red.Fprintf(os.Stderr, "%s ", Sym.Error)
	}
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Warning prints a warning message
func Warning(format string, a ...interface{}) {
	if Colors.Enabled {
		Colors.Yellow.Fprintf(os.Stderr, "%s ", Sym.Warning)
	}
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Info prints an info message
func Info(format string, a ...interface{}) {
	if Colors.Enabled {
		Colors.Blue.Printf("%s ", Sym.Info)
	}
	fmt.Printf(format+"\n", a...)
}

// Log prints a simple log message
func Log(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
}

// Dim prints dimmed text
func Dim(format string, a ...interface{}) {
	if Colors.Enabled {
		Colors.Dim.Printf(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

// DimString returns dimmed text as a string
func DimString(text string) string {
	if Colors.Enabled {
		return Colors.Dim.Sprint(text)
	}
	return text
}

// Bold returns bold text
func Bold(text string) string {
	if Colors.Enabled {
		return Colors.Bold.Sprint(text)
	}
	return text
}

// Cyan returns cyan text
func Cyan(text string) string {
	if Colors.Enabled {
		return Colors.Cyan.Sprint(text)
	}
	return text
}

// Green returns green text
func Green(text string) string {
	if Colors.Enabled {
		return Colors.Green.Sprint(text)
	}
	return text
}

// Yellow returns yellow text
func Yellow(text string) string {
	if Colors.Enabled {
		return Colors.Yellow.Sprint(text)
	}
	return text
}

// Red returns red text
func Red(text string) string {
	if Colors.Enabled {
		return Colors.Red.Sprint(text)
	}
	return text
}

// Magenta returns magenta text
func Magenta(text string) string {
	if Colors.Enabled {
		return Colors.Magenta.Sprint(text)
	}
	return text
}
