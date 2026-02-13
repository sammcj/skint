package commands

import (
	"fmt"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured providers",
		Long:    "Display a list of all configured LLM providers.",
		RunE:    runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)

	if len(cc.Cfg.Providers) == 0 {
		if cc.Cfg.OutputFormat == config.FormatJSON {
			fmt.Println(`{"providers":[]}`)
			return nil
		}
		ui.Warning("No providers configured")
		ui.NextSteps([]string{
			"Configure a provider: " + ui.Green("skint config"),
		})
		return nil
	}

	// JSON output
	if cc.Cfg.OutputFormat == config.FormatJSON {
		type providerJSON struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			Type        string `json:"type"`
			BaseURL     string `json:"base_url,omitempty"`
			Model       string `json:"model,omitempty"`
			Configured  bool   `json:"configured"`
		}

		var result []providerJSON
		for _, p := range cc.Cfg.Providers {
			configured := true
			if p.NeedsAPIKey() && p.GetAPIKey() == "" {
				configured = false
			}

			model := p.EffectiveModel()

			result = append(result, providerJSON{
				Name:        p.Name,
				DisplayName: p.DisplayName,
				Type:        p.Type,
				BaseURL:     p.BaseURL,
				Model:       model,
				Configured:  configured,
			})
		}

		return cc.Output(map[string]any{"providers": result})
	}

	// Plain output
	if cc.Cfg.OutputFormat == config.FormatPlain {
		for _, p := range cc.Cfg.Providers {
			fmt.Println(p.Name)
		}
		return nil
	}

	// Human-readable output
	ui.Log("\n%s (%d):\n", ui.Bold("Available Providers"), len(cc.Cfg.Providers))

	for _, p := range cc.Cfg.Providers {
		// Check if configured
		configured := true
		if p.NeedsAPIKey() && p.GetAPIKey() == "" {
			configured = false
		}

		ui.ListItem(configured, "%s", ui.Yellow(p.Name))

		if p.DisplayName != "" && p.DisplayName != p.Name {
			ui.Dim("          %s\n", p.DisplayName)
		}

		if p.Description != "" {
			ui.Dim("          %s\n", p.Description)
		}

		model := p.EffectiveModel()
		if model != "" {
			ui.Dim("          Model: %s\n", model)
		}
	}

	ui.Log("")
	ui.Log("Run: %s", ui.Green("skint use <name>"))

	return nil
}
