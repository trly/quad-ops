# Quad-Ops Development Memory

## Build & Test Commands
- Build: `task build`
- Run tests: `task test` (uses gotestsum under the hood)
- Run single test: `go test -v github.com/trly/quad-ops/internal/unit -run TestFromComposeService`
- Lint: `task lint` (runs golangci-lint)
- Format: `task fmt` (runs go fmt)

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
- Volume and network dependencies must use the -volume.service and -network.service suffix format (e.g., 'After=data-volume.service', not 'After=data.volume')
- By default, quad-ops creates containers with hostnames that match their service names (without the systemd- prefix)
- Container hostnames can be configured via `usePodmanDefaultNames` option in config.yaml (default: false)
- Setting `usePodmanDefaultNames: true` allows Podman to use its default naming scheme with systemd- prefix
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
- Fix unsupported quadlet keys in unit files (removed DNSEnabled, SecurityLabel, Privileged, CPUPeriod, CPUShares, CPUQuota, Memory, MemoryReservation, MemorySwap)
- Ensure fully qualified image names (docker.io/ prefix) to prevent quadlet warnings
- Fixed container name resolution for inter-container communication
- Fixed service dependency configuration for containers with custom naming
- Added NetworkAlias support to allow referring to services by their simple names (e.g., "db" instead of full hostname)

## Docker Compose Support
- `compose/reader.go`: Detects and reads Docker Compose files with robust error handling
- `unit/container.go`: Converts services to container units with `FromComposeService()`
- `unit/volume.go`: Converts volumes with `FromComposeVolume()`
- `unit/network.go`: Converts networks with `FromComposeNetwork()`
- `unit/compose_processor.go`: Orchestrates conversion with `ProcessComposeProjects()`
- `unit/dependency.go`: Manages bidirectional dependency relationships between services
- `unit/restart.go`: Implements dependency-aware service restart logic
- Project naming format: `<repo>-<folder>` (e.g., `test-photoprism` for repositories/home/test/photoprism)
- Supported file names: `docker-compose.yml`, `docker-compose.yaml`, `compose.yml`, `compose.yaml`

### Resource Constraints Support
- Comprehensive mapping of Docker Compose resource constraints to Podman Quadlet
- Support for both service-level fields and deploy section resource limits
- Following compose-go validation, uses either service-level OR deploy section (not both)
- For service-level constraints:
  - Memory: mem_limit, memory_reservation, memswap_limit
  - CPU: cpus, cpu_quota, cpu_shares (note: cpu_period is read but not used in quadlet files)
  - Process: pids_limit
- For deploy section constraints:
  - Memory: deploy.resources.limits.memory, deploy.resources.reservations.memory
  - CPU: deploy.resources.limits.cpus (converts to quota and shares)
- Intelligent conversion for CPU values - NanoCPUs to quota/shares, CPUS float to quota
- Important note: Memory and CPU constraints are tracked internally but not included in generated quadlet files as they're unsupported by Podman Quadlet
- Warning messages are generated for unsupported features defined in Docker Compose files

### Health Check Support
- Docker Compose health checks are mapped to Podman Quadlet health check directives
- The mapping follows this pattern: (compose → quadlet)
  - `healthcheck.test` → `HealthCmd`
  - `healthcheck.interval` → `HealthInterval`
  - `healthcheck.timeout` → `HealthTimeout`
  - `healthcheck.retries` → `HealthRetries`
  - `healthcheck.start_period` → `HealthStartPeriod`
  - `healthcheck.start_interval` → `HealthStartupInterval`
- When the `disable: true` flag is set in the health check config, no health check directives are generated

### Service-Specific Environment Files
- Automatically detects and uses service-specific environment files present in the compose directory
- Supported file patterns for a service named `service1`:
  - `.env.service1` - Hidden env file with service name suffix
  - `service1.env` - Service name with .env extension
  - `env/service1.env` - In env subdirectory
  - `envs/service1.env` - In envs subdirectory
- Environment files are automatically added to the Quadlet unit with `EnvironmentFile` directive
- Helps avoid issues with special characters (asterisks, spaces, etc.) in environment values

### Unit Naming Conventions
- Containers: `<project-name>-<service-name>.container`
- Volumes: `<project-name>-<volume-name>.volume`
- Networks: `<project-name>-<network-name>.network`
- Service file naming in systemd:
  - Container services: `<project-name>-<service-name>.service`
  - Volume services: `<project-name>-<volume-name>-volume.service`
  - Network services: `<project-name>-<network-name>-network.service`

#### Docker Compose Name Mapping
- Project name becomes a prefix for all units to maintain uniqueness
- Service names from compose files become part of container names
- Network aliases are automatically created to allow containers to reference each other by their simple Docker Compose service names
- Dependency relationships (depends_on) are converted to systemd unit dependencies
- This maintains Docker Compose's naming simplicity while conforming to Quadlet's systemd integration requirements

### Dependency Management
- Docker Compose `depends_on` relationships are converted to systemd's `After` and `Requires` directives
- Reverse dependencies are tracked and converted to `PartOf` relationships for proper restart propagation
- The dependency-aware restart logic only restarts the most foundational service when multiple dependent services change
- File content change detection ensures only services with actual changes are restarted

## Configuration
- Repository settings are defined in `config.yaml`
- Each repository must have a name and URL
- Optional settings include: `ref` (branch/tag), `composeDir` (subdirectory for Docker Compose files), `cleanup` policy, and `usePodmanDefaultNames`
- Cleanup policy: "keep" (default) or "delete" for auto-removal of units from deleted compose files
- `usePodmanDefaultNames`: Controls container hostname prefix (default: false). When false, container hostnames match service names without systemd- prefix

## Manual Build & Test Commands
- Build: `go build -o quad-ops cmd/quad-ops/main.go`
- Run tests: `go test -v ./...`
- Run single test: `go test -v github.com/trly/quad-ops/internal/unit -run TestFromComposeService`
- Lint: `mise exec -- golangci-lint run`
- Format: `go fmt ./...`

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

## Linting
- Always run `mise exec -- golangci-lint run` before committing to ensure code passes lint checks
- All code must pass linting before being merged
- Address all lint issues, especially those from the following linters:
  - errcheck: Always check error return values, use `if err := foo(); err != nil` or use `_ = foo()` to explicitly ignore
  - godot: Comment sentences must end with a period
  - gofmt: Use proper Go formatting (handled by `gofmt` formatter)
  - gosec: Address all security concerns
- When ignoring errors in deferred functions, use `defer func() { _ = file.Close() }()` pattern
- Common linting errors to avoid:
  - Unchecked errors in `defer` statements
  - Missing periods at the end of comments
  - Improper formatting, especially in multi-line conditions
  - Redundant newlines or whitespace
- golangci-lint v2.1.2 is used with golangci-lint-action v7 for GitHub Actions
- Run linting as part of the verification workflow after making changes
