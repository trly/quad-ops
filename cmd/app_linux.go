//go:build linux

package cmd

import (
	platsystemd "github.com/trly/quad-ops/internal/platform/systemd"
	"github.com/trly/quad-ops/internal/systemd"
)

// initPlatformComponents initializes Linux-specific platform components.
func (a *App) initPlatformComponents() error {
	a.Logger.Debug("Initializing platform: systemd (Linux)")

	// Create systemd factory for platform components
	systemdFactory := systemd.NewDefaultFactory(a.ConfigProvider, a.Logger)

	// Initialize renderer
	a.renderer = platsystemd.NewRenderer(a.Logger)

	// Initialize lifecycle
	unitManager := systemdFactory.GetUnitManager()
	connectionFactory := systemdFactory.GetConnectionFactory()
	a.lifecycle = platsystemd.NewLifecycle(unitManager, connectionFactory, a.Config.UserMode, a.Logger)

	return nil
}
