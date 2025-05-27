// Package unit provides quadlet unit generation and management functionality
package unit

import (
	"crypto/sha1" //nolint:gosec // Not used for security purposes, just content comparison
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/db"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/repository"
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

	unitRepo := repository.NewRepository(dbConn)

	// Use existing map if provided, otherwise create a new one
	processedUnits := existingProcessedUnits
	if processedUnits == nil {
		processedUnits = make(map[string]bool)
	}

	// Estimate total capacity for all projects (services + networks + volumes + potential builds)
	estimatedCapacity := 0
	for _, project := range projects {
		estimatedCapacity += len(project.Services) + len(project.Networks) + len(project.Volumes) + len(project.Services) // +services again for potential builds
	}
	changedUnits := make([]QuadletUnit, 0, estimatedCapacity)

	// Process each project
	for _, project := range projects {
		log.GetLogger().Info("Processing compose project", "project", project.Name, "services", len(project.Services), "networks", len(project.Networks), "volumes", len(project.Volumes))

		// Build the dependency graph for the project
		dependencyGraph, err := BuildServiceDependencyGraph(project)
		if err != nil {
			return processedUnits, fmt.Errorf("failed to build dependency graph for project %s: %w", project.Name, err)
		}

		// Process services (containers)
		if err := processServices(project, dependencyGraph, unitRepo, force, processedUnits, &changedUnits); err != nil {
			log.GetLogger().Error("Failed to process services", "error", err)
		}

		// Process volumes
		if err := processVolumes(project, unitRepo, force, processedUnits, &changedUnits); err != nil {
			log.GetLogger().Error("Failed to process volumes", "error", err)
		}

		// Process networks
		if err := processNetworks(project, unitRepo, force, processedUnits, &changedUnits); err != nil {
			log.GetLogger().Error("Failed to process networks", "error", err)
		}

		// Process secrets - note that in Podman, secrets are handled as part of containers
		// and don't need separate units like in Docker Swarm. The secret handling is already
		// implemented in the Container.FromComposeService method
	}

	// Clean up any orphaned units BEFORE restarting changed units to avoid dependency conflicts
	if doCleanup {
		if err := CleanupOrphanedUnits(unitRepo, processedUnits); err != nil {
			log.GetLogger().Error("Failed to clean up orphaned units", "error", err)
		}
		// Wait for systemd to fully process unit removals before proceeding with restarts
		time.Sleep(1 * time.Second)
	}

	// Reload systemd units if any changed
	if len(changedUnits) > 0 {
		// Create a map to store project dependency graphs
		projectDependencyGraphs := make(map[string]*ServiceDependencyGraph)

		// Store dependency graphs for each project processed
		for _, project := range projects {
			graph, err := BuildServiceDependencyGraph(project)
			if err != nil {
				log.GetLogger().Error("Failed to build dependency graph for project", "project", project.Name, "error", err)
				continue
			}
			projectDependencyGraphs[project.Name] = graph
		}

		// Use dependency-aware restart for changed units
		if err := RestartChangedUnits(changedUnits, projectDependencyGraphs); err != nil {
			log.GetLogger().Error("Failed to restart changed units", "error", err)
		}
	}

	return processedUnits, nil
}

// ProcessUnitFunc is the function signature for processing a single quadlet unit.
type ProcessUnitFunc func(unitRepo repository.Repository, unit *QuadletUnit, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) error

// CleanupOrphanedUnitsFunc is the function signature for cleaning up orphaned units.
type CleanupOrphanedUnitsFunc func(unitRepo repository.Repository, processedUnits map[string]bool) error

// WriteUnitFileFunc is the function signature for writing a unit file.
type WriteUnitFileFunc func(unitPath, content string) error

// UpdateUnitDatabaseFunc is the function signature for updating the unit database.
type UpdateUnitDatabaseFunc func(unitRepo repository.Repository, unit *QuadletUnit, content string) error

// Package variables for testing.
var (
	ProcessUnit          ProcessUnitFunc          = processUnit
	CleanupOrphanedUnits CleanupOrphanedUnitsFunc = cleanupOrphanedUnits
	WriteUnitFile        WriteUnitFileFunc        = writeUnitFile
	UpdateUnitDatabase   UpdateUnitDatabaseFunc   = updateUnitDatabase
)

