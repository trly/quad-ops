package unit

// BaseUnit provides common fields and methods for all unit types.
type BaseUnit struct {
	Name     string
	UnitType string
}

// Implement SystemdUnit interface methods.

// GetServiceName returns the full systemd service name based on unit type.
func (u *BaseUnit) GetServiceName() string {
	switch u.UnitType {
	case "container":
		return u.Name + ".service"
	default:
		return u.Name + "-" + u.UnitType + ".service"
	}
}

// GetUnitType returns the type of the unit.
func (u *BaseUnit) GetUnitType() string {
	return u.UnitType
}

// GetUnitName returns the name of the unit.
func (u *BaseUnit) GetUnitName() string {
	return u.Name
}

// Implement all systemd operations using the BaseSystemdUnit.

// GetStatus returns the current status of the unit.
func (u *BaseUnit) GetStatus() (string, error) {
	base := BaseSystemdUnit{Name: u.Name, Type: u.UnitType}
	return base.GetStatus()
}

// Start starts the unit.
func (u *BaseUnit) Start() error {
	base := BaseSystemdUnit{Name: u.Name, Type: u.UnitType}
	return base.Start()
}

// Stop stops the unit.
func (u *BaseUnit) Stop() error {
	base := BaseSystemdUnit{Name: u.Name, Type: u.UnitType}
	return base.Stop()
}

// Restart restarts the unit.
func (u *BaseUnit) Restart() error {
	base := BaseSystemdUnit{Name: u.Name, Type: u.UnitType}
	return base.Restart()
}

// Show displays the unit configuration and status.
func (u *BaseUnit) Show() error {
	base := BaseSystemdUnit{Name: u.Name, Type: u.UnitType}
	return base.Show()
}

// ResetFailed resets the failed state of the unit.
func (u *BaseUnit) ResetFailed() error {
	base := BaseSystemdUnit{Name: u.Name, Type: u.UnitType}
	return base.ResetFailed()
}
