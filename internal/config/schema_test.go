package config

import (
	"testing"
)

// TestProviderValidate covers validation rules for individual providers.
func TestProviderValidate(t *testing.T) {
	tests := []struct {
		name    string
		p       Provider
		wantErr bool
	}{
		{
			// The "native" builtin is a special case -- it uses Anthropic's
			// default endpoint so BaseURL is not required.
			name: "native builtin without BaseURL is valid",
			p: Provider{
				Name: "native",
				Type: ProviderTypeBuiltin,
			},
			wantErr: false,
		},
		{
			// Other builtin providers must supply a BaseURL.
			name: "non-native builtin without BaseURL is invalid",
			p: Provider{
				Name: "aws-bedrock",
				Type: ProviderTypeBuiltin,
			},
			wantErr: true,
		},
		{
			name: "builtin with BaseURL is valid",
			p: Provider{
				Name:    "aws-bedrock",
				Type:    ProviderTypeBuiltin,
				BaseURL: "https://bedrock.example.com",
			},
			wantErr: false,
		},
		{
			// Custom providers with a bogus APIType should be rejected.
			name: "custom with invalid APIType is rejected",
			p: Provider{
				Name:    "dodgy-custom",
				Type:    ProviderTypeCustom,
				BaseURL: "https://custom.example.com",
				APIType: "grpc",
			},
			wantErr: true,
		},
		{
			name: "custom with anthropic APIType is valid",
			p: Provider{
				Name:    "my-anthropic",
				Type:    ProviderTypeCustom,
				BaseURL: "https://custom.example.com",
				APIType: APITypeAnthropic,
			},
			wantErr: false,
		},
		{
			name: "custom with openai APIType is valid",
			p: Provider{
				Name:    "my-openai",
				Type:    ProviderTypeCustom,
				BaseURL: "https://custom.example.com",
				APIType: APITypeOpenAI,
			},
			wantErr: false,
		},
		{
			// Empty APIType is acceptable for custom providers.
			name: "custom with empty APIType is valid",
			p: Provider{
				Name:    "my-custom",
				Type:    ProviderTypeCustom,
				BaseURL: "https://custom.example.com",
				APIType: "",
			},
			wantErr: false,
		},
		{
			name: "openrouter with BaseURL is valid",
			p: Provider{
				Name:    "or-provider",
				Type:    ProviderTypeOpenRouter,
				BaseURL: "https://openrouter.ai/api",
			},
			wantErr: false,
		},
		{
			// Local providers are exempt from the BaseURL requirement.
			name: "local without BaseURL is valid",
			p: Provider{
				Name: "my-local",
				Type: ProviderTypeLocal,
			},
			wantErr: false,
		},
		{
			name: "local with BaseURL is also valid",
			p: Provider{
				Name:    "my-local-url",
				Type:    ProviderTypeLocal,
				BaseURL: "http://localhost:11434",
			},
			wantErr: false,
		},
		{
			name: "unknown provider type is invalid",
			p: Provider{
				Name: "mystery",
				Type: "alien",
			},
			wantErr: true,
		},
		{
			name: "empty provider type is invalid",
			p: Provider{
				Name: "no-type",
				Type: "",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.p.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

// TestConfigValidateDuplicateProviders checks that Config.Validate rejects
// duplicate provider names.
func TestConfigValidateDuplicateProviders(t *testing.T) {
	cfg := &Config{
		Version:      ConfigVersion,
		OutputFormat: FormatHuman,
		Providers: []*Provider{
			{Name: "same-name", Type: ProviderTypeLocal},
			{Name: "same-name", Type: ProviderTypeLocal},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for duplicate provider names, got nil")
	}
}

// TestConfigValidateEmptyProviderName checks that a provider with an empty
// name is rejected by Config.Validate.
func TestConfigValidateEmptyProviderName(t *testing.T) {
	cfg := &Config{
		Version:      ConfigVersion,
		OutputFormat: FormatHuman,
		Providers: []*Provider{
			{Name: "", Type: ProviderTypeLocal},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty provider name, got nil")
	}
}

// TestNeedsAPIKey verifies which provider types require an API key.
// Local providers and the "native" builtin should not need one.
func TestNeedsAPIKey(t *testing.T) {
	tests := []struct {
		name string
		p    Provider
		want bool
	}{
		{
			name: "native builtin does not need API key",
			p:    Provider{Name: "native", Type: ProviderTypeBuiltin},
			want: false,
		},
		{
			name: "non-native builtin needs API key",
			p:    Provider{Name: "zai", Type: ProviderTypeBuiltin, BaseURL: "https://api.z.ai/api/anthropic"},
			want: true,
		},
		{
			name: "openrouter needs API key",
			p:    Provider{Name: "my-or", Type: ProviderTypeOpenRouter, BaseURL: "https://openrouter.ai/api"},
			want: true,
		},
		{
			name: "local does not need API key",
			p:    Provider{Name: "ollama", Type: ProviderTypeLocal},
			want: false,
		},
		{
			name: "custom needs API key",
			p:    Provider{Name: "my-custom", Type: ProviderTypeCustom, BaseURL: "https://custom.example.com"},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.p.NeedsAPIKey()
			if got != tc.want {
				t.Errorf("NeedsAPIKey(): got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestEffectiveModel verifies the model selection priority:
// DefaultModel takes precedence, then Model, then empty.
func TestEffectiveModel(t *testing.T) {
	tests := []struct {
		name         string
		defaultModel string
		model        string
		want         string
	}{
		{
			name:         "returns DefaultModel when set",
			defaultModel: "claude-opus-4-20250514",
			model:        "claude-sonnet-4-20250514",
			want:         "claude-opus-4-20250514",
		},
		{
			name:         "returns Model when DefaultModel is empty",
			defaultModel: "",
			model:        "claude-sonnet-4-20250514",
			want:         "claude-sonnet-4-20250514",
		},
		{
			name:         "returns empty when both are empty",
			defaultModel: "",
			model:        "",
			want:         "",
		},
		{
			name:         "returns DefaultModel even when Model is also set",
			defaultModel: "preferred-model",
			model:        "fallback-model",
			want:         "preferred-model",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Provider{
				DefaultModel: tc.defaultModel,
				Model:        tc.model,
			}
			got := p.EffectiveModel()
			if got != tc.want {
				t.Errorf("EffectiveModel(): got %q, want %q", got, tc.want)
			}
		})
	}
}
