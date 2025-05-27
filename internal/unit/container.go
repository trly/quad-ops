package unit

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/util"
)

// Container represents the configuration for a container unit.
type Container struct {
	BaseUnit    // Embed the base struct
	Image       string
	Label       []string
	PublishPort []string
	Environment map[string]string
	// Environment file paths (will be sorted for deterministic output)
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
	// PodmanArgs contains direct podman run arguments for features not supported by Quadlet
	PodmanArgs []string
	// Health check settings
	HealthCmd           []string
	HealthInterval      string
	HealthTimeout       string
	HealthRetries       int
	HealthStartPeriod   string
	HealthStartInterval string

	// Resource constraints
	Memory            string
	MemoryReservation string
	MemorySwap        string
	CPUShares         int64
	CPUQuota          int64
	CPUPeriod         int64
	PidsLimit         int64

	// Advanced container configuration
	Ulimit           []string
	Sysctl           map[string]string
	sortedSysctlKeys []string
	Tmpfs            []string
	UserNS           string

	// Logging and monitoring configuration
	LogDriver        string
	LogOpt           map[string]string
	sortedLogOptKeys []string
	RestartPolicy    string
}

// NewContainer creates a new Container with the given name.
func NewContainer(name string) *Container {
	return &Container{
		BaseUnit: *NewBaseUnit(name, "container"),
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

	// Process resource constraints
	c.processServiceResources(service)

	// Process advanced container configuration
	c.processAdvancedConfig(service)

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

	// Process command - handle special case for multi-part commands with variables
	if len(service.Command) > 0 {
		// If there are multiple command parts that may include variables or special characters
		// join them into a single string to preserve the exact command structure
		commandStr := strings.Join(service.Command, " ")
		if strings.Contains(commandStr, "$") || strings.Contains(commandStr, "-c") || strings.Contains(commandStr, ", ") {
			// For commands with variables, shell flags, or comma-separated lists
			// Properly handle embedded quotes and special characters
			processedCmd := strings.ReplaceAll(commandStr, "\"$user\",", "$user,")
			processedCmd = strings.ReplaceAll(processedCmd, ", ", ",")
			c.Exec = []string{processedCmd}
		} else {
			c.Exec = service.Command
		}
	}

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
			c.PublishPort = append(c.PublishPort, port.Published+":"+strconv.Itoa(int(port.Target)))
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
			// In Docker Compose, the first element is typically the command type (NONE, CMD, CMD-SHELL)
			// For complex multi-part commands, we need to ensure the entire command is preserved
			if len(service.HealthCheck.Test) >= 2 && (service.HealthCheck.Test[0] == "CMD" || service.HealthCheck.Test[0] == "CMD-SHELL") {
				// Join all parts after the command type into a single string to preserve semicolons
				// Escape environment variables in health check commands to prevent premature expansion
				cmdStr := strings.Join(service.HealthCheck.Test[1:], " ")
				// For health check commands with environment variables, we need to handle them specially
				// The variables need to be escaped for systemd, but available in the container
				// We'll replace environment variable references with their literal values from the environment definition
				varRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
				cmdStr = varRegex.ReplaceAllStringFunc(cmdStr, func(match string) string {
					// Extract the variable name (without ${})
					varName := varRegex.FindStringSubmatch(match)[1]

					// Look up the variable in the container environment
					if val, exists := service.Environment[varName]; exists && val != nil {
						// Use the actual value from environment
						return *val
					}

					// If not found in environment, keep original reference but escape for systemd
					return "\\" + match
				})
				c.HealthCmd = []string{service.HealthCheck.Test[0], cmdStr}
			} else {
				c.HealthCmd = service.HealthCheck.Test
			}
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

// processServiceResources processes resource constraints from a Docker Compose service.
func (c *Container) processServiceResources(service types.ServiceConfig) {
	unsupportedFeatures := make([]string, 0, 10)

	c.processMemoryConstraints(service, &unsupportedFeatures)
	c.processCPUConstraints(service, &unsupportedFeatures)
	c.processSecurityOptions(service, &unsupportedFeatures)

	c.logUnsupportedFeatures(service.Name, unsupportedFeatures)
}

func (c *Container) processMemoryConstraints(service types.ServiceConfig, unsupportedFeatures *[]string) {
	// Handle service-level memory constraints
	if service.MemLimit != 0 {
		c.Memory = strconv.FormatInt(int64(service.MemLimit), 10)
		*unsupportedFeatures = append(*unsupportedFeatures, "Memory limits (mem_limit)")
		c.PodmanArgs = append(c.PodmanArgs, "--memory="+strconv.FormatInt(int64(service.MemLimit), 10))
	}

	if service.MemReservation != 0 {
		c.MemoryReservation = strconv.FormatInt(int64(service.MemReservation), 10)
		*unsupportedFeatures = append(*unsupportedFeatures, "Memory reservation (memory_reservation)")
		c.PodmanArgs = append(c.PodmanArgs, "--memory-reservation="+strconv.FormatInt(int64(service.MemReservation), 10))
	}

	if service.MemSwapLimit != 0 {
		c.MemorySwap = strconv.FormatInt(int64(service.MemSwapLimit), 10)
		*unsupportedFeatures = append(*unsupportedFeatures, "Memory swap (memswap_limit)")
		c.PodmanArgs = append(c.PodmanArgs, "--memory-swap="+strconv.FormatInt(int64(service.MemSwapLimit), 10))
	}

	// Handle deploy section memory constraints
	if service.Deploy != nil {
		if service.Deploy.Resources.Limits != nil && service.Deploy.Resources.Limits.MemoryBytes != 0 {
			c.Memory = strconv.FormatInt(int64(service.Deploy.Resources.Limits.MemoryBytes), 10)
			*unsupportedFeatures = append(*unsupportedFeatures, "Memory limits (deploy.resources.limits.memory)")
			c.PodmanArgs = append(c.PodmanArgs, "--memory="+strconv.FormatInt(int64(service.Deploy.Resources.Limits.MemoryBytes), 10))
		}

		if service.Deploy.Resources.Reservations != nil && service.Deploy.Resources.Reservations.MemoryBytes != 0 {
			c.MemoryReservation = strconv.FormatInt(int64(service.Deploy.Resources.Reservations.MemoryBytes), 10)
			*unsupportedFeatures = append(*unsupportedFeatures, "Memory reservation (deploy.resources.reservations.memory)")
			c.PodmanArgs = append(c.PodmanArgs, "--memory-reservation="+strconv.FormatInt(int64(service.Deploy.Resources.Reservations.MemoryBytes), 10))
		}
	}
}

func (c *Container) processCPUConstraints(service types.ServiceConfig, unsupportedFeatures *[]string) {
	// Set default CPU period for quota calculations
	var cpuPeriod int64 = 100000 // Default period in microseconds

	// Handle service-level CPU constraints
	if service.CPUPeriod != 0 {
		cpuPeriod = service.CPUPeriod
		*unsupportedFeatures = append(*unsupportedFeatures, "CPU period (cpu_period)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--cpu-period=%d", service.CPUPeriod))
	}
	c.CPUPeriod = cpuPeriod

	if service.CPUQuota != 0 {
		c.CPUQuota = service.CPUQuota
		*unsupportedFeatures = append(*unsupportedFeatures, "CPU quota (cpu_quota)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--cpu-quota=%d", service.CPUQuota))
	} else if service.CPUS != 0 {
		c.CPUQuota = int64(float64(service.CPUS) * float64(cpuPeriod))
		*unsupportedFeatures = append(*unsupportedFeatures, "CPU cores (cpus)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--cpus=%.2f", service.CPUS))
	}

	if service.CPUShares != 0 {
		c.CPUShares = service.CPUShares
		*unsupportedFeatures = append(*unsupportedFeatures, "CPU shares (cpu_shares)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--cpu-shares=%d", service.CPUShares))
	}

	// Handle deploy section CPU constraints
	if service.Deploy != nil && service.Deploy.Resources.Limits != nil && service.Deploy.Resources.Limits.NanoCPUs != 0 {
		if c.CPUQuota == 0 {
			c.CPUQuota = int64(float64(service.Deploy.Resources.Limits.NanoCPUs) * float64(cpuPeriod) / 1e9)
			*unsupportedFeatures = append(*unsupportedFeatures, "CPU limits (deploy.resources.limits.cpus)")
			cpus := float64(service.Deploy.Resources.Limits.NanoCPUs) / 1e9
			c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--cpus=%.2f", cpus))
		}

		if c.CPUShares == 0 {
			c.CPUShares = int64(float64(service.Deploy.Resources.Limits.NanoCPUs) / 1e9 * 1024)
		}
	}

	// Process limit
	if service.PidsLimit != 0 {
		c.PidsLimit = service.PidsLimit
	}
}

