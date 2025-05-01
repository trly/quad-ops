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
func ProcessComposeProjects(projects []*types.Project, force bool, processedUnits map[string]bool) error {
	dbConn, err := db.Connect()
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer func() { _ = dbConn.Close() }()

	unitRepo := NewUnitRepository(dbConn)

	// If processedUnits is nil, create a new map
	cleanupUnits := processedUnits == nil
	if cleanupUnits {
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

		// Process components by type
		processServicesInProject(project, dependencyTree, unitRepo, force, processedUnits, &changedUnits)
		processVolumesInProject(project, unitRepo, force, processedUnits, &changedUnits)
		processNetworksInProject(project, unitRepo, force, processedUnits, &changedUnits)
	}

	// Process changed units and cleanup if needed
	return handleChangesAndCleanup(projects, unitRepo, changedUnits, cleanupUnits, processedUnits)
}

// processUnit processes a single quadlet unit.
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
		if err := writeUnitFile(unitPath, content); err != nil {
			return fmt.Errorf("writing unit file for %s: %w", unit.Name, err)
		}

		// Update database
		if err := updateUnitDatabase(unitRepo, unit, content); err != nil {
			return fmt.Errorf("updating unit database for %s: %w", unit.Name, err)
		}

		// Add to changed units list for restart
		*changedUnits = append(*changedUnits, *unit)
	} else {
		// Even when the file hasn't changed, we still need to update the database
		// to ensure the unit's existence is recorded, but we don't add it to changedUnits
		if err := updateUnitDatabase(unitRepo, unit, content); err != nil {
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

	// Find repository ID for this unit
	var repositoryID int64

	// First check if the unit already exists and has a repository ID
	existingUnits, err := unitRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching existing units: %w", err)
	}

	// Try to find matching repository based on unit name
	dbConn, err := db.Connect()
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer func() { _ = dbConn.Close() }()

	repoRepo := db.NewRepositoryRepository(dbConn)

	// Find all repositories to match against
	repos, err := repoRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching repositories: %w", err)
	}

	// Find the longest matching repository name prefix
	var bestMatch db.Repository
	var bestMatchLen int

	for _, repo := range repos {
		if strings.HasPrefix(unit.Name, repo.Name+"-") && len(repo.Name) > bestMatchLen {
			bestMatch = repo
			bestMatchLen = len(repo.Name)
			// We'll set the cleanup policy later when checking if CleanupPolicy.Valid is true
		}
	}

	if bestMatchLen > 0 {
		repositoryID = bestMatch.ID
		// Use cleanup policy from repository if it's valid
		if bestMatch.CleanupPolicy.Valid {
			cleanupPolicy = bestMatch.CleanupPolicy.String
		}
	}

	// Check if the unit exists and update its cleanup policy if needed
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
		RepositoryID:  repositoryID,
		// CreatedAt removed - no need to update timestamp every time
	})
	return err
}

// Removed reloadUnits - replaced by RestartChangedUnits in restart.go

