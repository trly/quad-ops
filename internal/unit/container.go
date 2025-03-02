package unit

// ContainerConfig represents the configuration for a container in a Quadlet unit.
// It includes settings such as the container image, published ports, environment variables,
// volumes, networks, command, entrypoint, user, and other container-specific options.
type ContainerConfig struct {
	Image           string            `yaml:"image"`
	Label           []string          `yaml:"label"`
	PublishPort     []string          `yaml:"publish"`
	Environment     map[string]string `yaml:"environment"`
	EnvironmentFile string            `yaml:"environment_file"`
	Volume          []string          `yaml:"volume"`
	Network         []string          `yaml:"network"`
	Command         []string          `yaml:"command"`
	Entrypoint      []string          `yaml:"entrypoint"`
	User            string            `yaml:"user"`
	Group           string            `yaml:"group"`
	WorkingDir      string            `yaml:"working_dir"`
	PodmanArgs      []string          `yaml:"podman_args"`
	RunInit         bool              `yaml:"run_init"`
	Notify          bool              `yaml:"notify"`
	Privileged      bool              `yaml:"privileged"`
	ReadOnly        bool              `yaml:"read_only"`
	SecurityLabel   []string          `yaml:"security_label"`
	HostName        string            `yaml:"hostname"`
	Secrets         []SecretConfig    `yaml:"secrets"`
}

type SecretConfig struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`   // mount or env
	Target string `yaml:"target"` // defaults to secret name
	UID    int    `yaml:"uid"`    // defaults to 0, mount type only
	GID    int    `yaml:"gid"`    // defaults to 0, mount type only
	Mode   string `yaml:"mode"`   // defaults to 0444, mount type only
}
