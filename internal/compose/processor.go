// Package compose provides Docker Compose project processing functionality
package compose

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/sorting"
	"github.com/trly/quad-ops/internal/systemd"
	"github.com/trly/quad-ops/internal/unit"
)

// ProcessProjects processes Docker Compose projects and converts them to Podman systemd units.
// It accepts an existing processedUnits map to track units across multiple repository calls
// and a cleanup flag to control when orphaned unit cleanup should occur.
func ProcessProjects(projects []*types.Project, force bool, existingProcessedUnits map[string]bool, doCleanup bool) (map[string]bool, error) {
	unitRepo := repository.NewRepository()

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
	changedUnits := make([]unit.QuadletUnit, 0, estimatedCapacity)

	// Process each project
	for _, project := range projects {
		log.GetLogger().Info("Processing compose project", "project", project.Name, "services", len(project.Services), "networks", len(project.Networks), "volumes", len(project.Volumes))

		// Build the dependency graph for the project
		dependencyGraph, err := dependency.BuildServiceDependencyGraph(project)
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
		projectDependencyGraphs := make(map[string]*dependency.ServiceDependencyGraph)

		// Store dependency graphs for each project processed
		for _, project := range projects {
			graph, err := dependency.BuildServiceDependencyGraph(project)
			if err != nil {
				log.GetLogger().Error("Failed to build dependency graph for project", "project", project.Name, "error", err)
				continue
			}
			projectDependencyGraphs[project.Name] = graph
		}

		// Use dependency-aware restart for changed units
		// Convert QuadletUnit slice to systemd.UnitChange slice
		systemdUnits := make([]systemd.UnitChange, len(changedUnits))
		for i, unit := range changedUnits {
			systemdUnits[i] = systemd.UnitChange{
				Name: unit.Name,
				Type: unit.Type,
				Unit: unit.GetSystemdUnit(),
			}
		}

		if err := systemd.RestartChangedUnits(systemdUnits, projectDependencyGraphs); err != nil {
			log.GetLogger().Error("Failed to restart changed units", "error", err)
		}
	}

	return processedUnits, nil
}

// ProcessUnitFunc is the function signature for processing a single quadlet unit.
type ProcessUnitFunc func(unitRepo repository.Repository, unitItem *unit.QuadletUnit, force bool, processedUnits map[string]bool, changedUnits *[]unit.QuadletUnit) error

// CleanupOrphanedUnitsFunc is the function signature for cleaning up orphaned units.
type CleanupOrphanedUnitsFunc func(unitRepo repository.Repository, processedUnits map[string]bool) error

// WriteUnitFileFunc is the function signature for writing a unit file.
type WriteUnitFileFunc func(unitPath, content string) error

// UpdateUnitDatabaseFunc is the function signature for updating the unit database.
type UpdateUnitDatabaseFunc func(unitRepo repository.Repository, unitItem *unit.QuadletUnit, content string) error

// Package variables for testing.
var (
	ProcessUnit          ProcessUnitFunc          = processUnit
	CleanupOrphanedUnits CleanupOrphanedUnitsFunc = cleanupOrphanedUnits
	WriteUnitFile        WriteUnitFileFunc        = fs.WriteUnitFile
	UpdateUnitDatabase   UpdateUnitDatabaseFunc   = updateUnitDatabase
)

