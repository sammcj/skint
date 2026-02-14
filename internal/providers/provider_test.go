package providers

import (
	"fmt"
	"testing"

	"github.com/sammcj/skint/internal/config"
)

// assertEnvVars is a helper that verifies the env map contains exactly the expected keys and values.
func assertEnvVars(t *testing.T, got map[string]string, want map[string]string) {
	t.Helper()

	for k, wantV := range want {
		gotV, ok := got[k]
		if !ok {
			t.Errorf("missing expected env var %q (wanted %q)", k, wantV)
			continue
		}
		if gotV != wantV {
			t.Errorf("env var %q = %q, want %q", k, gotV, wantV)
		}
	}

	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("unexpected env var %q = %q", k, got[k])
		}
	}
}

func TestBuiltinProvider_GetEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		provider *BuiltinProvider
		want     map[string]string
	}{
		{
			name: "all fields populated with model mappings",
			provider: &BuiltinProvider{baseProvider: baseProvider{
				name:          "test",
				baseURL:       "https://example.com",
				apiKey:        "token123",
				model:         "test-model",
				modelMappings: map[string]string{"haiku": "h", "sonnet": "s", "opus": "o"},
			}},
			want: map[string]string{
				"ANTHROPIC_BASE_URL":             "https://example.com",
				"ANTHROPIC_AUTH_TOKEN":           "token123",
				"ANTHROPIC_MODEL":                "test-model",
				"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "h",
				"ANTHROPIC_DEFAULT_SONNET_MODEL": "s",
				"ANTHROPIC_DEFAULT_OPUS_MODEL":   "o",
			},
		},
		{
			name: "small model mapping sets ANTHROPIC_SMALL_FAST_MODEL",
			provider: &BuiltinProvider{baseProvider: baseProvider{
				name:          "kimi-like",
				baseURL:       "https://api.kimi.example",
				apiKey:        "key",
				model:         "kimi-k2.5",
				modelMappings: map[string]string{"small": "kimi-k2.5"},
			}},
			want: map[string]string{
				"ANTHROPIC_BASE_URL":         "https://api.kimi.example",
				"ANTHROPIC_AUTH_TOKEN":       "key",
				"ANTHROPIC_MODEL":            "kimi-k2.5",
				"ANTHROPIC_SMALL_FAST_MODEL": "kimi-k2.5",
			},
		},
		{
			name: "empty fields are omitted",
			provider: &BuiltinProvider{baseProvider: baseProvider{
				name: "native",
			}},
			want: map[string]string{},
		},
		{
			name: "base URL only",
			provider: &BuiltinProvider{baseProvider: baseProvider{
				name:    "minimal",
				baseURL: "https://api.example.com",
			}},
			want: map[string]string{
				"ANTHROPIC_BASE_URL": "https://api.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.GetEnvVars()
			assertEnvVars(t, got, tt.want)
		})
	}
}

func TestBuiltinProvider_GetEnvVars_KeyEnvVar(t *testing.T) {
	// When keyEnvVar is set, the API key should use that env var instead of ANTHROPIC_AUTH_TOKEN
	p := &BuiltinProvider{baseProvider: baseProvider{
		name:      "anthropic",
		apiKey:    "sk-ant-test",
		keyEnvVar: "ANTHROPIC_API_KEY",
	}}
	got := p.GetEnvVars()
	want := map[string]string{
		"ANTHROPIC_API_KEY": "sk-ant-test",
	}
	assertEnvVars(t, got, want)
}

func TestOpenRouterProvider_GetEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		provider *OpenRouterProvider
		want     map[string]string
	}{
		{
			name: "sets openrouter base URL and overrides all model tiers",
			provider: &OpenRouterProvider{baseProvider: baseProvider{
				name:   "test-or",
				apiKey: "sk-or-123",
				model:  "openai/gpt-4o",
			}},
			want: map[string]string{
				"ANTHROPIC_BASE_URL":             "https://openrouter.ai/api",
				"ANTHROPIC_AUTH_TOKEN":           "sk-or-123",
				"ANTHROPIC_API_KEY":              "",
				"ANTHROPIC_DEFAULT_OPUS_MODEL":   "openai/gpt-4o",
				"ANTHROPIC_DEFAULT_SONNET_MODEL": "openai/gpt-4o",
				"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "openai/gpt-4o",
				"ANTHROPIC_SMALL_FAST_MODEL":     "openai/gpt-4o",
			},
		},
		{
			name: "empty model omits tier overrides but still clears ANTHROPIC_API_KEY",
			provider: &OpenRouterProvider{baseProvider: baseProvider{
				name:   "or-no-model",
				apiKey: "sk-or-456",
			}},
			want: map[string]string{
				"ANTHROPIC_BASE_URL":   "https://openrouter.ai/api",
				"ANTHROPIC_AUTH_TOKEN": "sk-or-456",
				"ANTHROPIC_API_KEY":    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.GetEnvVars()
			assertEnvVars(t, got, tt.want)
		})
	}
}

