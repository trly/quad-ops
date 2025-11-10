// Package compose provides Docker Compose project processing functionality
package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/sorting"
)

// SpecConverter converts Docker Compose projects to service.Spec models.
type SpecConverter struct {
	workingDir string
}

// NewSpecConverter creates a new SpecConverter.
func NewSpecConverter(workingDir string) *SpecConverter {
	return &SpecConverter{
		workingDir: workingDir,
	}
}

// ConvertProject converts a Docker Compose project to a list of service specs.
// It normalizes multi-container setups into multiple Spec instances, handling
// services, volumes, networks, and build configurations.
func (sc *SpecConverter) ConvertProject(project *types.Project) ([]service.Spec, error) {
	if err := sc.validateProject(project); err != nil {
		return nil, err
	}

	specs := make([]service.Spec, 0, len(project.Services))

	// Convert each service to one or more Specs
	for serviceName, composeService := range project.Services {
		serviceSpecs, err := sc.convertService(serviceName, composeService, project)
		if err != nil {
			return nil, fmt.Errorf("failed to convert service %s: %w", serviceName, err)
		}
		specs = append(specs, serviceSpecs...)
	}

	return specs, nil
}

// convertService converts a single Docker Compose service to one or more service.Spec instances.
// Init containers are converted to separate specs with dependencies on the main service.
func (sc *SpecConverter) convertService(serviceName string, composeService types.ServiceConfig, project *types.Project) ([]service.Spec, error) {
	// Create service name
	sanitizedName := Prefix(project.Name, serviceName)

	// Convert extensions
	initContainers := sc.convertInitContainers(serviceName, composeService, project)
	envSecrets := sc.convertEnvSecrets(composeService)

	container := sc.convertContainer(composeService, serviceName, project)
	container.EnvSecrets = envSecrets

	// Create main service spec
	spec := service.Spec{
		Name:        sanitizedName,
		Description: fmt.Sprintf("Service %s from project %s", serviceName, project.Name),
		Container:   container,
		Volumes:     sc.convertServiceVolumes(composeService, project),
		Networks:    sc.convertServiceNetworks(composeService, project),
		DependsOn:   sc.convertDependencies(composeService.DependsOn, project.Name),
		Annotations: sc.convertLabels(composeService.Labels),
	}

	// Add dependencies on init containers
	if len(initContainers) > 0 {
		initDeps := make([]string, len(initContainers))
		for i, initSpec := range initContainers {
			initDeps[i] = initSpec.Name
		}
		spec.DependsOn = append(spec.DependsOn, initDeps...)
	}

	// Validate the main spec
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed for service %s: %w", serviceName, err)
	}

	// Combine specs: init containers first, then main service
	specs := append(initContainers, spec)

	return specs, nil
}

// convertContainer converts Docker Compose service config to service.Container.
func (sc *SpecConverter) convertContainer(composeService types.ServiceConfig, serviceName string, project *types.Project) service.Container {
	mounts := sc.convertVolumeMounts(composeService.Volumes, project)

	if configMounts, err := sc.convertConfigMounts(composeService.Configs, project, serviceName); err == nil {
		mounts = append(mounts, configMounts...)
	}

	if secretMounts, err := sc.convertSecretMounts(composeService.Secrets, project, serviceName); err == nil {
		mounts = append(mounts, secretMounts...)
	}

	container := service.Container{
		Image:             composeService.Image,
		Command:           composeService.Command,
		Env:               sc.convertEnvironment(composeService.Environment),
		EnvFiles:          sc.convertEnvFiles(composeService.EnvFiles, serviceName),
		WorkingDir:        composeService.WorkingDir,
		User:              composeService.User,
		Ports:             sc.convertPorts(composeService.Ports),
		Mounts:            mounts,
		Resources:         sc.convertResources(composeService.Deploy, composeService),
		RestartPolicy:     sc.convertRestartPolicy(composeService.Restart),
		Healthcheck:       sc.convertHealthcheck(composeService.HealthCheck),
		Security:          sc.convertSecurity(composeService),
		Build:             sc.convertBuild(composeService.Build, composeService, project),
		Labels:            sc.convertLabels(composeService.Labels),
		Hostname:          composeService.Hostname,
		ContainerName:     Prefix(project.Name, serviceName),
		Entrypoint:        composeService.Entrypoint,
		Init:              composeService.Init != nil && *composeService.Init,
		ReadOnly:          composeService.ReadOnly,
		Logging:           sc.convertLogging(composeService.Logging),
		Secrets:           sc.convertExternalSecrets(composeService.Secrets, project),
		Network:           sc.convertNetworkMode(composeService.NetworkMode, composeService.Networks, project),
		Tmpfs:             sc.convertTmpfs(composeService.Tmpfs),
		Ulimits:           sc.convertUlimits(composeService.Ulimits),
		Sysctls:           composeService.Sysctls,
		UserNS:            composeService.UserNSMode,
		PidMode:           composeService.Pid,
		IpcMode:           composeService.Ipc,
		CgroupMode:        composeService.Cgroup,
		ExtraHosts:        sc.convertExtraHosts(composeService.ExtraHosts),
		DNS:               sc.convertDNS(composeService.DNS),
		DNSSearch:         sc.convertDNSSearch(composeService.DNSSearch),
		DNSOptions:        sc.convertDNSOpts(composeService.DNSOpts),
		Devices:           sc.convertDevices(composeService.Devices),
		DeviceCgroupRules: sc.convertDeviceCgroupRules(composeService.DeviceCgroupRules),
		StopSignal:        composeService.StopSignal,
		StopGracePeriod:   sc.convertDuration(composeService.StopGracePeriod),
	}

	// Handle user/group parsing
	if container.User != "" {
		parts := strings.SplitN(container.User, ":", 2)
		if len(parts) == 2 {
			container.User = parts[0]
			container.Group = parts[1]
		}
	}

	return container
}

