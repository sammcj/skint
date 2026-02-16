package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ---------------------------------------------------------------------------
// Manager.Load()
// ---------------------------------------------------------------------------

func TestManagerLoad(t *testing.T) {
	t.Run("no file returns defaults", func(t *testing.T) {
		dir := t.TempDir()
		m, err := NewManagerWithPath(filepath.Join(dir, "config.yaml"))
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if err := m.Load(); err != nil {
			t.Fatalf("Load: %v", err)
		}
		cfg := m.Get()
		if cfg.Version != ConfigVersion {
			t.Errorf("Version: got %q, want %q", cfg.Version, ConfigVersion)
		}
		if cfg.OutputFormat != FormatHuman {
			t.Errorf("OutputFormat: got %q, want %q", cfg.OutputFormat, FormatHuman)
		}
		if !cfg.ColorEnabled {
			t.Error("ColorEnabled: expected true")
		}
	})

	t.Run("valid YAML is loaded", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		yamlContent := `version: "1.0"
default_provider: "native"
output_format: "json"
color_enabled: false
no_banner: true
providers:
  - name: native
    type: builtin
`
		if err := os.WriteFile(cfgPath, []byte(yamlContent), 0600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if err := m.Load(); err != nil {
			t.Fatalf("Load: %v", err)
		}
		cfg := m.Get()
		if cfg.DefaultProvider != "native" {
			t.Errorf("DefaultProvider: got %q, want %q", cfg.DefaultProvider, "native")
		}
		if cfg.OutputFormat != FormatJSON {
			t.Errorf("OutputFormat: got %q, want %q", cfg.OutputFormat, FormatJSON)
		}
		if cfg.ColorEnabled {
			t.Error("ColorEnabled: expected false")
		}
		if !cfg.NoBanner {
			t.Error("NoBanner: expected true")
		}
		if len(cfg.Providers) != 1 {
			t.Fatalf("Providers count: got %d, want 1", len(cfg.Providers))
		}
		if cfg.Providers[0].Name != "native" {
			t.Errorf("Providers[0].Name: got %q, want %q", cfg.Providers[0].Name, "native")
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(cfgPath, []byte("{{not yaml"), 0600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if err := m.Load(); err == nil {
			t.Fatal("expected error for invalid YAML, got nil")
		}
	})

	t.Run("symlink config file is rejected", func(t *testing.T) {
		dir := t.TempDir()
		realFile := filepath.Join(dir, "real.yaml")
		if err := os.WriteFile(realFile, []byte("version: \"1.0\"\n"), 0600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		symlinkPath := filepath.Join(dir, "config.yaml")
		if err := os.Symlink(realFile, symlinkPath); err != nil {
			t.Fatalf("Symlink: %v", err)
		}
		m, err := NewManagerWithPath(symlinkPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		err = m.Load()
		if err == nil {
			t.Fatal("expected error for symlink config, got nil")
		}
		if got := err.Error(); got != "config file is a symlink - refusing for security" {
			t.Errorf("error message: got %q", got)
		}
	})

	t.Run("creates config directory if missing", func(t *testing.T) {
		dir := t.TempDir()
		nested := filepath.Join(dir, "sub", "dir")
		cfgPath := filepath.Join(nested, "config.yaml")
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if err := m.Load(); err != nil {
			t.Fatalf("Load: %v", err)
		}
		info, err := os.Stat(nested)
		if err != nil {
			t.Fatalf("config dir was not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected config dir to be a directory")
		}
	})

	t.Run("validation error for invalid config content", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		yamlContent := `version: "1.0"
output_format: "invalid_format"
`
		if err := os.WriteFile(cfgPath, []byte(yamlContent), 0600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if err := m.Load(); err == nil {
			t.Fatal("expected validation error, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// Manager.Save() and round-trip
// ---------------------------------------------------------------------------

func TestManagerSaveAndRoundTrip(t *testing.T) {
	t.Run("save then load preserves config", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")

		// Save
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		cfg := m.Get()
		cfg.DefaultProvider = "my-local"
		cfg.OutputFormat = FormatPlain
		cfg.NoBanner = true
		cfg.ColorEnabled = false
		cfg.ClaudeArgs = []string{"--continue", "--verbose"}
		cfg.Providers = []*Provider{
			{Name: "my-local", Type: ProviderTypeLocal, BaseURL: "http://localhost:8080"},
		}
		m.Set(cfg)
		if err := m.Save(); err != nil {
			t.Fatalf("Save: %v", err)
		}

		// Load into a fresh manager
		m2, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath (reload): %v", err)
		}
		if err := m2.Load(); err != nil {
			t.Fatalf("Load: %v", err)
		}
		loaded := m2.Get()

		if loaded.DefaultProvider != "my-local" {
			t.Errorf("DefaultProvider: got %q, want %q", loaded.DefaultProvider, "my-local")
		}
		if loaded.OutputFormat != FormatPlain {
			t.Errorf("OutputFormat: got %q, want %q", loaded.OutputFormat, FormatPlain)
		}
		if loaded.ColorEnabled {
			t.Error("ColorEnabled: expected false")
		}
		if !loaded.NoBanner {
			t.Error("NoBanner: expected true")
		}
		if len(loaded.ClaudeArgs) != 2 || loaded.ClaudeArgs[0] != "--continue" || loaded.ClaudeArgs[1] != "--verbose" {
			t.Errorf("ClaudeArgs: got %v, want [--continue --verbose]", loaded.ClaudeArgs)
		}
		if len(loaded.Providers) != 1 || loaded.Providers[0].Name != "my-local" {
			t.Errorf("Providers: got %v", loaded.Providers)
		}
	})

	t.Run("save creates file with restricted permissions", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if err := m.Save(); err != nil {
			t.Fatalf("Save: %v", err)
		}
		info, err := os.Stat(cfgPath)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("file permissions: got %o, want 0600", perm)
		}
	})

	t.Run("save rejects invalid config", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		cfg := m.Get()
		cfg.OutputFormat = "bogus"
		m.Set(cfg)
		if err := m.Save(); err == nil {
			t.Fatal("expected validation error on Save, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// Manager accessors: Exists, ConfigFile, ConfigDir
// ---------------------------------------------------------------------------

func TestManagerAccessors(t *testing.T) {
	t.Run("Exists returns false for non-existent file", func(t *testing.T) {
		dir := t.TempDir()
		m, err := NewManagerWithPath(filepath.Join(dir, "no-such-config.yaml"))
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if m.Exists() {
			t.Error("Exists: expected false for missing file")
		}
	})

	t.Run("Exists returns true after Save", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if err := m.Save(); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if !m.Exists() {
			t.Error("Exists: expected true after Save")
		}
	})

	t.Run("ConfigFile and ConfigDir return expected paths", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if m.ConfigFile() != cfgPath {
			t.Errorf("ConfigFile: got %q, want %q", m.ConfigFile(), cfgPath)
		}
		if m.ConfigDir() != dir {
			t.Errorf("ConfigDir: got %q, want %q", m.ConfigDir(), dir)
		}
	})
}

// ---------------------------------------------------------------------------
// applyEnvOverrides
// ---------------------------------------------------------------------------

func TestApplyEnvOverrides(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name: "SKINT_DEFAULT_PROVIDER overrides default provider",
			envVars: map[string]string{
				"SKINT_DEFAULT_PROVIDER": "some-provider",
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.DefaultProvider != "some-provider" {
					t.Errorf("DefaultProvider: got %q, want %q", cfg.DefaultProvider, "some-provider")
				}
			},
		},
		{
			name: "SKINT_OUTPUT_FORMAT valid value is applied",
			envVars: map[string]string{
				"SKINT_OUTPUT_FORMAT": FormatJSON,
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.OutputFormat != FormatJSON {
					t.Errorf("OutputFormat: got %q, want %q", cfg.OutputFormat, FormatJSON)
				}
			},
		},
		{
			name: "SKINT_OUTPUT_FORMAT plain value is applied",
			envVars: map[string]string{
				"SKINT_OUTPUT_FORMAT": FormatPlain,
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.OutputFormat != FormatPlain {
					t.Errorf("OutputFormat: got %q, want %q", cfg.OutputFormat, FormatPlain)
				}
			},
		},
		{
			name: "SKINT_OUTPUT_FORMAT invalid value is ignored",
			envVars: map[string]string{
				"SKINT_OUTPUT_FORMAT": "xml",
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				// Should remain at the default (human)
				if cfg.OutputFormat != FormatHuman {
					t.Errorf("OutputFormat: got %q, want %q (invalid value should be ignored)", cfg.OutputFormat, FormatHuman)
				}
			},
		},
		{
			name: "SKINT_NO_COLOR disables colour",
			envVars: map[string]string{
				"SKINT_NO_COLOR": "1",
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.ColorEnabled {
					t.Error("ColorEnabled: expected false when SKINT_NO_COLOR is set")
				}
			},
		},
		{
			name: "NO_COLOR disables colour",
			envVars: map[string]string{
				"NO_COLOR": "1",
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.ColorEnabled {
					t.Error("ColorEnabled: expected false when NO_COLOR is set")
				}
			},
		},
		{
			name: "SKINT_NO_BANNER enables no-banner",
			envVars: map[string]string{
				"SKINT_NO_BANNER": "1",
			},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.NoBanner {
					t.Error("NoBanner: expected true when SKINT_NO_BANNER is set")
				}
			},
		},
		{
			name:    "no env vars leaves defaults untouched",
			envVars: map[string]string{},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.OutputFormat != FormatHuman {
					t.Errorf("OutputFormat: got %q, want %q", cfg.OutputFormat, FormatHuman)
				}
				if !cfg.ColorEnabled {
					t.Error("ColorEnabled: expected true (default)")
				}
				if cfg.NoBanner {
					t.Error("NoBanner: expected false (default)")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			dir := t.TempDir()
			m, err := NewManagerWithPath(filepath.Join(dir, "config.yaml"))
			if err != nil {
				t.Fatalf("NewManagerWithPath: %v", err)
			}
			m.applyEnvOverrides()
			tc.check(t, m.Get())
		})
	}
}

func TestApplyEnvOverridesViaLoad(t *testing.T) {
	t.Run("env overrides are applied during Load", func(t *testing.T) {
		t.Setenv("SKINT_DEFAULT_PROVIDER", "env-provider")
		t.Setenv("SKINT_OUTPUT_FORMAT", FormatPlain)
		t.Setenv("SKINT_NO_BANNER", "yes")

		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		yamlContent := `version: "1.0"
default_provider: "file-provider"
output_format: "json"
no_banner: false
providers:
  - name: env-provider
    type: builtin
    base_url: "https://example.com"
  - name: file-provider
    type: builtin
    base_url: "https://example.com"
`
		if err := os.WriteFile(cfgPath, []byte(yamlContent), 0600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		m, err := NewManagerWithPath(cfgPath)
		if err != nil {
			t.Fatalf("NewManagerWithPath: %v", err)
		}
		if err := m.Load(); err != nil {
			t.Fatalf("Load: %v", err)
		}
		cfg := m.Get()
		if cfg.DefaultProvider != "env-provider" {
			t.Errorf("DefaultProvider: got %q, want %q", cfg.DefaultProvider, "env-provider")
		}
		if cfg.OutputFormat != FormatPlain {
			t.Errorf("OutputFormat: got %q, want %q", cfg.OutputFormat, FormatPlain)
		}
		if !cfg.NoBanner {
			t.Error("NoBanner: expected true from env override")
		}
	})
}

// ---------------------------------------------------------------------------
// XDG directory functions
// ---------------------------------------------------------------------------

func TestGetConfigDir(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)
		got, err := getConfigDir()
		if err != nil {
			t.Fatalf("getConfigDir: %v", err)
		}
		want := filepath.Join(xdg, "skint")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to ~/.config/skint when XDG_CONFIG_HOME is unset", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		got, err := getConfigDir()
		if err != nil {
			t.Fatalf("getConfigDir: %v", err)
		}
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".config", "skint")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestGetDataDir(t *testing.T) {
	t.Run("uses XDG_DATA_HOME when set", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_DATA_HOME", xdg)
		got, err := GetDataDir()
		if err != nil {
			t.Fatalf("GetDataDir: %v", err)
		}
		want := filepath.Join(xdg, "skint")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to ~/.local/share/skint when XDG_DATA_HOME is unset", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		got, err := GetDataDir()
		if err != nil {
			t.Fatalf("GetDataDir: %v", err)
		}
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".local", "share", "skint")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestGetCacheDir(t *testing.T) {
	t.Run("uses XDG_CACHE_HOME when set", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", xdg)
		got, err := GetCacheDir()
		if err != nil {
			t.Fatalf("GetCacheDir: %v", err)
		}
		want := filepath.Join(xdg, "skint")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to ~/.cache/skint when XDG_CACHE_HOME is unset", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		got, err := GetCacheDir()
		if err != nil {
			t.Fatalf("GetCacheDir: %v", err)
		}
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".cache", "skint")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestGetBinDir(t *testing.T) {
	t.Run("uses SKINT_BIN when set", func(t *testing.T) {
		customBin := t.TempDir()
		t.Setenv("SKINT_BIN", customBin)
		got, err := GetBinDir()
		if err != nil {
			t.Fatalf("GetBinDir: %v", err)
		}
		if got != customBin {
			t.Errorf("got %q, want %q", got, customBin)
		}
	})

	t.Run("falls back to platform-specific path when SKINT_BIN is unset", func(t *testing.T) {
		t.Setenv("SKINT_BIN", "")
		got, err := GetBinDir()
		if err != nil {
			t.Fatalf("GetBinDir: %v", err)
		}
		home, _ := os.UserHomeDir()
		var want string
		if runtime.GOOS == "darwin" {
			want = filepath.Join(home, "bin")
		} else {
			want = filepath.Join(home, ".local", "bin")
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// NewManager (default constructor)
// ---------------------------------------------------------------------------

func TestNewManager(t *testing.T) {
	t.Run("with XDG_CONFIG_HOME set", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)
		m, err := NewManager()
		if err != nil {
			t.Fatalf("NewManager: %v", err)
		}
		wantDir := filepath.Join(xdg, "skint")
		if m.ConfigDir() != wantDir {
			t.Errorf("ConfigDir: got %q, want %q", m.ConfigDir(), wantDir)
		}
		wantFile := filepath.Join(wantDir, "config.yaml")
		if m.ConfigFile() != wantFile {
			t.Errorf("ConfigFile: got %q, want %q", m.ConfigFile(), wantFile)
		}
	})
}
