package tui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sammcj/skint/internal/config"
	"github.com/sammcj/skint/internal/providers"
	"github.com/sammcj/skint/internal/secrets"
)

// Screen represents the current screen in the TUI
type Screen int

const (
	ScreenMain Screen = iota
	ScreenProviderConfig
	ScreenAPIKeyInput
	ScreenOpenRouter
	ScreenCustomProvider
	ScreenConfirm
	ScreenSuccess
	ScreenError
)

// customFormFieldCount is the number of fields in the custom provider form
const customFormFieldCount = 6

// localFormFieldCount is the number of fields in the local provider config form
const localFormFieldCount = 3

// apiKeyFormFieldCount is the number of fields in the API key form (API key + model)
const apiKeyFormFieldCount = 2

// Model is the main TUI model
type Model struct {
	// State
	screen      Screen
	styles      Styles
	width       int
	height      int
	compact     bool

	// Data
	cfg        *config.Config
	registry   *providers.Registry
	secretsMgr *secrets.Manager

	// Components
	list         list.Model
	providerList []ProviderItem

	// Form state
	selectedProvider *providers.Definition
	apiKeyInput      string
	modelInput       string
	inputFocus       int
	inputError       string
	hasExistingKey   bool

	// Custom provider form fields
	customProviderName     string
	customProviderDisplay  string
	customProviderURL      string
	customProviderModel    string
	customProviderAPIType  string // "anthropic" or "openai"

	// Local provider form fields
	localProviderURL       string
	localProviderAuthToken string
	localProviderModel     string

	// Results
	message      string
	messageType  string // "success", "error", "info"
	done         bool
	resultAction string

	// Callbacks
	onProviderSelect func(string) error
	onConfigDone     func() error
}

// ProviderItem represents an item in the provider list
type ProviderItem struct {
	definition *providers.Definition
	configured bool
	active     bool
	category   string
	isAddNew   bool
}

func (p ProviderItem) FilterValue() string {
	if p.isAddNew {
		return "add new custom provider"
	}
	return p.definition.Name + " " + p.definition.DisplayName
}

func (p ProviderItem) Title() string {
	if p.isAddNew {
		return "+ Add New Provider"
	}
	status := "○"
	if p.configured {
		status = "✓"
	}
	return fmt.Sprintf("%s %s", status, p.definition.DisplayName)
}

func (p ProviderItem) Description() string {
	if p.isAddNew {
		return "Configure a custom API endpoint (OpenAI or Anthropic compatible)"
	}
	return p.definition.Description
}

// itemDelegate is the list item delegate
type itemDelegate struct {
	styles Styles
}

func (d itemDelegate) Height() int                             { return 2 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(ProviderItem)
	if !ok {
		return
	}

	var title, desc lipgloss.Style
	isSelected := index == m.Index()

	switch {
	case item.active && isSelected:
		// Active + selected: combine both indicators
		title = d.styles.ListActive.Foreground(d.styles.PrimaryColor)
		desc = d.styles.Dimmed.PaddingLeft(4)
	case item.active:
		title = d.styles.ListActive
		desc = d.styles.Dimmed.PaddingLeft(4)
	case isSelected:
		title = d.styles.ListSelected
		desc = d.styles.Dimmed.PaddingLeft(4)
	default:
		title = d.styles.ListItem.Foreground(d.styles.Normal.GetForeground())
		desc = d.styles.Dimmed.PaddingLeft(4)
	}

	// Color the status indicators / add-new styling
	titleStr := item.Title()
	if item.isAddNew {
		titleStr = strings.Replace(titleStr, "+", d.styles.Info.Render("+"), 1)
	} else if item.configured {
		titleStr = strings.Replace(titleStr, "✓", d.styles.Success.Render("✓"), 1)
	} else {
		titleStr = strings.Replace(titleStr, "○", d.styles.Dimmed.Render("○"), 1)
	}

	fmt.Fprint(w, title.Render(titleStr)+"\n")
	fmt.Fprint(w, desc.Render(item.Description()))
}

