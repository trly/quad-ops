# Regression Analysis: v0.21.2 → v0.22.0

**Epic**: quad-ops-hlf  
**Date**: 2025-11-09  
**Status**: Investigation Complete - 15 Regressions Identified

## Executive Summary

The v0.22.0 release introduced macOS support via a major architectural refactoring that changed the conversion pipeline from:
- **OLD**: Docker Compose → Quadlet Units → systemd (direct conversion)
- **NEW**: Docker Compose → Platform-Agnostic Specs → Platform Renderer → Lifecycle Manager

While this abstraction successfully enabled cross-platform support, it introduced **15 identified regressions** primarily related to:
1. **Dependency management** (volumes, networks, service ordering)
2. **Missing Compose feature mappings** (DNS, devices, extra_hosts)
3. **systemd reliability** (network-online.target, RequiresMountsFor)
4. **Name stability** (underscore→hyphen conversion)

## Architectural Changes

### v0.21.2 Architecture
```
internal/compose/service.go (FromComposeService)
    ↓
internal/unit/container.go (Quadlet-specific logic)
    ↓
Direct systemd unit generation
```

**Key characteristics:**
- Direct, tightly-coupled conversion
- All Compose features handled inline
- Implicit dependency resolution
- Extensive field-by-field mapping

### v0.22.0 Architecture
```
internal/compose/spec_converter.go
    ↓
internal/service/models.go (platform-agnostic)
    ↓
internal/platform/systemd/renderer.go OR
internal/platform/launchd/renderer.go
    ↓
Platform-specific lifecycle managers
```

**Key characteristics:**
- Abstraction layer for cross-platform support
- Separated concerns (parsing vs rendering)
- Explicit dependency wiring required
- Some features lost in translation

## Critical Regressions (Priority 0)

### 1. Over-broad Volume Dependencies (quad-ops-iaa)
**Problem**: All services depend on ALL project volumes, not just volumes they use.

**Root Cause**: `spec_converter.go:convertService` sets `spec.Volumes = convertProjectVolumes(project)` for every service.

**Impact**: Unnecessary ordering constraints, slow startup, potential deadlocks.

**Location**: `internal/compose/spec_converter.go` lines ~62-70

---

### 2. Missing Network Dependencies (quad-ops-p22)
**Problem**: Containers don't depend on `.network` units even when they use networks.

**Root Cause**: Renderer only adds dependencies from `Container.Network.ServiceNetworks`. If `convertNetworkMode` doesn't populate this, no dependencies are created.

**Impact**: Containers start before networks exist, causing failures.

**Location**: 
- `internal/compose/spec_converter.go` (convertNetworkMode)
- `internal/platform/systemd/renderer.go` lines ~158-182

---

### 3. External Networks Misclassified (quad-ops-lpb)
**Problem**: External networks are created with `External=false` and rendered as `.network` files.

**Root Cause**: `convertServiceNetworksList` creates Network with default driver "bridge" when network not in project.Networks.

**Impact**: Rendering unwanted network files, dependency on non-existent units.

**Location**: `internal/compose/spec_converter.go` lines ~286-300

---

### 4. Missing network-online.target (quad-ops-i12)
**Problem**: Container units don't wait for network initialization.

**Root Cause**: No `After=network-online.target` / `Wants=network-online.target` in rendered units.

**Impact**: Port binding failures, DNS issues at boot time.

**Compliance**: Violates Podman Quadlet best practices for networked containers.

**Location**: `internal/platform/systemd/renderer.go` line ~133

---

## High Priority Regressions (Priority 1)

### 5. Missing RequiresMountsFor (quad-ops-581)
**Problem**: Bind mounts don't declare mount point dependencies.

**Impact**: Container starts before NFS/network mounts are available.

**Compliance**: Violates systemd best practices for mount dependencies.

**Location**: `internal/platform/systemd/renderer.go` line ~145

---

### 6. Lost depends_on Conditions (quad-ops-gwj)
**Problem**: Compose v2 dependency conditions (service_started, service_healthy) are ignored.

**Root Cause**: `convertDependencies` doesn't parse condition field.

**Impact**: Lost ordering semantics from v0.21.2, potential race conditions.

**Location**: `internal/compose/spec_converter.go` line 68

---

### 7. Name Instability (quad-ops-ksi)
**Problem**: `SanitizeName` converts underscores to hyphens, breaking v0.21.2 compatibility.

**Impact**: 
- Renamed units break dependency edges
- Sync detects false "divergence"
- Migration path unclear

**Location**: `internal/service/validate.go` lines ~229-245

---

## Medium Priority Regressions (Priority 2)

