// Package unit provides quadlet unit types and generation functionality
package unit

import "github.com/trly/quad-ops/internal/systemd"

// BaseUnit provides common fields and methods for all unit types.
type BaseUnit struct {
	*systemd.BaseUnit
	Name     string
	UnitType string
}
