package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/models"
	"github.com/sammcj/skint/internal/providers"
)

// newAPIKeyScreenModel returns a model parked on the API key screen with a
// builtin provider selected and focus on the model field, ready to fetch.
func newAPIKeyScreenModel() *Model {
	m := NewModel(config.NewDefaultConfig(), nil)
	m.screen = ScreenAPIKeyInput
	m.selectedProvider = &providers.Definition{Name: "zai", BaseURL: "https://api.z.ai/api/anthropic"}
	m.inputFocus = 1 // model field
	return m
}

// TestModelsFetchedOpensPickerOnModelField is the positive control: results for
// the current fetch, still focused on the model field, open the picker.
func TestModelsFetchedOpensPickerOnModelField(t *testing.T) {
	m := newAPIKeyScreenModel()
	_ = m.triggerModelFetch() // bumps generation; the returned network cmd is not run
	gen := m.fetchGeneration

	model, _ := m.Update(modelsFetchedMsg{
		models:     []models.ModelInfo{{ID: "glm-5"}},
		generation: gen,
	})
	m = model.(*Model)

	if !m.modelPickerOpen {
		t.Error("picker should open when focus is on the model field and results arrive")
	}
}

// TestModelsFetchedDoesNotOpenPickerOffModelField covers the keystroke-hijack
// bug: a fetch fired from the model field must not open the picker once focus
// has moved to the API key field.
func TestModelsFetchedDoesNotOpenPickerOffModelField(t *testing.T) {
	m := newAPIKeyScreenModel()
	_ = m.triggerModelFetch()
	gen := m.fetchGeneration

	// User tabs to the API key field before the fetch completes.
	m.inputFocus = 0

	model, _ := m.Update(modelsFetchedMsg{
		models:     []models.ModelInfo{{ID: "glm-5"}},
		generation: gen,
	})
	m = model.(*Model)

	if m.modelPickerOpen {
		t.Error("picker must not open while focus is on the API key field")
	}
}

// TestModelsFetchedStaleGenerationIgnored covers the stale-provider variant:
// once the picker is reset (e.g. the user navigated away), a late result from
// the previous fetch must be discarded.
func TestModelsFetchedStaleGenerationIgnored(t *testing.T) {
	m := newAPIKeyScreenModel()
	_ = m.triggerModelFetch()
	staleGen := m.fetchGeneration

	m.resetModelPicker() // invalidates the in-flight fetch

	model, _ := m.Update(modelsFetchedMsg{
		models:     []models.ModelInfo{{ID: "glm-5"}},
		generation: staleGen,
	})
	m = model.(*Model)

	if m.modelPickerOpen {
		t.Error("stale fetch result must not open the picker")
	}
	if m.fetchedModels != nil {
		t.Error("stale fetch result must not populate fetchedModels")
	}
}

// TestCustomProviderFlowClearsStaleSelection covers the wrong-provider bug:
// entering the custom provider flow after configuring another provider must
// clear the stale selection so the success screen resolves the custom provider.
func TestCustomProviderFlowClearsStaleSelection(t *testing.T) {
	m := NewModel(config.NewDefaultConfig(), nil)
	// Simulate a provider configured earlier in the session.
	m.selectedProvider = &providers.Definition{Name: "zai"}

	// Press 'c' on the main screen to add a custom provider.
	model, _ := m.updateMainScreen(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = model.(*Model)

	if m.screen != ScreenCustomProvider {
		t.Fatalf("screen: got %v, want ScreenCustomProvider", m.screen)
	}
	if m.selectedProvider != nil {
		t.Fatal("selectedProvider must be cleared when entering the custom provider flow")
	}

	// Fill and submit the custom provider (no API key -> no secrets manager needed).
	m.customProviderName = "mycustom"
	m.customProviderURL = "https://api.example.com"
	m.customProviderModel = "some-model"
	m.customProviderAPIType = config.APITypeAnthropic

	model, _ = m.submitCustomProvider()
	m = model.(*Model)

	if m.screen != ScreenSuccess {
		t.Fatalf("screen after submit: got %v, want ScreenSuccess", m.screen)
	}

	// The success screen resolves the provider from selectedProvider first,
	// then customProviderName; with the stale selection cleared it must name
	// the custom provider.
	resolved := ""
	if m.selectedProvider != nil {
		resolved = m.selectedProvider.Name
	} else if m.customProviderName != "" {
		resolved = m.customProviderName
	}
	if resolved != "mycustom" {
		t.Errorf("resolved success provider: got %q, want %q", resolved, "mycustom")
	}
}
