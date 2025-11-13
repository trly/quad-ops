package systemd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/podman"
	"github.com/trly/quad-ops/internal/service"
)

// QuadletWriter provides a minimal INI writer for Quadlet unit files.
// It preserves exact formatting, ordering, and behavior of the current renderer.
// Uses manual formatting instead of go-ini to ensure precise control over output.
type QuadletWriter struct {
	sections []quadletSection
}

type quadletSection struct {
	name  string
	lines []string
}

// NewQuadletWriter creates a new Quadlet INI writer.
func NewQuadletWriter() *QuadletWriter {
	return &QuadletWriter{
		sections: make([]quadletSection, 0),
	}
}

// getOrCreateSection finds an existing section or creates a new one.
func (w *QuadletWriter) getOrCreateSection(name string) *quadletSection {
	for i := range w.sections {
		if w.sections[i].name == name {
			return &w.sections[i]
		}
	}
	// Create new section
	w.sections = append(w.sections, quadletSection{
		name:  name,
		lines: make([]string, 0),
	})
	return &w.sections[len(w.sections)-1]
}

// Set adds a single key=value pair to a section.
func (w *QuadletWriter) Set(sectionName, key, value string) {
	if value == "" {
		return
	}
	section := w.getOrCreateSection(sectionName)
	section.lines = append(section.lines, fmt.Sprintf("%s=%s", key, value))
}

// SetBool adds a boolean directive as yes/no.
func (w *QuadletWriter) SetBool(sectionName, key string, value bool) {
	if value {
		w.Set(sectionName, key, "yes")
	}
}

// Append adds a key multiple times with different values (for multi-value directives).
// Values are appended in the order provided.
func (w *QuadletWriter) Append(sectionName, key string, values ...string) {
	if len(values) == 0 {
		return
	}
	section := w.getOrCreateSection(sectionName)
	for _, value := range values {
		section.lines = append(section.lines, fmt.Sprintf("%s=%s", key, value))
	}
}

// AppendSorted adds multiple values for a key in sorted order.
func (w *QuadletWriter) AppendSorted(sectionName, key string, values ...string) {
	if len(values) == 0 {
		return
	}
	sorted := make([]string, len(values))
	copy(sorted, values)
	sort.Strings(sorted)
	w.Append(sectionName, key, sorted...)
}