func processUnit(unitRepo repository.Repository, unit *QuadletUnit, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) error {
	// Track this unit as processed
	unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
	processedUnits[unitKey] = true

	// Generate unit content
	content := GenerateQuadletUnit(*unit)

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
				log.GetLogger().Debug("Detected potential naming conflict", "existing", existingUnit.Name, "new", unit.Name)
				break
			}
		}
	}

	// If forcing update or content has changed or there's a naming conflict, write the file
	if force || hasChanged || hasNamingConflict {
		// When verbose, log that a change was detected
		if hasChanged {
			log.GetLogger().Debug("Unit content has changed", "name", unit.Name, "type", unit.Type)
		} else if hasNamingConflict {
			log.GetLogger().Debug("Unit naming scheme has changed", "name", unit.Name, "type", unit.Type)
		} else {
			log.GetLogger().Debug("Force updating unit", "name", unit.Name, "type", unit.Type)
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
	log.GetLogger().Debug("Content hash comparison",
		"existing", fmt.Sprintf("%x", getContentHash(string(existingContent))),
		"new", fmt.Sprintf("%x", getContentHash(content)))

	// Compare the actual content directly instead of hashes
	if string(existingContent) == content {
		log.GetLogger().Debug("Unit unchanged, skipping", "path", unitPath)
		return false
	}

	// Content is different
	return true
}

func writeUnitFile(unitPath, content string) error {
	log.GetLogger().Debug("Writing quadlet unit", "path", unitPath)

	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(unitPath), 0750); err != nil {
		return fmt.Errorf("failed to create quadlet directory: %w", err)
	}

	return os.WriteFile(unitPath, []byte(content), 0600)
}

func updateUnitDatabase(unitRepo repository.Repository, unit *QuadletUnit, content string) error {
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
				log.GetLogger().Debug("Updating cleanup policy", "name", existingUnit.Name, "type", existingUnit.Type, "old", existingUnit.CleanupPolicy, "new", cleanupPolicy)
			}
			break
		}
	}

	_, err = unitRepo.Create(&repository.Unit{
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

func cleanupOrphanedUnits(unitRepo repository.Repository, processedUnits map[string]bool) error {
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
			if isOrphaned {
				log.GetLogger().Info("Cleaning up orphaned unit", "unit", unitKey, "policy", dbUnit.CleanupPolicy)
			} else {
				log.GetLogger().Info("Cleaning up unit due to user mode mismatch", "unit", unitKey, "dbMode", dbUnit.UserMode, "currentMode", config.GetConfig().UserMode)
			}

			// First, stop the unit
			systemdUnit := &BaseSystemdUnit{
				Name: dbUnit.Name,
				Type: dbUnit.Type,
			}

			// Attempt to stop the unit, but continue with cleanup even if stop fails
			if err := systemdUnit.Stop(); err != nil {
				log.GetLogger().Warn("Error stopping unit during cleanup", "unit", unitKey, "error", err)
			} else {
				log.GetLogger().Debug("Successfully stopped unit during cleanup", "unit", unitKey)
			}

			// Note: ResetFailed is not called during cleanup because:
			// 1. We're removing the unit file entirely, so the failed state becomes irrelevant
			// 2. systemd automatically clears the unit's state when it's unloaded
			// 3. Calling ResetFailed on units being removed causes warnings about "Unit not loaded"

			// Remove the unit file
			unitPath := getUnitFilePath(dbUnit.Name, dbUnit.Type)
			if err := os.Remove(unitPath); err != nil {
				if !os.IsNotExist(err) {
					log.GetLogger().Error("Failed to remove unit file", "path", unitPath, "error", err)
				}
			} else {
				log.GetLogger().Debug("Removed unit file", "path", unitPath)
			}

			// For mode mismatches, we delete from the database, but the unit will be recreated
			// in the next processUnit call with the correct mode
			if err := unitRepo.Delete(dbUnit.ID); err != nil {
				log.GetLogger().Error("Failed to delete unit from database", "unit", unitKey, "error", err)
				continue
			}

			log.GetLogger().Info("Successfully cleaned up unit", "unit", unitKey)
		}
	}

	// Reload systemd after we've removed units
	if err := ReloadSystemd(); err != nil {
		log.GetLogger().Error("Error reloading systemd after cleanup", "error", err)
	}

	return nil
}

