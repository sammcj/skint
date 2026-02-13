package main

import (
	"fmt"
	"os"

	"github.com/sammcj/skint/internal/commands"
)

// version is set at build time via ldflags
var version = "dev"

func main() {
	// Create root command
	rootCmd := commands.NewRootCmd(version)

	// Add subcommands
	rootCmd.AddCommand(commands.NewConfigCmd())
	rootCmd.AddCommand(commands.NewUseCmd())
	rootCmd.AddCommand(commands.NewExecCmd())
	rootCmd.AddCommand(commands.NewListCmd())
	rootCmd.AddCommand(commands.NewInfoCmd())
	rootCmd.AddCommand(commands.NewTestCmd())
	rootCmd.AddCommand(commands.NewStatusCmd())
	rootCmd.AddCommand(commands.NewGenerateCmd())
	rootCmd.AddCommand(commands.NewMigrateCmd())
	rootCmd.AddCommand(commands.NewUninstallCmd())

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
