// Package sorting provides common utility functions for the application.
package sorting

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidateUnitName validates that a unit name is safe for use in shell commands.
// Unit names must follow systemd naming conventions to prevent command injection.
func ValidateUnitName(unitName string) error {
	if unitName == "" {
		return fmt.Errorf("unit name cannot be empty")
	}

	// Systemd unit names must match this pattern: alphanumeric, dots, dashes, underscores, @, and colons
	// This prevents injection of shell metacharacters like ;, |, &, $, etc.
	validUnitName := regexp.MustCompile(`^[a-zA-Z0-9._@:-]+$`)
	if !validUnitName.MatchString(unitName) {
		return fmt.Errorf("invalid unit name: contains unsafe characters")
	}

	// Additional length check to prevent extremely long names
	if len(unitName) > 256 {
		return fmt.Errorf("unit name too long")
	}

	return nil
}

// ValidatePath validates that a path doesn't contain path traversal sequences.
// It uses filepath.Clean to normalize the path and checks for traversal attempts.
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path to normalize it and resolve any traversal sequences
	cleanPath := filepath.Clean(path)

	// If the cleaned path is different and contains traversal, it's suspicious
	if cleanPath != path && strings.Contains(path, "..") {
		return fmt.Errorf("path contains path traversal sequence")
	}

	// Check if the cleaned path tries to go above the current directory for relative paths
	if !filepath.IsAbs(cleanPath) && strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("path attempts to traverse above working directory")
	}

	return nil
}

// ValidatePathWithinBase ensures a path stays within a base directory after cleaning.
// This is more secure than ValidatePath alone for critical file operations.
func ValidatePathWithinBase(path, basePath string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if basePath == "" {
		return "", fmt.Errorf("base path cannot be empty")
	}

	// Clean both paths to normalize them
	cleanPath := filepath.Clean(path)
	cleanBase := filepath.Clean(basePath)

	// Make paths absolute for proper comparison
	absBase, err := filepath.Abs(cleanBase)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}

	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(absBase, cleanPath)
	}

	// Clean the final path
	absPath = filepath.Clean(absPath)

	// Ensure the final path is within the base directory
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", fmt.Errorf("path escapes base directory")
	}

	return absPath, nil
}
