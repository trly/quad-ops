# Full Pipeline Simplification: Compose → Spec → Quadlet/Plist

Simplification of the complete conversion pipeline: **Compose YAML → service.Spec → Platform Artifacts (systemd units or launchd plists)**.

## The Problem

1. **Too many layers**: Compose → intermediate models → podman args → output (3+ steps when 2 would suffice)
2. **Duplication**: systemd and launchd both build podman arguments independently
3. **Scattered logic**: Helper functions spread across many files
4. **Test fragmentation**: Implementation detail tests instead of behavior validation
5. **Over-abstraction**: Intermediate Unit models add complexity without benefit

## Architectural Goals

**Primary goal:** Direct mapping from service.Spec to platform output formats

1. **Remove intermediate layers**: Spec → output directly (not Spec → Unit → output)
2. **Eliminate duplication**: Shared podman args builder for both platforms
3. **Consolidate helpers**: Merge scattered utilities into cohesive rendering logic
4. **Behavior-focused tests**: Table-driven and golden tests replace implementation checks
5. **Simpler data flow**: Make the pipeline easy to understand and debug

## Architecture Overview

```
Input: docker-compose.yml
   ↓
[Compose Package]
  ReadProjects() → *types.Project
  NewConverter(workingDir) → *Converter
  Convert(project) → []service.Spec
   ↓
[Service Spec] - Platform-agnostic domain model
  - Name, Description
  - Container (image, command, env, ports, mounts)
  - Volumes, Networks, DependsOn
   ↓
[Platform Renderers]
  ├─ Systemd: Spec → Quadlet unit files (.container, .network, .volume)
  └─ Launchd: Spec → Plist XML files
   ↓
Output: Unit files or plist artifacts
```

## Simplified Compose Package (~350 lines)

### convert.go (~320 lines)

```go
// Core types
type Converter struct {
    workingDir string
}

// Public API
func NewConverter(workingDir string) *Converter
func (c *Converter) Convert(project *types.Project) ([]service.Spec, error)

// Private: Organized by concern (12-14 focused methods)
// Container section (3 methods)
func (c *Converter) convertContainer(svc types.ServiceConfig, ...) service.Container
func (c *Converter) convertSecurity(svc types.ServiceConfig) service.Security
func (c *Converter) convertResources(deploy *types.DeployConfig) service.Resources

// Network/Volume section (2 methods)
func (c *Converter) convertMounts(volumes []types.ServiceVolumeConfig, ...) []service.Mount
func (c *Converter) convertNetworks(svc types.ServiceConfig, ...) []service.Network

// Ports/Health/Restart (3 methods)
func (c *Converter) convertPorts(ports []types.ServicePortConfig) []service.Port
func (c *Converter) convertHealthcheck(hc *types.HealthCheckConfig) *service.Healthcheck
func (c *Converter) convertRestart(restart string) service.RestartPolicy

// Validation/Helpers (3 methods)
func (c *Converter) validateProject(proj *types.Project) error
func (c *Converter) formatBytes(b types.UnitBytes) string
func (c *Converter) convertCPU(nanoCPUs types.NanoCPUs) (int64, int64)
```

### reader.go (~30 lines)

```go
// Thin wrapper over compose-go
func ReadProjects(path string) ([]*types.Project, error)
func ParseComposeFile(path string) (*types.Project, error)
func sanitizeProjectName(name string) string  // Local helper
```

### helpers.go (remove entirely)

- Inline `Prefix()` → 1 line in convert.go
- Inline `IsExternal()` → 2 lines in validate method
- Move `FindEnvFiles()` → reader.go if needed

**Total compose: ~350 lines**

---

## Simplified Systemd Renderer (~1200 lines)

Current: 5345 lines across 8 files with complex unit generation

### Minimal Structure

#### renderer.go (~400 lines)

```go
type Renderer struct {
    logger log.Logger
}

// Public API
func NewRenderer(logger log.Logger) *Renderer
func (r *Renderer) Name() string
func (r *Renderer) Render(ctx context.Context, specs []service.Spec) (*RenderResult, error)

// Private: Build units
func (r *Renderer) renderService(spec service.Spec) ([]Artifact, error)
func (r *Renderer) buildContainerUnit(spec service.Spec) (Artifact, error)
func (r *Renderer) buildNetworkUnit(network service.Network) (Artifact, error)
func (r *Renderer) buildVolumeUnit(volume service.Volume) (Artifact, error)
```

#### quadlet.go (~400 lines)

Direct mapping: service.Spec → Quadlet INI format

```go
// Quadlet INI writer
type QuadletWriter struct {
    buf *bytes.Buffer
}

func (w *QuadletWriter) Container(spec service.Spec, containerName string) string
func (w *QuadletWriter) AddImage(image string)
func (w *QuadletWriter) AddPorts(ports []service.Port)
func (w *QuadletWriter) AddMounts(mounts []service.Mount)
func (w *QuadletWriter) AddEnvironment(env map[string]string)
// ... simple setters for each Quadlet directive
```

