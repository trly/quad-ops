package compose

import (
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
)

// Repository defines the interface for unit repository operations.
type Repository interface {
	FindAll() ([]repository.Unit, error)
	Create(unit *repository.Unit) (*repository.Unit, error)
	Delete(id string) error
}

// SystemdManager defines the interface for systemd operations.
type SystemdManager interface {
	RestartChangedUnits(units []systemd.UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error
	ReloadSystemd() error
	StopUnit(name, unitType string) error
}

// FileSystem defines the interface for file system operations.
type FileSystem interface {
	GetUnitFilePath(name, unitType string) string
	HasUnitChanged(unitPath, content string) bool
	WriteUnitFile(unitPath, content string) error
	GetContentHash(content string) string
}
