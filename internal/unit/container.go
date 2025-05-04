package unit

import (
	"fmt"
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/util"
)

// Container represents the configuration for a container unit.
type Container struct {
	Image       string
	Label       []string
	PublishPort []string
	Environment map[string]string
	// Stores environment keys in sorted order for deterministic output
	sortedEnvKeys   []string
	EnvironmentFile []string
	Volume          []string
	Network         []string
	NetworkAlias    []string
	Exec            []string
	Entrypoint      []string
	User            string
	Group           string
	WorkingDir      string
	RunInit         *bool
	// Privileged removed - not supported by podman-systemd
	ReadOnly bool
	// SecurityLabel removed - not supported by podman-systemd
	// Use SecurityLabelType, SecurityLabelLevel, etc. instead
	HostName      string
	ContainerName string
	Secrets       []Secret
	// Health check settings
	HealthCmd           []string
	HealthInterval      string
	HealthTimeout       string
	HealthRetries       int
	HealthStartPeriod   string
	HealthStartInterval string

	// Systemd unit properties
	Name     string
	UnitType string
}

// NewContainer creates a new Container with the given name.
func NewContainer(name string) *Container {
	return &Container{
		Name:     name,
		UnitType: "container",
	}
}

// FromComposeService converts a Docker Compose service to a Podman Quadlet container configuration.
func (c *Container) FromComposeService(service types.ServiceConfig, projectName string) *Container {
	// Initialize RunInit to avoid nil pointer dereference
	c.RunInit = new(bool)
	*c.RunInit = true

	// Basic fields
	c.setBasicServiceFields(service)

	// Process ports
	c.processServicePorts(service)

	// Process environment variables and files
	c.processServiceEnvironment(service)

	// Process volumes
	c.processServiceVolumes(service, projectName)

	// Process networks
	c.processServiceNetworks(service, projectName)

	// Process health check configuration
	c.processServiceHealthCheck(service)

	// Process secrets
	c.processServiceSecrets(service)

	// Sort all container fields for deterministic output
	sortContainer(c)

	return c
}

// setBasicServiceFields sets simple fields directly from the service config.
func (c *Container) setBasicServiceFields(service types.ServiceConfig) {
	// No automatic image name conversion - use exactly what's provided in the compose file
	c.Image = service.Image
	c.Label = append(c.Label, service.Labels.AsList()...)
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

	// Privileged is not supported by podman-systemd
	c.ReadOnly = service.ReadOnly
	// SecurityLabel is not supported by podman-systemd
	c.HostName = service.Hostname
}

// processServicePorts converts service ports to container ports.
func (c *Container) processServicePorts(service types.ServiceConfig) {
	if len(service.Ports) > 0 {
		for _, port := range service.Ports {
			c.PublishPort = append(c.PublishPort, fmt.Sprintf("%s:%d", port.Published, port.Target))
		}
	}
}

// processServiceEnvironment handles environment variables and files.
func (c *Container) processServiceEnvironment(service types.ServiceConfig) {
	// Process environment variables
	if service.Environment != nil {
		if c.Environment == nil {
			c.Environment = make(map[string]string)
		}
		// Sort environment variables by key to ensure consistent output order
		keys := make([]string, 0, len(service.Environment))
		for k := range service.Environment {
			keys = append(keys, k)
		}
		// Sort the keys alphabetically
		sort.Strings(keys)

		// Use the sorted keys to get values
		for _, k := range keys {
			v := service.Environment[k]
			if v != nil {
				c.Environment[k] = *v
			}
		}
	}

	// Process environment files
	if len(service.EnvFiles) > 0 {
		for _, envFile := range service.EnvFiles {
			c.EnvironmentFile = append(c.EnvironmentFile, envFile.Path)
		}
	}
}

// processServiceVolumes handles volume mounts.
func (c *Container) processServiceVolumes(service types.ServiceConfig, projectName string) {
	if len(service.Volumes) > 0 {
		for _, vol := range service.Volumes {
			// Handle different volume types
			if vol.Type == "volume" {
				// Convert named volumes to Podman Quadlet format
				// This ensures proper systemd unit references for volumes defined in the compose file
				c.Volume = append(c.Volume, fmt.Sprintf("%s-%s.volume:%s", projectName, vol.Source, vol.Target))
			} else {
				// Regular bind mount or external volume - use as-is
				c.Volume = append(c.Volume, fmt.Sprintf("%s:%s", vol.Source, vol.Target))
			}
		}
	}
}

// processServiceNetworks handles network connections.
func (c *Container) processServiceNetworks(service types.ServiceConfig, projectName string) {
	if len(service.Networks) > 0 {
		for netName, net := range service.Networks {
			networkRef := ""

			// Check if network is a named network (project-defined) or a special network
			if netName != "host" && netName != "none" {
				// This is a project-defined network - format for Podman Quadlet with .network suffix
				networkRef = fmt.Sprintf("%s-%s.network", projectName, netName)
			} else if net != nil && len(net.Aliases) > 0 {
				// Network has aliases - use first alias
				networkRef = net.Aliases[0]
			} else {
				// Default or special network - use as is
				networkRef = netName
			}

			c.Network = append(c.Network, networkRef)

			// Add any network aliases specified in the compose file
			if net != nil && len(net.Aliases) > 0 {
				c.NetworkAlias = append(c.NetworkAlias, net.Aliases...)
			}
		}
	} else {
		// If no networks specified, create a default network using the project name
		// This ensures proper Quadlet format for the auto-generated network
		defaultNetworkRef := fmt.Sprintf("%s-default.network", projectName)
		c.Network = append(c.Network, defaultNetworkRef)
	}
}

