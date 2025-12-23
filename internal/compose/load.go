package compose

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

// LoadOptions contains optional configuration for Load.
type LoadOptions struct {
	// Workdir sets the base directory for resolving relative paths.
	// If not specified, the directory containing the compose file is used.
	Workdir string

	// Environment sets environment variables that will be used for
	// variable interpolation in the compose file.
	Environment map[string]string

	// EnvFiles specifies .env files to load before parsing the compose file.
	// Variables from these files will be available for interpolation.
	EnvFiles []string
}

// Load loads a single compose project from the filesystem and returns a validated Project.
//
// The path argument can be:
//   - A file path: loads that specific compose file
//   - A directory: looks for compose.yaml, compose.yml, docker-compose.yaml, or docker-compose.yml
//     in the root directory only (not recursive)
//
// For recursive directory scanning, use LoadAll instead.
//
// opts can be nil for default behavior.
//
// Load returns an error if the file cannot be found, contains invalid YAML,
// or fails validation against the compose specification.
func Load(ctx context.Context, path string, opts *LoadOptions) (*types.Project, error) {
	// Check context before doing any work
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use default options if none provided
	if opts == nil {
		opts = &LoadOptions{}
	}

	// Determine if path is a file or directory
	pathInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &fileNotFoundError{path: path, cause: err}
		}
		return nil, &pathError{path: path, cause: err}
	}

	var filePath string
	var workdir string

	if pathInfo.IsDir() {
		// Look for compose files in the directory
		found, dir := findComposeFile(path)
		if found == "" {
			return nil, &fileNotFoundError{path: path, cause: errors.New("no compose file found")}
		}
		filePath = found
		workdir = dir
	} else {
		// Use the provided file
		filePath = path
		workdir = filepath.Dir(path)
	}

	// Override workdir if specified
	if opts.Workdir != "" {
		workdir = opts.Workdir
	}

	// Load environment from env files and options
	envMap := make(map[string]string)

	// Load from specified env files
	for _, envFile := range opts.EnvFiles {
		if err := loadEnvFile(envFile, envMap); err != nil {
			return nil, &pathError{path: envFile, cause: err}
		}
	}

	// Load from default .env file in workdir if it exists
	defaultEnvFile := filepath.Join(workdir, ".env")
	if _, err := os.Stat(defaultEnvFile); err == nil {
		_ = loadEnvFile(defaultEnvFile, envMap)
	}

	// Merge provided environment variables (they take precedence)
	maps.Copy(envMap, opts.Environment)

	// Load config files using compose-go's loader
	configDetails, err := loader.LoadConfigFiles(ctx, []string{filePath}, workdir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &fileNotFoundError{path: filePath, cause: err}
		}
		if isYAMLError(err) {
			return nil, &invalidYAMLError{cause: err}
		}
		return nil, &pathError{path: filePath, cause: err}
	}

	// Merge environment into config details
	if configDetails.Environment == nil {
		configDetails.Environment = make(types.Mapping)
	}
	for key, val := range envMap {
		if _, exists := configDetails.Environment[key]; !exists {
			configDetails.Environment[key] = val
		}
	}

	// Derive project name from directory before loading so compose-go resolves
	// network/volume names correctly (e.g. "infrastructure_proxy" not "_proxy").
	projectName := filepath.Base(workdir)

	// Build loader options - always skip validation initially so we can set project name first
	loaderOpts := []func(*loader.Options){
		func(o *loader.Options) {
			o.SkipValidation = true
			o.SetProjectName(projectName, false)
		},
	}

	// Load and parse the compose file
	project, err := loader.LoadWithContext(ctx, *configDetails, loaderOpts...)
	if err != nil {
		if isYAMLError(err) {
			return nil, &invalidYAMLError{cause: err}
		}
		return nil, &loaderError{cause: err}
	}

	// Validate the project against compose specification
	if err := validateProject(ctx, project); err != nil {
		return nil, err
	}

	// Validate quadlet compatibility - compose files must always be quadlet compatible
	if err := validateQuadletCompatibility(ctx, project); err != nil {
		return nil, err
	}

	// Parse service dependencies and env secrets
	if err := parseServiceDependencies(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// LoadAll recursively discovers and loads all compose projects in a directory.
//
// The path argument should be a directory. LoadAll will recursively find all
// compose.yaml, compose.yml, docker-compose.yaml, and docker-compose.yml files
// and load each as a separate project.
//
// Returns a slice of loaded projects and their file paths, or an error if the
// path is invalid. Individual project load errors are collected and returned
// along with successfully loaded projects.
func LoadAll(ctx context.Context, path string, opts *LoadOptions) ([]LoadedProject, error) {
	// Check context before doing any work
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Verify path exists and is a directory
	pathInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &fileNotFoundError{path: path, cause: err}
		}
		return nil, &pathError{path: path, cause: err}
	}

	if !pathInfo.IsDir() {
		return nil, &pathError{path: path, cause: errors.New("path must be a directory")}
	}

	// Find all compose files
	composeFiles, err := findAllComposeFiles(path)
	if err != nil {
		return nil, err
	}

	if len(composeFiles) == 0 {
		return []LoadedProject{}, nil
	}

	// Load each compose file
	projects := make([]LoadedProject, 0, len(composeFiles))
	for _, filePath := range composeFiles {
		project, err := Load(ctx, filePath, opts)
		if err != nil {
			// Continue loading other projects even if one fails
			projects = append(projects, LoadedProject{
				FilePath: filePath,
				Project:  nil,
				Error:    err,
			})
			continue
		}

		projects = append(projects, LoadedProject{
			FilePath: filePath,
			Project:  project,
			Error:    nil,
		})
	}

	return projects, nil
}

