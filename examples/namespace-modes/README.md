# Namespace Modes Example

This example demonstrates support for Docker Compose PID, IPC, and cgroup namespace mode fields in quad-ops.

## Supported Namespace Modes

### PID Namespace (`pid`)

Controls how the container's PID namespace is configured:

- `host` - Use the host's PID namespace
- `service:name` - Share PID namespace with another service
- `container:name` - Share PID namespace with a specific container

**Example:**
```yaml
services:
  app:
    image: docker.io/library/alpine:latest
    pid: host
```

### IPC Namespace (`ipc`)

Controls inter-process communication namespace:

- `host` - Use the host's IPC namespace
- `shareable` - Container's IPC namespace can be shared
- `container:name` - Share IPC namespace with a specific container

**Example:**
```yaml
services:
  app:
    image: docker.io/library/alpine:latest
    ipc: shareable
```

### Cgroup Namespace (`cgroup`)

Controls cgroup namespace mode:

- `host` - Use the host's cgroup namespace
- `private` - Use a private cgroup namespace

**Example:**
```yaml
services:
  app:
    image: docker.io/library/alpine:latest
    cgroup: private
```

## How It Works

### Conversion
The fields are converted from Docker Compose to the service specification in `spec_converter.go`:
- `service.Pid` → `Container.PidMode`
- `service.Ipc` → `Container.IpcMode`
- `service.Cgroup` → `Container.CgroupMode`

### Rendering

#### systemd (Quadlet)
Since Quadlet doesn't have native directives for these namespace modes, they are rendered using `PodmanArgs`:
- `--pid=<mode>` for PID namespace
- `--ipc=<mode>` for IPC namespace
- `--cgroupns=<mode>` for cgroup namespace

**Example output:**
```ini
[Container]
Image=alpine:latest
PodmanArgs=--pid=host
PodmanArgs=--ipc=shareable
PodmanArgs=--cgroupns=private
```

#### launchd (macOS)
On macOS, the flags are added directly to the podman run command in `BuildPodmanArgs()`:
```bash
podman run --pid host --ipc shareable --cgroupns private alpine:latest
```

## Usage

1. Define your namespace modes in `compose.yml`
2. Run `quad-ops sync` to generate Quadlet units or launchd plists
3. The namespace modes will be properly configured for your platform

## Security Considerations

- Using `pid: host` gives the container access to all processes on the host
- Using `ipc: host` allows the container to access host IPC resources
- Using `cgroup: host` allows the container to view and potentially manipulate host cgroups
- These modes should only be used when necessary and with trusted containers