// AppendMap adds key=value pairs from a map in sorted key order.
func (w *QuadletWriter) AppendMap(sectionName, key string, m map[string]string, formatter func(k, v string) string) {
	if len(m) == 0 {
		return
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	section := w.getOrCreateSection(sectionName)
	for _, k := range keys {
		value := formatter(k, m[k])
		section.lines = append(section.lines, fmt.Sprintf("%s=%s", key, value))
	}
}

// AppendKVMap adds key=k=v pairs from a map in sorted key order (for --env, --label, etc).
func (w *QuadletWriter) AppendKVMap(sectionName, key string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	section := w.getOrCreateSection(sectionName)
	for _, k := range keys {
		v := m[k]
		if v == "" {
			continue
		}
		section.lines = append(section.lines, fmt.Sprintf("%s=%s=%s", key, k, v))
	}
}

// String renders the INI file to a string with proper Quadlet formatting.
func (w *QuadletWriter) String() string {
	var buf bytes.Buffer

	for i, section := range w.sections {
		// Write section header
		buf.WriteString(fmt.Sprintf("[%s]\n", section.name))

		// Write section lines
		for _, line := range section.lines {
			buf.WriteString(line)
			buf.WriteString("\n")
		}

		// Add blank line between sections (but not after the last one)
		if i < len(w.sections)-1 {
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

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

// ============================================================================
// Quadlet Unit Generators
// ============================================================================

// renderContainer renders a container service spec to a .container unit file.
func renderContainer(spec service.Spec) string {
	w := NewQuadletWriter()

	// Step 1: Build [Unit] section with dependencies
	writeUnitSection(w, spec)

	// Step 2: Build [Container] section
	writeContainerSection(w, spec)

	// Step 3: Build [Service] section
	writeServiceSection(w, spec)

	// Step 4: Build [Install] section
	w.Set("Install", "WantedBy", "default.target")

	return w.String()
}

// renderVolume renders a volume spec to a .volume unit file.
func renderVolume(vol service.Volume) string {
	w := NewQuadletWriter()

	// [Unit] section
	w.Set("Unit", "Description", fmt.Sprintf("Volume %s", vol.Name))

	// [Volume] section
	w.Set("Volume", "Label", "managed-by=quad-ops")
	w.Set("Volume", "VolumeName", vol.Name)

	if vol.Driver != "" && vol.Driver != "local" {
		w.Set("Volume", "Driver", vol.Driver)
	}

	w.AppendKVMap("Volume", "Options", vol.Options)

	w.AppendKVMap("Volume", "Label", vol.Labels)

	// Quadlet-specific extensions
	if vol.Quadlet != nil {
		w.AppendSorted("Volume", "ContainersConfModule", vol.Quadlet.ContainersConfModule...)
		w.AppendSorted("Volume", "GlobalArgs", vol.Quadlet.GlobalArgs...)
		w.AppendSorted("Volume", "PodmanArgs", vol.Quadlet.PodmanArgs...)
	}

	// [Install] section
	w.Set("Install", "WantedBy", "default.target")

	return w.String()
}

// renderNetwork renders a network spec to a .network unit file.
func renderNetwork(net service.Network) string {
	w := NewQuadletWriter()

	// [Unit] section
	if net.External {
		w.Set("Unit", "Description", fmt.Sprintf("External Network %s", net.Name))
	} else {
		w.Set("Unit", "Description", fmt.Sprintf("Network %s", net.Name))
	}

	// [Network] section
	w.Set("Network", "Label", "managed-by=quad-ops")
	w.Set("Network", "NetworkName", net.Name)

	// Early return for external networks
	if net.External {
		w.Set("Install", "WantedBy", "default.target")
		return w.String()
	}

	// Non-external network configuration
	if net.Driver != "" && net.Driver != "bridge" {
		w.Set("Network", "Driver", net.Driver)
	}

	// IPAM configuration
	if net.IPAM != nil && len(net.IPAM.Config) > 0 {
		config := net.IPAM.Config[0]
		w.Set("Network", "Subnet", config.Subnet)
		w.Set("Network", "Gateway", config.Gateway)
		w.Set("Network", "IPRange", config.IPRange)
	}

	w.SetBool("Network", "IPv6", net.IPv6)
	w.SetBool("Network", "Internal", net.Internal)

	w.AppendKVMap("Network", "Options", net.Options)

	w.AppendKVMap("Network", "Label", net.Labels)

	// Quadlet-specific extensions
	if net.Quadlet != nil {
		w.SetBool("Network", "DisableDNS", net.Quadlet.DisableDNS)
		w.AppendSorted("Network", "DNS", net.Quadlet.DNS...)
		w.AppendKVMap("Network", "Options", net.Quadlet.Options)
		w.AppendSorted("Network", "ContainersConfModule", net.Quadlet.ContainersConfModule...)
		w.AppendSorted("Network", "GlobalArgs", net.Quadlet.GlobalArgs...)
		w.AppendSorted("Network", "PodmanArgs", net.Quadlet.PodmanArgs...)
	}

	// [Install] section
	w.Set("Install", "WantedBy", "default.target")

	return w.String()
}

// renderBuild renders a build spec to a .build unit file.
func renderBuild(name, description string, build service.Build, dependsOn []string) string {
	w := NewQuadletWriter()

	// [Unit] section
	desc := fmt.Sprintf("Build %s", name)
	if description != "" {
		desc = fmt.Sprintf("Build %s", description)
	}
	w.Set("Unit", "Description", desc)
	w.Set("Unit", "WorkingDirectory", build.Context)

	// Add dependencies
	if len(dependsOn) > 0 {
		deps := make([]string, len(dependsOn))
		copy(deps, dependsOn)
		sort.Strings(deps)
		for _, dep := range deps {
			depUnit := formatDependency(dep)
			w.Append("Unit", "After", depUnit)
			w.Append("Unit", "Requires", depUnit)
		}
	}

	// [Build] section
	w.Set("Build", "Label", "managed-by=quad-ops")
	w.AppendSorted("Build", "ImageTag", build.Tags...)
	w.Set("Build", "File", build.Dockerfile)
	w.Set("Build", "SetWorkingDirectory", build.SetWorkingDirectory)
	w.Set("Build", "Target", build.Target)

	if build.Pull {
		w.Set("Build", "Pull", "always")
	}

	w.AppendKVMap("Build", "Environment", build.Args)

	w.AppendKVMap("Build", "Label", build.Labels)

	w.AppendSorted("Build", "Annotation", build.Annotations...)
	w.AppendSorted("Build", "Network", build.Networks...)
	w.AppendSorted("Build", "Volume", build.Volumes...)
	w.AppendSorted("Build", "Secret", build.Secrets...)

	// CacheFrom needs special formatting
	if len(build.CacheFrom) > 0 {
		sorted := make([]string, len(build.CacheFrom))
		copy(sorted, build.CacheFrom)
		sort.Strings(sorted)
		for _, cache := range sorted {
			w.Append("Build", "PodmanArgs", fmt.Sprintf("--cache-from=%s", cache))
		}
	}

	w.AppendSorted("Build", "PodmanArgs", build.PodmanArgs...)

	// [Install] section
	w.Set("Install", "WantedBy", "default.target")

	return w.String()
}

// ============================================================================
// Section Writers
// ============================================================================

// writeUnitSection builds the [Unit] section with all dependencies.
func writeUnitSection(w *QuadletWriter, spec service.Spec) {
	if spec.Description != "" {
		w.Set("Unit", "Description", spec.Description)
	}

	// Add network-online.target dependency if container uses networks or has published ports
	if len(spec.Container.Ports) > 0 || len(spec.Container.Network.ServiceNetworks) > 0 || spec.Container.Network.Mode != "" {
		w.Append("Unit", "After", "network-online.target")
		w.Append("Unit", "Wants", "network-online.target")
	}

	// Add RequiresMountsFor directives for bind mounts
	bindMountPaths := make([]string, 0)
	for _, mount := range spec.Container.Mounts {
		if mount.Type == service.MountTypeBind && mount.Source != "" {
			bindMountPaths = append(bindMountPaths, mount.Source)
		}
	}
	if len(bindMountPaths) > 0 {
		w.AppendSorted("Unit", "RequiresMountsFor", bindMountPaths...)
	}

	// DependsOn services - ONLY service-to-service dependencies
	// Quadlet automatically handles Volume=, Network=, and Image= dependencies
	if len(spec.DependsOn) > 0 {
		deps := make([]string, len(spec.DependsOn))
		copy(deps, spec.DependsOn)
		sort.Strings(deps)
		for _, dep := range deps {
			depUnit := formatDependency(dep)
			w.Append("Unit", "After", depUnit)
			w.Append("Unit", "Requires", depUnit)
		}
	}

	// External dependencies (cross-project) - Always add After=
	// Required deps: After= + Requires=
	// Optional deps: After= only (NOT Wants= - we don't want to auto-start them)
	if len(spec.ExternalDependencies) > 0 {
		// Sort for deterministic output
		externalDeps := make([]service.ExternalDependency, len(spec.ExternalDependencies))
		copy(externalDeps, spec.ExternalDependencies)
		sort.Slice(externalDeps, func(i, j int) bool {
			// Sort by project_service for consistent ordering (using Prefix logic)
			nameI := compose.Prefix(externalDeps[i].Project, externalDeps[i].Service)
			nameJ := compose.Prefix(externalDeps[j].Project, externalDeps[j].Service)
			return nameI < nameJ
		})

		for _, dep := range externalDeps {
			// Format: project_service.service (using Prefix for consistency with intra-project deps)
			unitName := compose.Prefix(dep.Project, dep.Service) + ".service"
			w.Append("Unit", "After", unitName)

			// Only required dependencies get Requires=
			if !dep.Optional {
				w.Append("Unit", "Requires", unitName)
			}
			// Note: We do NOT use Wants= for optional deps - we don't want to auto-start them
		}
	}
}

// writeContainerSection builds the [Container] section.
func writeContainerSection(w *QuadletWriter, spec service.Spec) {
	c := spec.Container

	w.Set("Container", "Label", "managed-by=quad-ops")

	// If this service has a build, use Quadlet .build reference
	// Quadlet automatically creates dependency on the .build unit
	if c.Build != nil {
		w.Set("Container", "Image", spec.Name+".build")
	} else {
		w.Set("Container", "Image", c.Image)
	}

	w.Set("Container", "ContainerName", c.ContainerName)
	w.Set("Container", "HostName", c.Hostname)

	writeEnvironment(w, c)
	writePorts(w, c)
	writeMounts(w, c)
	writeNetworks(w, c)
	writeDNS(w, c)
	writeDevices(w, c)
	writeExecution(w, c)
	writeHealthcheck(w, c)
	writeResources(w, c)
	writeSecurity(w, c)
	writeLogging(w, c)
	writeSecrets(w, c)
	writeAdvanced(w, spec, c)

	w.AppendKVMap("Container", "Label", c.Labels)
	w.AppendKVMap("Container", "Annotation", spec.Annotations)
}

// writeServiceSection builds the [Service] section.
func writeServiceSection(w *QuadletWriter, spec service.Spec) {
	// Configure init containers as oneshot services
	if strings.Contains(spec.Name, "-init-") {
		w.Set("Service", "Type", "oneshot")
		w.Set("Service", "RemainAfterExit", "yes")
	}

	restart := "no"
	switch spec.Container.RestartPolicy {
	case service.RestartPolicyAlways:
		restart = "always"
	case service.RestartPolicyOnFailure:
		restart = "on-failure"
	case service.RestartPolicyUnlessStopped:
		restart = "always"
	case service.RestartPolicyNo:
		restart = "no"
	}
	w.Set("Service", "Restart", restart)

	// Set timeout for image pull (default 15 minutes = 900 seconds)
	w.Set("Service", "TimeoutStartSec", "900")

	// Add stop timeout if configured (inline addStopTimeout)
	if spec.Container.StopGracePeriod > 0 {
		seconds := int(spec.Container.StopGracePeriod.Seconds())
		w.Set("Service", "StopTimeoutSec", fmt.Sprintf("%d", seconds))
	}
}

// writeEnvironment writes environment variables and files to container section.
func writeEnvironment(w *QuadletWriter, c service.Container) {
	w.AppendKVMap("Container", "Environment", c.Env)
	if len(c.EnvFiles) > 0 {
		w.AppendSorted("Container", "EnvironmentFile", c.EnvFiles...)
	}
}

// writePorts writes port mappings to container section.
func writePorts(w *QuadletWriter, c service.Container) {
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
	w.AppendSorted("Container", "PublishPort", ports...)
}

// writeMounts writes volume and tmpfs mounts to container section.
func writeMounts(w *QuadletWriter, c service.Container) {
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
		// Named volumes use Quadlet .volume reference syntax
		// Quadlet automatically creates dependency on the .volume unit
		if m.Type == service.MountTypeVolume {
			source = source + UnitSuffixVolume
		}
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

	w.AppendSorted("Container", "Volume", mounts...)
	w.AppendSorted("Container", "Tmpfs", tmpfsMounts...)

	if len(c.Tmpfs) > 0 {
		w.AppendSorted("Container", "Tmpfs", c.Tmpfs...)
	}
}

// writeNetworks writes network configuration to container section.
func writeNetworks(w *QuadletWriter, c service.Container) {
	if c.Network.Mode != "" && c.Network.Mode != "bridge" {
		w.Set("Container", "Network", c.Network.Mode)
	}
	if len(c.Network.Aliases) > 0 {
		w.AppendSorted("Container", "NetworkAlias", c.Network.Aliases...)
	}
	if len(c.Network.ServiceNetworks) > 0 {
		networks := make([]string, len(c.Network.ServiceNetworks))
		copy(networks, c.Network.ServiceNetworks)
		sort.Strings(networks)
		for _, net := range networks {
			w.Append("Container", "Network", net+UnitSuffixNetwork)
		}
	}
}

// writeDNS writes DNS configuration to container section.
func writeDNS(w *QuadletWriter, c service.Container) {
	if len(c.DNS) > 0 {
		w.AppendSorted("Container", "DNS", c.DNS...)
	}
	if len(c.DNSSearch) > 0 {
		w.AppendSorted("Container", "DNSSearch", c.DNSSearch...)
	}
	if len(c.DNSOptions) > 0 {
		w.AppendSorted("Container", "DNSOption", c.DNSOptions...)
	}
}

// writeDevices writes device mappings and cgroup rules to container section.
func writeDevices(w *QuadletWriter, c service.Container) {
	if len(c.Devices) > 0 {
		w.AppendSorted("Container", "AddDevice", c.Devices...)
	}
	if len(c.DeviceCgroupRules) > 0 {
		rules := make([]string, len(c.DeviceCgroupRules))
		for i, rule := range c.DeviceCgroupRules {
			rules[i] = fmt.Sprintf("--device-cgroup-rule=%s", rule)
		}
		w.Append("Container", "PodmanArgs", rules...)
	}
}

// writeExecution writes execution configuration to container section.
func writeExecution(w *QuadletWriter, c service.Container) {
	if len(c.Entrypoint) > 0 {
		w.Set("Container", "Entrypoint", strings.Join(c.Entrypoint, " "))
	}
	if len(c.Command) > 0 {
		w.Set("Container", "Exec", strings.Join(c.Command, " "))
	}
	w.Set("Container", "User", c.User)
	w.Set("Container", "Group", c.Group)
	w.Set("Container", "WorkingDir", c.WorkingDir)
	w.SetBool("Container", "RunInit", c.Init)
	w.SetBool("Container", "ReadOnly", c.ReadOnly)
}

// writeHealthcheck writes healthcheck configuration to container section.
func writeHealthcheck(w *QuadletWriter, c service.Container) {
	if c.Healthcheck == nil {
		return
	}
	hc := c.Healthcheck
	if len(hc.Test) > 0 {
		w.Set("Container", "HealthCmd", strings.Join(hc.Test, " "))
	}
	if hc.Interval > 0 {
		w.Set("Container", "HealthInterval", formatDuration(hc.Interval))
	}
	if hc.Timeout > 0 {
		w.Set("Container", "HealthTimeout", formatDuration(hc.Timeout))
	}
	if hc.Retries > 0 {
		w.Set("Container", "HealthRetries", fmt.Sprintf("%d", hc.Retries))
	}
	if hc.StartPeriod > 0 {
		w.Set("Container", "HealthStartPeriod", formatDuration(hc.StartPeriod))
	}
	if hc.StartInterval > 0 {
		w.Set("Container", "HealthStartupInterval", formatDuration(hc.StartInterval))
	}
}

// writeResources writes resource constraints to container section.
func writeResources(w *QuadletWriter, c service.Container) {
	w.Set("Container", "Memory", c.Resources.Memory)
	w.Set("Container", "ShmSize", c.Resources.ShmSize)
	if c.Resources.PidsLimit > 0 {
		w.Set("Container", "PidsLimit", fmt.Sprintf("%d", c.Resources.PidsLimit))
	} else if c.PidsLimit > 0 {
		w.Set("Container", "PidsLimit", fmt.Sprintf("%d", c.PidsLimit))
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
		w.AppendSorted("Container", "Ulimit", ulimits...)
	}
	w.AppendMap("Container", "Sysctl", c.Sysctls, func(k, v string) string {
		return fmt.Sprintf("%s=%s", k, v)
	})
}

// writeSecurity writes security configuration to container section.
func writeSecurity(w *QuadletWriter, c service.Container) {
	w.SetBool("Container", "SecurityLabelDisable", c.Security.Privileged)
	w.AppendSorted("Container", "AddCapability", c.Security.CapAdd...)
	w.AppendSorted("Container", "DropCapability", c.Security.CapDrop...)
	for _, opt := range c.Security.SecurityOpt {
		if strings.HasPrefix(opt, "label=") {
			labelValue := strings.TrimPrefix(opt, "label=")
			w.Set("Container", "SecurityLabelType", labelValue)
		} else {
			w.Set("Container", "SecurityLabelLevel", opt)
		}
	}
	w.SetBool("Container", "ReadOnlyTmpfs", c.Security.ReadonlyRootfs)
	if len(c.Security.GroupAdd) > 0 {
		groups := make([]string, len(c.Security.GroupAdd))
		copy(groups, c.Security.GroupAdd)
		sort.Strings(groups)
		for _, group := range groups {
			w.Append("Container", "PodmanArgs", fmt.Sprintf("--group-add=%s", group))
		}
	}
	w.Set("Container", "UserNS", c.UserNS)
}

// writeLogging writes logging configuration to container section.
func writeLogging(w *QuadletWriter, c service.Container) {
	w.Set("Container", "LogDriver", c.Logging.Driver)
	w.AppendMap("Container", "PodmanArgs", c.Logging.Options, func(k, v string) string {
		return fmt.Sprintf("--log-opt=%s=%s", k, v)
	})
}

// writeSecrets writes secrets configuration to container section.
func writeSecrets(w *QuadletWriter, c service.Container) {
	if len(c.Secrets) == 0 && len(c.EnvSecrets) == 0 {
		return
	}
	secretValues := make([]string, 0, len(c.Secrets)+len(c.EnvSecrets))

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
		secretValues = append(secretValues, secretStr)
	}

	envSecretKeys := make([]string, 0, len(c.EnvSecrets))
	for k := range c.EnvSecrets {
		envSecretKeys = append(envSecretKeys, k)
	}
	sort.Strings(envSecretKeys)
	for _, secretName := range envSecretKeys {
		envVarName := c.EnvSecrets[secretName]
		secretStr := fmt.Sprintf("%s,type=env,target=%s", secretName, envVarName)
		secretValues = append(secretValues, secretStr)
	}

	w.AppendSorted("Container", "Secret", secretValues...)
}

// writeAdvanced writes advanced configuration (PID, IPC, cgroup modes, extra hosts, stop signal) to container section.
func writeAdvanced(w *QuadletWriter, spec service.Spec, c service.Container) {
	w.AppendSorted("Container", "AddHost", c.ExtraHosts...)

	if c.StopSignal != "" {
		signal := strings.TrimPrefix(c.StopSignal, "SIG")
		w.Set("Container", "StopSignal", signal)
	}

	if c.PidMode != "" {
		w.Set("Container", "PodmanArgs", fmt.Sprintf("--pid=%s", c.PidMode))
	}
	if c.IpcMode != "" {
		w.Set("Container", "PodmanArgs", fmt.Sprintf("--ipc=%s", c.IpcMode))
	}
	if c.CgroupMode != "" {
		w.Set("Container", "PodmanArgs", fmt.Sprintf("--cgroupns=%s", c.CgroupMode))
	}
	// Use shared podman args builder for Quadlet-unsupported features
	// (memory-reservation, memory-swap, cpu-shares, cpu-quota, cpu-period, custom PodmanArgs)
	quadletArgs := podman.BuildQuadletPodmanArgs(spec)
	w.AppendSorted("Container", "PodmanArgs", quadletArgs...)
}

// ============================================================================
// Helper Utilities
// ============================================================================

// formatDependency formats a dependency name for use in unit file directives.
// If the dependency already has a unit type suffix (.network, .volume, etc.),
// it returns as-is. Otherwise, appends .service for service-to-service deps.
func formatDependency(dep string) string {
	// Check if dependency already has a known unit type suffix
	for _, suffix := range knownUnitSuffixes {
		if strings.HasSuffix(dep, suffix) {
			return dep
		}
	}

	// No unit type suffix found, default to .service
	return dep + UnitSuffixService
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
