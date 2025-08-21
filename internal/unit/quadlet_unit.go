package unit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/sorting"
)

// QuadletUnit represents the configuration for a Quadlet unit, which can include
// systemd, container, volume, network, pod, Kubernetes, image, and build settings.
type QuadletUnit struct {
	Name      string        `yaml:"name"`
	Type      string        `yaml:"type"`
	Systemd   SystemdConfig `yaml:"systemd"`
	Container Container     `yaml:"container,omitempty"`
	Volume    Volume        `yaml:"volume,omitempty"`
	Network   Network       `yaml:"network,omitempty"`
	Build     Build         `yaml:"build,omitempty"`
}

// SystemdConfig represents the configuration for a systemd unit.
// It includes settings such as the unit description, dependencies,
// restart policy, and other systemd-specific options.
type SystemdConfig struct {
	Description        string   `yaml:"description"`
	After              []string `yaml:"after"`
	Before             []string `yaml:"before"`
	Requires           []string `yaml:"requires"`
	Wants              []string `yaml:"wants"`
	Conflicts          []string `yaml:"conflicts"`
	PartOf             []string `yaml:"part_of"`              // Services that this unit is part of
	PropagatesReloadTo []string `yaml:"propagates_reload_to"` // Services that should be reloaded when this unit is reloaded
	RestartPolicy      string   `yaml:"restart_policy"`
	TimeoutStartSec    int      `yaml:"timeout_start_sec"`
	Type               string   `yaml:"type"`
	RemainAfterExit    bool     `yaml:"remain_after_exit"`
	WantedBy           []string `yaml:"wanted_by"`
}

// addBasicConfig adds basic container configuration like image and labels.
func (u *QuadletUnit) addBasicConfig(builder *strings.Builder) {
	if u.Container.Image != "" {
		builder.WriteString(formatKeyValue("Image", u.Container.Image))
	}
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	// Use centralized sorting function for consistent output
	sorting.SortAndIterateSlice(u.Container.Label, func(label string) {
		builder.WriteString(formatKeyValue("Label", label))
	})

	// Use centralized sorting function for ports
	sorting.SortAndIterateSlice(u.Container.PublishPort, func(port string) {
		builder.WriteString(formatKeyValue("PublishPort", port))
	})
}

// addEnvironmentConfig adds environment variables and environment files.
func (u *QuadletUnit) addEnvironmentConfig(builder *strings.Builder) {
	// Sort environment variables for consistent output
	envKeys := sorting.GetSortedMapKeys(u.Container.Environment)

	// Add environment variables in sorted order
	for _, k := range envKeys {
		fmt.Fprintf(builder, "Environment=%s=%s\n", k, u.Container.Environment[k])
	}
	// Use centralized sorting function for environment files
	sorting.SortAndIterateSlice(u.Container.EnvironmentFile, func(envFile string) {
		builder.WriteString(formatKeyValue("EnvironmentFile", envFile))
	})
}

// addVolumeNetworkConfig adds volume and network configuration.
func (u *QuadletUnit) addVolumeNetworkConfig(builder *strings.Builder) {
	// Use centralized sorting function for volumes
	sorting.SortAndIterateSlice(u.Container.Volume, func(vol string) {
		builder.WriteString(formatKeyValue("Volume", vol))
	})

	// Use centralized sorting function for networks
	sorting.SortAndIterateSlice(u.Container.Network, func(net string) {
		builder.WriteString(formatKeyValue("Network", net))
	})

	// Use centralized sorting function for network aliases
	sorting.SortAndIterateSlice(u.Container.NetworkAlias, func(alias string) {
		builder.WriteString(formatKeyValue("NetworkAlias", alias))
	})
}