func TestLocalProvider_GetEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		provider *LocalProvider
		want     map[string]string
	}{
		{
			name: "without auth token",
			provider: &LocalProvider{baseProvider: baseProvider{
				name:    "ollama",
				baseURL: "http://localhost:11434",
			}},
			want: map[string]string{
				"ANTHROPIC_BASE_URL": "http://localhost:11434",
			},
		},
		{
			name: "with auth token clears ANTHROPIC_API_KEY",
			provider: &LocalProvider{
				baseProvider: baseProvider{
					name:    "ollama-auth",
					baseURL: "http://localhost:11434",
				},
				authToken: "ollama",
			},
			want: map[string]string{
				"ANTHROPIC_BASE_URL":   "http://localhost:11434",
				"ANTHROPIC_AUTH_TOKEN": "ollama",
				"ANTHROPIC_API_KEY":    "",
			},
		},
		{
			name: "with model set",
			provider: &LocalProvider{
				baseProvider: baseProvider{
					name:    "lmstudio",
					baseURL: "http://localhost:1234",
					model:   "qwen2.5-coder",
				},
				authToken: "lmstudio",
			},
			want: map[string]string{
				"ANTHROPIC_BASE_URL":   "http://localhost:1234",
				"ANTHROPIC_AUTH_TOKEN": "lmstudio",
				"ANTHROPIC_API_KEY":    "",
				"ANTHROPIC_MODEL":      "qwen2.5-coder",
			},
		},
		{
			name: "without auth token and without model",
			provider: &LocalProvider{baseProvider: baseProvider{
				name:    "llamacpp",
				baseURL: "http://localhost:8000",
			}},
			want: map[string]string{
				"ANTHROPIC_BASE_URL": "http://localhost:8000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.GetEnvVars()
			assertEnvVars(t, got, tt.want)
		})
	}
}

func TestCustomProvider_GetEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		provider *CustomProvider
		want     map[string]string
	}{
		{
			name: "openai api type sets OPENAI env vars",
			provider: &CustomProvider{
				baseProvider: baseProvider{
					name:    "custom-openai",
					baseURL: "https://api.example.com",
					apiKey:  "key123",
					model:   "gpt-4",
				},
				apiType: "openai",
			},
			want: map[string]string{
				"OPENAI_BASE_URL": "https://api.example.com",
				"OPENAI_API_KEY":  "key123",
				"OPENAI_MODEL":    "gpt-4",
			},
		},
		{
			name: "anthropic api type sets ANTHROPIC env vars",
			provider: &CustomProvider{
				baseProvider: baseProvider{
					name:    "custom-anthropic",
					baseURL: "https://custom.anthropic.example",
					apiKey:  "sk-ant-custom",
					model:   "claude-3-sonnet",
				},
				apiType: "anthropic",
			},
			want: map[string]string{
				"ANTHROPIC_BASE_URL":   "https://custom.anthropic.example",
				"ANTHROPIC_AUTH_TOKEN": "sk-ant-custom",
				"ANTHROPIC_MODEL":      "claude-3-sonnet",
			},
		},
		{
			name: "empty api type defaults to anthropic behaviour",
			provider: &CustomProvider{
				baseProvider: baseProvider{
					name:    "custom-default",
					baseURL: "https://fallback.example",
					apiKey:  "fb-key",
					model:   "fb-model",
				},
				apiType: "",
			},
			want: map[string]string{
				"ANTHROPIC_BASE_URL":   "https://fallback.example",
				"ANTHROPIC_AUTH_TOKEN": "fb-key",
				"ANTHROPIC_MODEL":      "fb-model",
			},
		},
		{
			name: "openai with empty optional fields omits them",
			provider: &CustomProvider{
				baseProvider: baseProvider{
					name: "custom-sparse",
				},
				apiType: "openai",
			},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.GetEnvVars()
			assertEnvVars(t, got, tt.want)
		})
	}
}

