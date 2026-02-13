package commands

import (
	"fmt"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

// NewInfoCmd creates the info command
func NewInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <provider>",
		Short: "Show provider details",
		Long:  "Display detailed information about a specific provider.",
		Args:  cobra.ExactArgs(1),
		RunE:  runInfo,
	}
}

func runInfo(cmd *cobra.Command, args []string) error {
	cc := GetContext(cmd)
	name := args[0]

	p := cc.Cfg.GetProvider(name)
	if p == nil {
		return fmt.Errorf("provider not found: %s", name)
	}

	// JSON output
	if cc.Cfg.OutputFormat == config.FormatJSON {
		configured := true
		if p.NeedsAPIKey() && p.GetAPIKey() == "" {
			configured = false
		}

		return cc.Output(map[string]any{
			"name":           p.Name,
			"display_name":   p.DisplayName,
			"description":    p.Description,
			"type":           p.Type,
			"base_url":       p.BaseURL,
			"api_key_ref":    p.APIKeyRef,
			"default_model":  p.DefaultModel,
			"model":          p.Model,
			"model_mappings": p.ModelMappings,
			"configured":     configured,
		})
	}

	// Plain output
	if cc.Cfg.OutputFormat == config.FormatPlain {
		fmt.Printf("Name: %s\n", p.Name)
		fmt.Printf("Type: %s\n", p.Type)
		fmt.Printf("BaseURL: %s\n", p.BaseURL)
		return nil
	}

	// Human-readable output
	fmt.Println()
	ui.Log("%s: %s", ui.Bold("Provider"), ui.Yellow(p.Name))
	ui.Separator(40)

	if p.DisplayName != "" {
		ui.Log("Display Name: %s", p.DisplayName)
	}

	if p.Description != "" {
		ui.Log("Description:  %s", p.Description)
	}

	ui.Log("Type:         %s", p.Type)

	if p.BaseURL != "" {
		ui.Log("Base URL:     %s", p.BaseURL)
	}

	model := p.EffectiveModel()
	if model != "" {
		ui.Log("Model:        %s", model)
	}

	if p.NeedsAPIKey() {
		if p.GetAPIKey() != "" {
			ui.Log("API Key:      %s", ui.Green("configured"))
			if cc.Verbose {
				ui.Dim("              %s\n", ui.MaskKey(p.GetAPIKey()))
			}
		} else {
			ui.Log("API Key:      %s", ui.Red("not set"))
		}
	}

	if p.AuthToken != "" {
		ui.Log("Auth Token:   %s", ui.Green("configured"))
	}

	if len(p.ModelMappings) > 0 {
		ui.Log("Model Mappings:")
		for tier, model := range p.ModelMappings {
			ui.Dim("  %s: %s\n", tier, model)
		}
	}

	fmt.Println()

	return nil
}
