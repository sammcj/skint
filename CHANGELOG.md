# Changelog

<!-- Add changes following the format below - keep them concise and leave this comment as-is, use date +'%F %H:%M' for the date and local time  -->

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
