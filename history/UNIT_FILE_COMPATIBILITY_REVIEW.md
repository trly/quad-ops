# Unit File Compatibility Review

**Date**: November 8, 2025
**Subject**: Verification of quad-ops systemd unit file generation against Podman Quadlet spec
**Specification**: [podman-systemd.unit.5](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)

## Executive Summary

The quad-ops renderer generates Podman Quadlet-compatible `.container`, `.volume`, `.network`, and `.build` unit files. After comparing the implementation against the official Podman documentation, the generation is largely compatible with documented Quadlet directives. However, several potential issues and missing optimizations were identified that could improve robustness and feature coverage.

## Compatibility Assessment by Section

### ✅ [Unit] Section (COMPATIBLE)

The implementation correctly handles standard systemd unit directives:

- **Description** ✅ - Properly formatted
- **After/Requires** ✅ - Correctly chains dependencies for containers, volumes, networks, and build units
- **WorkingDirectory** ✅ - Used in build units only (appropriate)

**Assessment**: This section is well-implemented with proper dependency chains.

---

### ✅ [Container] Section (MOSTLY COMPATIBLE)

#### Implemented & Verified ✅

| Directive | Status | Notes |
|-----------|--------|-------|
| `Image=` | ✅ Required field present | Correctly positioned, always rendered |
| `ContainerName=` | ✅ Supported | From `Container.ContainerName` |
| `HostName=` | ✅ Supported | From `Container.Hostname` |
| `Environment=` | ✅ Supported | Map-based, properly sorted |
| `EnvironmentFile=` | ✅ Supported | From `Container.EnvFiles` array |
| `PublishPort=` | ✅ Supported | Proper format: `host:port/protocol` |
| `Volume=` | ✅ Supported | Bind and volume mounts with `ro` flag |
| `Tmpfs=` | ✅ Supported | From `Container.Tmpfs` |
| `Network=` | ✅ Supported | Mode + aliases + service networks |
| `Entrypoint=` | ✅ Supported | Space-joined array (see issue below) |
| `Exec=` | ✅ Supported | Command args, space-joined (see issue below) |
| `User=` | ✅ Supported | From `Container.User` |
| `Group=` | ✅ Supported | From `Container.Group` |
| `WorkingDir=` | ✅ Supported | From `Container.WorkingDir` |
| `RunInit=` | ✅ Supported | Boolean to "yes" conversion |
| `ReadOnly=` | ✅ Supported | Boolean to "yes" conversion |
| `Label=` | ✅ Supported | Multiple labels, includes managed-by tag |
| `HealthCmd=` | ✅ Supported | Proper CMD/CMD-SHELL handling |
| `HealthInterval=` | ✅ Supported | Duration formatting (30s, 1m, 1h) |
| `HealthTimeout=` | ✅ Supported | Duration formatting |
| `HealthRetries=` | ✅ Supported | Integer value |
| `HealthStartPeriod=` | ✅ Supported | Duration formatting |
| `HealthStartupInterval=` | ✅ Supported | Duration formatting (HealthStartInterval → HealthStartupInterval) |
| `PidsLimit=` | ✅ Supported | From `Container.Resources.PidsLimit` or `Container.PidsLimit` |
| `Ulimit=` | ✅ Supported | Format: `name=soft:hard` or `name=value` |
| `Sysctl=` | ✅ Supported | Format: `name=value` |
| `LogDriver=` | ✅ Supported | From `Container.Logging.Driver` |
| `LogOpt=` | ✅ Supported | From `Container.Logging.Options` |
| `Secret=` | ✅ Supported | Proper secret syntax with target/type/uid/gid/mode |
| `UserNS=` | ✅ Supported | From `Container.UserNS` |
| `PodmanArgs=` | ✅ Fallback mechanism | Used for capabilities, security options, and custom args |

#### Partially Implemented or Issues 🟡

1. **Entrypoint & Exec (Space-Joined Arrays)** 🟡
   - **Current**: Arrays joined with spaces: `Entrypoint=/foo /bar /baz`
   - **Issue**: Quadlet expects array elements but space-joining can break with arguments containing spaces
   - **Location**: `renderer.go`, lines 337-341
   - **Recommendation**: Document this limitation or enhance to properly escape arguments
   - **Impact**: Low (most entrypoints are simple), but should be noted

