package quadlet

// QuadletUnit represents the configuration for a Quadlet unit, which can include
// systemd, container, volume, network, pod, Kubernetes, image, and build settings.
type QuadletUnit struct {
	Name      string          `yaml:"name"`
	Type      string          `yaml:"type"`
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
	Documentation   string   `yaml:"documentation"`
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

type VolumeConfig struct {
	Label    []string `yaml:"label"`
	Device   string   `yaml:"device"`
	Options  []string `yaml:"options"`
	UID      int      `yaml:"uid"`
	GID      int      `yaml:"gid"`
	Mode     string   `yaml:"mode"`
	Chown    bool     `yaml:"chown"`
	Selinux  bool     `yaml:"selinux"`
	Copy     bool     `yaml:"copy"`
	Group    string   `yaml:"group"`
	Size     string   `yaml:"size"`
	Capacity string   `yaml:"capacity"`
	Type     string   `yaml:"type"`
}

type NetworkConfig struct {
	Label      []string `yaml:"label"`
	Driver     string   `yaml:"driver"`
	Gateway    string   `yaml:"gateway"`
	IPRange    string   `yaml:"ip_range"`
	Subnet     string   `yaml:"subnet"`
	IPv6       bool     `yaml:"ipv6"`
	Internal   bool     `yaml:"internal"`
	DNSEnabled bool     `yaml:"dns_enabled"`
	Options    []string `yaml:"options"`
}

type ImageConfig struct {
	Image      string   `yaml:"image"`
	PodmanArgs []string `yaml:"podman_args"`
}

type SecretConfig struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`   // mount or env
	Target string `yaml:"target"` // defaults to secret name
	UID    int    `yaml:"uid"`    // defaults to 0, mount type only
	GID    int    `yaml:"gid"`    // defaults to 0, mount type only
	Mode   string `yaml:"mode"`   // defaults to 0444, mount type only
}