func processUnit(unitRepo repository.Repository, unitItem *unit.QuadletUnit, force bool, processedUnits map[string]bool, changedUnits *[]unit.QuadletUnit) error {
	// Track this unit as processed
	unitKey := fmt.Sprintf("%s.%s", unitItem.Name, unitItem.Type)
	processedUnits[unitKey] = true

	// Generate unit content
	content := unit.GenerateQuadletUnit(*unitItem)

	// Get unit file path
	unitPath := fs.GetUnitFilePath(unitItem.Name, unitItem.Type)

	// Check if unit file content has changed
	hasChanged := fs.HasUnitChanged(unitPath, content)

	// Check for potential naming conflicts with existing units
	// This occurs when a unit with a different naming scheme exists
	hasNamingConflict := false
	existingUnits, err := unitRepo.FindAll()
	if err == nil {
		for _, existingUnit := range existingUnits {
			// If an existing unit with the same type exists that almost matches but differs in naming scheme
			if existingUnit.Type == unitItem.Type &&
				existingUnit.Name != unitItem.Name &&
				(strings.HasSuffix(existingUnit.Name, unitItem.Name) || strings.HasSuffix(unitItem.Name, existingUnit.Name)) {
				hasNamingConflict = true
				log.GetLogger().Debug("Detected potential naming conflict", "existing", existingUnit.Name, "new", unitItem.Name)
				break
			}
		}
	}

	// If forcing update or content has changed or there's a naming conflict, write the file
	if force || hasChanged || hasNamingConflict {
		// When verbose, log that a change was detected
		if hasChanged {
			log.GetLogger().Debug("Unit content has changed", "name", unitItem.Name, "type", unitItem.Type)
		} else if hasNamingConflict {
			log.GetLogger().Debug("Unit naming scheme has changed", "name", unitItem.Name, "type", unitItem.Type)
		} else {
			log.GetLogger().Debug("Force updating unit", "name", unitItem.Name, "type", unitItem.Type)
		}

		// Write the file
		if err := fs.WriteUnitFile(unitPath, content); err != nil {
			return fmt.Errorf("writing unit file for %s: %w", unitItem.Name, err)
		}

		// Track unit (no database update needed in systemd-based approach)
		if err := updateUnitDatabase(unitRepo, unitItem, content); err != nil {
			return fmt.Errorf("tracking unit for %s: %w", unitItem.Name, err)
		}

		// Add to changed units list for restart
		*changedUnits = append(*changedUnits, *unitItem)
	} else {
		// Even when the file hasn't changed, we still track the unit
		// to ensure the unit's existence is recorded, but we don't add it to changedUnits
		if err := updateUnitDatabase(unitRepo, unitItem, content); err != nil {
			return fmt.Errorf("tracking unit for %s: %w", unitItem.Name, err)
		}
	}

	return nil
}

// Helper functions extracted from the Processor struct.

func updateUnitDatabase(unitRepo repository.Repository, unitItem *unit.QuadletUnit, content string) error {
	// In the systemd-based approach, we don't need to store unit data
	// The repository handles inferring unit information from filesystem and systemd
	// This function is kept for compatibility but doesn't perform actual database operations

	log.GetLogger().Debug("Tracking unit", "name", unitItem.Name, "type", unitItem.Type)

	// Create a unit record for compatibility (repository will handle it appropriately)
	contentHash := fs.GetContentHash(content)
	_, err := unitRepo.Create(&repository.Unit{
		Name:     unitItem.Name,
		Type:     unitItem.Type,
		SHA1Hash: contentHash,
	})
	return err
}

func cleanupOrphanedUnits(unitRepo repository.Repository, processedUnits map[string]bool) error {
	existingUnits, err := unitRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching units from filesystem: %w", err)
	}

	for _, unit := range existingUnits {
		unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)

		// Check if unit is orphaned or if there's a mode mismatch
		isOrphaned := !processedUnits[unitKey]

		if isOrphaned {
			log.GetLogger().Info("Cleaning up orphaned unit", "unit", unitKey)

			// First, stop the unit
			systemdUnit := systemd.NewBaseUnit(unit.Name, unit.Type)

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
			unitPath := fs.GetUnitFilePath(unit.Name, unit.Type)
			if err := os.Remove(unitPath); err != nil {
				if !os.IsNotExist(err) {
					log.GetLogger().Error("Failed to remove unit file", "path", unitPath, "error", err)
				}
			} else {
				log.GetLogger().Debug("Removed unit file", "path", unitPath)
			}

			// For mode mismatches, we remove from filesystem, but the unit will be recreated
			// in the next processUnit call with the correct mode
			if err := unitRepo.Delete(unit.ID); err != nil {
				log.GetLogger().Error("Failed to remove unit tracking", "unit", unitKey, "error", err)
				continue
			}

			log.GetLogger().Info("Successfully cleaned up unit", "unit", unitKey)
		}
	}

	// Reload systemd after we've removed units
	if err := systemd.ReloadSystemd(); err != nil {
		log.GetLogger().Error("Error reloading systemd after cleanup", "error", err)
	}

	return nil
}

