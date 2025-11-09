// Package systemd provides systemd-specific platform implementations.
package systemd

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/systemd"
)

// Lifecycle implements the platform.Lifecycle interface for systemd.
type Lifecycle struct {
	unitManager           systemd.UnitManager
	connectionFactory     systemd.ConnectionFactory
	userMode              bool
	logger                log.Logger
	unitGenerationTimeout time.Duration // Maximum time to wait for units to be generated (default 5s)
}

// NewLifecycle creates a new systemd Lifecycle implementation.
func NewLifecycle(
	unitManager systemd.UnitManager,
	connectionFactory systemd.ConnectionFactory,
	userMode bool,
	logger log.Logger,
) *Lifecycle {
	return &Lifecycle{
		unitManager:           unitManager,
		connectionFactory:     connectionFactory,
		userMode:              userMode,
		logger:                logger,
		unitGenerationTimeout: 5 * time.Second, // Default 5s timeout for unit generation
	}
}

// SetUnitGenerationTimeout sets the timeout for waiting for units to be generated.
// This is primarily useful for testing.
func (l *Lifecycle) SetUnitGenerationTimeout(timeout time.Duration) {
	l.unitGenerationTimeout = timeout
}

// Name returns the platform name.
func (l *Lifecycle) Name() string {
	return "systemd"
}

