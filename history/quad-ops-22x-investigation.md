# quad-ops-22x: External Network Dependency Translation Issue

## Issue Summary
**ID**: quad-ops-22x
**Status**: open
**Priority**: High (P1)
**Title**: Fix external network dependency translation in Quadlet generator

The Quadlet generator fails with "unable to translate dependency" errors when services reference external networks from other projects.

## Problem Description

Services in different projects (e.g., `dozzle`, `llm`, `scrutiny`, `beszel`) are trying to share a single external network called `infrastructure-proxy` from an infrastructure project. However, the current code incorrectly applies the current project's prefix to external networks.

### Error Example
```
quadlet-generator: converting "dozzle-web.container": unable to translate dependency for dozzle-infrastructure-proxy.network
quadlet-generator: converting "llm-ollama.container": unable to translate dependency for llm-infrastructure-proxy.network
```

### Expected Behavior
- Compose file declares: `infrastructure-proxy` (external network from infrastructure project)
- Should reference: `infrastructure-proxy.network` (actual network unit)
- Should NOT reference: `dozzle-infrastructure-proxy.network` (with project prefix)

### Actual Behavior
- Code applies wrong prefix: `dozzle-infrastructure-proxy` (adds current project prefix)
- Quadlet looks for non-existent unit: `dozzle-infrastructure-proxy.network`

## Root Cause

**Location**: `internal/compose/spec_converter.go`, function `convertServiceNetworksList` (lines 900-950)

The issue is in handling of external networks in `convertServiceNetworksList`:

```go
// Line 908-922: When network not found in project.Networks
projectNet, exists := project.Networks[networkName]
if !exists {
    // Network declared by service but not in project networks
    // This can happen with external networks (from other projects)
    // Use network name as-is without applying current project prefix
    resolvedName := service.SanitizeName(networkName)
    // ... correctly uses network name as-is
    continue
}
```

**The bug**: The logic looks correct here - it uses `service.SanitizeName(networkName)` directly without prefixing when the network isn't found in `project.Networks`.

However, checking the tests and issue description more carefully:

## Actual Root Cause (Corrected)

Looking at the test case at line 856-872 of `spec_converter_test.go`:
```go
name:        "service with external network from different project (not in project.Networks)",
// ...
// Should NOT prefix with current project name
// The external network is from another project and should be used as-is
assert.Equal(t, service.SanitizeName("infrastructure-proxy"), spec.Networks[0].Name)
```

The test expects: `infrastructure-proxy` 
But the code is currently producing: `llm-infrastructure-proxy` (with project prefix)

## Critical Code Path: convertServiceNetworks

The issue is that there are TWO separate functions handling networks:

1. **`convertServiceNetworks()`** (line 889) - Calls either `convertServiceNetworksList` or `convertProjectNetworks`
2. **`convertProjectNetworks()`** (line 522) - Handles project-level network definitions

For **external networks declared in compose service** but **not in project.Networks**, the `convertServiceNetworksList` function should handle them. Looking at line 913:

```go
resolvedName := service.SanitizeName(networkName)
```

This is CORRECT - it doesn't apply a prefix.

## Diagnosis Needed

The actual bug needs verification by:

1. **Checking compose files**: Verify how external networks are declared (e.g., are they in `project.Networks` with `external: true` or just in service `networks:` declarations?)

2. **Tracing the code path**: 
   - If external network IS in `project.Networks` with `external: true`, then it goes through `convertServiceNetworksList` line 576-582 (checks IsExternal and correctly doesn't prefix)
   - If external network is NOT in `project.Networks`, it goes through line 568-573 (also correctly doesn't prefix)

3. **The actual problem**: The bug might be that external networks ARE being defined in `project.Networks` with `external: true`, but then the code at line 929 is still applying a prefix:

```go
// Don't apply project prefix to external networks
if !IsExternal(projectNet.External) && !strings.Contains(resolvedName, project.Name) {
    sanitizedName = service.SanitizeName(Prefix(project.Name, resolvedName))
}
```

Wait - this looks correct too. It checks `!IsExternal()` before applying prefix.

## Most Likely Issue

The problem is probably that external networks are being marked as external in the compose file, but when they're NOT found in `project.Networks`, the code path (line 913) is being used, which should be correct.

**The real issue**: The systemd renderer (not the spec converter) is receiving the correct network names but is then adding project prefixes when rendering dependencies.

Actually, reviewing the test at line 856-872 again - this test PASSES. So the spec converter is working correctly.

## Network Test Evidence

Looking at `network_dependencies_test.go` line 143-191 (`TestNetworkDependencies_ExternalNetworksInServiceNetworks`):

```go
// Project has both local and external networks
project := &types.Project{
    Name: "myapp",
    Networks: map[string]types.NetworkConfig{
        "default": {...},
        "infrastructure-proxy": {
            External: types.External(true),  // Marked as external
        },
    },
    Services: {...
        Networks: {
            "default": {},
            "infrastructure-proxy": {},  // Service uses both networks
        },
    },
}

// Expected output:
// ServiceNetworks should be: ["infrastructure-proxy", "myapp-default"]
// NOT: ["myapp-infrastructure-proxy", "myapp-default"]
```

This test **PASSES**, indicating the spec converter is correctly NOT prefixing external networks.

## The Real Issue

The problem is likely **not in `convertServiceNetworksList`** but rather:

1. **Different code path**: External networks might be declared in the compose file WITHOUT being in `project.Networks` at all
2. **Systemd renderer**: The renderer might be applying prefixes when rendering After/Requires directives
3. **Name mismatch**: There's a name mismatch issue filed as quad-ops-782

## Actual Root Cause (from quad-ops-782)

The core issue is a **name mismatch between storage and retrieval**:

- `spec.Networks` (actual network definitions) → rendered with one naming convention
- `spec.Container.Network.ServiceNetworks` (network references) → might use different naming

For external networks from other projects:
- Name comes from compose service declaration (e.g., `infrastructure-proxy`)
- Should stay as `infrastructure-proxy` in ServiceNetworks
- But when spec.Networks is populated, might apply different rules

## Current Code Status

The code at lines 576-582 in `convertNetworkMode` handles this:
```go
if IsExternal(projectNet.External) {
    // External network from another project - use as-is
    resolvedName := service.SanitizeName(networkName)
    mode.ServiceNetworks.append(mode.ServiceNetworks, resolvedName)
    continue
}
```

This is **CORRECT** and **TESTED**.

## The Actual Fix Needed

Review the issue description again - it says the bug is in `convertServiceNetworksList` when handling external networks **that aren't in project.Networks**. 

Looking at lines 568-573:
```go
if !exists {
    // External or undefined network - use as-is with sanitization
    // Don't apply current project prefix to external networks
    resolvedName := service.SanitizeName(networkName)
    mode.ServiceNetworks.append(mode.ServiceNetworks, resolvedName)
    continue
}
```

This already looks correct. The code:
1. Checks if network NOT in project.Networks
2. Uses `service.SanitizeName(networkName)` WITHOUT prefixing
3. Adds to ServiceNetworks

## Hypothesis: The Bug is Already Fixed

The code review suggests this bug may already be fixed. The test suite passes. Need to:

1. Verify the tests actually cover the failure scenario
2. Check if there's a different code path that's broken
3. Look at actual error messages from a real quadlet conversion
