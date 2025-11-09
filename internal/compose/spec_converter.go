// Package compose provides Docker Compose project processing functionality
package compose

import (
	"fmt"
	"path/filepath"
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
	// Create sanitized service name
	sanitizedName := service.SanitizeName(Prefix(project.Name, serviceName))

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
		Volumes:     sc.convertProjectVolumes(project),
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
	container := service.Container{
		Image:         composeService.Image,
		Command:       composeService.Command,
		Env:           sc.convertEnvironment(composeService.Environment),
		EnvFiles:      sc.convertEnvFiles(composeService.EnvFiles, serviceName),
		WorkingDir:    composeService.WorkingDir,
		User:          composeService.User,
		Ports:         sc.convertPorts(composeService.Ports),
		Mounts:        sc.convertVolumeMounts(composeService.Volumes, project),
		Resources:     sc.convertResources(composeService.Deploy),
		RestartPolicy: sc.convertRestartPolicy(composeService.Restart),
		Healthcheck:   sc.convertHealthcheck(composeService.HealthCheck),
		Security:      sc.convertSecurity(composeService),
		Build:         sc.convertBuild(composeService.Build, project),
		Labels:        sc.convertLabels(composeService.Labels),
		Hostname:      composeService.Hostname,
		ContainerName: service.SanitizeName(Prefix(project.Name, serviceName)),
		Entrypoint:    composeService.Entrypoint,
		Init:          composeService.Init != nil && *composeService.Init,
		ReadOnly:      composeService.ReadOnly,
		Logging:       sc.convertLogging(composeService.Logging),
		Secrets:       sc.convertSecrets(composeService.Secrets),
		Network:       sc.convertNetworkMode(composeService.NetworkMode, composeService.Networks, project),
		Tmpfs:         sc.convertTmpfs(composeService.Tmpfs),
		Ulimits:       sc.convertUlimits(composeService.Ulimits),
		Sysctls:       composeService.Sysctls,
		UserNS:        composeService.UserNSMode,
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
			if v.Bind != nil && v.Bind.Propagation != "" {
				mount.BindOptions = &service.BindOptions{
					Propagation: v.Bind.Propagation,
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
func (sc *SpecConverter) convertResources(deploy *types.DeployConfig) service.Resources {
	resources := service.Resources{}

	if deploy == nil || deploy.Resources.Limits == nil && deploy.Resources.Reservations == nil {
		return resources
	}

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
func (sc *SpecConverter) convertBuild(build *types.BuildConfig, project *types.Project) *service.Build {
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

// convertSecrets converts compose secrets to service.Secret.
func (sc *SpecConverter) convertSecrets(secrets []types.ServiceSecretConfig) []service.Secret {
	if len(secrets) == 0 {
		return nil
	}

	result := make([]service.Secret, 0, len(secrets))
	for _, s := range secrets {
		secret := service.Secret{
			Source: s.Source,
			Target: s.Target,
		}
		if s.UID != "" {
			secret.UID = s.UID
		}
		if s.GID != "" {
			secret.GID = s.GID
		}
		if s.Mode != nil {
			secret.Mode = fmt.Sprintf("%o", *s.Mode)
		}
		result = append(result, secret)
	}
	return result
}

// convertNetworkMode converts compose network mode to service.NetworkMode.
func (sc *SpecConverter) convertNetworkMode(networkMode string, networks map[string]*types.ServiceNetworkConfig, project *types.Project) service.NetworkMode {
	mode := service.NetworkMode{
		Mode:            networkMode,
		ServiceNetworks: make([]string, 0, len(networks)),
	}

	// Collect aliases and resolve network names
	for networkName, netConfig := range networks {
		if netConfig != nil && len(netConfig.Aliases) > 0 {
			mode.Aliases = append(mode.Aliases, netConfig.Aliases...)
		}

		// Resolve and add network name to ServiceNetworks
		projectNet, exists := project.Networks[networkName]
		if !exists {
			// External or undefined network - use as-is with sanitization
			// Don't apply current project prefix to external networks
			resolvedName := service.SanitizeName(networkName)
			mode.ServiceNetworks = append(mode.ServiceNetworks, resolvedName)
			continue
		}

		// Check if it's an external network
		if IsExternal(projectNet.External) {
			// External network from another project - use as-is
			resolvedName := service.SanitizeName(networkName)
			mode.ServiceNetworks = append(mode.ServiceNetworks, resolvedName)
			continue
		}

		// Resolve network name from project definition
		resolvedName := NameResolver(projectNet.Name, networkName)
		sanitizedName := service.SanitizeName(resolvedName)
		if !strings.Contains(resolvedName, project.Name) {
			sanitizedName = service.SanitizeName(Prefix(project.Name, resolvedName))
		}
		mode.ServiceNetworks = append(mode.ServiceNetworks, sanitizedName)
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

// convertUlimits converts compose ulimits to service.Ulimit.
func (sc *SpecConverter) convertUlimits(ulimits map[string]*types.UlimitsConfig) []service.Ulimit {
	if len(ulimits) == 0 {
		return nil
	}

	result := make([]service.Ulimit, 0, len(ulimits))
	for name, limit := range ulimits {
		if limit != nil {
			result = append(result, service.Ulimit{
				Name: name,
				Soft: int64(limit.Soft),
				Hard: int64(limit.Hard),
			})
		}
	}
	return result
}

// convertDependencies converts compose depends_on to service name list.
func (sc *SpecConverter) convertDependencies(dependsOn types.DependsOnConfig, projectName string) []string {
	if len(dependsOn) == 0 {
		return nil
	}

	result := make([]string, 0, len(dependsOn))
	for serviceName := range dependsOn {
		// Convert to sanitized service name
		result = append(result, service.SanitizeName(Prefix(projectName, serviceName)))
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
		sanitizedName := service.SanitizeName(volumeName)
		if !strings.Contains(volumeName, project.Name) {
			sanitizedName = service.SanitizeName(Prefix(project.Name, volumeName))
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
			resolvedName := service.SanitizeName(networkName)

			network := service.Network{
				Name: resolvedName,
				// Default driver for undefined networks
				Driver: "bridge",
			}
			result = append(result, network)
			continue
		}

		// Resolve network name from project definition
		resolvedName := NameResolver(projectNet.Name, networkName)
		sanitizedName := service.SanitizeName(resolvedName)
		if !strings.Contains(resolvedName, project.Name) {
			sanitizedName = service.SanitizeName(Prefix(project.Name, resolvedName))
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
		sanitizedName := service.SanitizeName(networkName)
		if !strings.Contains(networkName, project.Name) {
			sanitizedName = service.SanitizeName(Prefix(project.Name, networkName))
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
	baseName := service.SanitizeName(Prefix(project.Name, serviceName))

	for i, item := range initList {
		initMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		initSpec := service.Spec{
			Name:        fmt.Sprintf("%s-init-%d", baseName, i),
			Description: fmt.Sprintf("Init container %d for service %s", i, serviceName),
			Container: service.Container{
				Image:   sc.getStringFromMap(initMap, "image"),
				Command: sc.getStringSliceFromMap(initMap, "command"),
			},
			// Init containers don't need volumes/networks from project, only what's specified
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
