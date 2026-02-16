# Skint - Project Overview

## Purpose
Skint is a CLI launcher that wraps Claude Code with different LLM provider configurations. It sets environment variables (ANTHROPIC_BASE_URL, ANTHROPIC_AUTH_TOKEN, model mappings, etc.) then exec's into the `claude` binary.

## Tech Stack
- **Language**: Go 1.26+
- **CLI Framework**: spf13/cobra
- **TUI**: charmbracelet/bubbletea + lipgloss
- **CLI Output**: fatih/color
- **Config**: gopkg.in/yaml.v3, XDG-compliant paths
- **Secrets**: OS keyring (zalando/go-keyring) + AES-256-GCM encrypted file fallback
- **Build**: Makefile with ldflags version injection

## Package Structure
```
main.go                     Entry point
internal/commands/          Cobra command definitions (root.go has global state)
internal/config/            YAML config loading/saving, schema, migration
internal/providers/         Provider interface with 4 implementations
internal/models/            Model fetching from provider APIs
internal/launcher/          Builds env vars, exec's claude
internal/secrets/           Two-tier credential storage
internal/tui/               Bubble Tea interactive UI
internal/ui/                Simple non-interactive CLI components
```

## Key Patterns
- Provider types: `builtin`, `openrouter`, `local`, `custom`
- API types for custom: `anthropic`, `openai`
- Output formats: `human`, `json`, `plain`
- Env var overrides use `SKINT_` prefix
- Config version is `"1.0"` (string in YAML)
- `syscall.Exec` on Unix replaces process entirely
