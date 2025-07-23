# Agent Guidelines for quad-ops CLI Commands

## Overview
The `cmd/` package provides the command line interface for quad-ops using the [Cobra framework](https://cobra.dev/#concepts). Each command is implemented as a separate file with a consistent structure.

## Architecture & Structure
- **main.go**: Application entry point in `cmd/quad-ops/`
- **root.go**: Root command with global flags and configuration
- **Command files**: Individual commands (up.go, down.go, sync.go, etc.)
- **Subcommands**: Organized under parent commands (unit_*.go files under unit command)

## Command Structure Pattern
Each command follows this consistent pattern:
```go
type CommandName struct{}

func (c *CommandName) GetCobraCommand() *cobra.Command {
    // Command definition with flags, usage, examples
    return cmd
}
```

### Core Operations
- **up.go**: Start containers and services
- **down.go**: Stop containers and services  
- **sync.go**: Synchronize Git repositories and generate unit files

### Unit Management
- **unit.go**: Parent command for unit operations
- **unit_list.go**: List systemd unit files
- **unit_show.go**: Show unit file contents
- **unit_status.go**: Show unit status information

### Image Management  
- **image.go**: Parent command for image operations
- **image_pull.go**: Pull container images

### System Commands
- **config.go**: Show configuration information
- **update.go**: Update quad-ops binary
- **version.go**: Show version and check for updates

## Global Flags & Configuration
Defined in `root.go`

## Development Guidelines

### Adding New Commands
1. Create new file following naming pattern (e.g., `new_command.go`)
2. Implement the command struct with `GetCobraCommand()` method
3. Add command to parent in root.go or appropriate parent command
4. Include proper usage, examples, and flag definitions
5. Follow error handling patterns from existing commands
6. Update the hugo site. See: [AGENT.MD](../site/AGENT.md)

### Command Implementation Patterns
- Use `cobra.Command` struct for command definition
- Implement `Run` or `RunE` functions for command logic
- Add completion functions for dynamic flag values where applicable
- Include `PreRunE` for validation when needed
- Use consistent error handling and logging

### Flag Conventions
- Use consistent naming across similar commands
- Provide completion functions for enum-like flags
- Set appropriate default values
- Include clear descriptions and examples

### Testing Commands
- `go run cmd/quad-ops/main.go <command>` - Test individual commands
- Use `--help` flag to verify command structure and documentation
- Test flag validation and error handling paths

