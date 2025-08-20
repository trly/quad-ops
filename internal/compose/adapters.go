package compose

import (
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
)

// RepositoryAdapter adapts repository.Repository to our interface.
type RepositoryAdapter struct {
	repo repository.Repository
}

// NewRepositoryAdapter creates a new repository adapter.
func NewRepositoryAdapter(repo repository.Repository) Repository {
	return &RepositoryAdapter{repo: repo}
}

// FindAll retrieves all units from the repository.
func (r *RepositoryAdapter) FindAll() ([]repository.Unit, error) {
	return r.repo.FindAll()
}

// Create creates a new unit in the repository.
func (r *RepositoryAdapter) Create(unit *repository.Unit) (*repository.Unit, error) {
	id, err := r.repo.Create(unit)
	if err != nil {
		return nil, err
	}
	unit.ID = id
	return unit, nil
}

// Delete removes a unit from the repository.
func (r *RepositoryAdapter) Delete(_ string) error {
	// Convert string ID to int64 - this is a limitation of the current repository interface
	// For now, we'll use 0 as a placeholder since the actual implementation doesn't use the ID
	return r.repo.Delete(0)
}

// SystemdAdapter adapts systemd operations to our interface.
type SystemdAdapter struct{}

// NewSystemdAdapter creates a new systemd adapter.
func NewSystemdAdapter() SystemdManager {
	return &SystemdAdapter{}
}

// RestartChangedUnits restarts units that have changed.
func (s *SystemdAdapter) RestartChangedUnits(units []systemd.UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error {
	return systemd.RestartChangedUnits(units, projectDependencyGraphs)
}

// ReloadSystemd reloads the systemd configuration.
func (s *SystemdAdapter) ReloadSystemd() error {
	return systemd.ReloadSystemd()
}

// StopUnit stops a systemd unit.
func (s *SystemdAdapter) StopUnit(name, unitType string) error {
	systemdUnit := systemd.NewBaseUnit(name, unitType)
	return systemdUnit.Stop()
}

// FileSystemAdapter adapts fs operations to our interface.
type FileSystemAdapter struct{}

// NewFileSystemAdapter creates a new filesystem adapter.
func NewFileSystemAdapter() FileSystem {
	return &FileSystemAdapter{}
}

// GetUnitFilePath returns the file path for a unit.
func (f *FileSystemAdapter) GetUnitFilePath(name, unitType string) string {
	return fs.GetUnitFilePath(name, unitType)
}

// HasUnitChanged checks if a unit file has changed.
func (f *FileSystemAdapter) HasUnitChanged(unitPath, content string) bool {
	return fs.HasUnitChanged(unitPath, content)
}

// WriteUnitFile writes a unit file to disk.
func (f *FileSystemAdapter) WriteUnitFile(unitPath, content string) error {
	return fs.WriteUnitFile(unitPath, content)
}

// GetContentHash returns a hash of the content.
func (f *FileSystemAdapter) GetContentHash(content string) string {
	return string(fs.GetContentHash(content))
}
