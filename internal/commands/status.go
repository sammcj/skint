package commands

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// NewStatusCmd creates the status command
func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show installation status",
		Long:  "Display information about the current Skint installation.",
		RunE:  runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)
	version := cmd.Root().Version

	// Get directories
	configDir := cc.ConfigMgr.ConfigDir()
	dataDir, _ := config.GetDataDir()
	cacheDir, _ := config.GetCacheDir()
	binDir, _ := config.GetBinDir()

	// Check if Claude is installed
	claudePath, claudeErr := exec.LookPath("claude")

	// JSON output
	if cc.Cfg.OutputFormat == config.FormatJSON {
		result := map[string]any{
			"version":          version,
			"config_dir":       configDir,
			"data_dir":         dataDir,
			"cache_dir":        cacheDir,
			"bin_dir":          binDir,
			"provider_count":   len(cc.Cfg.Providers),
			"default_provider": cc.Cfg.DefaultProvider,
			"color_enabled":    cc.Cfg.ColorEnabled,
			"output_format":    cc.Cfg.OutputFormat,
			"go_version":       runtime.Version(),
			"platform":         fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		}

		if claudeErr == nil {
			result["claude_installed"] = true
			result["claude_path"] = claudePath
		} else {
			result["claude_installed"] = false
		}

		return cc.Output(result)
	}

	// Plain output
	if cc.Cfg.OutputFormat == config.FormatPlain {
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Config: %s\n", configDir)
		fmt.Printf("Providers: %d\n", len(cc.Cfg.Providers))
		return nil
	}

	// Human-readable output
	fmt.Println()
	ui.Box("SKINT STATUS", 50)
	fmt.Println()

	ui.Log("  Version:     %s", ui.Bold(version))
	ui.Log("  Config:      %s", configDir)
	ui.Log("  Data:        %s", dataDir)
	ui.Log("  Cache:       %s", cacheDir)
	ui.Log("  Bin:         %s", binDir)
	ui.Log("  Platform:    %s/%s", runtime.GOOS, runtime.GOARCH)
	fmt.Println()

	ui.Log("  Providers:   %s configured", ui.Bold(fmt.Sprintf("%d", len(cc.Cfg.Providers))))

	if cc.Cfg.DefaultProvider != "" {
		ui.Log("  Default:     %s", ui.Yellow(cc.Cfg.DefaultProvider))
	}

	if claudeErr == nil {
		ui.Log("  Claude:      %s (%s)", ui.Green("installed"), claudePath)
	} else {
		ui.Log("  Claude:      %s", ui.Red("not found"))
	}

	// Keyring status
	if cc.SecretsMgr != nil && cc.SecretsMgr.IsKeyringAvailable() {
		ui.Log("  Keyring:     %s", ui.Green("available"))
	} else {
		ui.Log("  Keyring:     %s (using file store)", ui.Yellow("unavailable"))
	}

	fmt.Println()

	return nil
}