#### unit.go (~100 lines)

```go
// Quadlet unit file structure
type Unit struct {
    Name       string  // e.g., "myservice.container"
    Path       string  // e.g., "/home/user/.config/containers/systemd/myservice.container"
    Content    string  // INI format
    Type       string  // "container", "network", "volume"
}
```

#### podman_args.go (~200 lines)

Build podman command line from service.Spec

```go
func BuildPodmanArgs(spec service.Spec, containerName string) []string
```

#### helpers.go (~100 lines)

Systemd-specific utilities (dependency ordering, label conversion, etc.)

**Total systemd: ~1200 lines**

---

## Simplified Launchd Renderer (~1200 lines)

Current: 5287 lines across 14 files with complex plist generation

### Minimal Structure

#### renderer.go (~300 lines)

```go
type Renderer struct {
    opts   Options
    logger log.Logger
}

// Public API
func NewRenderer(opts Options, logger log.Logger) (*Renderer, error)
func (r *Renderer) Name() string
func (r *Renderer) Render(ctx context.Context, specs []service.Spec) (*RenderResult, error)

// Private
func (r *Renderer) renderService(spec service.Spec) ([]Artifact, error)
func (r *Renderer) buildPlist(spec service.Spec) (*Plist, error)
```

#### plist.go (~300 lines)

Direct launchd plist generation

```go
type Plist struct {
    Label                string
    Program              string
    ProgramArguments     []string
    EnvironmentVariables map[string]string
    KeepAlive            interface{} // bool or map
    // ... other fields
}

func (p *Plist) Encode() []byte
```

#### podman_args.go (~200 lines)

Build podman command for launchd

```go
func BuildPodmanArgs(spec service.Spec, containerName string) []string
```

#### options.go (~150 lines)

Configuration and validation

```go
type Options struct {
    RepositoryDir string
    QuadletDir    string
    UserMode      bool
}

func (o *Options) Validate() error
```

#### helpers.go (~150 lines)

Launchd-specific utilities (path handling, label building, etc.)

#### lifecycle.go (~100 lines)

Service management (start, stop, restart)

```go
type Lifecycle struct { ... }
func (l *Lifecycle) Start(label string) error
func (l *Lifecycle) Stop(label string) error
```

**Total launchd: ~1200 lines**

---

## Key Simplifications Across the Pipeline

### 1. **Eliminate Duplication Between systemd and launchd**

Currently: Both write similar podman arguments in different ways

```
internal/platform/systemd/podmanargs.go       ~200 lines
internal/platform/launchd/podmanargs.go       ~200 lines
```

**Proposal**: Single shared `PodmanArgsBuilder` in `/internal/platform/podman_args.go`

```go
package platform

type PodmanArgsBuilder struct {
    spec service.Spec
    containerName string
}

func (b *PodmanArgsBuilder) Build() []string {
    // Single implementation used by both renderers
}
```

Both systemd and launchd import and use this, eliminating 200+ lines of duplication.

### 2. **Consolidate Mount Handling**

Currently fragmented:

- `internal/compose/spec_converter.go` - Converts compose volumes to mounts
- `internal/platform/systemd/*_test.go` - Tests bind mount generation
- `internal/platform/launchd/*` - Separate mount logic for plist

**Proposal**: Single canonical mount format in `service.Mount`

- Convert once in compose package
- Both renderers read and format for their platform
- Clear abstraction point

### 3. **Test-First Structure**

For each component, write table-driven tests that become the spec:

```go
// renderer_test.go - Single test file per renderer
var testCases = []struct {
    name     string
    spec     service.Spec      // Input: platform-agnostic
    expected string            // Expected: rendered output
}{
    {
        name: "basic container",
        spec: service.Spec{...},
        expected: "[Container]\nImage=...\n",
    },
}

func TestRenderer(t *testing.T) {
    for _, tc := range testCases {
        r := NewRenderer(...)
        result, _ := r.Render(context.Background(), []service.Spec{tc.spec})
        if got := result.Artifacts[0].Content; got != tc.expected {
            t.Errorf("got %q, want %q", got, tc.expected)
        }
    }
}
```

### 4. **Remove Unused Abstractions**

Delete from compose package:

- `Repository` interface - Never used
- `SystemdManager` interface - Platform-specific, doesn't belong
- `FileSystem` interface - Unused
- Test files that test internal implementation details

Delete from platform packages:

- Overly generic `Processor` interfaces
- `PluginRegistry` and dynamic dispatch (not needed for static renderers)
- Configuration inheritance and cascading (simplify with direct options)

### 5. **Direct Mapping Approach**

Instead of:

```
service.Spec
  → Container model
  → UnitFile model
  → String serialization
```

