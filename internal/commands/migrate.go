package commands

import (
	"fmt"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// NewMigrateCmd creates the migrate command
func NewMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from old bash version",
		Long: `Migrate configuration and API keys from the old bash version of Skint.

This imports:
  - API keys from ~/.local/share/skint/secrets.env
  - Provider configurations
  - Creates new YAML config file`,
		RunE: runMigrate,
	}

	cmd.Flags().Bool("import-secrets", true, "Import secrets from old installation")
	cmd.Flags().Bool("keep-old", false, "Keep old files after migration")

	return cmd
}

func runMigrate(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)
	importSecrets, _ := cmd.Flags().GetBool("import-secrets")
	keepOld, _ := cmd.Flags().GetBool("keep-old")

	// Check for old installation
	migration, err := config.NewMigration()
	if err != nil {
		return err
	}
	if !migration.HasOldInstallation() {
		return fmt.Errorf("no old installation found at %s", migration.SecretsFile())
	}

	// JSON output
	if cc.Cfg.OutputFormat == config.FormatJSON {
		newCfg, keys, err := migration.Import()
		if err != nil {
			return err
		}

		return cc.Output(map[string]any{
			"can_migrate": true,
			"providers":   len(newCfg.Providers),
			"secrets":     len(keys),
		})
	}

	// Plain output
	if cc.Cfg.OutputFormat == config.FormatPlain {
		fmt.Println("Migration available from old bash version")
		return nil
	}

	// Human-readable output
	fmt.Println()
	ui.Log("%s", ui.Bold("Migrate from old version"))
	fmt.Println()

	// Show what will be imported
	newCfg, keys, err := migration.Import()
	if err != nil {
		return fmt.Errorf("failed to analyse old installation: %w", err)
	}

	ui.Log("Found %d providers with %d API keys", len(newCfg.Providers), len(keys))
	fmt.Println()

	if importSecrets {
		// Confirm
		if !cc.YesMode {
			if !ui.Confirm("Proceed with migration?", true) {
				ui.Info("Cancelled")
				return nil
			}
		}

		// Run migration
		if err := cc.RunMigration(); err != nil {
			return err
		}

		// Clean up old files if requested
		if !keepOld && !cc.YesMode {
			fmt.Println()
			if ui.Confirm("Remove old installation files?", true) {
				if err := migration.Cleanup(); err != nil {
					ui.Warning("Failed to clean up: %v", err)
				} else {
					ui.Success("Old files removed")
				}
			}
		} else if !keepOld {
			_ = migration.Cleanup()
		}
	}

	return nil
}
