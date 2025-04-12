package unit

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

// Container represents the configuration for a container unit.
type Container struct {
	Image           string
	Label           []string
	PublishPort     []string
	Environment     map[string]string
	EnvironmentFile []string
	Volume          []string
	Network         []string
	Exec            []string
	Entrypoint      []string
	User            string
	Group           string
	WorkingDir      string
	RunInit         *bool
	Privileged      bool
	ReadOnly        bool
	SecurityLabel   []string
	HostName        string
	Secrets         []Secret

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

func (c *Container) FromComposeService(service types.ServiceConfig) *Container {
	c.Image = service.Image
	c.Label = append(c.Label, service.Labels.AsList()...)

	if len(service.Ports) > 0 {
		for _, port := range service.Ports {
			c.PublishPort = append(c.PublishPort, fmt.Sprintf("%s:%d", port.Published, port.Target))
		}
	}

	if service.Environment != nil {
		if c.Environment == nil {
			c.Environment = make(map[string]string)
		}
		for k, v := range service.Environment {
			if v != nil {
				c.Environment[k] = *v
			}
		}
	}

	if len(service.EnvFiles) > 0 {
		for _, envFile := range service.EnvFiles {
			c.EnvironmentFile = append(c.EnvironmentFile, envFile.Path)
		}
	}

	if len(service.Volumes) > 0 {
		for _, vol := range service.Volumes {
			c.Volume = append(c.Volume, fmt.Sprintf("%s:%s", vol.Source, vol.Target))
		}
	}

	if len(service.Networks) > 0 {
		for netName, net := range service.Networks {
			// If the network has aliases, use the first one
			if net != nil && len(net.Aliases) > 0 {
				c.Network = append(c.Network, net.Aliases[0])
			} else {
				// Otherwise, use the network name
				c.Network = append(c.Network, netName)
			}
		}
	}

	c.Exec = service.Command
	c.Entrypoint = service.Entrypoint
	c.User = service.User
	c.WorkingDir = service.WorkingDir
	// Handle the RunInit field - make sure it's not nil before assigning
	if service.Init != nil {
		c.RunInit = service.Init
	} else {
		// Set a default value for RunInit
		defaultInit := false
		c.RunInit = &defaultInit
	}
	c.Privileged = service.Privileged
	c.ReadOnly = service.ReadOnly
	c.SecurityLabel = append(c.SecurityLabel, service.SecurityOpt...)
	c.HostName = service.Hostname

	if len(service.Secrets) > 0 {
		for _, secret := range service.Secrets {
			unitSecret := Secret{
				Source: secret.Source,
				Target: secret.Target,
				UID:    secret.UID,
				GID:    secret.GID,
			}

			if secret.Mode == nil {
				defaultMode := types.FileMode(0644)
				unitSecret.Mode = defaultMode.String()
			} else {
				unitSecret.Mode = secret.Mode.String()
			}
			c.Secrets = append(c.Secrets, unitSecret)
		}
	}

	return c
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
	Source string
	Target string
	UID    string
	GID    string
	Mode   string
}
