package launchd

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
)

// Lifecycle implements platform.Lifecycle for macOS launchd.
type Lifecycle struct {
	opts   Options
	exec   execx.Runner
	logger log.Logger
}

// NewLifecycle creates a new launchd lifecycle manager.
func NewLifecycle(opts Options, exec execx.Runner, logger log.Logger) (*Lifecycle, error) {
	// Validate and normalize options
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	return &Lifecycle{
		opts:   opts,
		exec:   exec,
		logger: logger,
	}, nil
}

// Name returns the platform name.
func (l *Lifecycle) Name() string {
	return "launchd"
}

// Reload is a no-op for launchd (changes applied per-service on restart).
func (l *Lifecycle) Reload(_ context.Context) error {
	l.logger.Debug("Reload called (no-op for launchd)")
	return nil
}

// Start starts a service.
func (l *Lifecycle) Start(ctx context.Context, name string) error {
	// Check podman machine is running
	if err := l.checkPodmanMachine(ctx); err != nil {
		return fmt.Errorf("podman machine check failed: %w", err)
	}

	label := l.buildLabel(name)
	plistPath := l.buildPlistPath(label)
	domainTarget := l.buildDomainTarget(label)

	l.logger.Debug("Starting service",
		"service", name,
		"label", label,
		"domain", domainTarget,
	)

	// Try modern launchctl bootstrap + kickstart
	if err := l.runCommand(ctx, "launchctl", "bootstrap", l.opts.DomainID(), plistPath); err != nil {
		// If already bootstrapped, that's fine
		if !strings.Contains(err.Error(), "already loaded") && !strings.Contains(err.Error(), "service already loaded") {
			l.logger.Debug("Bootstrap failed, trying legacy load", "error", err)

			// Fallback to legacy load
			if err := l.runCommand(ctx, "launchctl", "load", "-w", plistPath); err != nil {
				return fmt.Errorf("failed to load service: %w", err)
			}
		}
	}

	// Enable the service
	_ = l.runCommand(ctx, "launchctl", "enable", domainTarget)

	// Kickstart (start) the service
	if err := l.runCommand(ctx, "launchctl", "kickstart", "-k", domainTarget); err != nil {
		// Fallback to legacy start
		if err := l.runCommand(ctx, "launchctl", "start", label); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
	}

	l.logger.Info("Service started", "service", name, "label", label)
	return nil
}

// Stop stops a service.
func (l *Lifecycle) Stop(ctx context.Context, name string) error {
	label := l.buildLabel(name)
	domainTarget := l.buildDomainTarget(label)

	l.logger.Debug("Stopping service",
		"service", name,
		"label", label,
		"domain", domainTarget,
	)

	// Try modern bootout
	if err := l.runCommand(ctx, "launchctl", "bootout", domainTarget); err != nil {
		// Fallback to legacy stop + unload
		_ = l.runCommand(ctx, "launchctl", "stop", label)

		plistPath := l.buildPlistPath(label)
		if err := l.runCommand(ctx, "launchctl", "unload", "-w", plistPath); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
	}

	l.logger.Info("Service stopped", "service", name, "label", label)
	return nil
}

// Restart restarts a service and reloads its plist configuration.
func (l *Lifecycle) Restart(ctx context.Context, name string) error {
	label := l.buildLabel(name)
	domainTarget := l.buildDomainTarget(label)
	plistPath := l.buildPlistPath(label)

	l.logger.Debug("Restarting service",
		"service", name,
		"label", label,
	)

	// Check podman machine is running
	if err := l.checkPodmanMachine(ctx); err != nil {
		return fmt.Errorf("podman machine check failed: %w", err)
	}

	// 1. Bootout (stop and unload) - ignore errors if not running
	_ = l.runCommand(ctx, "launchctl", "bootout", domainTarget)

	// 2. Bootstrap (reload plist)
	if err := l.runCommand(ctx, "launchctl", "bootstrap", l.opts.DomainID(), plistPath); err != nil {
		// Fallback to legacy load for older macOS
		if err := l.runCommand(ctx, "launchctl", "load", "-w", plistPath); err != nil {
			return fmt.Errorf("failed to reload plist: %w", err)
		}
	}

	// 3. Enable if possible
	_ = l.runCommand(ctx, "launchctl", "enable", domainTarget)

	// 4. Kickstart to start the service
	if err := l.runCommand(ctx, "launchctl", "kickstart", "-k", domainTarget); err != nil {
		// Fallback to legacy start
		if err := l.runCommand(ctx, "launchctl", "start", label); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
	}

	l.logger.Info("Service restarted", "service", name, "label", label)
	return nil
}

