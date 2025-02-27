package quadlet

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"quad-ops/internal/config"
	"quad-ops/internal/db"
	dbUnit "quad-ops/internal/db/model"
	"quad-ops/internal/git"
	"quad-ops/internal/systemd"
	"time"

	"gopkg.in/yaml.v3"
)

// ProcessManifests processes all YAML manifests from the given repository
func ProcessManifests(repo *git.Repository, cfg config.Config, force bool) error {
	manifestsPath := getManifestsPath(repo)

	if cfg.Verbose {
		log.Printf("processing manifests from repository: %s at path: %s", repo.URL, manifestsPath)
		log.Printf("output directory: %s", cfg.QuadletDir)
	}

	// Connect to the database
	dbConn, err := db.Connect(&cfg)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer dbConn.Close()

	unitRepo := db.NewUnitRepository(dbConn)

	files, err := findYamlFiles(manifestsPath)
	if err != nil {
		return err
	}

	if cfg.Verbose {
		log.Printf("found %d YAML files in manifests directory and subdirectories", len(files))
	}

	processedUnits, err := processYamlFiles(files, manifestsPath, repo, cfg, unitRepo, force)
	if err != nil {
		return err
	}

	return cleanupOrphanedUnits(processedUnits, cfg, unitRepo)
}

// getManifestsPath returns the full path to the manifests directory
func getManifestsPath(repo *git.Repository) string {
	if repo.ManifestDir != "" {
		return filepath.Join(repo.Path, repo.ManifestDir)
	}
	return repo.Path
}

// findYamlFiles returns all YAML files in the given directory and its subdirectories
func findYamlFiles(dirPath string) ([]string, error) {
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".yaml" {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", dirPath, err)
	}

	return files, nil
}

// processYamlFiles processes all YAML files and returns a map of processed units
func processYamlFiles(files []string, manifestsPath string, repo *git.Repository,
	cfg config.Config, unitRepo *db.UnitRepository, force bool) (map[string]bool, error) {
	processedUnits := make(map[string]bool)
	changedUnits := make([]QuadletUnit, 0)

	// First pass: Generate all unit files
	for _, file := range files {
		units, err := parseUnitsFromFile(file, manifestsPath, repo)
		if err != nil {
			log.Printf("error processing file %s: %v", file, err)
			continue
		}

		for _, unit := range units {
			unitKey := fmt.Sprintf("%s.%s", unit.Name, unit.Type)
			processedUnits[unitKey] = true

			content := GenerateQuadletUnit(unit, cfg.Verbose)
			unitPath := filepath.Join(cfg.QuadletDir, unitKey)

			// Check if unit has changed
			if !force {
				existingContent, err := os.ReadFile(unitPath)
				if err == nil && bytes.Equal(getContentHash(string(existingContent)), getContentHash(content)) {
					if cfg.Verbose {
						log.Printf("unit %s unchanged, skipping", unitKey)
					}
					continue
				}
			}

			// Write the unit file
			if cfg.Verbose {
				log.Printf("writing quadlet unit to: %s", unitPath)
			}
			if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
				log.Printf("error writing unit file %s: %v", unitKey, err)
				continue
			}

			changedUnits = append(changedUnits, unit)

			// Update database
			contentHash := getContentHash(content)
			if _, err := unitRepo.Create(&dbUnit.Unit{
				Name:          unit.Name,
				Type:          unit.Type,
				SHA1Hash:      contentHash,
				CleanupPolicy: "keep",
				CreatedAt:     time.Now(),
			}); err != nil {
				log.Printf("error updating database for unit %s: %v", unitKey, err)
			}
		}
	}

	// Reload systemd once for all changes
	if len(changedUnits) > 0 {
		systemd.ReloadSystemd(cfg)
		for _, unit := range changedUnits {
			systemd.RestartUnit(cfg, unit.Name, unit.Type)
		}
	}

	return processedUnits, nil
}

// parseUnitsFromFile parses all quadlet units from a single YAML file
func parseUnitsFromFile(filePath, manifestsPath string, repo *git.Repository) ([]QuadletUnit, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", filePath, err)
	}
	defer f.Close()

	var units []QuadletUnit
	decoder := yaml.NewDecoder(f)

	for {
		var unit QuadletUnit
		if err := decoder.Decode(&unit); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("parsing YAML: %w", err)
		}

		// Add documentation to the unit
		relPath, err := filepath.Rel(manifestsPath, filePath)
		if err == nil {
			if unit.Systemd.Documentation == nil {
				unit.Systemd.Documentation = make([]string, 0)
			}
			unit.Systemd.Documentation = append(unit.Systemd.Documentation, repo.URL)
			unit.Systemd.Documentation = append(unit.Systemd.Documentation, fmt.Sprintf("file://%s", relPath))
		}

		units = append(units, unit)
	}

	return units, nil
}

func cleanupOrphanedUnits(processedUnits map[string]bool, cfg config.Config, unitRepo *db.UnitRepository) error {
	dbUnits, err := unitRepo.List()
	if err != nil {
		return fmt.Errorf("error fetching units from database: %w", err)
	}

	for _, dbUnit := range dbUnits {
		unitKey := fmt.Sprintf("%s.%s", dbUnit.Name, dbUnit.Type)
		if !processedUnits[unitKey] && (dbUnit.CleanupPolicy == "delete") {
			unitPath := filepath.Join(cfg.QuadletDir, unitKey)

			if err := os.Remove(unitPath); err != nil {
				log.Printf("error removing orphaned unit %s: %v", unitPath, err)
				continue
			}

			if err := unitRepo.Delete(dbUnit.ID); err != nil {
				log.Printf("error deleting unit %s from database: %v", unitKey, err)
				continue
			}

			if cfg.Verbose {
				log.Printf("removed orphaned unit %s", unitPath)
			}
		}
	}

	return nil
}

// getContentHash generates a SHA1 hash of the given content
func getContentHash(content string) []byte {
	hash := sha1.New()
	hash.Write([]byte(content))
	return hash.Sum(nil)
}
