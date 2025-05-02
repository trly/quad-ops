// Package unit provides quadlet unit generation and management functionality
package unit

import (
	"crypto/sha1" //nolint:gosec // Not used for security purposes, just content comparison
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/db"
)

// ProcessComposeProjects processes Docker Compose projects and converts them to Podman systemd units.
// It accepts an existing processedUnits map to track units across multiple repository calls
// and a cleanup flag to control when orphaned unit cleanup should occur.
func ProcessComposeProjects(projects []*types.Project, force bool, existingProcessedUnits map[string]bool, doCleanup bool) (map[string]bool, error) {
	dbConn, err := db.Connect()
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer func() { _ = dbConn.Close() }()

	unitRepo := NewUnitRepository(dbConn)

	// Use existing map if provided, otherwise create a new one
	processedUnits := existingProcessedUnits
	if processedUnits == nil {
		processedUnits = make(map[string]bool)
	}

	changedUnits := make([]QuadletUnit, 0)

	// Process each project
	for _, project := range projects {
		if config.GetConfig().Verbose {
			log.Printf("processing compose project: %s (services: %d, networks: %d, volumes: %d)",
				project.Name, len(project.Services), len(project.Networks), len(project.Volumes))
		}

		// Build the bidirectional dependency tree for the project
		dependencyTree := BuildServiceDependencyTree(project)

		// Process services (containers)
		for serviceName, service := range project.Services {
			if config.GetConfig().Verbose {
				log.Printf("processing service: %s", serviceName)
			}

			// Create prefixed container name using project name to enable proper DNS resolution
			// Format: <project>-<service> (e.g., myproject-db, myproject-web)
			prefixedName := fmt.Sprintf("%s-%s", project.Name, serviceName)
			container := NewContainer(prefixedName)
			container = container.FromComposeService(service, project.Name)

			// Check if we should use Podman's default naming with systemd- prefix
			// By default, Podman prefixes container hostnames with "systemd-"
			// We can override this by setting the ContainerName in the unit file
			usePodmanNames := config.GetConfig().UsePodmanDefaultNames

			// Repository-specific setting overrides global setting if present
			for _, repo := range config.GetConfig().Repositories {
				if strings.Contains(project.Name, repo.Name) && repo.UsePodmanDefaultNames != usePodmanNames {
					usePodmanNames = repo.UsePodmanDefaultNames
					break
				}
			}

			// If we don't want Podman's default names, set ContainerName to override the systemd- prefix
			if !usePodmanNames {
				container.ContainerName = prefixedName
			}

			// Always add the service name as a NetworkAlias to allow using just the service name for connections
			// This makes Docker Compose files more portable by allowing references like 'db' instead of 'quad-ops-multi-service-db'
			container.NetworkAlias = append(container.NetworkAlias, serviceName)

			// Create the quadlet unit with proper systemd configuration
			quadletUnit := QuadletUnit{
				Name:      prefixedName, // Use prefixed name for DNS resolution
				Type:      "container",
				Container: *container,
				Systemd:   SystemdConfig{},
			}

			// Apply dependency relationships (both regular and reverse)
			ApplyDependencyRelationships(&quadletUnit, serviceName, dependencyTree, project.Name)

			// Process the quadlet unit
			if err := ProcessUnit(unitRepo, &quadletUnit, force, processedUnits, &changedUnits); err != nil {
				log.Printf("Error processing unit: %v", err)
			}
		}

		// Process volumes
		for volumeName, volumeConfig := range project.Volumes {
			if config.GetConfig().Verbose {
				log.Printf("processing volume: %s", volumeName)
			}

			// Check if we should use Podman's default naming with systemd- prefix
			// By default, Podman prefixes volume names with "systemd-"
			usePodmanNames := config.GetConfig().UsePodmanDefaultNames

			// Repository-specific setting overrides global setting if present
			for _, repo := range config.GetConfig().Repositories {
				if strings.Contains(project.Name, repo.Name) && repo.UsePodmanDefaultNames != usePodmanNames {
					usePodmanNames = repo.UsePodmanDefaultNames
					break
				}
			}

			// Create prefixed volume name using project name for consistency
			prefixedName := fmt.Sprintf("%s-%s", project.Name, volumeName)
			volume := NewVolume(prefixedName)
			volume = volume.FromComposeVolume(volumeName, volumeConfig)

			// Check if we should use Podman's default naming with systemd- prefix
			if !usePodmanNames {
				volume.VolumeName = prefixedName
			}

			// Create the quadlet unit
			quadletUnit := QuadletUnit{
				Name:   prefixedName,
				Type:   "volume",
				Volume: *volume,
			}

			// Process the quadlet unit
			if err := ProcessUnit(unitRepo, &quadletUnit, force, processedUnits, &changedUnits); err != nil {
				log.Printf("Error processing volume unit: %v", err)
			}
		}

		// Process networks
		for networkName, networkConfig := range project.Networks {
			if config.GetConfig().Verbose {
				log.Printf("processing network: %s", networkName)
			}

			// Check if we should use Podman's default naming with systemd- prefix
			// By default, Podman prefixes network names with "systemd-"
			usePodmanNames := config.GetConfig().UsePodmanDefaultNames

			// Repository-specific setting overrides global setting if present
			for _, repo := range config.GetConfig().Repositories {
				if strings.Contains(project.Name, repo.Name) && repo.UsePodmanDefaultNames != usePodmanNames {
					usePodmanNames = repo.UsePodmanDefaultNames
					break
				}
			}

			// Create prefixed network name using project name for consistency
			prefixedName := fmt.Sprintf("%s-%s", project.Name, networkName)
			network := NewNetwork(prefixedName)
			network = network.FromComposeNetwork(networkName, networkConfig)

			// Check if we should use Podman's default naming with systemd- prefix
			if !usePodmanNames {
				network.NetworkName = prefixedName
			}

			// Create the quadlet unit
			quadletUnit := QuadletUnit{
				Name:    prefixedName,
				Type:    "network",
				Network: *network,
			}

			// Process the quadlet unit
			if err := ProcessUnit(unitRepo, &quadletUnit, force, processedUnits, &changedUnits); err != nil {
				log.Printf("Error processing network unit: %v", err)
			}
		}

		// Process secrets - note that in Podman, secrets are handled as part of containers
		// and don't need separate units like in Docker Swarm. The secret handling is already
		// implemented in the Container.FromComposeService method
	}

	// Reload systemd units if any changed
	if len(changedUnits) > 0 {
		// Create a map to store project dependency trees
		projectDependencyTrees := make(map[string]map[string]*ServiceDependency)

		// Store dependency trees for each project processed
		for _, project := range projects {
			projectDependencyTrees[project.Name] = BuildServiceDependencyTree(project)
		}

		// Use dependency-aware restart for changed units
		if err := RestartChangedUnits(changedUnits, projectDependencyTrees); err != nil {
			log.Printf("Error restarting changed units: %v", err)
		}
	}

	// Clean up any orphaned units only if requested
	if doCleanup {
		if err := CleanupOrphanedUnits(unitRepo, processedUnits); err != nil {
			log.Printf("Error cleaning up orphaned units: %v", err)
		}
	}

	return processedUnits, nil
}

