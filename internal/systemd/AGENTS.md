# systemd Package Agent Guidelines

This directory converts Docker Compose YAML to systemd Quadlet unit files using `gopkg.in/ini.v1`.

## Key Concept: AllowShadows for Repeated Keys

Systemd unit files require repeated keys for directives that appear multiple times:

```ini
[Container]
Volume=/host/path:/container/path:rw
Volume=/another/path:/another/container/path:rw
Network=frontend
Network=backend
```

NOT indexed keys (which are incorrect for systemd):
```ini
Volume.0=/host/path:/container/path:rw
Volume.1=/another/path:/another/container/path:rw
Network.0=frontend
Network.1=backend
```

## Proper Usage of AllowShadows

### 1. Enable AllowShadows When Creating Files

Always create ini.File objects with shadow support enabled:

```go
file := ini.Empty(ini.LoadOptions{AllowShadows: true})
```

This is required for `Key.AddShadow()` to work correctly.

### 2. Building Unit Files with Repeated Keys

Use two separate data structures in your builder functions:

1. **sectionMap** - For single-valued keys:
   ```go
   sectionMap["Image"] = "nginx:latest"
   sectionMap["HostName"] = "myhost"
   ```

2. **shadowMap** - For repeated keys (map of key name to slice of values):
   ```go
   shadowMap["Volume"] = []string{"data:/data:rw", "/host/path:/container/path:rw"}
   shadowMap["Network"] = []string{"frontend", "backend"}
   ```

### 3. Writing Shadow Keys to INI Section

```go
// First add single-valued keys
for key, value := range sectionMap {
    _, _ = section.NewKey(key, value)
}

// Then add shadow keys
for key, values := range shadowMap {
    if len(values) == 0 {
        continue
    }
    k, _ := section.GetKey(key)
    if k == nil {
        k, _ = section.NewKey(key, values[0])
    } else {
        k.SetValue(values[0])
    }
    for _, v := range values[1:] {
        if err := k.AddShadow(v); err != nil {
            // Ignore errors - they should not occur in normal operation
            continue
        }
    }
}
```

## Testing Shadow Keys

### Reading Shadow Values

Use `Key.ValueWithShadows()` to retrieve all values including shadows:

```go
func getValues(unit Unit, key string) []string {
    section := unit.File.Section("Container")
    if section == nil {
        return []string{}
    }
    k := section.Key(key)
    if k == nil {
        return []string{}
    }
    return k.ValueWithShadows()
}
```

### Testing Repeated Keys

```go
func TestMultipleVolumes(t *testing.T) {
    svc := &types.ServiceConfig{
        Image: "nginx:latest",
        Volumes: []types.ServiceVolumeConfig{
            {Type: "volume", Source: "data", Target: "/data"},
            {Type: "bind", Source: "/host/path", Target: "/container/path"},
        },
    }
    unit := BuildContainer("web", svc)

    vals := getValues(unit, "Volume")
    assert.Len(t, vals, 2)
    assert.Contains(t, vals[0], "data")
    assert.Contains(t, vals[1], "/host/path")
}
```

### Verifying Output Format

To verify the generated unit file produces correct systemd syntax:

```go
var buf bytes.Buffer
unit.File.WriteTo(&buf)
// buf.String() will contain:
// [Container]
// Volume = data:/data:rw
// Volume = /host/path:/container/path:rw
```

## Fields Using Shadow Keys in container.go

The following compose fields are rendered as repeated systemd directives (use shadows):

- **AddCapability** - Additional Linux capabilities
- **AddDevice** - Device mappings
- **AddHost** - Host entries for /etc/hosts
- **DropCapability** - Dropped Linux capabilities
- **DNS** - DNS servers
- **DNSSearch** - DNS search domains
- **DNSOption** - DNS query options
- **EnvironmentFile** - Environment variable files
- **ExposeHostPort** - Exposed ports
- **PublishPort** - Published ports (host:container)
- **Tmpfs** - Tmpfs mount points
- **Volume** - Mount volumes and bind mounts
- **Network** - Service networks (in priority order)
- **Group** - Additional groups

## Metadata Fields vs Repeated Directives

### Do NOT Use Shadows For:

These are metadata fields that serialize multiple key=value pairs using dot notation:

- **Label** (Label.key=value) - OCI container/network labels
- **Sysctl** (Sysctl.name=value) - Sysctl settings
- **LogOpt** (LogOpt.key=value) - Logging options
- **Ulimit** (Ulimit.name=limits) - Resource limits
- **UIDMap/GIDMap** - User/group ID mappings
- **IPAM Subnet/Gateway/IPRange** (in network.go) - IPAM pool configuration

