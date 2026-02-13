package tui

import (
	"fmt"
	"net/http"
	"os"
	"time"

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
	Action           string // "", "test", "launch"
	SelectedProvider string
}

// LaunchFunc is called to launch claude with a specific provider.
// The caller (root command) provides this, wiring up the launcher and secrets.
type LaunchFunc func(providerName string) error

// RunInteractive runs the full interactive TUI for configuration.
// Loops back to the TUI after test actions; exits on quit or launch.
func RunInteractive(cfg *config.Config, secretsMgr *secrets.Manager, saveFn func() error, launchFn LaunchFunc) error {
	for {
		result, err := RunConfigTUI(cfg, secretsMgr)
		if err != nil {
			return err
		}

		// Save config if modified
		if saveFn != nil && result.Done {
			if err := saveFn(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		}

		switch result.Action {
		case "launch":
			providerName := cfg.DefaultProvider
			if providerName == "" || providerName == "native" {
				return launchFn("")
			}
			return launchFn(providerName)

		case "test":
			runProviderTests(cfg)
			// Loop back to TUI
			continue

		default:
			// Normal quit
			return nil
		}
	}
}

// runProviderTests tests connectivity to all configured providers
func runProviderTests(cfg *config.Config) {
	fmt.Print("\033[H\033[2J")
	fmt.Println("Testing Provider Connectivity")
	fmt.Println("-----------------------------")
	fmt.Println()

	tested := 0
	ok := 0
	failed := 0

	for _, p := range cfg.Providers {
		if !p.IsConfigured() {
			continue
		}

		testURL := p.BaseURL
		if testURL == "" {
			if p.Name == "native" {
				testURL = "https://api.anthropic.com"
			} else {
				continue
			}
		}

		tested++
		fmt.Printf("  %-20s ", p.DisplayName)

		client := &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, err := client.Get(testURL)
		if err != nil {
			fmt.Printf("✗ unreachable (%v)\n", err)
			failed++
			continue
		}
		resp.Body.Close()

		fmt.Printf("✓ reachable (HTTP %d)\n", resp.StatusCode)
		ok++
	}

	if tested == 0 {
		fmt.Println("  No configured providers to test.")
	}

	fmt.Println()
	fmt.Printf("Results: %d reachable, %d failed\n", ok, failed)
	fmt.Println()
	fmt.Println("Press Enter to return to Skint...")
	_, _ = fmt.Scanln()
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
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return false
	}

	return true
}