// getContentHash calculates a SHA1 hash for content storage and change tracking.
func getContentHash(content string) []byte {
	hash := sha1.New() //nolint:gosec // Not used for security purposes, just for content tracking
	hash.Write([]byte(content))
	return hash.Sum(nil)
}

// processServices processes all container services from a Docker Compose project.
func processServices(project *types.Project, dependencyGraph *ServiceDependencyGraph, unitRepo repository.Repository, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) error {
	for serviceName, service := range project.Services {
		log.GetLogger().Debug("Processing service", "service", serviceName)

		// Create prefixed container name using project name to enable proper DNS resolution
		// Format: <project>-<service> (e.g., myproject-db, myproject-web)
		prefixedName := fmt.Sprintf("%s-%s", project.Name, serviceName)

		// Check if service has a build section first
		if service.Build != nil {
			log.GetLogger().Debug("Processing build for service", "service", serviceName)

			// Create a build unit with the same prefixed name
			buildUnitName := fmt.Sprintf("%s-%s-build", project.Name, serviceName)
			build := NewBuild(buildUnitName)
			build = build.FromComposeBuild(*service.Build, service, project.Name)

			// Create the quadlet unit for the build
			// If build context is marked as "repo", replace it with the actual project working directory
			if build.SetWorkingDirectory == "repo" {
				// Use the project's working directory as the build context
				build.SetWorkingDirectory = project.WorkingDir
				log.GetLogger().Debug("Setting build context to project working directory",
					"service", serviceName, "context", build.SetWorkingDirectory)
			}

			// Remove the Target field if it's pointing to 'production' to prevent errors
			if build.Target == "production" {
				// Check if target stage exists in Dockerfile
				dockerfilePath := filepath.Join(project.WorkingDir, "Dockerfile")
				if _, err := os.Stat(dockerfilePath); err == nil {
					content, err := os.ReadFile(dockerfilePath) //nolint:gosec // Safe as path comes from project.WorkingDir, not user input
					if err == nil {
						if !strings.Contains(string(content), "FROM ") || !strings.Contains(string(content), " as production") {
							build.Target = ""
							log.GetLogger().Debug("Removing target='production' as it doesn't exist in Dockerfile",
								"service", serviceName)
						}
					}
				}
			}

			buildQuadletUnit := QuadletUnit{
				Name:  buildUnitName,
				Type:  "build",
				Build: *build,
				Systemd: SystemdConfig{
					// Ensure the build completes before the container starts
					RemainAfterExit: true,
				},
			}

			// Process the build unit
			if err := ProcessUnit(unitRepo, &buildQuadletUnit, force, processedUnits, changedUnits); err != nil {
				log.GetLogger().Error("Failed to process build unit", "error", err)
			}

			// Update the service image to reference the build unit
			service.Image = fmt.Sprintf("%s.build", buildUnitName)

			// Set up the dependency relationship between the container and the build
			// This ensures that the build completes before the container starts
			// Add build unit as a dependency for this service
			// Use the bare service name with -build suffix as the dependency name
			// This needs to match how dependency resolution works in ApplyDependencyRelationships
			buildName := fmt.Sprintf("%s-build", serviceName)
			if err := dependencyGraph.AddService(buildName); err != nil {
				log.GetLogger().Debug("Build service already exists in dependency graph", "service", buildName)
			}
			if err := dependencyGraph.AddDependency(serviceName, buildName); err != nil {
				log.GetLogger().Error("Failed to add build dependency", "service", serviceName, "dependency", buildName, "error", err)
			}
		}

		container := NewContainer(prefixedName)
		container = container.FromComposeService(service, project.Name)

		// Check for environment files in the project directory
		if project.WorkingDir != "" {
			// First check for general .env file
			generalEnvFile := fmt.Sprintf("%s/.env", project.WorkingDir)
			if _, err := os.Stat(generalEnvFile); err == nil {
				log.GetLogger().Debug("Adding general .env file to container unit", "service", serviceName, "file", generalEnvFile)
				container.EnvironmentFile = append(container.EnvironmentFile, generalEnvFile)
			}

			// Look for service-specific .env files with various naming patterns
			possibleEnvFiles := []string{
				fmt.Sprintf("%s/.env.%s", project.WorkingDir, serviceName),
				fmt.Sprintf("%s/%s.env", project.WorkingDir, serviceName),
				fmt.Sprintf("%s/env/%s.env", project.WorkingDir, serviceName),
				fmt.Sprintf("%s/envs/%s.env", project.WorkingDir, serviceName),
			}

			for _, envFilePath := range possibleEnvFiles {
				// Check if file exists
				if _, err := os.Stat(envFilePath); err == nil {
					log.GetLogger().Debug("Found service-specific environment file", "service", serviceName, "file", envFilePath)
					container.EnvironmentFile = append(container.EnvironmentFile, envFilePath)
				}
			}
		}

		// Check if we should use Podman's default naming with systemd- prefix
		usePodmanNames := getUsePodmanNames(project.Name)

		// If we don't want Podman's default names, set ContainerName to override the systemd- prefix
		if !usePodmanNames {
			container.ContainerName = prefixedName
		}

		// Always add the service name as a NetworkAlias to allow using just the service name for connections
		// This makes Docker Compose files more portable by allowing references like 'db' instead of 'quad-ops-multi-service-db'
		container.NetworkAlias = append(container.NetworkAlias, serviceName)

		// Also add custom hostname as a NetworkAlias if specified in the service
		if container.HostName != "" && container.HostName != serviceName {
			container.NetworkAlias = append(container.NetworkAlias, container.HostName)
		}

		// Create the quadlet unit with proper systemd configuration
		systemdConfig := SystemdConfig{}

		// Apply restart policy if set in the container
		if container.RestartPolicy != "" {
			systemdConfig.RestartPolicy = container.RestartPolicy
		}

		quadletUnit := QuadletUnit{
			Name:      prefixedName, // Use prefixed name for DNS resolution
			Type:      "container",
			Container: *container,
			Systemd:   systemdConfig,
		}

		// Apply dependency relationships (both regular and reverse)
		if err := ApplyDependencyRelationships(&quadletUnit, serviceName, dependencyGraph, project.Name); err != nil {
			log.GetLogger().Error("Failed to apply dependency relationships", "service", serviceName, "error", err)
		}

		// Process the quadlet unit
		if err := ProcessUnit(unitRepo, &quadletUnit, force, processedUnits, changedUnits); err != nil {
			log.GetLogger().Error("Failed to process unit", "error", err)
		}
	}
	return nil
}