// Status returns the status of a service.
func (l *Lifecycle) Status(ctx context.Context, name string) (*platform.ServiceStatus, error) {
	label := l.buildLabel(name)
	domainTarget := l.buildDomainTarget(label)

	status := &platform.ServiceStatus{
		Name:   name,
		Active: false,
		State:  "stopped",
	}

	// Try modern launchctl print
	output, err := l.runCommandOutput(ctx, "launchctl", "print", domainTarget)
	if err == nil {
		// Parse output for state and PID
		if strings.Contains(output, "state = running") {
			status.Active = true
			status.State = "running"
		}

		// Extract PID
		pidRegex := regexp.MustCompile(`pid = (\d+)`)
		if matches := pidRegex.FindStringSubmatch(output); len(matches) > 1 {
			if pid, err := strconv.Atoi(matches[1]); err == nil {
				status.PID = pid
			}
		}

		// Extract description/label
		status.Description = fmt.Sprintf("launchd service %s", label)
		return status, nil
	}

	// Fallback to legacy list
	output, err = l.runCommandOutput(ctx, "launchctl", "list")
	if err != nil {
		return status, nil
	}

	// Parse list output for this label
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, label) {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				// Format: PID Status Label
				if parts[0] != "-" {
					if pid, err := strconv.Atoi(parts[0]); err == nil {
						status.PID = pid
						status.Active = true
						status.State = "running"
					}
				}
			}
			break
		}
	}

	status.Description = fmt.Sprintf("launchd service %s", label)
	return status, nil
}

// StartMany starts multiple services (no dependency ordering).
func (l *Lifecycle) StartMany(ctx context.Context, names []string) map[string]error {
	results := make(map[string]error)
	for _, name := range names {
		results[name] = l.Start(ctx, name)
	}
	return results
}

// StopMany stops multiple services (no dependency ordering).
func (l *Lifecycle) StopMany(ctx context.Context, names []string) map[string]error {
	results := make(map[string]error)
	for _, name := range names {
		results[name] = l.Stop(ctx, name)
	}
	return results
}

// RestartMany restarts multiple services (no dependency ordering).
func (l *Lifecycle) RestartMany(ctx context.Context, names []string) map[string]error {
	results := make(map[string]error)
	for _, name := range names {
		results[name] = l.Restart(ctx, name)
	}
	return results
}

// checkPodmanMachine verifies podman machine is running.
func (l *Lifecycle) checkPodmanMachine(ctx context.Context) error {
	output, err := l.runCommandOutput(ctx, l.opts.PodmanPath, "machine", "inspect", "--format", "{{.State}}")
	if err != nil {
		return fmt.Errorf("podman machine not found (run: podman machine init && podman machine start): %w", err)
	}

	state := strings.TrimSpace(output)
	if state != "running" {
		return fmt.Errorf("podman machine is not running (current state: %s, run: podman machine start)", state)
	}

	return nil
}

// buildLabel creates a launchd label from service name.
func (l *Lifecycle) buildLabel(serviceName string) string {
	return SanitizeLabel(fmt.Sprintf("%s.%s", l.opts.LabelPrefix, serviceName))
}

// buildPlistPath returns the full path to a plist file.
func (l *Lifecycle) buildPlistPath(label string) string {
	return fmt.Sprintf("%s/%s.plist", l.opts.PlistDir, label)
}

// buildDomainTarget returns the domain/label target for launchctl commands.
func (l *Lifecycle) buildDomainTarget(label string) string {
	return fmt.Sprintf("%s/%s", l.opts.DomainID(), label)
}

// runCommand executes a command with optional sudo.
func (l *Lifecycle) runCommand(ctx context.Context, name string, args ...string) error {
	_, err := l.runCommandOutput(ctx, name, args...)
	return err
}

// runCommandOutput executes a command and returns output with optional sudo.
func (l *Lifecycle) runCommandOutput(ctx context.Context, name string, args ...string) (string, error) {
	// Prepend sudo if needed
	if l.opts.UseSudo && name != l.opts.PodmanPath {
		args = append([]string{name}, args...)
		name = "sudo"
	}

	output, err := l.exec.CombinedOutput(ctx, name, args...)
	if err != nil {
		return "", fmt.Errorf("%s %v failed: %w (output: %s)", name, args, err, string(output))
	}

	return string(output), nil
}
