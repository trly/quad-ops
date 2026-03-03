package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/git"
	"github.com/trly/quad-ops/internal/podman"
	"github.com/trly/quad-ops/internal/state"
	"github.com/trly/quad-ops/internal/systemd"
)

// SyncCmd represents the sync command that processes repositories and writes systemd unit files.
type SyncCmd struct {
	Rollback bool `help:"rollback to the previous known good configuration" default:"false"`
}

// repoResult holds the per-repository outputs accumulated during sync/rollback.
type repoResult struct {
	services   []string
	images     []string
	unitStates map[string]state.UnitState
}

// syncResult accumulates the outputs from processing all repositories.
type syncResult struct {
	oldManagedUnits map[string]struct{}
	newUnitStates   map[string]state.UnitState
	servicesToStart []string
	images          []string
	failed          int
	action          string
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
		return s.runLoop(globals, deployState, stateFilePath, s.rollbackRepo, "rollback")
	}

	return s.runLoop(globals, deployState, stateFilePath, s.syncRepo, "sync")
}

// repoProcessor processes a single repository and returns its result.
// Returning a nil *repoResult signals the repo should be skipped (not counted as failure).
type repoProcessor func(ctx context.Context, globals *Globals, deployState *state.State, repo repoConfig, repoPath string) (*repoResult, error)

// repoConfig captures the per-repo fields needed by processors.
type repoConfig struct {
	Name       string
	URL        string
	Ref        string
	ComposeDir string
}

// runLoop is the shared loop for sync and rollback. It iterates over
// configured repositories, calls the provided processor for each, then
// finalizes (stale cleanup, daemon reload, service start/restart).
func (s *SyncCmd) runLoop(globals *Globals, deployState *state.State, stateFilePath string, process repoProcessor, action string) error {
	ctx := context.Background()

	sr := &syncResult{
		oldManagedUnits: deployState.CollectAllManagedUnits(),
		newUnitStates:   make(map[string]state.UnitState),
		action:          action,
	}
	imageSet := make(map[string]struct{})

	for _, repo := range globals.AppCfg.Repositories {
		repoPath := filepath.Join(globals.AppCfg.GetRepositoryDir(), repo.Name)
		rc := repoConfig{Name: repo.Name, URL: repo.URL, Ref: repo.Ref, ComposeDir: repo.ComposeDir}

		result, err := process(ctx, globals, deployState, rc, repoPath)
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
			sr.failed++
			continue
		}
		if result == nil {
			continue
		}

		sr.servicesToStart = append(sr.servicesToStart, result.services...)
		for _, img := range result.images {
			imageSet[img] = struct{}{}
		}
		for k, v := range result.unitStates {
			sr.newUnitStates[k] = v
		}
	}

	// Prune managed units for repos removed from config
	configuredRepos := make(map[string]struct{}, len(globals.AppCfg.Repositories))
	for _, repo := range globals.AppCfg.Repositories {
		configuredRepos[repo.Name] = struct{}{}
	}
	deployState.PruneRemovedRepos(configuredRepos)

	sr.images = make([]string, 0, len(imageSet))
	for img := range imageSet {
		sr.images = append(sr.images, img)
	}

	return s.finalize(ctx, globals, deployState, stateFilePath, sr)
}

// syncRepo processes a single repository for the normal sync path.
func (s *SyncCmd) syncRepo(ctx context.Context, globals *Globals, deployState *state.State, repo repoConfig, repoPath string) (*repoResult, error) {
	if globals.Verbose {
		fmt.Printf("Syncing repository: %s\n", repo.Name)
	}

	gitRepo := git.New(repo.Name, repo.URL, repo.Ref, repo.ComposeDir, repoPath)
	if err := gitRepo.Sync(ctx); err != nil {
		return nil, fmt.Errorf("failed to sync git repository: %w", err)
	}

	commitHash, err := gitRepo.GetCurrentCommitHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit hash: %w", err)
	}
	if globals.Verbose {
		fmt.Printf("  Current revision: %s\n", commitHash[:7])
	}

	unitNames, images, unitStates, genErr := s.generateUnits(ctx, globals, repo.ComposeDir, repoPath)

	deployState.SetCommit(repo.Name, commitHash)
	deployState.SetManagedUnits(repo.Name, unitNames)

	if genErr != nil {
		return nil, genErr
	}

	return &repoResult{
		services:   containerServices(unitNames),
		images:     images,
		unitStates: unitStates,
	}, nil
}

