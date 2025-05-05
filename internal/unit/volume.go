package unit

import (
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/compose"
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
			Name:     name,
			UnitType: "volume",
		},
	}
}

// FromComposeVolume creates a Volume from a Docker Compose volume configuration.
func (v *Volume) FromComposeVolume(name string, volume types.VolumeConfig) *Volume {
	// Set the volume name using the common name resolver
	v.VolumeName = compose.NameResolver(volume.Name, name)

	// Set driver if specified
	if volume.Driver != "" {
		v.Driver = volume.Driver
	}

	// Convert driver options to volume options using the common converter
	v.Options = compose.OptionsConverter(volume.DriverOpts)

	// Add labels using the common converter
	v.Label = compose.LabelConverter(volume.Labels)

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
