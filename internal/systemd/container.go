package systemd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/config"
	"gopkg.in/ini.v1"
)

// BuildContainer converts a compose service into a container unit file.
// projectNetworks provides the project-level network configs so that external
// networks can be referenced by name rather than as Quadlet unit files.
func BuildContainer(projectName, serviceName string, svc *types.ServiceConfig, projectNetworks types.Networks, projectVolumes types.Volumes) Unit {
	file := ini.Empty(ini.LoadOptions{AllowShadows: true})
	section, _ := file.NewSection("Container")
	sectionMap := make(map[string]string)
	shadowMap := make(map[string][]string) // For keys with repeated values
	buildContainerSection(projectName, serviceName, svc, sectionMap, shadowMap, projectNetworks, projectVolumes)

	// Copy sectionMap to ini section
	for key, value := range sectionMap {
		_, _ = section.NewKey(key, value)
	}

	// Add shadow keys for repeated directives
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
				// Ignore errors adding shadows, they should not occur in normal operation
				continue
			}
		}
	}

	// Add [Service] section for restart policy
	if svc.Restart != "" {
		serviceSection, _ := file.NewSection("Service")
		_, _ = serviceSection.NewKey("Restart", mapRestartPolicy(svc.Restart))
	}

	// Add [Unit] section for service dependencies
	buildUnitSection(file, projectName, svc)

	// Add [Install] section so the unit starts on boot
	installSection, _ := file.NewSection("Install")
	if config.IsUserMode() {
		_, _ = installSection.NewKey("WantedBy", "default.target")
	} else {
		_, _ = installSection.NewKey("WantedBy", "multi-user.target")
	}

	return Unit{
		Name: fmt.Sprintf("%s-%s.container", projectName, serviceName),
		File: file,
	}
}

// buildUnitSection adds the [Unit] section with Requires/After directives
// based on intra-project service dependencies from depends_on.
func buildUnitSection(file *ini.File, projectName string, svc *types.ServiceConfig) {
	deps, ok := svc.Extensions["x-quad-ops-dependencies"].(map[string]string)
	if !ok || len(deps) == 0 {
		return
	}

	unitSection, _ := file.NewSection("Unit")
	unitShadows := make(map[string][]string)

	for depName := range deps {
		unitName := fmt.Sprintf("%s-%s.service", projectName, depName)
		unitShadows["Requires"] = append(unitShadows["Requires"], unitName)
		unitShadows["After"] = append(unitShadows["After"], unitName)
	}

	// Write shadow keys to [Unit] section
	for key, values := range unitShadows {
		if len(values) == 0 {
			continue
		}
		k, _ := unitSection.NewKey(key, values[0])
		for _, v := range values[1:] {
			if err := k.AddShadow(v); err != nil {
				continue
			}
		}
	}
}

// mapRestartPolicy converts Docker Compose restart policies to systemd equivalents.
func mapRestartPolicy(composeRestart string) string {
	switch composeRestart {
	case "no":
		return "no"
	case "always":
		return "always"
	case "on-failure":
		return "on-failure"
	case "unless-stopped":
		// systemd doesn't have "unless-stopped", use "always" as closest equivalent
		return "always"
	default:
		return composeRestart
	}
}

