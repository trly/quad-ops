package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"quad-ops/internal/config"
	"quad-ops/internal/git"
	"quad-ops/internal/quadlet"

	"gopkg.in/yaml.v3"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	repo := git.NewRepository(cfg.Paths.ManifestsDir, cfg.Git.RepoURL, cfg.Git.Target)
	if err := repo.SyncRepository(); err != nil {
		log.Fatal(err)
	}

	files, err := os.ReadDir(cfg.Paths.ManifestsDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".yaml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(cfg.Paths.ManifestsDir, file.Name()))
		if err != nil {
			log.Printf("Error reading file %s: %v", file.Name(), err)
			continue
		}

		var unit quadlet.QuadletUnit
		if err := yaml.Unmarshal(data, &unit); err != nil {
			log.Printf("Error parsing YAML from %s: %v", file.Name(), err)
			continue
		}

		content := quadlet.GenerateQuadletUnit(unit)
		unitPath := filepath.Join(cfg.Paths.QuadletDir, fmt.Sprintf("%s.%s", unit.Name, unit.Type))

		if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
			log.Printf("Error writing quadlet unit %s: %v", unitPath, err)
			continue
		}

		fmt.Printf("Generated Quadlet %s definition for %s\n", unit.Type, unit.Name)
	}
}
