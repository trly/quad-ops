# Quad-Ops Architecture

## Overview

Quad-Ops is architected around four core domain concepts that provide clear separation of concerns and enable cross-platform service management:

1. **compose** - Docker Compose file processing and conversion to platform-agnostic domain models
2. **service** - Platform-agnostic service models and lifecycle management
3. **repository** - Git repository synchronization and unit file storage
4. **config** - Application configuration management

## Architecture Principles

- **Domain-Driven Design**: Domain models are platform-agnostic and represent the business logic
- **Dependency Injection**: No global state; all services constructed with explicit dependencies
- **Model-Centric**: Domain models ARE the platform-neutral representation
- **Direct Orchestration**: Commands orchestrate components directly via dependency injection
- **Clear Boundaries**: Each domain has well-defined responsibilities and interfaces
- **Adapter Pattern**: Platform adapters render domain models to platform-specific artifacts

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI Layer (cmd/)                         │
│     Commands orchestrate components via injected dependencies│
│              daemon │ sync │ up │ down │ unit_*              │
└──────┬─────────┬──────────┬──────────┬──────────────────────┘
       │         │          │          │
┌──────▼─────────▼──────────▼──────────▼──────────────────────┐
│                    Component Layer                           │
│  ┌─────────┐  ┌─────────────────┐  ┌──────────┐  ┌────────┐│
│  │ Compose │  │ Service Models  │  │Repository│  │ Config ││
│  │Processor│  │ (platform-neutral)│ │          │  │        ││
│  └─────────┘  └─────────────────┘  └──────────┘  └────────┘│
│                                                               │
│  ┌──────────────────────┐                                    │
│  │  Platform Adapters   │                                    │
│  │ Systemd  │  Launchd  │                                    │
│  │Renderer  │  Renderer │                                    │
│  │Lifecycle │  Lifecycle│                                    │
│  └──────────────────────┘                                    │
└───────────────────────────────────────────────────────────────┘
```

**Key Concept**: Commands in `cmd/` orchestrate components directly. No intermediate orchestrator layer. Components are injected via `cmd.App` dependency container.

## Package Structure

```
quad-ops/
├── cmd/                              # CLI commands + orchestration
│   ├── daemon.go                     # Daemon command (periodic sync)
│   ├── sync.go                       # Sync command
│   ├── up.go, down.go                # Service lifecycle commands
│   ├── unit_*.go                     # Unit inspection commands
│   ├── root.go                       # RootCommand + App DI container + wiring
│   └── deps.go                       # Common dependencies (testing support)
│
├── internal/
│   ├── compose/                      # Docker Compose processing
│   │   ├── processor.go              # Docker Compose → Domain Models
│   │   └── options.go                # Compose processing options
│   │
│   ├── service/                      # Platform-agnostic domain models
│   │   ├── models.go                 # Spec, Container, Volume, Network, Build
│   │   └── validate.go               # Model validation
│   │
│   ├── platform/                     # Platform abstraction
│   │   ├── interfaces.go             # Renderer, Lifecycle, Artifact
│   │   │
│   │   ├── systemd/                  # Linux systemd adapter
│   │   │   ├── renderer.go           # Domain Models → systemd units
│   │   │   ├── lifecycle.go          # systemd lifecycle via dbus
│   │   │   └── options.go            # Systemd-specific options
│   │   │
│   │   └── launchd/                  # macOS launchd adapter (future)
│   │       ├── renderer.go           # Domain Models → launchd plists
│   │       ├── lifecycle.go          # launchd lifecycle via launchctl
│   │       └── options.go            # Launchd-specific options
│   │
│   ├── repository/                   # Git and artifact storage
│   │   ├── git.go                    # Git synchronization
│   │   ├── artifacts.go              # Artifact storage/retrieval
│   │   └── models.go                 # RepoRef, ArtifactChange
│   │
│   └── config/                       # Configuration management
│       ├── provider.go               # Config provider interface
│       ├── settings.go               # Settings struct
│       └── viper.go                  # Viper implementation
│
└── internal/ (supporting packages)
    ├── dependency/                   # Dependency graph
    ├── fs/                           # File system operations
    ├── log/                          # Logging
    ├── execx/                        # Command execution
    └── validate/                     # Validation
