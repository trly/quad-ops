# Quad-Ops Development Memory

## Build & Test Commands
- Build: `go build -o quad-ops main.go`
- Run tests: `go test -v ./...`
- Run single test: `go test -v github.com/trly/quad-ops/internal/unit -run TestFromComposeService`
- Lint: `golangci-lint run`

## Code Style
- Use gofmt for formatting
- Import order: stdlib, external, internal
- Error handling: Always check errors, use meaningful error messages
- Return early pattern for error handling
- Use pointers for methods that modify the receiver
- Variable naming: camelCase, descriptive names
- Tests use testify/assert package
- Test functions prefixed with "Test"
- Type definitions before function definitions
- Initialize maps and slices properly before use