//nolint:gocyclo // High complexity is necessary due to mapping many container configuration options
func buildContainerSection(projectName, serviceName string, svc *types.ServiceConfig, section map[string]string, shadows map[string][]string, projectNetworks types.Networks, projectVolumes types.Volumes) { // nolint:whitespace
	// Image: required field
	if svc.Image != "" {
		section["Image"] = svc.Image
	}

	// ContainerName: use explicit name if set, otherwise default to <project>-<service>
	if svc.ContainerName != "" {
		section["ContainerName"] = svc.ContainerName
	} else {
		section["ContainerName"] = fmt.Sprintf("%s-%s", projectName, serviceName)
	}

	// Entrypoint: override image entrypoint
	if len(svc.Entrypoint) > 0 {
		// ShellCommand is a []string, join them for systemd
		section["Entrypoint"] = strings.Join(svc.Entrypoint, " ")
	}

	// Exec: override command for the container
	if len(svc.Command) > 0 {
		section["Exec"] = strings.Join(svc.Command, " ")
	}

	// WorkingDir: working directory inside the container
	if svc.WorkingDir != "" {
		section["WorkingDir"] = svc.WorkingDir
	}

	// User: set the user for the container
	if svc.User != "" {
		section["User"] = svc.User
	}

	// Group: set the group (additional group specified via group_add or derived from user)
	shadows["Group"] = append(shadows["Group"], svc.GroupAdd...)

	// Hostname: set container hostname
	if svc.Hostname != "" {
		section["HostName"] = svc.Hostname
	}

	// DomainName: set container domain name
	if svc.DomainName != "" {
		section["HostName"] = svc.DomainName // Podman uses HostName for domain as well
	}

	// Pull: image pull policy
	if svc.PullPolicy != "" {
		section["Pull"] = svc.PullPolicy
	}

	// Labels: map compose labels to systemd Label= directives with dot-notation
	for k, v := range svc.Labels {
		section[fmt.Sprintf("Label.%s", k)] = v
	}

	// Annotations: OCI annotations (different from labels)
	// Check extension for annotations since compose spec may not have direct support
	if annotations, ok := svc.Extensions["x-quad-ops-annotations"].(map[string]interface{}); ok {
		for k, v := range annotations {
			if vStr, ok := v.(string); ok {
				shadows["Annotation"] = append(shadows["Annotation"], fmt.Sprintf("%s=%s", k, vStr))
			}
		}
	}

	// Environment: environment variables (use shadows for repeated Environment= keys)
	for k, v := range svc.Environment {
		// v is a *string in MappingWithEquals
		if v != nil {
			shadows["Environment"] = append(shadows["Environment"], fmt.Sprintf("%s=%s", k, *v))
		}
	}

	// EnvironmentFile: support for env files
	for _, envFile := range svc.EnvFiles {
		if envFile.Path != "" {
			shadows["EnvironmentFile"] = append(shadows["EnvironmentFile"], envFile.Path)
		}
	}

	// Secret: map environment variables to Podman secrets (x-quad-ops-env-secrets extension)
	// Format: Secret=secretname,type=env,target=ENVVAR
	// Requires Podman 4.5+
	if envSecrets, ok := svc.Extensions["x-quad-ops-env-secrets"].(map[string]string); ok {
		for secretName, envVar := range envSecrets {
			shadows["Secret"] = append(shadows["Secret"], fmt.Sprintf("%s,type=env,target=%s", secretName, envVar))
		}
	}

	// DNS: DNS servers
	shadows["DNS"] = append(shadows["DNS"], svc.DNS...)

	// DNSSearch: DNS search domains
	shadows["DNSSearch"] = append(shadows["DNSSearch"], svc.DNSSearch...)

	// DNSOption: DNS options
	shadows["DNSOption"] = append(shadows["DNSOption"], svc.DNSOpts...)

	// ExtraHosts: host entries to add (HostsList is map[string][]string)
	if len(svc.ExtraHosts) > 0 {
		// Convert HostsList to host:ip format for Podman
		for host, ips := range svc.ExtraHosts {
			for _, ip := range ips {
				shadows["AddHost"] = append(shadows["AddHost"], fmt.Sprintf("%s:%s", host, ip))
			}
		}
	}

	// Expose: ports to expose (without publishing)
	shadows["ExposeHostPort"] = append(shadows["ExposeHostPort"], svc.Expose...)

	// Ports: published ports
	for _, portCfg := range svc.Ports {
		// Format: HostIP:HostPort:ContainerPort/Protocol
		portStr := formatPort(portCfg)
		if portStr != "" {
			shadows["PublishPort"] = append(shadows["PublishPort"], portStr)
		}
	}

	// Volumes: mount volumes
	// Named volumes are mapped to Quadlet .volume unit references so that
	// systemd creates proper Requires/After dependencies. External volumes
	// are referenced by their Podman volume name directly.
	for _, vol := range svc.Volumes {
		if vol.Type == types.VolumeTypeVolume && vol.Source != "" {
			if projVol, ok := projectVolumes[vol.Source]; ok && bool(projVol.External) {
				if projVol.Name != "" {
					vol.Source = projVol.Name
				}
			} else {
				vol.Source = fmt.Sprintf("%s-%s.volume", projectName, vol.Source)
			}
		}
		shadows["Volume"] = append(shadows["Volume"], vol.String())
	}

	// Tmpfs: tmpfs mounts
	shadows["Tmpfs"] = append(shadows["Tmpfs"], svc.Tmpfs...)

	// Mounts: advanced mount options via extension
	// Note: Docker Compose uses Volumes with type field. For advanced Mount= options,
	// support via x-quad-ops-mounts extension
	if mounts, ok := svc.Extensions["x-quad-ops-mounts"].([]interface{}); ok {
		for _, mount := range mounts {
			if mountStr, ok := mount.(string); ok {
				shadows["Mount"] = append(shadows["Mount"], mountStr)
			}
		}
	}

	// Devices: map host devices into the container
	for _, device := range svc.Devices {
		deviceStr := formatDevice(device)
		if deviceStr != "" {
			shadows["AddDevice"] = append(shadows["AddDevice"], deviceStr)
		}
	}

	// Capabilities: Linux capabilities
	shadows["AddCapability"] = append(shadows["AddCapability"], svc.CapAdd...)
	shadows["DropCapability"] = append(shadows["DropCapability"], svc.CapDrop...)

	// SecurityOpt: security options mapped to Quadlet keys
	for _, opt := range svc.SecurityOpt {
		switch {
		case opt == "label=disable" || opt == "label:disable":
			section["SecurityLabelDisable"] = "true"
		case opt == "label=nested" || opt == "label:nested":
			section["SecurityLabelNested"] = "true"
		case strings.HasPrefix(opt, "label=type:") || strings.HasPrefix(opt, "label:type:"):
			section["SecurityLabelType"] = strings.TrimPrefix(strings.TrimPrefix(opt, "label=type:"), "label:type:")
		case strings.HasPrefix(opt, "label=level:") || strings.HasPrefix(opt, "label:level:"):
			section["SecurityLabelLevel"] = strings.TrimPrefix(strings.TrimPrefix(opt, "label=level:"), "label:level:")
		case strings.HasPrefix(opt, "label=filetype:") || strings.HasPrefix(opt, "label:filetype:"):
			section["SecurityLabelFileType"] = strings.TrimPrefix(strings.TrimPrefix(opt, "label=filetype:"), "label:filetype:")
		case opt == "no-new-privileges" || opt == "no-new-privileges:true" || opt == "no-new-privileges=true":
			section["NoNewPrivileges"] = "true"
		case strings.HasPrefix(opt, "apparmor=") || strings.HasPrefix(opt, "apparmor:"):
			section["AppArmor"] = strings.TrimPrefix(strings.TrimPrefix(opt, "apparmor="), "apparmor:")
		case strings.HasPrefix(opt, "seccomp=") || strings.HasPrefix(opt, "seccomp:"):
			section["SeccompProfile"] = strings.TrimPrefix(strings.TrimPrefix(opt, "seccomp="), "seccomp:")
		case strings.HasPrefix(opt, "mask=") || strings.HasPrefix(opt, "mask:"):
			shadows["Mask"] = append(shadows["Mask"], strings.TrimPrefix(strings.TrimPrefix(opt, "mask="), "mask:"))
		case strings.HasPrefix(opt, "unmask=") || strings.HasPrefix(opt, "unmask:"):
			shadows["Unmask"] = append(shadows["Unmask"], strings.TrimPrefix(strings.TrimPrefix(opt, "unmask="), "unmask:"))
		}
	}

	// Privileged: privileged mode (Quadlet has no native Privileged key, use PodmanArgs)
	if svc.Privileged {
		shadows["PodmanArgs"] = append(shadows["PodmanArgs"], "--privileged")
	}

	// Ipc: IPC mode
	if svc.Ipc != "" {
		section["Ipc"] = svc.Ipc
	}

	// Pid: PID mode
	if svc.Pid != "" {
		section["Pid"] = svc.Pid
	}

	// Networks: map service networks to Quadlet .network unit references,
	// except external networks which are referenced by their Podman network name.
	rawNetworks := svc.NetworksByPriority()
	networks := make([]string, 0, len(rawNetworks))
	for _, n := range rawNetworks {
		if net, ok := projectNetworks[n]; ok && bool(net.External) {
			// External networks already exist in Podman; use the network name directly.
			if net.Name != "" {
				networks = append(networks, net.Name)
			} else {
				networks = append(networks, n)
			}
		} else {
			networks = append(networks, fmt.Sprintf("%s-%s.network", projectName, n))
		}
	}

	// Network mode: set explicit mode if specified.
	// When a network mode is set, skip adding individual networks—Podman
	// rejects multiple Network= directives with non-bridge modes.
	if svc.NetworkMode != "" {
		section["Network"] = svc.NetworkMode
	} else if svc.Net != "" {
		section["Network"] = svc.Net
	} else {
		// Add networks as shadow keys
		shadows["Network"] = append(shadows["Network"], networks...)
	}

	// ReadOnly: read-only filesystem
	if svc.ReadOnly {
		section["ReadOnly"] = "true"
	}

	// ShmSize: shared memory size
	if svc.ShmSize > 0 {
		section["ShmSize"] = fmt.Sprintf("%d", svc.ShmSize)
	}

	// Sysctls: sysctl settings
	for k, v := range svc.Sysctls {
		section[fmt.Sprintf("Sysctl.%s", k)] = v
	}

	// Ulimits: resource limits
	for name, limit := range svc.Ulimits {
		if limit != nil {
			ulimitStr := formatUlimit(limit)
			if ulimitStr != "" {
				section[fmt.Sprintf("Ulimit.%s", name)] = ulimitStr
			}
		}
	}

	// Memory: memory limit
	if svc.MemLimit > 0 {
		section["Memory"] = fmt.Sprintf("%d", svc.MemLimit)
	}

	// MemSwapLimit: swap memory limit
	if svc.MemSwapLimit > 0 {
		section["MemorySwap"] = fmt.Sprintf("%d", svc.MemSwapLimit)
	}

	// MemReservation: memory reservation
	if svc.MemReservation > 0 {
		section["MemoryReservation"] = fmt.Sprintf("%d", svc.MemReservation)
	}

	// CPUs: CPU limit
	if svc.CPUS > 0 {
		section["Cpus"] = fmt.Sprintf("%g", svc.CPUS)
	}

	// CPUShares: CPU shares
	if svc.CPUShares > 0 {
		section["CpuWeight"] = fmt.Sprintf("%d", svc.CPUShares)
	}

	// CPUSet: CPU affinity
	if svc.CPUSet != "" {
		section["CpuSet"] = svc.CPUSet
	}

	// OomKillDisable: disable OOM killer
	if svc.OomKillDisable {
		section["OomScoreAdj"] = "-999" // Disable OOM killer via high negative score
	}

	// OomScoreAdj: OOM score adjustment
	if svc.OomScoreAdj != 0 {
		section["OomScoreAdj"] = fmt.Sprintf("%d", svc.OomScoreAdj)
	}

	// PidsLimit: PID limit
	if svc.PidsLimit > 0 {
		section["PidsLimit"] = fmt.Sprintf("%d", svc.PidsLimit)
	}

	// StopSignal: signal to stop the container
	if svc.StopSignal != "" {
		section["StopSignal"] = svc.StopSignal
	}

	// StopGracePeriod: grace period before killing
	if svc.StopGracePeriod != nil && time.Duration(*svc.StopGracePeriod) > 0 {
		section["StopTimeout"] = fmt.Sprintf("%.0f", time.Duration(*svc.StopGracePeriod).Seconds())
	}

	// Tty: allocate pseudo-terminal
	if svc.Tty {
		section["Tty"] = "true"
	}

	// StdinOpen: keep stdin open even if not attached
	if svc.StdinOpen {
		section["Interactive"] = "true"
	}

	// HealthCheck: health check configuration
	if svc.HealthCheck != nil && len(svc.HealthCheck.Test) > 0 {
		mapHealthCheck(svc.HealthCheck, section)
	}

	// Note: Restart policy is handled in BuildContainer as it belongs in [Service] section

	// Init: use init process
	if svc.Init != nil && *svc.Init {
		section["RunInit"] = "true"
	}

	// LogDriver: logging driver
	if svc.LogDriver != "" {
		section["LogDriver"] = svc.LogDriver
	}

	// LogOpt: logging options
	for k, v := range svc.LogOpt {
		section[fmt.Sprintf("LogOpt.%s", k)] = v
	}

	// x-quad-ops-podman-args: list of global podman arguments
	if globalArgs, ok := svc.Extensions["x-quad-ops-podman-args"].([]interface{}); ok {
		for _, arg := range globalArgs {
			if argStr, ok := arg.(string); ok {
				shadows["PodmanArgs"] = append(shadows["PodmanArgs"], argStr)
			}
		}
	}

	// x-quad-ops-container-args: list of container-specific podman arguments
	if containerArgs, ok := svc.Extensions["x-quad-ops-container-args"].([]interface{}); ok {
		for _, arg := range containerArgs {
			if argStr, ok := arg.(string); ok {
				shadows["PodmanArgs"] = append(shadows["PodmanArgs"], argStr)
			}
		}
	}
}

