package unit

import (
	"fmt"
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/systemd"
)

// Volume represents the configuration for a volume in a Quadlet unit.
type Volume struct {
	BaseUnit                      // Embed the base struct
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
		BaseUnit: BaseUnit{
			BaseUnit: systemd.NewBaseUnit(name, "volume"),
			Name:     name,
			UnitType: "volume",
		},
	}
}

// FromComposeVolume creates a Volume from a Docker Compose volume configuration.
func (v *Volume) FromComposeVolume(name string, volume types.VolumeConfig) *Volume {
	// Set the volume name - use volume.Name if set, otherwise use the key name
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

	// Sort all slices for deterministic output
	sortVolume(v)

	return v
}

// sortVolume ensures all slices in a volume config are sorted deterministically in-place.
func sortVolume(v *Volume) {
	// Sort all slices for deterministic output
	if len(v.ContainersConfModule) > 0 {
		sort.Strings(v.ContainersConfModule)
	}

	if len(v.GlobalArgs) > 0 {
		sort.Strings(v.GlobalArgs)
	}

	if len(v.Label) > 0 {
		sort.Strings(v.Label)
	}

	if len(v.Options) > 0 {
		sort.Strings(v.Options)
	}

	if len(v.PodmanArgs) > 0 {
		sort.Strings(v.PodmanArgs)
	}
}
