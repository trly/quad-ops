// Package compose provides Docker Compose file parsing and handling
package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/validate"
)

// ReadProjects reads all Docker Compose projects from a directory path.
func ReadProjects(path string) ([]*types.Project, error) {
	logger := log.NewLogger(false)
	return ReadProjectsWithLogger(path, logger)
}

// ReadProjectsWithLogger reads all Docker Compose projects from a directory path with a provided logger.
func ReadProjectsWithLogger(path string, logger log.Logger) ([]*types.Project, error) {
	var projects []*types.Project

	// Validate path before proceeding
	if path == "" {
		return nil, fmt.Errorf("empty compose directory path provided")
	}

	// Check if the directory exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("compose directory does not exist (check that the composeDir configuration points to a valid directory in the repository)")
		}
		return nil, fmt.Errorf("failed to access compose directory: %w", err)
	}

	// Ensure it's a directory
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	logger.Debug("Reading docker-compose files", "path", path)

	composeFilesFound := false

	logger.Debug("Walking directory to find compose files", "path", path)

	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// Log the error but continue walking if possible
			logger.Debug("Error accessing path", "path", filePath, "error", err)
			return nil
		}

		// Add verbose logging for all files
		logger.Debug("Examining path", "path", filePath, "isDir", info.IsDir(), "ext", filepath.Ext(filePath))

		if !info.IsDir() && (filepath.Ext(filePath) == ".yaml" || filepath.Ext(filePath) == ".yml") {
			// Check if the file name starts with docker-compose or compose
			baseName := filepath.Base(filePath)
			isComposeFile := false

			// Common Docker Compose file patterns
			if baseName == "docker-compose.yml" || baseName == "docker-compose.yaml" ||
				baseName == "compose.yml" || baseName == "compose.yaml" {
				isComposeFile = true
				logger.Debug("Found compose file", "path", filePath)
			}

			if isComposeFile {
				composeFilesFound = true
				project, err := ParseComposeFileWithLogger(filePath, logger)
				if err != nil {
					// Log parsing errors at error level so they're visible without verbose mode
					logger.Error("Error parsing compose file", "path", filePath, "error", err)
					// Continue processing other files
					return nil
				}
				projects = append(projects, project)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read docker-compose files: %w", err)
	}

	// No compose files found in the directory
	if !composeFilesFound {
		logger.Debug("No docker-compose files found", "path", path)
		// Return empty list instead of error, as this is not necessarily an error condition
	}

	return projects, nil
}

// ParseComposeFile parses a Docker Compose file at the specified path.
func ParseComposeFile(path string) (*types.Project, error) {
	logger := log.NewLogger(false)
	return ParseComposeFileWithLogger(path, logger)
}

