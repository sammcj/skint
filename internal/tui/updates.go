package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/providers"
)

func (m *Model) updateMainScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyRunes:
		switch msg.String() {
		case "q":
			if !m.list.SettingFilter() {
				m.done = true
				return m, tea.Quit
			}
		case "t":
			if !m.list.SettingFilter() {
				m.resultAction = "test"
				m.done = true
				return m, tea.Quit
			}
		case "o":
			if !m.list.SettingFilter() {
				m.screen = ScreenOpenRouter
				m.inputFocus = 0
				return m, nil
			}
		case "c", "a":
			if !m.list.SettingFilter() {
				m.screen = ScreenCustomProvider
				m.inputFocus = 0
				m.resetCustomProviderForm()
				return m, nil
			}
		case "e":
			if !m.list.SettingFilter() {
				if item, ok := m.list.SelectedItem().(ProviderItem); ok && !item.isAddNew {
					return m.handleProviderEdit(item)
				}
			}
		}
	case tea.KeyEsc:
		if !m.list.SettingFilter() {
			m.done = true
			return m, tea.Quit
		}
	case tea.KeyCtrlC:
		m.done = true
		return m, tea.Quit
	case tea.KeyEnter:
		if item, ok := m.list.SelectedItem().(ProviderItem); ok {
			if item.isAddNew {
				m.screen = ScreenCustomProvider
				m.inputFocus = 0
				m.resetCustomProviderForm()
				return m, nil
			}
			m.selectedProvider = item.definition
			return m.handleProviderSelect(item)
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) handleProviderSelect(item ProviderItem) (tea.Model, tea.Cmd) {
	def := item.definition
	p := m.cfg.GetProvider(def.Name)

	// Check if provider is already configured
	isConfigured := item.configured || (p != nil && (!p.NeedsAPIKey() || p.GetAPIKey() != ""))

	// If already configured, use it (set as current and quit)
	if isConfigured {
		m.selectedProvider = def
		m.message = fmt.Sprintf("✓ Using %s", def.DisplayName)
		m.messageType = "success"
		m.screen = ScreenSuccess
		m.done = true
		return m, nil
	}

	// Native provider needs no configuration
	if def.Name == "native" {
		if m.onProviderSelect != nil {
			if err := m.onProviderSelect(def.Name); err != nil {
				m.message = err.Error()
				m.messageType = "error"
				m.screen = ScreenError
			} else {
				m.message = fmt.Sprintf("✓ %s is ready to use", def.DisplayName)
				m.messageType = "success"
				m.screen = ScreenSuccess
				m.done = true
			}
		}
		return m, nil
	}

	// Local providers need a config form
	if def.Type == config.ProviderTypeLocal {
		m.initLocalProviderForm(def)
		m.screen = ScreenProviderConfig
		return m, nil
	}

	// Built-in providers need API key
	m.screen = ScreenAPIKeyInput
	m.apiKeyInput = ""
	m.inputError = ""
	m.inputFocus = 0
	return m, nil
}

func (m *Model) handleProviderEdit(item ProviderItem) (tea.Model, tea.Cmd) {
	def := item.definition
	p := m.cfg.GetProvider(def.Name)

	// Check if provider is configured
	isConfigured := item.configured || (p != nil && (!p.NeedsAPIKey() || p.GetAPIKey() != ""))

	if !isConfigured {
		// Not configured yet - just configure it
		m.selectedProvider = def
		return m.handleProviderSelect(item)
	}

	// Provider is configured - open appropriate edit screen
	m.selectedProvider = def

	switch def.Type {
	case config.ProviderTypeLocal:
		// Local providers - show config form with existing values
		m.localProviderURL = p.BaseURL
		m.localProviderAuthToken = p.AuthToken
		m.localProviderModel = p.EffectiveModel()
		m.inputFocus = 0
		m.inputError = ""
		m.screen = ScreenProviderConfig
	case config.ProviderTypeCustom:
		// Custom providers - open custom provider form with existing values
		m.customProviderName = p.Name
		m.customProviderDisplay = p.DisplayName
		m.customProviderURL = p.BaseURL
		m.customProviderModel = p.Model
		m.customProviderAPIType = p.APIType
		if m.customProviderAPIType == "" {
			m.customProviderAPIType = config.APITypeAnthropic
		}
		// Don't show API key (it's masked), but allow editing
		m.apiKeyInput = ""
		m.inputFocus = 0
		m.inputError = ""
		m.screen = ScreenCustomProvider
	default:
		// Built-in providers - open API key input
		m.screen = ScreenAPIKeyInput
		m.apiKeyInput = ""
		m.inputError = ""
		m.inputFocus = 0
	}

	return m, nil
}