// formatPort converts a ServicePortConfig to systemd PublishPort format.
func formatPort(cfg types.ServicePortConfig) string {
	// Format: HostIP:HostPort:ContainerPort/Protocol
	// Example: 8080:80 or 192.168.1.1:8080:80/tcp
	port := cfg.Published
	if port == "" {
		port = fmt.Sprintf("%d", cfg.Target)
	}

	if cfg.HostIP != "" {
		return fmt.Sprintf("%s:%s:%d/%s", cfg.HostIP, port, cfg.Target, cfg.Protocol)
	}
	return fmt.Sprintf("%s:%d/%s", port, cfg.Target, cfg.Protocol)
}

// formatDevice converts a DeviceMapping to systemd AddDevice format.
func formatDevice(device types.DeviceMapping) string {
	// Format: PathOnHost:PathInContainer:CgroupPermissions
	// DeviceMapping: Source, Target, Permissions
	if device.Source == "" {
		return ""
	}
	if device.Target == "" {
		return device.Source
	}
	if device.Permissions == "" {
		return fmt.Sprintf("%s:%s", device.Source, device.Target)
	}
	return fmt.Sprintf("%s:%s:%s", device.Source, device.Target, device.Permissions)
}

// formatUlimit converts an UlimitsConfig to systemd Ulimit format.
func formatUlimit(limit *types.UlimitsConfig) string {
	// Format: soft:hard or single
	if limit.Single != 0 {
		return fmt.Sprintf("%d", limit.Single)
	}
	if limit.Soft != 0 || limit.Hard != 0 {
		return fmt.Sprintf("%d:%d", limit.Soft, limit.Hard)
	}
	return ""
}

