# Style and Conventions

## Code Style
- Standard Go conventions (gofmt, golangci-lint)
- Pointer receivers for methods that modify state
- Value receivers for read-only operations
- Idiomatic error wrapping with `fmt.Errorf("%w")`
- Early returns for error handling
- Table-driven tests

## Naming
- Provider types are constants in `config/schema.go`
- API key refs use format `keyring:<name>` or `file:<name>`
- Environment variable overrides use `SKINT_` prefix

## Config
- XDG-compliant directories
- Config: `~/.config/skint/config.yaml`
- Data: `~/.local/share/skint/`
- Secrets: `~/.local/share/skint/secrets.enc`

## Security
- Symlink checks before reading sensitive files
- File permissions 0600 for config and secrets
- No API keys in error messages
- Environment vars stripped before injecting provider-specific ones

## Output
- Banner output to stderr
- Three output formats: `human`, `json`, `plain`
- fatih/color for CLI output, lipgloss for TUI

## Australian English
- Use Australian English spelling in all documentation and comments
