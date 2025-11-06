// Package cmd provides the command line interface for quad-ops
package cmd

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform/launchd"
	platsystemd "github.com/trly/quad-ops/internal/platform/systemd"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
	"github.com/trly/quad-ops/internal/validate"
)

// UnsupportedPlatformError is returned when a platform-specific feature is not available.
type UnsupportedPlatformError struct {
	Platform string
	Feature  string
}

func (e *UnsupportedPlatformError) Error() string {
	if e.Feature != "" {
		return fmt.Sprintf("%s is not supported on %s - quad-ops requires Linux (systemd) or macOS (launchd) for service management. Non-platform commands (version, validate) work on all platforms.", e.Feature, e.Platform)
	}
	return fmt.Sprintf("platform %s is not supported - quad-ops requires Linux (systemd) or macOS (launchd) for service management. Non-platform commands (version, validate) work on all platforms.", e.Platform)
}

// App holds the application dependencies for command line interface.
type App struct {
	Logger         log.Logger
	Config         *config.Settings
	ConfigProvider config.Provider
	Runner         execx.Runner
	FSService      *fs.Service

	// Phase 6: New architecture components (non-platform)
	ArtifactStore     repository.ArtifactStore  // Stores deployed platform artifacts
	RepoArtifactStore repository.ArtifactStore  // Stores managed artifacts in repository
	GitSyncer         repository.GitSyncer      // Syncs git repositories
	ComposeProcessor  ComposeProcessorInterface // Processes compose to service specs

	// Platform-specific components (lazy initialization)
	platformOnce sync.Once
	renderer     RendererInterface
	lifecycle    LifecycleInterface
	platformErr  error
	os           string // For testing, defaults to runtime.GOOS

	Validator    SystemValidator
	OutputFormat string
}

// NewApp creates a new App with all dependencies initialized.
// Platform-specific components (Renderer, Lifecycle) are initialized lazily on first use.
func NewApp(logger log.Logger, configProv config.Provider) (*App, error) {
	cfg := configProv.GetConfig()
	runner := execx.NewRealRunner()
	fsService := fs.NewServiceWithLogger(configProv, logger)

	// New architecture components (platform-independent)
	// ArtifactStore for deployed artifacts
	deployedBaseDir := cfg.QuadletDir
	artifactStore := repository.NewArtifactStore(fsService, logger, deployedBaseDir)
	// RepoArtifactStore for managed artifacts in repository
	repoBaseDir := cfg.RepositoryDir
	repoArtifactStore := repository.NewArtifactStore(fsService, logger, repoBaseDir)
	gitSyncer := repository.NewGitSyncer(configProv, logger)
	composeProcessor := newComposeProcessor(cfg)

	// Create validator with injected dependencies
	validator := validate.NewValidator(logger, runner)

	return &App{
		Logger:         logger,
		Config:         cfg,
		ConfigProvider: configProv,
		Runner:         runner,
		FSService:      fsService,

		// New architecture components (platform-independent)
		ArtifactStore:     artifactStore,
		RepoArtifactStore: repoArtifactStore,
		GitSyncer:         gitSyncer,
		ComposeProcessor:  composeProcessor,

		// Platform components initialized lazily
		os: runtime.GOOS,

		Validator: validator,
	}, nil
}

// initPlatform initializes platform-specific components (renderer, lifecycle).
// Uses sync.Once to ensure initialization happens only once.
// For testing, checks if renderer/lifecycle are already injected before initializing.
func (a *App) initPlatform() {
	a.platformOnce.Do(func() {
		// If renderer and lifecycle are already set (test injection), skip initialization
		if a.renderer != nil && a.lifecycle != nil {
			return
		}

		targetOS := a.os
		if targetOS == "" {
			targetOS = runtime.GOOS
		}

		switch targetOS {
		case "linux":
			// Systemd platform
			a.Logger.Debug("Initializing platform: systemd (Linux)")

			// Create systemd factory for platform components
			systemdFactory := systemd.NewDefaultFactory(a.ConfigProvider, a.Logger)

			// Initialize renderer
			a.renderer = platsystemd.NewRenderer(a.Logger)

			// Initialize lifecycle
			unitManager := systemdFactory.GetUnitManager()
			connectionFactory := systemdFactory.GetConnectionFactory()
			a.lifecycle = platsystemd.NewLifecycle(unitManager, connectionFactory, a.Config.UserMode, a.Logger)

		case "darwin":
			// Launchd platform (macOS)
			a.Logger.Debug("Initializing platform: launchd (macOS)")

			// Create launchd options from config settings
			launchdOpts := launchd.OptionsFromSettings(
				a.Config.RepositoryDir,
				a.Config.QuadletDir,
				a.Config.UserMode,
			)

			// Initialize renderer
			renderer, err := launchd.NewRenderer(launchdOpts, a.Logger)
			if err != nil {
				a.platformErr = fmt.Errorf("failed to create launchd renderer: %w", err)
				return
			}
			a.renderer = renderer

			// Initialize lifecycle
			lifecycle, err := launchd.NewLifecycle(launchdOpts, a.Runner, a.Logger)
			if err != nil {
				a.platformErr = fmt.Errorf("failed to create launchd lifecycle: %w", err)
				return
			}
			a.lifecycle = lifecycle

		default:
			a.platformErr = &UnsupportedPlatformError{
				Platform: targetOS,
				Feature:  "service management",
			}
		}
	})
}

// GetRenderer returns the platform renderer, initializing it if necessary.
func (a *App) GetRenderer(_ context.Context) (RendererInterface, error) {
	// If renderer is already set (test injection), return it directly
	if a.renderer != nil {
		return a.renderer, nil
	}

	a.initPlatform()
	if a.platformErr != nil {
		return nil, a.platformErr
	}
	return a.renderer, nil
}

// GetLifecycle returns the platform lifecycle manager, initializing it if necessary.
func (a *App) GetLifecycle(_ context.Context) (LifecycleInterface, error) {
	// If lifecycle is already set (test injection), return it directly
	if a.lifecycle != nil {
		return a.lifecycle, nil
	}

	a.initPlatform()
	if a.platformErr != nil {
		return nil, a.platformErr
	}

	// Validate that we got the expected platform implementation
	targetOS := a.os
	if targetOS == "" {
		targetOS = runtime.GOOS
	}
	if targetOS == "linux" {
		if _, ok := a.lifecycle.(*platsystemd.Lifecycle); !ok {
			a.Logger.Warn("Expected systemd lifecycle on Linux", "got", fmt.Sprintf("%T", a.lifecycle))
		}
	}

	return a.lifecycle, nil
}

// IsPlatformAvailable returns true if platform-specific features are available.
func (a *App) IsPlatformAvailable() bool {
	a.initPlatform()
	return a.platformErr == nil
}

// newComposeProcessor creates a new compose processor with the repository directory.
func newComposeProcessor(cfg *config.Settings) ComposeProcessorInterface {
	return compose.NewSpecProcessor(cfg.RepositoryDir)
}
