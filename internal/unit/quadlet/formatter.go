// Package quadlet handles quadlet unit file generation
package quadlet

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/trly/quad-ops/internal/unit/model"
)

// formatKeyValue formats a key-value pair as "key=value".
func formatKeyValue(key, value string) string {
	return fmt.Sprintf("%s=%s\n", key, value)
}

// formatKeyValueSlice formats a key with multiple values using space separation.
func formatKeyValueSlice(key string, values []string) string {
	// Make a copy to avoid modifying the original slice
	sorted := make([]string, len(values))
	copy(sorted, values)
	sort.Strings(sorted)
	return fmt.Sprintf("%s=%s\n", key, strings.Join(sorted, " "))
}

// formatSecret formats a secret configuration.
func formatSecret(secret model.Secret) string {
	secretOpts := []string{secret.Source}

	// Add type if specified (needed for env secrets)
	if secret.Type != "" {
		secretOpts = append(secretOpts, fmt.Sprintf("type=%s", secret.Type))
	}

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

// GenerateQuadletUnit generates a quadlet unit file content from a unit configuration.
func GenerateQuadletUnit(unit model.QuadletUnitConfig, verbose bool) string {
	if verbose {
		log.Printf("generating Quadlet unit for %s of type %s", unit.Name, unit.Type)
	}

	content := generateUnitSection(unit)

	switch unit.Type {
	case "container":
		content += generateContainerSection(unit)
	case "volume":
		content += generateVolumeSection(unit)
	case "network":
		content += generateNetworkSection(unit)
	}

	content += generateServiceSection(unit)
	return content
}

// generateUnitSection generates the [Unit] section of a systemd unit file.
func generateUnitSection(unit model.QuadletUnitConfig) string {
	content := "[Unit]\n"
	if unit.Systemd.Description != "" {
		content += formatKeyValue("Description", unit.Systemd.Description)
	}

	// Sort all systemd directives for consistent output
	if len(unit.Systemd.After) > 0 {
		slice := make([]string, len(unit.Systemd.After))
		copy(slice, unit.Systemd.After)
		sort.Strings(slice)
		content += formatKeyValueSlice("After", slice)
	}

	if len(unit.Systemd.Before) > 0 {
		slice := make([]string, len(unit.Systemd.Before))
		copy(slice, unit.Systemd.Before)
		sort.Strings(slice)
		content += formatKeyValueSlice("Before", slice)
	}

	if len(unit.Systemd.Requires) > 0 {
		slice := make([]string, len(unit.Systemd.Requires))
		copy(slice, unit.Systemd.Requires)
		sort.Strings(slice)
		content += formatKeyValueSlice("Requires", slice)
	}

	if len(unit.Systemd.Wants) > 0 {
		slice := make([]string, len(unit.Systemd.Wants))
		copy(slice, unit.Systemd.Wants)
		sort.Strings(slice)
		content += formatKeyValueSlice("Wants", slice)
	}

	if len(unit.Systemd.Conflicts) > 0 {
		slice := make([]string, len(unit.Systemd.Conflicts))
		copy(slice, unit.Systemd.Conflicts)
		sort.Strings(slice)
		content += formatKeyValueSlice("Conflicts", slice)
	}

	if len(unit.Systemd.PartOf) > 0 {
		slice := make([]string, len(unit.Systemd.PartOf))
		copy(slice, unit.Systemd.PartOf)
		sort.Strings(slice)
		content += formatKeyValueSlice("PartOf", slice)
	}

	if len(unit.Systemd.PropagatesReloadTo) > 0 {
		slice := make([]string, len(unit.Systemd.PropagatesReloadTo))
		copy(slice, unit.Systemd.PropagatesReloadTo)
		sort.Strings(slice)
		content += formatKeyValueSlice("PropagatesReloadTo", slice)
	}
	return content
}

// generateServiceSection generates the [Service] section of a systemd unit file.
func generateServiceSection(unit model.QuadletUnitConfig) string {
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

// generateContainerSection generates the [Container] section of a quadlet unit file.
func generateContainerSection(unit model.QuadletUnitConfig) string {
	content := "\n[Container]\n"
	if unit.Container.Image != "" {
		content += formatKeyValue("Image", unit.Container.Image)
	}
	content += formatKeyValue("Label", "managed-by=quad-ops")

	// Sort labels for consistent output
	slice := make([]string, len(unit.Container.Label))
	copy(slice, unit.Container.Label)
	sort.Strings(slice)
	for _, label := range slice {
		content += formatKeyValue("Label", label)
	}

	// Sort ports for consistent output
	slice = make([]string, len(unit.Container.PublishPort))
	copy(slice, unit.Container.PublishPort)
	sort.Strings(slice)
	for _, port := range slice {
		content += formatKeyValue("PublishPort", port)
	}

	// Sort environment variables for consistent output
	envKeys := make([]string, 0, len(unit.Container.Environment))
	for k := range unit.Container.Environment {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)

	// Add environment variables in sorted order
	for _, k := range envKeys {
		content += formatKeyValue("Environment", fmt.Sprintf("%s=%s", k, unit.Container.Environment[k]))
	}

	// Sort environment files for consistent output
	slice = make([]string, len(unit.Container.EnvironmentFile))
	copy(slice, unit.Container.EnvironmentFile)
	sort.Strings(slice)
	for _, envFile := range slice {
		content += formatKeyValue("EnvironmentFile", envFile)
	}

	// Sort volumes for consistent output
	slice = make([]string, len(unit.Container.Volume))
	copy(slice, unit.Container.Volume)
	sort.Strings(slice)
	for _, vol := range slice {
		content += formatKeyValue("Volume", vol)
	}

	// Sort networks for consistent output
	slice = make([]string, len(unit.Container.Network))
	copy(slice, unit.Container.Network)
	sort.Strings(slice)
	for _, net := range slice {
		content += formatKeyValue("Network", net)
	}

	// Sort network aliases for consistent output
	slice = make([]string, len(unit.Container.NetworkAlias))
	copy(slice, unit.Container.NetworkAlias)
	sort.Strings(slice)
	for _, alias := range slice {
		content += formatKeyValue("NetworkAlias", alias)
	}

	if len(unit.Container.Exec) > 0 {
		content += formatKeyValueSlice("Exec", unit.Container.Exec)
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
	if *unit.Container.RunInit {
		content += formatKeyValue("RunInit", "yes")
	}
	if unit.Container.ReadOnly {
		content += formatKeyValue("ReadOnly", "yes")
	}
	if unit.Container.HostName != "" {
		content += formatKeyValue("HostName", unit.Container.HostName)
	}
	// Set ContainerName to override systemd- prefix if useSystemdDNS is false
	if unit.Container.ContainerName != "" {
		content += formatKeyValue("ContainerName", unit.Container.ContainerName)
	}
	for _, secret := range unit.Container.Secrets {
		content += formatKeyValue("Secret", formatSecret(secret))
	}
	return content
}

// generateVolumeSection generates the [Volume] section of a quadlet unit file.
func generateVolumeSection(unit model.QuadletUnitConfig) string {
	content := "\n[Volume]\n"
	content += formatKeyValue("Label", "managed-by=quad-ops")

	// Sort labels for consistent output
	slice := make([]string, len(unit.Volume.Label))
	copy(slice, unit.Volume.Label)
	sort.Strings(slice)
	for _, label := range slice {
		content += formatKeyValue("Label", label)
	}

	// Set VolumeName to override systemd- prefix if configured
	if unit.Volume.VolumeName != "" {
		content += formatKeyValue("VolumeName", unit.Volume.VolumeName)
	}

	if unit.Volume.Device != "" {
		content += formatKeyValue("Device", unit.Volume.Device)
	}

	// Sort options for consistent output
	slice = make([]string, len(unit.Volume.Options))
	copy(slice, unit.Volume.Options)
	sort.Strings(slice)
	for _, opt := range slice {
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

// generateNetworkSection generates the [Network] section of a quadlet unit file.
func generateNetworkSection(unit model.QuadletUnitConfig) string {
	content := "\n[Network]\n"
	content += formatKeyValue("Label", "managed-by=quad-ops")

	// Sort labels for consistent output
	slice := make([]string, len(unit.Network.Label))
	copy(slice, unit.Network.Label)
	sort.Strings(slice)
	for _, label := range slice {
		content += formatKeyValue("Label", label)
	}

	// Set NetworkName to override systemd- prefix if configured
	if unit.Network.NetworkName != "" {
		content += formatKeyValue("NetworkName", unit.Network.NetworkName)
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

	// Sort options for consistent output
	slice = make([]string, len(unit.Network.Options))
	copy(slice, unit.Network.Options)
	sort.Strings(slice)
	for _, opt := range slice {
		content += formatKeyValue("Options", opt)
	}
	return content
}
