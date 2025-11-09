# Device Cgroup Rules Example

This example demonstrates how to use Docker Compose `device_cgroup_rules` field for fine-grained cgroup device permissions.

## Overview

Device cgroup rules allow you to control which devices a container can access without explicitly mounting them. This is useful for:

- Allowing access to device classes (e.g., all input devices)
- Providing read-only access to specific devices
- Dynamically allowing devices that may appear/disappear

## Format

Device cgroup rules use the format: `type major:minor permissions`

- **type**: Device type
  - `a` - All devices (block and character)
  - `b` - Block devices (disks, etc.)
  - `c` - Character devices (input devices, etc.)
  
- **major:minor**: Device numbers
  - Use specific numbers (e.g., `13:64`)
  - Use wildcards (e.g., `13:*` for all devices with major 13)
  - Use `*:*` to match all devices
  
- **permissions**: Access permissions
  - `r` - Read
  - `w` - Write
  - `m` - Create (mknod)

## Common Device Major Numbers

- `13` - /dev/input/* (input devices: mice, keyboards, joysticks)
- `8` - /dev/sd* (SCSI disk devices)
- `189` - /dev/bus/usb/* (USB devices)
- `226` - /dev/dri/* (DRM/GPU devices)

## Examples

### Allow all input devices
```yaml
device_cgroup_rules:
  - 'c 13:* rmw'
```

### Allow multiple device types
```yaml
device_cgroup_rules:
  - 'c 13:* rmw'     # Input devices
  - 'b 8:* rmw'      # SCSI disks
  - 'c 189:* rmw'    # USB devices
```

### Read-only access to specific device
```yaml
device_cgroup_rules:
  - 'c 13:64 r'
```

## Rendering

quad-ops renders device_cgroup_rules using `PodmanArgs` on both platforms:

### systemd (Linux)
```ini
[Container]
PodmanArgs=--device-cgroup-rule=c 13:* rmw
```

### launchd (macOS)
```
--device-cgroup-rule c 13:* rmw
```

## Testing

1. Sync the compose file:
   ```bash
   cd examples/device-cgroup-rules
   quad-ops sync
   ```

2. Start the service:
   ```bash
   quad-ops up input-device-app
   ```

3. Verify the container can access input devices (Linux only):
   ```bash
   podman exec device-cgroup-rules-input-device-app-1 ls /dev/input
   ```

## Note

Device cgroup rules are a Linux kernel feature. While quad-ops supports them on macOS, they may not have any effect depending on the Podman machine configuration.
