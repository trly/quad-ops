// Package repository provides data access layer for quad-ops units.
package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/log"
)

// Unit represents a unit managed by quad-ops.
type Unit struct {
	ID        int64
	Name      string
	Type      string
	SHA1Hash  []byte
	UpdatedAt time.Time
}

// Repository defines the interface for unit data access operations.
type Repository interface {
	FindAll() ([]Unit, error)
	FindByUnitType(unitType string) ([]Unit, error)
	FindByID(id int64) (Unit, error)
	Create(unit *Unit) (int64, error)
	Delete(id int64) error
}

// SystemdRepository implements Repository interface by querying systemd directly.
type SystemdRepository struct {
	conn      *dbus.Conn
	logger    log.Logger
	fsService *fs.Service
}

// NewRepository creates a new systemd-based unit repository.
func NewRepository(logger log.Logger, fsService *fs.Service) Repository {
	return &SystemdRepository{
		logger:    logger,
		fsService: fsService,
	}
}

// close closes the systemd connection if it exists.
func (r *SystemdRepository) close() {
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
}

// FindAll retrieves all quad-ops managed units by scanning systemd and the filesystem.
func (r *SystemdRepository) FindAll() ([]Unit, error) {
	// Don't require systemd connection for filesystem-based scanning
	// Just close any existing connection when done
	defer r.close()

	var units []Unit
	unitTypes := []string{"container", "volume", "network", "build"}

	for _, unitType := range unitTypes {
		typeUnits, err := r.FindByUnitType(unitType)
		if err != nil {
			r.logger.Debug("Error finding units by type", "type", unitType, "error", err)
			continue
		}
		units = append(units, typeUnits...)
	}

	return units, nil
}

// FindByUnitType retrieves units filtered by type.
func (r *SystemdRepository) FindByUnitType(unitType string) ([]Unit, error) {
	// Don't require systemd connection for filesystem-based scanning

	var units []Unit

	// Get the unit files directory
	unitFilesDir := r.fsService.GetUnitFilesDirectory()

	// Scan for unit files of the specified type
	err := filepath.Walk(unitFilesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking on errors
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if this is a quadlet unit file of the specified type
		if !strings.HasSuffix(path, "."+unitType) {
			return nil
		}

		// Extract unit name from filename
		filename := filepath.Base(path)
		unitName := strings.TrimSuffix(filename, "."+unitType)

		// Read and parse the unit file to get more details
		unit, err := r.parseUnitFromFile(path, unitName, unitType)
		if err != nil {
			r.logger.Debug("Error parsing unit file", "path", path, "error", err)
			return nil // Continue on errors
		}

		units = append(units, unit)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking unit files directory: %w", err)
	}

	return units, nil
}

// FindByID retrieves a single unit by ID (name for systemd-based approach).
func (r *SystemdRepository) FindByID(id int64) (Unit, error) {
	// For systemd-based approach, we'll treat the ID as a hash of name+type
	// This is a compatibility method - in practice, we should use name-based lookups
	units, err := r.FindAll()
	if err != nil {
		return Unit{}, err
	}

	for _, unit := range units {
		if unit.ID == id {
			return unit, nil
		}
	}

	return Unit{}, fmt.Errorf("unit with id %d not found", id)
}

// Create creates or updates unit information (systemd-based approach doesn't store data).
func (r *SystemdRepository) Create(unit *Unit) (int64, error) {
	// In the systemd-based approach, we don't actually store anything
	// The unit information is inferred from the filesystem and systemd state
	// We just return a fake ID based on the name+type hash
	id := int64(hash(unit.Name + unit.Type))
	return id, nil
}

// Delete removes unit information (systemd-based approach doesn't store data).
func (r *SystemdRepository) Delete(_ int64) error {
	// In the systemd-based approach, we don't actually delete anything from storage
	// The actual unit file removal is handled by the compose processor
	return nil
}

// parseUnitFromFile parses a unit file and extracts unit information.
func (r *SystemdRepository) parseUnitFromFile(filePath, unitName, unitType string) (Unit, error) {
	// Read the file content
	content, err := os.ReadFile(filePath) //nolint:gosec // Safe as path is validated through filepath.Walk
	if err != nil {
		return Unit{}, fmt.Errorf("reading unit file %s: %w", filePath, err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return Unit{}, fmt.Errorf("getting file info for %s: %w", filePath, err)
	}

	// Calculate content hash
	contentHash := fs.GetContentHash(string(content))

	// Generate a consistent ID based on name and type
	id := int64(hash(unitName + unitType))

	unit := Unit{
		ID:        id,
		Name:      unitName,
		Type:      unitType,
		SHA1Hash:  contentHash,
		UpdatedAt: fileInfo.ModTime(),
	}

	return unit, nil
}

// hash generates a simple hash for consistent ID generation.
func hash(s string) uint32 {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}