```

**Key Changes from Legacy:**
- No separate IR package - domain models live in `internal/service/models.go`
- **No separate app/orchestrator layer** - orchestration in cmd/
- Platform adapters are siblings under `internal/platform/`
- Compose processor returns `[]service.Spec` directly (no stuttering!)
- Simpler, flatter structure with clear responsibilities

## Core Interfaces

### Domain Models

```go
// internal/service/models.go
type Spec struct {
    Name         string
    Description  string
    Container    Container
    Volumes      []Volume
    Networks     []Network
    DependsOn    []string              // Service names
    Annotations  map[string]string
}

type Container struct {
    Image         string
    Command       []string
    Args          []string
    Env           map[string]string
    WorkingDir    string
    User          string
    Ports         []Port
    Mounts        []Mount
    Resources     Resources
    RestartPolicy RestartPolicy
    Healthcheck   *Healthcheck
    Security      Security
    Build         *Build
}

type Volume struct {
    Name     string
    Source   string
    Target   string
    ReadOnly bool
    Type     VolumeType    // "host" | "named" | "tmpfs"
    Options  map[string]string
}

type Network struct {
    Name    string
    Driver  string
    Options map[string]string
    IPAM    *IPAM
}
```

### Compose Processing

```go
// internal/compose/processor.go
type Processor interface {
    // Process parses Docker Compose files and returns platform-agnostic service specs
    Process(ctx context.Context, files []string, opts Options) ([]service.Spec, error)
}

type Options struct {
    ProjectName string
    EnvFiles    []string
    Profiles    []string
    WorkingDir  string
}
```

### Platform Abstraction

```go
// internal/platform/interfaces.go
type Artifact struct {
    Path    string
    Content []byte
    Mode    fs.FileMode
}

type Renderer interface {
    // Name returns the platform name (e.g., "systemd", "launchd")
    Name() string
    
    // Render converts platform-agnostic service specs to platform-specific artifacts
    Render(ctx context.Context, specs []service.Spec) ([]Artifact, error)
}

type Lifecycle interface {
    // Name returns the platform name
    Name() string
    
    // Reload reloads the service manager configuration
    Reload(ctx context.Context) error
    
    // Start starts a service
    Start(ctx context.Context, name string) error
    
    // Stop stops a service
    Stop(ctx context.Context, name string) error
    
    // Restart restarts a service
    Restart(ctx context.Context, name string) error
    
    // Status returns the status of a service
    Status(ctx context.Context, name string) (ServiceStatus, error)
}

type ServiceStatus struct {
    Name        string
    Active      bool
    State       string
    Description string
}
```

### Repository

```go
// internal/repository/models.go
type GitSyncer interface {
    SyncAll(ctx context.Context, repos []config.Repository) ([]SyncResult, error)
    SyncRepo(ctx context.Context, repo config.Repository) (SyncResult, error)
}

type SyncResult struct {
    Repo         config.Repository
    Changed      bool
    ChangedFiles []string
    Error        error
}

type ArtifactStore interface {
    Write(ctx context.Context, artifacts []platform.Artifact) error
    List(ctx context.Context) ([]platform.Artifact, error)
    Delete(ctx context.Context, paths []string) error
}
```

### Config

```go
// internal/config/provider.go
type Provider interface {
    GetConfig() *Settings
    SetConfig(*Settings)
    InitConfig() *Settings
    SetConfigFilePath(string)
}
```

### Command Dependency Container

```go
// cmd/root.go - Combined: RootCommand + App + DI wiring

