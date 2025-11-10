package systemd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/sorting"
)

// Unit type suffix constants for Quadlet unit files.
// These suffixes distinguish different types of managed resources in systemd dependencies.
const (
	UnitSuffixContainer = ".container"
	UnitSuffixNetwork   = ".network"
	UnitSuffixVolume    = ".volume"
	UnitSuffixBuild     = ".build"
	UnitSuffixService   = ".service"
)

// knownUnitSuffixes lists all recognized Quadlet unit type suffixes.
// Used for dependency resolution and validation.
var knownUnitSuffixes = []string{
	UnitSuffixNetwork,
	UnitSuffixVolume,
	".pod",
	".kube",
	UnitSuffixBuild,
	".image",
	".artifact",
	UnitSuffixService,
}

// Renderer implements platform.Renderer for systemd/Quadlet.
type Renderer struct {
	logger log.Logger
}

// NewRenderer creates a new systemd renderer.
func NewRenderer(logger log.Logger) *Renderer {
	return &Renderer{
		logger: logger,
	}
}

// Name returns the platform name.
func (r *Renderer) Name() string {
	return "systemd"
}

// Render converts service specs to systemd Quadlet unit files.
func (r *Renderer) Render(_ context.Context, specs []service.Spec) (*platform.RenderResult, error) {
	result := &platform.RenderResult{
		Artifacts:      make([]platform.Artifact, 0),
		ServiceChanges: make(map[string]platform.ChangeStatus),
	}

	for _, spec := range specs {
		artifacts, err := r.renderService(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to render service %s: %w", spec.Name, err)
		}

		artifactPaths := make([]string, 0, len(artifacts))
		contentHashes := make([]string, 0, len(artifacts))

		for _, artifact := range artifacts {
			artifactPaths = append(artifactPaths, artifact.Path)
			contentHashes = append(contentHashes, artifact.Hash)
		}

		combinedHash := r.combineHashes(contentHashes)

		result.Artifacts = append(result.Artifacts, artifacts...)
		result.ServiceChanges[spec.Name] = platform.ChangeStatus{
			Changed:       false,
			ArtifactPaths: artifactPaths,
			ContentHash:   combinedHash,
		}
	}

	return result, nil
}

// renderService renders a single service spec into one or more artifacts.
func (r *Renderer) renderService(spec service.Spec) ([]platform.Artifact, error) {
	artifacts := make([]platform.Artifact, 0)

	// Render volumes first
	for _, vol := range spec.Volumes {
		if !vol.External {
			content := r.renderVolume(vol)
			hash := r.computeHash(content)
			artifacts = append(artifacts, platform.Artifact{
				Path:    vol.Name + UnitSuffixVolume,
				Content: []byte(content),
				Mode:    0644,
				Hash:    hash,
			})
		}
	}

	// Render networks
	for _, net := range spec.Networks {
		if !net.External {
			content := r.renderNetwork(net)
			hash := r.computeHash(content)
			artifacts = append(artifacts, platform.Artifact{
				Path:    net.Name + UnitSuffixNetwork,
				Content: []byte(content),
				Mode:    0644,
				Hash:    hash,
			})
		}
	}

	// Render build unit if needed
	if spec.Container.Build != nil {
		content := r.renderBuild(spec.Name, spec.Description, *spec.Container.Build, spec.DependsOn)
		hash := r.computeHash(content)
		artifacts = append(artifacts, platform.Artifact{
			Path:    spec.Name + "-build" + UnitSuffixBuild,
			Content: []byte(content),
			Mode:    0644,
			Hash:    hash,
		})
	}

	// Render container unit
	content := r.renderContainer(spec)
	hash := r.computeHash(content)
	artifacts = append(artifacts, platform.Artifact{
		Path:    spec.Name + UnitSuffixContainer,
		Content: []byte(content),
		Mode:    0644,
		Hash:    hash,
	})

	return artifacts, nil
}

