package model

// BaseSystemdUnit provides common systemd unit operations.
type BaseSystemdUnit struct {
	Name string
	Type string
}

// GetServiceName returns the systemd service name for this unit.
func (u BaseSystemdUnit) GetServiceName() string {
	return u.Name + "." + u.Type
}

// GetUnitType returns the type of unit.
func (u BaseSystemdUnit) GetUnitType() string {
	return u.Type
}

// GetUnitName returns the name of the unit.
func (u BaseSystemdUnit) GetUnitName() string {
	return u.Name
}

// GetStatus returns the status of the unit.
func (u BaseSystemdUnit) GetStatus() (string, error) {
	// This will be implemented with proper systemd integration
	return "unknown", nil
}

// Start starts the unit.
func (u BaseSystemdUnit) Start() error {
	// This will be implemented with proper systemd integration
	return nil
}

// Stop stops the unit.
func (u BaseSystemdUnit) Stop() error {
	// This will be implemented with proper systemd integration
	return nil
}

// Restart restarts the unit.
func (u BaseSystemdUnit) Restart() error {
	// This will be implemented with proper systemd integration
	return nil
}

// Show displays the unit configuration.
func (u BaseSystemdUnit) Show() error {
	// This will be implemented with proper systemd integration
	return nil
}