Use direct generation:

```
service.Spec → [Write to string directly]
```

Example: systemd container unit

```go
func (r *Renderer) buildContainerUnit(spec service.Spec) (Artifact, error) {
    buf := &strings.Builder{}
    buf.WriteString("[Container]\n")
    buf.WriteString(fmt.Sprintf("Image=%s\n", spec.Container.Image))

    for _, port := range spec.Container.Ports {
        buf.WriteString(fmt.Sprintf("Publish=%s:%d:%d/%s\n",
            port.Host, port.HostPort, port.Container, port.Protocol))
    }

    // ... continue for each field

    return Artifact{
        Name: spec.Name + ".container",
        Content: buf.String(),
    }, nil
}
```

Minimal, clear, easy to understand.

---

## Implementation Order

### Phase 1: Compose Simplification (First)

The platform renderers depend on service.Spec quality, so fix compose first:

1. Write comprehensive convert tests
2. Consolidate spec_converter.go methods
3. Remove unused interfaces
4. Verify all tests pass

### Phase 2: Shared Podman Args

Both renderers need podman command generation:

1. Create `/internal/platform/podman_args.go` with unified builder
2. Update systemd renderer to use it
3. Update launchd renderer to use it
4. Delete duplicate implementations

### Phase 3: Systemd Renderer Simplification

1. Audit current renderer
2. Write table-driven tests for each unit type
3. Consolidate mount/volume/network logic
4. Direct string generation (no intermediate models)
5. Keep lifecycle.go (platform management, not rendering)

### Phase 4: Launchd Renderer Simplification

1. Simplify plist generation (direct XML from service.Spec)
2. Consolidate podman args (use shared builder from Phase 2)
3. Write table-driven tests
4. Remove overly generic abstractions

### Phase 5: Test Consolidation

Merge scattered tests into focused table-driven files:

- 16 test files → ~8 test files (one per component)
- Clear test cases with input/expected output
- Tests become documentation

---

## Expected Results

### Code Quality Improvements

- ✅ Single responsibility: each file does one thing
- ✅ Clear data flow: Compose → Spec → Artifacts
- ✅ No dead code or unused abstractions
- ✅ Table-driven tests easy to extend
- ✅ Direct mapping (easy to debug, modify)
- ✅ Reduced cognitive load (fewer files, simpler logic)

### Behavior Guarantees

- ✅ Golden tests ensure output equivalence
- ✅ Service specs unchanged (interface with renderers)
- ✅ Quadlet units identical to current
- ✅ Launchd plists identical to current
- ✅ No breaking changes to CLI or configuration

---

## Why This Matters

**Current state**: 12k lines of renderer code is hard to:

- Understand end-to-end
- Modify without breaking edge cases
- Test comprehensively
- Onboard new developers

**Simplified state**: focus on core logic:

- Clear input → output mapping
- Each transformation is traceable
- Tests validate expected behavior
- Easy to add new platform (just implement Renderer interface)

---

## Migration Strategy

**NO BACKWARD COMPATIBILITY REQUIRED**

This is a complete internal refactoring. Focus on clean, simplified architecture:

1. **Internal APIs**: Breaking changes to internal packages are acceptable and encouraged
2. **Public CLI**: Keep CLI command behavior stable (up, down, sync, etc.)
3. **Unit file output**: Generated quadlet/plist files must remain byte-for-byte identical
4. **Internal packages**: `internal/compose`, `internal/platform/*` can be completely restructured
5. **Tests**: Rewrite tests to match new simplified architecture

**What must stay stable:**

- CLI command interface and behavior
- Configuration file format
- Generated unit file output (golden test validation)

**What can change freely:**

- All internal package structures and APIs
- Method signatures in internal/compose
- Internal abstractions and interfaces
- Test file organization and structure

---

## Success Criteria

### Correctness

- ✅ All existing conversion tests pass (golden test, spec tests)
- ✅ All platform renderer tests pass (systemd, launchd)
- ✅ Generated units/plists are byte-for-byte identical
- ✅ Coverage maintained or improved

### Code Quality

- ✅ No unused abstractions or dead code
- ✅ All public APIs documented with examples

### Maintainability

- ✅ Table-driven tests, easy to add cases
- ✅ Clear separation: parsing, conversion, rendering
- ✅ Adding new platform requires only 1-2 files

---

## Risks & Mitigations

| Risk                    | Likelihood | Mitigation                       |
| ----------------------- | ---------- | -------------------------------- |
| Break systemd rendering | Low        | Golden tests, verify each phase  |
| Break launchd rendering | Low        | Golden tests, verify each phase  |
| Regress performance     | Very Low   | Simple direct mapping, no impact |
| Lose edge case handling | Medium     | Comprehensive table-driven tests |

**Mitigation strategy**: Test-first refactoring ensures no behavior regression.