// addExecutionConfig adds execution configuration like entrypoint, user, working directory.
func (u *QuadletUnit) addExecutionConfig(builder *strings.Builder) {
	if len(u.Container.Exec) > 0 {
		// Don't sort Exec commands as order matters
		builder.WriteString("Exec=" + strings.Join(u.Container.Exec, " ") + "\n")
	}
	if len(u.Container.Entrypoint) > 0 {
		// Don't sort Entrypoint commands as order matters
		builder.WriteString("Entrypoint=" + strings.Join(u.Container.Entrypoint, " ") + "\n")
	}
	if u.Container.User != "" {
		builder.WriteString(formatKeyValue("User", u.Container.User))
	}
	if u.Container.Group != "" {
		builder.WriteString(formatKeyValue("Group", u.Container.Group))
	}
	if u.Container.WorkingDir != "" {
		builder.WriteString(formatKeyValue("WorkingDir", u.Container.WorkingDir))
	}
	if *u.Container.RunInit {
		builder.WriteString(formatKeyValue("RunInit", "yes"))
	}
	// Privileged is not supported by podman-systemd
	if u.Container.ReadOnly {
		builder.WriteString(formatKeyValue("ReadOnly", "yes"))
	}
	// SecurityLabel is not supported by podman-systemd
	// Use specific labels like SecurityLabelType instead
	if u.Container.HostName != "" {
		builder.WriteString(formatKeyValue("HostName", u.Container.HostName))
	}
	// Set ContainerName to override systemd- prefix if useSystemdDNS is false
	if u.Container.ContainerName != "" {
		builder.WriteString(formatKeyValue("ContainerName", u.Container.ContainerName))
	}
}

// addHealthCheckConfig adds health check configuration.
func (u *QuadletUnit) addHealthCheckConfig(builder *strings.Builder) {
	// Special handling for health check commands with environment variables
	if len(u.Container.HealthCmd) > 0 {
		// For health checks, we need special handling to ensure environment variables
		// are preserved and properly escaped for systemd
		if len(u.Container.HealthCmd) == 2 && (u.Container.HealthCmd[0] == "CMD" || u.Container.HealthCmd[0] == "CMD-SHELL") {
			// For CMD-SHELL or CMD with specific command string, format specially to preserve env vars
			fmt.Fprintf(builder, "HealthCmd=%s %s\n", u.Container.HealthCmd[0], u.Container.HealthCmd[1])
		} else {
			// For other cases, use standard slice formatting
			builder.WriteString(formatKeyValueSlice("HealthCmd", u.Container.HealthCmd))
		}
	}
	if u.Container.HealthInterval != "" {
		builder.WriteString(formatKeyValue("HealthInterval", u.Container.HealthInterval))
	}
	if u.Container.HealthTimeout != "" {
		builder.WriteString(formatKeyValue("HealthTimeout", u.Container.HealthTimeout))
	}
	if u.Container.HealthRetries != 0 {
		fmt.Fprintf(builder, "HealthRetries=%d\n", u.Container.HealthRetries)
	}
	if u.Container.HealthStartPeriod != "" {
		builder.WriteString(formatKeyValue("HealthStartPeriod", u.Container.HealthStartPeriod))
	}
	if u.Container.HealthStartInterval != "" {
		builder.WriteString(formatKeyValue("HealthStartupInterval", u.Container.HealthStartInterval))
	}
}

// addResourceConstraints adds resource constraints like memory and CPU limits.
func (u *QuadletUnit) addResourceConstraints(builder *strings.Builder) {
	// Memory directives are not supported by Podman Quadlet
	// We keep these fields for internal calculations but don't include them in the output
	// if u.Container.Memory != "" {
	// 	builder.WriteString(formatKeyValue("Memory", u.Container.Memory))
	// }
	// if u.Container.MemoryReservation != "" {
	// 	builder.WriteString(formatKeyValue("MemoryReservation", u.Container.MemoryReservation))
	// }
	// if u.Container.MemorySwap != "" {
	// 	builder.WriteString(formatKeyValue("MemorySwap", u.Container.MemorySwap))
	// }
	// CPU directives are not supported by Podman Quadlet
	// We keep these fields for internal calculations but don't include them in the output
	// if u.Container.CPUShares != 0 {
	// 	builder.WriteString(formatKeyValue("CPUShares", fmt.Sprintf("%d", u.Container.CPUShares)))
	// }
	// if u.Container.CPUQuota != 0 {
	// 	builder.WriteString(formatKeyValue("CPUQuota", fmt.Sprintf("%d", u.Container.CPUQuota)))
	// }
	// CPUPeriod is no longer used directly as it's not supported in Podman Quadlet
	if u.Container.PidsLimit != 0 {
		fmt.Fprintf(builder, "PidsLimit=%d\n", u.Container.PidsLimit)
	}
}

