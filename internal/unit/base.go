// Package unit provides quadlet unit types and generation functionality
package unit

import "github.com/trly/quad-ops/internal/systemd"

// BaseUnit provides common fields and methods for all unit types.
type BaseUnit struct {
	*systemd.BaseUnit
	Name     string
	UnitType string
}

// NewBaseUnit creates a new BaseUnit with the given name and type.
func NewBaseUnit(name, unitType string) *BaseUnit {
	return &BaseUnit{
		BaseUnit: systemd.NewBaseUnit(name, unitType),
		Name:     name,
		UnitType: unitType,
	}
}