// ParseComposeFileWithLogger parses a Docker Compose file at the specified path with a provided logger.
func ParseComposeFileWithLogger(path string, logger log.Logger) (*types.Project, error) {
	ctx := context.Background()
	// Create a normalized project name based on the directory
	dirPath := filepath.Dir(path)
	projectName := filepath.Base(dirPath)

	// In production, let's format the project name based on repo and directory
	// For tests and edge cases, use a simple default name
	if projectName == "" || projectName == "." || projectName == "/" {
		projectName = "default"
	}

	// Sanitize project name to meet Docker Compose requirements:
	// must consist only of lowercase alphanumeric characters, hyphens, and underscores
	// and start with a letter or number
	projectName = sanitizeProjectName(projectName)

	// For production, always use directory name for project naming consistency
	// Extract repository name from path (assuming repositories/<reponame>/<folder/subfolder/etc> pattern)
	// Use last component of path for folder name regardless of composeDir setting
	dirComponents := strings.Split(dirPath, string(os.PathSeparator))
	if len(dirComponents) >= 3 {
		// Look for repositories/<reponame> pattern
		for i, component := range dirComponents {
			if component == "repositories" && i+1 < len(dirComponents) {
				repoName := dirComponents[i+1]
				// Always use the actual directory containing the compose file
				folderName := filepath.Base(dirPath)
				projectName = fmt.Sprintf("%s-%s", repoName, folderName)
				projectName = sanitizeProjectName(projectName)
				break
			}
		}
	}

	// Get directory containing compose file to look for .env file there
	composeDir := filepath.Dir(path)
	// Check if .env exists in the compose directory
	envPath := filepath.Join(composeDir, ".env")

	// Create slice of options
	projectOpts := []string{path}

	// Add explicit .env file if it exists in compose directory
	if _, err := os.Stat(envPath); err == nil {
		logger.Debug("Found .env file in compose directory", "path", envPath)

		// Validate file path before reading
		absPath, err := filepath.Abs(envPath)
		if err != nil {
			logger.Warn("Failed to get absolute path for .env file", "path", envPath, "error", err)
		} else {
			// Load environment variables directly from the file
			// This file path was constructed using filepath.Join and validated, so it's safe
			// #nosec G304 -- safe because we're reading a file from a path we constructed with filepath.Join
			environmentData, err := os.ReadFile(absPath)
			if err != nil {
				logger.Warn("Failed to read .env file", "path", absPath, "error", err)
			} else {
				// Parse .env file content and set environment variables
				envContent := string(environmentData)
				for _, line := range strings.Split(envContent, "\n") {
					// Skip empty lines or comments
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}

					// Parse KEY=VALUE format
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])

						// Create security validator
						validator := validate.NewSecretValidator(logger)

						// Validate environment variable key with extended validation
						if err := validate.EnvKey(key); err != nil {
							logger.Warn("Invalid environment variable key", "key", key, "error", err)
							continue
						}

						// Validate environment variable value for size and content
						if err := validator.ValidateEnvValue(key, value); err != nil {
							logger.Warn("Invalid environment variable value", "key", key, "error", err)
							continue
						}

						// Set environment variable
						if err := os.Setenv(key, value); err != nil {
							logger.Warn("Failed to set environment variable", "key", key, "error", err)
						} else {
							// Use sanitized logging for potentially sensitive values
							safeValue := validate.SanitizeForLogging(key, value)
							logger.Debug("Set environment variable from .env file", "key", key, "value", safeValue)
						}
					}
				}
			}
		}
	}

	options, err := cli.NewProjectOptions(
		projectOpts,
		cli.WithOsEnv,
		cli.WithDotEnv, // Will now find our copied .env file in the temp directory
		cli.WithName(projectName),
	)

	if err != nil {
		return nil, err
	}

	project, err := cli.ProjectFromOptions(ctx, options)
	if err != nil {
		return nil, err
	}

	// Set the working directory to allow access to environment files
	project.WorkingDir = filepath.Dir(path)

	return project, nil
}

// validateEnvKey validates that an environment variable key is safe.
func validateEnvKey(key string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	// Environment variable names should follow standard conventions
	// Allow alphanumeric characters and underscores, but not start with digits
	for i, r := range key {
		if i == 0 && (r >= '0' && r <= '9') {
			return fmt.Errorf("environment variable key cannot start with digit")
		}
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return fmt.Errorf("environment variable key contains invalid character: %c", r)
		}
	}

	// Prevent overriding critical system environment variables
	criticalVars := []string{"PATH", "HOME", "USER", "SHELL", "PWD", "OLDPWD", "TERM"}
	for _, critical := range criticalVars {
		if strings.EqualFold(key, critical) {
			return fmt.Errorf("cannot override critical system environment variable: %s", key)
		}
	}

	return nil
}

// sanitizeProjectName ensures the project name meets Docker Compose requirements:
// must consist only of lowercase alphanumeric characters, hyphens, and underscores
// and start with a letter or number.
func sanitizeProjectName(name string) string {
	if name == "" {
		return "default"
	}

	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace invalid characters with hyphens
	validChars := regexp.MustCompile(`[^a-z0-9_-]`)
	name = validChars.ReplaceAllString(name, "-")

	// Remove leading/trailing hyphens and underscores
	name = strings.Trim(name, "-_")

	// Ensure it starts with alphanumeric
	startsWithAlphanumeric := regexp.MustCompile(`^[a-z0-9]`)
	if !startsWithAlphanumeric.MatchString(name) {
		name = "p" + name // prefix with 'p' for 'project'
	}

	// Handle empty result or very short names
	if len(name) == 0 {
		return "default"
	}
	if len(name) == 1 && (name == "-" || name == "_") {
		return "default"
	}

	return name
}