// processServices processes all container services from a Docker Compose project.
func processServices(project *types.Project, dependencyGraph *dependency.ServiceDependencyGraph, unitRepo repository.Repository, force bool, processedUnits map[string]bool, changedUnits *[]unit.QuadletUnit) error {
	for serviceName, service := range project.Services {
		log.GetLogger().Debug("Processing service", "service", serviceName)

		prefixedName := fmt.Sprintf("%s-%s", project.Name, serviceName)

		// Process build if present
		if err := processBuildIfPresent(&service, serviceName, project, dependencyGraph, unitRepo, force, processedUnits, changedUnits); err != nil {
			return err
		}

		// Create and configure container
		container := createContainerFromService(service, prefixedName, serviceName, project)

		// Process init containers first
		initContainers, err := unit.ParseInitContainers(service)
		if err != nil {
			log.GetLogger().Error("Failed to parse init containers", "service", serviceName, "error", err)
			return err
		}

		var initUnitNames []string
		for i, initContainer := range initContainers {
			initName := fmt.Sprintf("%s-%s-init-%d", project.Name, serviceName, i)
			initContainerUnit := unit.CreateInitContainerUnit(initContainer, initName, container)

			// Create init quadlet unit
			initQuadletUnit := createInitQuadletUnit(initName, initContainerUnit)

			// Apply dependency relationships to init container (same as main service)
			if err := unit.ApplyDependencyRelationships(&initQuadletUnit, serviceName, dependencyGraph, project.Name); err != nil {
				log.GetLogger().Error("Failed to apply dependency relationships to init container", "init", initName, "error", err)
			}

			// Process the init unit
			if err := processUnit(unitRepo, &initQuadletUnit, force, processedUnits, changedUnits); err != nil {
				return err
			}

			initUnitNames = append(initUnitNames, initName+".service")
		}

		// Create main container quadlet unit
		quadletUnit := createQuadletUnit(prefixedName, container)

		// Add init container dependencies to main container
		for _, initUnitName := range initUnitNames {
			quadletUnit.Systemd.After = append(quadletUnit.Systemd.After, initUnitName)
			quadletUnit.Systemd.Requires = append(quadletUnit.Systemd.Requires, initUnitName)
		}

		// Apply dependencies and process
		if err := finishProcessingService(&quadletUnit, serviceName, dependencyGraph, project.Name, unitRepo, force, processedUnits, changedUnits); err != nil {
			return err
		}
	}
	return nil
}

