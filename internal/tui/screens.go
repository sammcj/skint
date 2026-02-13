package tui

import (
	"fmt"
	"strings"

	"github.com/sammcj/skint/internal/config"
)

func (m *Model) viewMainScreen() string {
	var b strings.Builder

	// Header
	header := m.styles.Title.Render("  Skint  ")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Subtitle
	configuredCount := 0
	for _, p := range m.cfg.Providers {
		if !p.NeedsAPIKey() || p.GetAPIKey() != "" {
			configuredCount++
		}
	}
	subtitle := m.styles.Subtitle.Render(fmt.Sprintf("Configure Providers (%d configured)", configuredCount))
	b.WriteString(subtitle)
	b.WriteString("\n")

	// List
	b.WriteString(m.styles.List.Render(m.list.View()))
	b.WriteString("\n")

	// Custom providers section
	customSection := m.styles.Box.Width(m.width - 8).Render(
		m.styles.BoxTitle.Render("Add Custom Provider") + "\n" +
			m.styles.Dimmed.Render("Press ") +
			m.styles.Success.Render("c") +
			m.styles.Dimmed.Render(" to configure a custom API endpoint (OpenAI or Anthropic compatible)"),
	)
	b.WriteString(customSection)
	b.WriteString("\n")

	// Help
	help := m.styles.Help.Render("↑/k ↓/j • enter: use • e: edit • a/c: add custom • t: test • q: quit")
	b.WriteString(m.styles.Footer.Render(help))

	return b.String()
}

func (m *Model) viewProviderConfig() string {
	var b strings.Builder

	header := m.styles.Title.Render(fmt.Sprintf("  %s  ", m.selectedProvider.DisplayName))
	b.WriteString(header)
	b.WriteString("\n\n")

	// Show provider info
	info := m.styles.Box.Width(m.width - 8).Render(
		m.styles.BoxTitle.Render("Setup Instructions") + "\n" +
			m.getLocalProviderInstructions(),
	)
	b.WriteString(info)
	b.WriteString("\n\n")

	// Status
	status := m.styles.Success.Render("✓ Ready to use")
	b.WriteString(status)
	b.WriteString("\n\n")

	// Help
	help := m.styles.Help.Render("enter: confirm • esc: back")
	b.WriteString(m.styles.Footer.Render(help))

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

	header := m.styles.Title.Render(fmt.Sprintf("  Configure %s  ", m.selectedProvider.DisplayName))
	b.WriteString(header)
	b.WriteString("\n\n")

	// Provider info
	info := m.styles.Box.Width(m.width - 8).Render(
		m.styles.Label.Render("Provider: ") + m.selectedProvider.DisplayName + "\n" +
			m.styles.Label.Render("Endpoint: ") + m.styles.Info.Render(m.selectedProvider.BaseURL),
	)
	b.WriteString(info)
	b.WriteString("\n\n")

	// API Key input
	b.WriteString(m.styles.Label.Render("API Key:"))
	b.WriteString("\n")

	// Show masked input
	masked := strings.Repeat("•", len(m.apiKeyInput))
	if masked == "" {
		masked = m.styles.Dimmed.Render("Type your API key...")
	}
	input := m.styles.Input.Width(m.width - 8).Render(masked)
	b.WriteString(input)
	b.WriteString("\n")

	// Error message
	if m.inputError != "" {
		b.WriteString(m.styles.Error.Render("✗ " + m.inputError))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Help
	help := m.styles.Help.Render("type: enter key • enter: confirm • esc: cancel")
	b.WriteString(m.styles.Footer.Render(help))

	return b.String()
}

func (m *Model) viewSuccess() string {
	var b strings.Builder

	header := m.styles.Title.Render("  Success!  ")
	b.WriteString(header)
	b.WriteString("\n\n")

	b.WriteString(m.styles.Success.Render(m.message))
	b.WriteString("\n\n")

	// Next steps
	providerName := ""
	if m.selectedProvider != nil {
		providerName = m.selectedProvider.Name
	} else if m.customProviderName != "" {
		// Custom provider was just created
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
	}

	// Help - different message based on whether we're quitting or returning
	helpText := "press any key to continue..."
	if m.done {
		helpText = "press any key to exit..."
	}
	help := m.styles.Help.Render(helpText)
	b.WriteString(m.styles.Footer.Render(help))

	return b.String()
}

func (m *Model) viewError() string {
	var b strings.Builder

	header := m.styles.Title.Render("  Error  ")
	b.WriteString(header)
	b.WriteString("\n\n")

	b.WriteString(m.styles.Error.Render("✗ " + m.message))
	b.WriteString("\n\n")

	// Help
	help := m.styles.Help.Render("press any key to continue...")
	b.WriteString(m.styles.Footer.Render(help))

	return b.String()
}

func (m *Model) viewCustomProvider() string {
	var b strings.Builder

	// Check if editing or adding
	existingProvider := m.cfg.GetProvider(m.customProviderName)
	isEditing := existingProvider != nil

	headerText := "  Add Custom Provider  "
	if isEditing {
		headerText = "  Edit Custom Provider  "
	}
	header := m.styles.Title.Render(headerText)
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

	// Form fields with consistent layout
	inputWidth := m.width - 20
	if inputWidth < 30 {
		inputWidth = 30
	}

	// Check if provider has saved API key for hint text
	hasSavedKey := existingProvider != nil && existingProvider.APIKeyRef != ""

	apiKeyHint := "optional"
	if hasSavedKey {
		apiKeyHint = "(saved - type to change)"
	}

	fields := []struct {
		label   string
		value   string
		focus   int
		hint    string
		mask    bool
		req     bool
	}{
		{"Name", m.customProviderName, 0, "lowercase-id", false, true},
		{"Display Name", m.customProviderDisplay, 1, "optional", false, false},
		{"Base URL", m.customProviderURL, 2, "https://api.example.com", false, true},
		{"API Key", m.apiKeyInput, 3, apiKeyHint, true, false},
		{"Model", m.customProviderModel, 4, "e.g., gpt-4o, claude-3-sonnet", false, true},
		{"API Type", m.customProviderAPIType, 5, "↑/↓ to change", false, true},
	}

	for _, f := range fields {
		labelStyle := m.styles.Label
		if m.inputFocus == f.focus {
			labelStyle = m.styles.InputPrompt
		}

		// Required indicator
		reqIndicator := ""
		if f.req {
			reqIndicator = m.styles.Error.Render("*")
		}

		// Label line
		b.WriteString(labelStyle.Render(f.label) + reqIndicator)
		b.WriteString("\n")

		// Display value
		displayValue := f.value
		if f.mask && displayValue != "" {
			displayValue = strings.Repeat("•", len(displayValue))
		}
		if displayValue == "" {
			displayValue = f.hint
		}

		// Input line with consistent styling
		var inputLine string
		if m.inputFocus == f.focus {
			// Active field with border
			inputLine = m.styles.Input.Width(inputWidth).Render(displayValue)
		} else {
			// Inactive field with dimmed styling
			if f.value == "" {
				inputLine = m.styles.Dimmed.Render("  " + displayValue)
			} else {
				inputLine = m.styles.Value.Render("  " + displayValue)
			}
		}
		b.WriteString(inputLine)
		b.WriteString("\n\n")
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

	// Help
	help := m.styles.Help.Render("↑/↓/tab: navigate • enter: submit • esc: cancel")
	b.WriteString(m.styles.Footer.Render(help))

	return b.String()
}