// Reload reloads the service manager configuration.
func (l *Lifecycle) Reload(ctx context.Context) error {
	l.logger.Debug("Reloading systemd daemon configuration")

	conn, err := l.connectionFactory.NewConnection(ctx, l.userMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.Reload(ctx); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	l.logger.Debug("Successfully reloaded systemd daemon")
	return nil
}

// Start starts a service.
func (l *Lifecycle) Start(ctx context.Context, name string) error {
	l.logger.Debug("Starting service", "name", name)

	conn, err := l.connectionFactory.NewConnection(ctx, l.userMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := name + ".service"
	ch, err := conn.StartUnit(ctx, serviceName, "replace")
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w", name, err)
	}

	select {
	case result := <-ch:
		if result != "done" {
			// Check if still activating.
			if err := l.waitForActivation(ctx, conn, serviceName); err != nil {
				return fmt.Errorf("service %s failed to start: %w", name, err)
			}
		}
	case <-ctx.Done():
		return fmt.Errorf("start operation cancelled: %w", ctx.Err())
	}

	l.logger.Debug("Successfully started service", "name", name)
	return nil
}

// Stop stops a service.
func (l *Lifecycle) Stop(ctx context.Context, name string) error {
	l.logger.Debug("Stopping service", "name", name)

	conn, err := l.connectionFactory.NewConnection(ctx, l.userMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := name + ".service"
	ch, err := conn.StopUnit(ctx, serviceName, "replace")
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w", name, err)
	}

	select {
	case result := <-ch:
		if result != "done" {
			return fmt.Errorf("service %s failed to stop: result=%s", name, result)
		}
	case <-ctx.Done():
		return fmt.Errorf("stop operation cancelled: %w", ctx.Err())
	}

	l.logger.Debug("Successfully stopped service", "name", name)
	return nil
}

// Restart restarts a service.
func (l *Lifecycle) Restart(ctx context.Context, name string) error {
	l.logger.Debug("Restarting service", "name", name)

	conn, err := l.connectionFactory.NewConnection(ctx, l.userMode)
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := name + ".service"
	ch, err := conn.RestartUnit(ctx, serviceName, "replace")
	if err != nil {
		return fmt.Errorf("failed to restart service %s: %w", name, err)
	}

	select {
	case result := <-ch:
		if result != "done" {
			// Check if still activating.
			if err := l.waitForActivation(ctx, conn, serviceName); err != nil {
				return fmt.Errorf("service %s failed to restart: %w", name, err)
			}
		}
	case <-ctx.Done():
		return fmt.Errorf("restart operation cancelled: %w", ctx.Err())
	}

	l.logger.Debug("Successfully restarted service", "name", name)
	return nil
}

// Status returns the status of a service.
func (l *Lifecycle) Status(ctx context.Context, name string) (*platform.ServiceStatus, error) {
	conn, err := l.connectionFactory.NewConnection(ctx, l.userMode)
	if err != nil {
		return nil, fmt.Errorf("error connecting to systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()

	serviceName := name + ".service"
	props, err := conn.GetUnitProperties(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get status for service %s: %w", name, err)
	}

	status := &platform.ServiceStatus{
		Name: name,
	}

	// Extract ActiveState.
	if activeState, ok := props["ActiveState"].(string); ok {
		status.Active = activeState == "active"
		status.State = activeState
	}

	// Extract SubState.
	if subState, ok := props["SubState"].(string); ok {
		status.SubState = subState
	}

	// Extract Description.
	if desc, ok := props["Description"].(string); ok {
		status.Description = desc
	}

	// Extract PID.
	if mainPID, ok := props["MainPID"].(uint32); ok && mainPID > 0 {
		status.PID = int(mainPID)
	}

	// Extract start time.
	if activeEnterTimestamp, ok := props["ActiveEnterTimestamp"].(uint64); ok && activeEnterTimestamp > 0 {
		// Convert microseconds since epoch to time.
		// #nosec G115 - timestamp is from systemd dbus, value is controlled.
		t := time.Unix(0, int64(activeEnterTimestamp)*1000)
		status.Since = t.Format(time.RFC3339)
	}

	// Extract error if failed.
	if result, ok := props["Result"].(string); ok && result != "success" {
		status.Error = fmt.Sprintf("Result: %s", result)

		// Add exit status if available.
		if execMainStatus, ok := props["ExecMainStatus"].(int32); ok && execMainStatus != 0 {
			status.Error += fmt.Sprintf(", Exit Code: %d", execMainStatus)
		}
	}

	return status, nil
}

// cleanupOrphanedRootlessportProcesses kills any orphaned rootlessport processes
// that may be holding port bindings from failed podman containers.
func (l *Lifecycle) cleanupOrphanedRootlessportProcesses(ctx context.Context) error {
	// Find rootlessport processes owned by current user
	cmd := exec.CommandContext(ctx, "pgrep", "-f", "rootlessport")
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 when no processes found, which is not an error for us
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			l.logger.Debug("No orphaned rootlessport processes found (pgrep exit code 1)")
			return nil
		}
		l.logger.Warn("Failed to check for rootlessport processes", "error", err)
		return nil // Don't fail the operation if we can't check
	}

	pids := strings.Fields(string(output))
	if len(pids) == 0 {
		return nil
	}

	l.logger.Info("Found orphaned rootlessport processes, cleaning up", "count", len(pids))

	// Kill the processes
	for _, pidStr := range pids {
		// Validate and convert PID to avoid command injection
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			l.logger.Warn("Invalid PID format, skipping", "pid", pidStr)
			continue
		}

		killCmd := exec.CommandContext(ctx, "kill", strconv.Itoa(pid)) // #nosec G204 - PID is validated as integer from pgrep output
		if err := killCmd.Run(); err != nil {
			l.logger.Warn("Failed to kill rootlessport process", "pid", pid, "error", err)
		} else {
			l.logger.Debug("Killed orphaned rootlessport process", "pid", pid)
		}
	}

	return nil
}

// StartMany starts multiple services in dependency order.
func (l *Lifecycle) StartMany(ctx context.Context, names []string) map[string]error {
	l.logger.Debug("Starting multiple services", "count", len(names))

	// Clean up any orphaned rootlessport processes before starting services
	if err := l.cleanupOrphanedRootlessportProcesses(ctx); err != nil {
		l.logger.Warn("Failed to cleanup orphaned rootlessport processes", "error", err)
		// Don't fail the operation, just log the warning
	}

	results := make(map[string]error)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(svcName string) {
			defer wg.Done()

			err := l.Start(ctx, svcName)
			mu.Lock()
			results[svcName] = err
			mu.Unlock()

			if err != nil {
				l.logger.Error("Failed to start service", "name", svcName, "error", err)
			}
		}(name)
	}

	wg.Wait()

	successCount := 0
	failedCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		} else {
			failedCount++
		}
	}

	// If any services failed, run cleanup again to remove orphaned rootlessport processes
	// that may have been left behind by the failed service starts
	if failedCount > 0 {
		l.logger.Debug("Services failed, running cleanup for orphaned processes")
		if err := l.cleanupOrphanedRootlessportProcesses(ctx); err != nil {
			l.logger.Warn("Failed to cleanup orphaned rootlessport processes after failures", "error", err)
		}
	}

	l.logger.Debug("Completed starting services",
		"total", len(names),
		"success", successCount,
		"failed", failedCount)

	return results
}