// LoadedProject represents a loaded compose project with its file path and any error.
type LoadedProject struct {
	FilePath string
	Project  *types.Project
	Error    error
}

// findAllComposeFiles recursively finds all compose files in a directory.
func findAllComposeFiles(path string) ([]string, error) {
	var composeFiles []string

	err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		name := d.Name()
		if name == "compose.yaml" || name == "compose.yml" || name == "docker-compose.yaml" || name == "docker-compose.yml" {
			composeFiles = append(composeFiles, filePath)
		}

		return nil
	})

	return composeFiles, err
}

// findComposeFile searches for a compose file in the given directory.
// It returns the full path to the first found compose file and the directory path.
// Returns empty string if no compose file is found.
func findComposeFile(dir string) (string, string) {
	candidates := []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
	}

	for _, name := range candidates {
		fullPath := filepath.Join(dir, name)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, dir
		}
	}

	return "", ""
}

// loadEnvFile loads key=value pairs from a .env file into the provided map.
func loadEnvFile(filePath string, envMap map[string]string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := parseEnvLines(string(content))
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Simple key=value parsing
		var key, val string
		for i, char := range line {
			if char == '=' {
				key = line[:i]
				val = line[i+1:]
				break
			}
		}
		if key != "" {
			envMap[key] = val
		}
	}

	return nil
}

