package providers

import (
	"fmt"

	"github.com/sammcj/skint/internal/config"
)

// Provider interface defines the methods all providers must implement
type Provider interface {
	// Name returns the provider's short name (e.g., "zai")
	Name() string

	// DisplayName returns the human-readable name (e.g., "Z.AI")
	DisplayName() string

	// Description returns a brief description
	Description() string

	// Type returns the provider type
	Type() string

	// BaseURL returns the API base URL
	BaseURL() string

	// GetAPIKey returns the API key (may be empty for local providers)
	GetAPIKey() string

	// SetAPIKey sets the API key
	SetAPIKey(key string)

	// NeedsAPIKey returns true if this provider requires an API key
	NeedsAPIKey() bool

	// GetEnvVars returns the environment variables to set for Claude
	GetEnvVars() map[string]string

	// GetModel returns the model to use (may be empty for default)
	GetModel() string

	// Validate checks if the provider is properly configured
	Validate() error
}

// baseProvider contains common provider functionality
type baseProvider struct {
	name          string
	displayName   string
	description   string
	providerType  string
	baseURL       string
	apiKey        string
	model         string
	modelMappings map[string]string
	needsAPIKey   bool
	keyEnvVar     string // env var name for API key (default: ANTHROPIC_AUTH_TOKEN)
}

func (p *baseProvider) Name() string {
	return p.name
}

func (p *baseProvider) DisplayName() string {
	if p.displayName != "" {
		return p.displayName
	}
	return p.name
}

func (p *baseProvider) Description() string {
	return p.description
}

func (p *baseProvider) Type() string {
	return p.providerType
}

func (p *baseProvider) BaseURL() string {
	return p.baseURL
}

func (p *baseProvider) GetAPIKey() string {
	return p.apiKey
}

func (p *baseProvider) SetAPIKey(key string) {
	p.apiKey = key
}

func (p *baseProvider) NeedsAPIKey() bool {
	return p.needsAPIKey
}

func (p *baseProvider) GetModel() string {
	return p.model
}

func (p *baseProvider) Validate() error {
	if p.name == "" {
		return fmt.Errorf("provider name is required")
	}
	if p.needsAPIKey && p.apiKey == "" {
		return fmt.Errorf("API key is required for %s", p.name)
	}
	return nil
}

// BuiltinProvider is a standard cloud provider with API key
type BuiltinProvider struct {
	baseProvider
}

// GetEnvVars returns the environment variables for Claude
func (p *BuiltinProvider) GetEnvVars() map[string]string {
	env := make(map[string]string)

	if p.baseURL != "" {
		env["ANTHROPIC_BASE_URL"] = p.baseURL
	}

	if p.apiKey != "" {
		envVar := "ANTHROPIC_AUTH_TOKEN"
		if p.keyEnvVar != "" {
			envVar = p.keyEnvVar
		}
		env[envVar] = p.apiKey
	}

	if p.model != "" {
		env["ANTHROPIC_MODEL"] = p.model
	}

	// Add model mappings
	for tier, model := range p.modelMappings {
		switch tier {
		case "haiku":
			env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
		case "sonnet":
			env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
		case "opus":
			env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
		case "small":
			env["ANTHROPIC_SMALL_FAST_MODEL"] = model
		}
	}

	return env
}

// OpenRouterProvider is an OpenRouter model provider
type OpenRouterProvider struct {
	baseProvider
}

// GetEnvVars returns the environment variables for Claude with OpenRouter
func (p *OpenRouterProvider) GetEnvVars() map[string]string {
	env := make(map[string]string)

	// OpenRouter uses native Anthropic API format
	env["ANTHROPIC_BASE_URL"] = "https://openrouter.ai/api"
	env["ANTHROPIC_AUTH_TOKEN"] = p.apiKey
	// ANTHROPIC_API_KEY must be explicitly set to empty so Claude Code doesn't
	// use a real Anthropic key from the user's environment, which would bypass
	// the OpenRouter proxy.
	env["ANTHROPIC_API_KEY"] = ""

	// Override all model tiers to use the selected model
	if p.model != "" {
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = p.model
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = p.model
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = p.model
		env["ANTHROPIC_SMALL_FAST_MODEL"] = p.model
	}

	return env
}

// LocalProvider is a local model provider (Ollama, LM Studio, etc.)
type LocalProvider struct {
	baseProvider
	authToken string
}

