package quadlet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"quad-ops/internal/systemd"

	"gopkg.in/yaml.v3"
)

func ProcessManifests(manifestsPath string, quadletDir string, userMode bool, verbose bool, force bool) error {

	if verbose {
		log.Printf("Processing manifests from: %s", manifestsPath)
		log.Printf("Output directory: %s", quadletDir)
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
		log.Printf("Found %d YAML files in manifests directory and subdirectories", len(files))
	}

	for _, file := range files {

		f, err := os.Open(file)
		if err != nil {
			log.Printf("Error opening file %s: %v", file, err)
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
				log.Printf("Error parsing YAML from %s: %v", file, err)
				continue
			}

			content := GenerateQuadletUnit(unit, verbose)
			unitPath := filepath.Join(quadletDir, fmt.Sprintf("%s.%s", unit.Name, unit.Type))

			existingContent, err := os.ReadFile(unitPath)
			if err == nil && !force {
				if getContentHash(string(existingContent)) == getContentHash(content) {
					if verbose {
						log.Printf("Unit %s.%s unchanged, skipping deployment", unit.Name, unit.Type)
					}
					continue
				}
			}

			if verbose {
				log.Printf("Writing quadlet unit to: %s", unitPath)
			}

			if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
				log.Printf("Error writing quadlet unit %s: %v", unitPath, err)
				continue
			}

			if err := systemd.ReloadAndRestartUnit(unit.Name, unit.Type, userMode, verbose); err != nil {
				log.Printf("Error reloading unit %s-%s: %v", unit.Name, unit.Type, err)
				continue
			}

			log.Printf("Generated Quadlet %s definition for %s\n", unit.Type, unit.Name)
		}

	}
	return nil
}

func getContentHash(content string) string {
	hash := sha256.New()
	hash.Write([]byte(content))
	return hex.EncodeToString(hash.Sum(nil))
}
