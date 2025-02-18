package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"quad-ops/internal/config"
	"quad-ops/internal/git"
	"quad-ops/internal/quadlet"
)

var verbose *bool

func main() {
	configPath := flag.String("config", "/etc/quad-ops/config.yaml", "Path to configuration file")
	dryRun := flag.Bool("dry-run", false, "Print actions without executing them")
	userMode := flag.Bool("user-mode", false, "Run quad-ops in user mode")
	daemon := flag.Bool("daemon", false, "Run as a background daemon")
	interval := flag.Int("interval", 300, "Update check interval in seconds when running as daemon")
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	if *daemon {
		runDaemon(*configPath, *dryRun, *userMode, *interval)
		return
	}

	runCheck(*configPath, *dryRun, *userMode)
}

func runDaemon(configPath string, dryRun, userMode bool, interval int) {
	log.Printf("Starting quad-ops daemon with %v second interval", interval)
	for {
		runCheck(configPath, dryRun, userMode)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func runCheck(configPath string, dryRun, userMode bool) {
	if *verbose {
		log.Printf("Using config file: %s", configPath)
	}

	cfg, err := config.LoadConfig(configPath, userMode, *verbose)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(cfg.QuadletDir, 0755); err != nil {
		log.Fatal("Failed to create quadlet directory:", err)
	}

	for _, repoConfig := range cfg.Repositories {
		if *verbose {
			log.Printf("Processing repository: %s", repoConfig.Name)
		}

		if !dryRun {
			repo := git.NewRepository(filepath.Join(cfg.RepositoryDir, repoConfig.Name), repoConfig.URL, repoConfig.Target, *verbose)
			if err := repo.SyncRepository(); err != nil {
				log.Printf("Error syncing repository %s: %v", repoConfig.Name, err)
				continue
			}

			manifestsPath := filepath.Join(cfg.RepositoryDir, repoConfig.Name)
			if err := quadlet.ProcessManifests(manifestsPath, cfg.QuadletDir, userMode, *verbose); err != nil {
				log.Printf("Error processing manifests for %s: %v", repoConfig.Name, err)
				continue
			}
		}
	}
}
