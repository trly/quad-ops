# Agent Guidelines for quad-ops Internal Packages

## Overview
The `internal/` directory contains the core application logic for quad-ops. It follows a clean architecture pattern with clear separation of concerns between domain logic, infrastructure services, and external integrations.

## Architecture & Structure

### Core Domain
- **app/**: Application entry points & orchestration
- **quadlet/**: Core domain functionality
  - **unit/**: Unit representation & generation  
  - **systemd/**: systemd integration & orchestration
- **graph/**: Pure dependency graph utilities

### Infrastructure Services
- **infra/**: Infrastructure services
  - **fs/**: File I/O operations
  - **git/**: Git repository management
  - **repo/**: Data access layer (renamed from repository)
- **config/**: Configuration management using Viper
- **log/**: Centralized logging infrastructure
- **validation/**: All validation functions

## Package Dependencies

### Data Flow
1. **app** → Main orchestrator using most other packages
2. **unit** → Core domain models used throughout
3. **systemd** → Execution layer for unit operations
4. **fs/git/repo** → Persistence and data access
5. **config/log/validation** → Infrastructure support

### Key Relationships
- `app` processes Docker Compose files using `unit` models
- `systemd` orchestrates unit operations with dependency awareness
- `fs` manages unit file persistence and change detection
- `git` handles repository synchronization
- `validation` ensures security and input validation across packages

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
- `app/AGENT.md` - Docker Compose processing patterns
- `config/AGENT.md` - Configuration management guidelines
- `graph/AGENT.md` - Dependency graph operations
- `infra/fs/AGENT.md` - File system operation patterns
- `infra/git/AGENT.md` - Git repository management
- `infra/repo/AGENT.md` - Data access patterns
- `log/AGENT.md` - Logging best practices
- `quadlet/systemd/AGENT.md` - Systemd integration guidelines
- `quadlet/unit/AGENT.md` - Unit definition and generation
- `validation/AGENT.md` - Validation and security practices
