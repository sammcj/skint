package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sammcj/skint/internal/config"
)

// renderModelPicker renders the model picker as a bordered overlay.
func (m *Model) renderModelPicker() string {
	if m.modelFetching {
		return m.styles.Dimmed.Render("  Fetching models...")
	}
	if m.modelFetchErr != "" {
		return m.styles.Dimmed.Render("  Could not fetch models: " + m.modelFetchErr)
	}
	if !m.modelPickerOpen || len(m.fetchedModels) == 0 {
		return ""
	}

	filtered := m.filteredModels()
	if len(filtered) == 0 {
		content := m.styles.Dimmed.Render("No models match filter")
		pickerWidth := m.width - 16
		pickerWidth = max(pickerWidth, 30)
		return m.styles.PickerBox.Width(pickerWidth).Render(content) + "\n"
	}

	var inner strings.Builder

	// Calculate visible window
	start := 0
	end := len(filtered)
	if end > maxPickerVisible {
		if m.modelPickerIdx >= maxPickerVisible {
			start = m.modelPickerIdx - maxPickerVisible + 1
		}
		end = start + maxPickerVisible
		if end > len(filtered) {
			end = len(filtered)
			start = end - maxPickerVisible
			start = max(start, 0)
		}
	}

	for i := start; i < end; i++ {
		mi := filtered[i]
		label := mi.Label()
		if i == m.modelPickerIdx {
			inner.WriteString(m.styles.ListSelected.Render("> " + label))
		} else {
			inner.WriteString(m.styles.Dimmed.Render("  " + label))
		}
		if i < end-1 {
			inner.WriteString("\n")
		}
	}

	if len(filtered) > maxPickerVisible {
		inner.WriteString("\n")
		inner.WriteString(m.styles.Dimmed.Render(fmt.Sprintf("(%d/%d shown, type to filter)", min(maxPickerVisible, len(filtered)), len(filtered))))
	}

	// Title line
	titleLine := m.styles.PickerBoxTitle.Render("Available Models")
	if filterVal := m.getModelValue(); filterVal != "" {
		titleLine += m.styles.Dimmed.Render(fmt.Sprintf(" [filter: %s]", filterVal))
	}

	pickerWidth := m.width - 16
	pickerWidth = max(pickerWidth, 30)
	return m.styles.PickerBox.Width(pickerWidth).Render(titleLine+"\n"+inner.String()) + "\n"
}

// renderFormField renders a single form field with consistent container styling.
// When focused: primary-coloured border. When unfocused: dim border container.
// The valueOverride parameter is used when the display value differs from the stored value
// (e.g., masked API keys). If empty, value is used as-is.
func (m *Model) renderFormField(label, value, hint string, focusIdx int, required, isMasked bool, inputWidth int) string {
	var b strings.Builder

	labelStyle := m.styles.Label
	if m.inputFocus == focusIdx {
		labelStyle = m.styles.InputPrompt
	}

	reqIndicator := ""
	if required {
		reqIndicator = m.styles.Error.Render("*")
	}

	b.WriteString(labelStyle.Render(label) + reqIndicator)
	b.WriteString("\n")

	displayValue := value
	isEmpty := value == "" || (isMasked && value == hint)
	if isEmpty {
		displayValue = hint
	}

	if m.inputFocus == focusIdx {
		// Focused: primary border
		b.WriteString(m.styles.Input.Width(inputWidth).Render(displayValue))
	} else {
		// Unfocused: dim border container
		if isEmpty {
			b.WriteString(m.styles.InputInactive.Width(inputWidth).Render(
				m.styles.Dimmed.Render(displayValue),
			))
		} else {
			b.WriteString(m.styles.InputInactive.Width(inputWidth).Render(
				m.styles.Value.Render(displayValue),
			))
		}
	}
	b.WriteString("\n")

	return b.String()
}

// modelPickerHelpHint returns help text for the model picker based on current state.
func (m *Model) modelPickerHelpHint() string {
	if m.modelPickerOpen {
		return "↑/↓: select model • enter: confirm • esc: close • type: filter"
	}
	if m.isOnModelField() && len(m.fetchedModels) > 0 {
		return "ctrl+f: re-fetch models"
	}
	if m.isOnModelField() {
		return "ctrl+f: fetch models"
	}
	return ""
}

