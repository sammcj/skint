package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHasOldInstallation(t *testing.T) {
	tests := []struct {
		name       string
		createFile bool
		want       bool
	}{
		{
			name:       "returns true when secrets.env exists",
			createFile: true,
			want:       true,
		},
		{
			name:       "returns false when secrets.env does not exist",
			createFile: false,
			want:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.createFile {
				if err := os.WriteFile(filepath.Join(dir, "secrets.env"), []byte("KEY=val\n"), 0o600); err != nil {
					t.Fatalf("failed to create secrets.env: %v", err)
				}
			}
			m := &Migration{dataDir: dir}
			got := m.HasOldInstallation()
			if got != tc.want {
				t.Errorf("HasOldInstallation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSecretsFile(t *testing.T) {
	dir := t.TempDir()
	m := &Migration{dataDir: dir}
	want := filepath.Join(dir, "secrets.env")
	if got := m.SecretsFile(); got != want {
		t.Errorf("SecretsFile() = %q, want %q", got, want)
	}
}

func TestUnescape(t *testing.T) {
	m := &Migration{}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "backslash",
			input: `hello\\world`,
			want:  `hello\world`,
		},
		{
			name:  "escaped double quote",
			input: `say \"hello\"`,
			want:  `say "hello"`,
		},
		{
			name:  "escaped single quote",
			input: `it\'s`,
			want:  `it's`,
		},
		{
			name:  "escaped newline",
			input: `line1\nline2`,
			want:  "line1\nline2",
		},
		{
			name:  "escaped tab",
			input: `col1\tcol2`,
			want:  "col1\tcol2",
		},
		{
			name:  "escaped dollar",
			input: `cost is \$5`,
			want:  `cost is $5`,
		},
		{
			name:  "no escapes passes through unchanged",
			input: "plain text 123",
			want:  "plain text 123",
		},
		{
			name:  "multiple escapes in one string",
			input: `say \"hello\" and \$cost is \$5`,
			want:  "say \"hello\" and $cost is $5",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := m.unescape(tc.input)
			if got != tc.want {
				t.Errorf("unescape(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestLoadSecrets(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name:    "valid key=value pairs",
			content: "FOO=bar\nBAZ=qux\n",
			want:    map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name:    "comments and blank lines are skipped",
			content: "# this is a comment\n\nKEY=value\n  # indented comment\n\n",
			want:    map[string]string{"KEY": "value"},
		},
		{
			name:    "double quoted values have quotes stripped",
			content: `MY_KEY="some value"` + "\n",
			want:    map[string]string{"MY_KEY": "some value"},
		},
		{
			name:    "single quoted values have quotes stripped",
			content: "MY_KEY='some value'\n",
			want:    map[string]string{"MY_KEY": "some value"},
		},
		{
			name:    "escaped characters are unescaped",
			content: `TOKEN=hello\\world` + "\n",
			want:    map[string]string{"TOKEN": `hello\world`},
		},
		{
			name:    "malformed lines without equals are skipped",
			content: "GOOD=value\nmalformed line\nALSO_GOOD=yes\n",
			want:    map[string]string{"GOOD": "value", "ALSO_GOOD": "yes"},
		},
		{
			name:    "value containing equals sign",
			content: "URL=https://example.com?foo=bar\n",
			want:    map[string]string{"URL": "https://example.com?foo=bar"},
		},
		{
			name:    "whitespace around key and value is trimmed",
			content: "  KEY  =  value  \n",
			want:    map[string]string{"KEY": "value"},
		},
		{
			name:    "empty file returns empty map",
			content: "",
			want:    map[string]string{},
		},
		{
			name:    "file with only comments and blanks returns empty map",
			content: "# comment\n\n# another\n",
			want:    map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			secretsPath := filepath.Join(dir, "secrets.env")
			if err := os.WriteFile(secretsPath, []byte(tc.content), 0o600); err != nil {
				t.Fatalf("failed to write secrets.env: %v", err)
			}

			m := &Migration{dataDir: dir}
			got, err := m.LoadSecrets()
			if err != nil {
				t.Fatalf("LoadSecrets() returned unexpected error: %v", err)
			}

			if len(got) != len(tc.want) {
				t.Fatalf("LoadSecrets() returned %d entries, want %d\ngot:  %v\nwant: %v", len(got), len(tc.want), got, tc.want)
			}
			for k, wantV := range tc.want {
				gotV, ok := got[k]
				if !ok {
					t.Errorf("missing key %q in result", k)
					continue
				}
				if gotV != wantV {
					t.Errorf("key %q: got %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}

func TestLoadSecrets_MissingFile(t *testing.T) {
	dir := t.TempDir()
	m := &Migration{dataDir: dir}
	_, err := m.LoadSecrets()
	if err == nil {
		t.Fatal("LoadSecrets() expected error for missing file, got nil")
	}
}

func TestLoadSecrets_SymlinkRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not reliable on Windows")
	}

	dir := t.TempDir()
	realFile := filepath.Join(dir, "real-secrets.env")
	if err := os.WriteFile(realFile, []byte("KEY=value\n"), 0o600); err != nil {
		t.Fatalf("failed to write real file: %v", err)
	}

	symlinkPath := filepath.Join(dir, "secrets.env")
	if err := os.Symlink(realFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	m := &Migration{dataDir: dir}
	_, err := m.LoadSecrets()
	if err == nil {
		t.Fatal("LoadSecrets() expected error for symlink, got nil")
	}
	if got := err.Error(); got != "secrets file is a symlink - refusing for security" {
		t.Errorf("unexpected error message: %q", got)
	}
}

func TestImport(t *testing.T) {
	t.Run("imports known provider with API key", func(t *testing.T) {
		dir := t.TempDir()
		content := "ZAI_API_KEY=test-zai-key-12345\n"
		if err := os.WriteFile(filepath.Join(dir, "secrets.env"), []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write secrets.env: %v", err)
		}

		m := &Migration{dataDir: dir}
		cfg, keys, err := m.Import()
		if err != nil {
			t.Fatalf("Import() returned unexpected error: %v", err)
		}

		// The native provider should always be present
		nativeFound := false
		for _, p := range cfg.Providers {
			if p.Name == "native" {
				nativeFound = true
			}
		}
		if !nativeFound {
			t.Error("expected native provider to be present in imported config")
		}

		// The zai provider should be present because we supplied its key
		zaiFound := false
		for _, p := range cfg.Providers {
			if p.Name == "zai" {
				zaiFound = true
				if p.Type != ProviderTypeBuiltin {
					t.Errorf("zai provider type: got %q, want %q", p.Type, ProviderTypeBuiltin)
				}
				if p.DisplayName != "Z.AI" {
					t.Errorf("zai DisplayName: got %q, want %q", p.DisplayName, "Z.AI")
				}
				if p.BaseURL != "https://api.z.ai/api/anthropic" {
					t.Errorf("zai BaseURL: got %q, want %q", p.BaseURL, "https://api.z.ai/api/anthropic")
				}
			}
		}
		if !zaiFound {
			t.Error("expected zai provider to be present in imported config")
		}

		// The keys map should contain the zai key
		if gotKey, ok := keys["zai"]; !ok {
			t.Error("keys map missing 'zai' entry")
		} else if gotKey != "test-zai-key-12345" {
			t.Errorf("keys[zai]: got %q, want %q", gotKey, "test-zai-key-12345")
		}
	})

	t.Run("imports local providers without API key", func(t *testing.T) {
		dir := t.TempDir()
		// Empty secrets file -- local providers should still appear
		if err := os.WriteFile(filepath.Join(dir, "secrets.env"), []byte(""), 0o600); err != nil {
			t.Fatalf("failed to write secrets.env: %v", err)
		}

		m := &Migration{dataDir: dir}
		cfg, _, err := m.Import()
		if err != nil {
			t.Fatalf("Import() returned unexpected error: %v", err)
		}

		localNames := map[string]bool{"ollama": false, "lmstudio": false, "llamacpp": false}
		for _, p := range cfg.Providers {
			if _, isLocal := localNames[p.Name]; isLocal {
				localNames[p.Name] = true
				if p.Type != ProviderTypeLocal {
					t.Errorf("provider %q type: got %q, want %q", p.Name, p.Type, ProviderTypeLocal)
				}
			}
		}
		for name, found := range localNames {
			if !found {
				t.Errorf("expected local provider %q in imported config", name)
			}
		}
	})

	t.Run("imports OpenRouter provider and model entries", func(t *testing.T) {
		dir := t.TempDir()
		content := "OPENROUTER_API_KEY=or-key-abc\nOPENROUTER_MODEL_FAST=anthropic/claude-3-haiku\n"
		if err := os.WriteFile(filepath.Join(dir, "secrets.env"), []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write secrets.env: %v", err)
		}

		m := &Migration{dataDir: dir}
		cfg, keys, err := m.Import()
		if err != nil {
			t.Fatalf("Import() returned unexpected error: %v", err)
		}

		// Check the OpenRouter model provider was created
		orFound := false
		for _, p := range cfg.Providers {
			if p.Name == "or-fast" {
				orFound = true
				if p.Type != ProviderTypeOpenRouter {
					t.Errorf("or-fast type: got %q, want %q", p.Type, ProviderTypeOpenRouter)
				}
				if p.Model != "anthropic/claude-3-haiku" {
					t.Errorf("or-fast Model: got %q, want %q", p.Model, "anthropic/claude-3-haiku")
				}
			}
		}
		if !orFound {
			t.Error("expected or-fast provider in imported config")
		}

		// The main OpenRouter key should be stored
		if gotKey, ok := keys["openrouter"]; !ok {
			t.Error("keys map missing 'openrouter' entry")
		} else if gotKey != "or-key-abc" {
			t.Errorf("keys[openrouter]: got %q, want %q", gotKey, "or-key-abc")
		}

		// The model-specific entry should also get the OpenRouter key
		if gotKey, ok := keys["or-fast"]; !ok {
			t.Error("keys map missing 'or-fast' entry")
		} else if gotKey != "or-key-abc" {
			t.Errorf("keys[or-fast]: got %q, want %q", gotKey, "or-key-abc")
		}
	})

	t.Run("skips builtin providers without API key in secrets", func(t *testing.T) {
		dir := t.TempDir()
		// No API keys at all
		if err := os.WriteFile(filepath.Join(dir, "secrets.env"), []byte(""), 0o600); err != nil {
			t.Fatalf("failed to write secrets.env: %v", err)
		}

		m := &Migration{dataDir: dir}
		cfg, _, err := m.Import()
		if err != nil {
			t.Fatalf("Import() returned unexpected error: %v", err)
		}

		// Builtin providers that need an API key (e.g. zai, minimax) should NOT be present
		for _, p := range cfg.Providers {
			if p.Type == ProviderTypeBuiltin && p.Name != "native" {
				t.Errorf("builtin provider %q should not be present without an API key", p.Name)
			}
		}
	})

	t.Run("returns error when secrets file missing", func(t *testing.T) {
		dir := t.TempDir()
		m := &Migration{dataDir: dir}
		_, _, err := m.Import()
		if err == nil {
			t.Fatal("Import() expected error for missing secrets file, got nil")
		}
	})
}

func TestCleanup(t *testing.T) {
	t.Run("removes old installation files", func(t *testing.T) {
		dir := t.TempDir()

		filesToCreate := []string{
			filepath.Join(dir, "secrets.env"),
			filepath.Join(dir, "banner"),
			filepath.Join(dir, "skint-full.sh"),
		}
		for _, f := range filesToCreate {
			if err := os.WriteFile(f, []byte("content"), 0o600); err != nil {
				t.Fatalf("failed to create %s: %v", f, err)
			}
		}

		m := &Migration{dataDir: dir}
		if err := m.Cleanup(); err != nil {
			t.Fatalf("Cleanup() returned unexpected error: %v", err)
		}

		for _, f := range filesToCreate {
			if _, err := os.Stat(f); !os.IsNotExist(err) {
				t.Errorf("file %s still exists after Cleanup()", f)
			}
		}
	})

	t.Run("succeeds when files do not exist", func(t *testing.T) {
		dir := t.TempDir()
		m := &Migration{dataDir: dir}
		if err := m.Cleanup(); err != nil {
			t.Fatalf("Cleanup() returned unexpected error when files are absent: %v", err)
		}
	})

	t.Run("removes empty data directory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "skint-data")
		if err := os.Mkdir(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
		// Create one file so cleanup has something to remove
		secretsPath := filepath.Join(subDir, "secrets.env")
		if err := os.WriteFile(secretsPath, []byte("X=Y\n"), 0o600); err != nil {
			t.Fatalf("failed to write secrets.env: %v", err)
		}

		m := &Migration{dataDir: subDir}
		if err := m.Cleanup(); err != nil {
			t.Fatalf("Cleanup() returned unexpected error: %v", err)
		}

		// The data directory should be removed because it's now empty
		if _, err := os.Stat(subDir); !os.IsNotExist(err) {
			t.Error("expected data directory to be removed after cleanup")
		}
	})

	t.Run("does not remove non-empty data directory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "skint-data")
		if err := os.Mkdir(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
		// Create both an expected file and an extra file
		if err := os.WriteFile(filepath.Join(subDir, "secrets.env"), []byte("X=Y\n"), 0o600); err != nil {
			t.Fatalf("failed to write secrets.env: %v", err)
		}
		if err := os.WriteFile(filepath.Join(subDir, "other.txt"), []byte("keep me"), 0o600); err != nil {
			t.Fatalf("failed to write other.txt: %v", err)
		}

		m := &Migration{dataDir: subDir}
		if err := m.Cleanup(); err != nil {
			t.Fatalf("Cleanup() returned unexpected error: %v", err)
		}

		// Directory should still exist because other.txt remains
		if _, err := os.Stat(subDir); os.IsNotExist(err) {
			t.Error("data directory should not be removed when it still contains files")
		}
	})
}