// processVolumes processes all volumes from a Docker Compose project.
func processVolumes(project *types.Project, unitRepo repository.Repository, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) error {
	for volumeName, volumeConfig := range project.Volumes {
		log.GetLogger().Debug("Processing volume", "volume", volumeName)

		// Check if we should use Podman's default naming with systemd- prefix
		usePodmanNames := getUsePodmanNames(project.Name)

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
		if err := ProcessUnit(unitRepo, &quadletUnit, force, processedUnits, changedUnits); err != nil {
			log.GetLogger().Error("Failed to process volume unit", "error", err)
		}
	}
	return nil
}

// processNetworks processes all networks from a Docker Compose project.
func processNetworks(project *types.Project, unitRepo repository.Repository, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) error {
	for networkName, networkConfig := range project.Networks {
		log.GetLogger().Debug("Processing network", "network", networkName)

		// Check if we should use Podman's default naming with systemd- prefix
		usePodmanNames := getUsePodmanNames(project.Name)

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
		if err := ProcessUnit(unitRepo, &quadletUnit, force, processedUnits, changedUnits); err != nil {
			log.GetLogger().Error("Failed to process network unit", "error", err)
		}
	}
	return nil
}

// getUsePodmanNames determines whether to use Podman's default naming scheme based on config and repository settings.
func getUsePodmanNames(projectName string) bool {
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