func (c *Container) processSecurityOptions(service types.ServiceConfig, unsupportedFeatures *[]string) {
	// Handle Privileged mode
	if service.Privileged {
		*unsupportedFeatures = append(*unsupportedFeatures, "Privileged mode")
		c.PodmanArgs = append(c.PodmanArgs, "--privileged")
	}

	// SecurityLabel handling
	if len(service.SecurityOpt) > 0 {
		for _, opt := range service.SecurityOpt {
			if opt == "label:disable" {
				*unsupportedFeatures = append(*unsupportedFeatures, "Security label disable")
				c.PodmanArgs = append(c.PodmanArgs, "--security-opt=label=disable")
			} else if strings.HasPrefix(opt, "label:") {
				*unsupportedFeatures = append(*unsupportedFeatures, "Security label options")
				c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--security-opt=%s", opt))
			}
		}
	}
}

func (c *Container) logUnsupportedFeatures(serviceName string, unsupportedFeatures []string) {
	if len(unsupportedFeatures) > 0 {
		for _, feature := range unsupportedFeatures {
			log.GetLogger().Warn(fmt.Sprintf("Service '%s' uses %s which is not directly supported by Podman Quadlet. Using PodmanArgs directive instead.", serviceName, feature))
		}
	}
}

// processAdvancedConfig processes advanced container configuration from a Docker Compose service.
func (c *Container) processAdvancedConfig(service types.ServiceConfig) {
	// Track unsupported features to warn about
	unsupportedFeatures := make([]string, 0, 15)

	// Process standard container configuration (directly supported by Quadlet)
	c.processStandardConfig(service)

	// Process features that need PodmanArgs (not directly supported by Quadlet)
	c.processCapabilities(service, &unsupportedFeatures)
	c.processDevices(service, &unsupportedFeatures)
	c.processDNSSettings(service, &unsupportedFeatures)
	c.processNamespaceSettings(service, &unsupportedFeatures)
	c.processResourceTuning(service, &unsupportedFeatures)
	c.processNetworkConfig(service, &unsupportedFeatures)
	c.processContainerRuntime(service, &unsupportedFeatures)

	// Log warnings for unsupported features but indicate we're handling them via PodmanArgs
	if len(unsupportedFeatures) > 0 {
		for _, feature := range unsupportedFeatures {
			log.GetLogger().Warn(fmt.Sprintf("Service '%s' uses %s which is not directly supported by Podman Quadlet. Using PodmanArgs directive instead.", service.Name, feature))
		}
	}

	// Process logging configuration
	c.processLoggingConfig(service)

	// Process restart policy
	c.processRestartPolicy(service)
}

