# Minimal Compose Package Reference

This document shows the bare minimum needed for Compose→Quadlet conversion.

## Core Responsibility

**Input**: Docker Compose YAML projects  
**Output**: `[]service.Spec` (platform-agnostic service definitions)  
**Target Format**: Podman Quadlet units (systemd services on Linux, launchd on macOS)

## Podman Quadlet Spec Mapping

Key directives from [podman-systemd.unit.5](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html):

### Container Section
```ini
[Container]
Image=
Command=
Entrypoint=
WorkingDir=
User=
Hostname=
Environment=
EnvironmentFiles=
Exec=
ExecStartPost=

# Resources
Memory=
CPUQuota=
CPUPeriod=
CPUShares=
MemoryReservation=
MemorySwap=
PidsLimit=

# Security
Privileged=
CapAdd=
CapDrop=
SecurityLabelType=
SecurityLabelLevel=
SecurityLabelNested=

# Networking
Publish=                    # host:container
Network=
DNS=
DNSSearch=
ExtraHosts=

# Mounts
Volume=                     # source:target:options
Mount=                      # type=bind,source=/,target=/,ro=true

# Devices
Devices=                    # host:container[:permissions]
DeviceCgroupRule=

# Restart & Health
Restart=                    # no|on-failure|always|unless-stopped
HealthCmd=
HealthInterval=
HealthRetries=
HealthStartPeriod=
HealthStartupCmd=
HealthStartupInterval=
HealthStartupRetries=
HealthTimeout=

# Init & Logging
Init=
LogDriver=
LogOpt=
```

### Install Section
```ini
[Install]
WantedBy=default.target    # or multi-user.target
RequiredBy=
```

### Service Section (systemd only)
```ini
[Service]
Type=oneshot              # for init containers
After=                    # dependencies
Requires=
Wants=
```

## Minimum Conversion Logic

### What MUST Convert
1. ✅ **Image** → Container.Image
2. ✅ **Command/Entrypoint** → Container.Command/Entrypoint
3. ✅ **Ports** → Container.Ports
4. ✅ **Volumes** → Container.Mounts (bind/volume)
5. ✅ **Environment** → Container.Env
6. ✅ **Resources.limits** → Container.Resources.Memory/CPU
7. ✅ **Networks** → Container.Network (mode) + Networks (attachments)
8. ✅ **Security** → Container.Security
9. ✅ **Healthcheck** → Container.Healthcheck
10. ✅ **Restart** → Container.RestartPolicy
11. ✅ **Dependencies** → Spec.DependsOn
12. ✅ **Labels** → Container.Labels

### What CAN'T Convert (Swarm orchestration)
- ❌ `deploy.placement` - Multi-node orchestration
- ❌ `deploy.replicas > 1` - Horizontal scaling
- ❌ `deploy.update_config` - Rolling updates
- ❌ `configs/secrets` with `driver` field - Swarm store
- ❌ `ports.mode: ingress` - Swarm load balancing

Reject these with clear error messages.

### What's Optional (Custom Extensions)
- **x-quad-ops-init**: Init containers (needs systemd)
- **x-podman-env-secrets**: Export secrets as env vars
- **env_files discovery**: Auto-find .env, .env.service, service.env

## Minimal Convert.go Structure

