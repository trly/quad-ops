package unit

import (
	"crypto/sha1"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/db"
)

// ProcessComposeProjects processes Docker Compose projects and converts them to Podman systemd units
func ProcessComposeProjects(projects []*types.Project, force bool) error {
	dbConn, err := db.Connect()
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer dbConn.Close()

	unitRepo := NewUnitRepository(dbConn)
	
	// Track processed units to handle orphaned cleanup later
	processedUnits := make(map[string]bool)
	changedUnits := make([]QuadletUnit, 0)

	// Process each project
	for _, project := range projects {
		if config.GetConfig().Verbose {
			log.Printf("processing compose project: %s (services: %d, networks: %d, volumes: %d)", 
				project.Name, len(project.Services), len(project.Networks), len(project.Volumes))
		}

		// Process services (containers)
		for serviceName, service := range project.Services {
			if config.GetConfig().Verbose {
				log.Printf("processing service: %s", serviceName)
			}

			// Create container unit
			container := NewContainer(serviceName)
			container = container.FromComposeService(service)

			// Create the quadlet unit
			quadletUnit := QuadletUnit{
				Name: serviceName,
				Type: "container",
				Container: *container,
			}

			// Process the quadlet unit
			processUnit(unitRepo, &quadletUnit, force, processedUnits, &changedUnits)
		}

		// Process volumes
		for volumeName, volumeConfig := range project.Volumes {
			if config.GetConfig().Verbose {
				log.Printf("processing volume: %s", volumeName)
			}

			// Create volume unit
			volume := NewVolume(volumeName)
			volume = volume.FromComposeVolume(volumeName, volumeConfig)

			// Create the quadlet unit
			quadletUnit := QuadletUnit{
				Name: volumeName,
				Type: "volume",
				Volume: *volume,
			}

			// Process the quadlet unit
			processUnit(unitRepo, &quadletUnit, force, processedUnits, &changedUnits)
		}

		// Process networks
		for networkName, networkConfig := range project.Networks {
			if config.GetConfig().Verbose {
				log.Printf("processing network: %s", networkName)
			}

			// Create network unit
			network := NewNetwork(networkName)
			network = network.FromComposeNetwork(networkName, networkConfig)

			// Create the quadlet unit
			quadletUnit := QuadletUnit{
				Name: networkName,
				Type: "network",
				Network: *network,
			}

			// Process the quadlet unit
			processUnit(unitRepo, &quadletUnit, force, processedUnits, &changedUnits)
		}

		// Process secrets - note that in Podman, secrets are handled as part of containers
		// and don't need separate units like in Docker Swarm. The secret handling is already
		// implemented in the Container.FromComposeService method
	}

	// Reload systemd units if any changed
	if len(changedUnits) > 0 {
		reloadUnits(changedUnits)
	}

	// Clean up any orphaned units
	cleanupOrphanedUnits(unitRepo, processedUnits)

	return nil
}

// processUnit processes a single quadlet unit
func processUnit(unitRepo Repository, unit *QuadletUnit, force bool, processedUnits map[string]bool, changedUnits *[]QuadletUnit) error {
	// Track this unit as processed
	unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
	processedUnits[unitKey] = true

	// Generate unit content
	content := GenerateQuadletUnit(*unit, config.GetConfig().Verbose)

	// Get unit file path
	unitPath := getUnitFilePath(unit.Name, unit.Type)

	// Check if unit has changed
	if !force && !hasUnitChanged(unitPath, content) {
		return nil
	}

	// Write unit file
	if err := writeUnitFile(unitPath, content); err != nil {
		return fmt.Errorf("writing unit file for %s: %w", unit.Name, err)
	}

	// Update database
	if err := updateUnitDatabase(unitRepo, unit, content); err != nil {
		return fmt.Errorf("updating unit database for %s: %w", unit.Name, err)
	}

	// Add to changed units list
	*changedUnits = append(*changedUnits, *unit)
	return nil
}

