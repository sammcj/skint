package ui

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// Box draws a box around content
func Box(title string, width int) {
	if width < 10 {
		width = 40
	}

	inner := width - 2

	// Truncate title if too long
	if len(title) > inner-4 {
		title = title[:inner-7] + "..."
	}

	// Center title
	pad := (inner - len(title)) / 2

	// Build horizontal line
	hline := strings.Repeat(Sym.BoxH, inner)

	// Print box
	fmt.Printf("%s%s%s\n", Sym.BoxTL, hline, Sym.BoxTR)
	fmt.Printf("%s%*s%s%*s%s\n", Sym.BoxV, pad, "", Bold(title), inner-pad-len(title), "", Sym.BoxV)
	fmt.Printf("%s%s%s\n", Sym.BoxBL, hline, Sym.BoxBR)
}

// Separator draws a horizontal separator
func Separator(width int) {
	if width < 1 {
		width = 40
	}
	Dim("%s", strings.Repeat(Sym.BoxH, width))
	fmt.Println()
}

// ListItem prints a list item
func ListItem(checked bool, format string, a ...interface{}) {
	if checked {
		if Colors.Enabled {
			Colors.Green.Printf("  %s ", Sym.Check)
		} else {
			fmt.Printf("  %s ", Sym.Check)
		}
	} else {
		Dim("  %s ", Sym.Uncheck)
	}
	fmt.Printf(format+"\n", a...)
}

// Prompt prints a prompt and returns user input
func Prompt(message, defaultValue string) string {
	promptText := message
	if defaultValue != "" {
		promptText = fmt.Sprintf("%s [%s]", message, defaultValue)
	}

	if Colors.Enabled {
		Colors.Cyan.Printf("%s: ", promptText)
	} else {
		fmt.Printf("%s: ", promptText)
	}

	var response string
	_, _ = fmt.Scanln(&response)

	if response == "" && defaultValue != "" {
		return defaultValue
	}

	return response
}

// Confirm asks for yes/no confirmation
func Confirm(message string, defaultYes bool) bool {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}

	if Colors.Enabled {
		Colors.Cyan.Printf("%s %s: ", message, hint)
	} else {
		fmt.Printf("%s %s: ", message, hint)
	}

	var response string
	_, _ = fmt.Scanln(&response)

	if response == "" {
		return defaultYes
	}

	return strings.EqualFold(response, "y") || strings.EqualFold(response, "yes")
}

// ConfirmDanger asks for dangerous confirmation with phrase
func ConfirmDanger(action, phrase string) bool {
	fmt.Println()
	Box("DANGER", 40)
	fmt.Println()

	if Colors.Enabled {
		Colors.Red.Println(action)
	} else {
		fmt.Println(action)
	}

	fmt.Println()
	fmt.Printf("Type %s to confirm: ", Bold(phrase))

	var response string
	_, _ = fmt.Scanln(&response)

	return response == phrase
}

// Spinner is a loading spinner
type Spinner struct {
	message string
	stop    chan bool
	running atomic.Bool
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		stop:    make(chan bool, 1),
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	if !Colors.Enabled {
		Info(s.message)
		return
	}

	s.running.Store(true)
	go func() {
		i := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				// Clear line
				fmt.Printf("\r\033[K")
				return
			case <-ticker.C:
				if Colors.Enabled {
					Colors.Blue.Printf("\r%s %s", Sym.Spinner[i%len(Sym.Spinner)], s.message)
				}
				i++
			}
		}
	}()
}

// Stop stops the spinner with a status
func (s *Spinner) Stop(success bool) {
	if !s.running.CompareAndSwap(true, false) {
		return
	}

	select {
	case s.stop <- true:
	default:
	}
	time.Sleep(50 * time.Millisecond) // Let the goroutine finish

	if success {
		Success("Done")
	} else {
		Error("Failed")
	}
}

// Table prints a simple table
func Table(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print headers
	for i, h := range headers {
		fmt.Printf("%-*s  ", widths[i], Bold(h))
	}
	fmt.Println()

	// Print separator
	for i := range headers {
		fmt.Printf("%-*s  ", widths[i], strings.Repeat("-", widths[i]))
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("%-*s  ", widths[i], cell)
			}
		}
		fmt.Println()
	}
}

// MaskKey masks an API key for display.
// For short keys (12 chars or fewer), only asterisks are shown to avoid
// revealing most of the key.
func MaskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 12 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// ErrorWithContext prints a detailed error with context
func ErrorWithContext(code, message, context, cause, solution string) {
	fmt.Fprintln(os.Stderr)

	if Colors.Enabled {
		Colors.Red.Fprint(os.Stderr, Bold("ERROR"))
		Dim(" [%s] ", code)
		Colors.Red.Fprintln(os.Stderr, Bold(message))
	} else {
		fmt.Fprintf(os.Stderr, "ERROR [%s] %s\n", code, message)
	}

	Dim("  Context:  %s\n", context)
	Dim("  Cause:    %s\n", cause)

	if Colors.Enabled {
		Colors.Cyan.Fprint(os.Stderr, "  Fix:      ")
		fmt.Fprintln(os.Stderr, solution)
	} else {
		fmt.Fprintf(os.Stderr, "  Fix:      %s\n", solution)
	}

	fmt.Fprintln(os.Stderr)
}

// NextSteps prints suggested next steps
func NextSteps(steps []string) {
	if !Colors.Enabled {
		fmt.Println("\nNext:")
		for _, step := range steps {
			fmt.Printf("  %s %s\n", Sym.Arrow, step)
		}
		return
	}

	fmt.Println()
	Colors.Bold.Println("Next:")
	for _, step := range steps {
		Colors.Cyan.Printf("  %s ", Sym.Arrow)
		fmt.Println(step)
	}
}
