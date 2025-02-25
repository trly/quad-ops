package quadlet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"quad-ops/internal/git"
	"quad-ops/internal/systemd"

	"gopkg.in/yaml.v3"
)

func ProcessManifests(repo *git.Repository, quadletDir string, userMode bool, verbose bool, force bool) error {
	manifestsPath := repo.Path

	if repo.ManifestDir != "" {
		manifestsPath = filepath.Join(manifestsPath, repo.ManifestDir)
	}

	if verbose {
		log.Printf("processing manifests from repository: %s at path: %s", repo.URL, manifestsPath)
		log.Printf("output directory: %s", quadletDir)
	}

	var files []string
	err := filepath.Walk(manifestsPath, func(path string, info os.FileInfo, err error) error {
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

	if verbose {
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

			content := GenerateQuadletUnit(unit, verbose)
			unitPath := filepath.Join(quadletDir, fmt.Sprintf("%s.%s", unit.Name, unit.Type))

			existingContent, err := os.ReadFile(unitPath)
			if err == nil && !force {
				if getContentHash(string(existingContent)) == getContentHash(content) {
					if verbose {
						log.Printf("unit %s.%s unchanged, skipping deployment", unit.Name, unit.Type)
					}
					continue
				}
			}

			if verbose {
				log.Printf("writing quadlet unit to: %s", unitPath)
			}

			if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
				log.Printf("error writing quadlet unit %s: %v", unitPath, err)
				continue
			}

			if err := systemd.ReloadAndRestartUnit(unit.Name, unit.Type, userMode, verbose); err != nil {
				log.Printf("error reloading unit %s-%s: %v", unit.Name, unit.Type, err)
				continue
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