func (m *Model) initLocalProviderForm(def *providers.Definition) {
	// Pre-populate from existing config if available, otherwise use definition defaults
	p := m.cfg.GetProvider(def.Name)
	if p != nil {
		m.localProviderURL = p.BaseURL
		m.localProviderAuthToken = p.AuthToken
		m.localProviderModel = p.EffectiveModel()
	} else {
		m.localProviderURL = def.BaseURL
		m.localProviderAuthToken = def.AuthToken
		m.localProviderModel = def.DefaultModel
	}
	m.inputFocus = 0
	m.inputError = ""
}

func (m *Model) updateProviderConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.screen = ScreenMain
		return m, nil
	case tea.KeyCtrlC:
		m.done = true
		return m, tea.Quit
	case tea.KeyTab, tea.KeyDown:
		m.inputFocus = (m.inputFocus + 1) % localFormFieldCount
		return m, nil
	case tea.KeyShiftTab, tea.KeyUp:
		m.inputFocus = (m.inputFocus + localFormFieldCount - 1) % localFormFieldCount
		return m, nil
	case tea.KeyEnter:
		// Validate and submit
		if m.localProviderURL == "" {
			m.inputError = "Base URL is required"
			m.inputFocus = 0
			return m, nil
		}
		if !strings.HasPrefix(m.localProviderURL, "http://") && !strings.HasPrefix(m.localProviderURL, "https://") {
			m.inputError = "URL must start with http:// or https://"
			m.inputFocus = 0
			return m, nil
		}
		return m.submitLocalProvider()
	case tea.KeyBackspace:
		m.inputError = ""
		switch m.inputFocus {
		case 0:
			if len(m.localProviderURL) > 0 {
				m.localProviderURL = m.localProviderURL[:len(m.localProviderURL)-1]
			}
		case 1:
			if len(m.localProviderAuthToken) > 0 {
				m.localProviderAuthToken = m.localProviderAuthToken[:len(m.localProviderAuthToken)-1]
			}
		case 2:
			if len(m.localProviderModel) > 0 {
				m.localProviderModel = m.localProviderModel[:len(m.localProviderModel)-1]
			}
		}
		return m, nil
	}

	// Handle rune input
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		m.inputError = ""
		for _, r := range msg.Runes {
			if r >= 32 && r < 127 {
				switch m.inputFocus {
				case 0:
					m.localProviderURL += string(r)
				case 1:
					m.localProviderAuthToken += string(r)
				case 2:
					m.localProviderModel += string(r)
				}
			}
		}
	}

	return m, nil
}

func (m *Model) submitLocalProvider() (tea.Model, tea.Cmd) {
	if m.selectedProvider == nil {
		return m, nil
	}

	provider := &config.Provider{
		Name:        m.selectedProvider.Name,
		Type:        m.selectedProvider.Type,
		DisplayName: m.selectedProvider.DisplayName,
		Description: m.selectedProvider.Description,
		BaseURL:     m.localProviderURL,
		AuthToken:   m.localProviderAuthToken,
		Model:       m.localProviderModel,
	}

	m.cfg.RemoveProvider(provider.Name)
	if err := m.cfg.AddProvider(provider); err != nil {
		m.message = err.Error()
		m.messageType = "error"
		m.screen = ScreenError
	} else {
		m.message = fmt.Sprintf("✓ %s configured", m.selectedProvider.DisplayName)
		m.messageType = "success"
		m.screen = ScreenSuccess
	}
	return m, nil
}