These use Quadlet's metadata serialization design with dot notation for readability.

### Do Use Shadows For:

These are actual repeated systemd directives that should appear multiple times:

- Any key explicitly documented as "This key can be listed multiple times" in Podman Quadlet docs
- Keys like Volume=, Network=, AddCapability=, Environment=, etc. listed above
- **Environment** (Environment=KEY=value) - Per Podman docs: "can be listed multiple times"

### Example:

**CORRECT - Metadata (use dot notation):**
```ini
[Container]
Label.app=myapp
Label.version=1.0
```

**CORRECT - Repeated Directives (use shadows):**
```ini
[Container]
Volume=/data:/data:rw
Volume=/host:/container:ro
Network=frontend
Network=backend
Environment=FOO=bar
Environment=BAZ=qux
```

**INCORRECT - Don't mix them:**
```ini
[Container]
Volume.0=/data:/data:rw          # WRONG - Volume should use shadows, not dots
Label=app=myapp                  # WRONG - Label should use dots, not shadows
Environment.FOO=bar              # WRONG - Environment should use shadows, not dots
```

## Service Dependencies ([Unit] Section)

The generator emits `[Unit]` section with `Requires=` and `After=` directives based on service dependencies from `depends_on`.

### Dependency Mapping

| Dependency Type | Directive |
|-----------------|-----------|
| Internal (same project) | `Requires=` + `After=` |

### Unit Naming Convention

Dependencies reference systemd service units using the pattern: `{project}-{service}.service`

### Example Output

```ini
[Unit]
Requires=myproject-db.service
After=myproject-db.service

[Container]
Image=myapp:latest
...
```

This allows `systemctl start myproject-web.service` to automatically start all required dependencies via systemd's native ordering.

## Restart Policy

The `Restart` directive belongs in the `[Service]` section, NOT the `[Container]` section.
Docker Compose restart policies are mapped to systemd equivalents:

| Compose | systemd |
|---------|---------|
| no | no |
| always | always |
| on-failure | on-failure |
| unless-stopped | always |

## Linting Considerations

When using `AddShadow()`, remember to handle the error return value:

```go
if err := k.AddShadow(value); err != nil {
    continue  // or handle appropriately
}
```

staticcheck will also suggest optimizations like:
```go
// Instead of:
for _, netName := range networks {
    shadows["Network"] = append(shadows["Network"], netName)
}

// Use:
shadows["Network"] = append(shadows["Network"], networks...)
```

## Missing Implementations (Per Podman Quadlet Docs)

These Podman Quadlet features are documented as repeatable but not yet implemented:

### Container Section
- **Annotation** - OCI annotations (different from Labels) - per docs: "This key can be listed multiple times"
- **Mount** - Advanced mount options (distinct from Volume) - per docs: "This key can be listed multiple times"
- **Secret** - Container secrets - per docs: "This key can be listed multiple times" (Podman 4.5+)
- **PodmanArgs** - Currently using indexed notation (GlobalArgs.0, .1, .2) but should use shadows per docs: "This key can be listed multiple times"

### Network Section
- **IPRange** - When multiple IP ranges needed - per docs: "This key can be listed multiple times"

### Priority Fixes
1. **Convert PodmanArgs to shadows** (high impact, wrong syntax currently)
2. **Implement Annotation support** (missing feature)
3. **Implement Mount support** (missing feature)
4. **Add IPRange shadow support** (missing feature)

## Validation Using podman-systemd-generator

To verify changes to unit file rendering, use `podman-systemd-generator` to validate the generated quadlet units:

### Workflow

1. Run the `gen-units` task to generate quadlet units:
   ```bash
   task gen-units
   ```
   This generates `.container`, `.network`, `.volume`, and `.kube` files in `build/generated-units/`

2. Validate generated units with podman-systemd-generator:
   ```bash
   podman-systemd-generator -f -r build/generated-units/
   ```

3. Review generated systemd unit files in the output directory

4. Verify services start correctly with generated units

### Directory Structure

- **Input**: `build/generated-units/` - Generated quadlet files (`.container`, `.network`, `.volume`, `.kube`)
- **Output**: `build/generated-units/*.service` - Generated systemd units from podman-systemd-generator

This directory is ephemeral and should not be committed to version control.

## References

- [gopkg.in/ini.v1 Documentation](https://pkg.go.dev/gopkg.in/ini.v1)
- [Systemd Unit File Format](https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html)
- [Podman Quadlet Documentation](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