// StopMany stops multiple services in reverse dependency order.
func (l *Lifecycle) StopMany(ctx context.Context, names []string) map[string]error {
	l.logger.Debug("Stopping multiple services", "count", len(names))

	results := make(map[string]error)
	var mu sync.Mutex

	// Stop in reverse order.
	reversed := make([]string, len(names))
	for i, name := range names {
		reversed[len(names)-1-i] = name
	}

	var wg sync.WaitGroup
	for _, name := range reversed {
		wg.Add(1)
		go func(svcName string) {
			defer wg.Done()

			err := l.Stop(ctx, svcName)
			mu.Lock()
			results[svcName] = err
			mu.Unlock()

			if err != nil {
				l.logger.Error("Failed to stop service", "name", svcName, "error", err)
			}
		}(name)
	}

	wg.Wait()

	successCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		}
	}

	l.logger.Debug("Completed stopping services",
		"total", len(names),
		"success", successCount,
		"failed", len(names)-successCount)

	return results
}

// waitForUnitGeneration waits for a unit to be generated by systemd with exponential backoff retry.
// It checks if the unit file exists and can be manipulated via D-Bus with exponential backoff
// (starting at 50ms, doubling each retry, up to unitGenerationTimeout).
func (l *Lifecycle) waitForUnitGeneration(ctx context.Context, serviceName string) error {
	maxWait := l.unitGenerationTimeout
	deadline := time.Now().Add(maxWait)

	backoff := 50 * time.Millisecond
	attempt := 1

	for {
		conn, err := l.connectionFactory.NewConnection(ctx, l.userMode)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("error connecting to systemd: %w", err)
		}

		// Try to get unit properties - if successful, unit exists and is accessible
		_, err = conn.GetUnitProperties(ctx, serviceName)
		_ = conn.Close()

		if err == nil {
			// Unit exists and is accessible
			l.logger.Debug("Unit generated successfully", "unit", serviceName, "attempts", attempt)
			return nil
		}

		// Check if we've exceeded the timeout
		if time.Now().Add(backoff).After(deadline) {
			l.logger.Error("Unit generation timeout", "unit", serviceName, "timeout", maxWait.String(), "attempts", attempt)
			return fmt.Errorf("unit %s failed to be generated after %s (%d retries)", serviceName, maxWait.String(), attempt)
		}

		// Log retry attempt at debug level
		l.logger.Debug("Unit not yet generated, retrying",
			"unit", serviceName,
			"attempt", attempt,
			"nextRetryIn", backoff.String(),
			"error", err)

		// Wait before retrying
		select {
		case <-time.After(backoff):
			// Continue to next attempt
		case <-ctx.Done():
			return fmt.Errorf("unit generation wait cancelled: %w", ctx.Err())
		}

		// Exponential backoff: double the wait time for next retry
		backoff *= 2

		attempt++
	}
}