// processStandardConfig processes standard configuration supported directly by Quadlet.
func (c *Container) processStandardConfig(service types.ServiceConfig) {
	// Process ulimits
	if len(service.Ulimits) > 0 {
		for name, ulimit := range service.Ulimits {
			if ulimit.Hard == ulimit.Soft {
				// Single value format
				c.Ulimit = append(c.Ulimit, fmt.Sprintf("%s=%d", name, ulimit.Soft))
			} else {
				// Soft:hard format
				c.Ulimit = append(c.Ulimit, fmt.Sprintf("%s=%d:%d", name, ulimit.Soft, ulimit.Hard))
			}
		}
	}

	// Process sysctls
	if len(service.Sysctls) > 0 {
		if c.Sysctl == nil {
			c.Sysctl = make(map[string]string)
		}
		for k, v := range service.Sysctls {
			c.Sysctl[k] = v
		}
	}

	// Process tmpfs
	if len(service.Tmpfs) > 0 {
		for _, tmpfs := range service.Tmpfs {
			c.Tmpfs = append(c.Tmpfs, tmpfs)
		}
	}

	// Process user namespace mode
	if service.UserNSMode != "" {
		c.UserNS = service.UserNSMode
	}
}

// processCapabilities handles Linux capabilities configuration.
func (c *Container) processCapabilities(service types.ServiceConfig, unsupportedFeatures *[]string) {
	// Process capabilities
	if len(service.CapAdd) > 0 {
		*unsupportedFeatures = append(*unsupportedFeatures, "Capability add (cap_add)")
		for _, cap := range service.CapAdd {
			c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--cap-add=%s", cap))
		}
	}

	if len(service.CapDrop) > 0 {
		*unsupportedFeatures = append(*unsupportedFeatures, "Capability drop (cap_drop)")
		for _, cap := range service.CapDrop {
			c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--cap-drop=%s", cap))
		}
	}
}

