# Agent Guidelines for quad-ops Internal Packages

## Overview
The `internal/` directory contains the core application logic for quad-ops. It follows a clean architecture pattern with clear separation of concerns between domain logic, infrastructure services, and external integrations.

## Architecture & Structure

### Core Domain
- **compose/**: Docker Compose file processing and conversion to Quadlet units
- **unit/**: Quadlet unit definitions and systemd unit file generation
- **dependency/**: Service dependency management using directed graphs

### Infrastructure Services
- **config/**: Application configuration management using Viper
- **fs/**: File system operations for unit file management
- **git/**: Git repository cloning, updating, and reference management
- **log/**: Centralized logging infrastructure
- **repository/**: Unit data access layer with systemd integration
- **systemd/**: Direct systemd unit management and orchestration

### Utilities
- **util/**: Common operations like sorting and iteration
- **validate/**: Input validation and security checks

## Package Dependencies

### Data Flow
1. **compose** → Main orchestrator using most other packages
2. **unit** → Core domain models used throughout
3. **systemd** → Execution layer for unit operations
4. **fs/git/repository** → Persistence and data access
5. **config/log/validate** → Infrastructure support

### Key Relationships
- `compose` processes Docker Compose files using `unit` models
- `systemd` orchestrates unit operations with dependency awareness
- `fs` manages unit file persistence and change detection
- `git` handles repository synchronization
- `validate` ensures security and input validation across packages

## Development Guidelines

### Adding New Packages
1. Follow the established patterns in existing packages
2. Define clear interfaces for testability and modularity
3. Use dependency injection where appropriate
4. Include comprehensive error handling and logging
5. Add validation for all inputs and configurations

### Package Design Principles
- Single responsibility per package
- Clear separation between domain logic and infrastructure
- Interfaces for external dependencies
- Consistent error handling patterns
- Security-first approach with input validation

### Testing Patterns
- Unit tests for individual package functionality
- Integration tests for cross-package interactions
- Mock external dependencies (git, systemd, filesystem)
- Test security validation and error conditions

## Package-Specific Guidelines
Each package has detailed guidelines in its own AGENT.md file:
- `compose/AGENT.md` - Docker Compose processing patterns
- `config/AGENT.md` - Configuration management guidelines
- `dependency/AGENT.md` - Dependency graph operations
- `fs/AGENT.md` - File system operation patterns
- `git/AGENT.md` - Git repository management
- `log/AGENT.md` - Logging best practices
- `repository/AGENT.md` - Data access patterns
- `systemd/AGENT.md` - Systemd integration guidelines
- `unit/AGENT.md` - Unit definition and generation
- `util/AGENT.md` - Utility function patterns
- `validate/AGENT.md` - Validation and security practices
