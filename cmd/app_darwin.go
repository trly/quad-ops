//go:build darwin

package cmd

import (
	"fmt"

	"github.com/trly/quad-ops/internal/platform/launchd"
)

// initPlatformComponents initializes macOS-specific platform components.
func (a *App) initPlatformComponents() error {
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
		return fmt.Errorf("failed to create launchd renderer: %w", err)
	}
	a.renderer = renderer

	// Initialize lifecycle
	lifecycle, err := launchd.NewLifecycle(launchdOpts, a.Runner, a.Logger)
	if err != nil {
		return fmt.Errorf("failed to create launchd lifecycle: %w", err)
	}
	a.lifecycle = lifecycle

	return nil
}
