package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestConfigMarshalUnmarshalRoundTrip verifies that a Config with providers
// survives YAML serialisation and deserialisation without data loss.
func TestConfigMarshalUnmarshalRoundTrip(t *testing.T) {
	original := &Config{
		Version:         ConfigVersion,
		DefaultProvider: "my-openrouter",
		OutputFormat:    FormatJSON,
		ColorEnabled:    true,
		NoBanner:        true,
		Providers: []*Provider{
			{
				Name:        "native",
				Type:        ProviderTypeBuiltin,
				DisplayName: "Native Anthropic",
				Description: "Direct Anthropic API",
			},
			{
				Name:        "my-openrouter",
				Type:        ProviderTypeOpenRouter,
				DisplayName: "OpenRouter",
				BaseURL:     "https://openrouter.ai/api",
				Model:       "anthropic/claude-opus-4-20250514",
				APIKeyRef:   "keyring:openrouter",
			},
			{
				Name:    "my-local",
				Type:    ProviderTypeLocal,
				BaseURL: "http://localhost:8080",
			},
			{
				Name:         "my-custom",
				Type:         ProviderTypeCustom,
				BaseURL:      "https://custom.example.com/v1",
				APIType:      APITypeOpenAI,
				DefaultModel: "gpt-4o",
				ModelMappings: map[string]string{
					"fast": "gpt-4o-mini",
					"slow": "gpt-4o",
				},
			},
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored Config
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Verify top-level fields
	if restored.Version != original.Version {
		t.Errorf("Version: got %q, want %q", restored.Version, original.Version)
	}
	if restored.DefaultProvider != original.DefaultProvider {
		t.Errorf("DefaultProvider: got %q, want %q", restored.DefaultProvider, original.DefaultProvider)
	}
	if restored.OutputFormat != original.OutputFormat {
		t.Errorf("OutputFormat: got %q, want %q", restored.OutputFormat, original.OutputFormat)
	}
	if restored.ColorEnabled != original.ColorEnabled {
		t.Errorf("ColorEnabled: got %v, want %v", restored.ColorEnabled, original.ColorEnabled)
	}
	if restored.NoBanner != original.NoBanner {
		t.Errorf("NoBanner: got %v, want %v", restored.NoBanner, original.NoBanner)
	}

	// Verify all providers survived the round trip
	if len(restored.Providers) != len(original.Providers) {
		t.Fatalf("provider count: got %d, want %d", len(restored.Providers), len(original.Providers))
	}

	for i, orig := range original.Providers {
		got := restored.Providers[i]
		if got.Name != orig.Name {
			t.Errorf("provider[%d].Name: got %q, want %q", i, got.Name, orig.Name)
		}
		if got.Type != orig.Type {
			t.Errorf("provider[%d].Type: got %q, want %q", i, got.Type, orig.Type)
		}
		if got.DisplayName != orig.DisplayName {
			t.Errorf("provider[%d].DisplayName: got %q, want %q", i, got.DisplayName, orig.DisplayName)
		}
		if got.BaseURL != orig.BaseURL {
			t.Errorf("provider[%d].BaseURL: got %q, want %q", i, got.BaseURL, orig.BaseURL)
		}
		if got.APIType != orig.APIType {
			t.Errorf("provider[%d].APIType: got %q, want %q", i, got.APIType, orig.APIType)
		}
		if got.DefaultModel != orig.DefaultModel {
			t.Errorf("provider[%d].DefaultModel: got %q, want %q", i, got.DefaultModel, orig.DefaultModel)
		}
		if got.Model != orig.Model {
			t.Errorf("provider[%d].Model: got %q, want %q", i, got.Model, orig.Model)
		}
		if got.APIKeyRef != orig.APIKeyRef {
			t.Errorf("provider[%d].APIKeyRef: got %q, want %q", i, got.APIKeyRef, orig.APIKeyRef)
		}
	}

	// Verify model mappings specifically for the custom provider
	customOrig := original.Providers[3]
	customGot := restored.Providers[3]
	if len(customGot.ModelMappings) != len(customOrig.ModelMappings) {
		t.Fatalf("custom provider ModelMappings length: got %d, want %d",
			len(customGot.ModelMappings), len(customOrig.ModelMappings))
	}
	for k, v := range customOrig.ModelMappings {
		if customGot.ModelMappings[k] != v {
			t.Errorf("custom provider ModelMappings[%q]: got %q, want %q", k, customGot.ModelMappings[k], v)
		}
	}
}

// TestGetProvider checks that GetProvider returns the right provider by name
// and nil for names that don't exist.
func TestGetProvider(t *testing.T) {
	cfg := &Config{
		Providers: []*Provider{
			{Name: "alpha", Type: ProviderTypeBuiltin, BaseURL: "https://alpha.example.com"},
			{Name: "beta", Type: ProviderTypeLocal},
		},
	}

	tests := []struct {
		name     string
		lookup   string
		wantNil  bool
		wantName string
	}{
		{
			name:     "existing provider returned correctly",
			lookup:   "alpha",
			wantNil:  false,
			wantName: "alpha",
		},
		{
			name:     "second provider returned correctly",
			lookup:   "beta",
			wantNil:  false,
			wantName: "beta",
		},
		{
			name:    "missing provider returns nil",
			lookup:  "gamma",
			wantNil: true,
		},
		{
			name:    "empty string returns nil",
			lookup:  "",
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cfg.GetProvider(tc.lookup)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil for lookup %q, got provider %q", tc.lookup, got.Name)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected provider %q, got nil", tc.wantName)
			}
			if got.Name != tc.wantName {
				t.Errorf("got Name %q, want %q", got.Name, tc.wantName)
			}
		})
	}
}

