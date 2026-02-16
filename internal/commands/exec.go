package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/sammcj/skint/internal/launcher"
	"github.com/sammcj/skint/internal/providers"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// NewExecCmd creates the exec command
func NewExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <command> [args...]",
		Short: "Execute a command with provider environment",
		Long: `Execute any command with the configured provider's environment variables set.

This allows you to run any command (not just Claude) with the provider's
API keys and endpoints configured in the environment.`,
		Example: `  skint exec claude --continue
  skint exec claude --dangerously-skip-permissions
  skint exec env | grep ANTHROPIC
  skint exec /bin/bash -c "echo \$ANTHROPIC_BASE_URL"`,
		RunE: runExec,
		// Disable flag parsing so all flags are passed to the command
		DisableFlagParsing: true,
	}

	return cmd
}

func runExec(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)

	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	// Get the default provider or the one specified
	providerName := cc.Cfg.DefaultProvider
	if providerName == "" {
		if len(cc.Cfg.Providers) == 0 {
			return fmt.Errorf("no providers configured. Run 'skint config' to add one")
		}
		if len(cc.Cfg.Providers) == 1 {
			providerName = cc.Cfg.Providers[0].Name
		} else {
			return fmt.Errorf("no default provider set and multiple providers configured. Use 'skint use <provider>' or set a default")
		}
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

	// Build environment -- remove conflicting vars first
	env := launcher.FilterEnvVars(os.Environ(), launcher.ConflictingEnvVars...)

	// Add provider-specific variables
	providerVars := provider.GetEnvVars()
	for key, value := range providerVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Show banner if enabled
	if !cc.Cfg.NoBanner && !cc.Quiet {
		ui.Log("Executing with %s", ui.Green(provider.DisplayName()))
	}

	// Get the command to execute
	command := args[0]
	commandArgs := args[1:]

	// If the command is "claude", check if it exists
	if command == "claude" {
		_, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude command not found. Please install Claude Code: https://claude.ai/install.sh")
		}
	}

	// Execute the command
	execCmd := exec.Command(command, commandArgs...)
	execCmd.Env = env
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}
