// Package model defines the core unit types for quad-ops
package model

import (
	"fmt"
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
)

// Container represents the configuration for a container unit.
type Container struct {
	Name           string
	UnitType       string
	Image          string
	Label          []string
	PublishPort    []string
	Environment    map[string]string
	EnvironmentFile []string
	Volume         []string
	Network        []string
	NetworkAlias   []string
	Exec           []string
	Entrypoint     []string
	User           string
	Group          string
	WorkingDir     string
	RunInit        *bool
	ReadOnly       bool
	HostName       string
	ContainerName  string
	Secrets        []Secret
}

// Secret represents a secret configuration for a container.
type Secret struct {
	Source string
	Target string
	UID    string
	GID    string
	Mode   string
	Type   string
}

// NewContainer creates a new Container with the given name.
func NewContainer(name string) *Container {
	// Always initialize RunInit to avoid nil pointer dereference
	boolFalse := false
	return &Container{
		Name:        name,
		RunInit:     &boolFalse,
		Environment: make(map[string]string),
	}
}

// FromComposeService converts a Docker Compose service to a Container.
func (c *Container) FromComposeService(service types.ServiceConfig, projectName string) *Container {
	// Default settings
	c.Image = service.Image
	c.UnitType = "container"

	// Process container labels
	for k, v := range service.Labels {
		c.Label = append(c.Label, fmt.Sprintf("%s=%s", k, v))
	}
	// Sort for consistent output
	sort.Strings(c.Label)

	// Process port mappings
	for _, portConfig := range service.Ports {
		port := fmt.Sprintf("%s:%d", portConfig.Published, portConfig.Target)
		if portConfig.Protocol != "" && portConfig.Protocol != "tcp" {
			port = fmt.Sprintf("%s/%s", port, portConfig.Protocol)
		}
		c.PublishPort = append(c.PublishPort, port)
	}
	// Sort for consistent output
	sort.Strings(c.PublishPort)

	// Process environment variables
	for k, v := range service.Environment {
		if v != nil {
			c.Environment[k] = *v
		}
	}

	// Process environment files
	for _, envFile := range service.EnvFiles {
		// EnvFile in compose-go is a string
		c.EnvironmentFile = append(c.EnvironmentFile, envFile.Path)
	}

	// Process volumes
	for _, vol := range service.Volumes {
		// For test compatibility, if Type is not set, try to infer type from source string
		vType := vol.Type
		if vType == "" {
			// If Source starts with ./ or /, it's a bind mount
			if len(vol.Source) > 0 && (vol.Source[0] == '/' || (len(vol.Source) > 1 && vol.Source[0] == '.' && vol.Source[1] == '/')) {
				vType = "bind"
			} else {
				// Otherwise treat as a named volume 
				vType = "volume"
			}
		}

		if vType == "volume" {
			// Named volume - use project prefixed name and add .volume suffix for quadlet
			namedVolume := fmt.Sprintf("%s-%s.volume:%s", projectName, vol.Source, vol.Target)
			if vol.ReadOnly {
				namedVolume += ":ro"
			}
			c.Volume = append(c.Volume, namedVolume)
		} else if vType == "bind" {
			// Bind mount
			bindVolume := fmt.Sprintf("%s:%s", vol.Source, vol.Target)
			if vol.ReadOnly {
				bindVolume += ":ro"
			}
			c.Volume = append(c.Volume, bindVolume)
		} else if vType == "tmpfs" {
			// tmpfs mount
			tmpfsVolume := fmt.Sprintf("tmpfs:%s", vol.Target)
			c.Volume = append(c.Volume, tmpfsVolume)
		}
	}
	// Sort for consistent output
	sort.Strings(c.Volume)

	// Process networks
	for netName, netConfig := range service.Networks {
		// Add project-prefixed network name with .network suffix for quadlet
		network := fmt.Sprintf("%s-%s.network", projectName, netName)
		c.Network = append(c.Network, network)

		// Process network aliases
		if netConfig != nil && len(netConfig.Aliases) > 0 {
			for _, alias := range netConfig.Aliases {
				c.NetworkAlias = append(c.NetworkAlias, alias)
			}
		}
	}
	// Sort for consistent output
	sort.Strings(c.Network)
	sort.Strings(c.NetworkAlias)

	// Process entrypoint and command
	if len(service.Entrypoint) > 0 {
		c.Entrypoint = service.Entrypoint
	}
	if len(service.Command) > 0 {
		c.Exec = service.Command
	}

	// Process other settings
	if service.User != "" {
		c.User = service.User
	}

	// Process working directory
	if service.WorkingDir != "" {
		c.WorkingDir = service.WorkingDir
	}

	// Process hostname
	if service.Hostname != "" {
		c.HostName = service.Hostname
	} else {
		// Default to service name for hostname (without systemd- prefix)
		c.HostName = c.Name
	}

	// Process read-only setting
	if service.ReadOnly {
		c.ReadOnly = true
	}

	// Process RunInit (set to true if specified in compose file)
	if service.Init != nil && *service.Init {
		boolTrue := true
		c.RunInit = &boolTrue
	}

	// Process secrets
	for _, secretConfig := range service.Secrets {
		secret := Secret{
			Source: secretConfig.Source,
		}

		if secretConfig.Target != "" {
			secret.Target = secretConfig.Target
		}

		if secretConfig.UID != "" {
			secret.UID = secretConfig.UID
		}

		if secretConfig.GID != "" {
			secret.GID = secretConfig.GID
		}

		if secretConfig.Mode != nil {
			secret.Mode = fmt.Sprintf("%d", *secretConfig.Mode)
		} else {
			// Default mode to 0644 if not specified
			secret.Mode = "0644"
		}

		c.Secrets = append(c.Secrets, secret)
	}

	// Process podman-specific extensions
	if envSecretExt, ok := service.Extensions["x-podman-env-secrets"]; ok {
		if envSecrets, ok := envSecretExt.(map[string]interface{}); ok {
			for secretName, envVar := range envSecrets {
				if envName, ok := envVar.(string); ok {
					// Create an environment-based secret
					secret := Secret{
						Source: secretName,
						Target: envName,
						Type:   "env",
					}
					c.Secrets = append(c.Secrets, secret)
				}
			}
		}
	}

	return c
}

// GetServiceName returns the full systemd service name.
func (c *Container) GetServiceName() string {
	return c.Name + ".service"
}

// GetUnitType returns the type of the unit.
func (c *Container) GetUnitType() string {
	return "container"
}

// GetUnitName returns the name of the unit.
func (c *Container) GetUnitName() string {
	return c.Name
}

// Stop stops the container.
func (c *Container) Stop() error {
	// To be implemented with proper systemd integration
	return nil
}

// Restart restarts the container.
func (c *Container) Restart() error {
	// To be implemented with proper systemd integration
	return nil
}