// NewModel creates a new TUI model
func NewModel(cfg *config.Config, secretsMgr *secrets.Manager) *Model {
	registry := providers.NewRegistry()
	styles := DefaultStyles()

	// Build provider list
	var items []list.Item
	providerItems := []ProviderItem{}

	// Add providers by category
	grouped := registry.GroupedList()

	// Native (always configured - no setup required)
	if native, ok := grouped["Native"]; ok {
		for _, def := range native {
			item := ProviderItem{
				definition: def,
				configured: true,
				active:     cfg.DefaultProvider == def.Name || cfg.DefaultProvider == "",
				category:   "Native",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// International
	if intl, ok := grouped["International"]; ok {
		for _, def := range intl {
			p := cfg.GetProvider(def.Name)
			configured := p != nil && p.IsConfigured()
			item := ProviderItem{
				definition: def,
				configured: configured,
				active:     cfg.DefaultProvider == def.Name,
				category:   "International",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// China
	if china, ok := grouped["China"]; ok {
		for _, def := range china {
			p := cfg.GetProvider(def.Name)
			configured := p != nil && p.IsConfigured()
			item := ProviderItem{
				definition: def,
				configured: configured,
				active:     cfg.DefaultProvider == def.Name,
				category:   "China",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// Local
	if local, ok := grouped["Local"]; ok {
		for _, def := range local {
			p := cfg.GetProvider(def.Name)
			configured := p != nil
			item := ProviderItem{
				definition: def,
				configured: configured,
				active:     cfg.DefaultProvider == def.Name,
				category:   "Local",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// Add existing custom providers
	for _, p := range cfg.Providers {
		if p.Type == config.ProviderTypeCustom {
			// Create a definition for the custom provider
			def := &providers.Definition{
				Name:        p.Name,
				DisplayName: p.DisplayName,
				Description: fmt.Sprintf("Custom %s endpoint", p.APIType),
				Type:        p.Type,
				BaseURL:     p.BaseURL,
			}
			item := ProviderItem{
				definition: def,
				configured: true,
				active:     cfg.DefaultProvider == p.Name,
				category:   "Custom",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// Sort items: active first, then configured, then by category, then by name
	sort.Slice(items, func(i, j int) bool {
		itemI := items[i].(ProviderItem)
		itemJ := items[j].(ProviderItem)

		// Active provider comes first
		if itemI.active != itemJ.active {
			return itemI.active && !itemJ.active
		}
		// Configured providers come next
		if itemI.configured != itemJ.configured {
			return itemI.configured && !itemJ.configured
		}
		// Then sort by category priority
		categoryPriority := map[string]int{
			"Custom":        0,
			"Native":        1,
			"International": 2,
			"China":         3,
			"Local":         4,
		}
		pi := categoryPriority[itemI.category]
		pj := categoryPriority[itemJ.category]
		if pi != pj {
			return pi < pj
		}
		// Finally sort by name
		return itemI.definition.Name < itemJ.definition.Name
	})

	// Add "Add New Provider" item at the end
	addNewItem := ProviderItem{isAddNew: true}
	items = append(items, addNewItem)
	providerItems = append(providerItems, addNewItem)

	// Create list
	delegate := itemDelegate{styles: styles}
	l := list.New(items, delegate, 0, 0)
	l.Title = "Select a Provider"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = styles.Title
	l.KeyMap = list.KeyMap{
		CursorUp:             key.NewBinding(key.WithKeys("up", "k")),
		CursorDown:           key.NewBinding(key.WithKeys("down", "j")),
		GoToStart:            key.NewBinding(key.WithKeys("home", "g")),
		GoToEnd:              key.NewBinding(key.WithKeys("end", "G")),
		Filter:               key.NewBinding(key.WithKeys("/")),
		ClearFilter:          key.NewBinding(key.WithKeys("esc")),
		CancelWhileFiltering: key.NewBinding(key.WithKeys("esc")),
		AcceptWhileFiltering: key.NewBinding(key.WithKeys("enter", "tab")),
		ShowFullHelp:         key.NewBinding(key.WithKeys("?")),
		CloseFullHelp:        key.NewBinding(key.WithKeys("?")),
		Quit:                 key.NewBinding(key.WithKeys("q", "esc")),
		ForceQuit:            key.NewBinding(key.WithKeys("ctrl+c")),
	}

	return &Model{
		screen:           ScreenMain,
		styles:           styles,
		cfg:              cfg,
		registry:         registry,
		secretsMgr:       secretsMgr,
		list:             l,
		providerList:     providerItems,
	}
}

// SetCompact enables compact mode for smaller terminals
func (m *Model) SetCompact(compact bool) {
	m.compact = compact
	if compact {
		m.styles = CompactStyles()
	}
}

// SetOnProviderSelect sets the callback for provider selection
func (m *Model) SetOnProviderSelect(fn func(string) error) {
	m.onProviderSelect = fn
}

// SetOnConfigDone sets the callback for when config is done
func (m *Model) SetOnConfigDone(fn func() error) {
	m.onConfigDone = fn
}

// refreshProviderList rebuilds the list items from current config state
func (m *Model) refreshProviderList() {
	var items []list.Item
	providerItems := []ProviderItem{}
	grouped := m.registry.GroupedList()

	// Native (always configured)
	if native, ok := grouped["Native"]; ok {
		for _, def := range native {
			item := ProviderItem{
				definition: def,
				configured: true,
				active:     m.cfg.DefaultProvider == def.Name || m.cfg.DefaultProvider == "",
				category:   "Native",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// International
	if intl, ok := grouped["International"]; ok {
		for _, def := range intl {
			p := m.cfg.GetProvider(def.Name)
			configured := p != nil && p.IsConfigured()
			item := ProviderItem{
				definition: def,
				configured: configured,
				active:     m.cfg.DefaultProvider == def.Name,
				category:   "International",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// China
	if china, ok := grouped["China"]; ok {
		for _, def := range china {
			p := m.cfg.GetProvider(def.Name)
			configured := p != nil && p.IsConfigured()
			item := ProviderItem{
				definition: def,
				configured: configured,
				active:     m.cfg.DefaultProvider == def.Name,
				category:   "China",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// Local
	if local, ok := grouped["Local"]; ok {
		for _, def := range local {
			p := m.cfg.GetProvider(def.Name)
			configured := p != nil
			item := ProviderItem{
				definition: def,
				configured: configured,
				active:     m.cfg.DefaultProvider == def.Name,
				category:   "Local",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// Custom providers
	for _, p := range m.cfg.Providers {
		if p.Type == config.ProviderTypeCustom {
			def := &providers.Definition{
				Name:        p.Name,
				DisplayName: p.DisplayName,
				Description: fmt.Sprintf("Custom %s endpoint", p.APIType),
				Type:        p.Type,
				BaseURL:     p.BaseURL,
			}
			item := ProviderItem{
				definition: def,
				configured: true,
				active:     m.cfg.DefaultProvider == p.Name,
				category:   "Custom",
			}
			items = append(items, item)
			providerItems = append(providerItems, item)
		}
	}

	// Sort
	sort.Slice(items, func(i, j int) bool {
		itemI := items[i].(ProviderItem)
		itemJ := items[j].(ProviderItem)
		if itemI.active != itemJ.active {
			return itemI.active && !itemJ.active
		}
		if itemI.configured != itemJ.configured {
			return itemI.configured && !itemJ.configured
		}
		categoryPriority := map[string]int{
			"Custom": 0, "Native": 1, "International": 2, "China": 3, "Local": 4,
		}
		pi := categoryPriority[itemI.category]
		pj := categoryPriority[itemJ.category]
		if pi != pj {
			return pi < pj
		}
		return itemI.definition.Name < itemJ.definition.Name
	})

	// Add "Add New Provider" at the end
	addNewItem := ProviderItem{isAddNew: true}
	items = append(items, addNewItem)
	providerItems = append(providerItems, addNewItem)

	m.list.SetItems(items)
	m.providerList = providerItems
}

// Init initialises the model
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Adjust list size
		listWidth := msg.Width - 4
		listHeight := msg.Height - 8
		if listWidth < 20 {
			listWidth = 20
		}
		if listHeight < 10 {
			listHeight = 10
		}
		m.list.SetSize(listWidth, listHeight)

		// Switch to compact mode for small terminals
		if msg.Height < 24 {
			m.SetCompact(true)
		}

	case tea.KeyMsg:
		switch m.screen {
		case ScreenMain:
			return m.updateMainScreen(msg)
		case ScreenProviderConfig:
			return m.updateProviderConfig(msg)
		case ScreenAPIKeyInput:
			return m.updateAPIKeyInput(msg)
		case ScreenCustomProvider:
			return m.updateCustomProvider(msg)
		case ScreenSuccess, ScreenError:
			// Any key returns to main screen (or quits if done)
			if m.screen == ScreenSuccess && m.done {
				return m, tea.Quit
			}
			m.refreshProviderList()
			m.screen = ScreenMain
			return m, nil
		}
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the UI
func (m *Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var content string

	switch m.screen {
	case ScreenMain:
		content = m.viewMainScreen()
	case ScreenProviderConfig:
		content = m.viewProviderConfig()
	case ScreenAPIKeyInput:
		content = m.viewAPIKeyInput()
	case ScreenCustomProvider:
		content = m.viewCustomProvider()
	case ScreenSuccess:
		content = m.viewSuccess()
	case ScreenError:
		content = m.viewError()
	default:
		content = m.viewMainScreen()
	}

	return m.styles.App.Render(content)
}

// IsDone returns true if the TUI is done
func (m *Model) IsDone() bool {
	return m.done
}

// GetResultAction returns the result action
func (m *Model) GetResultAction() string {
	return m.resultAction
}

// GetSelectedProvider returns the selected provider name
func (m *Model) GetSelectedProvider() string {
	if m.selectedProvider != nil {
		return m.selectedProvider.Name
	}
	return ""
}
