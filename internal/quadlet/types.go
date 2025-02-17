package quadlet

// QuadletUnit represents a unit configuration for Quadlet
type QuadletUnit struct {
	Name      string          `yaml:"name"`
	Type      string          `yaml:"type"`
	Systemd   SystemdConfig   `yaml:"systemd"`
	Container ContainerConfig `yaml:"container,omitempty"`
	Volume    VolumeConfig    `yaml:"volume,omitempty"`
	Network   NetworkConfig   `yaml:"network,omitempty"`
	Pod       PodConfig       `yaml:"pod,omitempty"`
	Kube      KubeConfig      `yaml:"kube,omitempty"`
	Image     ImageConfig     `yaml:"image,omitempty"`
	Build     BuildConfig     `yaml:"build,omitempty"`
}

// SystemdConfig represents systemd-specific configuration
type SystemdConfig struct {
	Description   string   `yaml:"description"`
	After         []string `yaml:"after"`
	RestartPolicy string   `yaml:"restart_policy"`
}

// ContainerConfig represents container-specific configuration
type ContainerConfig struct {
	Image       string   `yaml:"image"`
	Label       []string `yaml:"label"`
	PublishPort []string `yaml:"publish"`
}

// VolumeConfig represents volume-specific configuration
type VolumeConfig struct {
	Label []string `yaml:"label"`
}

// NetworkConfig represents network-specific configuration
type NetworkConfig struct {
	Label []string `yaml:"label"`
}

// PodConfig represents pod-specific configuration
type PodConfig struct {
	Label []string `yaml:"label"`
}

// KubeConfig represents Kubernetes manifest configuration
type KubeConfig struct {
	Path string `yaml:"path"`
}

// ImageConfig represents image-specific configuration
type ImageConfig struct {
	Image string `yaml:"image"`
}

// BuildConfig represents container build configuration
type BuildConfig struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}