// mapHealthCheck converts compose HealthCheckConfig to systemd health directives.
func mapHealthCheck(hc *types.HealthCheckConfig, section map[string]string) {
	// HealthCmd: the command to run
	if len(hc.Test) > 0 {
		// HealthCheckTest is []string, format as space-separated command
		cmdStr := strings.Join(hc.Test, " ")
		section["HealthCmd"] = cmdStr
	}

	// HealthInterval: check interval
	if hc.Interval != nil && time.Duration(*hc.Interval) > 0 {
		section["HealthInterval"] = time.Duration(*hc.Interval).String()
	}

	// HealthTimeout: check timeout
	if hc.Timeout != nil && time.Duration(*hc.Timeout) > 0 {
		section["HealthTimeout"] = time.Duration(*hc.Timeout).String()
	}

	// HealthRetries: number of retries
	if hc.Retries != nil && *hc.Retries > 0 {
		section["HealthRetries"] = strconv.FormatUint(*hc.Retries, 10)
	}

	// HealthStartPeriod: grace period before starting health checks
	if hc.StartPeriod != nil && time.Duration(*hc.StartPeriod) > 0 {
		section["HealthStartPeriod"] = time.Duration(*hc.StartPeriod).String()
	}

	// HealthStartupInterval: startup check interval
	if hc.StartInterval != nil && time.Duration(*hc.StartInterval) > 0 {
		section["HealthStartupInterval"] = time.Duration(*hc.StartInterval).String()
	}
}
