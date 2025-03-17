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
	Name      string          `yaml:"name"`
	Type      string          `yaml:"type"`
	Enabled   bool            `yaml:"enabled,omitempty"`
	AutoStart bool            `yaml:"auto_start,omitmpty"`
	Systemd   SystemdConfig   `yaml:"systemd"`
	Container ContainerConfig `yaml:"container,omitempty"`
	Volume    VolumeConfig    `yaml:"volume,omitempty"`
	Network   NetworkConfig   `yaml:"network,omitempty"`
	Image     ImageConfig     `yaml:"image,omitempty"`
}

// SystemdConfig represents the configuration for a systemd unit.
// It includes settings such as the unit description, dependencies,
// restart policy, and other systemd-specific options.
type SystemdConfig struct {
	Description     string   `yaml:"description"`
	Documentation   []string `yaml:"documentation"`
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

func (unit *QuadletUnit) generateContainerSection() string {
	content := "\n[Container]\n"
	if unit.Container.Image != "" {
		content += formatKeyValue("Image", unit.Container.Image)
	}
	content += formatKeyValue("Label", "managed-by=quad-ops")
	for _, label := range unit.Container.Label {
		content += formatKeyValue("Label", label)
	}
	for _, port := range unit.Container.PublishPort {
		content += formatKeyValue("PublishPort", port)
	}
	for k, v := range unit.Container.Environment {
		content += formatKeyValue("Environment", fmt.Sprintf("%s=%s", k, v))
	}
	if unit.Container.EnvironmentFile != "" {
		content += formatKeyValue("EnvironmentFile", unit.Container.EnvironmentFile)
	}
	for _, vol := range unit.Container.Volume {
		content += formatKeyValue("Volume", vol)
	}
	for _, net := range unit.Container.Network {
		content += formatKeyValue("Network", net)
	}
	if len(unit.Container.Command) > 0 {
		content += formatKeyValueSlice("Command", unit.Container.Command)
	}
	if len(unit.Container.Entrypoint) > 0 {
		content += formatKeyValueSlice("Entrypoint", unit.Container.Entrypoint)
	}
	if unit.Container.User != "" {
		content += formatKeyValue("User", unit.Container.User)
	}
	if unit.Container.Group != "" {
		content += formatKeyValue("Group", unit.Container.Group)
	}
	if unit.Container.WorkingDir != "" {
		content += formatKeyValue("WorkingDir", unit.Container.WorkingDir)
	}
	if len(unit.Container.PodmanArgs) > 0 {
		content += formatKeyValueSlice("PodmanArgs", unit.Container.PodmanArgs)
	}
	if unit.Container.RunInit {
		content += formatKeyValue("RunInit", "yes")
	}
	if unit.Container.Notify {
		content += formatKeyValue("Notify", "yes")
	}
	if unit.Container.Privileged {
		content += formatKeyValue("Privileged", "yes")
	}
	if unit.Container.ReadOnly {
		content += formatKeyValue("ReadOnly", "yes")
	}
	for _, label := range unit.Container.SecurityLabel {
		content += formatKeyValue("SecurityLabel", label)
	}
	if unit.Container.HostName != "" {
		content += formatKeyValue("HostName", unit.Container.HostName)
	}
	for _, secret := range unit.Container.Secrets {
		content += formatKeyValue("Secret", formatSecret(secret))
	}
	return content
}

func (unit *QuadletUnit) generateVolumeSection() string {
	content := "\n[Volume]\n"
	content += formatKeyValue("Label", "managed-by=quad-ops")
	for _, label := range unit.Volume.Label {
		content += formatKeyValue("Label", label)
	}
	if unit.Volume.Device != "" {
		content += formatKeyValue("Device", unit.Volume.Device)
	}
	for _, opt := range unit.Volume.Options {
		content += formatKeyValue("Options", opt)
	}
	if unit.Volume.Copy {
		content += formatKeyValue("Copy", "yes")
	}
	if unit.Volume.Group != "" {
		content += formatKeyValue("Group", unit.Volume.Group)
	}
	if unit.Volume.Type != "" {
		content += formatKeyValue("Type", unit.Volume.Type)
	}
	return content
}

func (unit *QuadletUnit) generateNetworkSection() string {
	content := "\n[Network]\n"
	content += formatKeyValue("Label", "managed-by=quad-ops")
	for _, label := range unit.Network.Label {
		content += formatKeyValue("Label", label)
	}
	if unit.Network.Driver != "" {
		content += formatKeyValue("Driver", unit.Network.Driver)
	}
	if unit.Network.Gateway != "" {
		content += formatKeyValue("Gateway", unit.Network.Gateway)
	}
	if unit.Network.IPRange != "" {
		content += formatKeyValue("IPRange", unit.Network.IPRange)
	}
	if unit.Network.Subnet != "" {
		content += formatKeyValue("Subnet", unit.Network.Subnet)
	}
	if unit.Network.IPv6 {
		content += formatKeyValue("IPv6", "yes")
	}
	if unit.Network.Internal {
		content += formatKeyValue("Internal", "yes")
	}
	if unit.Network.DNSEnabled {
		content += formatKeyValue("DNSEnabled", "yes")
	}
	for _, opt := range unit.Network.Options {
		content += formatKeyValue("Options", opt)
	}
	return content
}

func (unit *QuadletUnit) generateImageSection() string {
	content := "\n[Image]\n"
	if unit.Image.Image != "" {
		content += formatKeyValue("Image", unit.Image.Image)
	}
	if len(unit.Image.PodmanArgs) > 0 {
		content += formatKeyValueSlice("PodmanArgs", unit.Image.PodmanArgs)
	}
	return content
}

func (unit *QuadletUnit) generateUnitSection() string {
	content := "[Unit]\n"
	if unit.Systemd.Description != "" {
		content += formatKeyValue("Description", unit.Systemd.Description)
	}
	for _, documentation := range unit.Systemd.Documentation {
		content += formatKeyValue("Documentation", documentation)
	}
	if len(unit.Systemd.After) > 0 {
		content += formatKeyValueSlice("After", unit.Systemd.After)
	}
	if len(unit.Systemd.Before) > 0 {
		content += formatKeyValueSlice("Before", unit.Systemd.Before)
	}
	if len(unit.Systemd.Requires) > 0 {
		content += formatKeyValueSlice("Requires", unit.Systemd.Requires)
	}
	if len(unit.Systemd.Wants) > 0 {
		content += formatKeyValueSlice("Wants", unit.Systemd.Wants)
	}
	if len(unit.Systemd.Conflicts) > 0 {
		content += formatKeyValueSlice("Conflicts", unit.Systemd.Conflicts)
	}
	return content
}

func (unit *QuadletUnit) generateServiceSection() string {
	content := "\n[Service]\n"
	if unit.Systemd.Type != "" {
		content += formatKeyValue("Type", unit.Systemd.Type)
	}
	if unit.Systemd.RestartPolicy != "" {
		content += formatKeyValue("Restart", unit.Systemd.RestartPolicy)
	}
	if unit.Systemd.TimeoutStartSec != 0 {
		content += formatKeyValue("TimeoutStartSec", fmt.Sprintf("%d", unit.Systemd.TimeoutStartSec))
	}
	if unit.Systemd.RemainAfterExit {
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

func formatSecret(secret SecretConfig) string {
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

	return formatKeyValue("Secret", strings.Join(secretOpts, ","))
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
