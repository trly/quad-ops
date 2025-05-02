package unit

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/trly/quad-ops/internal/util"
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
}

// GetSystemdUnit returns the appropriate SystemdUnit implementation for this QuadletUnit.
func (u *QuadletUnit) GetSystemdUnit() SystemdUnit {
	switch u.Type {
	case "container":
		container := u.Container
		container.Name = u.Name
		container.UnitType = "container"
		return &container
	case "volume":
		volume := u.Volume
		volume.Name = u.Name
		volume.UnitType = "volume"
		return &volume
	case "network":
		network := u.Network
		network.Name = u.Name
		network.UnitType = "network"
		return &network
	default:
		// Default to base implementation
		return &BaseSystemdUnit{Name: u.Name, Type: u.Type}
	}
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

// Unit represents a record in the units table.
type Unit struct {
	ID            int64     `db:"id"`
	Name          string    `db:"name"`
	Type          string    `db:"type"`
	CleanupPolicy string    `db:"cleanup_policy"`
	SHA1Hash      []byte    `db:"sha1_hash"`
	UserMode      bool      `db:"user_mode"`
	CreatedAt     time.Time `db:"created_at"` // Set by database, but not updated on every change
}

func (u *QuadletUnit) generateContainerSection() string {
	content := "\n[Container]\n"
	if u.Container.Image != "" {
		content += formatKeyValue("Image", u.Container.Image)
	}
	content += formatKeyValue("Label", "managed-by=quad-ops")

	// Use centralized sorting function for consistent output
	util.SortAndIterateSlice(u.Container.Label, func(label string) {
		content += formatKeyValue("Label", label)
	})

	// Use centralized sorting function for ports
	util.SortAndIterateSlice(u.Container.PublishPort, func(port string) {
		content += formatKeyValue("PublishPort", port)
	})

	// Use sortedEnvKeys if available (populated by SortAllSlices), 
	// otherwise generate sorted keys on the fly
	var envKeys []string
	if len(u.Container.sortedEnvKeys) > 0 {
		envKeys = u.Container.sortedEnvKeys
	} else {
		// Sort environment variables for consistent output
		envKeys = util.GetSortedMapKeys(u.Container.Environment)
	}

	// Add environment variables in sorted order
	for _, k := range envKeys {
		content += formatKeyValue("Environment", fmt.Sprintf("%s=%s", k, u.Container.Environment[k]))
	}
	// Use centralized sorting function for environment files
	util.SortAndIterateSlice(u.Container.EnvironmentFile, func(envFile string) {
		content += formatKeyValue("EnvironmentFile", envFile)
	})

	// Use centralized sorting function for volumes
	util.SortAndIterateSlice(u.Container.Volume, func(vol string) {
		content += formatKeyValue("Volume", vol)
	})

	// Use centralized sorting function for networks
	util.SortAndIterateSlice(u.Container.Network, func(net string) {
		content += formatKeyValue("Network", net)
	})

	// Use centralized sorting function for network aliases
	util.SortAndIterateSlice(u.Container.NetworkAlias, func(alias string) {
		content += formatKeyValue("NetworkAlias", alias)
	})
	if len(u.Container.Exec) > 0 {
		content += formatKeyValueSlice("Exec", u.Container.Exec)
	}
	if len(u.Container.Entrypoint) > 0 {
		content += formatKeyValueSlice("Entrypoint", u.Container.Entrypoint)
	}
	if u.Container.User != "" {
		content += formatKeyValue("User", u.Container.User)
	}
	if u.Container.Group != "" {
		content += formatKeyValue("Group", u.Container.Group)
	}
	if u.Container.WorkingDir != "" {
		content += formatKeyValue("WorkingDir", u.Container.WorkingDir)
	}
	if *u.Container.RunInit {
		content += formatKeyValue("RunInit", "yes")
	}
	// Privileged is not supported by podman-systemd
	if u.Container.ReadOnly {
		content += formatKeyValue("ReadOnly", "yes")
	}
	// SecurityLabel is not supported by podman-systemd
	// Use specific labels like SecurityLabelType instead
	if u.Container.HostName != "" {
		content += formatKeyValue("HostName", u.Container.HostName)
	}
	// Set ContainerName to override systemd- prefix if useSystemdDNS is false
	if u.Container.ContainerName != "" {
		content += formatKeyValue("ContainerName", u.Container.ContainerName)
	}
	for _, secret := range u.Container.Secrets {
		content += formatKeyValue("Secret", formatSecret(secret))
	}
	return content
}

func (u *QuadletUnit) generateVolumeSection() string {
	content := "\n[Volume]\n"
	content += formatKeyValue("Label", "managed-by=quad-ops")

	// Use centralized sorting function for volume labels
	util.SortAndIterateSlice(u.Volume.Label, func(label string) {
		content += formatKeyValue("Label", label)
	})

	// Set VolumeName to override systemd- prefix if configured
	if u.Volume.VolumeName != "" {
		content += formatKeyValue("VolumeName", u.Volume.VolumeName)
	}

	if u.Volume.Device != "" {
		content += formatKeyValue("Device", u.Volume.Device)
	}

	// Use centralized sorting function for volume options
	util.SortAndIterateSlice(u.Volume.Options, func(opt string) {
		content += formatKeyValue("Options", opt)
	})
	if u.Volume.Copy {
		content += formatKeyValue("Copy", "yes")
	}
	if u.Volume.Group != "" {
		content += formatKeyValue("Group", u.Volume.Group)
	}
	if u.Volume.Type != "" {
		content += formatKeyValue("Type", u.Volume.Type)
	}
	return content
}

func (u *QuadletUnit) generateNetworkSection() string {
	content := "\n[Network]\n"
	content += formatKeyValue("Label", "managed-by=quad-ops")

	// Use centralized sorting function for network labels
	util.SortAndIterateSlice(u.Network.Label, func(label string) {
		content += formatKeyValue("Label", label)
	})

	// Set NetworkName to override systemd- prefix if configured
	if u.Network.NetworkName != "" {
		content += formatKeyValue("NetworkName", u.Network.NetworkName)
	}

	if u.Network.Driver != "" {
		content += formatKeyValue("Driver", u.Network.Driver)
	}
	if u.Network.Gateway != "" {
		content += formatKeyValue("Gateway", u.Network.Gateway)
	}
	if u.Network.IPRange != "" {
		content += formatKeyValue("IPRange", u.Network.IPRange)
	}
	if u.Network.Subnet != "" {
		content += formatKeyValue("Subnet", u.Network.Subnet)
	}
	if u.Network.IPv6 {
		content += formatKeyValue("IPv6", "yes")
	}
	if u.Network.Internal {
		content += formatKeyValue("Internal", "yes")
	}
	// DNSEnabled is not supported by podman-systemd

	// Use centralized sorting function for network options
	util.SortAndIterateSlice(u.Network.Options, func(opt string) {
		content += formatKeyValue("Options", opt)
	})
	return content
}

func (u *QuadletUnit) generateUnitSection() string {
	content := "[Unit]\n"
	if u.Systemd.Description != "" {
		content += formatKeyValue("Description", u.Systemd.Description)
	}

	// Sort all systemd directives for consistent output
	if len(u.Systemd.After) > 0 {
		content += formatKeyValueSlice("After", u.Systemd.After)
	}

	if len(u.Systemd.Before) > 0 {
		content += formatKeyValueSlice("Before", u.Systemd.Before)
	}

	if len(u.Systemd.Requires) > 0 {
		content += formatKeyValueSlice("Requires", u.Systemd.Requires)
	}

	if len(u.Systemd.Wants) > 0 {
		content += formatKeyValueSlice("Wants", u.Systemd.Wants)
	}

	if len(u.Systemd.Conflicts) > 0 {
		content += formatKeyValueSlice("Conflicts", u.Systemd.Conflicts)
	}

	if len(u.Systemd.PartOf) > 0 {
		content += formatKeyValueSlice("PartOf", u.Systemd.PartOf)
	}

	if len(u.Systemd.PropagatesReloadTo) > 0 {
		content += formatKeyValueSlice("PropagatesReloadTo", u.Systemd.PropagatesReloadTo)
	}
	return content
}

func (u *QuadletUnit) generateServiceSection() string {
	content := "\n[Service]\n"
	if u.Systemd.Type != "" {
		content += formatKeyValue("Type", u.Systemd.Type)
	}
	if u.Systemd.RestartPolicy != "" {
		content += formatKeyValue("Restart", u.Systemd.RestartPolicy)
	}
	if u.Systemd.TimeoutStartSec != 0 {
		content += formatKeyValue("TimeoutStartSec", fmt.Sprintf("%d", u.Systemd.TimeoutStartSec))
	}
	if u.Systemd.RemainAfterExit {
		content += formatKeyValue("RemainAfterExit", "yes")
	}
	return content
}

// GenerateQuadletUnit generates a quadlet unit file content from a unit configuration.
func GenerateQuadletUnit(unit QuadletUnit, verbose bool) string {
	if verbose {
		log.Printf("generating Quadlet unit for %s of type %s", unit.Name, unit.Type)
	}

	content := unit.generateUnitSection()

	switch unit.Type {
	case "container":
		content += unit.generateContainerSection()
	case "volume":
		content += unit.generateVolumeSection()
	case "network":
		content += unit.generateNetworkSection()
	}

	content += unit.generateServiceSection()
	return content
}

func formatKeyValue(key, value string) string {
	return fmt.Sprintf("%s=%s\n", key, value)
}

func formatKeyValueSlice(key string, values []string) string {
	// Create empty string slice to collect sorted values
	sortedValues := make([]string, 0, len(values))
	
	// Use our helper to collect values in sorted order
	util.SortAndIterateSlice(values, func(item string) {
		sortedValues = append(sortedValues, item)
	})
	
	// Join them with spaces
	return fmt.Sprintf("%s=%s\n", key, strings.Join(sortedValues, " "))
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
	keys := util.GetSortedMapKeys(options)
	
	// Add options in sorted order
	for _, k := range keys {
		secretOpts = append(secretOpts, fmt.Sprintf("%s=%s", k, options[k]))
	}

	return strings.Join(secretOpts, ",")
}
