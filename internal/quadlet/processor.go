package quadlet

import (
	"crypto/sha256"
	"encoding/hex"
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

func ProcessManifests(repo *git.Repository, cfg config.Config, force bool) error {
	manifestsPath := repo.Path

	dbConn, err := db.Connect(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()
	unitRepo := db.NewUnitRepository(dbConn)

	if repo.ManifestDir != "" {
		manifestsPath = filepath.Join(manifestsPath, repo.ManifestDir)
	}

	if cfg.Verbose {
		log.Printf("processing manifests from repository: %s at path: %s", repo.URL, manifestsPath)
		log.Printf("output directory: %s", cfg.QuadletDir)
	}

	var files []string
	err = filepath.Walk(manifestsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".yaml" {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("walking directory %s: %w", manifestsPath, err)
	}

	if cfg.Verbose {
		log.Printf("found %d YAML files in manifests directory and subdirectories", len(files))
	}

	for _, file := range files {

		f, err := os.Open(file)
		if err != nil {
			log.Printf("error opening file %s: %v", file, err)
			continue
		}
		defer f.Close()

		decoder := yaml.NewDecoder(f)
		for {
			var unit QuadletUnit
			if err := decoder.Decode(&unit); err != nil {
				if err.Error() == "EOF" {
					break
				}
				log.Printf("error parsing YAML from %s: %v", file, err)
				continue
			}

			relPath, err := filepath.Rel(manifestsPath, file)
			if err == nil {
				if unit.Systemd.Documentation == nil {
					unit.Systemd.Documentation = make([]string, 0)
				}
				unit.Systemd.Documentation = append(unit.Systemd.Documentation, repo.URL)
				unit.Systemd.Documentation = append(unit.Systemd.Documentation, fmt.Sprintf("file://%s", relPath))
			}

			content := GenerateQuadletUnit(unit, cfg.Verbose)
			unitPath := filepath.Join(cfg.QuadletDir, fmt.Sprintf("%s.%s", unit.Name, unit.Type))

			existingContent, err := os.ReadFile(unitPath)
			if err == nil && !force {
				if getContentHash(string(existingContent)) == getContentHash(content) {
					if cfg.Verbose {
						log.Printf("unit %s.%s unchanged, skipping deployment", unit.Name, unit.Type)
					}
					continue
				}
			}

			if cfg.Verbose {
				log.Printf("writing quadlet unit to: %s", unitPath)
			}

			if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
				log.Printf("error writing quadlet unit %s: %v", unitPath, err)
				continue
			}

			if err := systemd.ReloadAndRestartUnit(cfg, unit.Name, unit.Type); err != nil {
				log.Printf("error reloading unit %s-%s: %v", unit.Name, unit.Type, err)
				continue
			}

			unitRepo.Create(&dbUnit.Unit{
				Name:          unit.Name,
				Type:          unit.Type,
				CleanupPolicy: "keep",
				CreatedAt:     time.Now(),
			})

			if cfg.Verbose {
				log.Printf("unit %s.%s deployed successfully", unit.Name, unit.Type)
			}

			log.Printf("generated Quadlet %s definition for %s\n", unit.Type, unit.Name)
		}

	}
	return nil
}

func getContentHash(content string) string {
	hash := sha256.New()
	hash.Write([]byte(content))
	return hex.EncodeToString(hash.Sum(nil))
}