// Helper functions extracted from the Processor struct
func getUnitFilePath(name, unitType string) string {
	return filepath.Join(config.GetConfig().QuadletDir, fmt.Sprintf("%s.%s", name, unitType))
}

func hasUnitChanged(unitPath, content string) bool {
	existingContent, err := os.ReadFile(unitPath)
	if err == nil && string(getContentHash(string(existingContent))) == string(getContentHash(content)) {
		if config.GetConfig().Verbose {
			log.Printf("unit %s unchanged, skipping", unitPath)
		}
		return false
	}
	return true
}

func writeUnitFile(unitPath, content string) error {
	if config.GetConfig().Verbose {
		log.Printf("writing quadlet unit to: %s", unitPath)
	}
	return os.WriteFile(unitPath, []byte(content), 0644)
}

func updateUnitDatabase(unitRepo Repository, unit *QuadletUnit, content string) error {
	contentHash := getContentHash(content)

	// Use default cleanup policy
	cleanupPolicy := "keep"

	_, err := unitRepo.Create(&Unit{
		Name:          unit.Name,
		Type:          unit.Type,
		SHA1Hash:      contentHash,
		CleanupPolicy: cleanupPolicy,
		CreatedAt:     time.Now(),
	})
	return err
}

func reloadUnits(changedUnits []QuadletUnit) {
	err := ReloadSystemd()
	if err != nil {
		log.Printf("error reloading systemd units: %v", err)
		return
	}

	// Wait for systemd to process the changes
	time.Sleep(2 * time.Second)

	for _, unit := range changedUnits {
		systemdUnit := unit.GetSystemdUnit()
		err := systemdUnit.Restart()
		if err != nil {
			log.Printf("error restarting unit %s: %v", unit.Name, err)
		}
	}
}

func cleanupOrphanedUnits(unitRepo Repository, processedUnits map[string]bool) error {
	dbUnits, err := unitRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching units from database: %w", err)
	}

	for _, dbUnit := range dbUnits {
		unitKey := fmt.Sprintf("%s.%s", dbUnit.Name, dbUnit.Type)
		if !processedUnits[unitKey] && (dbUnit.CleanupPolicy == "delete") {
			if config.GetConfig().Verbose {
				log.Printf("cleaning up orphaned unit %s with policy %s", unitKey, dbUnit.CleanupPolicy)
			}

			// First, stop the unit
			systemdUnit := &BaseSystemdUnit{
				Name: dbUnit.Name,
				Type: dbUnit.Type,
			}

			// Attempt to stop the unit, but continue with cleanup even if stop fails
			if err := systemdUnit.Stop(); err != nil {
				log.Printf("warning: error stopping orphaned unit %s: %v", unitKey, err)
			} else if config.GetConfig().Verbose {
				log.Printf("successfully stopped orphaned unit %s", unitKey)
			}

			// Then remove the unit file
			unitPath := getUnitFilePath(dbUnit.Name, dbUnit.Type)
			if err := os.Remove(unitPath); err != nil {
				if !os.IsNotExist(err) {
					log.Printf("error removing orphaned unit file %s: %v", unitPath, err)
				}
			} else if config.GetConfig().Verbose {
				log.Printf("removed orphaned unit file %s", unitPath)
			}

			// Finally, remove from database
			if err := unitRepo.Delete(dbUnit.ID); err != nil {
				log.Printf("error deleting unit %s from database: %v", unitKey, err)
				continue
			}

			if config.GetConfig().Verbose {
				log.Printf("successfully cleaned up orphaned unit %s", unitKey)
			}
		}
	}

	// Reload systemd after we've removed units
	if err := ReloadSystemd(); err != nil {
		log.Printf("warning: error reloading systemd after cleanup: %v", err)
	}

	return nil
}

// getContentHash calculates a SHA1 hash for content comparison
func getContentHash(content string) []byte {
	hash := sha1.New()
	hash.Write([]byte(content))
	return hash.Sum(nil)
}