// GetEnvVars returns the environment variables for local providers
func (p *LocalProvider) GetEnvVars() map[string]string {
	env := make(map[string]string)

	env["ANTHROPIC_BASE_URL"] = p.baseURL

	if p.authToken != "" {
		env["ANTHROPIC_AUTH_TOKEN"] = p.authToken
		env["ANTHROPIC_API_KEY"] = ""
	}

	if p.model != "" {
		env["ANTHROPIC_MODEL"] = p.model
	}

	return env
}

// CustomProvider is a user-defined custom provider
type CustomProvider struct {
	baseProvider
	apiType string
}

// GetEnvVars returns the environment variables for custom providers
func (p *CustomProvider) GetEnvVars() map[string]string {
	env := make(map[string]string)

	switch p.apiType {
	case config.APITypeOpenAI:
		// OpenAI-compatible endpoint
		if p.baseURL != "" {
			env["OPENAI_BASE_URL"] = p.baseURL
		}
		if p.apiKey != "" {
			env["OPENAI_API_KEY"] = p.apiKey
		}
		if p.model != "" {
			env["OPENAI_MODEL"] = p.model
		}
	default:
		// Anthropic-compatible endpoint (default)
		if p.baseURL != "" {
			env["ANTHROPIC_BASE_URL"] = p.baseURL
		}
		if p.apiKey != "" {
			env["ANTHROPIC_AUTH_TOKEN"] = p.apiKey
		}
		if p.model != "" {
			env["ANTHROPIC_MODEL"] = p.model
		}
	}

	return env
}

// FromConfig creates a Provider from a config.Provider.
// Returns an error if the provider type is unknown.
func FromConfig(cp *config.Provider) (Provider, error) {
	bp := baseProvider{
		name:          cp.Name,
		displayName:   cp.DisplayName,
		description:   cp.Description,
		providerType:  cp.Type,
		baseURL:       cp.BaseURL,
		apiKey:        cp.GetAPIKey(),
		model:         cp.DefaultModel,
		modelMappings: cp.ModelMappings,
		needsAPIKey:   cp.NeedsAPIKey(),
		keyEnvVar:     cp.KeyEnvVar,
	}

	switch cp.Type {
	case config.ProviderTypeBuiltin:
		return &BuiltinProvider{baseProvider: bp}, nil
	case config.ProviderTypeOpenRouter:
		bp.model = cp.Model // OpenRouter uses Model field
		return &OpenRouterProvider{baseProvider: bp}, nil
	case config.ProviderTypeLocal:
		return &LocalProvider{
			baseProvider: bp,
			authToken:    cp.AuthToken,
		}, nil
	case config.ProviderTypeCustom:
		// For custom providers, use Model field if DefaultModel is empty
		if bp.model == "" && cp.Model != "" {
			bp.model = cp.Model
		}
		return &CustomProvider{
			baseProvider: bp,
			apiType:      cp.APIType,
		}, nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cp.Type)
	}
}

// Registry contains all built-in provider definitions
type Registry struct {
	definitions map[string]*Definition
}

// Definition is a provider template
type Definition struct {
	Name          string
	DisplayName   string
	Description   string
	Type          string
	BaseURL       string
	DefaultModel  string
	ModelMappings map[string]string
	AuthToken     string // For local providers
	KeyVar        string // Environment variable name for API key
	KeyEnvVar     string // env var name to set for Claude (default: ANTHROPIC_AUTH_TOKEN)
}

// NewRegistry creates a new provider registry with built-in definitions
func NewRegistry() *Registry {
	r := &Registry{
		definitions: make(map[string]*Definition),
	}
	r.registerBuiltins()
	return r
}

// Get retrieves a provider definition by name
func (r *Registry) Get(name string) (*Definition, bool) {
	def, ok := r.definitions[name]
	return def, ok
}

// List returns all provider definitions
func (r *Registry) List() []*Definition {
	defs := make([]*Definition, 0, len(r.definitions))
	for _, def := range r.definitions {
		defs = append(defs, def)
	}
	return defs
}

// GroupedList returns providers grouped by category
func (r *Registry) GroupedList() map[string][]*Definition {
	groups := map[string][]*Definition{
		"Native":        {},
		"International": {},
		"Local":         {},
	}

	for _, def := range r.definitions {
		switch def.Name {
		case "native", "anthropic":
			groups["Native"] = append(groups["Native"], def)
		case "ollama", "lmstudio", "llamacpp":
			groups["Local"] = append(groups["Local"], def)
		default:
			groups["International"] = append(groups["International"], def)
		}
	}

	return groups
}

