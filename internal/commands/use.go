package commands

import (
	"fmt"

	"github.com/sammcj/skint/internal/launcher"
	"github.com/sammcj/skint/internal/providers"
	"github.com/spf13/cobra"
)

// NewUseCmd creates the use command
func NewUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <provider> [args...]",
		Short: "Launch Claude with a specific provider",
		Long: `Launch Claude Code using the specified provider.

This sets the appropriate environment variables and execs Claude.
Any additional arguments are passed directly to Claude.`,
		Example: `  skint use zai                    # Use Z.AI
  skint use zai --model glm-4.7    # Override model
  skint use ollama --model qwen3   # Use local Ollama`,
		Args: cobra.MinimumNArgs(1),
		RunE: runUse,
	}
}

func runUse(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)
	providerName := args[0]
	claudeArgs := args[1:]

	// Check if claude is installed
	if err := launcher.CheckClaude(); err != nil {
		return err
	}

	// Resolve provider config and load API key
	p, err := cc.ResolveProvider(providerName)
	if err != nil {
		return err
	}

	// Convert to provider interface
	provider, err := providers.FromConfig(p)
	if err != nil {
		return fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	// Create launcher
	l, err := launcher.New(cc.Cfg)
	if err != nil {
		return fmt.Errorf("failed to create launcher: %w", err)
	}

	// Launch Claude
	// This will replace the current process on Unix
	return l.Launch(provider, claudeArgs)
}
