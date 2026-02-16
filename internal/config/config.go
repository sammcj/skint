package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Manager handles configuration loading and saving
type Manager struct {
	configDir  string
	configFile string
	config     *Config
}

// NewManager creates a new configuration manager
func NewManager() (*Manager, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	m := &Manager{
		configDir:  configDir,
		configFile: filepath.Join(configDir, "config.yaml"),
		config:     NewDefaultConfig(),
	}

	return m, nil
}

// NewManagerWithPath creates a manager with custom config path
func NewManagerWithPath(configPath string) (*Manager, error) {
	configDir := filepath.Dir(configPath)

	m := &Manager{
		configDir:  configDir,
		configFile: configPath,
		config:     NewDefaultConfig(),
	}

	return m, nil
}

// Load reads the configuration from disk
func (m *Manager) Load() error {
	// Ensure config directory exists
	if err := os.MkdirAll(m.configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(m.configFile); os.IsNotExist(err) {
		// No config file yet, use defaults
		return nil
	}

	// Check for symlink before reading (security)
	info, err := os.Lstat(m.configFile)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("config file is a symlink - refusing for security")
	}

	// Read file
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment overrides
	m.applyEnvOverrides()

	// Validate
	if err := m.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// Save writes the configuration to disk
func (m *Manager) Save() error {
	// Validate before saving
	if err := m.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(m.configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with secure permissions
	f, err := os.OpenFile(m.configFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close config file: %w", err)
	}

	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	return m.config
}

// Set updates the configuration
func (m *Manager) Set(cfg *Config) {
	m.config = cfg
}

// ConfigFile returns the path to the config file
func (m *Manager) ConfigFile() string {
	return m.configFile
}

// ConfigDir returns the configuration directory
func (m *Manager) ConfigDir() string {
	return m.configDir
}

// Exists returns true if the config file exists
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.configFile)
	return err == nil
}

// applyEnvOverrides applies SKINT_* environment variable overrides
func (m *Manager) applyEnvOverrides() {
	if v := os.Getenv("SKINT_DEFAULT_PROVIDER"); v != "" {
		m.config.DefaultProvider = v
	}
	if v := os.Getenv("SKINT_OUTPUT_FORMAT"); v != "" {
		switch v {
		case FormatHuman, FormatJSON, FormatPlain:
			m.config.OutputFormat = v
		default:
			fmt.Fprintf(os.Stderr, "warning: ignoring invalid SKINT_OUTPUT_FORMAT=%q (valid: %s, %s, %s)\n",
				v, FormatHuman, FormatJSON, FormatPlain)
		}
	}
	if os.Getenv("SKINT_NO_COLOR") != "" || os.Getenv("NO_COLOR") != "" {
		m.config.ColorEnabled = false
	}
	if os.Getenv("SKINT_NO_BANNER") != "" {
		m.config.NoBanner = true
	}
}

// getConfigDir returns the XDG-compliant config directory
func getConfigDir() (string, error) {
	// Check XDG_CONFIG_HOME
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "skint"), nil
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".config", "skint"), nil
}

// GetDataDir returns the XDG-compliant data directory
func GetDataDir() (string, error) {
	// Check XDG_DATA_HOME
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "skint"), nil
	}

	// Fall back to ~/.local/share
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".local", "share", "skint"), nil
}

// GetCacheDir returns the XDG-compliant cache directory
func GetCacheDir() (string, error) {
	// Check XDG_CACHE_HOME
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "skint"), nil
	}

	// Fall back to ~/.cache
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".cache", "skint"), nil
}

// GetBinDir returns the appropriate bin directory
func GetBinDir() (string, error) {
	// Check SKINT_BIN
	if bin := os.Getenv("SKINT_BIN"); bin != "" {
		return bin, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// macOS: ~/bin, Linux: ~/.local/bin
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "bin"), nil
	}

	return filepath.Join(home, ".local", "bin"), nil
}
