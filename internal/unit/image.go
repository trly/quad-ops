package unit

// ImageConfig represents the configuration for an image in a Quadlet unit.
type ImageConfig struct {
	Image      string   `yaml:"image"`
	PodmanArgs []string `yaml:"podman_args"`
}
