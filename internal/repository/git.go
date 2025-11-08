// Package repository provides data access layer for quad-ops artifacts and sync operations.
package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/log"
)

// SyncResult contains the result of syncing a single repository.
type SyncResult struct {
	Repository config.Repository // The repository that was synced
	Success    bool              // Whether the sync succeeded
	Error      error             // Error if sync failed
	Changed    bool              // Whether the repository changed (new commits)
	CommitHash string            // Current commit hash after sync
}

// GitSyncer manages synchronization of git repositories.
type GitSyncer interface {
	// SyncAll syncs all repositories in parallel and returns results for each.
	SyncAll(ctx context.Context, repos []config.Repository) ([]SyncResult, error)

	// SyncRepo syncs a single repository and returns the result.
	SyncRepo(ctx context.Context, repo config.Repository) SyncResult
}

// DefaultGitSyncer implements GitSyncer using the internal/git package.
type DefaultGitSyncer struct {
	configProvider config.Provider
	logger         log.Logger
}

// NewGitSyncer creates a new git syncer with dependency injection.
func NewGitSyncer(configProvider config.Provider, logger log.Logger) GitSyncer {
	return &DefaultGitSyncer{
		configProvider: configProvider,
		logger:         logger,
	}
}

// SyncAll syncs all repositories in parallel and returns results for each.
func (s *DefaultGitSyncer) SyncAll(ctx context.Context, repos []config.Repository) ([]SyncResult, error) {
	results := make([]SyncResult, len(repos))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, repo := range repos {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		wg.Add(1)
		go func(index int, repository config.Repository) {
			defer wg.Done()

			result := s.SyncRepo(ctx, repository)

			mu.Lock()
			results[index] = result
			mu.Unlock()
		}(i, repo)
	}

	wg.Wait()
	return results, nil
}

// SyncRepo syncs a single repository and returns the result.
func (s *DefaultGitSyncer) SyncRepo(ctx context.Context, repo config.Repository) SyncResult {
	result := SyncResult{
		Repository: repo,
		Success:    false,
		Changed:    false,
	}

	// Check context before starting
	select {
	case <-ctx.Done():
		result.Error = ctx.Err()
		return result
	default:
	}

	s.logger.Debug("Syncing repository", "name", repo.Name, "url", repo.URL)

	// Create git repository instance
	gitRepo := git.NewGitRepositoryWithLogger(repo, s.configProvider, s.logger)

	// Get current commit hash before sync (if repo exists)
	beforeHash, _ := s.getCurrentCommit(gitRepo)

	// Perform sync operation
	if err := gitRepo.SyncRepository(); err != nil {
		result.Error = fmt.Errorf("syncing repository %s: %w", repo.Name, err)
		s.logger.Debug("Repository sync failed", "name", repo.Name, "error", err)
		return result
	}

	// Get commit hash after sync
	afterHash, err := s.getCurrentCommit(gitRepo)
	if err != nil {
		result.Error = fmt.Errorf("getting commit hash for %s: %w", repo.Name, err)
		s.logger.Debug("Failed to get commit hash", "name", repo.Name, "error", err)
		return result
	}

	result.Success = true
	result.CommitHash = afterHash
	result.Changed = beforeHash != afterHash

	s.logger.Debug("Repository synced successfully",
		"name", repo.Name,
		"changed", result.Changed,
		"commit", result.CommitHash)

	return result
}

// getCurrentCommit gets the current commit hash from a git repository.
func (s *DefaultGitSyncer) getCurrentCommit(repo *git.Repository) (string, error) {
	if repo == nil {
		return "", nil
	}

	return repo.GetCurrentCommitHash()
}
