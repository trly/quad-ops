// Package launchd provides macOS launchd platform adapter for quad-ops.
package launchd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Domain represents launchd domain type.
type Domain string

const (
	// DomainUser represents user-level LaunchAgents.
	DomainUser Domain = "user"
	// DomainSystem represents system-level LaunchDaemons.
	DomainSystem Domain = "system"
)

// Options configures the launchd platform adapter.
type Options struct {
	// Domain specifies whether to use user (LaunchAgents) or system (LaunchDaemons).
	// Default: user
	Domain Domain

	// PodmanPath is the absolute path to the podman binary.
	// If empty, will be resolved from PATH or common locations.
	PodmanPath string

	// LabelPrefix is the prefix for launchd labels (e.g., "com.github.trly").
	// Default: "com.github.trly"
	LabelPrefix string

	// PlistDir is the directory where plist files will be written.
	// Default: ~/Library/LaunchAgents (user) or /Library/LaunchDaemons (system)
	PlistDir string

	// LogsDir is the directory where service logs will be written.
	// Default: ~/Library/Logs/quad-ops (user) or /var/log/quad-ops (system)
	LogsDir string

	// UID is the user ID for user domain operations.
	// Default: current user's UID
	UID int

	// UseSudo indicates whether to use sudo for system domain operations.
	// Default: false (will be set to true automatically for system domain if not root)
	UseSudo bool
}

// DefaultOptions returns default launchd options for the current user.
func DefaultOptions() Options {
	homeDir, _ := os.UserHomeDir()
	uid := os.Getuid()

	return Options{
		Domain:      DomainUser,
		LabelPrefix: "dev.trly.quad-ops",
		PlistDir:    filepath.Join(homeDir, "Library", "LaunchAgents"),
		LogsDir:     filepath.Join(homeDir, "Library", "Logs", "quad-ops"),
		UID:         uid,
		UseSudo:     false,
	}
}

// OptionsFromSettings creates launchd options from configuration settings.
// Respects user overrides while providing sensible defaults.
func OptionsFromSettings(_, quadletDir string, userMode bool) Options {
	homeDir, _ := os.UserHomeDir()
	uid := os.Getuid()

	domain := DomainUser
	if !userMode {
		domain = DomainSystem
	}

	// Use config values if provided, otherwise use defaults
	plistDir := quadletDir
	if plistDir == "" {
		if domain == DomainUser {
			plistDir = filepath.Join(homeDir, "Library", "LaunchAgents")
		} else {
			plistDir = "/Library/LaunchDaemons"
		}
	}

	logsDir := filepath.Join(homeDir, "Library", "Logs", "quad-ops")
	if domain == DomainSystem {
		logsDir = "/var/log/quad-ops"
	}

	return Options{
		Domain:      domain,
		LabelPrefix: "dev.trly.quad-ops",
		PlistDir:    plistDir,
		LogsDir:     logsDir,
		UID:         uid,
		UseSudo:     false,
	}
}

// Validate validates and normalizes options, resolving defaults.
func (o *Options) Validate() error {
	// Set defaults
	if o.Domain == "" {
		o.Domain = DomainUser
	}
	if o.LabelPrefix == "" {
		o.LabelPrefix = "dev.trly.quad-ops"
	}
	if o.UID == 0 {
		o.UID = os.Getuid()
	}

	// Validate domain
	if o.Domain != DomainUser && o.Domain != DomainSystem {
		return fmt.Errorf("invalid domain: %s (must be 'user' or 'system')", o.Domain)
	}

	// Set domain-specific defaults
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	if o.PlistDir == "" {
		if o.Domain == DomainUser {
			o.PlistDir = filepath.Join(homeDir, "Library", "LaunchAgents")
		} else {
			o.PlistDir = "/Library/LaunchDaemons"
		}
	}

	if o.LogsDir == "" {
		if o.Domain == DomainUser {
			o.LogsDir = filepath.Join(homeDir, "Library", "Logs", "quad-ops")
		} else {
			o.LogsDir = "/var/log/quad-ops"
		}
	}

	// Determine if sudo is needed for system domain
	if o.Domain == DomainSystem && os.Getuid() != 0 {
		o.UseSudo = true
	}

	// Resolve podman path
	if o.PodmanPath == "" {
		podmanPath, err := resolvePodmanPath()
		if err != nil {
			return fmt.Errorf("failed to resolve podman path: %w", err)
		}
		o.PodmanPath = podmanPath
	}

	// Verify podman exists
	if _, err := os.Stat(o.PodmanPath); err != nil {
		return fmt.Errorf("podman binary not found at %s: %w", o.PodmanPath, err)
	}

	// Ensure logs directory exists
	if err := os.MkdirAll(o.LogsDir, 0750); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Ensure plist directory exists
	if err := os.MkdirAll(o.PlistDir, 0750); err != nil {
		return fmt.Errorf("failed to create plist directory: %w", err)
	}

	return nil
}

// resolvePodmanPath attempts to find the podman binary.
func resolvePodmanPath() (string, error) {
	// Try exec.LookPath first
	if path, err := exec.LookPath("podman"); err == nil {
		return path, nil
	}

	// Try common Homebrew locations
	commonPaths := []string{
		"/opt/homebrew/bin/podman", // Apple Silicon
		"/usr/local/bin/podman",    // Intel
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("podman not found in PATH or common locations")
}

// DomainID returns the launchd domain identifier for launchctl commands.
func (o *Options) DomainID() string {
	if o.Domain == DomainSystem {
		return "system"
	}
	return fmt.Sprintf("gui/%d", o.UID)
}