// processUnit processes a single quadlet unit.
// ProcessUnitFunc is the function signature for processing a unit
type ProcessUnitFunc func(unitRepo Repository, unit *QuadletUnit, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) error

// CleanupOrphanedUnitsFunc is the function signature for cleaning up orphaned units
type CleanupOrphanedUnitsFunc func(unitRepo Repository, processedUnits map[string]bool) error

// WriteUnitFileFunc is the function signature for writing a unit file
type WriteUnitFileFunc func(unitPath, content string) error

// UpdateUnitDatabaseFunc is the function signature for updating the unit database
type UpdateUnitDatabaseFunc func(unitRepo Repository, unit *QuadletUnit, content string) error

// Package variables for testing
var (
	ProcessUnit          ProcessUnitFunc          = processUnit
	CleanupOrphanedUnits CleanupOrphanedUnitsFunc = cleanupOrphanedUnits
	WriteUnitFile        WriteUnitFileFunc        = writeUnitFile
	UpdateUnitDatabase   UpdateUnitDatabaseFunc   = updateUnitDatabase
)

func processUnit(unitRepo Repository, unit *QuadletUnit, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) error {
	// Track this unit as processed
	unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
	processedUnits[unitKey] = true

	// Generate unit content
	content := GenerateQuadletUnit(*unit, config.GetConfig().Verbose)

	// Get unit file path
	unitPath := getUnitFilePath(unit.Name, unit.Type)

	// Check if unit file content has changed
	hasChanged := hasUnitChanged(unitPath, content)

	// Check for potential naming conflicts due to usePodmanDefaultNames changes
	// This occurs when a unit with a different naming scheme exists
	hasNamingConflict := false
	existingUnits, err := unitRepo.FindAll()
	if err == nil {
		for _, existingUnit := range existingUnits {
			// If an existing unit with the same type exists that almost matches but differs in naming scheme,
			// this could indicate a usePodmanDefaultNames change
			if existingUnit.Type == unit.Type &&
				existingUnit.Name != unit.Name &&
				(strings.HasSuffix(existingUnit.Name, unit.Name) || strings.HasSuffix(unit.Name, existingUnit.Name)) {
				hasNamingConflict = true
				if config.GetConfig().Verbose {
					log.Printf("Detected potential naming conflict: existing=%s, new=%s",
						existingUnit.Name, unit.Name)
				}
				break
			}
		}
	}

	// If forcing update or content has changed or there's a naming conflict, write the file
	if force || hasChanged || hasNamingConflict {
		// When verbose, log that a change was detected
		if config.GetConfig().Verbose {
			if hasChanged {
				log.Printf("Unit content has changed: %s (%s)", unit.Name, unit.Type)
			} else if hasNamingConflict {
				log.Printf("Unit naming scheme has changed: %s (%s)", unit.Name, unit.Type)
			} else {
				log.Printf("Force updating unit: %s (%s)", unit.Name, unit.Type)
			}
		}

		// Write the file
		if err := WriteUnitFile(unitPath, content); err != nil {
			return fmt.Errorf("writing unit file for %s: %w", unit.Name, err)
		}

		// Update database
		if err := UpdateUnitDatabase(unitRepo, unit, content); err != nil {
			return fmt.Errorf("updating unit database for %s: %w", unit.Name, err)
		}

		// Add to changed units list for restart
		*changedUnits = append(*changedUnits, *unit)
	} else {
		// Even when the file hasn't changed, we still need to update the database
		// to ensure the unit's existence is recorded, but we don't add it to changedUnits
		if err := UpdateUnitDatabase(unitRepo, unit, content); err != nil {
			return fmt.Errorf("updating unit database for %s: %w", unit.Name, err)
		}
	}

	return nil
}