// renderContainer renders a container service spec to a .container unit file.
func (r *Renderer) renderContainer(spec service.Spec) string {
	var builder strings.Builder

	builder.WriteString("[Unit]\n")
	if spec.Description != "" {
		builder.WriteString(formatKeyValue("Description", spec.Description))
	}

	// Add network-online.target dependency if container uses networks or has published ports
	needsNetworkOnline := r.needsNetworkOnline(spec)
	if needsNetworkOnline {
		builder.WriteString("After=network-online.target\n")
		builder.WriteString("Wants=network-online.target\n")
	}

	// Add RequiresMountsFor directives for bind mounts
	bindMountPaths := r.collectBindMountPaths(spec.Container)
	if len(bindMountPaths) > 0 {
		sort.Strings(bindMountPaths)
		for _, path := range bindMountPaths {
			builder.WriteString(fmt.Sprintf("RequiresMountsFor=%s\n", path))
		}
	}

	if len(spec.DependsOn) > 0 {
		deps := make([]string, len(spec.DependsOn))
		copy(deps, spec.DependsOn)
		sort.Strings(deps)
		for _, dep := range deps {
			// If dependency already has a unit type suffix, use as-is
			// Otherwise, append .service for service-to-service dependencies
			depUnit := r.formatDependency(dep)
			builder.WriteString(fmt.Sprintf("After=%s\n", depUnit))
			builder.WriteString(fmt.Sprintf("Requires=%s\n", depUnit))
		}
	}

	// Add dependencies for volumes
	if len(spec.Volumes) > 0 {
		// Sort volumes by name for deterministic output
		volumes := make([]service.Volume, len(spec.Volumes))
		copy(volumes, spec.Volumes)
		sort.Slice(volumes, func(i, j int) bool {
			return volumes[i].Name < volumes[j].Name
		})

		for _, vol := range volumes {
			if !vol.External {
				builder.WriteString(fmt.Sprintf("After=%s%s\n", vol.Name, UnitSuffixVolume))
				builder.WriteString(fmt.Sprintf("Requires=%s%s\n", vol.Name, UnitSuffixVolume))
			}
		}
	}

	// Add dependencies for networks that this container actually uses.
	// ServiceNetworks contains sanitized names that exactly match spec.Networks names.
	// We only add dependencies for networks explicitly used by the container,
	// not for all project-level networks.
	externalNetworks := make(map[string]bool)

	// Build a map of which networks are external for reference
	for _, net := range spec.Networks {
		if net.External {
			externalNetworks[net.Name] = true
		}
	}

	if len(spec.Container.Network.ServiceNetworks) > 0 {
		// Sort for deterministic output
		networks := make([]string, len(spec.Container.Network.ServiceNetworks))
		copy(networks, spec.Container.Network.ServiceNetworks)
		sort.Strings(networks)

		for _, net := range networks {
			// Only add systemd dependencies (After/Requires) for networks that are created by quad-ops.
			// External networks are not created by quad-ops and should not appear in systemd dependencies,
			// as the quadlet-generator cannot translate dependencies to non-existent unit files.
			// External networks are still added to the container's Network directive for connectivity.
			if !externalNetworks[net] {
				builder.WriteString(fmt.Sprintf("After=%s%s\n", net, UnitSuffixNetwork))
				builder.WriteString(fmt.Sprintf("Requires=%s%s\n", net, UnitSuffixNetwork))
			}
		}
	}

	// Add dependencies for build
	if spec.Container.Build != nil {
		buildUnit := spec.Name + "-build" + UnitSuffixService
		builder.WriteString(fmt.Sprintf("After=%s\n", buildUnit))
		builder.WriteString(fmt.Sprintf("Requires=%s\n", buildUnit))
	}

	builder.WriteString("\n[Container]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	if spec.Container.Image != "" {
		builder.WriteString(formatKeyValue("Image", spec.Container.Image))
	}

	if spec.Container.ContainerName != "" {
		builder.WriteString(formatKeyValue("ContainerName", spec.Container.ContainerName))
	}

	if spec.Container.Hostname != "" {
		builder.WriteString(formatKeyValue("HostName", spec.Container.Hostname))
	}

	r.addEnvironment(&builder, spec.Container)
	r.addPorts(&builder, spec.Container)
	r.addMounts(&builder, spec.Container)
	r.addNetworks(&builder, spec)
	r.addDNS(&builder, spec.Container)
	r.addDevices(&builder, spec.Container)
	r.addDeviceCgroupRules(&builder, spec.Container)
	r.addExecution(&builder, spec.Container)
	r.addHealthcheck(&builder, spec.Container)
	r.addResources(&builder, spec.Container)
	r.addSecurity(&builder, spec.Container)
	r.addLogging(&builder, spec.Container)
	r.addSecrets(&builder, spec.Container)
	r.addExtraHosts(&builder, spec.Container)
	r.addStopConfiguration(&builder, spec.Container)
	r.addAdvanced(&builder, spec.Container)

	builder.WriteString("\n[Service]\n")

	// Configure init containers as oneshot services
	if strings.Contains(spec.Name, "-init-") {
		builder.WriteString(formatKeyValue("Type", "oneshot"))
		builder.WriteString(formatKeyValue("RemainAfterExit", "yes"))
	}

	restart := r.mapRestartPolicy(spec.Container.RestartPolicy)
	builder.WriteString(formatKeyValue("Restart", restart))

	// Set timeout for image pull (default 15 minutes = 900 seconds)
	// This prevents systemd's default 90-second timeout from killing long image pulls
	builder.WriteString(formatKeyValue("TimeoutStartSec", "900"))

	builder.WriteString("\n[Install]\n")
	builder.WriteString("WantedBy=default.target\n")

	return builder.String()
}

