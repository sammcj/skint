package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/skint/internal/models"
)

// modelFieldIndex returns the form field index for the model field on the current screen.
func (m *Model) modelFieldIndex() int {
	switch m.screen {
	case ScreenAPIKeyInput:
		return 1
	case ScreenProviderConfig:
		return 2
	case ScreenCustomProvider:
		return 4
	default:
		return -1
	}
}

// isOnModelField returns true if the input focus is on the model field.
func (m *Model) isOnModelField() bool {
	return m.inputFocus == m.modelFieldIndex()
}

// getModelValue returns the current model input value for the active screen.
func (m *Model) getModelValue() string {
	switch m.screen {
	case ScreenAPIKeyInput:
		return m.modelInput
	case ScreenProviderConfig:
		return m.localProviderModel
	case ScreenCustomProvider:
		return m.customProviderModel
	default:
		return ""
	}
}

// setModelValue sets the model input for the current screen.
func (m *Model) setModelValue(value string) {
	switch m.screen {
	case ScreenAPIKeyInput:
		m.modelInput = value
	case ScreenProviderConfig:
		m.localProviderModel = value
	case ScreenCustomProvider:
		m.customProviderModel = value
	}
}

// updateModelPicker handles key events when the model picker is open.
// Returns true if the event was consumed by the picker.
func (m *Model) updateModelPicker(msg tea.KeyMsg) bool {
	if !m.modelPickerOpen {
		return false
	}

	filtered := m.filteredModels()

	switch msg.Type {
	case tea.KeyEsc:
		m.modelPickerOpen = false
	case tea.KeyEnter:
		if len(filtered) > 0 && m.modelPickerIdx < len(filtered) {
			m.setModelValue(filtered[m.modelPickerIdx].ID)
		}
		m.modelPickerOpen = false
	case tea.KeyUp:
		if m.modelPickerIdx > 0 {
			m.modelPickerIdx--
		}
	case tea.KeyDown:
		if m.modelPickerIdx < len(filtered)-1 {
			m.modelPickerIdx++
		}
	case tea.KeyBackspace:
		current := m.getModelValue()
		if len(current) > 0 {
			m.setModelValue(current[:len(current)-1])
			m.modelPickerIdx = 0
		}
	case tea.KeyRunes:
		current := m.getModelValue()
		for _, r := range msg.Runes {
			if r >= 32 && r < 127 {
				current += string(r)
			}
		}
		m.setModelValue(current)
		m.modelPickerIdx = 0
	}
	return true
}

// triggerModelFetch starts an async model fetch if not already fetching.
func (m *Model) triggerModelFetch() tea.Cmd {
	if m.modelFetching {
		return nil
	}
	baseURL, apiKey, providerName := m.resolveProviderForFetch()
	if providerName == "" {
		return nil
	}
	m.modelFetching = true
	m.modelFetchErr = ""
	m.fetchedModels = nil
	m.modelPickerOpen = false
	m.modelPickerIdx = 0
	return fetchModelsCmd(baseURL, apiKey, providerName)
}

// modelsFetchedMsg is sent when an async model fetch completes.
type modelsFetchedMsg struct {
	models []models.ModelInfo
	err    error
}

// fetchModelsCmd returns a Bubble Tea command that fetches models asynchronously.
func fetchModelsCmd(baseURL, apiKey, providerName string) tea.Cmd {
	return func() tea.Msg {
		result := models.FetchModels(baseURL, apiKey, providerName)
		return modelsFetchedMsg{models: result.Models, err: result.Err}
	}
}

// maxPickerVisible is the maximum number of models to show in the picker at once.
const maxPickerVisible = 10

// filteredModels returns the subset of fetched models matching the current model input.
// The model input field doubles as the typeahead filter.
func (m *Model) filteredModels() []models.ModelInfo {
	filter := strings.ToLower(m.getModelValue())
	if filter == "" {
		return m.fetchedModels
	}
	var filtered []models.ModelInfo
	for _, mi := range m.fetchedModels {
		if strings.Contains(strings.ToLower(mi.ID), filter) ||
			strings.Contains(strings.ToLower(mi.DisplayName), filter) {
			filtered = append(filtered, mi)
		}
	}
	return filtered
}

// resetModelPicker clears all model picker state.
func (m *Model) resetModelPicker() {
	m.fetchedModels = nil
	m.modelPickerOpen = false
	m.modelPickerIdx = 0
	m.modelFetching = false
	m.modelFetchErr = ""
}

// resolveProviderForFetch determines the base URL, API key, and provider name
// to use for model fetching based on the current screen and selected provider.
func (m *Model) resolveProviderForFetch() (baseURL, apiKey, providerName string) {
	switch m.screen {
	case ScreenProviderConfig:
		// Local provider config screen
		if m.selectedProvider != nil {
			providerName = m.selectedProvider.Name
			baseURL = m.localProviderURL
		}
	case ScreenAPIKeyInput:
		// Built-in / OpenRouter provider
		if m.selectedProvider != nil {
			providerName = m.selectedProvider.Name
			baseURL = m.selectedProvider.BaseURL
			// Use the key being entered, or fall back to existing resolved key
			apiKey = m.apiKeyInput
			if apiKey == "" {
				if p := m.cfg.GetProvider(m.selectedProvider.Name); p != nil {
					apiKey = p.GetAPIKey()
				}
			}
		}
	case ScreenCustomProvider:
		providerName = m.customProviderName
		baseURL = m.customProviderURL
		apiKey = m.apiKeyInput
	}
	return baseURL, apiKey, providerName
}