// addAdvancedConfig adds advanced configuration like ulimit, tmpfs, sysctl.
func (u *QuadletUnit) addAdvancedConfig(builder *strings.Builder) {
	sorting.SortAndIterateSlice(u.Container.Ulimit, func(ulimit string) {
		builder.WriteString(formatKeyValue("Ulimit", ulimit))
	})

	sorting.SortAndIterateSlice(u.Container.Tmpfs, func(tmpfs string) {
		builder.WriteString(formatKeyValue("Tmpfs", tmpfs))
	})

	// Use sortedSysctlKeys if available, otherwise generate sorted keys on the fly
	var sysctlKeys []string
	if len(u.Container.sortedSysctlKeys) > 0 {
		sysctlKeys = u.Container.sortedSysctlKeys
	} else {
		// Sort sysctl variables for consistent output
		sysctlKeys = sorting.GetSortedMapKeys(u.Container.Sysctl)
	}

	// Add sysctl variables in sorted order
	for _, k := range sysctlKeys {
		fmt.Fprintf(builder, "Sysctl=%s=%s\n", k, u.Container.Sysctl[k])
	}

	if u.Container.UserNS != "" {
		builder.WriteString(formatKeyValue("UserNS", u.Container.UserNS))
	}

	// Add PodmanArgs for features not directly supported by Quadlet
	sorting.SortAndIterateSlice(u.Container.PodmanArgs, func(arg string) {
		builder.WriteString(formatKeyValue("PodmanArgs", arg))
	})
}

// addLoggingConfig adds logging configuration.
func (u *QuadletUnit) addLoggingConfig(builder *strings.Builder) {
	if u.Container.LogDriver != "" {
		builder.WriteString(formatKeyValue("LogDriver", u.Container.LogDriver))
	}

	// Add LogOpt options in sorted order
	var logOptKeys []string
	if len(u.Container.sortedLogOptKeys) > 0 {
		logOptKeys = u.Container.sortedLogOptKeys
	} else {
		// Sort log options for consistent output
		logOptKeys = sorting.GetSortedMapKeys(u.Container.LogOpt)
	}

	// Add log options in sorted order
	for _, k := range logOptKeys {
		fmt.Fprintf(builder, "LogOpt=%s=%s\n", k, u.Container.LogOpt[k])
	}
}

// addSecretsConfig adds secrets configuration.
func (u *QuadletUnit) addSecretsConfig(builder *strings.Builder) {
	for _, secret := range u.Container.Secrets {
		builder.WriteString(formatKeyValue("Secret", formatSecret(secret)))
	}
}

func (u *QuadletUnit) generateContainerSection() string {
	var builder strings.Builder
	builder.WriteString("\n[Container]\n")

	// Add configuration in logical groups
	u.addBasicConfig(&builder)
	u.addEnvironmentConfig(&builder)
	u.addVolumeNetworkConfig(&builder)
	u.addExecutionConfig(&builder)
	u.addHealthCheckConfig(&builder)
	u.addResourceConstraints(&builder)
	u.addAdvancedConfig(&builder)
	u.addLoggingConfig(&builder)
	u.addSecretsConfig(&builder)

	return builder.String()
}

