# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development

```bash
make build          # Build binary (version from git tag, ldflags injection)
make test           # Run all tests: go test -v ./...
make lint           # Run golangci-lint
make fmt            # Format code
make install        # Build and install to ~/.local/bin
make deps           # Download and tidy modules
```

Run a single test: `go test -v -run TestName ./internal/package/...`

<ARCHITECTURE>
Skint is a CLI launcher that wraps Claude Code with different LLM provider configurations. It sets environment variables (ANTHROPIC_BASE_URL, ANTHROPIC_AUTH_TOKEN, model mappings, etc.) then exec's into the `claude` binary.

**Package dependency flow:**
`cmd/skint/main.go` -> `internal/commands` -> `internal/{config,providers,launcher,secrets,tui,ui}`

**Key packages:**
- `commands/` - Cobra command definitions. Global state (`configMgr`, `secretsMgr`, `cfg`) lives in `root.go` and is initialised via `PersistentPreRunE`. All subcommands register in `main.go`.
- `config/` - YAML config loading/saving (XDG-compliant: `~/.config/skint/config.yaml`). `schema.go` defines `Config` and `Provider` structs. `config.go` has the `Manager`. `migrate.go` imports from the old bash version.
- `providers/` - `Provider` interface with four implementations: `BuiltinProvider`, `OpenRouterProvider`, `LocalProvider`, `CustomProvider`. All embed `baseProvider`. Registry of 10 built-in providers defined as data. `baseProvider.keyEnvVar` overrides the default env var name for the API key (used by the `anthropic` provider to set `ANTHROPIC_API_KEY` instead of `ANTHROPIC_AUTH_TOKEN`).
- `models/` - Model fetching from provider APIs. Strategies: OpenAI-compatible (`/v1/models`), Ollama (`/api/tags`), OpenRouter (public listing). Used by the TUI model picker.
- `launcher/` - Builds env vars from a `Provider`, strips conflicting ANTHROPIC_*/OPENAI_* vars from the current env, then uses `syscall.Exec` on Unix (process replacement for signal forwarding) or `exec.Command` on Windows.
- `secrets/` - Two-tier credential storage: OS keyring (primary) with AES-256-GCM encrypted file fallback (`~/.local/share/skint/secrets.enc`). API key refs use format `keyring:<name>` or `file:<name>`.
- `tui/` - Bubble Tea interactive UI. `model.go` is the main state machine. `modelpicker.go` handles async model fetching and picker overlay state. Handles provider selection, API key input, custom provider config.
- `ui/` - Simple non-interactive CLI components (colours, menus, prompts).

**Provider -> Environment Variable mapping** is the core logic. Each provider type generates different env vars via `GetEnvVars()`:
- Builtin: `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN` (or `ANTHROPIC_API_KEY` via `keyEnvVar`), model tier mappings
- OpenRouter: Same as builtin but routes through `openrouter.ai/api`, explicitly empties `ANTHROPIC_API_KEY`
- Local: `ANTHROPIC_BASE_URL` with optional auth, no API key required
- Custom: Either Anthropic-compatible or OpenAI-compatible (`OPENAI_BASE_URL`, `OPENAI_API_KEY`, `OPENAI_MODEL`)
</ARCHITECTURE>

<CONVENTIONS>
- Config version is `"1.0"` (string in YAML), provider types are constants in `config/schema.go`
- Provider types: `builtin`, `openrouter`, `local`, `custom`. API types for custom: `anthropic`, `openai`
- Output formats: `human`, `json`, `plain` - all commands should respect `outputFormat` global flag
- Environment variable overrides use `SKINT_` prefix (e.g. `SKINT_DEFAULT_PROVIDER`, `SKINT_VERBOSE`)
- Banner output goes to stderr, not stdout
- Running with no subcommand launches the interactive TUI; pressing 'u' or quitting with a provider set will launch claude
- `skint env` prints shell export statements for the active provider (for use with `eval "$(skint env)"` in shell profiles)
- `config.ClaudeArgs` (YAML: `claude_args`) holds default arguments passed to claude on launch (e.g. `["--continue"]`)
- `config.Provider.IsConfigured()` checks `APIKeyRef` (persisted) rather than `resolvedAPIKey` (runtime-only) - always prefer this over checking `GetAPIKey()`
- Provider categories in TUI: Native (`native`, `anthropic`), International, Local. No China category.
- The `anthropic` provider uses `KeyEnvVar: "ANTHROPIC_API_KEY"` and has no base URL (Claude Code defaults to api.anthropic.com)
</CONVENTIONS>

<GOTCHAS>
- `launcher.go` uses `syscall.Exec` on Unix which replaces the process entirely - code after the exec call never runs
- The `removeEnvVars` function in launcher parses env strings manually (splitting on first `=`) - entries without `=` are silently dropped
- `config.Provider` has both `APIKey` (for migration only, stored in YAML) and `resolvedAPIKey` (unexported, loaded at runtime from keyring/file) - always use `GetAPIKey()`/`SetResolvedAPIKey()`, never read `APIKey` directly
- No automated tests exist yet for TUI package
- `LaunchNative` in launcher exec's claude without any env var overrides; `Launch` sets provider-specific env vars
</GOTCHAS>