// addEnvironment adds environment variables and files.
func (r *Renderer) addEnvironment(builder *strings.Builder, c service.Container) {
	envKeys := sorting.GetSortedMapKeys(c.Env)
	for _, k := range envKeys {
		fmt.Fprintf(builder, "Environment=%s=%s\n", k, c.Env[k])
	}

	if len(c.EnvFiles) > 0 {
		sorted := make([]string, len(c.EnvFiles))
		copy(sorted, c.EnvFiles)
		sort.Strings(sorted)
		for _, f := range sorted {
			builder.WriteString(formatKeyValue("EnvironmentFile", f))
		}
	}
}

// addPorts adds port mappings.
func (r *Renderer) addPorts(builder *strings.Builder, c service.Container) {
	if len(c.Ports) == 0 {
		return
	}

	ports := make([]string, 0, len(c.Ports))
	for _, p := range c.Ports {
		portStr := ""
		if p.Host != "" {
			portStr = fmt.Sprintf("%s:%d:%d", p.Host, p.HostPort, p.Container)
		} else if p.HostPort > 0 {
			portStr = fmt.Sprintf("%d:%d", p.HostPort, p.Container)
		} else {
			portStr = fmt.Sprintf("%d", p.Container)
		}
		if p.Protocol != "" && p.Protocol != "tcp" {
			portStr += "/" + p.Protocol
		}
		ports = append(ports, portStr)
	}

	sort.Strings(ports)
	for _, p := range ports {
		builder.WriteString(formatKeyValue("PublishPort", p))
	}
}

// addMounts adds volume and bind mounts.
func (r *Renderer) addMounts(builder *strings.Builder, c service.Container) {
	if len(c.Mounts) == 0 {
		return
	}

	mounts := make([]string, 0, len(c.Mounts))
	tmpfsMounts := make([]string, 0)
	for _, m := range c.Mounts {
		if m.Type == service.MountTypeTmpfs {
			tmpfsStr := m.Target
			var options []string
			if m.ReadOnly {
				options = append(options, "ro")
			} else {
				options = append(options, "rw")
			}
			if m.TmpfsOptions != nil {
				if m.TmpfsOptions.Size != "" {
					options = append(options, "size="+m.TmpfsOptions.Size)
				}
				if m.TmpfsOptions.Mode != 0 {
					options = append(options, fmt.Sprintf("mode=%d", m.TmpfsOptions.Mode))
				}
				if m.TmpfsOptions.UID != 0 {
					options = append(options, fmt.Sprintf("uid=%d", m.TmpfsOptions.UID))
				}
				if m.TmpfsOptions.GID != 0 {
					options = append(options, fmt.Sprintf("gid=%d", m.TmpfsOptions.GID))
				}
			}
			if len(options) > 0 {
				tmpfsStr += ":" + strings.Join(options, ",")
			}
			tmpfsMounts = append(tmpfsMounts, tmpfsStr)
			continue
		}

		source := m.Source
		// Note: Do NOT append .volume suffix. Quadlet resolves named volumes automatically.
		// The .volume suffix is only needed in Unit file dependencies (After=, Requires=),
		// which are handled separately in renderContainer().
		mountStr := fmt.Sprintf("%s:%s", source, m.Target)

		var options []string
		if m.ReadOnly {
			options = append(options, "ro")
		}
		if m.BindOptions != nil && m.BindOptions.SELinux != "" {
			options = append(options, m.BindOptions.SELinux)
		}
		if len(options) > 0 {
			mountStr += ":" + strings.Join(options, ",")
		}

		mounts = append(mounts, mountStr)
	}

	sort.Strings(mounts)
	for _, m := range mounts {
		builder.WriteString(formatKeyValue("Volume", m))
	}

	if len(tmpfsMounts) > 0 {
		sort.Strings(tmpfsMounts)
		for _, t := range tmpfsMounts {
			builder.WriteString(formatKeyValue("Tmpfs", t))
		}
	}

	if len(c.Tmpfs) > 0 {
		sorted := make([]string, len(c.Tmpfs))
		copy(sorted, c.Tmpfs)
		sort.Strings(sorted)
		for _, t := range sorted {
			builder.WriteString(formatKeyValue("Tmpfs", t))
		}
	}
}

// addNetworks adds network configuration.
func (r *Renderer) addNetworks(builder *strings.Builder, spec service.Spec) {
	c := spec.Container

	if c.Network.Mode != "" && c.Network.Mode != "bridge" {
		builder.WriteString(formatKeyValue("Network", c.Network.Mode))
	}

	if len(c.Network.Aliases) > 0 {
		sorted := make([]string, len(c.Network.Aliases))
		copy(sorted, c.Network.Aliases)
		sort.Strings(sorted)
		for _, alias := range sorted {
			builder.WriteString(formatKeyValue("NetworkAlias", alias))
		}
	}

	// Add Network directives for service-specific networks with .network suffix.
	// ServiceNetworks contains sanitized names that exactly match spec.Networks names.
	// This enables service-to-service DNS resolution and automatic Quadlet dependencies.
	if len(c.Network.ServiceNetworks) > 0 {
		// Sort for deterministic output
		networks := make([]string, len(c.Network.ServiceNetworks))
		copy(networks, c.Network.ServiceNetworks)
		sort.Strings(networks)

		for _, net := range networks {
			builder.WriteString(formatKeyValue("Network", net+UnitSuffixNetwork))
		}
	}
	// Note: We do NOT have a fallback to project-level networks here.
	// The compose parser's convertServiceNetworks() already handles the logic of
	// what networks a service should use. If ServiceNetworks is empty, the container
	// will use the default network which is the correct behavior.
}

