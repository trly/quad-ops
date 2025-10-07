# Architecture Migration Status

This document tracks the progress of migrating quad-ops to the new architecture described in ARCHITECTURE.md.

## Completed

### Phase 1: Domain Models ✅
- [x] Created `internal/service/models.go` with platform-agnostic domain models:
  - `Spec` - Core service specification
  - `Container` - Container configuration
  - `Volume` - Volume definition
  - `Network` - Network definition  
  - `Build` - Build configuration
  - Supporting types (Port, Mount, Resources, Healthcheck, Security, etc.)
  
- [x] Created `internal/service/validate.go` for model validation:
  - Service name validation (systemd/filesystem safe)
  - Container configuration validation
  - Healthcheck validation
  - Build validation
  - Volume/Network validation
  - Name sanitization for cross-platform safety

### Phase 2 & 3: Platform Interfaces ✅
- [x] Created `internal/platform/interfaces.go`:
  - `Artifact` - Platform-specific artifact representation
  - `RenderResult` - Render output with change detection metadata
  - `ChangeStatus` - Per-service change tracking
  - `Renderer` - Interface for converting specs to artifacts
  - `ServiceStatus` - Rich service status information
  - `Lifecycle` - Interface for service lifecycle management
  - `Platform` - Combined Renderer + Lifecycle interface
  - Batch operations (StartMany, StopMany, RestartMany)

## In Progress

### Phase 2: Compose Processor Update
- [ ] Update `internal/compose/processor.go` to return `[]service.Spec`
- [ ] Remove QuadletUnit creation from compose package
- [ ] Normalize multi-container setups into multiple Spec instances
- [ ] Add golden tests for compose → service.Spec conversion
- [ ] Handle all compose features (profiles, env-files, build contexts, etc.)

## TODO

### Phase 4: Systemd Platform Adapter
- [ ] Create `internal/platform/systemd/renderer.go`:
  - Implement `Renderer` interface
  - Convert `service.Spec` → systemd unit files
  - Reuse existing quadlet generation logic from `internal/unit/`
  - Generate content hashes for change detection
  - Add golden file tests comparing to legacy output
  
- [ ] Create `internal/platform/systemd/lifecycle.go`:
  - Implement `Lifecycle` interface
  - Wrap existing dbus/systemctl operations
  - Implement batch operations
  - Add error handling and retry logic

### Phase 5: Repository Simplification
- [ ] Update `internal/repository/artifacts.go` for `ArtifactStore` interface
- [ ] Implement atomic write semantics (write temp + fsync + rename)
- [ ] Add content hash-based change detection
- [ ] Update `internal/repository/git.go` for `GitSyncer` interface
- [ ] Add tests for atomic operations and rollback

### Phase 6: CLI Command Updates
- [ ] Consolidate `cmd/app.go` and `cmd/root.go`:
  - Move App struct into root.go
  - Add `AppFromContext()` helper
  - Add platform selection logic (systemd vs launchd detection)
  - Wire new interfaces (Renderer, Lifecycle, ArtifactStore, GitSyncer)
  
- [ ] Update command RunE methods:
  - `sync.go`: GitSyncer → ComposeProcessor → Renderer → ArtifactStore → Lifecycle
  - `up.go`: Renderer → ArtifactStore → Lifecycle.Start
  - `down.go`: Lifecycle.Stop
  - `daemon.go`: Periodic sync orchestration
  - Unit commands: Use Lifecycle for status/restart/etc.
  
- [ ] Add CLI parity harness:
  - Capture stdout/stderr for key commands
  - Compare against baseline outputs
  - Validate exit codes
  - Run in CI

### Phase 7: Cleanup
- [ ] Delete deprecated packages:
  - `internal/unit/` (replaced by platform/systemd/renderer)
  - Old orchestration code
  - Unused adapter code
  
- [ ] Update all documentation
- [ ] Run `task build` and fix all issues
- [ ] Run `task test` and verify all tests pass
- [ ] Update README with new architecture

## Key Design Decisions

1. **No separate IR layer** - Domain models in `service.Spec` are platform-agnostic
2. **No app/orchestrator layer** - Commands orchestrate components directly in RunE
3. **Change detection in renderer** - Renderers return content hashes to enable selective restarts
4. **Atomic artifact writes** - ArtifactStore ensures all-or-nothing writes with rollback
5. **Batch lifecycle operations** - StartMany/StopMany/RestartMany for efficiency
6. **Platform selection at runtime** - Detect systemd vs launchd in App construction

## Testing Strategy

- Unit tests for domain models (validation, sanitization)
- Golden file tests for compose → service.Spec conversion
- Golden file tests for service.Spec → systemd unit rendering
- Integration tests with mocked Git/Renderer/Lifecycle
- CLI parity tests to ensure backward compatibility
- Performance tests with large compose fixtures (100+ services)

## Migration Notes

- Maintain CLI backward compatibility throughout
- Use feature flags to toggle new vs old paths during migration
- Keep legacy code until Phase 7 cleanup
- Run both old and new paths in parallel during Phase 6 to validate parity