func (u *QuadletUnit) generateVolumeSection() string {
	var builder strings.Builder
	builder.WriteString("\n[Volume]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	// Use centralized sorting function for volume labels
	sorting.SortAndIterateSlice(u.Volume.Label, func(label string) {
		builder.WriteString(formatKeyValue("Label", label))
	})

	// Set VolumeName to override systemd- prefix if configured
	if u.Volume.VolumeName != "" {
		builder.WriteString(formatKeyValue("VolumeName", u.Volume.VolumeName))
	}

	if u.Volume.Device != "" {
		builder.WriteString(formatKeyValue("Device", u.Volume.Device))
	}

	// Use centralized sorting function for volume options
	sorting.SortAndIterateSlice(u.Volume.Options, func(opt string) {
		builder.WriteString(formatKeyValue("Options", opt))
	})
	if u.Volume.Copy {
		builder.WriteString(formatKeyValue("Copy", "yes"))
	}
	if u.Volume.Group != "" {
		builder.WriteString(formatKeyValue("Group", u.Volume.Group))
	}
	if u.Volume.Type != "" {
		builder.WriteString(formatKeyValue("Type", u.Volume.Type))
	}
	return builder.String()
}

func (u *QuadletUnit) generateNetworkSection() string {
	var builder strings.Builder
	builder.WriteString("\n[Network]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	// Use centralized sorting function for network labels
	sorting.SortAndIterateSlice(u.Network.Label, func(label string) {
		builder.WriteString(formatKeyValue("Label", label))
	})

	// Set NetworkName to override systemd- prefix if configured
	if u.Network.NetworkName != "" {
		builder.WriteString(formatKeyValue("NetworkName", u.Network.NetworkName))
	}

	if u.Network.Driver != "" {
		builder.WriteString(formatKeyValue("Driver", u.Network.Driver))
	}
	if u.Network.Gateway != "" {
		builder.WriteString(formatKeyValue("Gateway", u.Network.Gateway))
	}
	if u.Network.IPRange != "" {
		builder.WriteString(formatKeyValue("IPRange", u.Network.IPRange))
	}
	if u.Network.Subnet != "" {
		builder.WriteString(formatKeyValue("Subnet", u.Network.Subnet))
	}
	if u.Network.IPv6 {
		builder.WriteString(formatKeyValue("IPv6", "yes"))
	}
	if u.Network.Internal {
		builder.WriteString(formatKeyValue("Internal", "yes"))
	}
	// DNSEnabled is not supported by podman-systemd

	// Use centralized sorting function for network options
	sorting.SortAndIterateSlice(u.Network.Options, func(opt string) {
		builder.WriteString(formatKeyValue("Options", opt))
	})
	return builder.String()
}

// generateBuildSection generates the [Build] section for a quadlet build unit.
func (u *QuadletUnit) generateBuildSection() string {
	var builder strings.Builder
	builder.WriteString("\n[Build]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	u.addBuildBasicConfig(&builder)
	u.addBuildMetadata(&builder)
	u.addBuildEnvironment(&builder)
	u.addBuildResources(&builder)
	u.addBuildOptions(&builder)

	return builder.String()
}

func (u *QuadletUnit) addBuildBasicConfig(builder *strings.Builder) {
	if len(u.Build.ImageTag) > 0 {
		for _, tag := range u.Build.ImageTag {
			builder.WriteString(formatKeyValue("ImageTag", tag))
		}
	}

	if u.Build.File != "" {
		builder.WriteString(formatKeyValue("File", u.Build.File))
	}

	if u.Build.SetWorkingDirectory != "" {
		builder.WriteString(formatKeyValue("SetWorkingDirectory", u.Build.SetWorkingDirectory))
	}
}

func (u *QuadletUnit) addBuildMetadata(builder *strings.Builder) {
	sorting.SortAndIterateSlice(u.Build.Label, func(label string) {
		builder.WriteString(formatKeyValue("Label", label))
	})

	sorting.SortAndIterateSlice(u.Build.Annotation, func(annotation string) {
		builder.WriteString(formatKeyValue("Annotation", annotation))
	})
}

func (u *QuadletUnit) addBuildEnvironment(builder *strings.Builder) {
	if len(u.Build.Env) > 0 {
		keys := make([]string, 0, len(u.Build.Env))
		for k := range u.Build.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(builder, "Environment=%s=%s\n", k, u.Build.Env[k])
		}
	}
}

func (u *QuadletUnit) addBuildResources(builder *strings.Builder) {
	sorting.SortAndIterateSlice(u.Build.Network, func(network string) {
		builder.WriteString(formatKeyValue("Network", network))
	})

	sorting.SortAndIterateSlice(u.Build.Volume, func(volume string) {
		builder.WriteString(formatKeyValue("Volume", volume))
	})

	sorting.SortAndIterateSlice(u.Build.Secret, func(secret string) {
		builder.WriteString(formatKeyValue("Secret", secret))
	})
}

func (u *QuadletUnit) addBuildOptions(builder *strings.Builder) {
	if u.Build.Target != "" {
		builder.WriteString(formatKeyValue("Target", u.Build.Target))
	}

	if u.Build.Pull != "" {
		builder.WriteString(formatKeyValue("Pull", u.Build.Pull))
	}

	sorting.SortAndIterateSlice(u.Build.PodmanArgs, func(arg string) {
		builder.WriteString(formatKeyValue("PodmanArgs", arg))
	})
}

func (u *QuadletUnit) generateUnitSection() string {
	var builder strings.Builder
	builder.WriteString("[Unit]\n")
	if u.Systemd.Description != "" {
		builder.WriteString(formatKeyValue("Description", u.Systemd.Description))
	}

	// Sort all systemd directives for consistent output
	if len(u.Systemd.After) > 0 {
		builder.WriteString(formatKeyValueSlice("After", u.Systemd.After))
	}

	if len(u.Systemd.Before) > 0 {
		builder.WriteString(formatKeyValueSlice("Before", u.Systemd.Before))
	}

	if len(u.Systemd.Requires) > 0 {
		builder.WriteString(formatKeyValueSlice("Requires", u.Systemd.Requires))
	}

	if len(u.Systemd.Wants) > 0 {
		builder.WriteString(formatKeyValueSlice("Wants", u.Systemd.Wants))
	}

	if len(u.Systemd.Conflicts) > 0 {
		builder.WriteString(formatKeyValueSlice("Conflicts", u.Systemd.Conflicts))
	}

	if len(u.Systemd.PartOf) > 0 {
		builder.WriteString(formatKeyValueSlice("PartOf", u.Systemd.PartOf))
	}

	if len(u.Systemd.PropagatesReloadTo) > 0 {
		builder.WriteString(formatKeyValueSlice("PropagatesReloadTo", u.Systemd.PropagatesReloadTo))
	}
	return builder.String()
}

func (u *QuadletUnit) generateServiceSection() string {
	var builder strings.Builder
	builder.WriteString("\n[Service]\n")
	if u.Systemd.Type != "" {
		builder.WriteString(formatKeyValue("Type", u.Systemd.Type))
	}
	if u.Systemd.RestartPolicy != "" {
		builder.WriteString(formatKeyValue("Restart", u.Systemd.RestartPolicy))
	}
	if u.Systemd.TimeoutStartSec != 0 {
		fmt.Fprintf(&builder, "TimeoutStartSec=%d\n", u.Systemd.TimeoutStartSec)
	}
	if u.Systemd.RemainAfterExit {
		builder.WriteString(formatKeyValue("RemainAfterExit", "yes"))
	}
	return builder.String()
}

// GenerateQuadletUnit generates a quadlet unit file content from a unit configuration.
func GenerateQuadletUnit(unit QuadletUnit, logger log.Logger) string {
	logger.Debug("Generating Quadlet unit", "name", unit.Name, "type", unit.Type)

	content := unit.generateUnitSection()

	switch unit.Type {
	case "container":
		content += unit.generateContainerSection()
	case "volume":
		content += unit.generateVolumeSection()
	case "network":
		content += unit.generateNetworkSection()
	case "build":
		content += unit.generateBuildSection()
	}

	content += unit.generateServiceSection()
	return content
}

func formatKeyValue(key, value string) string {
	return key + "=" + value + "\n"
}

func formatKeyValueSlice(key string, values []string) string {
	// Create empty string slice to collect sorted values
	sortedValues := make([]string, 0, len(values))

	// Use our helper to collect values in sorted order
	sorting.SortAndIterateSlice(values, func(item string) {
		sortedValues = append(sortedValues, item)
	})

	// Join them with spaces
	return key + "=" + strings.Join(sortedValues, " ") + "\n"
}

func formatSecret(secret Secret) string {
	// Always start with the source
	secretOpts := []string{secret.Source}

	// Add optional fields in a deterministic order
	// Create options in a specific order based on field name
	options := make(map[string]string)

	if secret.Type != "" {
		options["type"] = secret.Type
	}
	if secret.Target != "" {
		options["target"] = secret.Target
	}
	if secret.UID != "" {
		options["uid"] = secret.UID
	}
	if secret.GID != "" {
		options["gid"] = secret.GID
	}
	if secret.Mode != "" {
		options["mode"] = secret.Mode
	}

	// Get sorted keys for deterministic ordering
	keys := sorting.GetSortedMapKeys(options)

	// Add options in sorted order
	for _, k := range keys {
		secretOpts = append(secretOpts, k+"="+options[k])
	}

	return strings.Join(secretOpts, ",")
}
