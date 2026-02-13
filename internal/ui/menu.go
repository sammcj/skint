package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/providers"
)

// ErrTestRequested is returned when the user selects the "test" option from the menu.
var ErrTestRequested = errors.New("test requested")

// MenuItem represents a menu item
type MenuItem struct {
	Key         string
	DisplayName string
	Category    string
	Handler     func() error
}

// ProviderMenu handles the provider configuration menu
type ProviderMenu struct {
	items []MenuItem
}

// NewProviderMenu creates a new provider menu
func NewProviderMenu(cfg *config.Config, registry *providers.Registry, form *ConfigForm) *ProviderMenu {
	m := &ProviderMenu{}

	// Native
	m.addItem("native", "Anthropic direct", "NATIVE", func() error {
		return form.ConfigureBuiltin(cfg, "native")
	})

	// China providers
	for _, def := range registry.GroupedList()["China"] {
		def := def // capture range variable
		m.addItem(def.Name, def.DisplayName, "CHINA", func() error {
			return form.ConfigureBuiltin(cfg, def.Name)
		})
	}

	// International providers
	for _, def := range registry.GroupedList()["International"] {
		def := def // capture range variable
		m.addItem(def.Name, def.DisplayName, "INTERNATIONAL", func() error {
			return form.ConfigureBuiltin(cfg, def.Name)
		})
	}

	// Local providers
	for _, def := range registry.GroupedList()["Local"] {
		def := def // capture range variable
		m.addItem(def.Name, def.DisplayName, "LOCAL", func() error {
			return form.configureLocal(cfg, def.Name)
		})
	}

	// Advanced options
	m.addItem("openrouter", "100+ models (native API)", "ADVANCED", func() error {
		return form.ConfigureOpenRouter(cfg)
	})
	m.addItem("custom", "Anthropic-compatible", "ADVANCED", func() error {
		return form.ConfigureCustom(cfg)
	})

	return m
}

func (m *ProviderMenu) addItem(key, displayName, category string, handler func() error) {
	m.items = append(m.items, MenuItem{
		Key:         key,
		DisplayName: displayName,
		Category:    category,
		Handler:     handler,
	})
}

// Display shows the menu and returns the selected handler
func (m *ProviderMenu) Display(cfg *config.Config) (func() error, error) {
	fmt.Println()
	Box("SKINT CONFIGURATION", 54)
	fmt.Println()

	// Count configured providers
	configured := 0
	for _, p := range cfg.Providers {
		if !p.NeedsAPIKey() || p.GetAPIKey() != "" {
			configured++
		}
	}
	Dim("%d providers configured\n\n", configured)

	// Display menu by category
	currentCategory := ""
	for i, item := range m.items {
		if item.Category != currentCategory {
			if currentCategory != "" {
				fmt.Println()
			}
			Log("%s", Bold(item.Category))
			currentCategory = item.Category
		}

		p := cfg.GetProvider(item.Key)
		isConfigured := p != nil && (!p.NeedsAPIKey() || p.GetAPIKey() != "")
		ListItem(isConfigured, "%-2d %-12s %-24s", i+1, item.Key, item.DisplayName)
	}

	fmt.Println()
	Separator(54)
	Dim("  [t] Test providers  [q] Quit\n")
	fmt.Println()

	// Get choice
	choice := Prompt("Choose", "q")

	// Handle special commands
	choice = strings.ToLower(strings.TrimSpace(choice))
	switch choice {
	case "q", "quit", "exit":
		return nil, nil
	case "t", "test":
		// Return a special handler for test
		return func() error { return nil }, ErrTestRequested
	}

	// Find handler by index
	for i, item := range m.items {
		if choice == fmt.Sprintf("%d", i+1) {
			return item.Handler, nil
		}
	}

	// Find handler by name
	for _, item := range m.items {
		if choice == item.Key {
			return item.Handler, nil
		}
	}

	return nil, fmt.Errorf("invalid choice: %s", choice)
}
