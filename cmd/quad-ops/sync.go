package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
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

// runSync performs the normal sync: pull latest, generate units, record state,
// then restart changed services and start all services.
func (s *SyncCmd) runSync(globals *Globals, deployState *state.State, stateFilePath string) error {
	ctx := context.Background()
	failed := 0

	// Snapshot all currently managed units before sync
	oldManagedUnits := collectAllManagedUnits(deployState)

	// Track services and images to start for repos with zero failures
	var servicesToStart []string
	imageSet := make(map[string]struct{})
	allUnitStates := make(map[string]state.UnitState)

	for _, repo := range globals.AppCfg.Repositories {
		repoPath := filepath.Join(globals.AppCfg.GetRepositoryDir(), repo.Name)

		services, images, unitStates, err := s.syncRepository(ctx, globals, deployState, repo.Name, repo.URL, repo.Ref, repo.ComposeDir, repoPath)
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
			failed++
			continue
		}

		servicesToStart = append(servicesToStart, services...)
		for _, img := range images {
			imageSet[img] = struct{}{}
		}
		for k, v := range unitStates {
			allUnitStates[k] = v
		}
	}

	// Clear managed units for repos no longer in config
	configuredRepos := make(map[string]struct{})
	for _, repo := range globals.AppCfg.Repositories {
		configuredRepos[repo.Name] = struct{}{}
	}
	for repoName := range deployState.ManagedUnits {
		if _, ok := configuredRepos[repoName]; !ok {
			deployState.SetManagedUnits(repoName, nil)
		}
	}

	images := make([]string, 0, len(imageSet))
	for img := range imageSet {
		images = append(images, img)
	}

	return s.finalize(ctx, globals, deployState, stateFilePath, oldManagedUnits, allUnitStates, servicesToStart, images, failed, "sync")
}

// runRollback restores each repository to its previous commit and regenerates units.
func (s *SyncCmd) runRollback(globals *Globals, deployState *state.State, stateFilePath string) error {
	ctx := context.Background()
	failed := 0

	// Snapshot all currently managed units before rollback
	oldManagedUnits := collectAllManagedUnits(deployState)

	// Track services and images to start for repos with zero failures
	var servicesToStart []string
	imageSet := make(map[string]struct{})
	allUnitStates := make(map[string]state.UnitState)

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

		unitNames, images, unitStates, genErr := s.generateUnits(ctx, globals, repo.ComposeDir, repoPath)

		// Always record state to reflect what is actually on disk so that
		// stale unit detection remains accurate even after partial failures.
		deployState.SetCommit(repo.Name, prev)
		deployState.SetManagedUnits(repo.Name, unitNames)

		if genErr != nil {
			fmt.Printf("  ERROR: %v\n", genErr)
			failed++
			continue
		}

		// Only start services if all compose files in this repo passed
		for _, name := range unitNames {
			if strings.HasSuffix(name, ".container") {
				serviceName := strings.TrimSuffix(name, ".container") + ".service"
				servicesToStart = append(servicesToStart, serviceName)
			}
		}
		for _, img := range images {
			imageSet[img] = struct{}{}
		}
		for k, v := range unitStates {
			allUnitStates[k] = v
		}
	}

	allImages := make([]string, 0, len(imageSet))
	for img := range imageSet {
		allImages = append(allImages, img)
	}

	return s.finalize(ctx, globals, deployState, stateFilePath, oldManagedUnits, allUnitStates, servicesToStart, allImages, failed, "rollback")
}

// syncRepository processes a single repository and writes its units to quadletdir.
// Returns the list of service units to start, images to pull, unit states, for this repo.
func (s *SyncCmd) syncRepository(ctx context.Context, globals *Globals, deployState *state.State, name, url, ref, composeDir, repoPath string) ([]string, []string, map[string]state.UnitState, error) {
	if globals.Verbose {
		fmt.Printf("Syncing repository: %s\n", name)
	}

	// Sync the repository to the latest state
	gitRepo := git.New(name, url, ref, composeDir, repoPath)
	if err := gitRepo.Sync(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to sync git repository: %w", err)
	}

	// Get the current commit hash
	commitHash, err := gitRepo.GetCurrentCommitHash()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get current commit hash: %w", err)
	}
	if globals.Verbose {
		fmt.Printf("  Current revision: %s\n", commitHash[:7])
	}

	unitNames, images, unitStates, genErr := s.generateUnits(ctx, globals, composeDir, repoPath)

	// Always record state to reflect what is actually on disk so that
	// stale unit detection remains accurate even after partial failures.
	deployState.SetCommit(name, commitHash)
	deployState.SetManagedUnits(name, unitNames)

	if genErr != nil {
		return nil, nil, nil, genErr
	}

	// Derive service names from container units
	var services []string
	for _, unitName := range unitNames {
		if strings.HasSuffix(unitName, ".container") {
			serviceName := strings.TrimSuffix(unitName, ".container") + ".service"
			services = append(services, serviceName)
		}
	}

	return services, images, unitStates, nil
}

