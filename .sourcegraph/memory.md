# Quad-Ops Development Memory

## Project Overview
- Quad-Ops manages Podman containers through Quadlet by synchronizing from Git repositories
- Supports standard Docker Compose files (version 3.x)
- Creates systemd unit files for containers, volumes, and networks

## Documentation
- [podman-systemd (quadlet)](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
- [systemd.unit](https://www.freedesktop.org/software/systemd/man/systemd.unit.html)
- [compose-go-v2](https://pkg.go.dev/github.com/compose-spec/compose-go/v2)
- [compose-spec](https://github.com/compose-spec/compose-spec)

## Podman Quadlet Best Practices
- Always use fully qualified image names with registry prefix (docker.io/, quay.io/, etc.)
- Container dependencies must be expressed in systemd unit files using the service name format
- Use After/Requires with .service suffix (e.g., 'After=db.service', not 'After=db.container')
- For container DNS resolution, always use systemd- prefix (e.g., 'systemd-project-service')
- Avoid unsupported keys: DNSEnabled (network), Privileged and SecurityLabel (container)
- Named volumes require the '.volume' suffix in Volume= directives (e.g., 'Volume=data.volume:/data')
- Quadlet does not auto-create bind mount directories - they must exist before container start

## Key Components
- `internal/compose/`: Handles Docker Compose file reading and parsing
- `internal/unit/`: Converts Docker Compose objects to Podman Quadlet units
- `internal/git/`: Manages Git repository operations
- `internal/config/`: Handles application configuration
- `cmd/`: Contains CLI commands and entry points

## Important Bug Fixes
- Always initialize RunInit field in container.go to prevent nil pointer dereference
- Use proper project naming format for Docker Compose projects
- Handle nil networks when alias is not present in container.go
- Fix unsupported quadlet keys in unit files (removed DNSEnabled, SecurityLabel, Privileged)
- Ensure fully qualified image names (docker.io/ prefix) to prevent quadlet warnings
- Fixed container name resolution for inter-container communication

## Docker Compose Support
- `compose/reader.go`: Detects and reads Docker Compose files with robust error handling
- `unit/container.go`: Converts services to container units with `FromComposeService()`
- `unit/volume.go`: Converts volumes with `FromComposeVolume()`
- `unit/network.go`: Converts networks with `FromComposeNetwork()`
- `unit/compose_processor.go`: Orchestrates conversion with `ProcessComposeProjects()`
- Project naming format: `<repo>-<folder>` (e.g., `test-photoprism` for repositories/home/test/photoprism)
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
