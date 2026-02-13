package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/providers"
	"github.com/sammcj/skint/internal/secrets"
	"golang.org/x/term"
)

// ConfigForm handles interactive provider configuration
type ConfigForm struct {
	secretsMgr *secrets.Manager
	registry   *providers.Registry
	reader     *bufio.Reader
}

// NewConfigForm creates a new configuration form
func NewConfigForm(secretsMgr *secrets.Manager) *ConfigForm {
	return &ConfigForm{
		secretsMgr: secretsMgr,
		registry:   providers.NewRegistry(),
		reader:     bufio.NewReader(os.Stdin),
	}
}

// RunProviderMenu shows the provider configuration menu
func (f *ConfigForm) RunProviderMenu(cfg *config.Config) error {
	menu := NewProviderMenu(cfg, f.registry, f)

	for {
		handler, err := menu.Display(cfg)
		if err != nil {
			if errors.Is(err, ErrTestRequested) {
				return nil // Caller handles test
			}
			Error("%v", err)
			continue
		}
		if handler == nil {
			Info("Cancelled")
			return nil
		}
		return handler()
	}
}

// ConfigureBuiltin configures a built-in provider
func (f *ConfigForm) ConfigureBuiltin(cfg *config.Config, name string) error {
	def, ok := f.registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown provider: %s", name)
	}

	fmt.Println()
	Log("%s", Bold(fmt.Sprintf("Configure: %s", def.DisplayName)))
	if def.BaseURL != "" {
		Dim("Endpoint: %s\n\n", def.BaseURL)
	}

	// Native needs no config
	if name == "native" {
		Success("Native Anthropic is ready")
		NextSteps([]string{
			"Use it: " + Green("skint use native"),
		})
		return nil
	}

	// Check if already configured
	existing := cfg.GetProvider(name)
	var apiKey string

	if existing != nil && existing.APIKeyRef != "" {
		// Try to get existing key
		if key, err := f.secretsMgr.RetrieveByReference(existing.APIKeyRef); err == nil {
			fmt.Printf("Current key: %s\n", MaskKey(key))

			if Confirm("Change key?", false) {
				apiKey = f.promptSecret("API Key")
			} else {
				return nil
			}
		}
	} else {
		apiKey = f.promptSecret("API Key")
	}

	if apiKey == "" {
		Warning("No API key provided")
		return nil
	}

	if len(apiKey) < 8 {
		Error("API key too short (minimum 8 characters)")
		return nil
	}

	// Store the API key
	ref, err := f.secretsMgr.StoreWithReference(name, apiKey)
	if err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}

	// Create or update provider config
	provider := &config.Provider{
		Name:          def.Name,
		Type:          def.Type,
		DisplayName:   def.DisplayName,
		Description:   def.Description,
		BaseURL:       def.BaseURL,
		DefaultModel:  def.DefaultModel,
		ModelMappings: def.ModelMappings,
		APIKeyRef:     ref,
	}

	if existing != nil {
		cfg.RemoveProvider(name)
	}
	if err := cfg.AddProvider(provider); err != nil {
		return err
	}

	Success("API key saved for %s", def.DisplayName)
	NextSteps([]string{
		"Use it: " + Green("skint use "+name),
		"Test it: " + Green("skintest "+name),
	})

	return nil
}

func (f *ConfigForm) configureLocal(cfg *config.Config, name string) error {
	def, ok := f.registry.Get(name)
	if !ok {
		return fmt.Errorf("unknown provider: %s", name)
	}

	fmt.Println()
	Log("%s", Bold(fmt.Sprintf("Configure: %s", def.DisplayName)))
	Dim("Endpoint: %s\n\n", def.BaseURL)

	switch name {
	case "ollama":
		Log("Ollama serves local models with Anthropic-compatible API.")
		fmt.Println()
		Log("%s:", Bold("Setup"))
		fmt.Println("  1. Install Ollama: https://ollama.com")
		fmt.Println("  2. Pull a model: ollama pull qwen3-coder")
		fmt.Println("  3. Start serving: ollama serve")
		fmt.Println()
		Log("%s:", Bold("Recommended models"))
		Dim("  → qwen3-coder\n")
		Dim("  → glm-5\n")
		Dim("  → gpt-oss:20b\n")
		Dim("  → gpt-oss:120b\n")
	case "lmstudio":
		Log("LM Studio runs local models with Anthropic-compatible API.")
		fmt.Println()
		Log("%s:", Bold("Setup"))
		fmt.Println("  1. Install LM Studio: https://lmstudio.ai/download")
		fmt.Println("  2. Load a model in the app")
		fmt.Println("  3. Start the server (port 1234)")
		fmt.Println()
		Log("%s:", Bold("Usage"))
		Dim("  skint use lmstudio --model <model-name>\n")
	case "llamacpp":
		Log("llama.cpp's llama-server with Anthropic-compatible API.")
		fmt.Println()
		Log("%s:", Bold("Setup"))
		fmt.Println("  1. Build llama.cpp: https://github.com/ggml-org/llama.cpp")
		fmt.Println("  2. Start server:")
		Dim("     ./llama-server --model <model.gguf> --port 8000 --jinja\n")
		fmt.Println()
		Log("%s:", Bold("Usage"))
		Dim("  skint use llamacpp --model <model-name>\n")
	}

	fmt.Println()

	// Add provider if not exists
	if cfg.GetProvider(name) == nil {
		provider := &config.Provider{
			Name:        def.Name,
			Type:        def.Type,
			DisplayName: def.DisplayName,
			Description: def.Description,
			BaseURL:     def.BaseURL,
			AuthToken:   def.AuthToken,
		}
		if err := cfg.AddProvider(provider); err != nil {
			return err
		}
	}

	Success("Ready to use: %s", Green("skint use "+name))

	return nil
}