// processServiceHealthCheck converts health check configuration.
func (c *Container) processServiceHealthCheck(service types.ServiceConfig) {
	if service.HealthCheck != nil && !service.HealthCheck.Disable {
		if len(service.HealthCheck.Test) > 0 {
			c.HealthCmd = service.HealthCheck.Test
		}
		if service.HealthCheck.Interval != nil {
			c.HealthInterval = service.HealthCheck.Interval.String()
		}
		if service.HealthCheck.Timeout != nil {
			c.HealthTimeout = service.HealthCheck.Timeout.String()
		}
		if service.HealthCheck.Retries != nil {
			// Using a safe conversion for healthcheck retries
			c.HealthRetries = convertUint64ToInt(*service.HealthCheck.Retries)
		}
		if service.HealthCheck.StartPeriod != nil {
			c.HealthStartPeriod = service.HealthCheck.StartPeriod.String()
		}
		if service.HealthCheck.StartInterval != nil {
			c.HealthStartInterval = service.HealthCheck.StartInterval.String()
		}
	}
}

// processServiceSecrets converts Docker Compose secrets to Podman Quadlet secrets.
func (c *Container) processServiceSecrets(service types.ServiceConfig) {
	// Process standard file-based Docker Compose secrets
	for _, secret := range service.Secrets {
		// Create file-based secret (standard Docker behavior)
		targetPath := secret.Target
		if targetPath == "" {
			// If no target is specified, use default path /run/secrets/<source>
			targetPath = "/run/secrets/" + secret.Source
		}
		unitSecret := Secret{
			Source: secret.Source,
			Target: targetPath,
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

	// Process environment variable secrets separately
	if ext, ok := service.Extensions["x-podman-env-secrets"]; ok {
		if envSecrets, isMap := ext.(map[string]interface{}); isMap {
			for secretName, envVar := range envSecrets {
				if envVarStr, isString := envVar.(string); isString {
					// Create env-based secret
					envSecret := Secret{
						Source: secretName,
						Type:   "env",
						Target: envVarStr, // Target becomes the environment variable name
					}
					c.Secrets = append(c.Secrets, envSecret)
				}
			}
		}
	}
}

// convertUint64ToInt safely converts uint64 to int, preventing overflow.
func convertUint64ToInt(val uint64) int {
	const maxInt = int(^uint(0) >> 1) // Maximum int value
	if val > uint64(maxInt) {
		return maxInt
	}
	return int(val)
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

// GetStatus returns the current status of the unit.
func (c *Container) GetStatus() (string, error) {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.GetStatus()
}

// Start starts the unit.
func (c *Container) Start() error {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.Start()
}

// Stop stops the unit.
func (c *Container) Stop() error {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.Stop()
}

// Restart restarts the unit.
func (c *Container) Restart() error {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.Restart()
}

// Show displays the unit configuration and status.
func (c *Container) Show() error {
	base := BaseSystemdUnit{Name: c.Name, Type: "container"}
	return base.Show()
}

// Secret represents a container secret definition.
type Secret struct {
	Source string
	Target string
	UID    string
	GID    string
	Mode   string
	Type   string
}

// sortContainer ensures all slices in a container config are sorted deterministically in-place.
// This is called when the container is created to ensure all data structures are immediately sorted.
func sortContainer(container *Container) {
	// Sort environment variables (already sorted in FromComposeService, but ensure it's done everywhere)
	if len(container.Environment) > 0 {
		// Create a sorted list of environment keys for deterministic unit generation
		// Note: This doesn't change the map, just ensures deterministic unit file generation
		container.sortedEnvKeys = util.GetSortedMapKeys(container.Environment)
	}

	// Sort all slices for deterministic output
	if len(container.Label) > 0 {
		sort.Strings(container.Label)
	}

	if len(container.PublishPort) > 0 {
		sort.Strings(container.PublishPort)
	}

	if len(container.EnvironmentFile) > 0 {
		sort.Strings(container.EnvironmentFile)
	}

	if len(container.Volume) > 0 {
		sort.Strings(container.Volume)
	}

	if len(container.Network) > 0 {
		sort.Strings(container.Network)
	}

	if len(container.NetworkAlias) > 0 {
		sort.Strings(container.NetworkAlias)
	}

	if len(container.Exec) > 0 {
		sort.Strings(container.Exec)
	}

	if len(container.Entrypoint) > 0 {
		sort.Strings(container.Entrypoint)
	}

	// Sort HealthCmd if present
	if len(container.HealthCmd) > 0 {
		sort.Strings(container.HealthCmd)
	}

	// Sort secrets by source
	sort.Slice(container.Secrets, func(i, j int) bool {
		// Primary sort by Source
		if container.Secrets[i].Source != container.Secrets[j].Source {
			return container.Secrets[i].Source < container.Secrets[j].Source
		}
		// Secondary sort by Target (if Sources are equal)
		if container.Secrets[i].Target != container.Secrets[j].Target {
			return container.Secrets[i].Target < container.Secrets[j].Target
		}
		// Final sort by Type
		return container.Secrets[i].Type < container.Secrets[j].Type
	})
}