func (r *Registry) registerBuiltins() {
	builtins := []*Definition{
		{
			Name:        "native",
			DisplayName: "Claude Subscription",
			Description: "Uses your Claude subscription (no config needed)",
			Type:        config.ProviderTypeBuiltin,
		},
		{
			Name:        "anthropic",
			DisplayName: "Anthropic API",
			Description: "Direct Anthropic API access",
			Type:        config.ProviderTypeBuiltin,
			KeyVar:      "ANTHROPIC_API_KEY",
			KeyEnvVar:   "ANTHROPIC_API_KEY",
		},
		{
			Name:        "openrouter",
			DisplayName: "OpenRouter",
			Description: "OpenRouter API gateway (access multiple models)",
			Type:        config.ProviderTypeOpenRouter,
			BaseURL:     "https://openrouter.ai/api",
			KeyVar:      "OPENROUTER_API_KEY",
		},
		{
			Name:          "zai",
			DisplayName:   "Z.AI",
			Description:   "Z.AI International (GLM-5)",
			Type:          config.ProviderTypeBuiltin,
			BaseURL:       "https://api.z.ai/api/anthropic",
			DefaultModel:  "glm-5",
			ModelMappings: map[string]string{"haiku": "glm-5", "sonnet": "glm-5", "opus": "glm-5"},
			KeyVar:        "ZAI_API_KEY",
		},
		{
			Name:         "minimax",
			DisplayName:  "MiniMax",
			Description:  "MiniMax International (M2.5)",
			Type:         config.ProviderTypeBuiltin,
			BaseURL:      "https://api.minimax.io/anthropic",
			DefaultModel: "MiniMax-M2.5",
			KeyVar:       "MINIMAX_API_KEY",
		},
		{
			Name:          "kimi",
			DisplayName:   "Kimi",
			Description:   "Kimi K2.5",
			Type:          config.ProviderTypeBuiltin,
			BaseURL:       "https://api.kimi.com/coding/",
			DefaultModel:  "kimi-k2.5",
			ModelMappings: map[string]string{"small": "kimi-k2.5"},
			KeyVar:        "KIMI_API_KEY",
		},
		{
			Name:         "moonshot",
			DisplayName:  "Moonshot AI",
			Description:  "Moonshot AI (Kimi K2.5)",
			Type:         config.ProviderTypeBuiltin,
			BaseURL:      "https://api.moonshot.ai/anthropic",
			DefaultModel: "kimi-k2.5",
			KeyVar:       "MOONSHOT_API_KEY",
		},
		{
			Name:          "deepseek",
			DisplayName:   "DeepSeek",
			Description:   "DeepSeek Chat",
			Type:          config.ProviderTypeBuiltin,
			BaseURL:       "https://api.deepseek.com/anthropic",
			DefaultModel:  "deepseek-chat",
			ModelMappings: map[string]string{"small": "deepseek-chat"},
			KeyVar:        "DEEPSEEK_API_KEY",
		},
		{
			Name:        "ollama",
			DisplayName: "Ollama",
			Description: "Ollama local server",
			Type:        config.ProviderTypeLocal,
			BaseURL:     "http://localhost:11434",
			AuthToken:   "ollama",
		},
		{
			Name:        "lmstudio",
			DisplayName: "LM Studio",
			Description: "LM Studio local server",
			Type:        config.ProviderTypeLocal,
			BaseURL:     "http://localhost:1234",
			AuthToken:   "lmstudio",
		},
		{
			Name:        "llamacpp",
			DisplayName: "llama.cpp",
			Description: "llama.cpp local server",
			Type:        config.ProviderTypeLocal,
			BaseURL:     "http://localhost:8000",
		},
	}

	for _, def := range builtins {
		r.definitions[def.Name] = def
	}
}

// CreateProvider creates a provider instance from a definition
func (r *Registry) CreateProvider(name string, apiKey string) (Provider, error) {
	def, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	cp := &config.Provider{
		Name:          def.Name,
		Type:          def.Type,
		DisplayName:   def.DisplayName,
		Description:   def.Description,
		BaseURL:       def.BaseURL,
		DefaultModel:  def.DefaultModel,
		ModelMappings: def.ModelMappings,
		AuthToken:     def.AuthToken,
		KeyEnvVar:     def.KeyEnvVar,
	}

	provider, err := FromConfig(cp)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		provider.SetAPIKey(apiKey)
	}

	return provider, nil
}
