package model

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

// Volume represents the configuration for a volume in a Quadlet unit.
type Volume struct {
	Name                 string
	UnitType             string
	ContainersConfModule []string `yaml:"containers_conf_module"`
	Copy                 bool     `yaml:"copy"`
	Device               string   `yaml:"device"`
	Driver               string   `yaml:"driver"`
	GlobalArgs           []string `yaml:"global_args"`
	Group                string   `yaml:"group"`
	Image                string   `yaml:"image"`
	Label                []string `yaml:"label"`
	Options              []string `yaml:"options"`
	PodmanArgs           []string `yaml:"podman_args"`
	Type                 string   `yaml:"type"`
	User                 string   `yaml:"user"`
	VolumeName           string   `yaml:"volume_name"`
}

// NewVolume creates a new Volume with the given name.
func NewVolume(name string) *Volume {
	return &Volume{
		Name:     name,
		UnitType: "volume",
	}
}

// FromComposeVolume creates a Volume from a Docker Compose volume configuration.
func (v *Volume) FromComposeVolume(name string, volume types.VolumeConfig) *Volume {
	// Set the volume name (if specified in the compose file, otherwise use the key name)
	if volume.Name != "" {
		v.VolumeName = volume.Name
	} else {
		v.VolumeName = name
	}

	// Set driver if specified
	if volume.Driver != "" {
		v.Driver = volume.Driver
	}

	// Convert driver options to volume options
	if len(volume.DriverOpts) > 0 {
		for key, value := range volume.DriverOpts {
			v.Options = append(v.Options, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Add labels
	if len(volume.Labels) > 0 {
		v.Label = append(v.Label, volume.Labels.AsList()...)
	}

	return v
}

// GetServiceName returns the full systemd service name.
func (v *Volume) GetServiceName() string {
	return v.Name + "-volume.service"
}

// GetUnitType returns the type of the unit.
func (v *Volume) GetUnitType() string {
	return "volume"
}

// GetUnitName returns the name of the unit.
func (v *Volume) GetUnitName() string {
	return v.Name
}