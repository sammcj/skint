package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/launcher"
	"github.com/sammcj/skint/internal/providers"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// NewGenerateCmd creates the generate-scripts command
func NewGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate-scripts",
		Short: "Generate shell scripts for providers",
		Long: `Generate legacy shell scripts for all configured providers.

This creates scripts like 'skintai' in your bin directory for
backward compatibility with the old bash version.`,
		RunE: runGenerate,
	}
}

func runGenerate(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)

	// Get bin directory
	binDir, err := config.GetBinDir()
	if err != nil {
		return fmt.Errorf("failed to get bin directory: %w", err)
	}

	// Ensure bin directory exists
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Generate scripts for all providers
	generated := 0
	failed := 0

	for _, p := range cc.Cfg.Providers {
		// Load API key if needed
		if p.NeedsAPIKey() && p.GetAPIKey() == "" && p.APIKeyRef != "" {
			key, err := cc.SecretsMgr.RetrieveByReference(p.APIKeyRef)
			if err != nil {
				if cc.Verbose {
					ui.Warning("Skipping %s: API key not available", p.Name)
				}
				failed++
				continue
			}
			p.SetResolvedAPIKey(key)
		}

		// Generate script
		provider, err := providers.FromConfig(p)
		if err != nil {
			if cc.Verbose {
				ui.Warning("Skipping %s: %v", p.Name, err)
			}
			failed++
			continue
		}
		if err := launcher.GenerateScript(provider, binDir); err != nil {
			if cc.Verbose {
				ui.Warning("Failed to generate script for %s: %v", p.Name, err)
			}
			failed++
			continue
		}

		generated++
	}

	// Also generate the main skint command script
	if err := generateMainScript(binDir); err != nil {
		ui.Warning("Failed to generate main script: %v", err)
	}

	// Save banner
	if err := saveBanner(); err != nil && cc.Verbose {
		ui.Warning("Failed to save banner: %v", err)
	}

	// Output results
	if cc.Cfg.OutputFormat == config.FormatJSON {
		return cc.Output(map[string]any{
			"generated": generated,
			"failed":    failed,
			"bin_dir":   binDir,
		})
	}

	if cc.Cfg.OutputFormat == config.FormatPlain {
		fmt.Printf("Generated %d scripts in %s\n", generated, binDir)
		return nil
	}

	// Human-readable
	fmt.Println()
	ui.Success("Generated %d scripts in %s", generated, binDir)

	if failed > 0 {
		ui.Warning("Failed to generate %d scripts", failed)
	}

	// Check PATH
	path := os.Getenv("PATH")
	containsBinDir := false
	for _, p := range filepath.SplitList(path) {
		if p == binDir {
			containsBinDir = true
			break
		}
	}

	if !containsBinDir {
		ui.Warning("\n'%s' is not in your PATH.", binDir)
		ui.Info("Add it to your shell profile:")
		ui.Dim("  export PATH=\"%s:$PATH\"\n", binDir)
	}

	return nil
}

func generateMainScript(binDir string) error {
	scriptPath := filepath.Join(binDir, "skint")

	script := `#!/usr/bin/env bash
# Skint - Multi-provider launcher for Claude Code
# Generated wrapper script

set -euo pipefail

# Find the actual skint binary
SKINT_BIN=""

# Check if skint is in PATH
if command -v skint >/dev/null; then
    SKINT_BIN=$(command -v skint)
fi

# Fallback to common locations
if [[ -z "$SKINT_BIN" ]]; then
    for dir in "$HOME/.local/bin" "$HOME/bin" "/usr/local/bin"; do
        if [[ -x "$dir/skint" ]]; then
            SKINT_BIN="$dir/skint"
            break
        fi
    done
fi

if [[ -z "$SKINT_BIN" ]]; then
    echo "Error: skint binary not found" >&2
    echo "Please install Skint: https://github.com/sammcj/skint" >&2
    exit 1
fi

exec "$SKINT_BIN" "$@"
`

	return os.WriteFile(scriptPath, []byte(script), 0755)
}

func saveBanner() error {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	banner := `  ____ _       _   _
 / ___| | ___ | |_| |__   ___ _ __
| |   | |/ _ \| __| '_ \ / _ \ '__|
| |___| | (_) | |_| | | |  __/ |
 \____|_|\___/ \__|_| |_|\___|_|
`

	bannerPath := filepath.Join(dataDir, "banner")
	return os.WriteFile(bannerPath, []byte(banner), 0644)
}
