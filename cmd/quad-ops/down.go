package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/systemd"
)

type DownCmd struct {
	Project string   `arg:"" optional:"" help:"Project name to stop (stops all if not specified)"`
	Service []string `short:"s" help:"Specific service(s) to stop within the project"`
}

func (d *DownCmd) Run(globals *Globals) error {
	if globals.AppCfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	ctx := context.Background()

	// Collect all services to stop
	services, err := d.collectServices(ctx, globals)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		if globals.Verbose {
			fmt.Println("No services found to stop")
		}
		return nil
	}

	// Stop services using systemd D-Bus
	return stopServices(ctx, services, globals.Verbose)
}

// collectServices gathers the list of systemd service units to stop.
func (d *DownCmd) collectServices(ctx context.Context, globals *Globals) ([]string, error) {
	var services []string

	for _, repo := range globals.AppCfg.Repositories {
		// Filter by project if specified
		if d.Project != "" && repo.Name != d.Project {
			continue
		}

		repoPath := filepath.Join(globals.AppCfg.GetRepositoryDir(), repo.Name)
		composeSourceDir := repoPath
		if repo.ComposeDir != "" {
			composeSourceDir = filepath.Join(repoPath, repo.ComposeDir)
		}

		projects, err := compose.LoadAll(ctx, composeSourceDir, nil)
		if err != nil {
			if globals.Verbose {
				fmt.Printf("WARNING: failed to load projects from %s: %v\n", repo.Name, err)
			}
			continue
		}

		for _, lp := range projects {
			if lp.Error != nil {
				if globals.Verbose {
					fmt.Printf("WARNING: %s: %v\n", lp.FilePath, lp.Error)
				}
				continue
			}

			if lp.Project == nil {
				continue
			}

			projectName := lp.Project.Name

			for serviceName := range lp.Project.Services {
				// Filter by service if specified
				if len(d.Service) > 0 && !contains(d.Service, serviceName) {
					continue
				}

				// Unit name follows pattern: {project}-{service}.service
				unitName := fmt.Sprintf("%s-%s.service", projectName, serviceName)
				services = append(services, unitName)
			}
		}
	}

	return services, nil
}

// stopServices stops the given systemd service units.
func stopServices(ctx context.Context, services []string, verbose bool) error {
	if len(services) == 0 {
		return nil
	}

	client, err := systemd.New(ctx, systemd.ScopeAuto)
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer func() { _ = client.Close() }()

	if verbose {
		fmt.Printf("Stopping %d service(s)...\n", len(services))
	}

	err = client.Stop(ctx, services...)
	if verbose {
		fmt.Printf("Stopped %d service(s)\n", len(services))
	}
	if err != nil {
		return fmt.Errorf("some services failed to stop: %w", err)
	}
	return nil
}