// convertEnvironment converts compose environment to map.
func (sc *SpecConverter) convertEnvironment(env types.MappingWithEquals) map[string]string {
	if env == nil {
		return nil
	}
	result := make(map[string]string, len(env))
	for k, v := range env {
		if v != nil {
			result[k] = *v
		} else {
			result[k] = ""
		}
	}
	return result
}

// convertEnvFiles converts compose env files to list of paths.
func (sc *SpecConverter) convertEnvFiles(envFiles []types.EnvFile, serviceName string) []string {
	var result []string

	// Add compose-defined env files
	for _, ef := range envFiles {
		if ef.Path != "" {
			result = append(result, ef.Path)
		}
	}

	// Add discovered env files
	discovered := FindEnvFiles(serviceName, sc.workingDir)
	result = append(result, discovered...)

	// Sort for determinism
	sorting.SortStringSlice(result)
	return result
}

// convertPorts converts compose ports to service.Port.
func (sc *SpecConverter) convertPorts(ports []types.ServicePortConfig) []service.Port {
	if len(ports) == 0 {
		return nil
	}

	result := make([]service.Port, 0, len(ports))
	for _, p := range ports {
		// Parse Published port string to uint16
		var hostPort uint16
		if p.Published != "" {
			parsed, err := strconv.ParseUint(p.Published, 10, 16)
			if err == nil {
				hostPort = uint16(parsed)
			}
		}

		// Clamp container port to uint16 max to prevent overflow
		containerPort := p.Target
		if containerPort > 65535 {
			containerPort = 65535
		}

		port := service.Port{
			Host:      p.HostIP,
			HostPort:  hostPort,
			Container: uint16(containerPort), // #nosec G115 - clamped to uint16 max above
			Protocol:  p.Protocol,
		}
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}
		result = append(result, port)
	}
	return result
}

// convertVolumeMounts converts compose volume configs to service.Mount.
func (sc *SpecConverter) convertVolumeMounts(volumes []types.ServiceVolumeConfig, project *types.Project) []service.Mount {
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
			if v.Bind != nil {
				bindOpts := &service.BindOptions{}
				if v.Bind.Propagation != "" {
					bindOpts.Propagation = v.Bind.Propagation
				}
				if v.Bind.SELinux != "" {
					bindOpts.SELinux = v.Bind.SELinux
				}
				if bindOpts.Propagation != "" || bindOpts.SELinux != "" {
					mount.BindOptions = bindOpts
				}
			}
		case "volume":
			mount.Type = service.MountTypeVolume
			// Prefix named volume sources to match volume names
			if v.Source != "" {
				mount.Source = Prefix(project.Name, v.Source)
			}
		case "tmpfs":
			mount.Type = service.MountTypeTmpfs
			if v.Tmpfs != nil {
				tmpfsOpts := &service.TmpfsOptions{}
				if v.Tmpfs.Size > 0 {
					tmpfsOpts.Size = sc.formatBytes(v.Tmpfs.Size)
				}
				if v.Tmpfs.Mode != 0 {
					tmpfsOpts.Mode = v.Tmpfs.Mode
				}
				mount.TmpfsOptions = tmpfsOpts
			}
		default:
			// Auto-detect: if source is absolute path or starts with ./, it's a bind mount
			if v.Source != "" && (filepath.IsAbs(v.Source) || strings.HasPrefix(v.Source, "./") || strings.HasPrefix(v.Source, "../")) {
				mount.Type = service.MountTypeBind
			} else {
				mount.Type = service.MountTypeVolume
				// Prefix named volume sources to match volume names
				if v.Source != "" {
					mount.Source = Prefix(project.Name, v.Source)
				}
			}
		}

		result = append(result, mount)
	}
	return result
}

// convertResources converts compose deploy resources to service.Resources.
func (sc *SpecConverter) convertResources(deploy *types.DeployConfig, svc types.ServiceConfig) service.Resources {
	resources := service.Resources{}

	// Process deploy resources if present
	if deploy != nil && (deploy.Resources.Limits != nil || deploy.Resources.Reservations != nil) {
		// Limits
		if deploy.Resources.Limits != nil {
			if deploy.Resources.Limits.MemoryBytes > 0 {
				resources.Memory = sc.formatBytes(deploy.Resources.Limits.MemoryBytes)
			}
			if deploy.Resources.Limits.NanoCPUs > 0 {
				cpuQuota, cpuPeriod := sc.convertCPU(deploy.Resources.Limits.NanoCPUs)
				resources.CPUQuota = cpuQuota
				resources.CPUPeriod = cpuPeriod
			}
			if deploy.Resources.Limits.Pids > 0 {
				resources.PidsLimit = deploy.Resources.Limits.Pids
			}
		}

		// Reservations
		if deploy.Resources.Reservations != nil {
			if deploy.Resources.Reservations.MemoryBytes > 0 {
				resources.MemoryReservation = sc.formatBytes(deploy.Resources.Reservations.MemoryBytes)
			}
			// Derive CPUShares from reservations
			if deploy.Resources.Reservations.NanoCPUs > 0 {
				resources.CPUShares = sc.convertCPUShares(deploy.Resources.Reservations.NanoCPUs)
			}
		}
	}

	// MemSwapLimit from service-level field (not deploy.resources)
	if svc.MemSwapLimit > 0 {
		resources.MemorySwap = sc.formatBytes(svc.MemSwapLimit)
	}

	// ShmSize from service-level field
	if svc.ShmSize > 0 {
		resources.ShmSize = sc.formatBytes(svc.ShmSize)
	}

	return resources
}