2. **Capability Handling** 🟡
   - **Current**: Uses `PodmanArgs` fallback for `--cap-add` and `--cap-drop`
   - **Issue**: Podman Quadlet supports native `AddCapability=` and `DropCapability=` directives (not exposed by current models)
   - **Location**: `renderer.go`, lines 440-446
   - **Impact**: Works but less elegant; should consider adding `AddCapability` and `DropCapability` to the Container model

3. **Security Options via PodmanArgs** 🟡
   - **Current**: `SecurityOpt` handled via `PodmanArgs` (line 449)
   - **Impact**: Works but Quadlet doesn't have a direct SecurityOpt field; PodmanArgs is acceptable

4. **Missing Native Directives** 🟡
   - `AddDevice=` - Not supported
   - `AddHost=` - Not supported (DNS resolution via mount/HostName)
   - `Annotation=` - Not supported (only Labels supported)
   - `AppArmor=` - Not supported (in model but not rendered)
   - `AutoUpdate=` - Not supported
   - `DNS=` - Not supported (must use network configuration)
   - `GIDMap=`/`UIDMap=` - Not supported
   - `IP=` - Static IP assignment not supported
   - `Mount=` - Type-specific mounts not fully supported
   - `NoNewPrivileges=` - Not supported
   - `Notify=` - Not supported
   - `SeccompProfile=` - Not supported
   - `ShmSize=` - Not supported
   - `StopSignal=` - Not supported
   - `StopTimeout=` - Not supported
   - `Timezone=` - Not supported
   - `ReloadCmd=` - Not supported

#### Notable Omissions 🔴

**Resource Constraints Missing**:
- `Memory=` - Not rendered from `Container.Resources.Memory`
- `MemoryReservation=` - Not rendered from `Container.Resources.MemoryReservation`
- `MemorySwap=` - Not rendered from `Container.Resources.MemorySwap`
- `CPUShares=` - Not rendered from `Container.Resources.CPUShares`
- `CPUQuota=` - Not rendered from `Container.Resources.CPUQuota`
- `CPUPeriod=` - Not rendered from `Container.Resources.CPUPeriod`

**Assessment**: The `Resources` struct exists but is completely ignored in `addResources()` (line 402). This is a **functional gap**.

---

### ✅ [Service] Section (COMPATIBLE)

The implementation uses standard systemd directives:

- **Type** ✅ - Automatically set to `oneshot` for init containers, defaults to implicit notify type
- **Restart** ✅ - Mapped from service restart policies
- **RemainAfterExit** ✅ - Used with oneshot init containers
- **WantedBy** ✅ - Set to `default.target` (reasonable default)

**Missing Service-Level Optimizations**:
- `TimeoutStartSec` - Not set (Podman docs recommend `900` for image pulls)
- `TimeoutStopSec` - Not set
- `StandardOutput=journal` / `StandardError=journal` - Could improve logging integration

**Assessment**: Functionally correct but missing production optimizations for long-running services.

---

### ✅ [Volume] Section (COMPATIBLE)

- **VolumeName=** ✅ - Correctly set
- **Driver=** ✅ - Properly rendered (skipped if "local")
- **Options=** ✅ - Map-based format correct
- **Label=** ✅ - Multiple labels with managed-by tag

**Assessment**: Well-implemented.

---

### ✅ [Network] Section (COMPATIBLE)

- **NetworkName=** ✅ - Correctly set
- **Driver=** ✅ - Properly rendered (skipped if "bridge")
- **Subnet=** ✅ - From IPAM config
- **Gateway=** ✅ - From IPAM config
- **IPRange=** ✅ - From IPAM config
- **IPv6=** ✅ - Boolean to "yes"
- **Internal=** ✅ - Boolean to "yes"
- **Options=** ✅ - Map-based format correct
- **Label=** ✅ - Multiple labels with managed-by tag
- **DNS=** ✅ - Supported via Quadlet extension (lines 658-665)
- **DisableDNS=** ✅ - Supported via Quadlet extension

**Assessment**: Excellent coverage with Quadlet-specific extensions properly integrated.

---

### ✅ [Build] Section (COMPATIBLE)

