package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/launcher"
	"github.com/sammcj/skint/internal/providers"
	"github.com/sammcj/skint/internal/secrets"
	"github.com/sammcj/skint/internal/ui"
	"github.com/spf13/cobra"
)

type ctxKeyType struct{}

var ctxKey = ctxKeyType{}

// CmdContext holds all shared state for command execution, replacing package-level globals.
type CmdContext struct {
	ConfigMgr    *config.Manager
	SecretsMgr   *secrets.Manager
	Cfg          *config.Config
	Verbose      bool
	Quiet        bool
	YesMode      bool
	NoInput      bool
	NoColor      bool
	NoBanner     bool
	OutputFormat string
	BinDir       string

	// cfgFile is the user-supplied config path (empty = default)
	cfgFile string
}

// GetContext extracts the CmdContext from a cobra command's context.
func GetContext(cmd *cobra.Command) *CmdContext {
	cc, ok := cmd.Context().Value(ctxKey).(*CmdContext)
	if !ok {
		panic("commands.GetContext: CmdContext not found in command context -- was PersistentPreRunE skipped?")
	}
	return cc
}

// SetContext stores a CmdContext in the cobra command's context.
func SetContext(cmd *cobra.Command, cc *CmdContext) {
	cmd.SetContext(context.WithValue(cmd.Context(), ctxKey, cc))
}

// SaveConfig saves the current configuration to disk.
func (cc *CmdContext) SaveConfig() error {
	return cc.ConfigMgr.Save()
}

// Output formats data according to the configured output format.
func (cc *CmdContext) Output(data any) error {
	switch cc.Cfg.OutputFormat {
	case config.FormatJSON:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case config.FormatPlain:
		if m, ok := data.(map[string]any); ok {
			for k, v := range m {
				fmt.Printf("%s: %v\n", k, v)
			}
		} else {
			fmt.Printf("%v\n", data)
		}
	default:
		// Human format - handled by caller
	}
	return nil
}

// ResolveProvider looks up a provider by name from cfg or the built-in registry,
// loads its API key if needed, and returns the config.Provider ready for use.
func (cc *CmdContext) ResolveProvider(name string) (*config.Provider, error) {
	p := cc.Cfg.GetProvider(name)
	if p == nil {
		// Check if it's a built-in that hasn't been configured yet
		registry := providers.NewRegistry()
		def, ok := registry.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown provider: %s. Run 'skint list' to see available providers", name)
		}

		p = &config.Provider{
			Name:         def.Name,
			Type:         def.Type,
			DisplayName:  def.DisplayName,
			Description:  def.Description,
			BaseURL:      def.BaseURL,
			DefaultModel: def.DefaultModel,
			AuthToken:    def.AuthToken,
		}

		// For non-local providers, try to load a stored key
		if def.Type != config.ProviderTypeLocal && def.KeyVar != "" {
			ref := secrets.StorageTypeKeyring + ":" + name
			if !cc.SecretsMgr.IsKeyringAvailable() {
				ref = secrets.StorageTypeFile + ":" + name
			}
			p.APIKeyRef = ref
			key, err := cc.SecretsMgr.Retrieve(name)
			if err != nil {
				return nil, fmt.Errorf("provider %s not configured. Run 'skint config %s' to set it up", name, name)
			}
			p.SetResolvedAPIKey(key)
		}
	}

	// Load API key if needed and not already loaded
	if p.NeedsAPIKey() && p.GetAPIKey() == "" && p.APIKeyRef != "" {
		key, err := cc.SecretsMgr.RetrieveByReference(p.APIKeyRef)
		if err != nil {
			return nil, fmt.Errorf("failed to load API key for %s: %w", name, err)
		}
		p.SetResolvedAPIKey(key)
	}

	return p, nil
}

// LoadProviderKeys loads API keys for all configured providers.
func (cc *CmdContext) LoadProviderKeys() {
	for _, p := range cc.Cfg.Providers {
		if p.APIKeyRef == "" {
			continue
		}

		key, err := cc.SecretsMgr.RetrieveByReference(p.APIKeyRef)
		if err != nil {
			if cc.Verbose {
				ui.Warning("Failed to load key for %s: %v", p.Name, err)
			}
			continue
		}

		p.SetResolvedAPIKey(key)
	}
}

// CfgFileExists checks if the config file exists.
func (cc *CmdContext) CfgFileExists() bool {
	if cc.ConfigMgr == nil {
		return false
	}
	return cc.ConfigMgr.Exists()
}

// RunMigration migrates from the old bash version.
func (cc *CmdContext) RunMigration() error {
	migration, err := config.NewMigration()
	if err != nil {
		return err
	}
	newCfg, keys, err := migration.Import()
	if err != nil {
		return err
	}

	// Store all keys
	for providerName, apiKey := range keys {
		if _, err := cc.SecretsMgr.StoreWithReference(providerName, apiKey); err != nil {
			return fmt.Errorf("failed to store key for %s: %w", providerName, err)
		}
	}

	// Update API key references in config
	for _, p := range newCfg.Providers {
		if _, ok := keys[p.Name]; ok {
			if cc.SecretsMgr.IsKeyringAvailable() {
				p.APIKeyRef = fmt.Sprintf("keyring:%s", p.Name)
			} else {
				p.APIKeyRef = fmt.Sprintf("file:%s", p.Name)
			}
		}
	}

	// Merge with existing config if any
	if cc.Cfg != nil && len(cc.Cfg.Providers) > 0 {
		for _, p := range newCfg.Providers {
			if cc.Cfg.GetProvider(p.Name) == nil {
				cc.Cfg.Providers = append(cc.Cfg.Providers, p)
			}
		}
	} else {
		cc.Cfg = newCfg
	}

	cc.ConfigMgr.Set(cc.Cfg)

	// Save config
	if err := cc.ConfigMgr.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ui.Success("Migration complete! Migrated %d providers.", len(keys))

	// Offer to clean up old files
	if !cc.NoInput && !cc.Quiet {
		if ui.Confirm("Remove old installation files?", true) {
			if err := migration.Cleanup(); err != nil {
				ui.Warning("Failed to clean up old files: %v", err)
			} else {
				ui.Success("Old files removed.")
			}
		}
	}

	return nil
}

// LaunchClaude launches Claude Code with the specified provider's env vars.
// If providerName is empty, launches claude without any provider overrides (native).
// Uses cfg.ClaudeArgs as default arguments to the claude command.
func (cc *CmdContext) LaunchClaude(providerName string) error {
	if err := launcher.CheckClaude(); err != nil {
		return err
	}

	args := append([]string{}, cc.Cfg.ClaudeArgs...)

	if providerName == "" {
		// Native: launch claude without provider env vars
		l, err := launcher.New(cc.Cfg)
		if err != nil {
			return fmt.Errorf("failed to create launcher: %w", err)
		}
		return l.LaunchNative(args)
	}

	// Resolve provider and launch
	p, err := cc.ResolveProvider(providerName)
	if err != nil {
		return err
	}

	provider, err := providers.FromConfig(p)
	if err != nil {
		return fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	l, err := launcher.New(cc.Cfg)
	if err != nil {
		return fmt.Errorf("failed to create launcher: %w", err)
	}

	return l.Launch(provider, args)
}