// processDevices handles device mapping settings.
func (c *Container) processDevices(service types.ServiceConfig, unsupportedFeatures *[]string) {
	if len(service.Devices) > 0 {
		*unsupportedFeatures = append(*unsupportedFeatures, "Device mappings (devices)")
		for _, device := range service.Devices {
			// Extract the source path or use the full path if it's a string shorthand
			var devicePath string
			if device.Source != "" {
				devicePath = device.Source
			} else {
				// For shorthand string format in compose files (like "/dev/sda")
				devicePath = fmt.Sprintf("%v", device)
				// If devicePath contains curly braces, extract the path
				if strings.HasPrefix(devicePath, "{/") {
					parts := strings.Split(devicePath, " ")
					if len(parts) > 0 {
						// Extract path without leading { character
						devicePath = strings.TrimPrefix(parts[0], "{")
					}
				}
			}
			c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--device=%s", devicePath))
		}
	}
}

// processDNSSettings handles DNS configuration options.
func (c *Container) processDNSSettings(service types.ServiceConfig, unsupportedFeatures *[]string) {
	if len(service.DNS) > 0 {
		*unsupportedFeatures = append(*unsupportedFeatures, "DNS servers (dns)")
		for _, dns := range service.DNS {
			c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--dns=%s", dns))
		}
	}

	if len(service.DNSSearch) > 0 {
		*unsupportedFeatures = append(*unsupportedFeatures, "DNS search domains (dns_search)")
		for _, dnsSearch := range service.DNSSearch {
			c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--dns-search=%s", dnsSearch))
		}
	}

	if len(service.DNSOpts) > 0 {
		*unsupportedFeatures = append(*unsupportedFeatures, "DNS options (dns_opt)")
		for _, dnsOpt := range service.DNSOpts {
			c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--dns-opt=%s", dnsOpt))
		}
	}
}

// processNamespaceSettings handles IPC and PID namespace configuration.
func (c *Container) processNamespaceSettings(service types.ServiceConfig, unsupportedFeatures *[]string) {
	// Process IPC mode
	if service.Ipc != "" {
		*unsupportedFeatures = append(*unsupportedFeatures, "IPC mode (ipc)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--ipc=%s", service.Ipc))
	}

	// Process PID mode
	if service.Pid != "" {
		*unsupportedFeatures = append(*unsupportedFeatures, "PID mode (pid)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--pid=%s", service.Pid))
	}
}