### 8. Init Container Limitations (quad-ops-1wy)
**What was lost**: Volumes, networks, environment variables for init containers.

**Current state**: Only image/command supported.

**Use case impact**: DB migrations, schema initialization workflows broken.

---

### 9. Missing extra_hosts (quad-ops-h0f)
**v0.21.2**: Supported via `Container.AddHost` field.

**v0.22.0**: No `AddHost` field in models.go, no conversion in spec_converter.

---

### 10. Missing DNS Settings (quad-ops-4i2)
**v0.21.2**: Supported via PodmanArgs (--dns, --dns-search, --dns-opt).

**v0.22.0**: No conversion of service.DNS/DNSSearch/DNSOpts.

---

### 11. Missing Device Mappings (quad-ops-167)
**v0.21.2**: Supported via PodmanArgs (--device).

**v0.22.0**: No conversion of service.Devices.

---

### 12. Resource Constraint Audit Needed (quad-ops-5re)
**Task**: Verify convertResources + renderer.addResources fully map all Deploy.Resources fields.

**Fields to verify**:
- Memory limits/reservations
- CPU shares/quota/period
- PidsLimit
- MemorySwap

---

## Low Priority Regressions (Priority 3)

### 13. Volume nocopy Flag (quad-ops-xru)
**Missing**: nocopy option in volume mounts.

---

### 14. Bind Mount SELinux Flags (quad-ops-rqe)
**Missing**: z/Z relabel flags for bind mounts.

**Impact**: SELinux environments may have permission issues.

---

### 15. tmpfs Options (quad-ops-d4k)
**Missing**: size and other tmpfs-specific options.

---

## Analysis Sources

### Oracle Analysis
Comprehensive deep-dive analysis of refactoring changes focusing on:
- Dependency wiring logic
- External network handling
- Compose field mapping coverage
- Start order preservation

Key finding: "The v0.22.0 abstraction dropped several ordering and conversion behaviors that v0.21.2's direct Compose→Quadlet path handled implicitly."

### Librarian Analysis
Review of official Podman Quadlet documentation (podman-systemd.unit.5) revealed:
- Missing network-online.target dependencies (P0)
- Missing RequiresMountsFor for bind mounts (P1)
- Correct usage of `.network` and `.volume` suffixes (already fixed)
- Advanced directives not implemented (acceptable for v1.0)

### Git History Analysis
- v0.21.2 was last stable release
- v0.22.0 introduced via PR #46 (macOS support)
- Major package deletions:
  - `internal/unit/` (container.go, network.go, volume.go, build.go) - 10,812 lines deleted
  - `internal/compose/` old conversion logic - significant deletion
- Major package additions:
  - `internal/service/` (models, validation) - 1,115 lines
  - `internal/platform/systemd/` - 1,503 lines
  - `internal/platform/launchd/` - 2,689 lines
  - `internal/repository/` (artifacts, git) - 679 lines

**Net change**: +23,597 insertions, -10,815 deletions

---

## What's Been Attempted (Prior Work)

### Successfully Fixed Issues ✅

**External Network Handling** (quad-ops-1sn, quad-ops-6ph, quad-ops-0r1):
- Fixed external networks being incorrectly project-prefixed
- Commit 6ffe36f: Prevent project prefix on external networks
- Status: **FIXED** - External networks now work correctly

**Service-to-Network Mapping** (quad-ops-cn8):
- Restored v0.21.2 service DNS resolution
- Commit f959e8c: Fix service-to-service DNS resolution for multi-network services
- Commits 69e-4pn: Extract service-level network declarations
- Status: **FIXED** - Service networks properly mapped

**Volume/Network Dependency Suffixes** (quad-ops-eki, quad-ops-h26, quad-ops-712):
- Fixed incorrect `-volume.service` → `.volume`
- Fixed incorrect `-network.service` → `.network`
- Commit 6377d90: Correct volume and network dependency suffixes
- Removed incorrect `.volume` suffix from Volume= directive
- Status: **FIXED** - Dependency naming matches Quadlet spec

**Resource Constraints** (quad-ops-24q):
- Memory, MemoryReservation, MemorySwap rendering implemented
- CPU constraints via PodmanArgs (--cpu-quota, --cpu-shares, --cpu-period)
- Commits j0q, qgm, pmj: Implement resource constraint rendering
- Status: **MOSTLY FIXED** - Need audit (quad-ops-5re)

**Cross-Project Network Dependencies** (quad-ops-8oo):
- Removed fallback that added ALL project networks to containers
- Containers now only depend on networks they explicitly use
- Commit 1b328e7: Remove fallback network dependency logic
- Status: **FIXED** - But may have created quad-ops-p22 regression

