package unit

// BaseUnit provides common fields and methods for all unit types.
type BaseUnit struct {
	*BaseSystemdUnit
	Name     string
	UnitType string
}

// NewBaseUnit creates a new BaseUnit with the given name and type.
func NewBaseUnit(name, unitType string) *BaseUnit {
	return &BaseUnit{
		BaseSystemdUnit: &BaseSystemdUnit{Name: name, Type: unitType},
		Name:            name,
		UnitType:        unitType,
	}
}