// finalize performs post-sync/rollback cleanup: stale unit removal, state
// persistence, systemd daemon reload, restart of changed services, and
// service activation. It always runs — even on partial failure — so that
// successfully-processed repos stay consistent.
func (s *SyncCmd) finalize(ctx context.Context, globals *Globals, deployState *state.State, stateFilePath string, oldManagedUnits map[string]struct{}, newUnitStates map[string]state.UnitState, servicesToStart, images []string, failed int, action string) error {
	newManagedUnits := collectAllManagedUnits(deployState)
	staleUnits := diffUnits(oldManagedUnits, newManagedUnits)

	if len(staleUnits) > 0 {
		s.cleanupStaleUnits(ctx, globals, deployState, staleUnits)
	}

	// Determine which services need restart before updating stored hashes
	changedUnits := deployState.ChangedUnits(newUnitStates)
	var servicesToRestart []string
	for _, unit := range changedUnits {
		if strings.HasSuffix(unit, ".container") {
			serviceName := strings.TrimSuffix(unit, ".container") + ".service"
			servicesToRestart = append(servicesToRestart, serviceName)
		}
	}

	// Update stored unit states
	for name, us := range newUnitStates {
		deployState.SetUnitState(name, us)
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

	// Pull images before starting services to avoid systemd timeout
	if err := pullImages(images, globals.Verbose); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	// Restart services whose unit definitions or bind-mounted files changed
	if len(servicesToRestart) > 0 {
		if globals.Verbose {
			fmt.Printf("Restarting %d changed service(s)...\n", len(servicesToRestart))
		}
		if err := client.Restart(ctx, servicesToRestart...); err != nil {
			return fmt.Errorf("some services failed to restart: %w", err)
		}
		if globals.Verbose {
			fmt.Printf("Restarted %d changed service(s)\n", len(servicesToRestart))
		}
	}

	// Exclude already-restarted services from the start list
	if len(servicesToRestart) > 0 {
		restartedSet := make(map[string]struct{}, len(servicesToRestart))
		for _, svc := range servicesToRestart {
			restartedSet[svc] = struct{}{}
		}
		var remaining []string
		for _, svc := range servicesToStart {
			if _, restarted := restartedSet[svc]; !restarted {
				remaining = append(remaining, svc)
			}
		}
		servicesToStart = remaining
	}

	// Start all services to ensure everything is running.
	// Quadlet-generated units are produced by systemd's generator and cannot
	// be enabled (they are transient); DaemonReload + Start is sufficient.
	// Start is idempotent for already-running services.
	if len(servicesToStart) > 0 {
		if globals.Verbose {
			fmt.Printf("Starting %d service(s)...\n", len(servicesToStart))
		}
		if err := client.Start(ctx, servicesToStart...); err != nil {
			return fmt.Errorf("some services failed to start: %w", err)
		}
		if globals.Verbose {
			fmt.Printf("Started %d service(s)\n", len(servicesToStart))
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d repository(ies) failed to %s", failed, action)
	}

	return nil
}

// generateUnits loads compose files, writes the resulting quadlet units,
// and returns the list of unit filenames written, images referenced, and
// unit states for change detection.
func (s *SyncCmd) generateUnits(ctx context.Context, globals *Globals, composeDir, repoPath string) ([]string, []string, map[string]state.UnitState, error) {
	// Determine the directory containing compose files
	composeSourceDir := repoPath
	if composeDir != "" {
		composeSourceDir = filepath.Join(repoPath, composeDir)
	}

	// Load all compose projects from the repository
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
			fmt.Printf("  WARNING: failed to convert %s: %v\n", lp.FilePath, err)
			continue
		}

		// Write each unit to a file in quadletdir
		if err := s.writeUnits(units, quadletDir); err != nil {
			return unitNames, nil, nil, fmt.Errorf("failed to write units for %s: %w", lp.FilePath, err)
		}

		// Compute unit states for change detection and collect metadata
		for _, u := range units {
			unitNames = append(unitNames, u.Name)

			if strings.HasSuffix(u.Name, ".container") {
				us := computeUnitState(u, lp.Project, repoPath)
				unitStates[u.Name] = us
			}
		}

		// Collect images for pre-pulling
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

// cleanupStaleUnits stops, disables, and removes quadlet unit files
// that are no longer defined by any compose project, and cleans up
// their stored unit states.
func (s *SyncCmd) cleanupStaleUnits(ctx context.Context, globals *Globals, deployState *state.State, staleUnits []string) {
	quadletDir := globals.AppCfg.GetQuadletDir()

	// Identify container services to stop and disable
	var servicesToStop []string
	for _, unit := range staleUnits {
		if strings.HasSuffix(unit, ".container") {
			serviceName := strings.TrimSuffix(unit, ".container") + ".service"
			servicesToStop = append(servicesToStop, serviceName)
		}
	}

	if len(servicesToStop) > 0 {
		client, err := systemd.New(ctx, systemd.ScopeAuto)
		if err != nil {
			fmt.Printf("  WARNING: failed to connect to systemd for cleanup: %v\n", err)
		} else {
			if globals.Verbose {
				fmt.Printf("Stopping %d stale service(s)...\n", len(servicesToStop))
			}
			if err := client.Stop(ctx, servicesToStop...); err != nil {
				fmt.Printf("  WARNING: failed to stop some stale services: %v\n", err)
			}
			if err := client.Disable(ctx, servicesToStop...); err != nil {
				fmt.Printf("  WARNING: failed to disable some stale services: %v\n", err)
			}
			_ = client.Close()
		}
	}

	// Remove stale unit files from quadlet directory and clean up unit states
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

// pullImages pulls the given container images using podman.
func pullImages(images []string, verbose bool) error {
	if len(images) == 0 {
		return nil
	}

	total := len(images)
	if verbose {
		fmt.Printf("Pulling %d image(s)...\n", total)
	}

	for i, image := range images {
		if verbose {
			fmt.Printf("  [%d/%d] Pulling %s\n", i+1, total, image)
		}

		cmd := exec.Command("podman", "pull", image) //nolint:gosec // image names from validated compose files
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to pull image %s: %w", image, err)
			}
		} else {
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to pull image %s: %w\n%s", image, err, string(output))
			}
		}
	}

	return nil
}

