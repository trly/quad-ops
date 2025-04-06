package unit

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/trly/quad-ops/internal/config"
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
	Image     Image         `yaml:"image,omitempty"`
}

// GetSystemdUnit returns the appropriate SystemdUnit implementation for this QuadletUnit
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
	case "image":
		image := u.Image
		image.Name = u.Name
		image.UnitType = "image"
		return &image
	default:
		// Default to base implementation
		return &BaseSystemdUnit{Name: u.Name, Type: u.Type}
	}
}

// SystemdConfig represents the configuration for a systemd unit.
// It includes settings such as the unit description, dependencies,
// restart policy, and other systemd-specific options.
type SystemdConfig struct {
	Description     string   `yaml:"description"`
	After           []string `yaml:"after"`
	Before          []string `yaml:"before"`
	Requires        []string `yaml:"requires"`
	Wants           []string `yaml:"wants"`
	Conflicts       []string `yaml:"conflicts"`
	RestartPolicy   string   `yaml:"restart_policy"`
	TimeoutStartSec int      `yaml:"timeout_start_sec"`
	Type            string   `yaml:"type"`
	RemainAfterExit bool     `yaml:"remain_after_exit"`
	WantedBy        []string `yaml:"wanted_by"`
}

// Unit represents a record in the units table
type Unit struct {
	ID            int64     `db:"id"`
	Name          string    `db:"name"`
	Type          string    `db:"type"`
	CleanupPolicy string    `db:"cleanup_policy"`
	SHA1Hash      []byte    `db:"sha1_hash"`
	CreatedAt     time.Time `db:"created_at"`
}

func (u *QuadletUnit) generateContainerSection() string {
	content := "\n[Container]\n"
	if u.Container.Image != "" {
		content += formatKeyValue("Image", u.Container.Image)
	}
	content += formatKeyValue("Label", "managed-by=quad-ops")
	for _, label := range u.Container.Label {
		content += formatKeyValue("Label", label)
	}
	for _, port := range u.Container.PublishPort {
		content += formatKeyValue("PublishPort", port)
	}
	for k, v := range u.Container.Environment {
		content += formatKeyValue("Environment", fmt.Sprintf("%s=%s", k, v))
	}
	if u.Container.EnvironmentFile != "" {
		content += formatKeyValue("EnvironmentFile", u.Container.EnvironmentFile)
	}
	for _, vol := range u.Container.Volume {
		content += formatKeyValue("Volume", vol)
	}
	for _, net := range u.Container.Network {
		content += formatKeyValue("Network", net)
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
	if len(u.Container.PodmanArgs) > 0 {
		content += formatKeyValueSlice("PodmanArgs", u.Container.PodmanArgs)
	}
	if u.Container.RunInit {
		content += formatKeyValue("RunInit", "yes")
	}
	if u.Container.Notify {
		content += formatKeyValue("Notify", "yes")
	}
	if u.Container.Privileged {
		content += formatKeyValue("Privileged", "yes")
	}
	if u.Container.ReadOnly {
		content += formatKeyValue("ReadOnly", "yes")
	}
	for _, label := range u.Container.SecurityLabel {
		content += formatKeyValue("SecurityLabel", label)
	}
	if u.Container.HostName != "" {
		content += formatKeyValue("HostName", u.Container.HostName)
	}
	for _, secret := range u.Container.Secrets {
		content += formatKeyValue("Secret", formatSecret(secret))
	}
	return content
}

func (u *QuadletUnit) generateVolumeSection() string {
	content := "\n[Volume]\n"
	content += formatKeyValue("Label", "managed-by=quad-ops")
	for _, label := range u.Volume.Label {
		content += formatKeyValue("Label", label)
	}
	if u.Volume.Device != "" {
		content += formatKeyValue("Device", u.Volume.Device)
	}
	for _, opt := range u.Volume.Options {
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
	for _, label := range u.Network.Label {
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
	if u.Network.DNSEnabled {
		content += formatKeyValue("DNSEnabled", "yes")
	}
	for _, opt := range u.Network.Options {
		content += formatKeyValue("Options", opt)
	}
	return content
}

func (u *QuadletUnit) generateImageSection() string {
	content := "\n[Image]\n"
	if u.Image.Image != "" {
		content += formatKeyValue("Image", u.Image.Image)
	}
	if len(u.Image.PodmanArgs) > 0 {
		content += formatKeyValueSlice("PodmanArgs", u.Image.PodmanArgs)
	}
	return content
}

func (u *QuadletUnit) generateUnitSection() string {
	content := "[Unit]\n"
	if u.Systemd.Description != "" {
		content += formatKeyValue("Description", u.Systemd.Description)
	}
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
	case "image":
		content += unit.generateImageSection()
	}

	content += unit.generateServiceSection()
	return content
}

func formatKeyValue(key, value string) string {
	return fmt.Sprintf("%s=%s\n", key, value)
}

func formatKeyValueSlice(key string, values []string) string {
	return fmt.Sprintf("%s=%s\n", key, strings.Join(values, " "))
}

func formatSecret(secret Secret) string {
	secretOpts := []string{secret.Name}

	if secret.Type != "" {
		secretOpts = append(secretOpts, fmt.Sprintf("type=%s", secret.Type))
	}
	if secret.Target != "" {
		secretOpts = append(secretOpts, fmt.Sprintf("target=%s", secret.Target))
	}
	if secret.UID != 0 {
		secretOpts = append(secretOpts, fmt.Sprintf("uid=%d", secret.UID))
	}
	if secret.GID != 0 {
		secretOpts = append(secretOpts, fmt.Sprintf("gid=%d", secret.GID))
	}
	if secret.Mode != "" {
		secretOpts = append(secretOpts, fmt.Sprintf("mode=%s", secret.Mode))
	}

	return strings.Join(secretOpts, ",")
}

func (p *Processor) processUnit(unit *QuadletUnit, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) error {
	unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
	processedUnits[unitKey] = true

	content := GenerateQuadletUnit(*unit, p.verbose)
	unitPath := filepath.Join(config.GetConfig().QuadletDir, unitKey)

	if !force && !p.hasUnitChanged(unitPath, content) {
		return nil
	}

	if err := p.writeUnitFile(unitPath, content); err != nil {
		return err
	}

	*changedUnits = append(*changedUnits, *unit)
	return p.updateUnitDatabase(unit, content)
}