// addExecution adds execution configuration.
func (r *Renderer) addExecution(builder *strings.Builder, c service.Container) {
	if len(c.Entrypoint) > 0 {
		builder.WriteString("Entrypoint=" + strings.Join(c.Entrypoint, " ") + "\n")
	}

	if len(c.Command) > 0 {
		builder.WriteString("Exec=" + strings.Join(c.Command, " ") + "\n")
	}

	if c.User != "" {
		builder.WriteString(formatKeyValue("User", c.User))
	}

	if c.Group != "" {
		builder.WriteString(formatKeyValue("Group", c.Group))
	}

	if c.WorkingDir != "" {
		builder.WriteString(formatKeyValue("WorkingDir", c.WorkingDir))
	}

	if c.Init {
		builder.WriteString(formatKeyValue("RunInit", "yes"))
	}

	if c.ReadOnly {
		builder.WriteString(formatKeyValue("ReadOnly", "yes"))
	}
}

// addHealthcheck adds healthcheck configuration.
func (r *Renderer) addHealthcheck(builder *strings.Builder, c service.Container) {
	if c.Healthcheck == nil {
		return
	}

	hc := c.Healthcheck
	if len(hc.Test) > 0 {
		if len(hc.Test) == 2 && (hc.Test[0] == "CMD" || hc.Test[0] == "CMD-SHELL") {
			fmt.Fprintf(builder, "HealthCmd=%s %s\n", hc.Test[0], hc.Test[1])
		} else {
			builder.WriteString("HealthCmd=" + strings.Join(hc.Test, " ") + "\n")
		}
	}

	if hc.Interval > 0 {
		builder.WriteString(formatKeyValue("HealthInterval", formatDuration(hc.Interval)))
	}

	if hc.Timeout > 0 {
		builder.WriteString(formatKeyValue("HealthTimeout", formatDuration(hc.Timeout)))
	}

	if hc.Retries > 0 {
		fmt.Fprintf(builder, "HealthRetries=%d\n", hc.Retries)
	}

	if hc.StartPeriod > 0 {
		builder.WriteString(formatKeyValue("HealthStartPeriod", formatDuration(hc.StartPeriod)))
	}

	if hc.StartInterval > 0 {
		builder.WriteString(formatKeyValue("HealthStartupInterval", formatDuration(hc.StartInterval)))
	}
}

// addResources adds resource constraints.
func (r *Renderer) addResources(builder *strings.Builder, c service.Container) {
	// Memory constraints (Quadlet native directives)
	if c.Resources.Memory != "" {
		builder.WriteString(formatKeyValue("Memory", c.Resources.Memory))
	}

	if c.Resources.MemoryReservation != "" {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--memory-reservation %s", c.Resources.MemoryReservation)))
	}

	if c.Resources.MemorySwap != "" {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--memory-swap %s", c.Resources.MemorySwap)))
	}

	if c.Resources.ShmSize != "" {
		builder.WriteString(formatKeyValue("ShmSize", c.Resources.ShmSize))
	}

	// CPU constraints (rendered as PodmanArgs)
	if c.Resources.CPUShares > 0 {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cpu-shares %d", c.Resources.CPUShares)))
	}

	if c.Resources.CPUQuota > 0 {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cpu-quota %d", c.Resources.CPUQuota)))
	}

	if c.Resources.CPUPeriod > 0 {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cpu-period %d", c.Resources.CPUPeriod)))
	}

	// Use Resources.PidsLimit as canonical source; Container.PidsLimit is deprecated
	if c.Resources.PidsLimit > 0 {
		fmt.Fprintf(builder, "PidsLimit=%d\n", c.Resources.PidsLimit)
	} else if c.PidsLimit > 0 {
		fmt.Fprintf(builder, "PidsLimit=%d\n", c.PidsLimit)
	}

	if len(c.Ulimits) > 0 {
		ulimits := make([]string, 0, len(c.Ulimits))
		for _, u := range c.Ulimits {
			if u.Soft == u.Hard {
				ulimits = append(ulimits, fmt.Sprintf("%s=%d", u.Name, u.Soft))
			} else {
				ulimits = append(ulimits, fmt.Sprintf("%s=%d:%d", u.Name, u.Soft, u.Hard))
			}
		}
		sort.Strings(ulimits)
		for _, u := range ulimits {
			builder.WriteString(formatKeyValue("Ulimit", u))
		}
	}

	if len(c.Sysctls) > 0 {
		keys := sorting.GetSortedMapKeys(c.Sysctls)
		for _, k := range keys {
			fmt.Fprintf(builder, "Sysctl=%s=%s\n", k, c.Sysctls[k])
		}
	}
}

