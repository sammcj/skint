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
	overrides  envOverrides
}

// envOverrides records persisted config values that were replaced by SKINT_*
// environment overrides at Load time. Save reverts to these so transient env
// settings are never written to disk. A nil pointer means the field was not
// overridden.
type envOverrides struct {
	defaultProvider *fieldOverride[string]
	outputFormat    *fieldOverride[string]
	colorEnabled    *fieldOverride[bool]
	noBanner        *fieldOverride[bool]
}

// fieldOverride pairs the persisted value with the env value that replaced it.
type fieldOverride[T comparable] struct {
	persisted T
	applied   T
}

// revert returns the value Save should persist: the pre-override value while the
// runtime value still equals the applied override, otherwise the runtime value -
// a deliberate change (e.g. the TUI setting a new default provider) must win.
func (o *fieldOverride[T]) revert(current T) T {
	if o != nil && current == o.applied {
		return o.persisted
	}
	return current
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

	// Clear any legacy plaintext API keys (migration artifact)
	for _, p := range m.config.Providers {
		if p.APIKey != "" && p.APIKeyRef != "" {
			p.APIKey = ""
		}
	}

	// Apply environment overrides
	m.applyEnvOverrides()

	// A SKINT_DEFAULT_PROVIDER naming an unknown provider is non-fatal.
	m.resolveDefaultProviderOverride()

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

	// Check for symlink before writing (security)
	if info, err := os.Lstat(m.configFile); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("config file is a symlink - refusing to write for security")
		}
	}

	// Revert env overrides so transient settings are not persisted.
	toSave := m.configForSave()

	// Marshal to YAML
	data, err := yaml.Marshal(&toSave)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return m.writeAtomic(data)
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

// applyEnvOverrides applies SKINT_* environment variable overrides, recording
// the pre-override values so Save can revert them (see envOverrides).
func (m *Manager) applyEnvOverrides() {
	m.overrides = envOverrides{}

	if v := os.Getenv("SKINT_DEFAULT_PROVIDER"); v != "" {
		m.overrides.defaultProvider = &fieldOverride[string]{persisted: m.config.DefaultProvider, applied: v}
		m.config.DefaultProvider = v
	}
	if v := os.Getenv("SKINT_OUTPUT_FORMAT"); v != "" {
		switch v {
		case FormatHuman, FormatJSON, FormatPlain:
			m.overrides.outputFormat = &fieldOverride[string]{persisted: m.config.OutputFormat, applied: v}
			m.config.OutputFormat = v
		default:
			fmt.Fprintf(os.Stderr, "warning: ignoring invalid SKINT_OUTPUT_FORMAT=%q (valid: %s, %s, %s)\n",
				v, FormatHuman, FormatJSON, FormatPlain)
		}
	}
	if os.Getenv("SKINT_NO_COLOR") != "" || os.Getenv("NO_COLOR") != "" {
		m.overrides.colorEnabled = &fieldOverride[bool]{persisted: m.config.ColorEnabled, applied: false}
		m.config.ColorEnabled = false
	}
	if os.Getenv("SKINT_NO_BANNER") != "" {
		m.overrides.noBanner = &fieldOverride[bool]{persisted: m.config.NoBanner, applied: true}
		m.config.NoBanner = true
	}
}

// resolveDefaultProviderOverride handles a SKINT_DEFAULT_PROVIDER that names an
// unknown provider: rather than failing validation, warn and fall back to the
// persisted default.
func (m *Manager) resolveDefaultProviderOverride() {
	if m.overrides.defaultProvider == nil {
		return
	}
	name := m.config.DefaultProvider
	if name == "native" || m.config.GetProvider(name) != nil {
		return
	}
	fmt.Fprintf(os.Stderr, "warning: SKINT_DEFAULT_PROVIDER=%q not found in config; using %q\n",
		name, m.overrides.defaultProvider.persisted)
	m.config.DefaultProvider = m.overrides.defaultProvider.persisted
	m.overrides.defaultProvider = nil
}

// configForSave returns a copy of the config with env overrides reverted to
// their persisted values, so transient env settings are not written to disk.
// Fields deliberately changed at runtime since the override was applied are
// kept (see fieldOverride.revert).
func (m *Manager) configForSave() Config {
	c := *m.config
	c.DefaultProvider = m.overrides.defaultProvider.revert(c.DefaultProvider)
	c.OutputFormat = m.overrides.outputFormat.revert(c.OutputFormat)
	c.ColorEnabled = m.overrides.colorEnabled.revert(c.ColorEnabled)
	c.NoBanner = m.overrides.noBanner.revert(c.NoBanner)
	return c
}

// writeAtomic writes data to the config file atomically: it writes to a temp
// file in the same directory, syncs, then renames over the target. A crash
// mid-write leaves the existing config intact.
func (m *Manager) writeAtomic(data []byte) error {
	tmp, err := os.CreateTemp(m.configDir, ".config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op after a successful rename

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to set temp config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write config file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to sync config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close config file: %w", err)
	}
	if err := os.Rename(tmpPath, m.configFile); err != nil {
		return fmt.Errorf("failed to replace config file: %w", err)
	}
	return nil
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
