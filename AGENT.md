# quad-ops Development Guide

## Commands
- **Build**: `go build -o quad-ops cmd/quad-ops/main.go`
- **Test all**: `gotestsum --format pkgname --format-icons text -- -coverprofile=coverage.out -v ./...`
- **Test single package**: `go test -v ./internal/validate`
- **Lint**: `golangci-lint run`
- **Format**: `go fmt ./...`
- **Main entry**: `cmd/quad-ops/main.go`

## Code Style
- **Imports**: Use `goimports` formatting, group std → external → internal
- **Naming**: Use Go conventions - PascalCase for exported, camelCase for unexported
- **Error handling**: Always check errors, use descriptive error messages like `fmt.Errorf("command not mocked: %s", key)`
- **Types**: Use Go's built-in types, explicit interfaces (Repository, Provider patterns)
- **Comments**: Package comments start with package name, function comments for exported items
- **Testing**: Use testify/assert, create temp files with `os.CreateTemp()`, defer cleanup
- **Structure**: Organize by domain packages in `internal/` - config, db, git, unit, validate, etc.
- **CLI**: Use cobra/viper pattern for commands and configuration
- **License**: Include MIT license header in new files