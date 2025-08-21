// Package cmd provides the command line interface for quad-ops
package cmd

import (
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/systemd"
	"github.com/trly/quad-ops/internal/validate"
)

// App holds the application dependencies for command line interface.
type App struct {
	Logger         log.Logger
	Config         *config.Settings
	ConfigProvider config.Provider
	Runner         execx.Runner
	FSService      *fs.Service
	UnitRepo       repository.Repository
	UnitManager    systemd.UnitManager
	Validator      *validate.Validator
}

// NewApp creates a new App with all dependencies initialized.
func NewApp(logger log.Logger, configProv config.Provider) *App {
	cfg := configProv.GetConfig()
	runner := execx.NewRealRunner()
	fsService := fs.NewServiceWithLogger(configProv, logger)
	unitRepo := repository.NewRepository(logger, fsService)

	// Create systemd factory and get unit manager
	systemdFactory := systemd.NewDefaultFactory(configProv, logger)
	unitManager := systemdFactory.GetUnitManager()

	// Create validator with injected dependencies
	validator := validate.NewValidator(logger, runner)

	return &App{
		Logger:         logger,
		Config:         cfg,
		ConfigProvider: configProv,
		Runner:         runner,
		FSService:      fsService,
		UnitRepo:       unitRepo,
		UnitManager:    unitManager,
		Validator:      validator,
	}
}