// addSecurity adds security configuration.
func (r *Renderer) addSecurity(builder *strings.Builder, c service.Container) {
	if c.Security.Privileged {
		builder.WriteString(formatKeyValue("PodmanArgs", "--privileged"))
	}

	for _, cap := range c.Security.CapAdd {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cap-add=%s", cap)))
	}

	for _, cap := range c.Security.CapDrop {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cap-drop=%s", cap)))
	}

	for _, opt := range c.Security.SecurityOpt {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--security-opt=%s", opt)))
	}

	for _, group := range c.Security.GroupAdd {
		builder.WriteString(formatKeyValue("GroupAdd", group))
	}

	if c.UserNS != "" {
		builder.WriteString(formatKeyValue("UserNS", c.UserNS))
	}
}

// addLogging adds logging configuration.
func (r *Renderer) addLogging(builder *strings.Builder, c service.Container) {
	if c.Logging.Driver != "" {
		builder.WriteString(formatKeyValue("LogDriver", c.Logging.Driver))
	}

	if len(c.Logging.Options) > 0 {
		keys := sorting.GetSortedMapKeys(c.Logging.Options)
		for _, k := range keys {
			fmt.Fprintf(builder, "LogOpt=%s=%s\n", k, c.Logging.Options[k])
		}
	}
}

// addSecrets adds secrets configuration.
func (r *Renderer) addSecrets(builder *strings.Builder, c service.Container) {
	if len(c.Secrets) == 0 && len(c.EnvSecrets) == 0 {
		return
	}

	secrets := make([]service.Secret, len(c.Secrets))
	copy(secrets, c.Secrets)
	sort.Slice(secrets, func(i, j int) bool {
		return secrets[i].Source < secrets[j].Source
	})

	for _, s := range secrets {
		secretStr := s.Source
		if s.Target != "" {
			secretStr += ",target=" + s.Target
		}
		if s.Type != "" {
			secretStr += ",type=" + s.Type
		}
		if s.UID != "" {
			secretStr += ",uid=" + s.UID
		}
		if s.GID != "" {
			secretStr += ",gid=" + s.GID
		}
		if s.Mode != "" {
			secretStr += ",mode=" + s.Mode
		}
		builder.WriteString(formatKeyValue("Secret", secretStr))
	}

	// Add environment secrets
	envSecretKeys := sorting.GetSortedMapKeys(c.EnvSecrets)
	for _, secretName := range envSecretKeys {
		envVarName := c.EnvSecrets[secretName]
		secretStr := fmt.Sprintf("%s,type=env,target=%s", secretName, envVarName)
		builder.WriteString(formatKeyValue("Secret", secretStr))
	}
}

// addExtraHosts adds extra host-to-IP mappings.
func (r *Renderer) addDNS(builder *strings.Builder, c service.Container) {
	// Add DNS servers
	if len(c.DNS) > 0 {
		sorted := make([]string, len(c.DNS))
		copy(sorted, c.DNS)
		sort.Strings(sorted)

		for _, dns := range sorted {
			builder.WriteString(formatKeyValue("DNS", dns))
		}
	}

	// Add DNS search domains
	if len(c.DNSSearch) > 0 {
		sorted := make([]string, len(c.DNSSearch))
		copy(sorted, c.DNSSearch)
		sort.Strings(sorted)

		for _, domain := range sorted {
			builder.WriteString(formatKeyValue("DNSSearch", domain))
		}
	}

	// Add DNS options
	if len(c.DNSOptions) > 0 {
		sorted := make([]string, len(c.DNSOptions))
		copy(sorted, c.DNSOptions)
		sort.Strings(sorted)

		for _, opt := range sorted {
			builder.WriteString(formatKeyValue("DNSOption", opt))
		}
	}
}

func (r *Renderer) addDevices(builder *strings.Builder, c service.Container) {
	if len(c.Devices) == 0 {
		return
	}

	// Devices are already in "host:container" or "host:container:permissions" format from converter
	sorted := make([]string, len(c.Devices))
	copy(sorted, c.Devices)
	sort.Strings(sorted)

	for _, device := range sorted {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--device=%s", device)))
	}
}

func (r *Renderer) addDeviceCgroupRules(builder *strings.Builder, c service.Container) {
	if len(c.DeviceCgroupRules) == 0 {
		return
	}

	// Device cgroup rules are in "type major:minor permissions" format from converter
	sorted := make([]string, len(c.DeviceCgroupRules))
	copy(sorted, c.DeviceCgroupRules)
	sort.Strings(sorted)

	for _, rule := range sorted {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--device-cgroup-rule=%s", rule)))
	}
}