- **ImageTag=** ✅ - From Build.Tags
- **File=** ✅ - From Build.Dockerfile
- **SetWorkingDirectory=** ✅ - From Build.SetWorkingDirectory
- **Target=** ✅ - From Build.Target
- **Pull=** ✅ - Boolean mapped to "always"
- **Environment=** ✅ - From Build.Args
- **Label=** ✅ - From Build.Labels
- **Annotation=** ✅ - From Build.Annotations
- **Network=** ✅ - From Build.Networks
- **Volume=** ✅ - From Build.Volumes
- **Secret=** ✅ - From Build.Secrets
- **PodmanArgs=** ✅ - From Build.CacheFrom and Build.PodmanArgs

**Assessment**: Complete coverage of build directives.

---

### ✅ [Install] Section (COMPATIBLE)

- **WantedBy=** ✅ - Set to `default.target`

**Note**: This is reasonable for user/system services. Quadlet documentation notes that units are transient by default; enabling via `[Install]` is correct approach.

---

### ✅ [Quadlet] Section (NOT USED)

The code does NOT generate a `[Quadlet]` section, which is appropriate. The `[Quadlet]` section is optional and used for advanced options like:
- `DefaultDependencies=false`
- `PodRun=` 
- Other Quadlet-specific meta options

**Assessment**: Correct to omit (uses sane defaults).

---

## Critical Issues Identified

### 🔴 Issue 1: Memory and CPU Resource Constraints Not Rendered

**Severity**: High
**Files**: `renderer.go` lines 402-409, `service/models.go` lines 96-105

The `Resources` struct is defined and supported in service specs, but completely ignored during rendering. This means Docker Compose memory/CPU limits are silently dropped.

**Current Code**:
```go
func (r *Renderer) addResources(builder *strings.Builder, c service.Container) {
    if c.Resources.PidsLimit > 0 {
        fmt.Fprintf(builder, "PidsLimit=%d\n", c.Resources.PidsLimit)
    }
    // PidsLimit duplicated check immediately after
    // Memory, CPU fields never checked
    if len(c.Ulimits) > 0 {
        // ...
    }
}
```

**Fix Required**:
```go
// Add these to [Container] section rendering
if c.Resources.Memory != "" {
    builder.WriteString(formatKeyValue("Memory", c.Resources.Memory))
}
if c.Resources.MemoryReservation != "" {
    builder.WriteString(formatKeyValue("MemoryReservation", c.Resources.MemoryReservation))
}
if c.Resources.MemorySwap != "" {
    builder.WriteString(formatKeyValue("MemorySwap", c.Resources.MemorySwap))
}
// Note: CPU directives (CPUShares, CPUQuota, CPUPeriod) require --cpu-shares, --cpus, etc.
// May need to use PodmanArgs or add new Quadlet fields if available
```

### 🔴 Issue 2: Duplicate PidsLimit Check

**Severity**: Medium
**File**: `renderer.go`, lines 403-408

```go
if c.Resources.PidsLimit > 0 {
    fmt.Fprintf(builder, "PidsLimit=%d\n", c.Resources.PidsLimit)
}
if c.PidsLimit > 0 {
    fmt.Fprintf(builder, "PidsLimit=%d\n", c.PidsLimit)
}
```

This allows two independent PidsLimit fields to both be rendered, causing duplicate directives (invalid). The model should consolidate to one source.

---

## Important Limitations

### 1. **Entrypoint/Exec Array Escaping** 🟡
Arrays are joined with spaces without shell escaping. Example:
```
Exec=sh -c echo 'hello world'  // Breaks if original is ["sh", "-c", "echo 'hello world'"]
```

This works for simple cases but fails with spaces in arguments. Consider documenting or fixing.

### 2. **Network Connectivity Timeout** 🟡
Podman docs recommend setting `TimeoutStartSec=900` because image pulls can take a long time. Current code:
```go
builder.WriteString("\n[Service]\n")
// No TimeoutStartSec set
builder.WriteString(formatKeyValue("Restart", restart))
```

This could cause systemd to timeout during image pull on slow connections.

