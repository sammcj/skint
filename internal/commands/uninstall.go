package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// NewUninstallCmd creates the uninstall command
func NewUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Skint completely",
		Long: `Remove all Skint configuration, data, and generated files.

This will delete:
  - Configuration directory (~/.config/skint
  - Data directory (~/.local/share/skint
  - Cache directory (~/.cache/skint
  - Generated scripts (skint-*)`,
		RunE: runUninstall,
	}
}

func runUninstall(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)

	// Get directories
	configDir := cc.ConfigMgr.ConfigDir()
	dataDir, _ := config.GetDataDir()
	cacheDir, _ := config.GetCacheDir()
	binDir, _ := config.GetBinDir()

	// JSON output
	if cc.Cfg.OutputFormat == config.FormatJSON {
		return cc.Output(map[string]any{
			"would_remove": []string{
				configDir,
				dataDir,
				cacheDir,
				binDir + "/skint*",
			},
		})
	}

	// Plain output
	if cc.Cfg.OutputFormat == config.FormatPlain {
		fmt.Println("Would remove:")
		fmt.Printf("  %s\n", configDir)
		fmt.Printf("  %s\n", dataDir)
		fmt.Printf("  %s\n", cacheDir)
		fmt.Printf("  %s/skint*\n", binDir)
		return nil
	}

	// Human-readable output
	fmt.Println()
	ui.Log("%s", ui.Bold("Uninstall Skint"))
	fmt.Println()
	ui.Log("This will remove:")
	ui.Dim("  %s %s\n", ui.Sym.Arrow, configDir)
	ui.Dim("  %s %s\n", ui.Sym.Arrow, dataDir)
	ui.Dim("  %s %s\n", ui.Sym.Arrow, cacheDir)
	ui.Dim("  %s %s/skint*\n", ui.Sym.Arrow, binDir)
	fmt.Println()

	// Confirm
	if !cc.YesMode {
		if !ui.ConfirmDanger("Remove all Skint files", "delete skint") {
			ui.Info("Cancelled")
			return nil
		}
	}

	// Spinner
	spinner := ui.NewSpinner("Removing files...")
	spinner.Start()

	// Remove directories
	dirs := []string{configDir, dataDir, cacheDir}
	for _, dir := range dirs {
		if dir != "" {
			_ = os.RemoveAll(dir)
		}
	}

	// Remove scripts from bin directory
	if binDir != "" {
		entries, _ := os.ReadDir(binDir)
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "skint-") || name == "skint" {
				_ = os.Remove(filepath.Join(binDir, name))
			}
		}
	}

	spinner.Stop(true)

	ui.Success("Skint uninstalled")
	return nil
}