// App struct (dependency container) - injected into all commands via context
type App struct {
    Logger            log.Logger
    ConfigProvider    config.Provider
    GitSyncer         repository.GitSyncer
    ComposeProcessor  compose.Processor
    PlatformRenderer  platform.Renderer
    PlatformLifecycle platform.Lifecycle
    ArtifactStore     repository.ArtifactStore
    Validator         SystemValidator
    OutputFormat      string
}

// NewApp creates and wires all application dependencies
func NewApp(logger log.Logger, configProv config.Provider) *App {
    cfg := configProv.GetConfig()
    
    // Create platform (systemd or launchd based on runtime.GOOS)
    var renderer platform.Renderer
    var lifecycle platform.Lifecycle
    
    if runtime.GOOS == "darwin" {
        renderer = launchd.NewRenderer(...)
        lifecycle = launchd.NewLifecycle(...)
    } else {
        renderer = systemd.NewRenderer(...)
        lifecycle = systemd.NewLifecycle(...)
    }
    
    return &App{
        Logger:            logger,
        ConfigProvider:    configProv,
        GitSyncer:         repository.NewGitSyncer(...),
        ComposeProcessor:  compose.NewProcessor(logger),
        PlatformRenderer:  renderer,
        PlatformLifecycle: lifecycle,
        ArtifactStore:     repository.NewArtifactStore(cfg.QuadletDir),
        Validator:         validate.NewValidator(logger, runner),
    }
}

// Helper to retrieve App from command context
func AppFromContext(ctx context.Context) (*App, bool) {
    v := ctx.Value(appContextKey)
    a, ok := v.(*App)
    return a, ok
}
```

## Data Flow: Git Sync → Compose Processing → Service Deployment

```
1. CLI sync command (cmd/sync.go RunE method)
   ↓
2. Uses injected app.* dependencies:
   - results := app.GitSyncer.SyncAll(repos)
   ↓
3. For each changed repo:
   ├─→ specs := app.ComposeProcessor.Process(composeFiles, opts)
   ├─→ artifacts := app.PlatformRenderer.Render(specs)
   ├─→ app.ArtifactStore.Write(artifacts)
   ├─→ app.PlatformLifecycle.Reload()
   └─→ app.PlatformLifecycle.Restart(serviceName)
```

**Direct Orchestration:**
- Commands orchestrate components directly via dependency injection
- No intermediate orchestrator/use-case layer
- Simpler, more direct code flow

## Example: Sync Command Orchestration

```go
// cmd/sync.go
func (c *SyncCommand) Run(ctx context.Context, app *App, opts SyncOptions) error {
    // 1. Sync git repositories
    results, err := app.GitSyncer.SyncAll(ctx, app.Config.Repositories)
    if err != nil {
        return fmt.Errorf("git sync failed: %w", err)
    }
    
    // 2. Process each changed repository
    for _, result := range results {
        if !result.Changed && !opts.Force {
            continue
        }
        
        // 3. Parse Docker Compose files
        composeOpts := compose.Options{
            ProjectName: result.Repo.Name,
            WorkingDir:  result.Path,
        }
        specs, err := app.ComposeProcessor.Process(ctx, result.ComposeFiles, composeOpts)
        if err != nil {
            return fmt.Errorf("compose processing failed: %w", err)
        }
        
        // 4. Render to platform-specific artifacts
        artifacts, err := app.PlatformRenderer.Render(ctx, specs)
        if err != nil {
            return fmt.Errorf("rendering failed: %w", err)
        }
        
        // 5. Write artifacts to disk
        if err := app.ArtifactStore.Write(ctx, artifacts); err != nil {
            return fmt.Errorf("artifact write failed: %w", err)
        }
        
        // 6. Reload service manager
        if err := app.PlatformLifecycle.Reload(ctx); err != nil {
            return fmt.Errorf("reload failed: %w", err)
        }
        
        // 7. Restart changed services
        for _, spec := range specs {
            if err := app.PlatformLifecycle.Restart(ctx, spec.Name); err != nil {
                app.Logger.Error("Failed to restart service", "service", spec.Name, "error", err)
            }
        }
    }
    
    return nil
}
```

## Component Responsibilities

### Compose
- Parse Docker Compose YAML files
- Convert Compose definitions to platform-agnostic `service.Spec` models
- Normalize multi-container definitions into multiple service specs
- Resolve environment variables and profiles
- **Does NOT**: Write files, manage lifecycle, know about platforms

### Service Models
- Define platform-agnostic service specification structure (`service.Spec`)
- Contain all necessary configuration (container, volumes, networks, dependencies)
- Validate business rules
- **Are**: The single source of truth for service definitions

### Platform Adapters
- Render domain models to platform-specific artifacts (.service files, .plist files)
- Implement lifecycle operations (start/stop/restart/status) via platform APIs
- Platform-specific optimizations and mappings
- **Does NOT**: Parse Compose, sync git, store artifacts

### Repository
- Sync git repositories containing Compose files
- Store and retrieve platform artifacts
- Track which files have changed
- **Does NOT**: Parse Compose, render artifacts, manage service lifecycle

### Config
- Load and provide application configuration
- Manage configuration sources (files, env vars)
- **Does NOT**: Business logic, service management

### Commands (cmd/)
- Parse CLI flags and options
- Validate system requirements (PreRunE)
- **Orchestrate components** directly in RunE methods
- Coordinate git sync → compose → render → store → lifecycle workflow
- Handle errors and user output

## Dependency Graph

```
cmd/ → compose/service/repository/config/platform
        ↓
     service models
        ↓
   platform adapters
        ↓
   external libs
