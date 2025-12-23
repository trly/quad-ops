package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/systemd"
)

type UpCmd struct {
	Project string   `arg:"" optional:"" help:"Project name to start (starts all if not specified)"`
	Service []string `short:"s" help:"Specific service(s) to start within the project"`
}

func (u *UpCmd) Run(globals *Globals) error {
	if globals.AppCfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	ctx := context.Background()

	// Collect all services and images to start
	services, images, err := u.collectServicesAndImages(ctx, globals)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		if globals.Verbose {
			fmt.Println("No services found to start")
		}
		return nil
	}

	// Pull images first to avoid timeout during service start
	if err := pullImages(images, globals.Verbose); err != nil {
		return err
	}

	// Start services using systemd D-Bus
	return startServices(ctx, services, globals.Verbose)
}

// collectServicesAndImages gathers the list of systemd service units and their images.
// Services with missing secrets are skipped with a warning.
func (u *UpCmd) collectServicesAndImages(ctx context.Context, globals *Globals) ([]string, []string, error) {
	var services []string
	imageSet := make(map[string]struct{})

	// Get available secrets once for efficiency
	availableSecrets, secretsErr := compose.GetAvailablePodmanSecrets(ctx)
	if secretsErr != nil {
		fmt.Printf("WARNING: failed to query podman secrets: %v\n", secretsErr)
	}

	for _, repo := range globals.AppCfg.Repositories {
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

			// Filter by project if specified (match compose project name)
			if u.Project != "" && projectName != u.Project {
				continue
			}

			for serviceName, svc := range lp.Project.Services {
				// Filter by service if specified
				if len(u.Service) > 0 && !contains(u.Service, serviceName) {
					continue
				}

				// Check for missing secrets
				if missingSecrets := compose.CheckServiceSecrets(svc, availableSecrets); len(missingSecrets) > 0 {
					fmt.Printf("WARNING: skipping service %s-%s: missing secrets %v\n",
						projectName, serviceName, missingSecrets)
					continue
				}

				// Unit name follows pattern: {project}-{service}.service
				unitName := fmt.Sprintf("%s-%s.service", projectName, serviceName)
				services = append(services, unitName)

				// Collect image for pre-pulling
				if svc.Image != "" {
					imageSet[svc.Image] = struct{}{}
				}
			}
		}
	}

	images := make([]string, 0, len(imageSet))
	for img := range imageSet {
		images = append(images, img)
	}

	return services, images, nil
}

// pullImages pulls the given container images using podman.
func pullImages(images []string, verbose bool) error {
	if len(images) == 0 {
		return nil
	}

	total := len(images)
	if verbose {
		fmt.Printf("Pulling %d image(s)...\n", total)
	}

	for i, image := range images {
		if verbose {
			fmt.Printf("  [%d/%d] Pulling %s\n", i+1, total, image)
		}

		cmd := exec.Command("podman", "pull", image) //nolint:gosec // image names from validated compose files
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to pull image %s: %w", image, err)
			}
		} else {
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to pull image %s: %w\n%s", image, err, string(output))
			}
		}
	}

	return nil
}

// startServices starts the given systemd service units.
func startServices(ctx context.Context, services []string, verbose bool) error {
	if len(services) == 0 {
		return nil
	}

	client, err := systemd.New(ctx, systemd.ScopeAuto)
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer func() { _ = client.Close() }()

	if verbose {
		fmt.Printf("Starting %d service(s)...\n", len(services))
	}

	err = client.Start(ctx, services...)
	if verbose {
		fmt.Printf("Started %d service(s)\n", len(services))
	}
	if err != nil {
		return fmt.Errorf("some services failed to start: %w", err)
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
