package unit

import "github.com/trly/quad-ops/internal/systemd"

// SystemdUnit is an alias for the systemd.Unit interface for backward compatibility.
//
// Deprecated: Use systemd.Unit interface directly. This alias will be removed in v2.0.
type SystemdUnit = systemd.Unit

// BaseSystemdUnit is an alias for the systemd.BaseUnit for backward compatibility.
//
// Deprecated: Use systemd.BaseUnit directly. This alias will be removed in v2.0.
type BaseSystemdUnit = systemd.BaseUnit

// Legacy function aliases for backward compatibility

// ReloadSystemd reloads the systemd configuration.
//
// Deprecated: Use systemd.ReloadSystemd() directly. This function will be removed in v2.0.
func ReloadSystemd() error {
	return systemd.ReloadSystemd()
}

// GetUnitStatus returns the status of a systemd unit.
//
// Deprecated: Use systemd.Unit interface methods instead. This function will be removed in v2.0.
func GetUnitStatus(unitName string, unitType string) (string, error) {
	return systemd.GetUnitStatus(unitName, unitType)
}

// RestartUnit restarts a systemd unit.
//
// Deprecated: Use systemd.Unit interface methods instead. This function will be removed in v2.0.
func RestartUnit(unitName string, unitType string) error {
	return systemd.RestartUnit(unitName, unitType)
}

// StartUnit starts a systemd unit.
//
// Deprecated: Use systemd.Unit interface methods instead. This function will be removed in v2.0.
func StartUnit(unitName string, unitType string) error {
	return systemd.StartUnit(unitName, unitType)
}

// StopUnit stops a systemd unit.
//
// Deprecated: Use systemd.Unit interface methods instead. This function will be removed in v2.0.
func StopUnit(unitName string, unitType string) error {
	return systemd.StopUnit(unitName, unitType)
}

// ShowUnit displays information about a systemd unit.
//
// Deprecated: Use systemd.Unit interface methods instead. This function will be removed in v2.0.
func ShowUnit(unitName string, unitType string) error {
	return systemd.ShowUnit(unitName, unitType)
}

// ResetFailedUnit resets the "failed" state of a systemd unit.
//
// Deprecated: Use systemd.Unit interface methods instead. This function will be removed in v2.0.
func ResetFailedUnit(unitName string, unitType string) error {
	return systemd.ResetFailedUnit(unitName, unitType)
}