func (r *Renderer) addExtraHosts(builder *strings.Builder, c service.Container) {
	if len(c.ExtraHosts) == 0 {
		return
	}

	// ExtraHosts is already in "hostname:ip" format from converter
	// Sort for determinism (should already be sorted, but ensure it)
	sorted := make([]string, len(c.ExtraHosts))
	copy(sorted, c.ExtraHosts)
	sort.Strings(sorted)

	for _, host := range sorted {
		builder.WriteString(formatKeyValue("AddHost", host))
	}
}

// addStopConfiguration adds stop signal and timeout configuration.
func (r *Renderer) addStopConfiguration(builder *strings.Builder, c service.Container) {
	// Map stop signal to Quadlet StopSignal directive
	// If empty, Quadlet will use the container's default (SIGTERM)
	if c.StopSignal != "" {
		// Remove "SIG" prefix if present (e.g., "SIGTERM" -> "TERM")
		signal := strings.TrimPrefix(c.StopSignal, "SIG")
		builder.WriteString(formatKeyValue("StopSignal", signal))
	}

	// Map stop grace period to Quadlet StopTimeoutSec directive
	// Convert duration to seconds. Default is 10s if not specified.
	if c.StopGracePeriod > 0 {
		seconds := int(c.StopGracePeriod.Seconds())
		fmt.Fprintf(builder, "StopTimeoutSec=%d\n", seconds)
	}
}

// addAdvanced adds advanced Podman arguments.
func (r *Renderer) addAdvanced(builder *strings.Builder, c service.Container) {
	// Namespace modes (no native Quadlet directives, use PodmanArgs)
	if c.PidMode != "" {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--pid=%s", c.PidMode)))
	}
	if c.IpcMode != "" {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--ipc=%s", c.IpcMode)))
	}
	if c.CgroupMode != "" {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cgroupns=%s", c.CgroupMode)))
	}

	if len(c.PodmanArgs) > 0 {
		sorted := make([]string, len(c.PodmanArgs))
		copy(sorted, c.PodmanArgs)
		sort.Strings(sorted)
		for _, arg := range sorted {
			builder.WriteString(formatKeyValue("PodmanArgs", arg))
		}
	}

	if len(c.Labels) > 0 {
		keys := sorting.GetSortedMapKeys(c.Labels)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Label", fmt.Sprintf("%s=%s", k, c.Labels[k])))
		}
	}
}

// renderVolume renders a volume spec to a .volume unit file.
func (r *Renderer) renderVolume(vol service.Volume) string {
	var builder strings.Builder

	builder.WriteString("[Unit]\n")
	builder.WriteString(formatKeyValue("Description", fmt.Sprintf("Volume %s", vol.Name)))

	builder.WriteString("\n[Volume]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	if vol.Name != "" {
		builder.WriteString(formatKeyValue("VolumeName", vol.Name))
	}

	if vol.Driver != "" && vol.Driver != "local" {
		builder.WriteString(formatKeyValue("Driver", vol.Driver))
	}

	if len(vol.Options) > 0 {
		keys := sorting.GetSortedMapKeys(vol.Options)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Options", fmt.Sprintf("%s=%s", k, vol.Options[k])))
		}
	}

	if len(vol.Labels) > 0 {
		keys := sorting.GetSortedMapKeys(vol.Labels)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Label", fmt.Sprintf("%s=%s", k, vol.Labels[k])))
		}
	}

	// Add Quadlet-specific extensions if present
	if vol.Quadlet != nil {
		if len(vol.Quadlet.ContainersConfModule) > 0 {
			sorted := make([]string, len(vol.Quadlet.ContainersConfModule))
			copy(sorted, vol.Quadlet.ContainersConfModule)
			sort.Strings(sorted)
			for _, module := range sorted {
				builder.WriteString(formatKeyValue("ContainersConfModule", module))
			}
		}

		if len(vol.Quadlet.GlobalArgs) > 0 {
			sorted := make([]string, len(vol.Quadlet.GlobalArgs))
			copy(sorted, vol.Quadlet.GlobalArgs)
			sort.Strings(sorted)
			for _, arg := range sorted {
				builder.WriteString(formatKeyValue("GlobalArgs", arg))
			}
		}

		if len(vol.Quadlet.PodmanArgs) > 0 {
			sorted := make([]string, len(vol.Quadlet.PodmanArgs))
			copy(sorted, vol.Quadlet.PodmanArgs)
			sort.Strings(sorted)
			for _, arg := range sorted {
				builder.WriteString(formatKeyValue("PodmanArgs", arg))
			}
		}
	}

	builder.WriteString("\n[Install]\n")
	builder.WriteString("WantedBy=default.target\n")

	return builder.String()
}