// parseEnvLines splits env file content into individual lines, handling basic cases.
func parseEnvLines(content string) []string {
	var lines []string
	var current string

	for _, char := range content {
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		} else if char != '\r' {
			current += string(char)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

// parseEnvSecretsMapping extracts and validates the x-podman-env-secrets extension.
// Maps Podman secret names to environment variable names.
// Returns a map of secret name -> env var name, or an error if invalid.
func parseEnvSecretsMapping(serviceName string, service types.ServiceConfig) (map[string]string, error) {
	if service.Extensions == nil {
		return map[string]string{}, nil
	}

	envSecretsRaw, ok := service.Extensions["x-podman-env-secrets"]
	if !ok || envSecretsRaw == nil {
		return map[string]string{}, nil
	}

	// Parse as map[string]interface{}: secret name -> env var name
	envSecretsMap, ok := envSecretsRaw.(map[string]interface{})
	if !ok {
		return nil, &validationError{
			message: fmt.Sprintf(
				"invalid x-podman-env-secrets in service %q: must be an object, got %T",
				serviceName, envSecretsRaw,
			),
		}
	}

	result := make(map[string]string)
	for secretName, envVarRaw := range envSecretsMap {
		envVar, ok := envVarRaw.(string)
		if !ok {
			return nil, &validationError{
				message: fmt.Sprintf(
					"invalid x-podman-env-secrets in service %q: environment variable name for %q must be string, got %T",
					serviceName, secretName, envVarRaw,
				),
			}
		}

		// Validate secret name: ^[a-zA-Z0-9_.-]+$
		if !isValidSecretName(secretName) {
			return nil, &validationError{
				message: fmt.Sprintf(
					"invalid x-podman-env-secrets in service %q: invalid secret name %q (must match ^[a-zA-Z0-9_.-]+$)",
					serviceName, secretName,
				),
			}
		}

		// Validate env variable name: ^[A-Z_][A-Z0-9_]*$
		if !isValidEnvVarName(envVar) {
			return nil, &validationError{
				message: fmt.Sprintf(
					"invalid x-podman-env-secrets in service %q: invalid environment variable name %q (must match ^[A-Z_][A-Z0-9_]*$)",
					serviceName, envVar,
				),
			}
		}

		result[secretName] = envVar
	}

	return result, nil
}

// isValidEnvVarName validates environment variable naming: ^[A-Z_][A-Z0-9_]*$ .
func isValidEnvVarName(name string) bool {
	if name == "" {
		return false
	}
	first := rune(name[0])
	if first != '_' && !isUpperAlpha(first) {
		return false
	}
	for _, ch := range name[1:] {
		if !isUpperAlphaNumeric(ch) && ch != '_' {
			return false
		}
	}
	return true
}

// isValidSecretName validates Podman secret naming: ^[a-zA-Z0-9_.-]+$ .
func isValidSecretName(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range name {
		if !isAlphaNumeric(ch) && ch != '_' && ch != '.' && ch != '-' {
			return false
		}
	}
	return true
}

// isUpperAlpha checks if a rune is an uppercase letter.
func isUpperAlpha(ch rune) bool {
	return ch >= 'A' && ch <= 'Z'
}

// isUpperAlphaNumeric checks if a rune is an uppercase letter or digit.
func isUpperAlphaNumeric(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

// parseServiceDependencies extracts intra-project dependencies from depends_on
// and parses env secrets for all services in the project.
func parseServiceDependencies(ctx context.Context, project *types.Project) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for serviceName, service := range project.Services {
		// Parse intra-project dependencies from depends_on
		intraProjectDeps := parseIntraProjectDependencies(service)

		// Parse environment variable to secret mappings from x-podman-env-secrets extension
		envSecrets, err := parseEnvSecretsMapping(serviceName, service)
		if err != nil {
			return err
		}

		// Store dependencies back in service for later access
		if service.Extensions == nil {
			service.Extensions = make(map[string]any)
		}

		if len(intraProjectDeps) > 0 {
			service.Extensions["x-quad-ops-dependencies"] = intraProjectDeps
		}

		// Store parsed env secrets mapping for systemd conversion
		if len(envSecrets) > 0 {
			service.Extensions["x-quad-ops-env-secrets"] = envSecrets
		}
	}

	return nil
}

// parseIntraProjectDependencies extracts depends_on conditions from a service.
// Returns a map of service name to condition string (e.g., "service_started").
func parseIntraProjectDependencies(service types.ServiceConfig) map[string]string {
	intraProjectDeps := make(map[string]string)

	for serviceName, dep := range service.DependsOn {
		condition := dep.Condition
		if condition == "" {
			condition = "service_started" // compose-spec default
		}
		intraProjectDeps[serviceName] = condition
	}

	return intraProjectDeps
}

// isAlphaNumeric checks if a rune is a letter (upper or lowercase) or digit.
func isAlphaNumeric(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}
