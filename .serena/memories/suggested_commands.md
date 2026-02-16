# Suggested Commands

## Build & Development
```bash
make build          # Build binary (version from git tag, ldflags injection)
make test           # Run all tests: go test -v ./...
make lint           # Run golangci-lint
make fmt            # Format code
make install        # Build and install to ~/.local/bin
make deps           # Download and tidy modules
```

## Single Test
```bash
go test -v -run TestName ./internal/package/...
```

## Go Modernity Check
```bash
go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...
```

## System Utilities (macOS/Darwin)
```bash
git status          # Check git status
git diff            # View changes
ls -la              # List files
grep -rn "pattern" ./internal/  # Search code
```
