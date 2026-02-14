package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Migration handles importing configuration from the bash version
type Migration struct {
	dataDir string
}

// NewMigration creates a new migration helper
func NewMigration() (*Migration, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, err
	}
	return &Migration{dataDir: dataDir}, nil
}

// HasOldInstallation returns true if the bash version is installed
func (m *Migration) HasOldInstallation() bool {
	secretsFile := filepath.Join(m.dataDir, "secrets.env")
	_, err := os.Stat(secretsFile)
	return err == nil
}

// SecretsFile returns the path to the old secrets file
func (m *Migration) SecretsFile() string {
	return filepath.Join(m.dataDir, "secrets.env")
}

// OldEntry represents a provider configuration from the bash version
type OldEntry struct {
	Name        string
	DisplayName string
	KeyVar      string
	BaseURL     string
	Model       string
	ModelOpts   map[string]string
	APIKey      string
	IsLocal     bool
}

// LoadSecrets loads the old secrets.env file
func (m *Migration) LoadSecrets() (map[string]string, error) {
	secretsFile := m.SecretsFile()

	// Check for symlink
	info, err := os.Lstat(secretsFile)
	if err != nil {
		return nil, fmt.Errorf("secrets file not found: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("secrets file is a symlink - refusing for security")
	}

	// Check permissions
	// Note: We can't easily check permissions across platforms, so we just read

	file, err := os.Open(secretsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open secrets file: %w", err)
	}
	defer file.Close()

	secrets := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=value format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		// Handle escaped characters
		value = m.unescape(value)

		secrets[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read secrets file: %w", err)
	}

	return secrets, nil
}

// unescape handles bash escape sequences
func (m *Migration) unescape(s string) string {
	// Simple unescaping for common cases
	replacements := map[string]string{
		`\\`: `\`,
		`\"`: `"`,
		`\'`: `'`,
		`\n`: "\n",
		`\t`: "\t",
		`\$`: `$`,
	}

	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}

	return s
}

// ProviderDefinitions maps old provider names to their definitions
var ProviderDefinitions = map[string]OldEntry{
	"native": {
		Name:        "native",
		DisplayName: "Claude Subscription",
		KeyVar:      "",
		BaseURL:     "",
		Model:       "",
	},
	"zai": {
		Name:        "zai",
		DisplayName: "Z.AI",
		KeyVar:      "ZAI_API_KEY",
		BaseURL:     "https://api.z.ai/api/anthropic",
		Model:       "glm-5",
		ModelOpts:   map[string]string{"haiku": "glm-5", "sonnet": "glm-5", "opus": "glm-5"},
	},
	"minimax": {
		Name:        "minimax",
		DisplayName: "MiniMax",
		KeyVar:      "MINIMAX_API_KEY",
		BaseURL:     "https://api.minimax.io/anthropic",
		Model:       "MiniMax-M2.5",
	},
	"kimi": {
		Name:        "kimi",
		DisplayName: "Kimi",
		KeyVar:      "KIMI_API_KEY",
		BaseURL:     "https://api.kimi.com/coding/",
		Model:       "kimi-k2.5",
		ModelOpts:   map[string]string{"small": "kimi-k2.5"},
	},
	"moonshot": {
		Name:        "moonshot",
		DisplayName: "Moonshot AI",
		KeyVar:      "MOONSHOT_API_KEY",
		BaseURL:     "https://api.moonshot.ai/anthropic",
		Model:       "kimi-k2.5",
	},
	"deepseek": {
		Name:        "deepseek",
		DisplayName: "DeepSeek",
		KeyVar:      "DEEPSEEK_API_KEY",
		BaseURL:     "https://api.deepseek.com/anthropic",
		Model:       "deepseek-chat",
		ModelOpts:   map[string]string{"small": "deepseek-chat"},
	},
	"ollama": {
		Name:        "ollama",
		DisplayName: "Ollama",
		KeyVar:      "@ollama",
		BaseURL:     "http://localhost:11434",
		IsLocal:     true,
	},
	"lmstudio": {
		Name:        "lmstudio",
		DisplayName: "LM Studio",
		KeyVar:      "@lmstudio",
		BaseURL:     "http://localhost:1234",
		IsLocal:     true,
	},
	"llamacpp": {
		Name:        "llamacpp",
		DisplayName: "llama.cpp",
		KeyVar:      "@",
		BaseURL:     "http://localhost:8000",
		IsLocal:     true,
	},
}

// Import imports providers from the old secrets.env
func (m *Migration) Import() (*Config, map[string]string, error) {
	secrets, err := m.LoadSecrets()
	if err != nil {
		return nil, nil, err
	}

	cfg := NewDefaultConfig()
	keysToStore := make(map[string]string) // provider name -> API key

	// Import built-in providers
	for name, def := range ProviderDefinitions {
		provider := &Provider{
			Name:          name,
			Type:          ProviderTypeBuiltin,
			DisplayName:   def.DisplayName,
			BaseURL:       def.BaseURL,
			DefaultModel:  def.Model,
			ModelMappings: def.ModelOpts,
		}

		if def.IsLocal {
			provider.Type = ProviderTypeLocal
			if def.KeyVar != "" && def.KeyVar != "@" {
				provider.AuthToken = def.KeyVar[1:] // Remove @ prefix
			}
		} else if def.KeyVar != "" {
			// Check if API key exists in secrets
			if key, ok := secrets[def.KeyVar]; ok && key != "" {
				keysToStore[name] = key
			}
		}

		// Only add if API key exists or it's a local/native provider
		if def.Name == "native" || def.IsLocal || keysToStore[name] != "" {
			cfg.Providers = append(cfg.Providers, provider)
		}
	}

	// Import OpenRouter models
	orPattern := regexp.MustCompile(`^OPENROUTER_MODEL_([A-Z_]+)$`)
	for key, value := range secrets {
		if key == "OPENROUTER_API_KEY" {
			// This is the main OpenRouter key - store it
			keysToStore["openrouter"] = value
			continue
		}

		matches := orPattern.FindStringSubmatch(key)
		if matches != nil {
			name := "or-" + strings.ToLower(strings.ReplaceAll(matches[1], "_", "-"))
			provider := &Provider{
				Name:        name,
				Type:        ProviderTypeOpenRouter,
				DisplayName: fmt.Sprintf("OpenRouter %s", matches[1]),
				BaseURL:     "https://openrouter.ai/api",
				Model:       value,
			}
			// Use the main OpenRouter API key
			if orKey, ok := secrets["OPENROUTER_API_KEY"]; ok {
				keysToStore[name] = orKey
			}
			cfg.Providers = append(cfg.Providers, provider)
		}
	}

	// Import custom providers (look for patterns like *_API_KEY with corresponding BASE_URL)
	customPattern := regexp.MustCompile(`^([A-Z_]+)_API_KEY$`)
	for key := range secrets {
		matches := customPattern.FindStringSubmatch(key)
		if matches == nil {
			continue
		}

		prefix := matches[1]
		baseURLKey := fmt.Sprintf("SKINT_%s_API_KEY_BASE_URL", prefix)

		// Skip known providers
		known := false
		for _, def := range ProviderDefinitions {
			if def.KeyVar == key {
				known = true
				break
			}
		}
		if known || key == "OPENROUTER_API_KEY" {
			continue
		}

		// Check for base URL
		if baseURL, ok := secrets[baseURLKey]; ok {
			name := strings.ToLower(strings.ReplaceAll(prefix, "_", "-"))
			provider := &Provider{
				Name:        name,
				Type:        ProviderTypeCustom,
				DisplayName: name,
				BaseURL:     baseURL,
			}
			if apiKey, ok := secrets[key]; ok {
				keysToStore[name] = apiKey
			}
			cfg.Providers = append(cfg.Providers, provider)
		}
	}

	return cfg, keysToStore, nil
}

// Cleanup removes the old installation files
func (m *Migration) Cleanup() error {
	files := []string{
		m.SecretsFile(),
		filepath.Join(m.dataDir, "banner"),
		filepath.Join(m.dataDir, "skint-full.sh"),
	}

	for _, f := range files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", f, err)
		}
	}

	// Try to remove the data directory if empty
	os.Remove(m.dataDir)

	return nil
}
