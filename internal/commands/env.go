package commands

import (
	"fmt"
	"sort"

	"github.com/sammcj/skint/internal/providers"
	"github.com/spf13/cobra"
)

// NewEnvCmd creates the env command
func NewEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env [provider]",
		Short: "Print shell export statements for a provider",
		Long: `Print shell export statements for the active (or specified) provider.

Add this to your shell profile to have Claude always use the configured provider:

  eval "$(skint env)"

Or for a specific provider:

  eval "$(skint env openrouter)"`,
		Args: cobra.MaximumNArgs(1),
		RunE: runEnv,
	}

	cmd.Flags().Bool("unset", false, "print unset statements instead (to clear provider env vars)")

	return cmd
}

func runEnv(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)

	unset, _ := cmd.Flags().GetBool("unset")
	if unset {
		return printUnsetStatements()
	}

	// Determine which provider to use
	providerName := cc.Cfg.DefaultProvider
	if len(args) > 0 {
		providerName = args[0]
	}

	if providerName == "" || providerName == "native" {
		// Native Anthropic - no env vars needed, just unset any existing ones
		fmt.Println("# skint: using native Anthropic (no env overrides)")
		return printUnsetStatements()
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

	// Get env vars
	envVars := provider.GetEnvVars()

	fmt.Printf("# skint: provider %s\n", provider.DisplayName())

	// Print in sorted order for deterministic output
	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := envVars[k]
		if v == "" {
			// Empty value means unset it
			fmt.Printf("unset %s\n", k)
		} else {
			fmt.Printf("export %s='%s'\n", k, v)
		}
	}

	return nil
}

func printUnsetStatements() error {
	vars := []string{
		"ANTHROPIC_BASE_URL",
		"ANTHROPIC_AUTH_TOKEN",
		"ANTHROPIC_API_KEY",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_SMALL_FAST_MODEL",
		"OPENAI_BASE_URL",
		"OPENAI_API_KEY",
		"OPENAI_MODEL",
	}
	for _, v := range vars {
		fmt.Printf("unset %s\n", v)
	}
	return nil
}
