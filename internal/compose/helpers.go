package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Prefix creates a prefixed resource name using project name for consistency.
func Prefix(projectName, resourceName string) string {
	return fmt.Sprintf("%s-%s", projectName, resourceName)
}

// FindEnvFiles discovers environment files for a service in a working directory.
func FindEnvFiles(serviceName, workingDir string) []string {
	if workingDir == "" {
		return nil
	}

	var envFiles []string

	// General .env file
	generalEnvFile := filepath.Join(workingDir, ".env")
	if _, err := os.Stat(generalEnvFile); err == nil {
		envFiles = append(envFiles, generalEnvFile)
	}

	// Service-specific .env files
	possibleEnvFiles := []string{
		filepath.Join(workingDir, fmt.Sprintf(".env.%s", serviceName)),
		filepath.Join(workingDir, fmt.Sprintf("%s.env", serviceName)),
		filepath.Join(workingDir, "env", fmt.Sprintf("%s.env", serviceName)),
		filepath.Join(workingDir, "envs", fmt.Sprintf("%s.env", serviceName)),
	}

	for _, envFilePath := range possibleEnvFiles {
		if _, err := os.Stat(envFilePath); err == nil {
			envFiles = append(envFiles, envFilePath)
		}
	}

	return envFiles
}

// HasNamingConflict checks for potential naming conflicts with existing units.
func HasNamingConflict(repo Repository, unitName, unitType string) bool {
	existingUnits, err := repo.FindAll()
	if err != nil {
		return false
	}

	for _, existingUnit := range existingUnits {
		// If an existing unit with the same type exists that almost matches but differs in naming scheme
		if existingUnit.Type == unitType &&
			existingUnit.Name != unitName &&
			(strings.HasSuffix(existingUnit.Name, unitName) || strings.HasSuffix(unitName, existingUnit.Name)) {
			// Debug logging removed to avoid dependency on global logger in utility function
			return true
		}
	}
	return false
}

// IsExternal checks if a resource configuration indicates it's externally managed.
func IsExternal(external interface{}) bool {
	if external == nil {
		return false
	}

	switch v := external.(type) {
	case bool:
		return v
	case *bool:
		return v != nil && *v
	default:
		// Handle types.External which is a custom bool type with underlying bool
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Bool {
			return rv.Bool()
		}
		return false
	}
}