```

**Dependency Rules:**
- `cmd` depends directly on `compose`, `service`, `repository`, `config`, `platform`
- `cmd.App` holds all injected component dependencies
- `cmd.factory.NewApp()` creates and wires all components
- `compose` produces `service.Spec` models
- `platform` adapters consume `service.Spec` models
- No circular dependencies
- Platform adapters are independent of each other
- Commands orchestrate components in their RunE methods

## CLI Command → Component Mapping

Commands orchestrate components directly in their `RunE` methods:

| Command        | Components Used (via app.*)                            |
| -------------- | ------------------------------------------------------ |
| `daemon`       | GitSyncer, ComposeProcessor, Renderer, Lifecycle (periodic) |
| `sync`         | GitSyncer, ComposeProcessor, Renderer, ArtifactStore, Lifecycle |
| `up`           | ComposeProcessor (optional), Renderer, ArtifactStore, Lifecycle |
| `down`         | Lifecycle, ArtifactStore (optional cleanup)            |
| `unit list`    | ArtifactStore, Lifecycle                               |
| `unit status`  | Lifecycle                                              |
| `unit restart` | Lifecycle                                              |
| `validate`     | ComposeProcessor                                       |

## Migration Strategy

**Note:** No backward compatibility needed for internal packages - only CLI must maintain compatibility.

### Phase 1: Create Domain Models
- Create `internal/service/models.go` with Spec, Container, Volume, Network, Build structs
- Create `internal/service/validate.go` for model validation
- Move existing container/volume/network logic to use new models

### Phase 2: Refactor Compose
- Update `internal/compose/processor.go` to return `[]service.Spec`
- Remove QuadletUnit creation
- Normalize multi-container setups into multiple Spec instances
- Add tests with new model outputs

### Phase 3: Create Platform Interfaces
- Create `internal/platform/interfaces.go` (Renderer, Lifecycle, Artifact)
- Define contracts independent of systemd

### Phase 4: Systemd Platform Adapter
- Create `internal/platform/systemd/renderer.go`
  - Input: `[]service.Spec`
  - Output: `[]platform.Artifact` (systemd unit files)
  - Reuse existing quadlet generation logic
- Create `internal/platform/systemd/lifecycle.go`
  - Wrap existing dbus/systemctl operations
- Add tests for rendering Spec → systemd units

### Phase 5: Repository Simplification
- Update `internal/repository/artifacts.go` for ArtifactStore
- Update `internal/repository/git.go` for GitSyncer
- Remove old unit repository if no longer needed

### Phase 6: Update CLI Commands
- Consolidate `cmd/app.go` and `cmd/root.go` into single `cmd/root.go`:
  - Move App struct and NewApp() into root.go
  - Add AppFromContext() helper for safer context access
  - Remove App.Config field (use ConfigProvider.GetConfig() instead)
  - Platform selection (systemd vs launchd) in NewApp()
- Update each `cmd/*.go` RunE method to orchestrate components directly:
  - `sync.go`: GitSyncer → ComposeProcessor → Renderer → ArtifactStore → Lifecycle
  - `up.go`: Renderer → ArtifactStore → Lifecycle.Start
  - `down.go`: Lifecycle.Stop
  - etc.
- Keep all flags, command names, and UX identical
- Test end-to-end for regressions

### Phase 7: Cleanup
- Delete old packages (`internal/unit/quadlet_unit.go`, etc.)
- Delete deprecated interfaces
- Remove any remaining app/orchestrator layer code
- Update all documentation

## Benefits of This Architecture

1. **Simpler**: No separate IR layer - domain models ARE platform-agnostic
2. **Direct**: No orchestrator layer - commands orchestrate components directly
3. **Cross-Platform**: Easy to add launchd, Windows services, etc.
4. **Testable**: All components have clear interfaces and dependencies
5. **Maintainable**: Clear separation of concerns, obvious data flow
6. **Extensible**: Add platforms without touching compose/repository
7. **DI-First**: No global state, all dependencies injected via cmd.App
8. **Model-Centric**: Single source of truth in `service.Spec` models
9. **Lint-Friendly**: No stuttering names (`service.Spec` not `service.Service`)
10. **Standard Go**: Follows typical Go CLI patterns (orchestrate in RunE)

## Example: Adding launchd Support

1. Create `internal/platform/launchd/`
2. Implement `Renderer` interface:
   ```go
   func (r *LaunchdRenderer) Render(ctx context.Context, specs []service.Spec) ([]platform.Artifact, error) {
       // Convert service.Spec models to .plist files
   }
   ```
3. Implement `Lifecycle` interface using `launchctl` commands
4. Update platform selection in `cmd/factory.go`:
   ```go
   if runtime.GOOS == "darwin" {
       renderer = launchd.NewRenderer(...)
       lifecycle = launchd.NewLifecycle(...)
   }
   ```
5. **No changes needed** in compose, service models, repository, or commands!

## Key Differences from Original Design

### What Changed
- ❌ Removed `internal/core/ir` package
- ❌ Removed separate IR types (AppGraph, UnitSpec)
- ❌ Removed `internal/app` layer entirely (redundant with cmd)
- ❌ Removed orchestrator types (unnecessary abstraction)
- ❌ Removed complex multi-layer abstraction
- ❌ Removed stuttering names (no more `service.Service`)
- ✅ Domain models (`service.Spec`) are platform-neutral
- ✅ Commands orchestrate components directly in RunE methods
- ✅ Direct flow: cmd → components (via DI)
- ✅ Simpler package structure (no app layer)
- ✅ Fewer concepts to understand
- ✅ Passes golangci-lint without exceptions

### What Stayed the Same
- ✅ Platform abstraction (Renderer, Lifecycle interfaces)
- ✅ Dependency injection throughout
- ✅ Clear separation of concerns
- ✅ Testability
- ✅ CLI backward compatibility

## Design Philosophy

**"Make it as simple as possible, but no simpler"**

- Domain models can be platform-agnostic without a separate IR layer
- Commands can orchestrate directly without an app/orchestrator layer
- Platform adapters handle platform-specific rendering
- Less abstraction = easier to understand and maintain
- Pragmatic over theoretical purity
- Standard Go CLI patterns over custom architectures
