package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/state"
	"github.com/trly/quad-ops/internal/systemd"
)

// SyncCmd represents the sync command that processes repositories and writes systemd unit files.
type SyncCmd struct {
	Rollback bool `help:"rollback to the previous known good configuration" default:"false"`
}

// Run executes the sync command by:
// 1. Processing each repository's current revision.
// 2. Loading compose files from the repository.
// 3. Converting compose specs to systemd units.
// 4. Writing units to the quadlet directory.
func (s *SyncCmd) Run(globals *Globals) error {
	if globals.AppCfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	if len(globals.AppCfg.Repositories) == 0 {
		if globals.Verbose {
			fmt.Println("No repositories configured")
		}
		return nil
	}

	stateFilePath := globals.AppCfg.GetStateFilePath()
	deployState, err := state.Load(stateFilePath)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	if s.Rollback {
		return s.runRollback(globals, deployState, stateFilePath)
	}

	return s.runSync(globals, deployState, stateFilePath)
}

// runSync performs the normal sync: pull latest, generate units, record state.
func (s *SyncCmd) runSync(globals *Globals, deployState *state.State, stateFilePath string) error {
	ctx := context.Background()
	failed := 0

	for _, repo := range globals.AppCfg.Repositories {
		repoPath := filepath.Join(globals.AppCfg.GetRepositoryDir(), repo.Name)

		if err := s.syncRepository(ctx, globals, deployState, repo.Name, repo.URL, repo.Ref, repo.ComposeDir, repoPath); err != nil {
			fmt.Printf("  ERROR: %v\n", err)
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d repository(ies) failed to sync", failed)
	}

	if err := deployState.Save(stateFilePath); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Reload systemd daemon to pick up new/changed units
	client, err := systemd.New(ctx, systemd.ScopeAuto)
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.DaemonReload(ctx); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	if globals.Verbose {
		fmt.Println("Reloaded systemd daemon")
	}

	return nil
}

// runRollback restores each repository to its previous commit and regenerates units.
func (s *SyncCmd) runRollback(globals *Globals, deployState *state.State, stateFilePath string) error {
	ctx := context.Background()
	failed := 0

	for _, repo := range globals.AppCfg.Repositories {
		repoPath := filepath.Join(globals.AppCfg.GetRepositoryDir(), repo.Name)
		prev := deployState.GetPrevious(repo.Name)
		if prev == "" {
			fmt.Printf("  WARNING: no previous state for %s, skipping\n", repo.Name)
			continue
		}

		if globals.Verbose {
			fmt.Printf("Rolling back repository: %s to %s\n", repo.Name, prev[:7])
		}

		gitRepo := git.New(repo.Name, repo.URL, prev, repo.ComposeDir, repoPath)
		if err := gitRepo.CheckoutRef(prev); err != nil {
			fmt.Printf("  ERROR: %v\n", err)
			failed++
			continue
		}

		if err := s.generateUnits(ctx, globals, gitRepo, repo.ComposeDir, repoPath); err != nil {
			fmt.Printf("  ERROR: %v\n", err)
			failed++
			continue
		}

		// Swap current and previous so another rollback goes forward again
		deployState.SetCommit(repo.Name, prev)
	}

	if failed > 0 {
		return fmt.Errorf("%d repository(ies) failed to rollback", failed)
	}

	if err := deployState.Save(stateFilePath); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	client, err := systemd.New(ctx, systemd.ScopeAuto)
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.DaemonReload(ctx); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	if globals.Verbose {
		fmt.Println("Reloaded systemd daemon")
	}

	return nil
}

// syncRepository processes a single repository and writes its units to quadletdir.
func (s *SyncCmd) syncRepository(ctx context.Context, globals *Globals, deployState *state.State, name, url, ref, composeDir, repoPath string) error {
	if globals.Verbose {
		fmt.Printf("Syncing repository: %s\n", name)
	}

	// Sync the repository to the latest state
	gitRepo := git.New(name, url, ref, composeDir, repoPath)
	if err := gitRepo.Sync(ctx); err != nil {
		return fmt.Errorf("failed to sync git repository: %w", err)
	}

	// Get the current commit hash
	commitHash, err := gitRepo.GetCurrentCommitHash()
	if err != nil {
		return fmt.Errorf("failed to get current commit hash: %w", err)
	}
	if globals.Verbose {
		fmt.Printf("  Current revision: %s\n", commitHash[:7])
	}

	if err := s.generateUnits(ctx, globals, gitRepo, composeDir, repoPath); err != nil {
		return err
	}

	deployState.SetCommit(name, commitHash)
	return nil
}

// generateUnits loads compose files and writes the resulting quadlet units.
func (s *SyncCmd) generateUnits(ctx context.Context, globals *Globals, _ *git.Repository, composeDir, repoPath string) error {
	// Determine the directory containing compose files
	composeSourceDir := repoPath
	if composeDir != "" {
		composeSourceDir = filepath.Join(repoPath, composeDir)
	}

	// Load all compose projects from the repository
	loadedProjects, err := compose.LoadAll(ctx, composeSourceDir, nil)
	if err != nil {
		return fmt.Errorf("failed to load compose files: %w", err)
	}

	if len(loadedProjects) == 0 {
		if globals.Verbose {
			fmt.Printf("  No compose files found in %s\n", composeSourceDir)
		}
		return nil
	}

	quadletDir := globals.AppCfg.GetQuadletDir()

	// Process each loaded project
	for _, lp := range loadedProjects {
		if lp.Error != nil {
			fmt.Printf("  WARNING: failed to load %s: %v\n", lp.FilePath, lp.Error)
			continue
		}

		if lp.Project == nil {
			continue
		}

		// Filter out services with missing secrets
		skippedSecrets, secretsErr := compose.FilterServicesWithMissingSecrets(ctx, lp.Project, nil)
		if secretsErr != nil {
			fmt.Printf("  WARNING: failed to query podman secrets: %v\n", secretsErr)
		}
		for _, ms := range skippedSecrets {
			fmt.Printf("  WARNING: skipping service %s-%s: missing secrets %v\n",
				lp.Project.Name, ms.ServiceName, ms.MissingSecrets)
		}

		// Convert compose project to systemd units
		units, err := systemd.Convert(lp.Project)
		if err != nil {
			return fmt.Errorf("failed to convert compose project %s: %w", lp.FilePath, err)
		}

		// Write each unit to a file in quadletdir
		if err := s.writeUnits(units, quadletDir); err != nil {
			return fmt.Errorf("failed to write units for %s: %w", lp.FilePath, err)
		}

		if globals.Verbose {
			fmt.Printf("  Generated %d unit(s) from %s", len(units), filepath.Base(lp.FilePath))
			if len(skippedSecrets) > 0 {
				fmt.Printf(" (skipped %d service(s) with missing secrets)", len(skippedSecrets))
			}
			fmt.Println()
		}
	}

	return nil
}

// writeUnits writes each unit to a separate file in the quadlet directory.
func (s *SyncCmd) writeUnits(units []systemd.Unit, quadletDir string) error {
	if err := os.MkdirAll(quadletDir, 0o755); err != nil {
		return fmt.Errorf("failed to create quadlet directory: %w", err)
	}

	for _, unit := range units {
		filename := filepath.Join(quadletDir, unit.Name)

		f, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create unit file %s: %w", filename, err)
		}

		if err := unit.WriteUnit(f); err != nil {
			_ = f.Close()
			return fmt.Errorf("failed to write unit file %s: %w", filename, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close unit file %s: %w", filename, err)
		}
	}

	return nil
}
