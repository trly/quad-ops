package unit

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/db"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/logger"
	"gopkg.in/yaml.v3"
)

type UnitProcessor struct {
	repo     *git.GitRepository
	unitRepo UnitRepository
	verbose  bool
}

func NewUnitProcessor(repo *git.GitRepository, unitRepo UnitRepository) *UnitProcessor {
	return &UnitProcessor{
		repo:     repo,
		unitRepo: unitRepo,
		verbose:  config.GetConfig().Verbose,
	}
}

func (p *UnitProcessor) Process(force bool) error {
	manifestsPath := p.getManifestsPath()

	if p.verbose {
		log.Printf("processing manifests from repository: %s at path: %s", p.repo.URL, manifestsPath)
		log.Printf("output directory: %s", config.GetConfig().QuadletDir)
	}

	files, err := p.findYamlFiles(manifestsPath)
	if err != nil {
		return err
	}

	processedUnits, err := p.processYamlFiles(files, manifestsPath, force)
	if err != nil {
		return err
	}

	p.cleanupOrphanedUnits(processedUnits)
	p.startAllManagedContainers()

	return nil
}

func (p *UnitProcessor) getManifestsPath() string {
	if p.repo.ManifestDir != "" {
		return filepath.Join(p.repo.Path, p.repo.ManifestDir)
	}
	return p.repo.Path
}

func (p *UnitProcessor) findYamlFiles(dirPath string) ([]string, error) {
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

func (p *UnitProcessor) processYamlFiles(files []string, manifestsPath string, force bool) (map[string]bool, error) {
	processedUnits := make(map[string]bool)
	changedUnits := make([]QuadletUnit, 0)

	for _, file := range files {
		units, err := p.parseUnitsFromFile(file, manifestsPath)
		if err != nil {
			log.Printf("error processing file %s: %v", file, err)
			continue
		}

		for _, unit := range units {
			if err := p.processUnit(&unit, force, processedUnits, &changedUnits); err != nil {
				log.Printf("error processing unit %s: %v", unit.Name, err)
			}
		}
	}

	if len(changedUnits) > 0 {
		p.reloadUnits(changedUnits)
	}

	return processedUnits, nil
}

func (p *UnitProcessor) hasUnitChanged(unitPath, content string) bool {
	existingContent, err := os.ReadFile(unitPath)
	if err == nil && bytes.Equal(getContentHash(string(existingContent)), getContentHash(content)) {
		if p.verbose {
			log.Printf("unit %s unchanged, skipping", unitPath)
		}
		return false
	}
	return true
}

func (p *UnitProcessor) writeUnitFile(unitPath, content string) error {
	if p.verbose {
		log.Printf("writing quadlet unit to: %s", unitPath)
	}
	return os.WriteFile(unitPath, []byte(content), 0644)
}

func (p *UnitProcessor) updateUnitDatabase(unit *QuadletUnit, content string) error {
	contentHash := getContentHash(content)
	
	// Use the repository's cleanup policy instead of hardcoding to "keep"
	cleanupPolicy := "keep"
	if p.repo.Cleanup == "delete" {
		cleanupPolicy = "delete"
	}
	
	_, err := p.unitRepo.Create(&Unit{
		Name:          unit.Name,
		Type:          unit.Type,
		SHA1Hash:      contentHash,
		CleanupPolicy: cleanupPolicy,
		CreatedAt:     time.Now(),
	})
	return err
}

func (p *UnitProcessor) reloadUnits(changedUnits []QuadletUnit) {
	err := ReloadSystemd()
	if err != nil {
		logger.GetLogger().Error("error reloading systemd units", "err", err)
	}

	time.Sleep(2 * time.Second)

	for _, unit := range changedUnits {
		systemdUnit := unit.GetSystemdUnit()
		err := systemdUnit.Restart()
		if err != nil {
			logger.GetLogger().Error("error restarting unit", "unit", unit.Name, "err", err)
		}
	}
}

func ProcessManifests(repo *git.GitRepository, force bool) error {
	dbConn, err := db.Connect()
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer dbConn.Close()

	unitRepo := NewUnitRepository(dbConn)
	processor := NewUnitProcessor(repo, unitRepo)

	return processor.Process(force)
}

func (p *UnitProcessor) parseUnitsFromFile(filePath, manifestsPath string) ([]QuadletUnit, error) {
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
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parsing YAML: %w", err)
		}

		units = append(units, unit)
	}

	return units, nil
}

func (p *UnitProcessor) cleanupOrphanedUnits(processedUnits map[string]bool) error {
	dbUnits, err := p.unitRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching units from database: %w", err)
	}

	for _, dbUnit := range dbUnits {
		unitKey := fmt.Sprintf("%s.%s", dbUnit.Name, dbUnit.Type)
		if !processedUnits[unitKey] && (dbUnit.CleanupPolicy == "delete") {
			if p.verbose {
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
			} else if p.verbose {
				log.Printf("successfully stopped orphaned unit %s", unitKey)
			}
			
			// Then remove the unit file
			unitPath := filepath.Join(config.GetConfig().QuadletDir, unitKey)
			if err := os.Remove(unitPath); err != nil {
				if !os.IsNotExist(err) {
					log.Printf("error removing orphaned unit file %s: %v", unitPath, err)
				}
			} else if p.verbose {
				log.Printf("removed orphaned unit file %s", unitPath)
			}
			
			// Finally, remove from database
			if err := p.unitRepo.Delete(dbUnit.ID); err != nil {
				log.Printf("error deleting unit %s from database: %v", unitKey, err)
				continue
			}
			
			if p.verbose {
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

func (p *UnitProcessor) startAllManagedContainers() error {
	units, err := p.unitRepo.FindByUnitType("container")
	if err != nil {
		return fmt.Errorf("error fetching units from database: %w", err)
	}

	for _, unit := range units {
		systemdUnit := &BaseSystemdUnit{
			Name: unit.Name,
			Type: unit.Type,
		}
		
		if err := systemdUnit.Start(); err != nil {
			log.Printf("error starting unit %s: %v", unit.Name, err)
		}
	}

	return nil
}

func getContentHash(content string) []byte {
	hash := sha1.New()
	hash.Write([]byte(content))
	return hash.Sum(nil)
}
