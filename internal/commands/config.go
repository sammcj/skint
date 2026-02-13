package commands

import (
	"fmt"
	"strings"

	"github.com/sammcj/skint/internal/providers"
	"github.com/sammcj/skint/internal/tui"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [provider]",
		Short: "Configure providers",
		Long: `Configure LLM providers for use with Claude Code.

Launch an interactive TUI to configure providers, or specify a provider name to configure it directly.`,
		Example: `  skint config           # Interactive TUI
  skint config zai       # Configure Z.AI
  skint config openrouter # Configure OpenRouter`,
		RunE: runConfig,
	}

	cmd.AddCommand(NewConfigAddCmd())
	cmd.AddCommand(NewConfigRemoveCmd())

	return cmd
}

func runConfig(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)

	// Check if provider name was given
	if len(args) > 0 {
		return configureProviderWithTUI(cc, args[0])
	}

	// Always use TUI
	return tui.RunInteractive(cc.Cfg, cc.SecretsMgr, cc.SaveConfig)
}

func configureProviderWithTUI(cc *CmdContext, name string) error {
	// Check if it's a valid provider
	registry := providers.NewRegistry()
	if _, ok := registry.Get(name); !ok && name != "openrouter" && name != "custom" {
		return fmt.Errorf("unknown provider: %s", name)
	}

	// Run TUI with pre-selected provider
	result, err := tui.RunConfigTUI(cc.Cfg, cc.SecretsMgr)
	if err != nil {
		return err
	}

	// Save config if modified
	if result.Done {
		if err := cc.SaveConfig(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	return nil
}

// NewConfigAddCmd creates the config add command
func NewConfigAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <provider>",
		Short: "Add a new provider",
		Long:  "Add a new provider configuration using the interactive TUI.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := GetContext(cmd)
			return configureProviderWithTUI(cc, args[0])
		},
	}
}

// NewConfigRemoveCmd creates the config remove command
func NewConfigRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <provider>",
		Aliases: []string{"rm"},
		Short:   "Remove a provider configuration",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := GetContext(cmd)
			name := args[0]

			if !cc.YesMode {
				if !ui.Confirm(fmt.Sprintf("Remove provider '%s'?", name), false) {
					ui.Info("Cancelled")
					return nil
				}
			}

			// Get provider to find API key ref
			p := cc.Cfg.GetProvider(name)
			if p == nil {
				return fmt.Errorf("provider not found: %s", name)
			}

			// Remove from config
			if !cc.Cfg.RemoveProvider(name) {
				return fmt.Errorf("failed to remove provider: %s", name)
			}

			// Try to delete API key
			if p.APIKeyRef != "" {
				if _, keyName, ok := strings.Cut(p.APIKeyRef, ":"); ok && keyName != "" {
					_ = cc.SecretsMgr.Delete(keyName)
				}
			}

			// Save config
			if err := cc.SaveConfig(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			ui.Success("Removed provider: %s", name)
			return nil
		},
	}
}