// collectAllManagedUnits returns a set of all unit filenames across all repositories.
func collectAllManagedUnits(deployState *state.State) map[string]struct{} {
	result := make(map[string]struct{})
	for _, units := range deployState.ManagedUnits {
		for _, u := range units {
			result[u] = struct{}{}
		}
	}
	return result
}

// diffUnits returns unit names present in oldUnits but not in newUnits.
func diffUnits(oldUnits, newUnits map[string]struct{}) []string {
	var stale []string
	for u := range oldUnits {
		if _, ok := newUnits[u]; !ok {
			stale = append(stale, u)
		}
	}
	return stale
}

// computeUnitState computes content and bind mount hashes for change detection.
// It hashes the rendered unit file content and any bind-mounted regular files
// whose source paths are within the project directory.
func computeUnitState(unit systemd.Unit, project *types.Project, repoPath string) state.UnitState {
	var buf bytes.Buffer
	_, _ = unit.File.WriteTo(&buf)
	contentHash := fmt.Sprintf("%x", sha256.Sum256(buf.Bytes()))

	bindMountHashes := collectBindMountHashes(project, repoPath)

	return state.UnitState{
		ContentHash:     contentHash,
		BindMountHashes: bindMountHashes,
	}
}

// collectBindMountHashes computes SHA256 hashes for bind-mounted regular files
// within the project directory. Files outside the project dir, directories,
// and unreadable files are skipped.
func collectBindMountHashes(project *types.Project, repoPath string) map[string]string {
	hashes := make(map[string]string)
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return hashes
	}

	for _, svc := range project.Services {
		for _, vol := range svc.Volumes {
			if vol.Type != types.VolumeTypeBind || vol.Source == "" {
				continue
			}

			source := vol.Source
			if !filepath.IsAbs(source) {
				source = filepath.Join(project.WorkingDir, source)
			}

			absSource, err := filepath.Abs(source)
			if err != nil {
				continue
			}

			// Only hash files within the project directory
			if !strings.HasPrefix(absSource, absRepoPath+string(filepath.Separator)) {
				continue
			}

			info, err := os.Stat(absSource)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}

			data, err := os.ReadFile(absSource)
			if err != nil {
				continue
			}

			hashes[absSource] = fmt.Sprintf("%x", sha256.Sum256(data))
		}
	}

	return hashes
}
