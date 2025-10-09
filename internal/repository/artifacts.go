// Package repository provides data access layer for quad-ops artifacts and sync operations.
package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
)

// ArtifactStore manages writing, reading, and deleting platform artifacts with
// atomic writes and change detection.
type ArtifactStore interface {
	// Write atomically writes artifacts to disk and returns paths that changed.
	// Uses hash comparison to detect changes and atomic write semantics (temp + fsync + rename).
	Write(ctx context.Context, artifacts []platform.Artifact) ([]string, error)

	// List returns all currently stored artifacts by scanning the artifact directory.
	List(ctx context.Context) ([]platform.Artifact, error)

	// Delete removes artifacts at the specified paths.
	Delete(ctx context.Context, paths []string) error
}

// FileArtifactStore implements ArtifactStore using the file system.
type FileArtifactStore struct {
	fsService *fs.Service
	logger    log.Logger
	baseDir   string // Base directory for artifacts (e.g., quadlet dir)
}

// NewArtifactStore creates a new file-based artifact store.
func NewArtifactStore(fsService *fs.Service, logger log.Logger, baseDir string) ArtifactStore {
	return &FileArtifactStore{
		fsService: fsService,
		logger:    logger,
		baseDir:   baseDir,
	}
}

// Write atomically writes artifacts to disk and returns paths that changed.
func (s *FileArtifactStore) Write(ctx context.Context, artifacts []platform.Artifact) ([]string, error) {
	changedPaths := make([]string, 0, len(artifacts))

	for _, artifact := range artifacts {
		select {
		case <-ctx.Done():
			return changedPaths, ctx.Err()
		default:
		}

		targetPath := filepath.Join(s.baseDir, artifact.Path)

		// Check if content changed using hash comparison
		if !s.hasChanged(targetPath, artifact.Content, artifact.Hash) {
			s.logger.Debug("Artifact unchanged, skipping", "path", artifact.Path)
			continue
		}

		// Perform atomic write: temp file -> fsync -> rename
		if err := s.atomicWrite(targetPath, artifact.Content, artifact.Mode); err != nil {
			return changedPaths, fmt.Errorf("writing artifact %s: %w", artifact.Path, err)
		}

		s.logger.Debug("Artifact written", "path", artifact.Path, "hash", artifact.Hash)
		changedPaths = append(changedPaths, artifact.Path)
	}

	return changedPaths, nil
}

// List returns all currently stored artifacts by scanning the artifact directory.
func (s *FileArtifactStore) List(ctx context.Context) ([]platform.Artifact, error) {
	var artifacts []platform.Artifact

	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			// Log and continue on walk errors
			s.logger.Debug("Error walking directory", "path", path, "error", err)
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Read artifact content
		content, err := os.ReadFile(path) //nolint:gosec // Path validated through filepath.Walk
		if err != nil {
			s.logger.Debug("Error reading artifact", "path", path, "error", err)
			return nil
		}

		// Calculate relative path from base directory
		relPath, err := filepath.Rel(s.baseDir, path)
		if err != nil {
			s.logger.Debug("Error calculating relative path", "path", path, "error", err)
			return nil
		}

		// Calculate content hash
		hash := fmt.Sprintf("%x", s.calculateHash(content))

		artifact := platform.Artifact{
			Path:    relPath,
			Content: content,
			Mode:    info.Mode(),
			Hash:    hash,
		}

		artifacts = append(artifacts, artifact)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking artifact directory: %w", err)
	}

	return artifacts, nil
}

// Delete removes artifacts at the specified paths.
func (s *FileArtifactStore) Delete(ctx context.Context, paths []string) error {
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		targetPath := filepath.Join(s.baseDir, path)

		if err := os.Remove(targetPath); err != nil {
			if os.IsNotExist(err) {
				s.logger.Debug("Artifact already deleted", "path", path)
				continue
			}
			return fmt.Errorf("deleting artifact %s: %w", path, err)
		}

		s.logger.Debug("Artifact deleted", "path", path)
	}

	return nil
}

// hasChanged checks if an artifact's content has changed using hash comparison.
func (s *FileArtifactStore) hasChanged(targetPath string, newContent []byte, expectedHash string) bool {
	existingContent, err := os.ReadFile(targetPath) //nolint:gosec // Path is constructed internally
	if err != nil {
		// File doesn't exist or can't be read, so it has changed
		return true
	}

	// Calculate hash of existing content
	existingHash := fmt.Sprintf("%x", s.calculateHash(existingContent))

	// If hash is provided in artifact, use it; otherwise calculate new hash
	var newHash string
	if expectedHash != "" {
		newHash = expectedHash
	} else {
		newHash = fmt.Sprintf("%x", s.calculateHash(newContent))
	}

	s.logger.Debug("Content hash comparison",
		"existing", existingHash,
		"new", newHash,
		"path", targetPath)

	return existingHash != newHash
}

// atomicWrite writes content atomically using temp file -> fsync -> rename pattern.
func (s *FileArtifactStore) atomicWrite(targetPath string, content []byte, mode os.FileMode) error {
	// Ensure parent directory exists
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0750); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Create temporary file in same directory (required for atomic rename)
	tempFile, err := os.CreateTemp(parentDir, ".artifact-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure cleanup on failure
	defer func() {
		_ = os.Remove(tempPath) // Cleanup - ignore error
	}()

	// Write content to temp file
	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close() // Cleanup - ignore error
		return fmt.Errorf("writing to temp file: %w", err)
	}

	// Set file permissions
	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close() // Cleanup - ignore error
		return fmt.Errorf("setting file permissions: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close() // Cleanup - ignore error
		return fmt.Errorf("syncing temp file: %w", err)
	}

	// Close temp file before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Atomic rename to target path
	if err := os.Rename(tempPath, targetPath); err != nil {
		return fmt.Errorf("renaming temp file to target: %w", err)
	}

	return nil
}

// calculateHash computes SHA256 hash for content comparison.
func (s *FileArtifactStore) calculateHash(content []byte) []byte {
	hash := sha256.New()
	hash.Write(content)
	return hash.Sum(nil)
}