func (m *Model) viewMainScreen() string {
	var b strings.Builder

	// Compact single-line header
	configuredCount := 0
	for _, pi := range m.providerList {
		if pi.configured {
			configuredCount++
		}
	}

	activeDisplayName := "Claude Subscription"
	if m.cfg.DefaultProvider != "" {
		activeDisplayName = m.cfg.DefaultProvider
		for _, pi := range m.providerList {
			if pi.definition != nil && pi.definition.Name == m.cfg.DefaultProvider {
				activeDisplayName = pi.definition.DisplayName
				break
			}
		}
	}

	sep := m.styles.HeaderSep.Render(" · ")
	header := m.styles.HeaderLine.Render("Skint") +
		sep + m.styles.Dimmed.Render("active: ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(activeDisplayName) +
		sep + m.styles.Dimmed.Render(fmt.Sprintf("%d configured", configuredCount)) +
		sep + m.styles.Success.Render("✓") + m.styles.Dimmed.Render(" configured  ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render("█") + m.styles.Dimmed.Render(" active")
	b.WriteString(header)
	b.WriteString("\n\n")

	// List
	b.WriteString(m.styles.List.Render(m.list.View()))
	b.WriteString("\n")

	// Two-line help bar
	navHelp := m.styles.Help.Render("↑/k ↓/j navigate  enter select  esc back")
	actHelp := m.styles.Help.Render("e edit  a/c add custom  u launch  t test  q quit")
	b.WriteString(m.styles.Footer.Render(navHelp + "\n" + actHelp))

	return b.String()
}

func (m *Model) viewProviderConfig() string {
	var b strings.Builder

	// Check if editing or adding
	existingProvider := m.cfg.GetProvider(m.selectedProvider.Name)
	isEditing := existingProvider != nil

	// Compact header with breadcrumb
	action := "Configure"
	if isEditing {
		action = "Edit"
	}
	header := m.styles.HeaderLine.Render("Skint") +
		m.styles.HeaderSep.Render(" › ") +
		m.styles.Subtitle.Render(fmt.Sprintf("%s %s", action, m.selectedProvider.DisplayName))
	b.WriteString(header)
	b.WriteString("\n")

	// Show provider info
	info := m.styles.Box.Width(m.width - 8).Render(
		m.styles.BoxTitle.Render("Setup Instructions") + "\n" +
			m.getLocalProviderInstructions(),
	)
	b.WriteString(info)
	b.WriteString("\n\n")

	// Form fields with consistent containers
	inputWidth := m.width - 20
	inputWidth = max(inputWidth, 30)

	fields := []struct {
		label string
		value string
		focus int
		hint  string
		req   bool
	}{
		{"Base URL", m.localProviderURL, 0, m.selectedProvider.BaseURL, true},
		{"Auth Token", m.localProviderAuthToken, 1, "optional", false},
		{"Model", m.localProviderModel, 2, "e.g., qwen3-coder", false},
	}

	for _, f := range fields {
		b.WriteString(m.renderFormField(f.label, f.value, f.hint, f.focus, f.req, false, inputWidth))

		// Render model picker after the model field
		if f.focus == 2 {
			pickerView := m.renderModelPicker()
			if pickerView != "" {
				b.WriteString(pickerView)
			}
		}
		b.WriteString("\n")
	}

	// Error message
	if m.inputError != "" {
		b.WriteString(m.styles.Error.Render("✗ " + m.inputError))
		b.WriteString("\n")
	}

	// Two-line help
	navHelp := m.styles.Help.Render("↑/↓/tab navigate  enter save  esc back")
	actHelp := ""
	if hint := m.modelPickerHelpHint(); hint != "" {
		actHelp = m.styles.Help.Render(hint)
	}
	helpContent := navHelp
	if actHelp != "" {
		helpContent += "\n" + actHelp
	}
	b.WriteString(m.styles.Footer.Render(helpContent))

	return b.String()
}

func (m *Model) getLocalProviderInstructions() string {
	switch m.selectedProvider.Name {
	case "ollama":
		return `Ollama serves local models with an Anthropic-compatible API.

Setup:
  1. Install Ollama: https://ollama.com
  2. Pull a model: ollama pull qwen3-coder
  3. Start serving: ollama serve

Recommended models:
  • qwen3-coder
  • glm-5
  • gpt-oss:20b`
	case "lmstudio":
		return `LM Studio runs local models with an Anthropic-compatible API.

Setup:
  1. Install LM Studio: https://lmstudio.ai/download
  2. Load a model in the app
  3. Start the server (port 1234)

Usage:
  skint use lmstudio --model <model-name>`
	case "llamacpp":
		return `llama.cpp's llama-server with Anthropic-compatible API.

Setup:
  1. Build llama.cpp: https://github.com/ggml-org/llama.cpp
  2. Start server:
     ./llama-server --model <model.gguf> --port 8000 --jinja

Usage:
  skint use llamacpp --model <model-name>`
	default:
		return m.selectedProvider.Description
	}
}

func (m *Model) viewAPIKeyInput() string {
	var b strings.Builder

	// Compact header with breadcrumb
	header := m.styles.HeaderLine.Render("Skint") +
		m.styles.HeaderSep.Render(" › ") +
		m.styles.Subtitle.Render(fmt.Sprintf("Configure %s", m.selectedProvider.DisplayName))
	b.WriteString(header)
	b.WriteString("\n")

	// Provider info
	endpoint := m.selectedProvider.BaseURL
	if endpoint == "" {
		endpoint = "(default)"
	}
	info := m.styles.Box.Width(m.width - 8).Render(
		m.styles.Label.Render("Provider: ") + m.selectedProvider.DisplayName + "\n" +
			m.styles.Label.Render("Endpoint: ") + m.styles.Info.Render(endpoint),
	)
	b.WriteString(info)
	b.WriteString("\n\n")

	inputWidth := m.width - 20
	inputWidth = max(inputWidth, 30)

	// API Key field
	apiKeyRequired := !m.hasExistingKey
	emptyPlaceholder := "Type your API key..."
	if m.hasExistingKey {
		emptyPlaceholder = "Key saved - leave blank to keep, or type to replace"
	}
	masked := strings.Repeat("•", len(m.apiKeyInput))
	if masked == "" {
		masked = emptyPlaceholder
	}
	b.WriteString(m.renderFormField("API Key", masked, emptyPlaceholder, 0, apiKeyRequired, true, inputWidth))

	// Model field
	modelRequired := m.selectedProvider.DefaultModel == "" && len(m.selectedProvider.ModelMappings) == 0
	modelHint := "e.g., anthropic/claude-sonnet-4"
	if m.selectedProvider.DefaultModel != "" {
		modelHint = m.selectedProvider.DefaultModel
	}
	b.WriteString(m.renderFormField("Model", m.modelInput, modelHint, 1, modelRequired, false, inputWidth))

	// Model picker
	pickerView := m.renderModelPicker()
	if pickerView != "" {
		b.WriteString(pickerView)
	}
	b.WriteString("\n")

	// Error message
	if m.inputError != "" {
		b.WriteString(m.styles.Error.Render("✗ " + m.inputError))
		b.WriteString("\n")
	}

	// Two-line help
	navHelp := m.styles.Help.Render("↑/↓/tab navigate  enter save  esc cancel")
	actHelp := ""
	if hint := m.modelPickerHelpHint(); hint != "" {
		actHelp = m.styles.Help.Render(hint)
	}
	helpContent := navHelp
	if actHelp != "" {
		helpContent += "\n" + actHelp
	}
	b.WriteString(m.styles.Footer.Render(helpContent))

	return b.String()
}

func (m *Model) viewSuccess() string {
	var b strings.Builder

	// Compact header
	header := m.styles.HeaderLine.Render("Skint") +
		m.styles.HeaderSep.Render(" › ") +
		m.styles.Success.Render("Success")
	b.WriteString(header)
	b.WriteString("\n\n")

	b.WriteString(m.styles.Success.Render(m.message))
	b.WriteString("\n\n")

	// Next steps
	providerName := ""
	if m.selectedProvider != nil {
		providerName = m.selectedProvider.Name
	} else if m.customProviderName != "" {
		providerName = m.customProviderName
	}
	if providerName != "" {
		next := m.styles.Box.Width(m.width - 8).Render(
			m.styles.BoxTitle.Render("Next Steps") + "\n" +
				m.styles.Info.Render("→") + " Use it: " + m.styles.Success.Render("skint use "+providerName) + "\n" +
				m.styles.Info.Render("→") + " Test it: " + m.styles.Success.Render("skint test "+providerName),
		)
		b.WriteString(next)
		b.WriteString("\n\n")

		// Styled action buttons
		var continueBtn, launchBtn string
		if m.successOption == 0 {
			continueBtn = m.styles.ButtonActive.Render("Continue")
			launchBtn = m.styles.ButtonInactive.Render(fmt.Sprintf("Launch Claude with %s", providerName))
		} else {
			continueBtn = m.styles.ButtonInactive.Render("Continue")
			launchBtn = m.styles.ButtonActive.Render(fmt.Sprintf("Launch Claude with %s", providerName))
		}
		b.WriteString(continueBtn + "  " + launchBtn)
		b.WriteString("\n\n")
	}

	// Help
	if providerName != "" {
		help := m.styles.Help.Render("←/→ select  enter confirm  esc back")
		b.WriteString(m.styles.Footer.Render(help))
	} else {
		helpText := "press any key to continue..."
		if m.done {
			helpText = "press any key to exit..."
		}
		b.WriteString(m.styles.Footer.Render(m.styles.Help.Render(helpText)))
	}

	return b.String()
}

func (m *Model) viewError() string {
	var b strings.Builder

	// Compact header
	header := m.styles.HeaderLine.Render("Skint") +
		m.styles.HeaderSep.Render(" › ") +
		m.styles.Error.Render("Error")
	b.WriteString(header)
	b.WriteString("\n\n")

	b.WriteString(m.styles.Error.Render("✗ " + m.message))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Footer.Render(m.styles.Help.Render("press any key to continue...")))

	return b.String()
}

func (m *Model) viewCustomProvider() string {
	var b strings.Builder

	// Check if editing or adding
	existingProvider := m.cfg.GetProvider(m.customProviderName)
	isEditing := existingProvider != nil

	// Compact header with breadcrumb
	action := "Add Custom Provider"
	if isEditing {
		action = "Edit Custom Provider"
	}
	header := m.styles.HeaderLine.Render("Skint") +
		m.styles.HeaderSep.Render(" › ") +
		m.styles.Subtitle.Render(action)
	b.WriteString(header)
	b.WriteString("\n")

	// Instructions box
	instructions := m.styles.Box.Width(m.width - 8).Render(
		m.styles.BoxTitle.Render("Configuration Guide") + "\n" +
			m.styles.Dimmed.Render("Configure any OpenAI or Anthropic compatible API endpoint.") + "\n\n" +
			m.styles.Label.Render("Examples:") + "\n" +
			"  • OpenAI: " + m.styles.Info.Render("https://api.openai.com") + " → /v1/chat/completions\n" +
			"  • Anthropic: " + m.styles.Info.Render("https://api.anthropic.com") + " → /messages\n" +
			"  • Local: " + m.styles.Info.Render("http://localhost:8000") + " → your custom endpoint",
	)
	b.WriteString(instructions)
	b.WriteString("\n\n")

	// Form fields with consistent containers
	inputWidth := m.width - 20
	inputWidth = max(inputWidth, 30)

	// Check if provider has saved API key for hint text
	hasSavedKey := existingProvider != nil && existingProvider.APIKeyRef != ""

	apiKeyHint := "optional"
	if hasSavedKey {
		apiKeyHint = "(saved - type to change)"
	}

	// Mask API key value for display
	maskedAPIKey := m.apiKeyInput
	if maskedAPIKey != "" {
		maskedAPIKey = strings.Repeat("•", len(maskedAPIKey))
	}

	fields := []struct {
		label    string
		value    string
		focus    int
		hint     string
		isMasked bool
		req      bool
	}{
		{"Name", m.customProviderName, 0, "lowercase-id", false, true},
		{"Display Name", m.customProviderDisplay, 1, "optional", false, false},
		{"Base URL", m.customProviderURL, 2, "https://api.example.com", false, true},
		{"API Key", maskedAPIKey, 3, apiKeyHint, true, false},
		{"Model", m.customProviderModel, 4, "e.g., gpt-4o, claude-3-sonnet", false, true},
		{"API Type", m.customProviderAPIType, 5, "↑/↓ to change", false, true},
	}

	for _, f := range fields {
		b.WriteString(m.renderFormField(f.label, f.value, f.hint, f.focus, f.req, f.isMasked, inputWidth))

		// Render model picker after the model field
		if f.focus == 4 {
			pickerView := m.renderModelPicker()
			if pickerView != "" {
				b.WriteString(pickerView)
			}
		}
	}

	// API Type explanation
	apiTypeBox := m.styles.Box.Width(m.width - 8).Render(
		m.styles.Label.Render("API Type: ") +
			m.styles.Success.Render("• ") + m.styles.Info.Render(config.APITypeAnthropic) + m.styles.Dimmed.Render(" (messages endpoint)   ") +
			m.styles.Success.Render("• ") + m.styles.Info.Render(config.APITypeOpenAI) + m.styles.Dimmed.Render(" (/v1/chat/completions)"),
	)
	b.WriteString(apiTypeBox)

	// Error message
	if m.inputError != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.Error.Render("✗ " + m.inputError))
	}

	b.WriteString("\n")

	// Two-line help
	navHelp := m.styles.Help.Render("↑/↓/tab navigate  enter submit  esc cancel")
	actHelp := ""
	if hint := m.modelPickerHelpHint(); hint != "" {
		actHelp = m.styles.Help.Render(hint)
	}
	helpContent := navHelp
	if actHelp != "" {
		helpContent += "\n" + actHelp
	}
	b.WriteString(m.styles.Footer.Render(helpContent))

	return b.String()
}
