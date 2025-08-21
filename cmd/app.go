// Package cmd provides the command line interface for quad-ops
package cmd

import (
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
)

// App holds the application dependencies for command line interface.
type App struct {
	Logger         log.Logger
	Config         *config.Settings
	ConfigProvider config.Provider
	FSService      *fs.Service
	UnitRepo       repository.Repository
	UnitManager    systemd.UnitManager
}

// NewApp creates a new App with all dependencies initialized.
func NewApp(logger log.Logger, configProv config.Provider) *App {
	cfg := configProv.GetConfig()
	fsService := fs.NewServiceWithLogger(configProv, logger)
	unitRepo := repository.NewRepository(logger, fsService)

	// Create systemd factory and get unit manager
	systemdFactory := systemd.NewDefaultFactory(configProv, logger)
	unitManager := systemdFactory.GetUnitManager()

	return &App{
		Logger:         logger,
		Config:         cfg,
		ConfigProvider: configProv,
		FSService:      fsService,
		UnitRepo:       unitRepo,
		UnitManager:    unitManager,
	}
}