// rollbackRepo processes a single repository for the rollback path.
func (s *SyncCmd) rollbackRepo(ctx context.Context, globals *Globals, deployState *state.State, repo repoConfig, repoPath string) (*repoResult, error) {
	prev := deployState.GetPrevious(repo.Name)
	if prev == "" {
		fmt.Printf("  WARNING: no previous state for %s, skipping\n", repo.Name)
		return nil, nil
	}

	if globals.Verbose {
		fmt.Printf("Rolling back repository: %s to %s\n", repo.Name, prev[:7])
	}

	gitRepo := git.New(repo.Name, repo.URL, prev, repo.ComposeDir, repoPath)
	if err := gitRepo.CheckoutRef(prev); err != nil {
		return nil, err
	}

	unitNames, images, unitStates, genErr := s.generateUnits(ctx, globals, repo.ComposeDir, repoPath)

	deployState.SetCommit(repo.Name, prev)
	deployState.SetManagedUnits(repo.Name, unitNames)

	if genErr != nil {
		return nil, genErr
	}

	return &repoResult{
		services:   containerServices(unitNames),
		images:     images,
		unitStates: unitStates,
	}, nil
}

// finalize performs post-sync/rollback cleanup: stale unit removal, state
// persistence, systemd daemon reload, restart of changed services, and
// service activation.
func (s *SyncCmd) finalize(ctx context.Context, globals *Globals, deployState *state.State, stateFilePath string, sr *syncResult) error {
	newManagedUnits := deployState.CollectAllManagedUnits()
	staleUnits := state.DiffUnits(sr.oldManagedUnits, newManagedUnits)

	client, err := systemd.New(ctx, systemd.ScopeAuto)
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer func() { _ = client.Close() }()

	if len(staleUnits) > 0 {
		s.cleanupStaleUnits(ctx, globals, deployState, client, staleUnits)
	}

	// Determine which services need restart before updating stored hashes
	changedServices := containerServices(deployState.ChangedUnits(sr.newUnitStates))

	// Update stored unit states
	for name, us := range sr.newUnitStates {
		deployState.SetUnitState(name, us)
	}

	if err := deployState.Save(stateFilePath); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if err := client.DaemonReload(ctx); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	if globals.Verbose {
		fmt.Println("Reloaded systemd daemon")
	}

	pullResult, err := podman.PullImages(sr.images, deployState.ImageDigests, globals.Verbose)
	if err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}
	for image, digest := range pullResult.UpdatedDigests {
		deployState.SetImageDigest(image, digest)
	}
	if len(pullResult.UpdatedDigests) > 0 {
		if err := deployState.Save(stateFilePath); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}
	}

	// Restart services whose unit definitions or bind-mounted files changed
	if len(changedServices) > 0 {
		if globals.Verbose {
			fmt.Printf("Restarting %d changed service(s)...\n", len(changedServices))
		}
		if err := client.Restart(ctx, changedServices...); err != nil {
			return fmt.Errorf("some services failed to restart: %w", err)
		}
		if globals.Verbose {
			fmt.Printf("Restarted %d changed service(s)\n", len(changedServices))
		}
	}

	// Exclude already-restarted services from the start list
	if len(changedServices) > 0 {
		restartedSet := make(map[string]struct{}, len(changedServices))
		for _, svc := range changedServices {
			restartedSet[svc] = struct{}{}
		}
		var remaining []string
		for _, svc := range sr.servicesToStart {
			if _, restarted := restartedSet[svc]; !restarted {
				remaining = append(remaining, svc)
			}
		}
		sr.servicesToStart = remaining
	}

	// Start all services to ensure everything is running.
	if len(sr.servicesToStart) > 0 {
		if globals.Verbose {
			fmt.Printf("Starting %d service(s)...\n", len(sr.servicesToStart))
		}
		if err := client.Start(ctx, sr.servicesToStart...); err != nil {
			return fmt.Errorf("some services failed to start: %w", err)
		}
		if globals.Verbose {
			fmt.Printf("Started %d service(s)\n", len(sr.servicesToStart))
		}
	}

	if sr.failed > 0 {
		return fmt.Errorf("%d repository(ies) failed to %s", sr.failed, sr.action)
	}

	return nil
}

