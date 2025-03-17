package unit

type VolumeConfig struct {
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