func TestFromConfig(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Provider
		wantType      string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "builtin provider type",
			cfg: &config.Provider{
				Name:          "zai",
				Type:          config.ProviderTypeBuiltin,
				BaseURL:       "https://api.z.ai/api/anthropic",
				DefaultModel:  "glm-5",
				ModelMappings: map[string]string{"haiku": "glm-5"},
			},
			wantType: "*providers.BuiltinProvider",
		},
		{
			name: "native builtin does not need API key",
			cfg: &config.Provider{
				Name: "native",
				Type: config.ProviderTypeBuiltin,
			},
			wantType: "*providers.BuiltinProvider",
		},
		{
			name: "openrouter provider type uses Model field",
			cfg: &config.Provider{
				Name:    "or",
				Type:    config.ProviderTypeOpenRouter,
				BaseURL: "https://openrouter.ai/api",
				Model:   "openai/gpt-4o",
			},
			wantType: "*providers.OpenRouterProvider",
		},
		{
			name: "local provider type",
			cfg: &config.Provider{
				Name:      "ollama",
				Type:      config.ProviderTypeLocal,
				BaseURL:   "http://localhost:11434",
				AuthToken: "ollama",
			},
			wantType: "*providers.LocalProvider",
		},
		{
			name: "custom provider type",
			cfg: &config.Provider{
				Name:    "my-custom",
				Type:    config.ProviderTypeCustom,
				BaseURL: "https://custom.example",
				APIType: config.APITypeOpenAI,
				Model:   "gpt-4",
			},
			wantType: "*providers.CustomProvider",
		},
		{
			name: "unknown provider type returns error",
			cfg: &config.Provider{
				Name: "bogus",
				Type: "imaginary",
			},
			wantErr:       true,
			wantErrSubstr: "unknown provider type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromConfig(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.wantErrSubstr != "" {
					if !containsSubstring(err.Error(), tt.wantErrSubstr) {
						t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrSubstr)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotType := typeName(got)
			if gotType != tt.wantType {
				t.Errorf("got type %q, want %q", gotType, tt.wantType)
			}
		})
	}
}

func TestFromConfig_NativeBuiltinNoAPIKey(t *testing.T) {
	// The native provider should not require an API key and should validate
	// successfully without one. This is the regression test for the fix where
	// NeedsAPIKey() was incorrectly returning true for native.
	cp := &config.Provider{
		Name: "native",
		Type: config.ProviderTypeBuiltin,
	}

	p, err := FromConfig(cp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.NeedsAPIKey() {
		t.Error("native provider should not need an API key")
	}

	// Validate should pass without an API key
	if err := p.Validate(); err != nil {
		t.Errorf("native provider should validate without API key, got: %v", err)
	}

	// GetEnvVars should return empty map (no proxy configuration)
	env := p.GetEnvVars()
	if len(env) != 0 {
		t.Errorf("native provider GetEnvVars() should be empty, got %v", env)
	}
}

func TestFromConfig_AnthropicAPIProvider(t *testing.T) {
	// The anthropic provider should use ANTHROPIC_API_KEY instead of ANTHROPIC_AUTH_TOKEN,
	// should need an API key, and should not require a base URL.
	cp := &config.Provider{
		Name:      "anthropic",
		Type:      config.ProviderTypeBuiltin,
		KeyEnvVar: "ANTHROPIC_API_KEY",
	}
	cp.SetResolvedAPIKey("sk-ant-real-key")

	p, err := FromConfig(cp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !p.NeedsAPIKey() {
		t.Error("anthropic provider should need an API key")
	}

	env := p.GetEnvVars()
	wantEnv := map[string]string{
		"ANTHROPIC_API_KEY": "sk-ant-real-key",
	}
	assertEnvVars(t, env, wantEnv)
}

func TestFromConfig_OpenRouterUsesModelField(t *testing.T) {
	// Verify that OpenRouter specifically reads from the Model field (not DefaultModel).
	cp := &config.Provider{
		Name:         "or-model-check",
		Type:         config.ProviderTypeOpenRouter,
		BaseURL:      "https://openrouter.ai/api",
		DefaultModel: "should-not-be-used",
		Model:        "anthropic/claude-3.5-sonnet",
	}
	cp.SetResolvedAPIKey("sk-or-test")

	p, err := FromConfig(cp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := p.GetEnvVars()
	// The OpenRouter provider should use the Model field value for tier overrides.
	if got := env["ANTHROPIC_DEFAULT_SONNET_MODEL"]; got != "anthropic/claude-3.5-sonnet" {
		t.Errorf("ANTHROPIC_DEFAULT_SONNET_MODEL = %q, want %q", got, "anthropic/claude-3.5-sonnet")
	}
}

func TestFromConfig_CustomFallsBackToModelField(t *testing.T) {
	// When DefaultModel is empty, custom providers should use the Model field.
	cp := &config.Provider{
		Name:    "custom-fallback",
		Type:    config.ProviderTypeCustom,
		BaseURL: "https://api.example.com",
		APIType: config.APITypeOpenAI,
		Model:   "gpt-4-turbo",
	}
	cp.SetResolvedAPIKey("key-456")

	p, err := FromConfig(cp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := p.GetModel(); got != "gpt-4-turbo" {
		t.Errorf("GetModel() = %q, want %q", got, "gpt-4-turbo")
	}

	env := p.GetEnvVars()
	if got := env["OPENAI_MODEL"]; got != "gpt-4-turbo" {
		t.Errorf("OPENAI_MODEL = %q, want %q", got, "gpt-4-turbo")
	}
}

// containsSubstring checks whether s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// typeName returns the type name including package prefix and pointer indicator.
func typeName(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", v)
}