func (m *Model) updateAPIKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.screen = ScreenMain
		m.apiKeyInput = ""
		m.inputError = ""
		return m, nil
	case tea.KeyCtrlC:
		m.done = true
		return m, tea.Quit
	case tea.KeyEnter:
		if m.apiKeyInput == "" {
			m.inputError = "API key is required"
			return m, nil
		}
		if len(m.apiKeyInput) < 8 {
			m.inputError = "API key too short (minimum 8 characters)"
			return m, nil
		}
		// Store API key
		ref, err := m.secretsMgr.StoreWithReference(m.selectedProvider.Name, m.apiKeyInput)
		if err != nil {
			m.inputError = fmt.Sprintf("Failed to store API key: %v", err)
			return m, nil
		}

		// Create or update provider config
		provider := &config.Provider{
			Name:          m.selectedProvider.Name,
			Type:          m.selectedProvider.Type,
			DisplayName:   m.selectedProvider.DisplayName,
			Description:   m.selectedProvider.Description,
			BaseURL:       m.selectedProvider.BaseURL,
			DefaultModel:  m.selectedProvider.DefaultModel,
			ModelMappings: m.selectedProvider.ModelMappings,
			APIKeyRef:     ref,
		}

		m.cfg.RemoveProvider(provider.Name)
		if err := m.cfg.AddProvider(provider); err != nil {
			m.inputError = err.Error()
			return m, nil
		}

		m.message = fmt.Sprintf("✓ %s configured successfully", m.selectedProvider.DisplayName)
		m.messageType = "success"
		m.screen = ScreenSuccess
		m.done = true
		m.apiKeyInput = ""
		return m, nil
	case tea.KeyBackspace:
		if len(m.apiKeyInput) > 0 {
			m.apiKeyInput = m.apiKeyInput[:len(m.apiKeyInput)-1]
		}
		m.inputError = ""
		return m, nil
	}

	// Handle rune input for API key
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		for _, r := range msg.Runes {
			// Only accept printable ASCII for API keys
			if r >= 32 && r < 127 {
				m.apiKeyInput += string(r)
			}
		}
		m.inputError = ""
	}

	return m, nil
}

// updateCustomProvider handles input for the custom provider form
func (m *Model) updateCustomProvider(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.screen = ScreenMain
		m.resetCustomProviderForm()
		return m, nil
	case tea.KeyCtrlC:
		m.done = true
		return m, tea.Quit
	case tea.KeyTab, tea.KeyDown:
		// Cycle through form fields
		m.inputFocus = (m.inputFocus + 1) % customFormFieldCount
		return m, nil
	case tea.KeyShiftTab, tea.KeyUp:
		// Cycle backwards
		m.inputFocus = (m.inputFocus + customFormFieldCount - 1) % customFormFieldCount
		return m, nil
	case tea.KeyEnter:
		// If on API type field, toggle between options
		if m.inputFocus == 5 {
			if m.customProviderAPIType == config.APITypeAnthropic {
				m.customProviderAPIType = config.APITypeOpenAI
			} else {
				m.customProviderAPIType = config.APITypeAnthropic
			}
			return m, nil
		}
		// Try to submit if all fields filled
		if m.customProviderName != "" && m.customProviderURL != "" && m.customProviderModel != "" {
			return m.submitCustomProvider()
		}
		m.inputFocus = (m.inputFocus + 1) % customFormFieldCount
		return m, nil
	case tea.KeyBackspace:
		m.inputError = ""
		switch m.inputFocus {
		case 0:
			if len(m.customProviderName) > 0 {
				m.customProviderName = m.customProviderName[:len(m.customProviderName)-1]
			}
		case 1:
			if len(m.customProviderDisplay) > 0 {
				m.customProviderDisplay = m.customProviderDisplay[:len(m.customProviderDisplay)-1]
			}
		case 2:
			if len(m.customProviderURL) > 0 {
				m.customProviderURL = m.customProviderURL[:len(m.customProviderURL)-1]
			}
		case 3:
			if len(m.apiKeyInput) > 0 {
				m.apiKeyInput = m.apiKeyInput[:len(m.apiKeyInput)-1]
			}
		case 4:
			if len(m.customProviderModel) > 0 {
				m.customProviderModel = m.customProviderModel[:len(m.customProviderModel)-1]
			}
		}
		return m, nil
	}

	// Handle rune input
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		m.inputError = ""
		for _, r := range msg.Runes {
			if r >= 32 && r < 127 {
				switch m.inputFocus {
				case 0:
					m.customProviderName += string(r)
				case 1:
					m.customProviderDisplay += string(r)
				case 2:
					m.customProviderURL += string(r)
				case 3:
					m.apiKeyInput += string(r)
				case 4:
					m.customProviderModel += string(r)
				}
			}
		}
	}

	return m, nil
}

