// Package compose provides Docker Compose project processing functionality
package compose

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/service"
)

// Converter converts Docker Compose projects to service.Spec models.
type Converter struct {
	workingDir string
}

// NewConverter creates a new Converter.
func NewConverter(workingDir string) *Converter {
	return &Converter{
		workingDir: workingDir,
	}
}

// copyStringMap creates a shallow copy of a string map using stdlib maps.Clone.
// Returns nil if input is nil or empty.
func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	return maps.Clone(in)
}

// formatBytes converts bytes to human-readable format.
func formatBytes(bytes types.UnitBytes) string {
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

// ValidateProjectName validates that a project name follows Docker Compose naming requirements.
// Project names must:
//   - Start with a lowercase letter or digit
//   - Contain only lowercase letters, digits, dashes, and underscores
//   - Not be empty
func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Pattern matches: lowercase letters, digits, dashes, underscores
	// Must start with lowercase letter or digit
	validNamePattern := regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("invalid project name %q: must contain only lowercase letters, digits, dashes, underscores and start with letter/digit", name)
	}

	return nil
}

// ValidateServiceName validates that a service name follows Docker Compose naming requirements.
// Service names must:
//   - Start with an alphanumeric character (upper or lowercase)
//   - Contain only alphanumeric characters, underscores, periods, and dashes
//   - Not be empty
func ValidateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Pattern matches: alphanumeric, underscores, periods, dashes
	// Must start with alphanumeric
	validNamePattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("invalid service name %q: must contain only alphanumeric characters, underscores, periods, dashes and start with alphanumeric", name)
	}

	return nil
}

// ConvertProject converts a Docker Compose project to a list of service specs.
// It normalizes multi-container setups into multiple Spec instances, handling
// services, volumes, networks, and build configurations.
func (c *Converter) ConvertProject(project *types.Project) ([]service.Spec, error) {
	if err := c.validateProject(project); err != nil {
		return nil, err
	}

	specs := make([]service.Spec, 0, len(project.Services))

	// Convert each service to one or more Specs
	for serviceName, composeService := range project.Services {
		serviceSpecs, err := c.convertService(serviceName, composeService, project)
		if err != nil {
			return nil, fmt.Errorf("failed to convert service %s: %w", serviceName, err)
		}
		specs = append(specs, serviceSpecs...)
	}

	return specs, nil
}

