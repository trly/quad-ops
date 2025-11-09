# Regression Fix Status - v0.22.0 Investigation

**Epic**: quad-ops-hlf  
**Last Updated**: 2025-11-09  
**Total Regressions**: 15 identified, 1 already fixed

## Summary

Of the 15 regressions identified in the v0.22.0 refactoring:
- **1 already fixed** (external networks)
- **14 remain open** (dependencies, features, mount options)
- **5 have related prior work** (partial fixes or related investigations)

## Already Fixed ✅

### quad-ops-lpb: External Networks (P0)
**Status**: CLOSED - Fixed in commits 6ffe36f, quad-ops-0r1, quad-ops-6ph, quad-ops-1sn

External networks are no longer incorrectly project-prefixed. Services from different projects can now share infrastructure networks correctly.

**Before**: `llm-infrastructure-proxy.network` (incorrect)  
**After**: `infrastructure-proxy.network` (correct)

## Regressions with Prior Work (Partial/Related Fixes)

### quad-ops-p22: Missing Network Dependencies (P0)
**Status**: OPEN - Likely regression from quad-ops-8oo fix

**Prior Work**: 
- quad-ops-8oo (commit 1b328e7): Removed fallback network logic that caused cross-project errors
- quad-ops-cn8 (commit f959e8c): Restored service-to-network mapping
- quad-ops-eki (commit 6377d90): Fixed dependency suffix naming

**Issue**: The fix for quad-ops-8oo correctly removed over-broad network dependencies but may have broken default network assignment for containers without explicit networks.

**Investigation Needed**: Review interaction between convertNetworkMode, renderer fallback logic, and quad-ops-8oo changes.

---

### quad-ops-5re: Resource Constraint Audit (P2)
**Status**: OPEN - Partial implementation exists

**Prior Work**:
- quad-ops-24q: Identified missing resource rendering
- Commits j0q, qgm, pmj: Implemented Memory/CPU rendering
- Memory, MemoryReservation, MemorySwap → Quadlet directives
- CPU constraints → PodmanArgs (--cpu-quota, --cpu-shares, --cpu-period)

**Remaining**: Audit completeness against v0.21.2, verify MemorySwap, pids_limit coverage.

---

### quad-ops-ksi: Name Instability (P1)
**Status**: OPEN - Conflicts with prior fix

**Prior Work**:
- quad-ops-6rg (commit c25e201): Normalized names (underscores → hyphens)
- Improved DNS compliance
- Fixed underscore/hyphen mismatches in volume references

**Conflict**: DNS compliance fix broke v0.21.2 compatibility. Services/volumes with underscores now have different names.

**Resolution Options**:
1. Feature flag: `preserve_v021_names` (default: false)
2. Migration tool: rename-detector + user prompt
3. Document as breaking change with migration guide

---

### quad-ops-gwj: depends_on Conditions (P1)
**Status**: OPEN - Related to ongoing dependency work

**Related Issues**:
- quad-ops-ts0: Add dependency validation and cycle detection
- quad-ops-5wg: Implement dependency ordering in launchd
- quad-ops-aez: Platform-specific dependency mapping
- quad-ops-xd3: Test dependency handling

**Gap**: Compose v2 dependency conditions (service_started, service_healthy) not parsed or preserved.

---

### quad-ops-iaa: Over-broad Volume Dependencies (P0)
**Status**: OPEN - Not directly addressed, but related work exists

**Related Fixes**:
- quad-ops-h26 (commit 9d53372): Removed incorrect `.volume` suffix
- quad-ops-712: Fixed volume name escaping
- quad-ops-3kb: Fixed underscore vs hyphen mismatch

**Gap**: All services still depend on ALL project volumes, not just volumes they use.

## Remaining Regressions (No Prior Work)

### Priority 0 (Critical)
- **quad-ops-i12**: Missing network-online.target dependencies (NEW)

### Priority 1 (High)
- **quad-ops-581**: Missing RequiresMountsFor directives (NEW)

### Priority 2 (Medium)
- **quad-ops-1wy**: Init container limitations (volumes/networks/env dropped)
- **quad-ops-h0f**: Missing extra_hosts (AddHost)
- **quad-ops-4i2**: Missing DNS settings (dns/dns_search/dns_opt)
- **quad-ops-167**: Missing device mappings

### Priority 3 (Low)
- **quad-ops-xru**: Missing volume nocopy flag
- **quad-ops-rqe**: Missing bind mount z/Z flags
- **quad-ops-d4k**: Missing tmpfs size options

## Systemd/Lifecycle Issues (Related Context)

While investigating regressions, several systemd-specific issues were identified:

- **quad-ops-22e**: Race condition - generator not finished when services restart
- **quad-ops-6sz**: Unit existence verification with retry (commit f50aa36)
- **quad-ops-494**: TimeoutStartSec=900 for image pulls (commit b50e2df)
- **quad-ops-h0s**: Service activation timeout handling
- **quad-ops-yx7**: Document Quadlet generator requirements
- **quad-ops-3zb**: Add diagnostic tooling for generator issues

These are operational issues, not regressions from v0.22.0, but relevant for overall reliability.

## Recommended Approach

### Phase 1: Resolve Inter-Dependencies
1. Investigate quad-ops-p22 interaction with quad-ops-8oo fix
2. Decide on quad-ops-ksi resolution (feature flag vs breaking change)
3. Complete quad-ops-5re audit using v0.21.2 as reference

### Phase 2: Critical Fixes
4. quad-ops-iaa: Volume dependency scoping
5. quad-ops-i12: network-online.target
6. quad-ops-581: RequiresMountsFor

### Phase 3: Dependency Completeness
7. quad-ops-gwj: depends_on conditions
8. Validate with existing dependency work (ts0, 5wg, aez, xd3)

### Phase 4: Feature Parity
9. quad-ops-1wy: Init container capabilities
10. quad-ops-h0f, 4i2, 167: Missing Compose features

### Phase 5: Mount Options
11. quad-ops-xru, rqe, d4k: Volume/bind mount options

## Git Archaeology Notes

**Key Commits to Review**:
- `92dd0a3`: macOS refactor (v0.22.0)
- `f959e8c`: Restore service-to-service DNS
- `6377d90`: Fix volume/network suffixes
- `6ffe36f`: Fix external network prefixing
- `1b328e7`: Remove fallback network logic
- `c25e201`: Normalize unit names
- `9d53372`: Remove `.volume` suffix

**Deleted Code to Review**:
- `internal/unit/container.go`: Direct Compose→Quadlet conversion (965 lines)
- `internal/compose/service.go`: Old service processing (238 lines)
- Look for: network handling, volume mounting, dependency logic, Compose feature mappings

## Next Steps

1. **Review git history**: Compare v0.21.2 network/volume/dependency code with v0.22.0
2. **Run examples**: Deploy v0.21.2-era compose files with current code to identify runtime gaps
3. **Prioritize fixes**: Focus on P0 issues that block multi-service deployments
4. **Test continuously**: Each fix should include regression tests against v0.21.2 behavior
5. **Document changes**: Update AGENTS.md with lessons learned about abstraction pitfalls

## References

- Full analysis: [regression-analysis-v0.22.0.md](file:///Users/trly/src/github.com/trly/quad-ops/history/regression-analysis-v0.22.0.md)
- Epic: quad-ops-hlf
- Related discussions: https://github.com/trly/quad-ops/discussions/52