func processBuildIfPresent(service *types.ServiceConfig, serviceName string, project *types.Project, dependencyGraph *dependency.ServiceDependencyGraph, unitRepo repository.Repository, force bool, processedUnits map[string]bool, changedUnits *[]unit.QuadletUnit) error {
	if service.Build == nil {
		return nil
	}

	log.GetLogger().Debug("Processing build for service", "service", serviceName)

	buildUnitName := fmt.Sprintf("%s-%s-build", project.Name, serviceName)
	build := unit.NewBuild(buildUnitName)
	build = build.FromComposeBuild(*service.Build, *service, project.Name)

	// Configure build context
	if build.SetWorkingDirectory == "repo" {
		build.SetWorkingDirectory = project.WorkingDir
		log.GetLogger().Debug("Setting build context to project working directory",
			"service", serviceName, "context", build.SetWorkingDirectory)
	}

	// Handle production target
	if err := handleProductionTarget(build, serviceName, project.WorkingDir); err != nil {
		log.GetLogger().Debug("Error checking Dockerfile for production target", "error", err)
	}

	buildQuadletUnit := unit.QuadletUnit{
		Name:  buildUnitName,
		Type:  "build",
		Build: *build,
		Systemd: unit.SystemdConfig{
			RemainAfterExit: true,
		},
	}

	// Process the build unit
	if err := ProcessUnit(unitRepo, &buildQuadletUnit, force, processedUnits, changedUnits); err != nil {
		log.GetLogger().Error("Failed to process build unit", "error", err)
	}

	// Update service image and dependencies
	service.Image = fmt.Sprintf("%s.build", buildUnitName)
	return addBuildDependency(dependencyGraph, serviceName)
}

func handleProductionTarget(build *unit.Build, serviceName, workingDir string) error {
	if build.Target != "production" {
		return nil
	}

	// Use the more robust path validation that handles filepath.Clean internally
	validDockerfilePath, err := sorting.ValidatePathWithinBase("Dockerfile", workingDir)
	if err != nil {
		return fmt.Errorf("invalid dockerfile path for service %s: %w", serviceName, err)
	}

	dockerfilePath := validDockerfilePath

	if _, err := os.Stat(dockerfilePath); err != nil {
		return err
	}

	content, err := os.ReadFile(dockerfilePath) //nolint:gosec
	if err != nil {
		return err
	}

	if !strings.Contains(string(content), "FROM ") || !strings.Contains(string(content), " as production") {
		build.Target = ""
		log.GetLogger().Debug("Removing target='production' as it doesn't exist in Dockerfile", "service", serviceName)
	}
	return nil
}

func addBuildDependency(dependencyGraph *dependency.ServiceDependencyGraph, serviceName string) error {
	buildName := fmt.Sprintf("%s-build", serviceName)
	if err := dependencyGraph.AddService(buildName); err != nil {
		log.GetLogger().Debug("Build service already exists in dependency graph", "service", buildName)
	}
	if err := dependencyGraph.AddDependency(serviceName, buildName); err != nil {
		log.GetLogger().Error("Failed to add build dependency", "service", serviceName, "dependency", buildName, "error", err)
		return err
	}
	return nil
}

func createContainerFromService(service types.ServiceConfig, prefixedName, serviceName string, project *types.Project) *unit.Container {
	container := unit.NewContainer(prefixedName)
	container = container.FromComposeService(service, project)

	// Add environment files
	addEnvironmentFiles(container, serviceName, project.WorkingDir)

	// Configure container naming
	configureContainerNaming(container, prefixedName, serviceName)

	return container
}

func addEnvironmentFiles(container *unit.Container, serviceName, workingDir string) {
	if workingDir == "" {
		return
	}

	// General .env file
	generalEnvFile := fmt.Sprintf("%s/.env", workingDir)
	if _, err := os.Stat(generalEnvFile); err == nil {
		log.GetLogger().Debug("Adding general .env file to container unit", "service", serviceName, "file", generalEnvFile)
		container.EnvironmentFile = append(container.EnvironmentFile, generalEnvFile)
	}

	// Service-specific .env files
	possibleEnvFiles := []string{
		fmt.Sprintf("%s/.env.%s", workingDir, serviceName),
		fmt.Sprintf("%s/%s.env", workingDir, serviceName),
		fmt.Sprintf("%s/env/%s.env", workingDir, serviceName),
		fmt.Sprintf("%s/envs/%s.env", workingDir, serviceName),
	}

	for _, envFilePath := range possibleEnvFiles {
		if _, err := os.Stat(envFilePath); err == nil {
			log.GetLogger().Debug("Found service-specific environment file", "service", serviceName, "file", envFilePath)
			container.EnvironmentFile = append(container.EnvironmentFile, envFilePath)
		}
	}
}

