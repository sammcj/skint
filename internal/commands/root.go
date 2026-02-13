package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/secrets"
	"github.com/sammcj/skint/internal/tui"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// RootCmd is the root command
type RootCmd struct {
	*cobra.Command
}

// NewRootCmd creates the root command
func NewRootCmd(version string) *RootCmd {
	cc := &CmdContext{
		OutputFormat: "human",
	}

	root := &cobra.Command{
		Use:   "skint",
		Short: "Multi-provider launcher for Claude Code",
		Long: `Skint - One CLI to switch between Claude Code providers instantly.

Skint makes it easy to use Claude Code with different LLM providers
like Z.AI, MiniMax, Kimi, DeepSeek, OpenRouter, and local models via
Ollama, LM Studio, or llama.cpp.`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Ensure context is set before initialize runs
			cmd.SetContext(context.WithValue(cmd.Context(), ctxKey, cc))
			return initialize(cc)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := GetContext(cmd)
			return tui.RunInteractive(cc.Cfg, cc.SecretsMgr, cc.SaveConfig, cc.LaunchClaude)
		},
	}

	// Ensure a base context exists for the root command
	root.SetContext(context.Background())

	// Bind flags directly to CmdContext fields
	root.PersistentFlags().StringVar(&cc.cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/skint/config.yaml)")
	root.PersistentFlags().BoolVarP(&cc.Verbose, "verbose", "v", false, "verbose output")
	root.PersistentFlags().BoolVarP(&cc.Quiet, "quiet", "q", false, "minimal output")
	root.PersistentFlags().BoolVarP(&cc.YesMode, "yes", "y", false, "auto-confirm prompts")
	root.PersistentFlags().BoolVar(&cc.NoInput, "no-input", false, "non-interactive mode")
	root.PersistentFlags().BoolVar(&cc.NoColor, "no-color", false, "disable colours")
	root.PersistentFlags().BoolVar(&cc.NoBanner, "no-banner", false, "hide banner")
	root.PersistentFlags().StringVar(&cc.OutputFormat, "output", "human", "output format: human, json, plain")
	root.PersistentFlags().StringVar(&cc.BinDir, "bin-dir", "", "binary directory (default is ~/.local/bin on Linux, ~/bin on macOS)")

	return &RootCmd{root}
}

// initialize sets up the configuration and secrets managers
func initialize(cc *CmdContext) error {
	// Handle environment variable overrides
	if os.Getenv("SKINT_VERBOSE") == "1" {
		cc.Verbose = true
	}
	if os.Getenv("SKINT_QUIET") == "1" {
		cc.Quiet = true
	}
	if os.Getenv("SKINT_YES") == "1" {
		cc.YesMode = true
	}
	if os.Getenv("SKINT_NO_INPUT") == "1" {
		cc.NoInput = true
	}
	if os.Getenv("NO_COLOR") != "" {
		cc.NoColor = true
	}
	if os.Getenv("SKINT_NO_BANNER") == "1" {
		cc.NoBanner = true
	}
	if v := os.Getenv("SKINT_OUTPUT_FORMAT"); v != "" {
		cc.OutputFormat = v
	}

	// Create config manager
	var err error
	if cc.cfgFile != "" {
		cc.ConfigMgr, err = config.NewManagerWithPath(cc.cfgFile)
	} else {
		cc.ConfigMgr, err = config.NewManager()
	}
	if err != nil {
		return fmt.Errorf("failed to initialise config: %w", err)
	}

	// Load config
	if err := cc.ConfigMgr.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cc.Cfg = cc.ConfigMgr.Get()

	// Apply CLI flags to config
	if cc.NoColor {
		cc.Cfg.ColorEnabled = false
	}
	if cc.NoBanner {
		cc.Cfg.NoBanner = true
	}
	if cc.OutputFormat != "" {
		cc.Cfg.OutputFormat = cc.OutputFormat
	}

	// Initialise UI
	ui.Init(cc.Cfg)

	// Create secrets manager
	cc.SecretsMgr, err = secrets.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialise secrets: %w", err)
	}

	// Check for old installation and offer migration
	migration, err := config.NewMigration()
	if err != nil {
		return err
	}
	if migration.HasOldInstallation() && !cc.NoInput && !cc.CfgFileExists() {
		// Auto-migrate in quiet mode
		if cc.Quiet {
			if err := cc.RunMigration(); err != nil {
				return fmt.Errorf("auto-migration failed: %w", err)
			}
		} else {
			ui.Info("Existing Skint installation detected.")
			if ui.Confirm("Migrate from old version?", true) {
				if err := cc.RunMigration(); err != nil {
					return fmt.Errorf("migration failed: %w", err)
				}
			}
		}
	}

	// Load API keys for providers
	cc.LoadProviderKeys()

	return nil
}