// TestAddProvider verifies adding providers, including duplicate rejection.
func TestAddProvider(t *testing.T) {
	tests := []struct {
		name      string
		initial   []*Provider
		add       *Provider
		wantErr   bool
		wantCount int
	}{
		{
			name:    "add to empty config succeeds",
			initial: []*Provider{},
			add: &Provider{
				Name:    "native",
				Type:    ProviderTypeBuiltin,
				BaseURL: "", // native is exempt from BaseURL requirement
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "add new provider succeeds",
			initial: []*Provider{
				{Name: "native", Type: ProviderTypeBuiltin},
			},
			add: &Provider{
				Name: "my-local",
				Type: ProviderTypeLocal,
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "duplicate name is rejected",
			initial: []*Provider{
				{Name: "native", Type: ProviderTypeBuiltin},
			},
			add: &Provider{
				Name: "native",
				Type: ProviderTypeBuiltin,
			},
			wantErr:   true,
			wantCount: 1,
		},
		{
			name:    "invalid provider is rejected",
			initial: []*Provider{},
			add: &Provider{
				Name: "bad",
				Type: "nonexistent",
			},
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Providers: tc.initial}
			err := cfg.AddProvider(tc.add)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cfg.Providers) != tc.wantCount {
				t.Errorf("provider count: got %d, want %d", len(cfg.Providers), tc.wantCount)
			}
		})
	}
}

// TestRemoveProvider verifies removing providers by name.
func TestRemoveProvider(t *testing.T) {
	tests := []struct {
		name       string
		initial    []*Provider
		remove     string
		wantResult bool
		wantCount  int
	}{
		{
			name: "remove existing provider",
			initial: []*Provider{
				{Name: "alpha", Type: ProviderTypeBuiltin},
				{Name: "beta", Type: ProviderTypeLocal},
			},
			remove:     "alpha",
			wantResult: true,
			wantCount:  1,
		},
		{
			name: "remove nonexistent provider returns false",
			initial: []*Provider{
				{Name: "alpha", Type: ProviderTypeBuiltin},
			},
			remove:     "gamma",
			wantResult: false,
			wantCount:  1,
		},
		{
			name:       "remove from empty config returns false",
			initial:    []*Provider{},
			remove:     "anything",
			wantResult: false,
			wantCount:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Providers: tc.initial}
			got := cfg.RemoveProvider(tc.remove)
			if got != tc.wantResult {
				t.Errorf("RemoveProvider(%q): got %v, want %v", tc.remove, got, tc.wantResult)
			}
			if len(cfg.Providers) != tc.wantCount {
				t.Errorf("provider count after removal: got %d, want %d", len(cfg.Providers), tc.wantCount)
			}
		})
	}
}