// ConfigureOpenRouter configures OpenRouter
func (f *ConfigForm) ConfigureOpenRouter(cfg *config.Config) error {
	fmt.Println()
	Log("%s", Bold("Configure: OpenRouter"))
	Dim("Access 100+ models via native Anthropic API\n")
	Cyan("Get API key: https://openrouter.ai/keys\n")
	fmt.Println()

	// Get or prompt for API key
	var apiKey string
	existingKey, _ := f.secretsMgr.Retrieve("openrouter")

	if existingKey != "" {
		fmt.Printf("Current key: %s\n", MaskKey(existingKey))

		if Confirm("Change API key?", false) {
			apiKey = f.promptSecret("New API Key")
		}
	} else {
		apiKey = f.promptSecret("API Key")
	}

	if apiKey != "" {
		if err := f.secretsMgr.Store("openrouter", apiKey); err != nil {
			return fmt.Errorf("failed to store API key: %w", err)
		}
	}

	// List existing OpenRouter models
	fmt.Println()
	Log("%s", Bold("Configured OpenRouter models"))
	hasModels := false
	for _, p := range cfg.Providers {
		if p.Type == config.ProviderTypeOpenRouter {
			fmt.Printf("  %s\n", Green(p.Name))
			hasModels = true
		}
	}
	if !hasModels {
		Dim("  (none)\n")
	}

	// Ask if user wants to add a model
	fmt.Println()
	if Confirm("Add a model?", true) {
		return f.addOpenRouterModel(cfg)
	}

	return nil
}

func (f *ConfigForm) addOpenRouterModel(cfg *config.Config) error {
	fmt.Print("\nModel ID (e.g. openai/gpt-4o) or 'q' to quit: ")
	modelID, err := f.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	modelID = strings.TrimSpace(modelID)

	if modelID == "" || modelID == "q" {
		return nil
	}

	// Generate default short name
	defaultName := modelID
	if idx := strings.LastIndex(modelID, "/"); idx >= 0 {
		defaultName = modelID[idx+1:]
	}
	defaultName = sanitizeName(defaultName)

	fmt.Printf("Short name [%s]: ", defaultName)
	shortName, err := f.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	shortName = strings.TrimSpace(shortName)
	if shortName == "" {
		shortName = defaultName
	}

	// Validate name
	if !isValidName(shortName) {
		Error("Invalid name. Use lowercase letters, digits, hyphens, and underscores only.")
		return nil
	}

	// Create full name
	fullName := "or-" + shortName

	// Remove existing if present
	cfg.RemoveProvider(fullName)

	// Store API key reference
	ref := secrets.StorageTypeKeyring + ":openrouter"
	if !f.secretsMgr.IsKeyringAvailable() {
		ref = secrets.StorageTypeFile + ":openrouter"
	}

	// Add provider
	provider := &config.Provider{
		Name:        fullName,
		Type:        config.ProviderTypeOpenRouter,
		DisplayName: fmt.Sprintf("OpenRouter %s", shortName),
		BaseURL:     "https://openrouter.ai/api",
		Model:       modelID,
		APIKeyRef:   ref,
	}

	if err := cfg.AddProvider(provider); err != nil {
		return err
	}

	Success("Added OpenRouter model: %s", fullName)

	return nil
}

// ConfigureCustom configures a custom provider
func (f *ConfigForm) ConfigureCustom(cfg *config.Config) error {
	fmt.Println()
	Log("%s", Bold("Configure: Custom Provider"))
	Dim("For any Anthropic-compatible endpoint\n")
	fmt.Println()

	fmt.Print("Provider name (lowercase): ")
	name, err := f.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	name = strings.TrimSpace(name)

	if !isValidName(name) {
		Error("Invalid name. Use lowercase letters, digits, hyphens, and underscores only.")
		return nil
	}

	fmt.Print("Base URL: ")
	baseURL, err := f.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	baseURL = strings.TrimSpace(baseURL)

	if baseURL == "" || (!strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://")) {
		Error("Invalid URL. Must start with http:// or https://")
		return nil
	}

	apiKey := f.promptSecret("API Key")

	if name == "" || baseURL == "" {
		Error("Name and base URL are required")
		return nil
	}

	// Store API key
	ref, err := f.secretsMgr.StoreWithReference(name, apiKey)
	if err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}

	// Remove existing if present
	cfg.RemoveProvider(name)

	// Add provider
	provider := &config.Provider{
		Name:        name,
		Type:        config.ProviderTypeCustom,
		DisplayName: name,
		BaseURL:     baseURL,
		APIKeyRef:   ref,
	}

	if err := cfg.AddProvider(provider); err != nil {
		return err
	}

	Success("Created custom provider: %s", name)

	return nil
}

// promptSecret prompts for a secret (password) input
func (f *ConfigForm) promptSecret(prompt string) string {
	fmt.Printf("%s: ", prompt)

	// Try to use terminal for hidden input
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		// Do not fall back to echoing input -- that would display the secret
		fmt.Fprintln(os.Stderr, "\nWarning: unable to read secret input (no terminal available)")
		return ""
	}

	fmt.Println()
	return strings.TrimSpace(string(bytePassword))
}

// Helper functions

func sanitizeName(name string) string {
	// Remove non-alphanumeric characters except hyphens
	result := make([]rune, 0, len(name))
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}
	return string(result)
}

func isValidName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}
