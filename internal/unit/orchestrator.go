package unit

import (
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/systemd"
)

// Change tracks changes to a unit.
type Change struct {
	Name string
	Type string
	Hash []byte
}

// StartUnitDependencyAware starts or restarts a unit while being dependency-aware.
func StartUnitDependencyAware(unitName string, unitType string, dependencyGraph *dependency.ServiceDependencyGraph) error {
	return systemd.StartUnitDependencyAware(unitName, unitType, dependencyGraph)
}

// RestartChangedUnits restarts all changed units in dependency-aware order.
func RestartChangedUnits(changedUnits []QuadletUnit, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error {
	// Convert QuadletUnit slice to systemd.UnitChange slice
	systemdUnits := make([]systemd.UnitChange, len(changedUnits))
	for i, unit := range changedUnits {
		systemdUnits[i] = systemd.UnitChange{
			Name: unit.Name,
			Type: unit.Type,
			Unit: unit.GetSystemdUnit(),
		}
	}

	return systemd.RestartChangedUnits(systemdUnits, projectDependencyGraphs)
}
