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

type Processor struct {
	repo     *git.Repository
	unitRepo *Repository
	verbose  bool
}

func NewProcessor(repo *git.Repository, unitRepo *Repository) *Processor {
	return &Processor{
		repo:     repo,
		unitRepo: unitRepo,
		verbose:  config.GetConfig().Verbose,
	}
}

func (p *Processor) Process(force bool) error {
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

	return p.cleanupOrphanedUnits(processedUnits)
}

func (p *Processor) getManifestsPath() string {
	if p.repo.ManifestDir != "" {
		return filepath.Join(p.repo.Path, p.repo.ManifestDir)
	}
	return p.repo.Path
}

func (p *Processor) findYamlFiles(dirPath string) ([]string, error) {
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

func (p *Processor) processYamlFiles(files []string, manifestsPath string, force bool) (map[string]bool, error) {
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

func (p *Processor) hasUnitChanged(unitPath, content string) bool {
	existingContent, err := os.ReadFile(unitPath)
	if err == nil && bytes.Equal(getContentHash(string(existingContent)), getContentHash(content)) {
		if p.verbose {
			log.Printf("unit %s unchanged, skipping", unitPath)
		}
		return false
	}
	return true
}

func (p *Processor) writeUnitFile(unitPath, content string) error {
	if p.verbose {
		log.Printf("writing quadlet unit to: %s", unitPath)
	}
	return os.WriteFile(unitPath, []byte(content), 0644)
}

func (p *Processor) updateUnitDatabase(unit *QuadletUnit, content string) error {
	contentHash := getContentHash(content)
	_, err := p.unitRepo.Create(&Unit{
		Name:          unit.Name,
		Type:          unit.Type,
		SHA1Hash:      contentHash,
		CleanupPolicy: "keep",
		CreatedAt:     time.Now(),
	})
	return err
}

func (p *Processor) reloadUnits(changedUnits []QuadletUnit) {
	err := ReloadSystemd()
	if err != nil {
		logger.GetLogger().Error("error reloading systemd units", "err", err)
	}
	for _, unit := range changedUnits {
		err := RestartUnit(unit.Name, unit.Type)
		if err != nil {
			logger.GetLogger().Error("error restarting unit", "unit", unit.Name, "err", err)
		}
	}
}

func ProcessManifests(repo *git.Repository, force bool) error {
	dbConn, err := db.Connect()
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer dbConn.Close()

	unitRepo := NewUnitRepository(dbConn)
	processor := NewProcessor(repo, unitRepo)

	return processor.Process(force)
}

func (p *Processor) parseUnitsFromFile(filePath, manifestsPath string) ([]QuadletUnit, error) {
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

		relPath, err := filepath.Rel(manifestsPath, filePath)
		if err == nil {
			if unit.Systemd.Documentation == nil {
				unit.Systemd.Documentation = make([]string, 0)
			}
			unit.Systemd.Documentation = append(unit.Systemd.Documentation, p.repo.URL)
			unit.Systemd.Documentation = append(unit.Systemd.Documentation, fmt.Sprintf("file://%s", relPath))
		}

		units = append(units, unit)
	}

	return units, nil
}

func (p *Processor) cleanupOrphanedUnits(processedUnits map[string]bool) error {
	dbUnits, err := p.unitRepo.FindAll()
	if err != nil {
		return fmt.Errorf("error fetching units from database: %w", err)
	}

	for _, dbUnit := range dbUnits {
		unitKey := fmt.Sprintf("%s.%s", dbUnit.Name, dbUnit.Type)
		if !processedUnits[unitKey] && (dbUnit.CleanupPolicy == "delete") {
			unitPath := filepath.Join(config.GetConfig().QuadletDir, unitKey)

			if err := os.Remove(unitPath); err != nil {
				log.Printf("error removing orphaned unit %s: %v", unitPath, err)
				continue
			}

			if err := p.unitRepo.Delete(dbUnit.ID); err != nil {
				log.Printf("error deleting unit %s from database: %v", unitKey, err)
				continue
			}

			if p.verbose {
				log.Printf("removed orphaned unit %s", unitPath)
			}
		}
	}

	return nil
}

func getContentHash(content string) []byte {
	hash := sha1.New()
	hash.Write([]byte(content))
	return hash.Sum(nil)
}