```go
package compose

import (
    "fmt"
    "path/filepath"
    "strings"
    
    "github.com/compose-spec/compose-go/v2/types"
    "github.com/trly/quad-ops/internal/service"
)

// Converter converts Docker Compose projects to service.Spec models.
type Converter struct {
    workingDir string
}

// NewConverter creates a new Converter.
func NewConverter(workingDir string) *Converter {
    return &Converter{workingDir: workingDir}
}

// Convert converts a Docker Compose project to service specs.
func (c *Converter) Convert(project *types.Project) ([]service.Spec, error) {
    if project == nil {
        return nil, fmt.Errorf("project is nil")
    }
    
    if err := c.validateProject(project); err != nil {
        return nil, err
    }
    
    specs := make([]service.Spec, 0, len(project.Services))
    
    for serviceName, composeService := range project.Services {
        spec, err := c.convertService(serviceName, composeService, project)
        if err != nil {
            return nil, fmt.Errorf("convert service %s: %w", serviceName, err)
        }
        specs = append(specs, spec...)
    }
    
    return specs, nil
}

// Private methods organized by concern:

// convertService converts one compose service to service.Spec(s)
// Returns slice to support init containers creating separate specs
func (c *Converter) convertService(name string, svc types.ServiceConfig, proj *types.Project) ([]service.Spec, error) {
    spec := service.Spec{
        Name:        prefix(proj.Name, name),
        Description: fmt.Sprintf("Service %s from %s", name, proj.Name),
        Container:   c.convertContainer(svc, name, proj),
        Volumes:     c.convertVolumes(svc, proj),
        Networks:    c.convertNetworks(svc, proj),
        DependsOn:   c.convertDependencies(svc.DependsOn),
    }
    
    if err := spec.Validate(); err != nil {
        return nil, fmt.Errorf("validate %s: %w", name, err)
    }
    
    return []service.Spec{spec}, nil
}

// convertContainer converts compose service to service.Container
func (c *Converter) convertContainer(svc types.ServiceConfig, serviceName string, proj *types.Project) service.Container {
    container := service.Container{
        Image:           svc.Image,
        Command:         svc.Command,
        Entrypoint:      svc.Entrypoint,
        WorkingDir:      svc.WorkingDir,
        User:            svc.User,
        Hostname:        svc.Hostname,
        Env:             c.convertEnv(svc.Environment),
        EnvFiles:        c.discoverEnvFiles(serviceName),
        Ports:           c.convertPorts(svc.Ports),
        Mounts:          c.convertMounts(svc.Volumes, proj),
        Resources:       c.convertResources(svc.Deploy, svc),
        RestartPolicy:   c.convertRestart(svc.Restart),
        Healthcheck:     c.convertHealthcheck(svc.HealthCheck),
        Security:        c.convertSecurity(svc),
        Init:            svc.Init != nil && *svc.Init,
        ReadOnly:        svc.ReadOnly,
        Network:         c.convertNetworkMode(svc),
        DNS:             svc.DNS,
        DNSSearch:       svc.DNSSearch,
        ExtraHosts:      svc.ExtraHosts,
        Devices:         svc.Devices,
        StopSignal:      svc.StopSignal,
        Labels:          svc.Labels,
    }
    
    // Handle user:group parsing
    if container.User != "" && strings.Contains(container.User, ":") {
        parts := strings.SplitN(container.User, ":", 2)
        container.User = parts[0]
        container.Group = parts[1]
    }
    
    return container
}

// convertEnv converts compose env to map
func (c *Converter) convertEnv(env types.MappingWithEquals) map[string]string {
    result := make(map[string]string, len(env))
    for k, v := range env {
        if v != nil {
            result[k] = *v
        }
    }
    return result
}

// convertPorts converts compose ports to service.Port
func (c *Converter) convertPorts(ports []types.ServicePortConfig) []service.Port {
    if len(ports) == 0 {
        return nil
    }
    result := make([]service.Port, 0, len(ports))
    for _, p := range ports {
        port := service.Port{
            Host:      p.HostIP,
            HostPort:  p.PublishedPort,
            Container: uint16(p.Target),
            Protocol:  p.Protocol,
        }
        if port.Protocol == "" {
            port.Protocol = "tcp"
        }
        result = append(result, port)
    }
    return result
}

// convertMounts converts compose volumes to service.Mount
func (c *Converter) convertMounts(volumes []types.ServiceVolumeConfig, proj *types.Project) []service.Mount {
    if len(volumes) == 0 {
        return nil
    }
    result := make([]service.Mount, 0, len(volumes))
    for _, v := range volumes {
        mount := service.Mount{
            Source:   v.Source,
            Target:   v.Target,
            ReadOnly: v.ReadOnly,
            Options:  make(map[string]string),
        }
        
        // Determine mount type
        switch v.Type {
        case "bind":
            mount.Type = service.MountTypeBind
        case "volume":
            mount.Type = service.MountTypeVolume
            if v.Source != "" {
                mount.Source = prefix(proj.Name, v.Source)
            }
        case "tmpfs":
            mount.Type = service.MountTypeTmpfs
        default:
            // Auto-detect: absolute path or ./ = bind, else volume
            if filepath.IsAbs(v.Source) || strings.HasPrefix(v.Source, "./") {
                mount.Type = service.MountTypeBind
            } else {
                mount.Type = service.MountTypeVolume
                if v.Source != "" {
                    mount.Source = prefix(proj.Name, v.Source)
                }
            }
        }
        result = append(result, mount)
    }
    return result
}

// convertResources converts deploy.resources to service.Resources
func (c *Converter) convertResources(deploy *types.DeployConfig, svc types.ServiceConfig) service.Resources {
    resources := service.Resources{}
    if deploy == nil || deploy.Resources.Limits == nil {
        return resources
    }
    
    if deploy.Resources.Limits.MemoryBytes > 0 {
        resources.Memory = c.formatBytes(deploy.Resources.Limits.MemoryBytes)
    }
    if deploy.Resources.Limits.NanoCPUs > 0 {
        resources.CPUQuota, resources.CPUPeriod = c.convertCPU(deploy.Resources.Limits.NanoCPUs)
    }
    if deploy.Resources.Limits.Pids > 0 {
        resources.PidsLimit = deploy.Resources.Limits.Pids
    }
    
    return resources
}

// convertRestart converts compose restart string to service.RestartPolicy
func (c *Converter) convertRestart(restart string) service.RestartPolicy {
    switch restart {
    case "always":
        return service.RestartPolicyAlways
    case "unless-stopped":
        return service.RestartPolicyUnlessStopped
    case "on-failure":
        return service.RestartPolicyOnFailure
    default:
        return service.RestartPolicyNo
    }
}

// convertHealthcheck converts compose healthcheck to service.Healthcheck
func (c *Converter) convertHealthcheck(hc *types.HealthCheckConfig) *service.Healthcheck {
    if hc == nil || hc.Disable {
        return nil
    }
    return &service.Healthcheck{
        Test:     hc.Test,
        Retries:  int(*hc.Retries),
        Interval: time.Duration(*hc.Interval),
        Timeout:  time.Duration(*hc.Timeout),
    }
}

// convertSecurity converts compose security to service.Security
func (c *Converter) convertSecurity(svc types.ServiceConfig) service.Security {
    return service.Security{
        Privileged:  svc.Privileged,
        CapAdd:      svc.CapAdd,
        CapDrop:     svc.CapDrop,
        SecurityOpt: svc.SecurityOpt,
        GroupAdd:    svc.GroupAdd,
    }
}

// convertNetworkMode converts compose network config to service.NetworkMode
func (c *Converter) convertNetworkMode(svc types.ServiceConfig) service.NetworkMode {
    // Implementation depends on network mode (host, bridge, custom, etc)
    return service.NetworkMode{} // Placeholder
}

// convertNetworks converts service networks to []service.Network
func (c *Converter) convertNetworks(svc types.ServiceConfig, proj *types.Project) []service.Network {
    // Convert attached networks
    return nil // Placeholder
}

// convertDependencies converts depends_on to service names
func (c *Converter) convertDependencies(deps map[string]types.ServiceDependency) []string {
    if len(deps) == 0 {
        return nil
    }
    result := make([]string, 0, len(deps))
    for name := range deps {
        result = append(result, name)
    }
    return result
}

// validateProject validates Swarm features not supported
func (c *Converter) validateProject(proj *types.Project) error {
    for name, cfg := range proj.Configs {
        if cfg.Driver != "" {
            return fmt.Errorf("config %q uses driver (Swarm): use file/content/environment instead", name)
        }
    }
    for name, sec := range proj.Secrets {
        if sec.Driver != "" {
            return fmt.Errorf("secret %q uses driver (Swarm): use file/content/environment instead", name)
        }
    }
    return nil
}

// Helper functions

// prefix creates "projectName-resourceName"
func prefix(proj, resource string) string {
    return fmt.Sprintf("%s-%s", proj, resource)
}

// formatBytes converts bytes to human-readable (k, m, g)
func (c *Converter) formatBytes(b types.UnitBytes) string {
    // k, m, g conversion
    return ""
}

// convertCPU converts nanoCPUs to quota/period
func (c *Converter) convertCPU(nanoCPUs types.NanoCPUs) (int64, int64) {
    period := int64(100000)
    quota := int64(float64(nanoCPUs) * float64(period))
    return quota, period
}

// discoverEnvFiles finds .env, .env.service, service.env files
func (c *Converter) discoverEnvFiles(serviceName string) []string {
    // Return []string of discovered env files
    return nil
}
```

## Reader.go (Minimal)

```go
// ReadProjects loads compose files from directory
func ReadProjects(path string) ([]*types.Project, error) { ... }

// ParseComposeFile parses single compose file
func ParseComposeFile(path string) (*types.Project, error) { ... }
```

## Key Insight

The core logic is **simple direct mapping**:

- Compose fields → Service model fields
- Type conversions (nanoCPUs → CPU quota, bytes → human readable)
- Name prefixing (project-service naming)
- Swarm validation (reject unsupported features)
- Extension handling (init containers, env secrets)

Everything else is scaffolding and can be simplified or removed.

## Testing Strategy

**One comprehensive test file** with table-driven approach:

```go
// convert_test.go
var convertTests = []struct {
    name     string
    compose  string    // YAML input
    expected service.Spec
    wantErr  bool
}{
    {
        name: "basic image and command",
        compose: `...`,
        expected: service.Spec{...},
    },
    // ... more cases
}

func TestConverter(t *testing.T) {
    for _, tt := range convertTests {
        t.Run(tt.name, func(t *testing.T) {
            // Parse compose, convert, compare
        })
    }
}
```

This replaces 8 fragmented test files with focused, easy-to-understand tests.
