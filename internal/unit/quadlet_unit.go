package unit

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
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
	CreatedAt     time.Time `db:"created_at"` // Set by database, but not updated on every change
}

func (u *QuadletUnit) generateContainerSection() string {
	content := "\n[Container]\n"
	if u.Container.Image != "" {
		content += formatKeyValue("Image", u.Container.Image)
	}
	content += formatKeyValue("Label", "managed-by=quad-ops")

	// Sort labels for consistent output
	slice := make([]string, len(u.Container.Label))
	copy(slice, u.Container.Label)
	sort.Strings(slice)
	for _, label := range slice {
		content += formatKeyValue("Label", label)
	}

	// Sort ports for consistent output
	slice = make([]string, len(u.Container.PublishPort))
	copy(slice, u.Container.PublishPort)
	sort.Strings(slice)
	for _, port := range slice {
		content += formatKeyValue("PublishPort", port)
	}

	// Sort environment variables for consistent output
	envKeys := make([]string, 0, len(u.Container.Environment))
	for k := range u.Container.Environment {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)

	// Add environment variables in sorted order
	for _, k := range envKeys {
		content += formatKeyValue("Environment", fmt.Sprintf("%s=%s", k, u.Container.Environment[k]))
	}
	// Sort environment files for consistent output
	slice = make([]string, len(u.Container.EnvironmentFile))
	copy(slice, u.Container.EnvironmentFile)
	sort.Strings(slice)
	for _, envFile := range slice {
		content += formatKeyValue("EnvironmentFile", envFile)
	}

	// Sort volumes for consistent output
	slice = make([]string, len(u.Container.Volume))
	copy(slice, u.Container.Volume)
	sort.Strings(slice)
	for _, vol := range slice {
		content += formatKeyValue("Volume", vol)
	}

	// Sort networks for consistent output
	slice = make([]string, len(u.Container.Network))
	copy(slice, u.Container.Network)
	sort.Strings(slice)
	for _, net := range slice {
		content += formatKeyValue("Network", net)
	}

	// Sort network aliases for consistent output
	slice = make([]string, len(u.Container.NetworkAlias))
	copy(slice, u.Container.NetworkAlias)
	sort.Strings(slice)
	for _, alias := range slice {
		content += formatKeyValue("NetworkAlias", alias)
	}
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

	// Sort labels for consistent output
	slice := make([]string, len(u.Volume.Label))
	copy(slice, u.Volume.Label)
	sort.Strings(slice)
	for _, label := range slice {
		content += formatKeyValue("Label", label)
	}

	if u.Volume.Device != "" {
		content += formatKeyValue("Device", u.Volume.Device)
	}

	// Sort options for consistent output
	slice = make([]string, len(u.Volume.Options))
	copy(slice, u.Volume.Options)
	sort.Strings(slice)
	for _, opt := range slice {
		content += formatKeyValue("Options", opt)
	}
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

	// Sort labels for consistent output
	slice := make([]string, len(u.Network.Label))
	copy(slice, u.Network.Label)
	sort.Strings(slice)
	for _, label := range slice {
		content += formatKeyValue("Label", label)
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

	// Sort options for consistent output
	slice = make([]string, len(u.Network.Options))
	copy(slice, u.Network.Options)
	sort.Strings(slice)
	for _, opt := range slice {
		content += formatKeyValue("Options", opt)
	}
	return content
}

func (u *QuadletUnit) generateUnitSection() string {
	content := "[Unit]\n"
	if u.Systemd.Description != "" {
		content += formatKeyValue("Description", u.Systemd.Description)
	}

	// Sort all systemd directives for consistent output
	if len(u.Systemd.After) > 0 {
		slice := make([]string, len(u.Systemd.After))
		copy(slice, u.Systemd.After)
		sort.Strings(slice)
		content += formatKeyValueSlice("After", slice)
	}

	if len(u.Systemd.Before) > 0 {
		slice := make([]string, len(u.Systemd.Before))
		copy(slice, u.Systemd.Before)
		sort.Strings(slice)
		content += formatKeyValueSlice("Before", slice)
	}

	if len(u.Systemd.Requires) > 0 {
		slice := make([]string, len(u.Systemd.Requires))
		copy(slice, u.Systemd.Requires)
		sort.Strings(slice)
		content += formatKeyValueSlice("Requires", slice)
	}

	if len(u.Systemd.Wants) > 0 {
		slice := make([]string, len(u.Systemd.Wants))
		copy(slice, u.Systemd.Wants)
		sort.Strings(slice)
		content += formatKeyValueSlice("Wants", slice)
	}

	if len(u.Systemd.Conflicts) > 0 {
		slice := make([]string, len(u.Systemd.Conflicts))
		copy(slice, u.Systemd.Conflicts)
		sort.Strings(slice)
		content += formatKeyValueSlice("Conflicts", slice)
	}

	if len(u.Systemd.PartOf) > 0 {
		slice := make([]string, len(u.Systemd.PartOf))
		copy(slice, u.Systemd.PartOf)
		sort.Strings(slice)
		content += formatKeyValueSlice("PartOf", slice)
	}

	if len(u.Systemd.PropagatesReloadTo) > 0 {
		slice := make([]string, len(u.Systemd.PropagatesReloadTo))
		copy(slice, u.Systemd.PropagatesReloadTo)
		sort.Strings(slice)
		content += formatKeyValueSlice("PropagatesReloadTo", slice)
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
	// Make a copy to avoid modifying the original slice
	sorted := make([]string, len(values))
	copy(sorted, values)
	sort.Strings(sorted)
	return fmt.Sprintf("%s=%s\n", key, strings.Join(sorted, " "))
}

func formatSecret(secret Secret) string {
	secretOpts := []string{secret.Source}

	if secret.Target != "" {
		secretOpts = append(secretOpts, fmt.Sprintf("target=%s", secret.Target))
	}
	if secret.UID != "" {
		secretOpts = append(secretOpts, fmt.Sprintf("uid=%s", secret.UID))
	}
	if secret.GID != "" {
		secretOpts = append(secretOpts, fmt.Sprintf("gid=%s", secret.GID))
	}
	if secret.Mode != "" {
		secretOpts = append(secretOpts, fmt.Sprintf("mode=%s", secret.Mode))
	}

	return strings.Join(secretOpts, ",")
}