// processResourceTuning handles resource tuning options like shared memory and cgroups.
func (c *Container) processResourceTuning(service types.ServiceConfig, unsupportedFeatures *[]string) {
	// Process SHM size
	if service.ShmSize != 0 {
		*unsupportedFeatures = append(*unsupportedFeatures, "Shared memory size (shm_size)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--shm-size=%d", service.ShmSize))
	}

	// Process cgroup parent
	if service.CgroupParent != "" {
		*unsupportedFeatures = append(*unsupportedFeatures, "Cgroup parent (cgroup_parent)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--cgroup-parent=%s", service.CgroupParent))
	}

	// Process storage options
	if len(service.StorageOpt) > 0 {
		*unsupportedFeatures = append(*unsupportedFeatures, "Storage options (storage_opt)")
		for k, v := range service.StorageOpt {
			c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--storage-opt=%s=%s", k, v))
		}
	}
}

// processNetworkConfig handles network-related configuration like MAC address.
func (c *Container) processNetworkConfig(service types.ServiceConfig, unsupportedFeatures *[]string) {
	// Process MAC address
	if service.MacAddress != "" {
		*unsupportedFeatures = append(*unsupportedFeatures, "MAC address (mac_address)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--mac-address=%s", service.MacAddress))
	}
}

// processContainerRuntime handles container runtime settings.
func (c *Container) processContainerRuntime(service types.ServiceConfig, unsupportedFeatures *[]string) {
	// Process runtime
	if service.Runtime != "" {
		*unsupportedFeatures = append(*unsupportedFeatures, "Runtime (runtime)")
		c.PodmanArgs = append(c.PodmanArgs, fmt.Sprintf("--runtime=%s", service.Runtime))
	}
}

// processLoggingConfig processes logging configuration from a Docker Compose service.
func (c *Container) processLoggingConfig(service types.ServiceConfig) {
	// Handle logging driver
	if service.LogDriver != "" {
		c.LogDriver = service.LogDriver
	}

	// Handle logging options
	if len(service.LogOpt) > 0 {
		if c.LogOpt == nil {
			c.LogOpt = make(map[string]string)
		}
		for k, v := range service.LogOpt {
			c.LogOpt[k] = v
		}
	}
}

// processRestartPolicy processes restart policy from a Docker Compose service.
func (c *Container) processRestartPolicy(service types.ServiceConfig) {
	// Map Docker Compose restart policy to systemd equivalent
	switch service.Restart {
	case "no":
		c.RestartPolicy = "no"
	case "always":
		c.RestartPolicy = "always"
	case "on-failure":
		c.RestartPolicy = "on-failure"
	case "unless-stopped":
		// unless-stopped doesn't have an exact systemd equivalent
		// 'always' is the closest match as it will restart unless explicitly stopped
		c.RestartPolicy = "always"
	default:
		// Use systemd default which is 'no'
		c.RestartPolicy = "no"
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
	// Environment variables will be sorted on-demand during unit file generation

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

	// Sort PodmanArgs for deterministic output
	if len(container.PodmanArgs) > 0 {
		sort.Strings(container.PodmanArgs)
	}

	// Sort advanced configuration slices
	if len(container.Ulimit) > 0 {
		sort.Strings(container.Ulimit)
	}

	if len(container.Tmpfs) > 0 {
		sort.Strings(container.Tmpfs)
	}

	// Sort sysctls keys for deterministic output
	if len(container.Sysctl) > 0 {
		container.sortedSysctlKeys = util.GetSortedMapKeys(container.Sysctl)
	}

	// Sort LogOpt keys for deterministic output
	if len(container.LogOpt) > 0 {
		container.sortedLogOptKeys = util.GetSortedMapKeys(container.LogOpt)
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
