// Package systemd provides systemd unit management operations.
package systemd

// Unit defines the interface for managing systemd units.
type Unit interface {
	// GetServiceName returns the full systemd service name
	GetServiceName() string

	// GetUnitType returns the type of the unit (container, volume, network, etc.)
	GetUnitType() string

	// GetUnitName returns the name of the unit
	GetUnitName() string

	// GetStatus returns the current status of the unit
	GetStatus() (string, error)

	// Start starts the unit
	Start() error

	// Stop stops the unit
	Stop() error

	// Restart restarts the unit
	Restart() error

	// Show displays the unit configuration and status
	Show() error

	// ResetFailed resets the failed state of the unit
	ResetFailed() error
}

// BaseUnit provides common implementation for all systemd units.
type BaseUnit struct {
	Name string
	Type string
}

// NewBaseUnit creates a new BaseUnit with the given name and type.
func NewBaseUnit(name, unitType string) *BaseUnit {
	return &BaseUnit{Name: name, Type: unitType}
}

// GetServiceName returns the full systemd service name based on unit type.
func (u *BaseUnit) GetServiceName() string {
	switch u.Type {
	case "container":
		return u.Name + ".service"
	default:
		return u.Name + "-" + u.Type + ".service"
	}
}

// GetUnitType returns the type of the unit.
func (u *BaseUnit) GetUnitType() string {
	return u.Type
}

// GetUnitName returns the name of the unit.
func (u *BaseUnit) GetUnitName() string {
	return u.Name
}
