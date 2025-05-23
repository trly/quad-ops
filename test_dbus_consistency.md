# SystemD DBus vs Exec.Command Consistency Improvement

## Changes Made

### Before
The `getUnitFailureDetails()` function used inconsistent approaches:
- Most systemd operations used the dbus library (`conn.StartUnitContext`, `conn.StopUnitContext`, etc.)
- Unit failure details used `exec.Command()` calls:
  - `systemctl --user status` for unit status
  - `journalctl --user-unit` for logs

### After
Now consistently uses dbus for all systemd property access:
- **Unit Status**: Retrieved via `conn.GetUnitPropertiesContext()` instead of `systemctl status`
- **Detailed Properties**: LoadState, ActiveState, SubState, Result, MainPID, ExecMainStatus
- **Logs**: Still uses `journalctl` (only remaining exec.Command) because systemd dbus doesn't expose log retrieval

### Benefits
1. **Consistency**: All systemd operations now use the same dbus interface
2. **Performance**: dbus calls are more efficient than spawning processes
3. **Reliability**: No dependency on external executable paths
4. **Better Error Handling**: Structured property access vs parsing text output
5. **User/System Mode**: Automatic handling via `getSystemdConnection()`

### Properties Now Available via DBus
- `LoadState`: Whether unit file is loaded
- `ActiveState`: Current activation state (active, failed, etc.)
- `SubState`: Detailed state (running, dead, failed, etc.)
- `Result`: Result of last operation (success, exit-code, etc.)
- `MainPID`: Process ID of main service process
- `ExecMainStatus`: Exit code of main process
- `Description`: Human-readable description
- `FragmentPath`: Path to unit file

### Remaining exec.Command Usage
- `journalctl` for log retrieval (necessary - dbus doesn't provide log access)
- `systemctl --version` in validation (appropriate for version checking)
- Command runner interface in validation (testable abstraction)

This change improves consistency while maintaining necessary functionality where dbus limitations exist.