// convertService converts a single Docker Compose service to one or more service.Spec instances.
// Init containers are converted to separate specs with dependencies on the main service.
func (c *Converter) convertService(serviceName string, composeService types.ServiceConfig, project *types.Project) ([]service.Spec, error) {
	// Create service name
	sanitizedName := Prefix(project.Name, serviceName)

	// Convert extensions
	initContainers := c.convertInitContainers(serviceName, composeService, project)

	container, err := c.convertContainer(composeService, serviceName, project)
	if err != nil {
		return nil, fmt.Errorf("failed to convert container: %w", err)
	}

	// Dependencies - convert compose depends_on to service name list
	var deps []string
	if len(composeService.DependsOn) > 0 {
		deps = make([]string, 0, len(composeService.DependsOn))
		for serviceName := range composeService.DependsOn {
			// All conditions (service_started, service_healthy, service_completed_successfully)
			// map to systemd After/Requires directives
			deps = append(deps, Prefix(project.Name, serviceName))
		}
		sort.Strings(deps)
	}

	// Extract external dependencies (cross-project)
	externalDeps, err := c.ExtractExternalDependencies(composeService)
	if err != nil {
		return nil, fmt.Errorf("failed to parse external dependencies: %w", err)
	}

	// Create main service spec
	spec := service.Spec{
		Name:                 sanitizedName,
		Description:          fmt.Sprintf("Service %s from project %s", serviceName, project.Name),
		Container:            container,
		Volumes:              c.convertVolumesForService(composeService, project),
		Networks:             c.convertNetworksForService(composeService, project),
		DependsOn:            deps,
		ExternalDependencies: externalDeps,
		Annotations:          copyStringMap(composeService.Labels),
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
func (c *Converter) convertContainer(composeService types.ServiceConfig, serviceName string, project *types.Project) (service.Container, error) {
	mounts, err := c.convertMounts(composeService.Volumes, composeService.Configs, composeService.Secrets, project, serviceName)
	if err != nil {
		return service.Container{}, err
	}

	dns, dnsSearch, dnsOptions := buildDNSConfig(composeService)
	extraHosts := buildExtraHosts(composeService)
	tmpfs := buildTmpfs(composeService)
	devices := buildDevices(composeService)
	deviceCgroupRules := buildDeviceCgroupRules(composeService)
	env := buildEnv(composeService)
	envSecrets, fileSecrets := c.convertSecrets(composeService, project)
	envFiles := buildEnvFiles(composeService, serviceName, c.workingDir)
	restartPolicy := buildRestartPolicy(composeService)
	healthcheck := buildHealthcheck(composeService)
	logging := buildLogging(composeService)
	networkMode := buildNetworkMode(composeService, project)
	ulimits := buildUlimits(composeService)
	security := buildSecurity(composeService)
	build := buildBuild(composeService, project)

	container := service.Container{
		Image:             composeService.Image,
		Command:           composeService.Command,
		Env:               env,
		EnvFiles:          envFiles,
		WorkingDir:        composeService.WorkingDir,
		User:              composeService.User,
		Ports:             c.convertPorts(composeService.Ports),
		Mounts:            mounts,
		Resources:         c.convertResources(composeService.Deploy, composeService),
		RestartPolicy:     restartPolicy,
		Healthcheck:       healthcheck,
		Security:          security,
		Build:             build,
		Labels:            copyStringMap(composeService.Labels),
		Hostname:          composeService.Hostname,
		ContainerName:     toContainerName(Prefix(project.Name, serviceName)),
		Entrypoint:        composeService.Entrypoint,
		Init:              composeService.Init != nil && *composeService.Init,
		ReadOnly:          composeService.ReadOnly,
		Logging:           logging,
		EnvSecrets:        envSecrets,
		Secrets:           fileSecrets,
		Network:           networkMode,
		Tmpfs:             tmpfs,
		Ulimits:           ulimits,
		Sysctls:           composeService.Sysctls,
		UserNS:            composeService.UserNSMode,
		PidMode:           composeService.Pid,
		IpcMode:           composeService.Ipc,
		CgroupMode:        composeService.Cgroup,
		ExtraHosts:        extraHosts,
		DNS:               dns,
		DNSSearch:         dnsSearch,
		DNSOptions:        dnsOptions,
		Devices:           devices,
		DeviceCgroupRules: deviceCgroupRules,
		StopSignal:        composeService.StopSignal,
	}

	// Stop grace period
	if composeService.StopGracePeriod != nil {
		container.StopGracePeriod = time.Duration(*composeService.StopGracePeriod)
	}

	// Handle user/group parsing
	if container.User != "" {
		parts := strings.SplitN(container.User, ":", 2)
		if len(parts) == 2 {
			container.User = parts[0]
			container.Group = parts[1]
		}
	}

	return container, nil
}

// buildDNSConfig extracts DNS configuration from compose service.
func buildDNSConfig(composeService types.ServiceConfig) (dns, dnsSearch, dnsOptions []string) {
	if len(composeService.DNS) > 0 {
		dns = append([]string(nil), composeService.DNS...)
		sort.Strings(dns)
	}
	if len(composeService.DNSSearch) > 0 {
		dnsSearch = append([]string(nil), composeService.DNSSearch...)
		sort.Strings(dnsSearch)
	}
	if len(composeService.DNSOpts) > 0 {
		dnsOptions = append([]string(nil), composeService.DNSOpts...)
		sort.Strings(dnsOptions)
	}
	return
}

// buildExtraHosts extracts extra hosts from compose service.
func buildExtraHosts(composeService types.ServiceConfig) []string {
	if len(composeService.ExtraHosts) == 0 {
		return nil
	}
	extraHosts := composeService.ExtraHosts.AsList(":")
	sort.Strings(extraHosts)
	return extraHosts
}

// buildTmpfs extracts tmpfs configuration from compose service.
func buildTmpfs(composeService types.ServiceConfig) []string {
	if len(composeService.Tmpfs) == 0 {
		return nil
	}
	return []string(composeService.Tmpfs)
}

// buildDevices converts compose devices to device strings.
func buildDevices(composeService types.ServiceConfig) []string {
	if len(composeService.Devices) == 0 {
		return nil
	}
	devices := make([]string, 0, len(composeService.Devices))
	for _, device := range composeService.Devices {
		deviceStr := device.Source
		if device.Target != "" {
			deviceStr = fmt.Sprintf("%s:%s", device.Source, device.Target)
		}
		if device.Permissions != "" {
			deviceStr = fmt.Sprintf("%s:%s", deviceStr, device.Permissions)
		}
		devices = append(devices, deviceStr)
	}
	sort.Strings(devices)
	return devices
}

// buildDeviceCgroupRules extracts device cgroup rules from compose service.
func buildDeviceCgroupRules(composeService types.ServiceConfig) []string {
	if len(composeService.DeviceCgroupRules) == 0 {
		return nil
	}
	rules := append([]string(nil), composeService.DeviceCgroupRules...)
	sort.Strings(rules)
	return rules
}

// buildEnv converts compose environment to map[string]string.
func buildEnv(composeService types.ServiceConfig) map[string]string {
	if composeService.Environment == nil {
		return nil
	}
	env := make(map[string]string, len(composeService.Environment))
	for k, v := range composeService.Environment {
		if v != nil {
			env[k] = *v
		} else {
			env[k] = ""
		}
	}
	return env
}

// buildEnvFiles collects environment files from compose service and auto-discovered files.
func buildEnvFiles(composeService types.ServiceConfig, serviceName, workingDir string) []string {
	var envFiles []string
	for _, ef := range composeService.EnvFiles {
		if ef.Path != "" {
			envFiles = append(envFiles, ef.Path)
		}
	}
	envFiles = append(envFiles, FindEnvFiles(serviceName, workingDir)...)
	sort.Strings(envFiles)
	return envFiles
}

// buildRestartPolicy converts compose restart policy to service.RestartPolicy.
func buildRestartPolicy(composeService types.ServiceConfig) service.RestartPolicy {
	switch composeService.Restart {
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

// buildHealthcheck converts compose healthcheck to service.Healthcheck.
func buildHealthcheck(composeService types.ServiceConfig) *service.Healthcheck {
	if composeService.HealthCheck == nil || composeService.HealthCheck.Disable {
		return nil
	}

	healthcheck := &service.Healthcheck{
		Test: composeService.HealthCheck.Test,
	}

	// Convert retries (uint64 pointer to int)
	if composeService.HealthCheck.Retries != nil {
		retries := *composeService.HealthCheck.Retries
		if retries > 2147483647 {
			retries = 2147483647
		}
		healthcheck.Retries = int(retries) // #nosec G115 - clamped to int max above
	}

	// Convert durations
	if composeService.HealthCheck.Interval != nil {
		healthcheck.Interval = time.Duration(*composeService.HealthCheck.Interval)
	}
	if composeService.HealthCheck.Timeout != nil {
		healthcheck.Timeout = time.Duration(*composeService.HealthCheck.Timeout)
	}
	if composeService.HealthCheck.StartPeriod != nil {
		healthcheck.StartPeriod = time.Duration(*composeService.HealthCheck.StartPeriod)
	}
	if composeService.HealthCheck.StartInterval != nil {
		healthcheck.StartInterval = time.Duration(*composeService.HealthCheck.StartInterval)
	}

	return healthcheck
}

// buildLogging converts compose logging to service.Logging.
func buildLogging(composeService types.ServiceConfig) service.Logging {
	if composeService.Logging == nil {
		return service.Logging{}
	}
	return service.Logging{
		Driver:  composeService.Logging.Driver,
		Options: composeService.Logging.Options,
	}
}

// buildNetworkMode converts compose networks to service.NetworkMode.
func buildNetworkMode(composeService types.ServiceConfig, project *types.Project) service.NetworkMode {
	networkMode := service.NetworkMode{
		Mode:            composeService.NetworkMode,
		ServiceNetworks: make([]string, 0, len(composeService.Networks)),
	}

	if len(composeService.Networks) == 0 && composeService.NetworkMode != "host" {
		// Implicit network assignment - use project networks
		for networkName, projectNet := range project.Networks {
			if IsExternal(projectNet.External) {
				continue
			}
			resolvedName := networkName
			if projectNet.Name != "" {
				resolvedName = projectNet.Name
			}
			sanitizedName := resolvedName
			if !strings.Contains(resolvedName, project.Name) {
				sanitizedName = Prefix(project.Name, resolvedName)
			}
			networkMode.ServiceNetworks = append(networkMode.ServiceNetworks, sanitizedName)
		}
	} else {
		// Explicit networks
		for netName, netConfig := range composeService.Networks {
			if netConfig != nil && len(netConfig.Aliases) > 0 {
				networkMode.Aliases = append(networkMode.Aliases, netConfig.Aliases...)
			}
			projectNet, exists := project.Networks[netName]
			if !exists || IsExternal(projectNet.External) {
				networkMode.ServiceNetworks = append(networkMode.ServiceNetworks, netName)
				continue
			}
			resolvedName := netName
			if projectNet.Name != "" {
				resolvedName = projectNet.Name
			}
			sanitizedName := resolvedName
			if !strings.Contains(resolvedName, project.Name) {
				sanitizedName = Prefix(project.Name, resolvedName)
			}
			networkMode.ServiceNetworks = append(networkMode.ServiceNetworks, sanitizedName)
		}
	}

	if networkMode.Mode == "" {
		networkMode.Mode = "bridge"
	}
	sort.Strings(networkMode.ServiceNetworks)

	return networkMode
}

// buildUlimits converts compose ulimits to service.Ulimit slice.
func buildUlimits(composeService types.ServiceConfig) []service.Ulimit {
	if len(composeService.Ulimits) == 0 {
		return nil
	}
	ulimits := make([]service.Ulimit, 0, len(composeService.Ulimits))
	for name, limit := range composeService.Ulimits {
		if limit != nil {
			soft, hard := int64(limit.Soft), int64(limit.Hard)
			if limit.Single > 0 {
				soft = int64(limit.Single)
				hard = int64(limit.Single)
			}
			ulimits = append(ulimits, service.Ulimit{
				Name: name,
				Soft: soft,
				Hard: hard,
			})
		}
	}
	return ulimits
}

// buildSecurity converts compose security options to service.Security.
func buildSecurity(composeService types.ServiceConfig) service.Security {
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

// buildBuild converts compose build config to service.Build.
func buildBuild(composeService types.ServiceConfig, project *types.Project) *service.Build {
	if composeService.Build == nil {
		return nil
	}

	// Convert build args
	var buildArgs map[string]string
	if composeService.Build.Args != nil {
		buildArgs = make(map[string]string, len(composeService.Build.Args))
		for k, v := range composeService.Build.Args {
			if v != nil {
				buildArgs[k] = *v
			} else {
				buildArgs[k] = ""
			}
		}
	}

	build := &service.Build{
		Context:    composeService.Build.Context,
		Dockerfile: composeService.Build.Dockerfile,
		Target:     composeService.Build.Target,
		Args:       buildArgs,
		Labels:     copyStringMap(composeService.Build.Labels),
		Pull:       composeService.Build.Pull,
		Tags:       composeService.Build.Tags,
	}

	// Convert build context path
	if build.Context == "" {
		build.Context = "."
	}
	if !filepath.IsAbs(build.Context) {
		build.Context = filepath.Join(project.WorkingDir, build.Context)
	}

	// Set dockerfile path
	if build.Dockerfile == "" {
		build.Dockerfile = "Dockerfile"
	}

	// Set working directory for build
	build.SetWorkingDirectory = project.WorkingDir

	return build
}

// convertPorts converts compose ports to service.Port.
func (c *Converter) convertPorts(ports []types.ServicePortConfig) []service.Port {
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

// convertMounts converts all mount sources (volumes, configs, secrets) to service.Mount.
func (c *Converter) convertMounts(volumes []types.ServiceVolumeConfig, configs []types.ServiceConfigObjConfig, secrets []types.ServiceSecretConfig, project *types.Project, serviceName string) ([]service.Mount, error) {
	result := make([]service.Mount, 0, len(volumes)+len(configs)+len(secrets))

	// Add volume mounts (bind, named volumes, tmpfs)
	for _, v := range volumes {
		result = append(result, c.convertVolumeMount(v, project))
	}

	// Add config mounts
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
			if modeVal > 0777 {
				return nil, fmt.Errorf("invalid file mode for config %q: %o", cfg.Source, modeVal)
			}
			m := uint32(modeVal) // #nosec G115 - validated range 0-0777
			mode = &m
		}

		mount, err := c.convertFileObjectToMount(types.FileObjectConfig(projectCfg), cfg.Target, mode, project.Name, "configs", cfg.Source, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to convert config %q: %w", cfg.Source, err)
		}
		if mount.Target == "" {
			mount.Target = "/" + cfg.Source
		}

		result = append(result, mount)
	}

	// Add secret mounts
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
			if modeVal > 0777 {
				return nil, fmt.Errorf("invalid file mode for secret %q: %o", sec.Source, modeVal)
			}
			mode = uint32(modeVal) // #nosec G115 - validated range 0-0777
		}

		mount, err := c.convertFileObjectToMount(types.FileObjectConfig(projectSec), sec.Target, &mode, project.Name, "secrets", sec.Source, 0400)
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

// convertVolumeMount converts a single volume config to a mount.
func (c *Converter) convertVolumeMount(v types.ServiceVolumeConfig, project *types.Project) service.Mount {
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
		if v.Source != "" {
			// Don't prefix external volumes - they're managed outside this project
			if pv, ok := project.Volumes[v.Source]; !ok || !IsExternal(pv.External) {
				mount.Source = Prefix(project.Name, v.Source)
			} else {
				mount.Source = v.Source
			}
		}
	case "tmpfs":
		mount.Type = service.MountTypeTmpfs
		if v.Tmpfs != nil {
			tmpfsOpts := &service.TmpfsOptions{}
			if v.Tmpfs.Size > 0 {
				tmpfsOpts.Size = formatBytes(v.Tmpfs.Size)
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
			if v.Source != "" {
				// Don't prefix external volumes - they're managed outside this project
				if pv, ok := project.Volumes[v.Source]; !ok || !IsExternal(pv.External) {
					mount.Source = Prefix(project.Name, v.Source)
				} else {
					mount.Source = v.Source
				}
			}
		}
	}

	return mount
}

// convertResources converts compose deploy resources to service.Resources.
func (c *Converter) convertResources(deploy *types.DeployConfig, svc types.ServiceConfig) service.Resources {
	resources := service.Resources{}

	// Process deploy resources if present
	if deploy != nil && (deploy.Resources.Limits != nil || deploy.Resources.Reservations != nil) {
		// Limits
		if deploy.Resources.Limits != nil {
			if deploy.Resources.Limits.MemoryBytes > 0 {
				resources.Memory = formatBytes(deploy.Resources.Limits.MemoryBytes)
			}
			if deploy.Resources.Limits.NanoCPUs > 0 {
				// Convert nanoCPUs to quota and period
				// NanoCPUs is a float32 (e.g., 0.5 means 50% of one CPU)
				// Standard CPU period is 100000 microseconds (100ms)
				resources.CPUPeriod = 100000
				resources.CPUQuota = int64(float64(deploy.Resources.Limits.NanoCPUs) * float64(resources.CPUPeriod))
			}
			if deploy.Resources.Limits.Pids > 0 {
				resources.PidsLimit = deploy.Resources.Limits.Pids
			}
		}

		// Reservations
		if deploy.Resources.Reservations != nil {
			if deploy.Resources.Reservations.MemoryBytes > 0 {
				resources.MemoryReservation = formatBytes(deploy.Resources.Reservations.MemoryBytes)
			}
			if deploy.Resources.Reservations.NanoCPUs > 0 {
				// Convert nanoCPUs to CPU shares
				// CPU shares are relative weights (default 1024 = 1 CPU)
				resources.CPUShares = int64(float64(deploy.Resources.Reservations.NanoCPUs) * 1024)
			}
		}
	}

	// MemSwapLimit from service-level field (not deploy.resources)
	if svc.MemSwapLimit > 0 {
		resources.MemorySwap = formatBytes(svc.MemSwapLimit)
	}

	// ShmSize from service-level field
	if svc.ShmSize > 0 {
		resources.ShmSize = formatBytes(svc.ShmSize)
	}

	return resources
}

// convertVolumesForService converts volume declarations to service.Volume.
// Only returns named volumes that the service actually mounts, not all project volumes.
// External volumes are marked but not prefixed (managed outside this project).
func (c *Converter) convertVolumesForService(composeService types.ServiceConfig, project *types.Project) []service.Volume {
	if len(composeService.Volumes) == 0 {
		return nil
	}

	// Collect unique named volumes used by this service
	usedVolumes := make(map[string]bool)
	for _, mount := range composeService.Volumes {
		// Only track named volumes (not bind mounts or tmpfs)
		if mount.Type == "volume" {
			if mount.Source != "" {
				usedVolumes[mount.Source] = true
			}
		} else if mount.Type == "" && mount.Source != "" {
			// Infer type when not explicit: named volume if doesn't look like a path
			if !filepath.IsAbs(mount.Source) &&
				!strings.HasPrefix(mount.Source, "./") &&
				!strings.HasPrefix(mount.Source, "../") &&
				!strings.ContainsAny(mount.Source, "/\\") {
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
			// Use as-is (may be from another project or pre-existing)
			volume := service.Volume{
				Name:     volumeName,
				Driver:   "local",
				External: false,
			}
			result = append(result, volume)
			continue
		}

		// Resolve volume name from project definition
		resolvedName := volumeName
		if projectVol.Name != "" {
			resolvedName = projectVol.Name
		}

		// Apply project prefix unless external
		var sanitizedName string
		if IsExternal(projectVol.External) {
			sanitizedName = resolvedName
		} else {
			sanitizedName = Prefix(project.Name, resolvedName)
		}

		volume := service.Volume{
			Name:     sanitizedName,
			Driver:   projectVol.Driver,
			Options:  projectVol.DriverOpts,
			Labels:   copyStringMap(projectVol.Labels),
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

// convertNetworksForService converts network declarations to service.Network.
// If a service declares specific networks, it connects to those networks.
// If no networks are declared, falls back to project-level networks (excluding external).
// Creates a default network if none exist.
func (c *Converter) convertNetworksForService(composeService types.ServiceConfig, project *types.Project) []service.Network {
	var result []service.Network

	// If service declares specific networks, use those
	if len(composeService.Networks) > 0 {
		result = make([]service.Network, 0, len(composeService.Networks))

		for networkName := range composeService.Networks {
			projectNet, exists := project.Networks[networkName]
			if !exists {
				// Network declared by service but not in project networks
				// Treat as external (from another project/managed outside)
				// Don't prefix - use as-is to allow cross-project networking
				network := service.Network{
					Name:     networkName,
					Driver:   "bridge",
					External: true,
				}
				result = append(result, network)
				continue
			}

			// Resolve network name from project definition
			resolvedName := networkName
			if projectNet.Name != "" {
				resolvedName = projectNet.Name
			}

			// Apply project prefix unless external
			var sanitizedName string
			if IsExternal(projectNet.External) {
				sanitizedName = resolvedName
			} else {
				sanitizedName = Prefix(project.Name, resolvedName)
			}

			network := service.Network{
				Name:     sanitizedName,
				Driver:   projectNet.Driver,
				Options:  projectNet.DriverOpts,
				Labels:   copyStringMap(projectNet.Labels),
				Internal: projectNet.Internal,
				IPv6:     projectNet.EnableIPv6 != nil && *projectNet.EnableIPv6,
				External: IsExternal(projectNet.External),
			}

			// Convert IPAM if present
			if projectNet.Ipam.Driver != "" || len(projectNet.Ipam.Config) > 0 {
				network.IPAM = c.convertIPAM(&projectNet.Ipam)
			}

			result = append(result, network)
		}
	} else {
		// Fall back to project-level networks (skip external)
		result = make([]service.Network, 0, len(project.Networks))

		for name, net := range project.Networks {
			// Skip external networks
			if IsExternal(net.External) {
				continue
			}

			// Resolve network name
			networkName := name
			if net.Name != "" {
				networkName = net.Name
			}
			sanitizedName := Prefix(project.Name, networkName)

			network := service.Network{
				Name:     sanitizedName,
				Driver:   net.Driver,
				Options:  net.DriverOpts,
				Labels:   copyStringMap(net.Labels),
				Internal: net.Internal,
				IPv6:     net.EnableIPv6 != nil && *net.EnableIPv6,
				External: IsExternal(net.External),
			}

			// Convert IPAM if present
			if net.Ipam.Driver != "" || len(net.Ipam.Config) > 0 {
				network.IPAM = c.convertIPAM(&net.Ipam)
			}

			result = append(result, network)
		}
	}

	// Sort for determinism
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// convertIPAM converts compose IPAM to service.IPAM.
func (c *Converter) convertIPAM(ipam *types.IPAMConfig) *service.IPAM {
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

// ExtractExternalDependencies parses x-quad-ops-depends-on extension to extract cross-project dependencies.
// It validates project and service names according to Docker Compose specification.
// Returns nil if the extension is not present.
func (c *Converter) ExtractExternalDependencies(composeService types.ServiceConfig) ([]service.ExternalDependency, error) {
	extension, exists := composeService.Extensions["x-quad-ops-depends-on"]
	if !exists {
		return nil, nil
	}

	depList, ok := extension.([]interface{})
	if !ok {
		return nil, fmt.Errorf("x-quad-ops-depends-on must be a list")
	}

	externalDeps := make([]service.ExternalDependency, 0, len(depList))
	for i, item := range depList {
		depMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("x-quad-ops-depends-on[%d]: must be a map with 'project' and 'service' keys", i)
		}

		// Extract fields with type-safe helpers
		project, _ := depMap["project"].(string)
		svc, _ := depMap["service"].(string)
		optional, _ := depMap["optional"].(bool)

		// Validate required fields
		if project == "" || svc == "" {
			return nil, fmt.Errorf("x-quad-ops-depends-on[%d]: must specify both 'project' and 'service'", i)
		}

		// Validate project name according to compose spec
		if err := ValidateProjectName(project); err != nil {
			return nil, fmt.Errorf("x-quad-ops-depends-on[%d]: invalid project name: %w", i, err)
		}

		// Validate service name according to compose spec
		if err := ValidateServiceName(svc); err != nil {
			return nil, fmt.Errorf("x-quad-ops-depends-on[%d]: invalid service name: %w", i, err)
		}

		dep := service.ExternalDependency{
			Project:  project,
			Service:  svc,
			Optional: optional,
		}
		externalDeps = append(externalDeps, dep)
	}

	return externalDeps, nil
}

// convertInitContainers converts x-quad-ops-init extension to init container specs.
// Init containers are only supported on Linux due to systemd dependency requirements.
func (c *Converter) convertInitContainers(serviceName string, composeService types.ServiceConfig, project *types.Project) []service.Spec {
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
	mainEnv := buildEnv(composeService)
	mainMounts, _ := c.convertMounts(composeService.Volumes, nil, nil, project, serviceName)
	mainNetwork := buildNetworkMode(composeService, project)
	mainVolumes := c.convertVolumesForService(composeService, project)
	mainNetworks := c.convertNetworksForService(composeService, project)

	for i, item := range initList {
		initMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		image, command, initEnv, initVolumes := parseInitContainerConfig(initMap)

		// Build init container config, inheriting from main service
		container := service.Container{
			Image:   image,
			Command: command,
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
			container.Mounts, _ = c.convertInitVolumesToMounts(initVolumes, project, serviceName)
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

// parseInitContainerConfig extracts init container fields from extension config map.
func parseInitContainerConfig(initMap map[string]interface{}) (image string, command []string, env map[string]string, volumes []string) {
	// Extract image (string)
	if v, ok := initMap["image"].(string); ok {
		image = v
	}

	// Extract command (string or []interface{})
	if v, ok := initMap["command"]; ok {
		switch cmd := v.(type) {
		case []interface{}:
			for _, item := range cmd {
				if s, ok := item.(string); ok {
					command = append(command, s)
				}
			}
		case string:
			command = []string{cmd}
		}
	}

	// Extract environment (map[string]interface{} -> map[string]string)
	if v, ok := initMap["environment"].(map[string]interface{}); ok {
		env = make(map[string]string, len(v))
		for k, val := range v {
			if s, ok := val.(string); ok {
				env[k] = s
			}
		}
	}

	// Extract volumes (string or []interface{})
	if v, ok := initMap["volumes"]; ok {
		switch vols := v.(type) {
		case []interface{}:
			for _, item := range vols {
				if s, ok := item.(string); ok {
					volumes = append(volumes, s)
				}
			}
		case string:
			volumes = []string{vols}
		}
	}

	return
}

// convertInitVolumesToMounts converts init volume strings to mounts.
// Supports "source:target" or "source:target:ro" format (subset of Docker volume syntax).
// SELinux and propagation flags not supported in x-quad-ops-init volumes.
func (c *Converter) convertInitVolumesToMounts(initVolumes []string, project *types.Project, serviceName string) ([]service.Mount, error) {
	volumeConfigs := make([]types.ServiceVolumeConfig, 0, len(initVolumes))
	for _, v := range initVolumes {
		parts := strings.Split(v, ":")
		if len(parts) < 2 {
			continue
		}
		vc := types.ServiceVolumeConfig{
			Source: parts[0],
			Target: parts[1],
		}
		// Parse options (third segment): ro, rw, or comma-separated
		if len(parts) > 2 {
			for _, opt := range strings.Split(parts[2], ",") {
				if strings.TrimSpace(opt) == "ro" {
					vc.ReadOnly = true
				}
			}
		}
		volumeConfigs = append(volumeConfigs, vc)
	}
	// Use unified convertMounts (no configs/secrets for init containers, cannot error)
	return c.convertMounts(volumeConfigs, nil, nil, project, serviceName)
}

// validateProject validates project-level configs, secrets, project name, and service names.
func (c *Converter) validateProject(project *types.Project) error {
	// Validate project name
	if err := ValidateProjectName(project.Name); err != nil {
		return err
	}

	// Validate all service names
	for serviceName := range project.Services {
		if err := ValidateServiceName(serviceName); err != nil {
			return err
		}
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

// convertFileObjectToMount converts a FileObjectConfig to a bind mount.
// Handles file, content, and environment sources by creating temp files as needed.
func (c *Converter) convertFileObjectToMount(obj types.FileObjectConfig, target string, mode *uint32, projectName, kind, name string, defaultMode uint32) (service.Mount, error) {
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

	// File source - use existing file
	if obj.File != "" {
		sourcePath := obj.File
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(c.workingDir, sourcePath)
		}

		if _, err := os.Stat(sourcePath); err != nil {
			return mount, fmt.Errorf("file source %q not found: %w", obj.File, err)
		}

		mount.Source = sourcePath
		return mount, nil
	}

	// Content or environment source - create temp file
	var data []byte
	if obj.Content != "" {
		data = []byte(obj.Content)
	} else if obj.Environment != "" {
		value := os.Getenv(obj.Environment)
		if value == "" {
			return mount, fmt.Errorf("environment variable %q is not set or empty", obj.Environment)
		}
		data = []byte(value)
	} else {
		return mount, fmt.Errorf("no valid local source (file, content, or environment) provided")
	}

	// Create temp directory: /tmp/quad-ops/{project}/{kind}/
	tempDir := filepath.Join(os.TempDir(), "quad-ops", projectName, kind)
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return mount, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Write temp file with atomic permissions
	filePath := filepath.Join(tempDir, name)
	if _, err := os.Stat(filePath); err == nil {
		if err := os.Chmod(filePath, 0600); err != nil {
			return mount, fmt.Errorf("failed to make file writable: %w", err)
		}
	}
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return mount, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Chmod(filePath, fileMode); err != nil {
		return mount, fmt.Errorf("failed to set file mode: %w", err)
	}

	mount.Source = filePath
	return mount, nil
}

// convertSecrets converts both env secrets (x-podman-env-secrets) and external secrets.
// Returns envSecrets map and external secrets slice, both deterministically sorted.
func (c *Converter) convertSecrets(composeService types.ServiceConfig, project *types.Project) (map[string]string, []service.Secret) {
	// Convert env secrets from x-podman-env-secrets extension
	var envSecrets map[string]string
	if extension, exists := composeService.Extensions["x-podman-env-secrets"]; exists {
		if envSecretsMap, ok := extension.(map[string]interface{}); ok {
			envSecrets = make(map[string]string)
			for secretName, envVar := range envSecretsMap {
				if envVarStr, ok := envVar.(string); ok {
					envSecrets[secretName] = envVarStr
				}
			}
		}
	}

	// Convert external secrets to service.Secret for Quadlet Secret= directive
	fileSecrets := make([]service.Secret, 0)
	if len(composeService.Secrets) > 0 {
		for _, sec := range composeService.Secrets {
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
			fileSecrets = append(fileSecrets, secret)
		}

		// Sort for determinism
		sort.Slice(fileSecrets, func(i, j int) bool {
			return fileSecrets[i].Source < fileSecrets[j].Source
		})
	}

	return envSecrets, fileSecrets
}

// Prefix creates a prefixed resource name using project name and resource name.
// No sanitization is performed; the projectName must already be valid according to
// the service name regex: ^[a-zA-Z0-9][a-zA-Z0-9_.-]*$.
// If the name is already prefixed (starts with projectName_ or projectName-), returns it unchanged.
func Prefix(projectName, resourceName string) string {
	// Don't double-prefix
	if strings.HasPrefix(resourceName, projectName+"_") || strings.HasPrefix(resourceName, projectName+"-") {
		return resourceName
	}
	return fmt.Sprintf("%s_%s", projectName, resourceName)
}

// toContainerName converts resource names to valid Podman container names.
// Replaces underscores with hyphens since Podman requires DNS-compatible names.
func toContainerName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

// FindEnvFiles discovers environment files for a service in a working directory.
func FindEnvFiles(serviceName, workingDir string) []string {
	if workingDir == "" {
		return nil
	}

	var envFiles []string

	// General .env file
	generalEnvFile := filepath.Join(workingDir, ".env")
	if _, err := os.Stat(generalEnvFile); err == nil {
		envFiles = append(envFiles, generalEnvFile)
	}

	// Service-specific .env files
	possibleEnvFiles := []string{
		filepath.Join(workingDir, fmt.Sprintf(".env.%s", serviceName)),
		filepath.Join(workingDir, fmt.Sprintf("%s.env", serviceName)),
		filepath.Join(workingDir, "env", fmt.Sprintf("%s.env", serviceName)),
		filepath.Join(workingDir, "envs", fmt.Sprintf("%s.env", serviceName)),
	}

	for _, envFilePath := range possibleEnvFiles {
		if _, err := os.Stat(envFilePath); err == nil {
			envFiles = append(envFiles, envFilePath)
		}
	}

	return envFiles
}

// IsExternal checks if a resource configuration indicates it's externally managed.
func IsExternal(external interface{}) bool {
	if external == nil {
		return false
	}

	switch v := external.(type) {
	case bool:
		return v
	case *bool:
		return v != nil && *v
	default:
		// Handle types.External which is a custom bool type with underlying bool
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Bool {
			return rv.Bool()
		}
		return false
	}
}
