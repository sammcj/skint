# Changelog

<!-- Add changes following the format below - keep them concise and leave this comment as-is, use date +'%F %H:%M' for the date and local time  -->

## 2026-07-06 17:05

### Fixed

- **Model selection**: builtin and local providers now export `ANTHROPIC_MODEL` from the user-selected `Model` field (previously only `DefaultModel` was used, so the `anthropic` builtin and local providers silently dropped the chosen model). `EffectiveModel()` now prefers `Model` over `DefaultModel`, and `FromConfig` applies this uniformly
- **TUI**: async model-fetch results no longer hijack keystrokes. Fetches are tagged with a generation counter; stale results (fetch superseded, picker reset, or focus moved off the model field) are discarded, so a late fetch can no longer open the picker while the user types their API key
- **TUI**: entering the custom provider flow now clears any stale `selectedProvider`, so the success screen launches the provider just configured rather than a previously selected one
- `skint use <provider> --flag` no longer errors with "unknown flag"; the command now passes flags through to claude (`DisableFlagParsing`), matching `skint exec`
- **Security**: generated provider scripts are written `0700` (owner-only) instead of `0755`, as they embed plaintext API keys; `generate-scripts` now warns that keys are embedded
- **Critical**: removed `generateMainScript`, which wrote a `skint` bash wrapper into the bin dir that could overwrite the installed Go binary and exec itself in an infinite loop
- **Config**: `SKINT_*` / `NO_COLOR` environment overrides are no longer written to `config.yaml` on save (reverted to persisted values before marshalling); deliberate runtime changes (e.g. selecting a new default provider in the TUI) still persist even when the same field is env-overridden
- **Config**: `SKINT_DEFAULT_PROVIDER` naming an unknown provider is now non-fatal, warns to stderr and falls back to the persisted default instead of breaking every command
- **Config**: `Save()` is now atomic (temp file + `fsync` + rename), so a crash mid-write can't destroy the config

### Added

- TUI test suite (`internal/tui/tui_test.go`) covering the fetch-generation and stale-selection fixes

### Fixed
- `skint use` now passes `--resume` and `--continue` flags through to claude (previously only the TUI path did)
- README incorrectly listed `-c` as shorthand for both `--config` and `--continue`; `--config` has no shorthand
- Reset `ClaudeExtraArgs` before appending to prevent potential accumulation

## 2026-03-03 22:07

### Added
- `--resume <session-id>` and `--continue` / `-c` flags to pass through to claude for session resumption
## 2026-03-26

### Fixed

- **Security**: Shell injection in `env` command via unescaped single quotes in export values
- **Security**: `config.Save()` now checks for symlinks before writing (matching `Load()` behaviour)
- **Security**: `GenerateScript` now properly shell-escapes display names and env var values
- **Critical**: `ScreenOpenRouter` TUI dead-end state - pressing 'o' now correctly navigates to OpenRouter config
- Selecting "native" provider in TUI caused validation error on save because `Validate()` required it to be in the providers list
- Non-deterministic `unescape()` in migration due to map iteration order - now uses ordered slice
- OpenRouter `FromConfig` unconditionally blanking model when only `DefaultModel` was set
- `BuiltinProvider.GetEnvVars` now clears the conflicting API key env var (AUTH_TOKEN vs API_KEY)
- `LocalProvider.GetEnvVars` now always clears API key vars to prevent env leakage via `skint env`
- All `ui/` output functions (`Success`, `Info`, `Log`, `Dim`), `components.go`, and `menu.go` now write to stderr, preventing contamination of `eval "$(skint env)"` stdout
- `ErrorWithContext` no longer mixes stdout and stderr
- `Box` right-border ANSI misalignment when colours are enabled
- `exec` command now propagates the child process exit code instead of always exiting 1
- Legacy plaintext `APIKey` field cleared from config on load when `APIKeyRef` exists
- `fetchOpenRouter` now respects the `baseURL` parameter instead of always hitting the public endpoint

### Removed

- Dead `ScreenOpenRouter` and `ScreenConfirm` TUI screen constants
- Non-existent "China" provider category reference in `ui/menu.go`

### Added

- `env` command now respects `--output json` and `--output plain` flags (shell comment header only emitted in default shell mode)

## 2026-02-16 18:00

### Improved

- Deduplicated env var filtering logic: extracted shared `FilterEnvVars()` and `ConflictingEnvVars` into `internal/launcher/env.go`, removed duplicate in `commands/exec.go`
- Replaced `go-homedir` dependency with stdlib `os.UserHomeDir()` (available since Go 1.12)
- Made provider `NewRegistry()` a `sync.Once` singleton to avoid redundant re-registration
- Replaced `sort.Slice` with `slices.SortFunc` + `cmp.Compare` in model sorting
- Initialised `ui.Colors` and `ui.Sym` with safe defaults at declaration to prevent nil-pointer panics if `Init()` hasn't run

### Fixed

- `ui.Success()` used builtin `println()` (writes to stderr) instead of `fmt.Println()` (stdout)

### Added

- Tests for `ui.MaskKey()` (`internal/ui/components_test.go`)
- Tests for config `Manager`: Load, Save, round-trip, env overrides, XDG paths (`internal/config/manager_test.go`)
- Tests for config migration: `unescape`, `LoadSecrets`, `HasOldInstallation`, `Import`, `Cleanup` (`internal/config/migrate_test.go`)