// generateUnits loads compose files, writes the resulting quadlet units,
// and returns the list of unit filenames written, images referenced, and
// unit states for change detection.
func (s *SyncCmd) generateUnits(ctx context.Context, globals *Globals, composeDir, repoPath string) ([]string, []string, map[string]state.UnitState, error) {
	composeSourceDir := repoPath
	if composeDir != "" {
		composeSourceDir = filepath.Join(repoPath, composeDir)
	}

	loadedProjects, err := compose.LoadAll(ctx, composeSourceDir, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load compose files: %w", err)
	}

	if len(loadedProjects) == 0 {
		if globals.Verbose {
			fmt.Printf("  No compose files found in %s\n", composeSourceDir)
		}
		return nil, nil, nil, nil
	}

	quadletDir := globals.AppCfg.GetQuadletDir()
	var unitNames []string
	imageSet := make(map[string]struct{})
	unitStates := make(map[string]state.UnitState)

	for _, lp := range loadedProjects {
		if lp.Error != nil {
			fmt.Printf("  WARNING: failed to load %s: %v\n", lp.FilePath, lp.Error)
			continue
		}

		if lp.Project == nil {
			continue
		}

		skippedSecrets, secretsErr := compose.FilterServicesWithMissingSecrets(ctx, lp.Project, nil)
		if secretsErr != nil {
			fmt.Printf("  WARNING: failed to query podman secrets: %v\n", secretsErr)
		}
		for _, ms := range skippedSecrets {
			fmt.Printf("  WARNING: skipping service %s-%s: missing secrets %v\n",
				lp.Project.Name, ms.ServiceName, ms.MissingSecrets)
		}

		units, err := systemd.Convert(lp.Project)
		if err != nil {
			fmt.Printf("  WARNING: failed to convert %s: %v\n", lp.FilePath, err)
			continue
		}

		if err := systemd.WriteUnits(units, quadletDir); err != nil {
			return unitNames, nil, nil, fmt.Errorf("failed to write units for %s: %w", lp.FilePath, err)
		}

		for _, u := range units {
			unitNames = append(unitNames, u.Name)

			if strings.HasSuffix(u.Name, ".container") {
				svcName := strings.TrimPrefix(u.Name, lp.Project.Name+"-")
				svcName = strings.TrimSuffix(svcName, ".container")
				if svc, ok := lp.Project.Services[svcName]; ok {
					us := systemd.ComputeUnitState(u, &svc, lp.Project.WorkingDir, repoPath)
					unitStates[u.Name] = us
				}
			}
		}

		for _, svc := range lp.Project.Services {
			if svc.Image != "" {
				imageSet[svc.Image] = struct{}{}
			}
		}

		if globals.Verbose {
			fmt.Printf("  Generated %d unit(s) from %s", len(units), filepath.Base(lp.FilePath))
			if len(skippedSecrets) > 0 {
				fmt.Printf(" (skipped %d service(s) with missing secrets)", len(skippedSecrets))
			}
			fmt.Println()
		}
	}

	images := make([]string, 0, len(imageSet))
	for img := range imageSet {
		images = append(images, img)
	}

	return unitNames, images, unitStates, nil
}

// cleanupStaleUnits stops, disables, and removes quadlet unit files
// that are no longer defined by any compose project, and cleans up
// their stored unit states.
func (s *SyncCmd) cleanupStaleUnits(ctx context.Context, globals *Globals, deployState *state.State, client systemd.Client, staleUnits []string) {
	quadletDir := globals.AppCfg.GetQuadletDir()

	servicesToStop := containerServices(staleUnits)

	if len(servicesToStop) > 0 {
		if globals.Verbose {
			fmt.Printf("Stopping %d stale service(s)...\n", len(servicesToStop))
		}
		if err := client.Stop(ctx, servicesToStop...); err != nil {
			fmt.Printf("  WARNING: failed to stop some stale services: %v\n", err)
		}
		if err := client.Disable(ctx, servicesToStop...); err != nil {
			fmt.Printf("  WARNING: failed to disable some stale services: %v\n", err)
		}
	}

	for _, unit := range staleUnits {
		path := filepath.Join(quadletDir, unit)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Printf("  WARNING: failed to remove stale unit %s: %v\n", unit, err)
		} else if globals.Verbose {
			fmt.Printf("  Removed stale unit: %s\n", unit)
		}
		deployState.RemoveUnitState(unit)
	}
}

// containerServices returns the systemd service names for any .container
// units in the provided list (e.g. "app.container" → "app.service").
func containerServices(unitNames []string) []string {
	var services []string
	for _, name := range unitNames {
		if strings.HasSuffix(name, ".container") {
			services = append(services, strings.TrimSuffix(name, ".container")+".service")
		}
	}
	return services
}
