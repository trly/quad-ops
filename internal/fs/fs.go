// Package fs provides file system operations for quadlet unit management
package fs

import (
	"crypto/sha1" //nolint:gosec // Not used for security purposes, just content comparison
	"fmt"
	"os"
	"path/filepath"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/log"
)

// Service provides file system operations with configurable paths.
type Service struct {
	configProvider config.Provider
	logger         log.Logger
}

// NewService creates a new filesystem service with the given config provider.
func NewService(configProvider config.Provider) *Service {
	return &Service{
		configProvider: configProvider,
		logger:         log.NewLogger(configProvider.GetConfig().Verbose),
	}
}

// NewServiceWithLogger creates a new filesystem service with explicit logger injection.
func NewServiceWithLogger(configProvider config.Provider, logger log.Logger) *Service {
	return &Service{
		configProvider: configProvider,
		logger:         logger,
	}
}

// GetUnitFilePath returns the full path for a quadlet unit file.
func (s *Service) GetUnitFilePath(name, unitType string) string {
	return filepath.Join(s.configProvider.GetConfig().QuadletDir, fmt.Sprintf("%s.%s", name, unitType))
}

// GetUnitFilesDirectory returns the directory where quadlet unit files are stored.
func (s *Service) GetUnitFilesDirectory() string {
	return s.configProvider.GetConfig().QuadletDir
}

// HasUnitChanged checks if the content of a unit file has changed.
func (s *Service) HasUnitChanged(unitPath, content string) bool {
	existingContent, err := os.ReadFile(unitPath) //nolint:gosec // Safe as path is internally constructed, not user-controlled
	if err != nil {
		// File doesn't exist or can't be read, so it has changed
		return true
	}

	// If verbose logging is enabled, print hash comparison details
	s.logger.Debug("Content hash comparison",
		"existing", fmt.Sprintf("%x", GetContentHash(string(existingContent))),
		"new", fmt.Sprintf("%x", GetContentHash(content)))

	// Compare the actual content directly instead of hashes
	if string(existingContent) == content {
		s.logger.Debug("Unit unchanged, skipping", "path", unitPath)
		return false
	}

	// Content is different
	return true
}

// WriteUnitFile writes unit content to the specified file path.
func (s *Service) WriteUnitFile(unitPath, content string) error {
	s.logger.Debug("Writing quadlet unit", "path", unitPath)

	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(unitPath), 0750); err != nil {
		return fmt.Errorf("failed to create quadlet directory: %w", err)
	}

	return os.WriteFile(unitPath, []byte(content), 0600)
}

// GetContentHash calculates a SHA1 hash for content storage and change tracking.
func (s *Service) GetContentHash(content string) string {
	return string(GetContentHash(content))
}

// GetContentHash calculates a SHA1 hash for content storage and change tracking.
func GetContentHash(content string) []byte {
	hash := sha1.New() //nolint:gosec // Not used for security purposes, just for content tracking
	hash.Write([]byte(content))
	return hash.Sum(nil)
}