// CleanupOrphanedUnits cleans up units that are no longer in use or belong to removed repositories.
// This function is exported so it can be called directly from the sync command.
func CleanupOrphanedUnits(unitRepo Repository, processedUnits map[string]bool) error {
	dbUnits, err := unitRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching units from database: %w", err)
	}

	// Get active repository IDs
	dbConn, err := db.Connect()
	if err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}
	defer func() { _ = dbConn.Close() }()

	repoRepo := db.NewRepositoryRepository(dbConn)

	// Make sure repositories in database match config
	if err := repoRepo.SyncFromConfig(); err != nil {
		return fmt.Errorf("error syncing repositories from config: %w", err)
	}

	// Get all active repositories after sync
	activeRepos, err := repoRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching repositories: %w", err)
	}

	activeRepoIDs := make(map[int64]bool)
	for _, repo := range activeRepos {
		activeRepoIDs[repo.ID] = true
	}

	for _, dbUnit := range dbUnits {
		unitKey := fmt.Sprintf("%s.%s", dbUnit.Name, dbUnit.Type)

		// Check if unit is orphaned, has no repository association, or if there's a mode mismatch
		isOrphaned := !processedUnits[unitKey] && (dbUnit.CleanupPolicy == "delete")
		hasModeMismatch := dbUnit.UserMode != config.GetConfig().UserMode && processedUnits[unitKey]
		repoRemoved := dbUnit.RepositoryID != 0 && !activeRepoIDs[dbUnit.RepositoryID] && (dbUnit.CleanupPolicy == "delete")

		if isOrphaned || hasModeMismatch || repoRemoved {
			if config.GetConfig().Verbose {
				if isOrphaned {
					log.Printf("cleaning up orphaned unit %s with policy %s", unitKey, dbUnit.CleanupPolicy)
				} else if repoRemoved {
					// Get the repository name for logging
					log.Printf("cleaning up unit %s because its repository was removed from config", unitKey)
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

// processServicesInProject processes all services (containers) in a project.
func processServicesInProject(project *types.Project, dependencyTree map[string]*ServiceDependency, unitRepo Repository, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) {
	for serviceName, service := range project.Services {
		if config.GetConfig().Verbose {
			log.Printf("processing service: %s", serviceName)
		}

		// Create prefixed container name using project name to enable proper DNS resolution
		prefixedName := fmt.Sprintf("%s-%s", project.Name, serviceName)
		container := NewContainer(prefixedName)
		container = container.FromComposeService(service, project.Name)

		// Check if we should use Podman's default naming with systemd- prefix
		usePodmanNames := getPodmanNamingPreference(project.Name)

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
		if err := processUnit(unitRepo, &quadletUnit, force, processedUnits, changedUnits); err != nil {
			log.Printf("Error processing unit: %v", err)
		}
	}
}

// processVolumesInProject processes all volumes in a project.
func processVolumesInProject(project *types.Project, unitRepo Repository, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) {
	for volumeName, volumeConfig := range project.Volumes {
		if config.GetConfig().Verbose {
			log.Printf("processing volume: %s", volumeName)
		}

		// Check if we should use Podman's default naming
		usePodmanNames := getPodmanNamingPreference(project.Name)

		// Create prefixed volume name using project name for consistency
		prefixedName := fmt.Sprintf("%s-%s", project.Name, volumeName)
		volume := NewVolume(prefixedName)
		volume = volume.FromComposeVolume(volumeName, volumeConfig)

		// Set explicit volume name if not using Podman defaults
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
		if err := processUnit(unitRepo, &quadletUnit, force, processedUnits, changedUnits); err != nil {
			log.Printf("Error processing volume unit: %v", err)
		}
	}
}

// processNetworksInProject processes all networks in a project.
func processNetworksInProject(project *types.Project, unitRepo Repository, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) {
	for networkName, networkConfig := range project.Networks {
		if config.GetConfig().Verbose {
			log.Printf("processing network: %s", networkName)
		}

		// Check if we should use Podman's default naming
		usePodmanNames := getPodmanNamingPreference(project.Name)

		// Create prefixed network name using project name for consistency
		prefixedName := fmt.Sprintf("%s-%s", project.Name, networkName)
		network := NewNetwork(prefixedName)
		network = network.FromComposeNetwork(networkName, networkConfig)

		// Set explicit network name if not using Podman defaults
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
		if err := processUnit(unitRepo, &quadletUnit, force, processedUnits, changedUnits); err != nil {
			log.Printf("Error processing network unit: %v", err)
		}
	}
}

// getPodmanNamingPreference determines if Podman's default naming scheme should be used for a project.
func getPodmanNamingPreference(projectName string) bool {
	usePodmanNames := config.GetConfig().UsePodmanDefaultNames

	// Repository-specific setting overrides global setting if present
	for _, repo := range config.GetConfig().Repositories {
		if strings.Contains(projectName, repo.Name) && repo.UsePodmanDefaultNames != usePodmanNames {
			usePodmanNames = repo.UsePodmanDefaultNames
			break
		}
	}

	return usePodmanNames
}

// handleChangesAndCleanup manages restarting changed units and cleaning up orphaned units.
func handleChangesAndCleanup(projects []*types.Project, unitRepo Repository, changedUnits []QuadletUnit, shouldCleanup bool, processedUnits map[string]bool) error {
	// Handle changed units - reload/restart as needed
	if len(changedUnits) > 0 {
		projectDependencyTrees := make(map[string]map[string]*ServiceDependency)
		for _, project := range projects {
			projectDependencyTrees[project.Name] = BuildServiceDependencyTree(project)
		}

		if err := RestartChangedUnits(changedUnits, projectDependencyTrees); err != nil {
			log.Printf("Error restarting changed units: %v", err)
		}
	}

	// Handle cleanup if needed
	// Always run cleanup to handle repository removal detection
	if shouldCleanup {
		if err := CleanupOrphanedUnits(unitRepo, processedUnits); err != nil {
			log.Printf("Error cleaning up orphaned units: %v", err)
		}
	}

	return nil
}