// renderNetwork renders a network spec to a .network unit file.
func (r *Renderer) renderNetwork(net service.Network) string {
	var builder strings.Builder

	builder.WriteString("[Unit]\n")
	builder.WriteString(formatKeyValue("Description", fmt.Sprintf("Network %s", net.Name)))

	builder.WriteString("\n[Network]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	if net.Name != "" {
		builder.WriteString(formatKeyValue("NetworkName", net.Name))
	}

	if net.Driver != "" && net.Driver != "bridge" {
		builder.WriteString(formatKeyValue("Driver", net.Driver))
	}

	if net.IPAM != nil && len(net.IPAM.Config) > 0 {
		config := net.IPAM.Config[0]
		if config.Subnet != "" {
			builder.WriteString(formatKeyValue("Subnet", config.Subnet))
		}
		if config.Gateway != "" {
			builder.WriteString(formatKeyValue("Gateway", config.Gateway))
		}
		if config.IPRange != "" {
			builder.WriteString(formatKeyValue("IPRange", config.IPRange))
		}
	}

	if net.IPv6 {
		builder.WriteString(formatKeyValue("IPv6", "yes"))
	}

	if net.Internal {
		builder.WriteString(formatKeyValue("Internal", "yes"))
	}

	if len(net.Options) > 0 {
		keys := sorting.GetSortedMapKeys(net.Options)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Options", fmt.Sprintf("%s=%s", k, net.Options[k])))
		}
	}

	if len(net.Labels) > 0 {
		keys := sorting.GetSortedMapKeys(net.Labels)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Label", fmt.Sprintf("%s=%s", k, net.Labels[k])))
		}
	}

	// Add Quadlet-specific extensions if present
	if net.Quadlet != nil {
		if net.Quadlet.DisableDNS {
			builder.WriteString(formatKeyValue("DisableDNS", "yes"))
		}

		if len(net.Quadlet.DNS) > 0 {
			sorted := make([]string, len(net.Quadlet.DNS))
			copy(sorted, net.Quadlet.DNS)
			sort.Strings(sorted)
			for _, dns := range sorted {
				builder.WriteString(formatKeyValue("DNS", dns))
			}
		}

		if len(net.Quadlet.Options) > 0 {
			keys := sorting.GetSortedMapKeys(net.Quadlet.Options)
			for _, k := range keys {
				builder.WriteString(formatKeyValue("Options", fmt.Sprintf("%s=%s", k, net.Quadlet.Options[k])))
			}
		}

		if len(net.Quadlet.ContainersConfModule) > 0 {
			sorted := make([]string, len(net.Quadlet.ContainersConfModule))
			copy(sorted, net.Quadlet.ContainersConfModule)
			sort.Strings(sorted)
			for _, module := range sorted {
				builder.WriteString(formatKeyValue("ContainersConfModule", module))
			}
		}

		if len(net.Quadlet.GlobalArgs) > 0 {
			sorted := make([]string, len(net.Quadlet.GlobalArgs))
			copy(sorted, net.Quadlet.GlobalArgs)
			sort.Strings(sorted)
			for _, arg := range sorted {
				builder.WriteString(formatKeyValue("GlobalArgs", arg))
			}
		}

		if len(net.Quadlet.PodmanArgs) > 0 {
			sorted := make([]string, len(net.Quadlet.PodmanArgs))
			copy(sorted, net.Quadlet.PodmanArgs)
			sort.Strings(sorted)
			for _, arg := range sorted {
				builder.WriteString(formatKeyValue("PodmanArgs", arg))
			}
		}
	}

	builder.WriteString("\n[Install]\n")
	builder.WriteString("WantedBy=default.target\n")

	return builder.String()
}

