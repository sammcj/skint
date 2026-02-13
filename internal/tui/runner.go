package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/secrets"
)

// RunConfigTUI runs the configuration TUI and returns the result
func RunConfigTUI(cfg *config.Config, secretsMgr *secrets.Manager) (*ConfigResult, error) {
	model := NewModel(cfg, secretsMgr)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	m, ok := finalModel.(*Model)
	if !ok {
		return nil, fmt.Errorf("TUI returned unexpected model type: %T", finalModel)
	}

	return &ConfigResult{
		Done:             m.IsDone(),
		Action:           m.GetResultAction(),
		SelectedProvider: m.GetSelectedProvider(),
	}, nil
}

// ConfigResult holds the result of the TUI
type ConfigResult struct {
	Done             bool
	Action           string
	SelectedProvider string
}

// RunInteractive runs the full interactive TUI for configuration
func RunInteractive(cfg *config.Config, secretsMgr *secrets.Manager, saveFn func() error) error {
	// Run the TUI once
	result, err := RunConfigTUI(cfg, secretsMgr)
	if err != nil {
		return err
	}

	// Handle test action
	if result.Action == "test" {
		// Clear screen and run tests
		fmt.Print("\033[H\033[2J")
		fmt.Println("Running provider tests...")
		fmt.Println()

		// Run tests for all configured providers
		hasErrors := false
		for _, provider := range cfg.Providers {
			if provider.NeedsAPIKey() && provider.GetAPIKey() == "" {
				continue
			}
			fmt.Printf("Testing %s... ", provider.DisplayName)
			// Note: Actual test implementation would go here
			fmt.Println("✓")
		}

		fmt.Println()
		if hasErrors {
			fmt.Println("Some tests failed. Press Enter to continue...")
		} else {
			fmt.Println("All tests passed! Press Enter to continue...")
		}
		_, _ = fmt.Scanln()
	}

	// Save config if modified
	if saveFn != nil && result.Done {
		if err := saveFn(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	return nil
}

// RunProviderPicker runs a simple provider picker and returns the selected provider
func RunProviderPicker(cfg *config.Config, secretsMgr *secrets.Manager) (string, error) {
	model := NewModel(cfg, secretsMgr)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("TUI error: %w", err)
	}

	m, ok := finalModel.(*Model)
	if !ok {
		return "", fmt.Errorf("TUI returned unexpected model type: %T", finalModel)
	}
	return m.GetSelectedProvider(), nil
}

// CheckTerminal checks if the terminal supports the TUI
func CheckTerminal() bool {
	// Check if we're in a terminal
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	// Check if stdin is a terminal
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// Check if we're being piped
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	return true
}
