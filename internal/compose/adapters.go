package compose

import (
	"github.com/trly/quad-ops/internal/config"
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
type SystemdAdapter struct {
	unitManager  systemd.UnitManager
	orchestrator systemd.Orchestrator
}

// NewSystemdAdapter creates a new systemd adapter with dependency injection.
func NewSystemdAdapter(unitManager systemd.UnitManager, orchestrator systemd.Orchestrator) SystemdManager {
	return &SystemdAdapter{
		unitManager:  unitManager,
		orchestrator: orchestrator,
	}
}

// RestartChangedUnits restarts units that have changed.
func (s *SystemdAdapter) RestartChangedUnits(units []systemd.UnitChange, projectDependencyGraphs map[string]*dependency.ServiceDependencyGraph) error {
	return s.orchestrator.RestartChangedUnits(units, projectDependencyGraphs)
}

// ReloadSystemd reloads the systemd configuration.
func (s *SystemdAdapter) ReloadSystemd() error {
	return s.unitManager.ReloadSystemd()
}

// StopUnit stops a systemd unit.
func (s *SystemdAdapter) StopUnit(name, unitType string) error {
	return s.unitManager.Stop(name, unitType)
}

// FileSystemAdapter adapts fs operations to our interface.
type FileSystemAdapter struct {
	fsService *fs.Service
}

// NewFileSystemAdapter creates a new filesystem adapter with config provider.
func NewFileSystemAdapter(configProvider config.Provider) FileSystem {
	return &FileSystemAdapter{
		fsService: fs.NewService(configProvider),
	}
}

// NewFileSystemAdapterWithConfig creates a new filesystem adapter with config provider.
// This is an alias for NewFileSystemAdapter to maintain backward compatibility.
func NewFileSystemAdapterWithConfig(configProvider config.Provider) FileSystem {
	return NewFileSystemAdapter(configProvider)
}

// GetUnitFilePath returns the file path for a unit.
func (f *FileSystemAdapter) GetUnitFilePath(name, unitType string) string {
	return f.fsService.GetUnitFilePath(name, unitType)
}

// HasUnitChanged checks if a unit file has changed.
func (f *FileSystemAdapter) HasUnitChanged(unitPath, content string) bool {
	return f.fsService.HasUnitChanged(unitPath, content)
}

// WriteUnitFile writes a unit file to disk.
func (f *FileSystemAdapter) WriteUnitFile(unitPath, content string) error {
	return f.fsService.WriteUnitFile(unitPath, content)
}

// GetContentHash returns a hash of the content.
func (f *FileSystemAdapter) GetContentHash(content string) string {
	return f.fsService.GetContentHash(content)
}
