# Test Plan for Restart/Cleanup Order Fix

## Issue
When changing quad-ops configuration from having repositories to `repositories: []`, units fail to restart due to dependency conflicts between cleanup and restart operations.

## Root Cause
The original sequence was:
1. Process new units → create unit files
2. Restart changed units → systemd reload + restart (while old units still exist)
3. Cleanup orphaned units → remove old unit files + systemd reload

This caused dependency conflicts because restart operations happened while stale dependencies existed.

## Fix Applied
Changed the sequence to:
1. Process new units → create unit files  
2. **Cleanup orphaned units FIRST** → remove old unit files + systemd reload
3. Wait 1 second for systemd to process removals
4. Restart changed units → systemd reload + restart (old units now gone)

Additional improvements:
- Added LoadState check before restart to avoid attempting restart on non-loaded units
- Added better error handling for unit existence checks

## Test Instructions
1. Start with a working config with repositories
2. Run `./quad-ops sync -u -v` to verify working state
3. Change config to `repositories: []`
4. Run `./quad-ops sync -u -v` again
5. Verify no "dependency" or "failed" restart errors occur
6. Verify orphaned units are cleanly removed

Expected behavior: Clean transition with no restart failures.