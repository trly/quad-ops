# Phase 1: Consolidation Plan - spec_converter.go (48 → 12 methods)

## Current State (48 methods)
- 1 public: `ConvertProject`
- 47 private helpers split across multiple categories

## Target State (12 methods)
1. `ConvertProject` - Main entry point (public)
2. `convertService` - Service-level conversion
3. `convertContainer` - Container config conversion
4. `convertVolumeMounts` - Volume mount conversion
5. `convertConfigMounts` - Config mount conversion
6. `convertSecretMounts` - Secret mount conversion
7. `convertServiceNetworks` - Service network conversion
8. `convertServiceVolumes` - Service volume conversion
9. `convertProjectNetworks` - Project network conversion
10. `convertProjectVolumes` - Project volume conversion
11. `convertInitContainers` - Init container conversion (keep for special Linux logic)
12. `convertFileObjectToMount` - File object mount conversion

## Consolidation Strategy

### Remove (consolidate inline or move to helpers):
- Type converters (should be functions, not methods):
  - `convertEnvironment`, `convertBuildArgs`, `convertLabels` → Inline in callers
  - `convertPorts`, `convertTmpfs`, `convertExtraHosts` → Inline in `convertContainer`
  - `convertDNS`, `convertDNSSearch`, `convertDNSOpts`, `convertDevices`, `convertDeviceCgroupRules` → Inline
  - `convertUlimits` → Inline in `convertContainer`
  - `convertDuration` → Direct time.Duration conversion (trivial)

- Simple field converters (consolidate with context):
  - `convertRestartPolicy`, `convertHealthcheck`, `convertSecurity`, `convertLogging` → Inline in `convertContainer`
  - `convertResources`, `convertCPU`, `convertCPUShares`, `convertBuild` → Consolidate in `convertContainer` section

- Network/dependency converters (consolidate):
  - `convertNetworkMode` → Inline in network section
  - `convertServiceNetworksList`, `convertServiceNetworks` → Single method
  - `convertDependencies`, `convertIPAM` → Inline

- Data extraction helpers:
  - `getStringFromMap`, `getMapFromMap`, `getStringSliceFromMap` → Keep as package functions (useful)
  - `convertEnvFiles`, `convertEnvSecrets` → Keep as is

- Utility functions (move to helpers.go):
  - `ensureProjectTempDir`, `writeTempFile` → Potential for helpers.go
  - `formatBytes` → Convert to function, move to helpers
  - `validateProject` → Consolidate into ConvertProject or move to helpers

- Parsing/conversion:
  - `convertInitVolumeMounts` → Inline in convertInitContainers
  - `convertExternalSecrets` → Consolidate with convertSecretMounts

## Implementation Steps

1. Consolidate primitive type converters inline in their callers
2. Consolidate related converters where they're used
3. Keep only high-level, reusable conversion methods
4. Move utility functions (formatBytes, ensure/write temp) to helpers or inline
5. Delete unused interfaces (Repository, SystemdManager, FileSystem)
6. Run all tests to verify equivalence

## Code Density Goals
- Remove ~30% of lines through consolidation
- Maintain 100% test coverage
- Ensure byte-for-byte identical outputs
