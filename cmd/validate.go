// Package cmd provides the command line interface for quad-ops
/*
Copyright © 2025 Travis Lyons travis.lyons@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/spf13/cobra"
	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/validate"
)

// ValidateCommand represents the validate command for quad-ops CLI.
type ValidateCommand struct{}

var (
	repoURL              string
	repoRef              string
	composeDir           string
	validatePath         string
	skipClone            bool
	tempDir              string
	checkSysRequirements bool
)

// GetCobraCommand returns the cobra command for validate operations.
func (c *ValidateCommand) GetCobraCommand() *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate [path]",
		Short: "Validates Docker Compose files and quad-ops extensions in a repository, directory, or single file",
		Long: `Validates Docker Compose files and quad-ops extensions in a repository, directory, or single file.

Can clone a git repository and validate all Docker Compose files within it, validate all 
compose files in a local directory, or validate a single compose file. Perfect for CI/CD 
pipelines and development workflows. The validation checks for:

- Valid Docker Compose file syntax
- Quad-ops extension compatibility 
- Security requirements for secrets and environment variables
- Service dependency graph integrity
- Build configuration validity

Examples:
  # Validate files in current directory
  quad-ops validate

  # Validate files in specific directory  
  quad-ops validate /path/to/compose/files

  # Validate a single compose file (great for CI)
  quad-ops validate docker-compose.yml
  quad-ops validate /path/to/my-service.compose.yml

  # Clone and validate a git repository
  quad-ops validate --repo https://github.com/user/repo.git

  # Clone specific branch/tag and validate
  quad-ops validate --repo https://github.com/user/repo.git --ref main

  # Validate specific compose directory in repository
  quad-ops validate --repo https://github.com/user/repo.git --compose-dir services`,

		Args: cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			// Determine path to validate
			if len(args) > 0 {
				validatePath = args[0]
			} else if repoURL == "" {
				validatePath = "."
			}

			// Validate arguments
			if repoURL != "" && validatePath != "" && validatePath != "." {
				return fmt.Errorf("cannot specify both repository URL and local path")
			}

			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			// Check system requirements if requested
			if checkSysRequirements {
				if err := validate.SystemRequirements(); err != nil {
					log.GetLogger().Error("System requirements not met", "error", err)
					return fmt.Errorf("system requirements not met: %w", err)
				}
			}

			var targetPath string
			var cleanup func() error

			// Handle repository cloning
			if repoURL != "" {
				path, cleanupFn, err := cloneRepository()
				if err != nil {
					return err
				}
				targetPath = path
				cleanup = cleanupFn
				defer func() {
					if cleanup != nil {
						if err := cleanup(); err != nil {
							log.GetLogger().Warn("Failed to cleanup temporary directory", "error", err)
						}
					}
				}()
			} else {
				targetPath = validatePath
			}

			// Handle compose directory subdirectory
			if composeDir != "" {
				targetPath = filepath.Join(targetPath, composeDir)
			}

			// Validate the path
			return validateCompose(targetPath)
		},
	}

	validateCmd.Flags().StringVar(&repoURL, "repo", "", "Git repository URL to clone and validate")
	validateCmd.Flags().StringVar(&repoRef, "ref", "main", "Git reference (branch/tag/commit) to checkout")
	validateCmd.Flags().StringVar(&composeDir, "compose-dir", "", "Subdirectory within repository containing compose files")
	validateCmd.Flags().BoolVar(&skipClone, "skip-clone", false, "Skip cloning if repository already exists locally")
	validateCmd.Flags().StringVar(&tempDir, "temp-dir", "", "Custom temporary directory for cloning (default: system temp)")
	validateCmd.Flags().BoolVar(&checkSysRequirements, "check-system", false, "Check system requirements (systemd, podman) before validation")

	return validateCmd
}

// cloneRepository handles git repository cloning and returns the path and cleanup function.
func cloneRepository() (string, func() error, error) {
	log.GetLogger().Info("Cloning repository for validation", "url", repoURL, "ref", repoRef)

	// Create temporary repository config for cloning
	repoConfig := config.Repository{
		Name:      "validate-temp",
		URL:       repoURL,
		Reference: repoRef,
	}

	// Create git repository instance
	gitRepo := git.NewGitRepository(repoConfig)

	// Override the default path to use a temporary directory with safe naming
	var tempPath string
	if tempDir != "" {
		// Ensure we create a subdirectory to prevent accidental deletion of user directory
		tempPath = filepath.Join(tempDir, "quad-ops-validate")
	} else {
		tempPath = filepath.Join(os.TempDir(), "quad-ops-validate")
	}
	gitRepo.Path = tempPath

	// Validation: ensure the path has our expected suffix for safety
	if !strings.HasSuffix(gitRepo.Path, "quad-ops-validate") {
		return "", nil, fmt.Errorf("invalid temporary path for security reasons: %s", gitRepo.Path)
	}

	// Check if we should skip clone
	if skipClone && isValidGitRepo(gitRepo.Path) {
		log.GetLogger().Info("Skipping clone, using existing repository", "path", gitRepo.Path)
		return gitRepo.Path, func() error { return nil }, nil
	}

	// Remove existing directory if it exists (only if it ends with our suffix)
	if _, err := os.Stat(gitRepo.Path); err == nil {
		if err := os.RemoveAll(gitRepo.Path); err != nil {
			return "", nil, fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	// Perform the clone
	if err := gitRepo.SyncRepository(); err != nil {
		return "", nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Return path and cleanup function
	cleanup := func() error {
		if !skipClone && strings.HasSuffix(gitRepo.Path, "quad-ops-validate") {
			return os.RemoveAll(gitRepo.Path)
		}
		return nil
	}

	return gitRepo.Path, cleanup, nil
}

// isValidGitRepo checks if the given path contains a valid git repository.
func isValidGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if stat, err := os.Stat(gitDir); err != nil || !stat.IsDir() {
		return false
	}
	return true
}

// isComposeFile checks if the given path appears to be a YAML file that could be a Docker Compose file.
func isComposeFile(path string) bool {
	ext := filepath.Ext(strings.ToLower(path))
	if ext != ".yml" && ext != ".yaml" {
		return false
	}

	// Quick check: try to read first few lines to see if it looks like a compose file
	file, err := os.Open(filepath.Clean(path)) // #nosec G304 - path is validated upstream
	if err != nil {
		return false
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log error or handle as appropriate - for validation context, we can ignore
			_ = closeErr
		}
	}()

	// Read first 1KB to check for compose-like content
	buffer := make([]byte, 1024)
	n, _ := file.Read(buffer)
	content := string(buffer[:n])

	// Look for common compose file indicators
	return strings.Contains(content, "services:") ||
		strings.Contains(content, "version:") ||
		strings.Contains(content, "networks:") ||
		strings.Contains(content, "volumes:")
}

// validateCompose validates Docker Compose files in the given path (file or directory).
func validateCompose(path string) error {
	log.GetLogger().Info("Validating Docker Compose files", "path", path)

	// Check if path exists
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	}
	if err != nil {
		return fmt.Errorf("failed to access path: %w", err)
	}

	var projects []*types.Project

	if stat.IsDir() {
		// Handle directory - read all compose projects
		projects, err = compose.ReadProjects(path)
		if err != nil {
			return fmt.Errorf("failed to read compose projects: %w", err)
		}
	} else {
		// Handle single file
		if !isComposeFile(path) {
			return fmt.Errorf("file does not appear to be a Docker Compose file: %s", path)
		}

		project, err := compose.ParseComposeFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse compose file: %w", err)
		}

		projects = []*types.Project{project}
	}

	if len(projects) == 0 {
		log.GetLogger().Warn("No Docker Compose files found in path", "path", path)
		return nil
	}

	log.GetLogger().Info("Found compose projects for validation", "count", len(projects))

	// Validate each project
	var validationErrors []string
	validProjectCount := 0

	for _, project := range projects {
		log.GetLogger().Info("Validating project", "name", project.Name, "services", len(project.Services), "networks", len(project.Networks), "volumes", len(project.Volumes))

		if err := validateProject(project); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Project %s: %v", project.Name, err))
			log.GetLogger().Error("Project validation failed", "project", project.Name, "error", err)
		} else {
			validProjectCount++
			log.GetLogger().Info("Project validation passed", "project", project.Name)
		}
	}

	// Print summary
	log.GetLogger().Info("Validation completed",
		"totalProjects", len(projects),
		"validProjects", validProjectCount,
		"errors", len(validationErrors))

	if len(validationErrors) > 0 {
		fmt.Printf("\nValidation Summary:\n")
		fmt.Printf("✓ Valid projects: %d\n", validProjectCount)
		fmt.Printf("✗ Invalid projects: %d\n", len(validationErrors))
		fmt.Printf("\nValidation Errors:\n")
		for _, err := range validationErrors {
			fmt.Printf("  • %s\n", err)
		}
		return fmt.Errorf("validation failed with %d errors", len(validationErrors))
	}

	fmt.Printf("\n✓ All %d projects validated successfully\n", validProjectCount)
	return nil
}

// validateProject validates a single Docker Compose project for quad-ops compatibility.
func validateProject(project *types.Project) error {
	validator := validate.NewSecretValidator()

	// Validate services
	for serviceName, service := range project.Services {
		if err := validateService(serviceName, service, validator); err != nil {
			return fmt.Errorf("service %s: %w", serviceName, err)
		}
	}

	// Validate networks
	for networkName, network := range project.Networks {
		if err := validateNetwork(networkName, network); err != nil {
			return fmt.Errorf("network %s: %w", networkName, err)
		}
	}

	// Validate volumes
	for volumeName, volume := range project.Volumes {
		if err := validateVolume(volumeName, volume); err != nil {
			return fmt.Errorf("volume %s: %w", volumeName, err)
		}
	}

	// Validate secrets
	for secretName, secret := range project.Secrets {
		if err := validateSecret(secretName, secret, validator); err != nil {
			return fmt.Errorf("secret %s: %w", secretName, err)
		}
	}

	return nil
}

// validateService validates a Docker Compose service configuration.
func validateService(_ string, service types.ServiceConfig, validator *validate.SecretValidator) error {
	// Validate environment variables
	for key, value := range service.Environment {
		if err := validate.EnvKey(key); err != nil {
			return fmt.Errorf("invalid environment key %s: %w", key, err)
		}

		if value != nil {
			if err := validator.ValidateEnvValue(key, *value); err != nil {
				return fmt.Errorf("invalid environment value for %s: %w", key, err)
			}
		}
	}

	// Validate secrets
	for _, secretRef := range service.Secrets {
		if err := validator.ValidateSecretName(secretRef.Source); err != nil {
			return fmt.Errorf("invalid secret reference %s: %w", secretRef.Source, err)
		}

		if secretRef.Target != "" {
			if err := validator.ValidateSecretTarget(secretRef.Target); err != nil {
				return fmt.Errorf("invalid secret target %s: %w", secretRef.Target, err)
			}
		}
	}

	// Validate build configuration if present
	if service.Build != nil {
		if err := validateBuild(service.Build); err != nil {
			return fmt.Errorf("build configuration: %w", err)
		}
	}

	// Validate init containers (quad-ops extension)
	if err := validateInitContainers(service); err != nil {
		return fmt.Errorf("init containers: %w", err)
	}

	return nil
}

// validateBuild validates Docker Compose build configuration.
func validateBuild(build *types.BuildConfig) error {
	if build.Context == "" {
		return fmt.Errorf("build context cannot be empty")
	}

	// Validate build args
	for key, value := range build.Args {
		if err := validate.EnvKey(key); err != nil {
			return fmt.Errorf("invalid build arg key %s: %w", key, err)
		}

		if value != nil && len(*value) > validate.MaxEnvValueSize {
			return fmt.Errorf("build arg value for %s exceeds maximum size (%d bytes, max: %d)",
				key, len(*value), validate.MaxEnvValueSize)
		}
	}

	return nil
}

// validateInitContainers validates init containers (quad-ops extension).
func validateInitContainers(service types.ServiceConfig) error {
	// Check for init container labels (quad-ops extension)
	initContainerLabels := []string{
		"quad-ops.init-containers",
		"quad-ops.init",
	}

	for _, label := range initContainerLabels {
		if value, exists := service.Labels[label]; exists {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("init container label %s cannot be empty", label)
			}
			// Additional validation could be added here for init container format
		}
	}

	return nil
}

// validateNetwork validates Docker Compose network configuration.
func validateNetwork(_ string, network types.NetworkConfig) error {
	// Basic network validation
	if bool(network.External) && network.Driver != "" {
		return fmt.Errorf("external networks cannot specify driver")
	}

	return nil
}

// validateVolume validates Docker Compose volume configuration.
func validateVolume(_ string, volume types.VolumeConfig) error {
	// Basic volume validation
	if bool(volume.External) && volume.Driver != "" {
		return fmt.Errorf("external volumes cannot specify driver")
	}

	return nil
}

// validateSecret validates Docker Compose secret configuration.
func validateSecret(secretName string, secret types.SecretConfig, validator *validate.SecretValidator) error {
	if err := validator.ValidateSecretName(secretName); err != nil {
		return fmt.Errorf("invalid secret name: %w", err)
	}

	// Validate secret source
	if secret.File != "" {
		// Allow relative paths but warn about potential security issues
		if !filepath.IsAbs(secret.File) {
			log.GetLogger().Debug("Secret uses relative file path", "secret", secretName, "path", secret.File)
			// Check if path tries to escape current directory
			if strings.Contains(secret.File, "..") {
				return fmt.Errorf("secret file path contains directory traversal: %s", secret.File)
			}
		}
	}

	return nil
}
