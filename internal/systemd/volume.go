package systemd

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/ini.v1"
)

// BuildVolume converts a compose volume into a volume unit file.
func BuildVolume(projectName, volName string, vol *types.VolumeConfig) Unit {
	file := ini.Empty()
	section, _ := file.NewSection("Volume")
	sectionMap := make(map[string]string)
	buildVolumeSection(volName, vol, sectionMap)
	// Copy sectionMap to ini section
	for key, value := range sectionMap {
		_, _ = section.NewKey(key, value)
	}

	return Unit{
		Name: fmt.Sprintf("%s-%s.volume", projectName, volName),
		File: file,
	}
}

func buildVolumeSection(_ string, vol *types.VolumeConfig, section map[string]string) {
	// Driver mapping
	if vol.Driver != "" {
		section["Driver"] = vol.Driver
	}

	// VolumeName: custom name or defaults to systemd-$name
	if vol.Name != "" {
		section["VolumeName"] = vol.Name
	}

	// Labels: map compose labels to systemd Label= directives
	// Uses dot-notation for multi-value serialization: Label.key=value
	for k, v := range vol.Labels {
		section[fmt.Sprintf("Label.%s", k)] = v
	}

	// DriverOpts mapping to Podman systemd directives
	if len(vol.DriverOpts) > 0 {
		mapDriverOpts(vol.DriverOpts, section)
	}

	// x-quad-ops-podman-args: list of global podman arguments
	if globalArgs, ok := vol.Extensions["x-quad-ops-podman-args"].([]interface{}); ok {
		for i, arg := range globalArgs {
			if argStr, ok := arg.(string); ok {
				section[fmt.Sprintf("GlobalArgs.%d", i)] = argStr
			}
		}
	}

	// x-quad-ops-volume-args: list of volume-specific podman arguments
	if volumeArgs, ok := vol.Extensions["x-quad-ops-volume-args"].([]interface{}); ok {
		for i, arg := range volumeArgs {
			if argStr, ok := arg.(string); ok {
				section[fmt.Sprintf("PodmanArgs.%d", i)] = argStr
			}
		}
	}
}

// mapDriverOpts maps compose driver options to Podman systemd [Volume] directives.
// See: https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html#volume-units-volume
func mapDriverOpts(opts map[string]string, section map[string]string) {
	for k, v := range opts {
		switch k {
		case "copy":
			// Copy=true → --opt copy
			if v == "true" {
				section["Copy"] = "true"
			}

		case "device":
			// Device=tmpfs → --opt device=tmpfs
			section["Device"] = v

		case "group":
			// Group=192 → --opt "o=group=192"
			section["Group"] = v

		case "image":
			// Image=quay.io/centos/centos:latest → --opt image=...
			section["Image"] = v

		case "options", "o":
			// Options=XYZ → --opt "o=XYZ"
			section["Options"] = v

		case "type":
			// Type=type → filesystem type of Device
			section["Type"] = v

		case "user", "uid":
			// User=123 → --opt "o=uid=123"
			section["User"] = v

		case "path":
			// Legacy path option, not part of standard Podman directives
			// Keep for backward compatibility
			if v != "" {
				section["Path"] = v
			}

		// Skip known systemd-specific options that shouldn't be in driver options
		case "module", "containers-conf-module":
			section["ContainersConfModule"] = v

		default:
			// Ignore unknown driver options to avoid polluting the section
			// with compose-specific or driver-specific settings
		}
	}
}
