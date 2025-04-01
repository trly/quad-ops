package unit

import "fmt"

// Container represents the configuration for a container in a Quadlet unit.
// It includes settings such as the container image, published ports, environment variables,
// volumes, networks, command, entrypoint, user, and other container-specific options.
type Container struct {
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
	Secrets         []Secret          `yaml:"secrets"`
	
	// Systemd unit properties
	Name     string
	UnitType string
}

// NewContainer creates a new Container with the given name
func NewContainer(name string) *Container {
	return &Container{
		Name:     name,
		UnitType: "container",
	}
}

// GetServiceName returns the full systemd service name
func (c *Container) GetServiceName() string {
	return c.Name + ".service"
}

// GetUnitType returns the type of the unit
func (c *Container) GetUnitType() string {
	return "container"
}

// GetUnitName returns the name of the unit
func (c *Container) GetUnitName() string {
	return c.Name
}

// GetStatus returns the current status of the unit
func (c *Container) GetStatus() (string, error) {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.GetStatus()
}

// Start starts the unit
func (c *Container) Start() error {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.Start()
}

// Stop stops the unit
func (c *Container) Stop() error {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.Stop()
}

// Restart restarts the unit
func (c *Container) Restart() error {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.Restart()
}

// Show displays the unit configuration and status
func (c *Container) Show() error {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.Show()
}

type Secret struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`   // mount or env
	Target string `yaml:"target"` // defaults to secret name
	UID    int    `yaml:"uid"`    // defaults to 0, mount type only
	GID    int    `yaml:"gid"`    // defaults to 0, mount type only
	Mode   string `yaml:"mode"`   // defaults to 0444, mount type only
}

func (sc Secret) Validate() error {
	if sc.Type != "mount" && sc.Type != "env" {
		return fmt.Errorf("invalid secret type: %s", sc.Type)
	}

	if sc.Type == "mount" && (sc.UID == 0 || sc.GID == 0 || sc.Mode == "") {
		return fmt.Errorf("missing required fields for mount secret: UID, GID, Mode")
	}

	if sc.Type == "env" && sc.Target == "" {
		return fmt.Errorf("missing required field for env secret: Target")
	}

	return nil
}