**Name Normalization** (quad-ops-6rg):
- Implemented DNS-compliant naming (underscores → hyphens)
- Commit c25e201: Normalize unit names for DNS compliance
- Status: **PARTIAL** - Creates quad-ops-ksi compatibility issue

### Active/Related Issues 🔄

**Dependency Ordering**:
- quad-ops-kgm: Review network dependency handling for unnecessary coupling
- quad-ops-xmd: Apply dependency formatting fix to launchd
- quad-ops-5wg: Implement dependency ordering in launchd lifecycle
- quad-ops-aez: Add platform-specific dependency mapping
- quad-ops-ts0: Add dependency validation and cycle detection
- quad-ops-xd3: Test dependency handling across platforms

**Lifecycle & Systemd**:
- quad-ops-h0s: Service activation timeout handling for long-starting services
- quad-ops-22e: Race condition - systemd generator not finished when services restarted
- quad-ops-6sz: Add unit existence verification with retry (commit f50aa36)
- quad-ops-494: TimeoutStartSec=900 for image pull resilience (commit b50e2df)

**Documentation**:
- quad-ops-yx7: Document systemd Quadlet generator requirements
- quad-ops-3zb: Add diagnostic tooling to detect generator issues
- quad-ops-58x: Document CPU constraint limitations

### Key Insights from Prior Work

1. **External Networks Work Now**: The external network prefixing issue (quad-ops-lpb) was already fixed in commit 6ffe36f.

2. **Dependency Suffix Fix Created Gap**: Fixing quad-ops-eki removed the fallback network logic, which may have inadvertently created quad-ops-p22 (missing network dependencies).

3. **Name Normalization Trade-off**: Fixing underscore/hyphen consistency (quad-ops-6rg) improved DNS compliance but broke v0.21.2 compatibility (quad-ops-ksi).

4. **Service Networks Restored**: The service-to-network mapping that was lost in v0.22.0 was restored in commit f959e8c.

5. **Resource Constraints Mostly Done**: Memory/CPU rendering implemented, but needs full audit against v0.21.2.

## Recommended Fix Order

### Phase 1: Critical Dependency Fixes (P0)
1. **quad-ops-iaa** - Volume dependency scoping (NEW - not attempted)
2. **quad-ops-p22** - Network dependency wiring (REGRESSION from quad-ops-8oo fix)
3. ~~quad-ops-lpb~~ - External network classification (**FIXED** in 6ffe36f)
4. **quad-ops-i12** - network-online.target (NEW - not attempted)

**Impact**: Fixes "dependencies, sync order" issues reported by user.

### Phase 2: Reliability & Compatibility (P1)
5. quad-ops-581 - RequiresMountsFor
6. quad-ops-gwj - depends_on conditions
7. quad-ops-ksi - Name stability

**Impact**: Improves reliability and preserves v0.21.2 compatibility.

### Phase 3: Feature Parity (P2)
8. quad-ops-1wy - Init container capabilities
9. quad-ops-h0f - extra_hosts
10. quad-ops-4i2 - DNS settings
11. quad-ops-167 - Device mappings
12. quad-ops-5re - Resource audit

**Impact**: Restores missing Compose features.

### Phase 4: Edge Cases (P3)
13-15. Mount option flags

**Impact**: Handles advanced use cases.

---

## Testing Strategy

### Regression Test Suite
Create test cases covering:
- Multi-service compose with selective volume usage
- External vs managed network handling
- Service dependency chains with conditions
- Underscored service/volume/network names
- Init containers with volumes/env
- All missing Compose features

### Golden File Updates
After each fix:
1. Update golden test files in `internal/compose/testdata/golden/`
2. Verify renderer output matches expected Quadlet format
3. Run full test suite: `task test`

### Manual Validation
Deploy real-world compose files from v0.21.2 era:
1. Verify unit names match v0.21.2
2. Check dependency ordering (`systemctl list-dependencies`)
3. Validate startup sequence
4. Test rollback/restart scenarios

---

## References

- **Epic**: [quad-ops-hlf](file:///Users/trly/src/github.com/trly/quad-ops/.beads/issues.jsonl)
- **Git Tags**: v0.21.2 (working), v0.22.0 (refactoring), v0.22.3 (current)
- **Podman Docs**: https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html
- **Refactoring PR**: #46 (commit 92dd0a3)

---

## Next Steps

1. **No implementation** - per user request, investigation only
2. Review this analysis document
3. Prioritize fixes based on production impact
4. Consider creating feature branches for parallel work on P0/P1 issues
5. Update AGENTS.md with lessons learned about abstraction layer pitfalls
