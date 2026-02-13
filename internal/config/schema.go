package config

import (
	"fmt"
)

// ConfigVersion is the current configuration file format version
const ConfigVersion = "1.0"

// Config represents the complete Skint configuration
type Config struct {
	Version         string      `yaml:"version" mapstructure:"version"`
	DefaultProvider string      `yaml:"default_provider" mapstructure:"default_provider"`
	OutputFormat    string      `yaml:"output_format" mapstructure:"output_format"`
	ColorEnabled    bool        `yaml:"color_enabled" mapstructure:"color_enabled"`
	NoBanner        bool        `yaml:"no_banner" mapstructure:"no_banner"`
	ClaudeArgs      []string    `yaml:"claude_args,omitempty" mapstructure:"claude_args"`
	Providers       []*Provider `yaml:"providers" mapstructure:"providers"`
}

// Provider represents a single LLM provider configuration
type Provider struct {
	// Core identification
	Name        string `yaml:"name" mapstructure:"name"`
	Type        string `yaml:"type" mapstructure:"type"`
	DisplayName string `yaml:"display_name" mapstructure:"display_name"`
	Description string `yaml:"description" mapstructure:"description"`

	// Connection details
	BaseURL string `yaml:"base_url,omitempty" mapstructure:"base_url"`
	APIKey  string `yaml:"api_key,omitempty" mapstructure:"api_key"` // For migration only

	// API key reference format: "keyring:<name>" or "file:<name>"
	APIKeyRef string `yaml:"api_key_ref,omitempty" mapstructure:"api_key_ref"`

	// Model configuration
	// DefaultModel is the primary model for builtin providers (the provider's default offering).
	// Model is the specific model ID for OpenRouter/custom providers (user-selected).
	// Use EffectiveModel() to get whichever is set.
	DefaultModel  string            `yaml:"default_model,omitempty" mapstructure:"default_model"`
	Model         string            `yaml:"model,omitempty" mapstructure:"model"`
	ModelMappings  map[string]string `yaml:"model_mappings,omitempty" mapstructure:"model_mappings"`

	// Local provider specific
	AuthToken string `yaml:"auth_token,omitempty" mapstructure:"auth_token"`

	// Custom provider specific
	APIType string `yaml:"api_type,omitempty" mapstructure:"api_type"` // "anthropic" or "openai"

	// Env var override for API key (e.g. ANTHROPIC_API_KEY instead of ANTHROPIC_AUTH_TOKEN)
	KeyEnvVar string `yaml:"key_env_var,omitempty" mapstructure:"key_env_var"`

	// Internal: loaded from keyring/file
	resolvedAPIKey string
}

// Provider types
const (
	ProviderTypeBuiltin    = "builtin"
	ProviderTypeOpenRouter = "openrouter"
	ProviderTypeLocal      = "local"
	ProviderTypeCustom     = "custom"
)

// API types for custom providers
const (
	APITypeAnthropic = "anthropic"
	APITypeOpenAI    = "openai"
)

// Output formats
const (
	FormatHuman = "human"
	FormatJSON  = "json"
	FormatPlain = "plain"
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Version == "" {
		c.Version = ConfigVersion
	}

	if c.OutputFormat == "" {
		c.OutputFormat = FormatHuman
	}

	if c.OutputFormat != FormatHuman && c.OutputFormat != FormatJSON && c.OutputFormat != FormatPlain {
		return fmt.Errorf("invalid output format: %s", c.OutputFormat)
	}

	// Validate providers
	names := make(map[string]bool)
	for i, p := range c.Providers {
		if p.Name == "" {
			return fmt.Errorf("provider at index %d has no name", i)
		}
		if names[p.Name] {
			return fmt.Errorf("duplicate provider name: %s", p.Name)
		}
		names[p.Name] = true

		if err := p.Validate(); err != nil {
			return fmt.Errorf("provider %s: %w", p.Name, err)
		}
	}

	// Validate default provider exists
	if c.DefaultProvider != "" {
		if _, ok := names[c.DefaultProvider]; !ok {
			return fmt.Errorf("default provider %s not found in providers list", c.DefaultProvider)
		}
	}

	return nil
}

// Validate checks if the provider configuration is valid
func (p *Provider) Validate() error {
	if p.Type == "" {
		return fmt.Errorf("provider type is required")
	}

	validTypes := map[string]bool{
		ProviderTypeBuiltin:    true,
		ProviderTypeOpenRouter: true,
		ProviderTypeLocal:      true,
		ProviderTypeCustom:     true,
	}
	if !validTypes[p.Type] {
		return fmt.Errorf("invalid provider type: %s", p.Type)
	}

	// Built-in, openrouter, and custom providers need base URL.
	// Exceptions: "native" and "anthropic" use Anthropic's default endpoint.
	if p.Type != ProviderTypeLocal && p.Name != "native" && p.Name != "anthropic" && p.BaseURL == "" {
		return fmt.Errorf("base_url is required for %s providers", p.Type)
	}

	// Custom providers must have a valid API type
	if p.Type == ProviderTypeCustom && p.APIType != "" && p.APIType != APITypeAnthropic && p.APIType != APITypeOpenAI {
		return fmt.Errorf("invalid api_type %q: must be %q or %q", p.APIType, APITypeAnthropic, APITypeOpenAI)
	}

	return nil
}

// GetProvider retrieves a provider by name
func (c *Config) GetProvider(name string) *Provider {
	for _, p := range c.Providers {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// AddProvider adds a provider to the configuration
func (c *Config) AddProvider(p *Provider) error {
	if c.GetProvider(p.Name) != nil {
		return fmt.Errorf("provider %s already exists", p.Name)
	}
	if err := p.Validate(); err != nil {
		return err
	}
	c.Providers = append(c.Providers, p)
	return nil
}

// RemoveProvider removes a provider by name
func (c *Config) RemoveProvider(name string) bool {
	for i, p := range c.Providers {
		if p.Name == name {
			c.Providers = append(c.Providers[:i], c.Providers[i+1:]...)
			return true
		}
	}
	return false
}

// SetResolvedAPIKey sets the resolved API key (from keyring/file)
func (p *Provider) SetResolvedAPIKey(key string) {
	p.resolvedAPIKey = key
}

// GetAPIKey returns the resolved API key
func (p *Provider) GetAPIKey() string {
	return p.resolvedAPIKey
}

// EffectiveModel returns the model to use: DefaultModel for builtin providers,
// Model for OpenRouter/custom providers, whichever is set.
func (p *Provider) EffectiveModel() string {
	if p.DefaultModel != "" {
		return p.DefaultModel
	}
	return p.Model
}

// NeedsAPIKey returns true if this provider requires an API key.
// Local providers and the native Anthropic provider do not need one.
func (p *Provider) NeedsAPIKey() bool {
	return p.Type != ProviderTypeLocal && p.Name != "native"
}

// IsConfigured returns true if this provider has been fully configured.
// Checks APIKeyRef (persisted reference) rather than the runtime-resolved key,
// so it works correctly for providers configured during the current session.
func (p *Provider) IsConfigured() bool {
	if !p.NeedsAPIKey() {
		return true
	}
	return p.APIKeyRef != "" || p.resolvedAPIKey != ""
}

// NewDefaultConfig creates a new configuration with sensible defaults
func NewDefaultConfig() *Config {
	return &Config{
		Version:      ConfigVersion,
		OutputFormat: FormatHuman,
		ColorEnabled: true,
		NoBanner:     false,
		Providers:    []*Provider{},
	}
}
