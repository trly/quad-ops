// Package platform provides platform abstraction interfaces for cross-platform service management.
package platform

import (
	"context"
	"io/fs"

	"github.com/trly/quad-ops/internal/service"
)

// Artifact represents a platform-specific artifact (unit file, plist, etc.).
type Artifact struct {
	Path    string      // Relative path where artifact should be written
	Content []byte      // Artifact content
	Mode    fs.FileMode // File permissions
	Hash    string      // Content hash (for change detection)
}

// RenderResult contains artifacts plus metadata for change detection.
type RenderResult struct {
	Artifacts      []Artifact              // Generated artifacts
	ServiceChanges map[string]ChangeStatus // Per-service change status
}

// ChangeStatus indicates whether a service's artifacts changed.
type ChangeStatus struct {
	Changed       bool     // Whether artifacts changed
	ArtifactPaths []string // Paths to this service's artifacts
	ContentHash   string   // Combined hash of all artifacts
}

// Renderer converts platform-agnostic service specs to platform-specific artifacts.
type Renderer interface {
	// Name returns the platform name (e.g., "systemd", "launchd").
	Name() string

	// Render converts service specs to platform-specific artifacts.
	// Returns artifacts and per-service change metadata.
	Render(ctx context.Context, specs []service.Spec) (*RenderResult, error)
}

// ServiceStatus represents the status of a service.
type ServiceStatus struct {
	Name        string // Service name
	Active      bool   // Whether service is active
	State       string // Platform-specific state (e.g., "running", "stopped")
	SubState    string // Platform-specific sub-state (optional)
	Description string // Human-readable status description
	PID         int    // Process ID (0 if not running)
	Since       string // Time service started (empty if not running)
	Error       string // Error message if failed (optional)
}

// Lifecycle manages the lifecycle of platform services.
type Lifecycle interface {
	// Name returns the platform name.
	Name() string

	// Reload reloads the service manager configuration (e.g., systemctl daemon-reload).
	Reload(ctx context.Context) error

	// Start starts a service.
	Start(ctx context.Context, name string) error

	// Stop stops a service.
	Stop(ctx context.Context, name string) error

	// Restart restarts a service.
	Restart(ctx context.Context, name string) error

	// Status returns the status of a service.
	Status(ctx context.Context, name string) (*ServiceStatus, error)

	// StartMany starts multiple services in dependency order.
	// Returns a map of service name to error (nil if successful).
	StartMany(ctx context.Context, names []string) map[string]error

	// StopMany stops multiple services in reverse dependency order.
	// Returns a map of service name to error (nil if successful).
	StopMany(ctx context.Context, names []string) map[string]error

	// RestartMany restarts multiple services in dependency order.
	// Returns a map of service name to error (nil if successful).
	RestartMany(ctx context.Context, names []string) map[string]error
}

// Platform combines Renderer and Lifecycle for a complete platform adapter.
type Platform interface {
	Renderer
	Lifecycle
}
