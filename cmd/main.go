package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"quad-ops/internal/config"
	"quad-ops/internal/db"
	"quad-ops/internal/git"
	"quad-ops/internal/quadlet"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

var verbose *bool

func main() {
	configPath := flag.String("config", "/etc/quad-ops/config.yaml", "Path to configuration file")
	dryRun := flag.Bool("dry-run", false, "Print actions without executing them")
	userMode := flag.Bool("user-mode", false, "Run quad-ops in user mode")
	daemon := flag.Bool("daemon", false, "Run as a background daemon")
	interval := flag.Int("interval", 300, "Update check interval in seconds when running as daemon")
	force := flag.Bool("force", false, "Force deployment and restart of all units")
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	if *verbose {
		log.Printf("Using config file: %s", *configPath)
	}

	cfg, err := config.LoadConfig(*configPath, *userMode, *verbose)
	if err != nil {
		log.Fatal(err)
	}

	m, err := db.GetMigrationInstance("sqlite3://"+cfg.DBPath, *verbose)
	if err != nil {
		log.Fatalf("Could not initialize migrations: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Error running migrations: %v", err)
	} else if *verbose {
		if err == migrate.ErrNoChange {
			log.Println("No new migrations to apply")
		} else {
			log.Println("Successfully applied migrations")
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Error running migrations: %v", err)
	}

	if *daemon {
		runDaemon(*cfg, *dryRun, *userMode, *interval)
		return
	}

	runCheck(*cfg, *dryRun, *userMode, *force)
}

func runDaemon(cfg config.Config, dryRun, userMode bool, interval int) {
	log.Printf("Starting quad-ops daemon with %v second interval", interval)
	for {
		runCheck(cfg, dryRun, userMode, false)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func runCheck(cfg config.Config, dryRun, userMode bool, force bool) {

	if err := os.MkdirAll(cfg.QuadletDir, 0755); err != nil {
		log.Fatal("Failed to create quadlet directory:", err)
	}

	for _, repoConfig := range cfg.Repositories {
		if !dryRun {
			if *verbose {
				log.Printf("Processing repository: %s", repoConfig.Name)
			}

			repo := git.NewRepository(filepath.Join(cfg.RepositoryDir, repoConfig.Name), repoConfig.URL, repoConfig.Target, *verbose)
			if err := repo.SyncRepository(); err != nil {
				log.Printf("Error syncing repository %s: %v", repoConfig.Name, err)
				continue
			}

			if err := quadlet.ProcessManifests(repo, cfg.QuadletDir, userMode, *verbose, force); err != nil {
				log.Printf("Error processing manifests for %s: %v", repoConfig.Name, err)
				continue
			}
		}
	}
}