// formatBytes converts bytes to human-readable format.
func (sc *SpecConverter) formatBytes(bytes types.UnitBytes) string {
	b := int64(bytes)
	if b < 1024 {
		return fmt.Sprintf("%d", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%dk", b/1024)
	}
	if b < 1024*1024*1024 {
		return fmt.Sprintf("%dm", b/(1024*1024))
	}
	return fmt.Sprintf("%dg", b/(1024*1024*1024))
}

// convertCPU converts nanoCPUs to quota and period.
func (sc *SpecConverter) convertCPU(nanoCPUs types.NanoCPUs) (quota int64, period int64) {
	// NanoCPUs is a float32 (e.g., 0.5 means 50% of one CPU)
	if nanoCPUs == 0 {
		return 0, 0
	}

	// Standard CPU period is 100000 microseconds (100ms)
	period = 100000
	quota = int64(float64(nanoCPUs) * float64(period))
	return quota, period
}

// convertCPUShares converts nanoCPUs to CPU shares.
func (sc *SpecConverter) convertCPUShares(nanoCPUs types.NanoCPUs) int64 {
	// CPU shares are relative weights (default 1024 = 1 CPU)
	// NanoCPUs is a float32 (e.g., 0.5 means 50% of one CPU)
	if nanoCPUs == 0 {
		return 0
	}
	return int64(float64(nanoCPUs) * 1024)
}

// convertRestartPolicy converts compose restart to service.RestartPolicy.
func (sc *SpecConverter) convertRestartPolicy(restart string) service.RestartPolicy {
	switch restart {
	case "no":
		return service.RestartPolicyNo
	case "always":
		return service.RestartPolicyAlways
	case "on-failure":
		return service.RestartPolicyOnFailure
	case "unless-stopped":
		return service.RestartPolicyUnlessStopped
	default:
		return service.RestartPolicyNo
	}
}

// convertHealthcheck converts compose healthcheck to service.Healthcheck.
func (sc *SpecConverter) convertHealthcheck(hc *types.HealthCheckConfig) *service.Healthcheck {
	if hc == nil || hc.Disable {
		return nil
	}

	healthcheck := &service.Healthcheck{
		Test: hc.Test,
	}

	// Convert retries (uint64 pointer to int)
	if hc.Retries != nil {
		// Clamp retries to int max to prevent overflow
		retries := *hc.Retries
		if retries > 2147483647 {
			retries = 2147483647
		}
		healthcheck.Retries = int(retries) // #nosec G115 - clamped to int max above
	}

	// Convert durations
	if hc.Interval != nil {
		healthcheck.Interval = time.Duration(*hc.Interval)
	}
	if hc.Timeout != nil {
		healthcheck.Timeout = time.Duration(*hc.Timeout)
	}
	if hc.StartPeriod != nil {
		healthcheck.StartPeriod = time.Duration(*hc.StartPeriod)
	}
	if hc.StartInterval != nil {
		healthcheck.StartInterval = time.Duration(*hc.StartInterval)
	}

	return healthcheck
}

// convertSecurity converts compose security settings to service.Security.
func (sc *SpecConverter) convertSecurity(composeService types.ServiceConfig) service.Security {
	security := service.Security{
		Privileged:     composeService.Privileged,
		CapAdd:         composeService.CapAdd,
		CapDrop:        composeService.CapDrop,
		SecurityOpt:    composeService.SecurityOpt,
		ReadonlyRootfs: composeService.ReadOnly,
		GroupAdd:       composeService.GroupAdd,
	}

	// Parse security_opt for specific fields
	for _, opt := range composeService.SecurityOpt {
		parts := strings.SplitN(opt, "=", 2)
		if len(parts) != 2 {
			continue
		}

		switch parts[0] {
		case "seccomp":
			security.SeccompProfile = parts[1]
		case "apparmor":
			security.AppArmorProfile = parts[1]
		case "label":
			if strings.HasPrefix(parts[1], "type:") {
				security.SELinuxType = strings.TrimPrefix(parts[1], "type:")
			}
		}
	}

	return security
}

// convertBuild converts compose build config to service.Build.
func (sc *SpecConverter) convertBuild(build *types.BuildConfig, _ types.ServiceConfig, project *types.Project) *service.Build {
	if build == nil {
		return nil
	}

	buildSpec := &service.Build{
		Context:    build.Context,
		Dockerfile: build.Dockerfile,
		Target:     build.Target,
		Args:       sc.convertBuildArgs(build.Args),
		Labels:     sc.convertLabels(build.Labels),
		Pull:       build.Pull,
		Tags:       build.Tags,
	}

	// Convert build context path
	if buildSpec.Context == "" {
		buildSpec.Context = "."
	}
	if !filepath.IsAbs(buildSpec.Context) {
		buildSpec.Context = filepath.Join(project.WorkingDir, buildSpec.Context)
	}

	// Set dockerfile path
	if buildSpec.Dockerfile == "" {
		buildSpec.Dockerfile = "Dockerfile"
	}

	// Set working directory for build
	buildSpec.SetWorkingDirectory = project.WorkingDir

	return buildSpec
}

// convertBuildArgs converts compose build args to map.
func (sc *SpecConverter) convertBuildArgs(args types.MappingWithEquals) map[string]string {
	if args == nil {
		return nil
	}
	result := make(map[string]string, len(args))
	for k, v := range args {
		if v != nil {
			result[k] = *v
		} else {
			result[k] = ""
		}
	}
	return result
}

// convertLogging converts compose logging to service.Logging.
func (sc *SpecConverter) convertLogging(logging *types.LoggingConfig) service.Logging {
	if logging == nil {
		return service.Logging{}
	}

	return service.Logging{
		Driver:  logging.Driver,
		Options: logging.Options,
	}
}

// convertNetworkMode converts compose network mode to service.NetworkMode.
func (sc *SpecConverter) convertNetworkMode(networkMode string, networks map[string]*types.ServiceNetworkConfig, project *types.Project) service.NetworkMode {
	mode := service.NetworkMode{
		Mode:            networkMode,
		ServiceNetworks: make([]string, 0, len(networks)),
	}

	// If the service doesn't explicitly declare networks, populate ServiceNetworks
	// with the project's default networks. This ensures containers depend on the
	// correct network units even when using implicit network assignment.
	if len(networks) == 0 {
		for networkName, projectNet := range project.Networks {
			// Skip external networks - they don't have .network units
			if IsExternal(projectNet.External) {
				continue
			}

			// Resolve and add network name
			resolvedName := NameResolver(projectNet.Name, networkName)
			sanitizedName := resolvedName
			if !strings.Contains(resolvedName, project.Name) {
				sanitizedName = Prefix(project.Name, resolvedName)
			}
			mode.ServiceNetworks = append(mode.ServiceNetworks, sanitizedName)
		}
	} else {
		// Collect aliases and resolve network names for explicitly declared networks
		for networkName, netConfig := range networks {
			if netConfig != nil && len(netConfig.Aliases) > 0 {
				mode.Aliases = append(mode.Aliases, netConfig.Aliases...)
			}

			// Resolve and add network name to ServiceNetworks
			projectNet, exists := project.Networks[networkName]
			if !exists {
				// External or undefined network - use as-is
				// Don't apply current project prefix to external networks
				mode.ServiceNetworks = append(mode.ServiceNetworks, networkName)
				continue
			}

			// Check if it's an external network
			if IsExternal(projectNet.External) {
				// External network from another project - use as-is
				mode.ServiceNetworks = append(mode.ServiceNetworks, networkName)
				continue
			}

			// Resolve network name from project definition
			resolvedName := NameResolver(projectNet.Name, networkName)
			sanitizedName := resolvedName
			if !strings.Contains(resolvedName, project.Name) {
				sanitizedName = Prefix(project.Name, resolvedName)
			}
			mode.ServiceNetworks = append(mode.ServiceNetworks, sanitizedName)
		}
	}

	// If no explicit mode, default to bridge
	if mode.Mode == "" {
		mode.Mode = "bridge"
	}

	// Sort for determinism
	sort.Strings(mode.ServiceNetworks)

	return mode
}

// convertTmpfs converts compose tmpfs to string slice.
func (sc *SpecConverter) convertTmpfs(tmpfs types.StringList) []string {
	if len(tmpfs) == 0 {
		return nil
	}
	return []string(tmpfs)
}

// convertExtraHosts converts compose extra_hosts to "hostname:ip" string slice.
// Docker Compose HostsList is a map[string][]string where key is hostname and value is list of IPs.
// We convert this to Quadlet's AddHost format which expects "hostname:ip" strings.
// If a hostname has multiple IPs, we generate one entry per IP.
func (sc *SpecConverter) convertExtraHosts(extraHosts types.HostsList) []string {
	if len(extraHosts) == 0 {
		return nil
	}

	// Use HostsList.AsList() to convert to "hostname:ip" format
	// The ":" separator matches Quadlet's AddHost directive format
	result := extraHosts.AsList(":")

	// Sort for determinism
	sorting.SortStringSlice(result)
	return result
}

// convertDNS converts compose dns to DNS servers slice.
func (sc *SpecConverter) convertDNS(dns []string) []string {
	if len(dns) == 0 {
		return nil
	}

	result := make([]string, len(dns))
	copy(result, dns)

	// Sort for determinism
	sorting.SortStringSlice(result)
	return result
}

// convertDNSSearch converts compose dns_search to DNS search domains slice.
func (sc *SpecConverter) convertDNSSearch(dnsSearch []string) []string {
	if len(dnsSearch) == 0 {
		return nil
	}

	result := make([]string, len(dnsSearch))
	copy(result, dnsSearch)

	// Sort for determinism
	sorting.SortStringSlice(result)
	return result
}

// convertDNSOpts converts compose dns_opt to DNS options slice.
func (sc *SpecConverter) convertDNSOpts(dnsOpts []string) []string {
	if len(dnsOpts) == 0 {
		return nil
	}

	result := make([]string, len(dnsOpts))
	copy(result, dnsOpts)

	// Sort for determinism
	sorting.SortStringSlice(result)
	return result
}

// convertDevices converts compose devices to device mappings slice.
func (sc *SpecConverter) convertDevices(devices []types.DeviceMapping) []string {
	if len(devices) == 0 {
		return nil
	}

	result := make([]string, 0, len(devices))
	for _, device := range devices {
		// Build device string: source:target or source:target:permissions
		deviceStr := device.Source
		if device.Target != "" {
			deviceStr = fmt.Sprintf("%s:%s", device.Source, device.Target)
		}
		if device.Permissions != "" {
			deviceStr = fmt.Sprintf("%s:%s", deviceStr, device.Permissions)
		}
		result = append(result, deviceStr)
	}

	// Sort for determinism
	sorting.SortStringSlice(result)
	return result
}

// convertDeviceCgroupRules converts compose device_cgroup_rules to slice.
// Rules are in format "type major:minor permissions" (e.g., "c 13:* rmw").
func (sc *SpecConverter) convertDeviceCgroupRules(rules []string) []string {
	if len(rules) == 0 {
		return nil
	}

	result := make([]string, len(rules))
	copy(result, rules)

	// Sort for determinism
	sorting.SortStringSlice(result)
	return result
}

// convertUlimits converts compose ulimits to service.Ulimit.
// Handles both short syntax (nproc: 512 -> Single field) and long syntax (soft/hard fields).
func (sc *SpecConverter) convertUlimits(ulimits map[string]*types.UlimitsConfig) []service.Ulimit {
	if len(ulimits) == 0 {
		return nil
	}

	result := make([]service.Ulimit, 0, len(ulimits))
	for name, limit := range ulimits {
		if limit != nil {
			soft, hard := int64(limit.Soft), int64(limit.Hard)
			// Handle short syntax: nproc: 512 sets both soft and hard to 512
			if limit.Single > 0 {
				soft = int64(limit.Single)
				hard = int64(limit.Single)
			}
			result = append(result, service.Ulimit{
				Name: name,
				Soft: soft,
				Hard: hard,
			})
		}
	}
	return result
}

// convertDependencies converts compose depends_on to service name list.
// Parses Compose v2 dependency conditions (service_started, service_healthy, service_completed_successfully).
// All conditions map to systemd After/Requires directives (health checking cannot be enforced in systemd dependencies).
// Missing or unknown conditions are treated as service_started.
func (sc *SpecConverter) convertDependencies(dependsOn types.DependsOnConfig, projectName string) []string {
	if len(dependsOn) == 0 {
		return nil
	}

	result := make([]string, 0, len(dependsOn))
	for serviceName, dep := range dependsOn {
		// Parse condition to validate it's a known type
		// Valid conditions: service_started, service_healthy, service_completed_successfully
		// All map to After/Requires in systemd (health enforcement requires external tooling)
		switch dep.Condition {
		case "service_started", "service_healthy", "service_completed_successfully", "":
			// All valid conditions and empty (default to service_started)
		default:
			// Unknown condition - treat as service_started but continue
			// In future, could log warning here
		}

		// Convert to service name
		result = append(result, Prefix(projectName, serviceName))
	}

	// Sort for determinism
	sorting.SortStringSlice(result)
	return result
}

// convertLabels converts compose labels to map.
func (sc *SpecConverter) convertLabels(labels types.Labels) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	result := make(map[string]string, len(labels))
	for k, v := range labels {
		result[k] = v
	}
	return result
}