// RestartMany restarts multiple services in dependency order.
func (l *Lifecycle) RestartMany(ctx context.Context, names []string) map[string]error {
	l.logger.Debug("Restarting multiple services", "count", len(names))

	// Clean up any orphaned rootlessport processes before restarting services
	if err := l.cleanupOrphanedRootlessportProcesses(ctx); err != nil {
		l.logger.Warn("Failed to cleanup orphaned rootlessport processes", "error", err)
		// Don't fail the operation, just log the warning
	}

	// Verify units exist after reload - retry with exponential backoff to give systemd generator time to run
	l.logger.Info("Verifying unit files were generated", "count", len(names))
	for _, name := range names {
		serviceName := name + ".service"
		if err := l.waitForUnitGeneration(ctx, serviceName); err != nil {
			// Unit failed to generate - return error for this service
			l.logger.Error("Unit generation verification failed", "service", name, "error", err)
			results := make(map[string]error)
			results[name] = err
			return results
		}
	}

	results := make(map[string]error)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(svcName string) {
			defer wg.Done()

			err := l.Restart(ctx, svcName)
			mu.Lock()
			results[svcName] = err
			mu.Unlock()

			if err != nil {
				l.logger.Error("Failed to restart service", "name", svcName, "error", err)
			}
		}(name)
	}

	wg.Wait()

	successCount := 0
	failedCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		} else {
			failedCount++
		}
	}

	// If any services failed, run cleanup again to remove orphaned rootlessport processes
	// that may have been left behind by the failed service restarts
	if failedCount > 0 {
		l.logger.Debug("Services failed, running cleanup for orphaned processes")
		if err := l.cleanupOrphanedRootlessportProcesses(ctx); err != nil {
			l.logger.Warn("Failed to cleanup orphaned rootlessport processes after failures", "error", err)
		}
	}

	l.logger.Debug("Completed restarting services",
		"total", len(names),
		"success", successCount,
		"failed", failedCount)

	return results
}

// waitForActivation waits for a service to finish activating.
func (l *Lifecycle) waitForActivation(ctx context.Context, conn systemd.Connection, serviceName string) error {
	activeState, err := conn.GetUnitProperty(ctx, serviceName, "ActiveState")
	if err != nil {
		return fmt.Errorf("failed to get active state: %w", err)
	}

	state := activeState.Value.Value().(string)
	if state == "active" {
		return nil
	}

	if state != "activating" {
		// Get detailed error info.
		props, err := conn.GetUnitProperties(ctx, serviceName)
		if err == nil {
			if result, ok := props["Result"].(string); ok && result != "success" {
				errMsg := fmt.Sprintf("state=%s, result=%s", state, result)
				if exitCode, ok := props["ExecMainStatus"].(int32); ok && exitCode != 0 {
					errMsg += fmt.Sprintf(", exit_code=%d", exitCode)
				}
				l.logger.Error("Service failed to start", "name", serviceName, "error_details", errMsg)
				return fmt.Errorf("%s", errMsg)
			}
		}
		l.logger.Error("Service in unexpected state", "name", serviceName, "state", state)
		return fmt.Errorf("unexpected state: %s", state)
	}

	// Wait for activation to complete.
	subState, err := conn.GetUnitProperty(ctx, serviceName, "SubState")
	subStateStr := "unknown"
	if err == nil {
		subStateStr = subState.Value.Value().(string)
	}

	l.logger.Debug("Service activating, waiting for completion",
		"name", serviceName,
		"subState", subStateStr)

	// Determine wait time based on sub-state.
	waitTime := 10 * time.Second
	if subStateStr == "start" {
		waitTime = 60 * time.Second // Longer for image pulls.
	}

	timer := time.NewTimer(waitTime)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Check final state.
		finalState, err := conn.GetUnitProperty(ctx, serviceName, "ActiveState")
		if err != nil {
			return fmt.Errorf("failed to get final state: %w", err)
		}

		finalStateStr := finalState.Value.Value().(string)
		if finalStateStr == "active" {
			l.logger.Debug("Service successfully activated", "name", serviceName)
			return nil
		}

		// Get error details.
		props, err := conn.GetUnitProperties(ctx, serviceName)
		if err == nil {
			if result, ok := props["Result"].(string); ok && result != "success" {
				errMsg := fmt.Sprintf("state=%s, result=%s", finalStateStr, result)
				if exitCode, ok := props["ExecMainStatus"].(int32); ok && exitCode != 0 {
					errMsg += fmt.Sprintf(", exit_code=%d", exitCode)
				}
				l.logger.Error("Service activation timeout", "name", serviceName, "error_details", errMsg)
				return fmt.Errorf("%s", errMsg)
			}
		}

		l.logger.Error("Service activation timeout", "name", serviceName, "final_state", finalStateStr)
		return fmt.Errorf("failed to activate: state=%s", finalStateStr)

	case <-ctx.Done():
		return fmt.Errorf("activation wait cancelled: %w", ctx.Err())
	}
}
