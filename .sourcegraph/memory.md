# Quad-Ops Development Memory

## Project Overview
- Quad-Ops manages Podman containers through Quadlet by synchronizing from Git repositories
- Supports standard Docker Compose files (version 3.x)
- Creates systemd unit files for containers, volumes, and networks

## Key Components
- `internal/compose/`: Handles Docker Compose file reading and parsing
- `internal/unit/`: Converts Docker Compose objects to Podman Quadlet units
- `internal/git/`: Manages Git repository operations
- `internal/config/`: Handles application configuration
- `cmd/`: Contains CLI commands and entry points

## Docker Compose Support
- `compose/reader.go`: Detects and reads Docker Compose files with robust error handling
- `unit/container.go`: Converts services to container units with `FromComposeService()`
- `unit/volume.go`: Converts volumes with `FromComposeVolume()`
- `unit/network.go`: Converts networks with `FromComposeNetwork()`
- `unit/compose_processor.go`: Orchestrates conversion with `ProcessComposeProjects()`
- Supported file names: `docker-compose.yml`, `docker-compose.yaml`, `compose.yml`, `compose.yaml`

## Configuration
- Repository settings are defined in `config.yaml`
- Each repository must have a name and URL
- Optional settings include: `ref` (branch/tag), `composeDir` (subdirectory for Docker Compose files), and `cleanup` policy
- Cleanup policy: "keep" (default) or "delete" for auto-removal of units from deleted compose files

## Build & Test Commands
- Build: `go build -o quad-ops cmd/quad-ops/main.go`
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