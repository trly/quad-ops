// Package service provides platform-agnostic service domain models.
package service

import (
	"time"
)

// Spec represents a platform-agnostic service specification.
// It is the core domain model that gets converted from Docker Compose
// and rendered to platform-specific artifacts (systemd units, launchd plists, etc.).
type Spec struct {
	Name        string            // Service name (unique identifier)
	Description string            // Human-readable description
	Container   Container         // Container configuration
	Volumes     []Volume          // Volume mounts
	Networks    []Network         // Network attachments
	DependsOn   []string          // Service dependencies (service names)
	Annotations map[string]string // Platform-agnostic metadata
}

// Container represents container runtime configuration.
type Container struct {
	Image         string            // Container image (name:tag)
	Command       []string          // Override CMD
	Args          []string          // Additional arguments
	Env           map[string]string // Environment variables
	EnvFiles      []string          // Environment files to load
	WorkingDir    string            // Working directory
	User          string            // User to run as
	Group         string            // Group to run as
	Ports         []Port            // Port mappings
	Mounts        []Mount           // File/directory mounts
	Resources     Resources         // Resource constraints
	RestartPolicy RestartPolicy     // Restart behavior
	Healthcheck   *Healthcheck      // Health check configuration
	Security      Security          // Security settings
	Build         *Build            // Build configuration (if image needs building)
	Labels        map[string]string // Container labels
	Hostname      string            // Container hostname
	ContainerName string            // Explicit container name
	Entrypoint    []string          // Override ENTRYPOINT
	Init          bool              // Run init inside container
	ReadOnly      bool              // Read-only root filesystem
	Logging       Logging           // Logging configuration
	Secrets       []Secret          // Secrets to mount
	Network       NetworkMode       // Network mode configuration
	Tmpfs         []string          // Tmpfs mounts
	Ulimits       []Ulimit          // Ulimit settings
	Sysctls       map[string]string // Sysctl settings
	UserNS        string            // User namespace mode
	PodmanArgs    []string          // Additional Podman arguments
	PidsLimit     int64             // Maximum PIDs
}

// Port represents a port mapping.
type Port struct {
	Host      string // Host address (optional)
	HostPort  uint16 // Host port
	Container uint16 // Container port
	Protocol  string // "tcp" or "udp" (default: tcp)
}

// Mount represents a filesystem mount.
type Mount struct {
	Source      string            // Source path or volume name
	Target      string            // Container path
	Type        MountType         // "bind", "volume", "tmpfs"
	ReadOnly    bool              // Read-only mount
	Options     map[string]string // Mount options
	BindOptions *BindOptions      // Bind-specific options
}

// MountType represents the type of mount.
type MountType string

const (
	MountTypeBind   MountType = "bind"
	MountTypeVolume MountType = "volume"
	MountTypeTmpfs  MountType = "tmpfs"
)

// BindOptions represents bind mount options.
type BindOptions struct {
	Propagation string // "private", "shared", "slave", "rshared", "rslave"
}

// Resources represents resource constraints.
type Resources struct {
	Memory            string  // Memory limit (e.g., "512m", "2g")
	MemoryReservation string  // Memory soft limit
	MemorySwap        string  // Memory + swap limit
	CPUShares         int64   // CPU shares (relative weight)
	CPUQuota          int64   // CPU quota in microseconds
	CPUPeriod         int64   // CPU period in microseconds
	PidsLimit         int64   // Maximum PIDs
}

// RestartPolicy represents the container restart policy.
type RestartPolicy string

const (
	RestartPolicyNo        RestartPolicy = "no"
	RestartPolicyAlways    RestartPolicy = "always"
	RestartPolicyOnFailure RestartPolicy = "on-failure"
	RestartPolicyUnlessStopped RestartPolicy = "unless-stopped"
)

// Healthcheck represents a health check configuration.
type Healthcheck struct {
	Test          []string      // Health check command
	Interval      time.Duration // Check interval
	Timeout       time.Duration // Check timeout
	Retries       int           // Consecutive failures before unhealthy
	StartPeriod   time.Duration // Initialization grace period
	StartInterval time.Duration // Interval during start period
}

// Security represents security settings.
type Security struct {
	Privileged       bool     // Run with elevated privileges
	CapAdd           []string // Linux capabilities to add
	CapDrop          []string // Linux capabilities to drop
	SecurityOpt      []string // Security options
	ReadonlyRootfs   bool     // Read-only root filesystem
	SELinuxType      string   // SELinux type label
	AppArmorProfile  string   // AppArmor profile
	SeccompProfile   string   // Seccomp profile
}

// Build represents container build configuration.
type Build struct {
	Context               string            // Build context path
	Dockerfile            string            // Dockerfile path
	Target                string            // Build target
	Args                  map[string]string // Build arguments
	Labels                map[string]string // Image labels
	CacheFrom             []string          // Cache sources
	Pull                  bool              // Always pull base image
	Networks              []string          // Networks for build
	Volumes               []string          // Volumes for build
	Secrets               []string          // Secrets for build
	Tags                  []string          // Image tags
	Annotations           []string          // Image annotations
	SetWorkingDirectory   string            // Working directory for build
	PodmanArgs            []string          // Additional Podman build args
}

// Logging represents logging configuration.
type Logging struct {
	Driver  string            // Log driver (json-file, journald, etc.)
	Options map[string]string // Driver-specific options
}

// Secret represents a secret to mount in the container.
type Secret struct {
	Source string // Secret source identifier
	Target string // Target path in container (optional)
	UID    string // Owner UID (optional)
	GID    string // Owner GID (optional)
	Mode   string // File permissions (optional)
	Type   string // Secret type (optional)
}

// NetworkMode represents network configuration mode.
type NetworkMode struct {
	Mode    string   // "bridge", "host", "none", "container:<name>", "service:<name>"
	Aliases []string // Network aliases
}

// Ulimit represents a ulimit setting.
type Ulimit struct {
	Name string // Ulimit name
	Soft int64  // Soft limit
	Hard int64  // Hard limit
}

// Volume represents a named volume definition.
type Volume struct {
	Name     string            // Volume name
	Driver   string            // Volume driver (default: local)
	Options  map[string]string // Driver options
	Labels   map[string]string // Volume labels
	External bool              // External volume (not managed)
}

// Network represents a network definition.
type Network struct {
	Name     string            // Network name
	Driver   string            // Network driver (bridge, overlay, etc.)
	Options  map[string]string // Driver options
	Labels   map[string]string // Network labels
	IPAM     *IPAM             // IP address management
	Internal bool              // Internal network (no external access)
	IPv6     bool              // Enable IPv6
	External bool              // External network (not managed)
}

// IPAM represents IP address management configuration.
type IPAM struct {
	Driver  string       // IPAM driver
	Config  []IPAMConfig // IPAM configurations
	Options map[string]string // Driver options
}

// IPAMConfig represents a single IPAM configuration.
type IPAMConfig struct {
	Subnet  string // Subnet in CIDR format
	Gateway string // Gateway address
	IPRange string // IP range for allocation
}
