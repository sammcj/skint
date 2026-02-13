# GitHub Copilot Instructions for Skint

## Project Overview

**Skint** is a CLI launcher that wraps Claude Code with different LLM provider configurations. It sets environment variables (`ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, model mappings, etc.) then exec's into the `claude` binary.

## Development Setup

```bash
make build      # Build binary
make test       # Run tests
make lint       # golangci-lint
make fmt        # Format code
make deps       # Download and tidy modules
```

## Project Structure

```
skint/
├── main.go                  # Entry point
├── internal/
│   ├── commands/            # Cobra command definitions
│   ├── config/              # YAML config loading/saving (XDG-compliant)
│   ├── providers/           # Provider interface and implementations
│   ├── launcher/            # Builds env vars and exec's claude
│   ├── secrets/             # OS keyring + encrypted file fallback
│   ├── tui/                 # Bubble Tea interactive UI
│   └── ui/                  # Simple CLI components
├── Makefile
└── go.mod
```

## Code Standards

- Follow Go best practices and idiomatic patterns
- Use Australian English spelling throughout code and documentation
- Keep functions under 50 lines, files under 700 lines
- Explicit error handling, early returns, small interfaces
- Table-driven tests, one assertion per test where practical

## Key Conventions

- Config version is `"1.0"` (string in YAML)
- Provider types: `builtin`, `openrouter`, `local`, `custom`
- API types for custom providers: `anthropic`, `openai`
- Output formats: `human`, `json`, `plain`
- Environment variable overrides use `SKINT_` prefix
- Banner output goes to stderr, not stdout