func (m *Model) submitCustomProvider() (tea.Model, tea.Cmd) {
	// Validate inputs
	if m.customProviderName == "" {
		m.inputError = "Provider name is required"
		m.inputFocus = 0
		return m, nil
	}

	// Validate name format (lowercase, alphanumeric, hyphens only)
	for _, r := range m.customProviderName {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			m.inputError = "Name must be lowercase alphanumeric with hyphens/underscores only"
			m.inputFocus = 0
			return m, nil
		}
	}

	if m.customProviderURL == "" {
		m.inputError = "Base URL is required"
		m.inputFocus = 2
		return m, nil
	}

	// Validate URL format
	if !strings.HasPrefix(m.customProviderURL, "http://") && !strings.HasPrefix(m.customProviderURL, "https://") {
		m.inputError = "URL must start with http:// or https://"
		m.inputFocus = 2
		return m, nil
	}

	if m.customProviderModel == "" {
		m.inputError = "Model name is required"
		m.inputFocus = 4
		return m, nil
	}

	// Set default API type if not set
	if m.customProviderAPIType == "" {
		m.customProviderAPIType = config.APITypeAnthropic
	}

	// Set default display name if not provided
	displayName := m.customProviderDisplay
	if displayName == "" {
		displayName = m.customProviderName
	}

	// Store API key if provided
	var apiKeyRef string
	if m.apiKeyInput != "" {
		ref, err := m.secretsMgr.StoreWithReference(m.customProviderName, m.apiKeyInput)
		if err != nil {
			m.inputError = fmt.Sprintf("Failed to store API key: %v", err)
			return m, nil
		}
		apiKeyRef = ref
	}

	// Create provider config
	provider := &config.Provider{
		Name:        m.customProviderName,
		Type:        config.ProviderTypeCustom,
		DisplayName: displayName,
		Description: fmt.Sprintf("Custom %s provider", m.customProviderAPIType),
		BaseURL:     m.customProviderURL,
		Model:       m.customProviderModel,
		APIKeyRef:   apiKeyRef,
		APIType:     m.customProviderAPIType,
	}

	// Remove existing if present
	m.cfg.RemoveProvider(provider.Name)

	// Add provider
	if err := m.cfg.AddProvider(provider); err != nil {
		m.inputError = err.Error()
		return m, nil
	}

	m.message = fmt.Sprintf("✓ Custom provider '%s' added", displayName)
	m.messageType = "success"
	m.screen = ScreenSuccess
	// Don't quit - return to main screen so user can select it
	m.resetCustomProviderForm()
	return m, nil
}

func (m *Model) resetCustomProviderForm() {
	m.customProviderName = ""
	m.customProviderDisplay = ""
	m.customProviderURL = ""
	m.customProviderModel = ""
	m.customProviderAPIType = config.APITypeAnthropic
	m.apiKeyInput = ""
	m.inputFocus = 0
	m.inputError = ""
}