// convertProjectVolumes converts project-level volumes to service.Volume.
func (sc *SpecConverter) convertProjectVolumes(project *types.Project) []service.Volume {
	if len(project.Volumes) == 0 {
		return nil
	}

	result := make([]service.Volume, 0, len(project.Volumes))
	for name, vol := range project.Volumes {
		// Resolve volume name
		volumeName := NameResolver(vol.Name, name)
		// Apply prefix only if not already prefixed
		sanitizedName := volumeName
		if !strings.Contains(volumeName, project.Name) {
			sanitizedName = Prefix(project.Name, volumeName)
		}

		// Skip external volumes
		if IsExternal(vol.External) {
			continue
		}

		volume := service.Volume{
			Name:     sanitizedName,
			Driver:   vol.Driver,
			Options:  vol.DriverOpts,
			Labels:   sc.convertLabels(vol.Labels),
			External: IsExternal(vol.External),
		}

		result = append(result, volume)
	}
	return result
}

// convertServiceVolumes converts service-level volume declarations to service.Volume.
// Only returns volumes that the service actually mounts, not all project volumes.
func (sc *SpecConverter) convertServiceVolumes(composeService types.ServiceConfig, project *types.Project) []service.Volume {
	if len(composeService.Volumes) == 0 {
		return nil
	}

	// Collect unique named volumes used by this service
	usedVolumes := make(map[string]bool)
	for _, mount := range composeService.Volumes {
		// Only track named volumes (not bind mounts or tmpfs)
		if mount.Type == "volume" || (mount.Type == "" && mount.Source != "" &&
			!filepath.IsAbs(mount.Source) &&
			!strings.HasPrefix(mount.Source, "./") &&
			!strings.HasPrefix(mount.Source, "../")) {
			if mount.Source != "" {
				usedVolumes[mount.Source] = true
			}
		}
	}

	if len(usedVolumes) == 0 {
		return nil
	}

	result := make([]service.Volume, 0, len(usedVolumes))

	// Convert each used volume
	for volumeName := range usedVolumes {
		projectVol, exists := project.Volumes[volumeName]
		if !exists {
			// Volume declared by service but not in project volumes
			// This can happen with external volumes from other projects
			volume := service.Volume{
				Name:   volumeName,
				Driver: "local",
			}
			result = append(result, volume)
			continue
		}

		// Resolve volume name from project definition
		resolvedName := NameResolver(projectVol.Name, volumeName)
		sanitizedName := resolvedName

		// Don't apply project prefix to external volumes
		if !IsExternal(projectVol.External) && !strings.Contains(resolvedName, project.Name) {
			sanitizedName = Prefix(project.Name, resolvedName)
		}

		volume := service.Volume{
			Name:     sanitizedName,
			Driver:   projectVol.Driver,
			Options:  projectVol.DriverOpts,
			Labels:   sc.convertLabels(projectVol.Labels),
			External: IsExternal(projectVol.External),
		}

		result = append(result, volume)
	}

	// Sort for determinism
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// convertServiceNetworks converts service-level network declarations to service.Network.
// If a service declares specific networks, it should be connected to those networks.
// If no networks are declared, falls back to project-level networks.
func (sc *SpecConverter) convertServiceNetworks(composeService types.ServiceConfig, project *types.Project) []service.Network {
	// If service declares specific networks, use those
	if len(composeService.Networks) > 0 {
		return sc.convertServiceNetworksList(composeService.Networks, project)
	}

	// Fall back to project-level networks
	return sc.convertProjectNetworks(project)
}

// convertServiceNetworksList converts a service's network declarations to service.Network.
func (sc *SpecConverter) convertServiceNetworksList(networks map[string]*types.ServiceNetworkConfig, project *types.Project) []service.Network {
	if len(networks) == 0 {
		return nil
	}

	result := make([]service.Network, 0, len(networks))

	for networkName := range networks {
		projectNet, exists := project.Networks[networkName]
		if !exists {
			// Network declared by service but not in project networks
			// This can happen with external networks (from other projects)
			// Use network name as-is without applying current project prefix
			network := service.Network{
				Name:     networkName,
				Driver:   "bridge",
				External: true, // Mark as external since it's not defined in this project
			}
			result = append(result, network)
			continue
		}

		// Resolve network name from project definition
		resolvedName := NameResolver(projectNet.Name, networkName)
		sanitizedName := resolvedName

		// Don't apply project prefix to external networks
		if !IsExternal(projectNet.External) && !strings.Contains(resolvedName, project.Name) {
			sanitizedName = Prefix(project.Name, resolvedName)
		}

		network := service.Network{
			Name:     sanitizedName,
			Driver:   projectNet.Driver,
			Options:  projectNet.DriverOpts,
			Labels:   sc.convertLabels(projectNet.Labels),
			Internal: projectNet.Internal,
			IPv6:     projectNet.EnableIPv6 != nil && *projectNet.EnableIPv6,
			External: IsExternal(projectNet.External),
		}

		// Convert IPAM if present
		if projectNet.Ipam.Driver != "" || len(projectNet.Ipam.Config) > 0 {
			network.IPAM = sc.convertIPAM(&projectNet.Ipam)
		}

		result = append(result, network)
	}

	// Sort for determinism
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// convertProjectNetworks converts project-level networks to service.Network.
func (sc *SpecConverter) convertProjectNetworks(project *types.Project) []service.Network {
	if len(project.Networks) == 0 {
		return nil
	}

	result := make([]service.Network, 0, len(project.Networks))
	for name, net := range project.Networks {
		// Resolve network name
		networkName := NameResolver(net.Name, name)
		// Apply prefix only if not already prefixed
		sanitizedName := networkName
		if !strings.Contains(networkName, project.Name) {
			sanitizedName = Prefix(project.Name, networkName)
		}

		// Skip external networks
		if IsExternal(net.External) {
			continue
		}

		network := service.Network{
			Name:     sanitizedName,
			Driver:   net.Driver,
			Options:  net.DriverOpts,
			Labels:   sc.convertLabels(net.Labels),
			Internal: net.Internal,
			IPv6:     net.EnableIPv6 != nil && *net.EnableIPv6,
			External: IsExternal(net.External),
		}

		// Convert IPAM if present
		if net.Ipam.Driver != "" || len(net.Ipam.Config) > 0 {
			network.IPAM = sc.convertIPAM(&net.Ipam)
		}

		result = append(result, network)
	}
	return result
}

// convertIPAM converts compose IPAM to service.IPAM.
func (sc *SpecConverter) convertIPAM(ipam *types.IPAMConfig) *service.IPAM {
	if ipam == nil {
		return nil
	}

	result := &service.IPAM{
		Driver: ipam.Driver,
	}

	// Convert IPAM configs
	if len(ipam.Config) > 0 {
		result.Config = make([]service.IPAMConfig, 0, len(ipam.Config))
		for _, cfg := range ipam.Config {
			result.Config = append(result.Config, service.IPAMConfig{
				Subnet:  cfg.Subnet,
				Gateway: cfg.Gateway,
				IPRange: cfg.IPRange,
			})
		}
	}

	return result
}

// convertInitContainers converts x-quad-ops-init extension to init container specs.
// Init containers are only supported on Linux due to systemd dependency requirements.
func (sc *SpecConverter) convertInitContainers(serviceName string, composeService types.ServiceConfig, project *types.Project) []service.Spec {
	// Init containers require systemd dependencies, so only implement on Linux
	if runtime.GOOS != "linux" {
		return nil
	}

	extension, exists := composeService.Extensions["x-quad-ops-init"]
	if !exists {
		return nil
	}

	initList, ok := extension.([]interface{})
	if !ok {
		return nil
	}

	specs := make([]service.Spec, 0, len(initList))
	baseName := Prefix(project.Name, serviceName)

	// Pre-convert main service resources that init containers can inherit
	mainEnv := sc.convertEnvironment(composeService.Environment)
	mainMounts := sc.convertVolumeMounts(composeService.Volumes, project)
	mainNetwork := sc.convertNetworkMode(composeService.NetworkMode, composeService.Networks, project)
	mainVolumes := sc.convertServiceVolumes(composeService, project)
	mainNetworks := sc.convertServiceNetworks(composeService, project)

	for i, item := range initList {
		initMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Parse init container specific config (optional overrides)
		initEnv := sc.getMapFromMap(initMap, "environment")
		initVolumes := sc.getStringSliceFromMap(initMap, "volumes")

		// Build init container config, inheriting from main service
		container := service.Container{
			Image:   sc.getStringFromMap(initMap, "image"),
			Command: sc.getStringSliceFromMap(initMap, "command"),
			Env:     mainEnv,     // Inherit main service env by default
			Mounts:  mainMounts,  // Inherit main service mounts by default
			Network: mainNetwork, // Inherit main service network mode
		}

		// Override env if init container specifies custom environment
		if len(initEnv) > 0 {
			container.Env = initEnv
		}

		// Override mounts if init container specifies custom volumes
		if len(initVolumes) > 0 {
			// Convert init-specific volumes
			container.Mounts = sc.convertInitVolumeMounts(initVolumes, project)
		}

		initSpec := service.Spec{
			Name:        fmt.Sprintf("%s-init-%d", baseName, i),
			Description: fmt.Sprintf("Init container %d for service %s", i, serviceName),
			Container:   container,
			// Share main service's volumes and networks so init can access them
			Volumes:   mainVolumes,
			Networks:  mainNetworks,
			DependsOn: []string{}, // No dependencies for init containers themselves
		}

		// Validate init container spec
		if err := initSpec.Validate(); err != nil {
			// Skip invalid init containers but don't fail the whole conversion
			continue
		}

		specs = append(specs, initSpec)
	}

	return specs
}

// convertInitVolumeMounts converts init container volume strings to service.Mount.
// Format: "source:target" or "source:target:ro".
func (sc *SpecConverter) convertInitVolumeMounts(volumes []string, project *types.Project) []service.Mount {
	if len(volumes) == 0 {
		return nil
	}

	result := make([]service.Mount, 0, len(volumes))
	for _, v := range volumes {
		parts := strings.Split(v, ":")
		if len(parts) < 2 {
			continue
		}

		mount := service.Mount{
			Source:   parts[0],
			Target:   parts[1],
			ReadOnly: len(parts) > 2 && parts[2] == "ro",
			Options:  make(map[string]string),
		}

		// Determine mount type
		if filepath.IsAbs(parts[0]) || strings.HasPrefix(parts[0], "./") || strings.HasPrefix(parts[0], "../") {
			mount.Type = service.MountTypeBind
		} else {
			mount.Type = service.MountTypeVolume
			// Prefix named volume sources to match volume names
			mount.Source = Prefix(project.Name, parts[0])
		}

		result = append(result, mount)
	}
	return result
}

// convertEnvSecrets converts x-podman-env-secrets extension to env secrets map.
func (sc *SpecConverter) convertEnvSecrets(composeService types.ServiceConfig) map[string]string {
	extension, exists := composeService.Extensions["x-podman-env-secrets"]
	if !exists {
		return nil
	}

	envSecrets, ok := extension.(map[string]interface{})
	if !ok {
		return nil
	}

	result := make(map[string]string)
	for secretName, envVar := range envSecrets {
		if envVarStr, ok := envVar.(string); ok {
			result[secretName] = envVarStr
		}
	}

	return result
}

// getStringFromMap extracts a string value from a map.
func (sc *SpecConverter) getStringFromMap(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getMapFromMap extracts a map[string]string from a map.
func (sc *SpecConverter) getMapFromMap(m map[string]interface{}, key string) map[string]string {
	if val, exists := m[key]; exists {
		if mapVal, ok := val.(map[string]interface{}); ok {
			result := make(map[string]string)
			for k, v := range mapVal {
				if strVal, ok := v.(string); ok {
					result[k] = strVal
				}
			}
			return result
		}
	}
	return nil
}

// getStringSliceFromMap extracts a string slice from a map.
func (sc *SpecConverter) getStringSliceFromMap(m map[string]interface{}, key string) []string {
	if val, exists := m[key]; exists {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, item := range slice {
				if str, ok := item.(string); ok {
					result[i] = str
				}
			}
			return result
		}
		if str, ok := val.(string); ok {
			// Single string, convert to slice
			return []string{str}
		}
	}
	return nil
}

// validateProject validates project-level configs, secrets, and project name.
func (sc *SpecConverter) validateProject(project *types.Project) error {
	// Validate project name matches service name regex
	serviceNameRegex := "^[a-zA-Z0-9][a-zA-Z0-9_.-]*$"
	matched, err := regexp.MatchString(serviceNameRegex, project.Name)
	if err != nil {
		return fmt.Errorf("failed to validate project name regex: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid project name %q: must start with alphanumeric and contain only alphanumeric, hyphen, underscore, or dot", project.Name)
	}

	for name, cfg := range project.Configs {
		if cfg.Driver != "" {
			return fmt.Errorf("config %q uses 'driver' which is Swarm-specific and not supported. Use file/content/environment sources instead", name)
		}
	}

	for name, secret := range project.Secrets {
		if secret.Driver != "" {
			return fmt.Errorf("secret %q uses 'driver' which is Swarm-specific and not supported. Use file/content/environment sources instead", name)
		}
	}

	return nil
}

// ensureProjectTempDir creates and returns a temporary directory for the project.
func (sc *SpecConverter) ensureProjectTempDir(projectName, kind string) (string, error) {
	tempBase := filepath.Join(os.TempDir(), "quad-ops", projectName, kind)
	if err := os.MkdirAll(tempBase, 0700); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	return tempBase, nil
}

// writeTempFile writes content to a file in the given directory with the specified mode.
func (sc *SpecConverter) writeTempFile(dir, name string, data []byte, mode os.FileMode) (string, error) {
	filePath := filepath.Join(dir, name)

	if _, err := os.Stat(filePath); err == nil {
		if err := os.Chmod(filePath, 0600); err != nil {
			return "", fmt.Errorf("failed to make file writable: %w", err)
		}
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Chmod(filePath, mode); err != nil {
		return "", fmt.Errorf("failed to set file mode: %w", err)
	}
	return filePath, nil
}

// convertConfigMounts converts compose configs with local sources to bind mounts.
func (sc *SpecConverter) convertConfigMounts(configs []types.ServiceConfigObjConfig, project *types.Project, serviceName string) ([]service.Mount, error) {
	if len(configs) == 0 {
		return nil, nil
	}

	result := make([]service.Mount, 0, len(configs))

	for _, cfg := range configs {
		projectCfg, exists := project.Configs[cfg.Source]
		if !exists {
			return nil, fmt.Errorf("config %q referenced by service %q not found in project configs", cfg.Source, serviceName)
		}

		if IsExternal(projectCfg.External) {
			continue
		}

		var mode *uint32
		if cfg.Mode != nil {
			modeVal := *cfg.Mode
			if modeVal > 0777 || modeVal < 0 {
				return nil, fmt.Errorf("invalid file mode for config %q: %o", cfg.Source, modeVal)
			}
			m := uint32(modeVal) // #nosec G115 - validated range 0-0777
			mode = &m
		}

		mount, err := sc.convertFileObjectToMount(types.FileObjectConfig(projectCfg), cfg.Target, mode, project.Name, "configs", cfg.Source, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to convert config %q: %w", cfg.Source, err)
		}
		if mount.Target == "" {
			mount.Target = "/" + cfg.Source
		}

		result = append(result, mount)
	}

	return result, nil
}

// convertSecretMounts converts compose secrets with local sources to bind mounts.
func (sc *SpecConverter) convertSecretMounts(secrets []types.ServiceSecretConfig, project *types.Project, serviceName string) ([]service.Mount, error) {
	if len(secrets) == 0 {
		return nil, nil
	}

	result := make([]service.Mount, 0, len(secrets))

	for _, sec := range secrets {
		projectSec, exists := project.Secrets[sec.Source]
		if !exists {
			return nil, fmt.Errorf("secret %q referenced by service %q not found in project secrets", sec.Source, serviceName)
		}

		if IsExternal(projectSec.External) {
			continue
		}

		mode := uint32(0400)
		if sec.Mode != nil {
			modeVal := *sec.Mode
			if modeVal > 0777 || modeVal < 0 {
				return nil, fmt.Errorf("invalid file mode for secret %q: %o", sec.Source, modeVal)
			}
			mode = uint32(modeVal) // #nosec G115 - validated range 0-0777
		}

		mount, err := sc.convertFileObjectToMount(types.FileObjectConfig(projectSec), sec.Target, &mode, project.Name, "secrets", sec.Source, 0400)
		if err != nil {
			return nil, fmt.Errorf("failed to convert secret %q: %w", sec.Source, err)
		}
		if mount.Target == "" {
			mount.Target = "/run/secrets/" + sec.Source
		}

		result = append(result, mount)
	}

	return result, nil
}

// convertFileObjectToMount converts a FileObjectConfig to a bind mount.
func (sc *SpecConverter) convertFileObjectToMount(obj types.FileObjectConfig, target string, mode *uint32, projectName, kind, name string, defaultMode uint32) (service.Mount, error) {
	mount := service.Mount{
		Type:     service.MountTypeBind,
		Target:   target,
		ReadOnly: true,
		Options:  make(map[string]string),
	}

	fileMode := os.FileMode(defaultMode)
	if mode != nil {
		fileMode = os.FileMode(*mode)
	}

	if obj.File != "" {
		sourcePath := obj.File
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(sc.workingDir, sourcePath)
		}

		if _, err := os.Stat(sourcePath); err != nil {
			return mount, fmt.Errorf("file source %q not found: %w", obj.File, err)
		}

		mount.Source = sourcePath
		return mount, nil
	}

	if obj.Content != "" {
		tempDir, err := sc.ensureProjectTempDir(projectName, kind)
		if err != nil {
			return mount, err
		}

		filePath, err := sc.writeTempFile(tempDir, name, []byte(obj.Content), fileMode)
		if err != nil {
			return mount, err
		}

		mount.Source = filePath
		return mount, nil
	}

	if obj.Environment != "" {
		value := os.Getenv(obj.Environment)
		if value == "" {
			return mount, fmt.Errorf("environment variable %q is not set or empty", obj.Environment)
		}

		tempDir, err := sc.ensureProjectTempDir(projectName, kind)
		if err != nil {
			return mount, err
		}

		filePath, err := sc.writeTempFile(tempDir, name, []byte(value), fileMode)
		if err != nil {
			return mount, err
		}

		mount.Source = filePath
		return mount, nil
	}

	return mount, fmt.Errorf("no valid local source (file, content, or environment) provided")
}

// convertExternalSecrets converts external secrets to service.Secret for Quadlet Secret= directive.
func (sc *SpecConverter) convertExternalSecrets(secrets []types.ServiceSecretConfig, project *types.Project) []service.Secret {
	result := make([]service.Secret, 0, len(secrets))

	for _, sec := range secrets {
		projectSec, exists := project.Secrets[sec.Source]
		if !exists || !IsExternal(projectSec.External) {
			continue
		}

		secret := service.Secret{
			Source: sec.Source,
			Target: sec.Target,
		}
		if sec.UID != "" {
			secret.UID = sec.UID
		}
		if sec.GID != "" {
			secret.GID = sec.GID
		}
		if sec.Mode != nil {
			secret.Mode = fmt.Sprintf("%o", *sec.Mode)
		}
		result = append(result, secret)
	}

	return result
}

// convertDuration converts a compose duration pointer to time.Duration.
// Returns zero if the pointer is nil.
func (sc *SpecConverter) convertDuration(d *types.Duration) time.Duration {
	if d == nil {
		return 0
	}
	return time.Duration(*d)
}