// Helper functions extracted from the Processor struct.
func getUnitFilePath(name, unitType string) string {
	return filepath.Join(config.GetConfig().QuadletDir, fmt.Sprintf("%s.%s", name, unitType))
}

func hasUnitChanged(unitPath, content string) bool {
	existingContent, err := os.ReadFile(unitPath) //nolint:gosec // Safe as path is internally constructed, not user-controlled
	if err != nil {
		// File doesn't exist or can't be read, so it has changed
		return true
	}

	// If verbose logging is enabled, print hash comparison details
	if config.GetConfig().Verbose {
		log.Printf("Existing content hash: %x", getContentHash(string(existingContent)))
		log.Printf("New content hash: %x", getContentHash(content))
	}

	// Compare the actual content directly instead of hashes
	if string(existingContent) == content {
		if config.GetConfig().Verbose {
			log.Printf("unit %s unchanged, skipping", unitPath)
		}
		return false
	}

	// Content is different
	return true
}

func writeUnitFile(unitPath, content string) error {
	if config.GetConfig().Verbose {
		log.Printf("writing quadlet unit to: %s", unitPath)
	}
	return os.WriteFile(unitPath, []byte(content), 0600)
}

func updateUnitDatabase(unitRepo Repository, unit *QuadletUnit, content string) error {
	contentHash := getContentHash(content)

	// Get repository cleanup policy from config
	cleanupPolicy := "keep" // Default

	// Check for repository-specific cleanup policy
	for _, repo := range config.GetConfig().Repositories {
		if strings.Contains(unit.Name, repo.Name) && repo.Cleanup != "" {
			cleanupPolicy = repo.Cleanup
			break
		}
	}

	// Check if the unit exists and update its cleanup policy if needed
	existingUnits, err := unitRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching existing units: %w", err)
	}

	for _, existingUnit := range existingUnits {
		if existingUnit.Name == unit.Name && existingUnit.Type == unit.Type {
			if existingUnit.CleanupPolicy != cleanupPolicy {
				if config.GetConfig().Verbose {
					log.Printf("Updating cleanup policy for %s.%s from %s to %s",
						existingUnit.Name, existingUnit.Type, existingUnit.CleanupPolicy, cleanupPolicy)
				}
			}
			break
		}
	}

	_, err = unitRepo.Create(&Unit{
		Name:          unit.Name,
		Type:          unit.Type,
		SHA1Hash:      contentHash,
		CleanupPolicy: cleanupPolicy,
		UserMode:      config.GetConfig().UserMode,
		// CreatedAt removed - no need to update timestamp every time
	})
	return err
}

