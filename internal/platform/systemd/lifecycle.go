// Package systemd provides systemd-specific platform implementations.
package systemd

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/systemd"
)

// Lifecycle implements the platform.Lifecycle interface for systemd.
type Lifecycle struct {
	unitManager       systemd.UnitManager
	connectionFactory systemd.ConnectionFactory
	userMode          bool
	logger            log.Logger
}

// NewLifecycle creates a new systemd Lifecycle implementation.
func NewLifecycle(
	unitManager systemd.UnitManager,
	connectionFactory systemd.ConnectionFactory,
	userMode bool,
	logger log.Logger,
) *Lifecycle {
	return &Lifecycle{
		unitManager:       unitManager,
		connectionFactory: connectionFactory,
		userMode:          userMode,
		logger:            logger,
	}
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

// StartMany starts multiple services in dependency order.
func (l *Lifecycle) StartMany(ctx context.Context, names []string) map[string]error {
	l.logger.Debug("Starting multiple services", "count", len(names))

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
	for _, err := range results {
		if err == nil {
			successCount++
		}
	}

	l.logger.Debug("Completed starting services",
		"total", len(names),
		"success", successCount,
		"failed", len(names)-successCount)

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

// RestartMany restarts multiple services in dependency order.
func (l *Lifecycle) RestartMany(ctx context.Context, names []string) map[string]error {
	l.logger.Debug("Restarting multiple services", "count", len(names))

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
	for _, err := range results {
		if err == nil {
			successCount++
		}
	}

	l.logger.Debug("Completed restarting services",
		"total", len(names),
		"success", successCount,
		"failed", len(names)-successCount)

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
				return fmt.Errorf("state=%s, result=%s", state, result)
			}
		}
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
					errMsg += fmt.Sprintf(", exit_code=%s", strconv.Itoa(int(exitCode)))
				}
				return fmt.Errorf("%s", errMsg)
			}
		}

		return fmt.Errorf("failed to activate: state=%s", finalStateStr)

	case <-ctx.Done():
		return fmt.Errorf("activation wait cancelled: %w", ctx.Err())
	}
}
