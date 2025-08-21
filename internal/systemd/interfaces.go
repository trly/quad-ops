// Package systemd provides systemd unit management operations.
package systemd

import (
	"context"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/trly/quad-ops/internal/dependency"
)

// Connection wraps systemd D-Bus operations for testability.
type Connection interface {
	// GetUnitProperty gets a property of a systemd unit.
	GetUnitProperty(ctx context.Context, unitName, propertyName string) (*dbus.Property, error)

	// GetUnitProperties gets all properties of a systemd unit.
	GetUnitProperties(ctx context.Context, unitName string) (map[string]interface{}, error)

	// StartUnit starts a systemd unit.
	StartUnit(ctx context.Context, unitName, mode string) (chan string, error)

	// StopUnit stops a systemd unit.
	StopUnit(ctx context.Context, unitName, mode string) (chan string, error)

	// RestartUnit restarts a systemd unit.
	RestartUnit(ctx context.Context, unitName, mode string) (chan string, error)

	// ResetFailedUnit resets the failed state of a unit.
	ResetFailedUnit(ctx context.Context, unitName string) error

	// Reload reloads systemd configuration.
	Reload(ctx context.Context) error

	// Close closes the connection.
	Close() error
}

// UnitManager manages systemd units and their operations.
type UnitManager interface {
	// GetUnit creates a Unit interface for the given name and type.
	GetUnit(name, unitType string) Unit

	// GetStatus returns the current status of a unit.
	GetStatus(unitName, unitType string) (string, error)

	// Start starts a unit.
	Start(unitName, unitType string) error

	// Stop stops a unit.
	Stop(unitName, unitType string) error

	// Restart restarts a unit.
	Restart(unitName, unitType string) error

	// Show displays unit configuration and status.
	Show(unitName, unitType string) error

	// ResetFailed resets the failed state of a unit.
	ResetFailed(unitName, unitType string) error

	// ReloadSystemd reloads systemd configuration.
	ReloadSystemd() error

	// GetUnitFailureDetails gets detailed failure information for a unit.
	GetUnitFailureDetails(unitName string) string
}

// Orchestrator handles dependency-aware unit management.
type Orchestrator interface {
	// StartUnitDependencyAware starts or restarts a unit with dependency awareness.
	StartUnitDependencyAware(unitName, unitType string, dependencyGraph *dependency.ServiceDependencyGraph) error

	// RestartChangedUnits restarts all changed units in dependency-aware order.
	RestartChangedUnits(changedUnits []UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error
}

// ContextProvider provides context for systemd operations.
type ContextProvider interface {
	// GetContext returns a context for systemd operations.
	GetContext() context.Context
}

// TextCaser provides text casing operations.
type TextCaser interface {
	// Title converts text to title case.
	Title(text string) string
}

// ConnectionFactory creates Connection instances.
type ConnectionFactory interface {
	// NewConnection creates a new systemd connection based on configuration.
	NewConnection(ctx context.Context, userMode bool) (Connection, error)
}