// Removed reloadUnits - replaced by RestartChangedUnits in restart.go

func cleanupOrphanedUnits(unitRepo Repository, processedUnits map[string]bool) error {
	dbUnits, err := unitRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching units from database: %w", err)
	}

	for _, dbUnit := range dbUnits {
		unitKey := fmt.Sprintf("%s.%s", dbUnit.Name, dbUnit.Type)

		// Check if unit is orphaned or if there's a mode mismatch
		isOrphaned := !processedUnits[unitKey] && (dbUnit.CleanupPolicy == "delete")
		hasModeMismatch := dbUnit.UserMode != config.GetConfig().UserMode && processedUnits[unitKey]

		if isOrphaned || hasModeMismatch {
			if config.GetConfig().Verbose {
				if isOrphaned {
					log.Printf("cleaning up orphaned unit %s with policy %s", unitKey, dbUnit.CleanupPolicy)
				} else {
					log.Printf("cleaning up unit %s due to user mode mismatch: DB=%t, Current=%t",
						unitKey, dbUnit.UserMode, config.GetConfig().UserMode)
				}
			}

			// First, stop the unit
			systemdUnit := &BaseSystemdUnit{
				Name: dbUnit.Name,
				Type: dbUnit.Type,
			}

			// Attempt to stop the unit, but continue with cleanup even if stop fails
			if err := systemdUnit.Stop(); err != nil {
				log.Printf("warning: error stopping unit %s: %v", unitKey, err)
			} else if config.GetConfig().Verbose {
				log.Printf("successfully stopped unit %s", unitKey)
			}

			// Then remove the unit file
			unitPath := getUnitFilePath(dbUnit.Name, dbUnit.Type)
			if err := os.Remove(unitPath); err != nil {
				if !os.IsNotExist(err) {
					log.Printf("error removing unit file %s: %v", unitPath, err)
				}
			} else if config.GetConfig().Verbose {
				log.Printf("removed unit file %s", unitPath)
			}

			// For mode mismatches, we delete from the database, but the unit will be recreated
			// in the next processUnit call with the correct mode
			if err := unitRepo.Delete(dbUnit.ID); err != nil {
				log.Printf("error deleting unit %s from database: %v", unitKey, err)
				continue
			}

			if config.GetConfig().Verbose {
				log.Printf("successfully cleaned up unit %s", unitKey)
			}
		}
	}

	// Reload systemd after we've removed units
	if err := ReloadSystemd(); err != nil {
		log.Printf("warning: error reloading systemd after cleanup: %v", err)
	}

	return nil
}

// getContentHash calculates a SHA1 hash for content storage and change tracking.
func getContentHash(content string) []byte {
	hash := sha1.New() //nolint:gosec // Not used for security purposes, just for content tracking
	hash.Write([]byte(content))
	return hash.Sum(nil)
}