### 3. **Missing Security Directives** 🟡
The following are supported by Quadlet but not exposed in the model:
- `AddDevice=` - Device passthrough
- `Annotation=` - OCI annotations (exists but not rendered for containers)
- `AppArmor=` - AppArmor profile (exists in model but not rendered)
- `AutoUpdate=` - Automatic image updates
- `SeccompProfile=` - Seccomp profiles
- `StopSignal=` - Custom stop signal
- `StopTimeout=` - Stop timeout
- `Timezone=` - Timezone inside container

### 4. **Incomplete IPAM Support** 🟡
Only first IPAM config is used (line 618):
```go
if net.IPAM != nil && len(net.IPAM.Config) > 0 {
    config := net.IPAM.Config[0]  // ← Only first config
    // ...
}
```

Podman doesn't support multiple subnets per network via Quadlet, so this is correct, but should be validated/documented.

### 5. **Service Network Dependencies** 🟡
The code adds dependencies to ALL networks in the spec, even if the container doesn't use them (lines 318-330):

```go
if len(c.Network.ServiceNetworks) > 0 {
    // Uses service-specific networks ✅
} else {
    // Falls back to ALL networks in spec ⚠️ 
    networks := make([]string, 0, len(spec.Networks))
    for _, net := range spec.Networks {
        if !net.External {
            networks = append(networks, net.Name+".network")
        }
    }
    // ...
}
```

This can create unnecessary dependencies and ordering issues.

---

## Recommendations

### High Priority (Blocking Features)

1. **Implement memory/CPU resource rendering** (Issue #1)
   - Add `Memory=`, `MemoryReservation=`, `MemorySwap=` rendering
   - Research CPU constraint mapping (CPUShares, CPUQuota, CPUPeriod vs native Quadlet)
   - Add test coverage

2. **Fix duplicate PidsLimit** (Issue #2)
   - Consolidate to single source in Container model
   - Remove duplicate from Resources struct or document precedence

### Medium Priority (Production Readiness)

3. **Add TimeoutStartSec for long operations**
   ```go
   // In [Service] section rendering
   builder.WriteString("TimeoutStartSec=900\n")
   ```

4. **Document/fix Entrypoint/Exec space escaping**
   - Either enhance to properly escape arguments
   - Or document limitation in migration guide

5. **Validate/document network dependencies**
   - Consider only adding networks the service explicitly joins
   - Add validation to warn on mismatched network assignments

### Low Priority (Enhanced Features)

6. **Add support for missing directives**
   - `AddDevice=`, `Annotation=`, `AppArmor=`, `AutoUpdate=`, `SeccompProfile=`, etc.
   - Extend Container/Build/Volume/Network models as needed
   - Prioritize based on user demand

7. **Enhance logging integration**
   - Add `StandardOutput=journal`, `StandardError=journal` options
   - Consider structured logging improvements

---

## Verification Checklist

- [x] All required directives are present
- [x] Directive formats match Podman spec
- [x] Dependencies are properly chained
- [x] Section structure is valid
- [ ] All resource constraints are rendered
- [ ] TimeoutStartSec appropriately set
- [ ] Array escaping is safe
- [ ] Network dependencies are minimal

---

## Test Coverage Status

The test suite (`renderer_test.go`) covers:
- ✅ Basic container rendering
- ✅ Init container rendering (oneshot)
- ✅ Volume rendering and dependencies
- ✅ Network rendering with IPAM
- ✅ Build rendering
- ✅ Port mappings
- ✅ Environment variables
- ✅ Mounts and tmpfs

**Missing Tests**:
- [ ] Memory/CPU resource constraints
- [ ] All directive combinations
- [ ] Special characters in Entrypoint/Exec
- [ ] Missing directives from Podman spec

---

## Conclusion

The quad-ops systemd renderer is **substantially compatible** with Podman Quadlet specification. The implementation correctly handles:
- Standard systemd unit format and sections
- Container, network, volume, and build directives
- Proper dependency chaining
- Sorting for determinism
- Quadlet-specific extensions

**Critical gaps**:
- Memory/CPU resources not rendered (data loss)
- Duplicate PidsLimit validation issue

**Recommended focus**:
1. Implement memory/CPU resource rendering immediately
2. Fix duplicate PidsLimit
3. Add TimeoutStartSec for production readiness
4. Document limitations with space-joined arrays

The codebase is well-structured for adding these improvements, and no architectural changes are needed.