func configureContainerNaming(container *unit.Container, prefixedName string, serviceName string) {
	// Use quad-ops preferred naming (no systemd- prefix)
	container.ContainerName = prefixedName

	// Add service name as NetworkAlias for portability
	container.NetworkAlias = append(container.NetworkAlias, serviceName)

	// Add custom hostname as NetworkAlias if different from service name
	if container.HostName != "" && container.HostName != serviceName {
		container.NetworkAlias = append(container.NetworkAlias, container.HostName)
	}
}

func createQuadletUnit(prefixedName string, container *unit.Container) unit.QuadletUnit {
	systemdConfig := unit.SystemdConfig{}

	if container.RestartPolicy != "" {
		systemdConfig.RestartPolicy = container.RestartPolicy
	}

	return unit.QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *container,
		Systemd:   systemdConfig,
	}
}

func createInitQuadletUnit(prefixedName string, container *unit.Container) unit.QuadletUnit {
	systemdConfig := unit.SystemdConfig{
		Type:            "oneshot",
		RemainAfterExit: true,
	}

	return unit.QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *container,
		Systemd:   systemdConfig,
	}
}

func finishProcessingService(quadletUnit *unit.QuadletUnit, serviceName string, dependencyGraph *dependency.ServiceDependencyGraph, projectName string, unitRepo repository.Repository, force bool, processedUnits map[string]bool, changedUnits *[]unit.QuadletUnit) error {
	// Apply dependency relationships
	if err := unit.ApplyDependencyRelationships(quadletUnit, serviceName, dependencyGraph, projectName); err != nil {
		log.GetLogger().Error("Failed to apply dependency relationships", "service", serviceName, "error", err)
	}

	// Process the quadlet unit
	if err := ProcessUnit(unitRepo, quadletUnit, force, processedUnits, changedUnits); err != nil {
		log.GetLogger().Error("Failed to process unit", "error", err)
		return err
	}
	return nil
}

// processVolumes processes all volumes from a Docker Compose project.
func processVolumes(project *types.Project, unitRepo repository.Repository, force bool, processedUnits map[string]bool, changedUnits *[]unit.QuadletUnit) error {
	for volumeName, volumeConfig := range project.Volumes {
		log.GetLogger().Debug("Processing volume", "volume", volumeName)

		// Skip external volumes - they are managed externally and should not be created by quad-ops
		if bool(volumeConfig.External) {
			log.GetLogger().Debug("Skipping external volume", "volume", volumeName)
			continue
		}

		// Create prefixed volume name using project name for consistency
		prefixedName := fmt.Sprintf("%s-%s", project.Name, volumeName)
		volume := unit.NewVolume(prefixedName)
		volume = volume.FromComposeVolume(volumeName, volumeConfig)

		// Use quad-ops preferred naming (no systemd- prefix)
		volume.VolumeName = prefixedName

		// Create the quadlet unit
		quadletUnit := unit.QuadletUnit{
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
func processNetworks(project *types.Project, unitRepo repository.Repository, force bool, processedUnits map[string]bool, changedUnits *[]unit.QuadletUnit) error {
	for networkName, networkConfig := range project.Networks {
		log.GetLogger().Debug("Processing network", "network", networkName)

		// Skip external networks - they are managed externally and should not be created by quad-ops
		if bool(networkConfig.External) {
			log.GetLogger().Debug("Skipping external network", "network", networkName)
			continue
		}

		// Create prefixed network name using project name for consistency
		prefixedName := fmt.Sprintf("%s-%s", project.Name, networkName)
		network := unit.NewNetwork(prefixedName)
		network = network.FromComposeNetwork(networkName, networkConfig)

		// Use quad-ops preferred naming (no systemd- prefix)
		network.NetworkName = prefixedName

		// Create the quadlet unit
		quadletUnit := unit.QuadletUnit{
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