// renderBuild renders a build spec to a .build unit file.
func (r *Renderer) renderBuild(name, description string, build service.Build, dependsOn []string) string {
	var builder strings.Builder

	builder.WriteString("[Unit]\n")
	if description != "" {
		builder.WriteString(formatKeyValue("Description", fmt.Sprintf("Build %s", description)))
	} else {
		builder.WriteString(formatKeyValue("Description", fmt.Sprintf("Build %s", name)))
	}

	if build.Context != "" {
		builder.WriteString(formatKeyValue("WorkingDirectory", build.Context))
	}

	if len(dependsOn) > 0 {
		deps := make([]string, len(dependsOn))
		copy(deps, dependsOn)
		sort.Strings(deps)
		for _, dep := range deps {
			// If dependency already has a unit type suffix, use as-is
			// Otherwise, append .service for service-to-service dependencies
			depUnit := r.formatDependency(dep)
			builder.WriteString(fmt.Sprintf("After=%s\n", depUnit))
			builder.WriteString(fmt.Sprintf("Requires=%s\n", depUnit))
		}
	}

	builder.WriteString("\n[Build]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	if len(build.Tags) > 0 {
		sorted := make([]string, len(build.Tags))
		copy(sorted, build.Tags)
		sort.Strings(sorted)
		for _, tag := range sorted {
			builder.WriteString(formatKeyValue("ImageTag", tag))
		}
	}

	if build.Dockerfile != "" {
		builder.WriteString(formatKeyValue("File", build.Dockerfile))
	}

	if build.SetWorkingDirectory != "" {
		builder.WriteString(formatKeyValue("SetWorkingDirectory", build.SetWorkingDirectory))
	}

	if build.Target != "" {
		builder.WriteString(formatKeyValue("Target", build.Target))
	}

	if build.Pull {
		builder.WriteString(formatKeyValue("Pull", "always"))
	}

	if len(build.Args) > 0 {
		keys := sorting.GetSortedMapKeys(build.Args)
		for _, k := range keys {
			fmt.Fprintf(&builder, "Environment=%s=%s\n", k, build.Args[k])
		}
	}

	if len(build.Labels) > 0 {
		keys := sorting.GetSortedMapKeys(build.Labels)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Label", fmt.Sprintf("%s=%s", k, build.Labels[k])))
		}
	}

	if len(build.Annotations) > 0 {
		sorted := make([]string, len(build.Annotations))
		copy(sorted, build.Annotations)
		sort.Strings(sorted)
		for _, a := range sorted {
			builder.WriteString(formatKeyValue("Annotation", a))
		}
	}

	if len(build.Networks) > 0 {
		sorted := make([]string, len(build.Networks))
		copy(sorted, build.Networks)
		sort.Strings(sorted)
		for _, n := range sorted {
			builder.WriteString(formatKeyValue("Network", n))
		}
	}

	if len(build.Volumes) > 0 {
		sorted := make([]string, len(build.Volumes))
		copy(sorted, build.Volumes)
		sort.Strings(sorted)
		for _, v := range sorted {
			builder.WriteString(formatKeyValue("Volume", v))
		}
	}

	if len(build.Secrets) > 0 {
		sorted := make([]string, len(build.Secrets))
		copy(sorted, build.Secrets)
		sort.Strings(sorted)
		for _, s := range sorted {
			builder.WriteString(formatKeyValue("Secret", s))
		}
	}

	if len(build.CacheFrom) > 0 {
		sorted := make([]string, len(build.CacheFrom))
		copy(sorted, build.CacheFrom)
		sort.Strings(sorted)
		for _, cache := range sorted {
			builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cache-from=%s", cache)))
		}
	}

	if len(build.PodmanArgs) > 0 {
		sorted := make([]string, len(build.PodmanArgs))
		copy(sorted, build.PodmanArgs)
		sort.Strings(sorted)
		for _, arg := range sorted {
			builder.WriteString(formatKeyValue("PodmanArgs", arg))
		}
	}

	builder.WriteString("\n[Install]\n")
	builder.WriteString("WantedBy=default.target\n")

	return builder.String()
}

// formatDependency formats a dependency name for use in unit file directives.
// If the dependency already has a unit type suffix (.network, .volume, etc.),
// it returns as-is. Otherwise, appends .service for service-to-service deps.
func (r *Renderer) formatDependency(dep string) string {
	// Check if dependency already has a known unit type suffix
	for _, suffix := range knownUnitSuffixes {
		if strings.HasSuffix(dep, suffix) {
			return dep
		}
	}

	// No unit type suffix found, default to .service
	return dep + UnitSuffixService
}

// mapRestartPolicy maps service.RestartPolicy to systemd restart value.
func (r *Renderer) mapRestartPolicy(policy service.RestartPolicy) string {
	switch policy {
	case service.RestartPolicyAlways:
		return "always"
	case service.RestartPolicyOnFailure:
		return "on-failure"
	case service.RestartPolicyUnlessStopped:
		return "always"
	case service.RestartPolicyNo:
		return "no"
	default:
		return "no"
	}
}

// formatKeyValue formats a key-value pair for unit files.
func formatKeyValue(key, value string) string {
	return fmt.Sprintf("%s=%s\n", key, value)
}

// formatDuration formats a time.Duration for systemd (e.g., "30s", "1m").
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

// computeHash computes SHA256 hash of content.
func (r *Renderer) computeHash(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// combineHashes combines multiple hashes into a single hash.
func (r *Renderer) combineHashes(hashes []string) string {
	h := sha256.New()
	for _, hash := range hashes {
		h.Write([]byte(hash))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// needsNetworkOnline determines if a container needs network-online.target dependency.
// Returns true if the container uses any networks (including host mode) or has published ports.
func (r *Renderer) needsNetworkOnline(spec service.Spec) bool {
	// Container has published ports - needs network
	if len(spec.Container.Ports) > 0 {
		return true
	}

	// Container uses explicit networks
	if len(spec.Container.Network.ServiceNetworks) > 0 {
		return true
	}

	// Container uses special network modes (host, bridge, etc.)
	if spec.Container.Network.Mode != "" {
		return true
	}

	return false
}

// collectBindMountPaths collects source paths from bind mounts.
// Returns a list of host paths that need RequiresMountsFor directives.
func (r *Renderer) collectBindMountPaths(c service.Container) []string {
	if len(c.Mounts) == 0 {
		return nil
	}

	paths := make([]string, 0)
	for _, mount := range c.Mounts {
		// Only bind mounts need RequiresMountsFor
		if mount.Type == service.MountTypeBind && mount.Source != "" {
			paths = append(paths, mount.Source)
		}
	}

	return paths
}
