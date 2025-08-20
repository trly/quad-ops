// Package compose provides Docker Compose project processing functionality
package compose

import (
	"log/slog"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/unit"
)

// Processor handles the conversion of Docker Compose projects to Podman systemd units.
type Processor struct {
	repo             Repository
	systemd          SystemdManager
	fs               FileSystem
	logger           *slog.Logger
	force            bool
	processedUnits   map[string]bool
	changedUnits     []unit.QuadletUnit
	dependencyGraphs map[string]*dependency.ServiceDependencyGraph
}

// NewProcessor creates a new Processor with the given dependencies.
func NewProcessor(repo Repository, systemd SystemdManager, fs FileSystem, logger *slog.Logger, force bool) *Processor {
	return &Processor{
		repo:             repo,
		systemd:          systemd,
		fs:               fs,
		logger:           logger,
		force:            force,
		processedUnits:   make(map[string]bool),
		changedUnits:     make([]unit.QuadletUnit, 0),
		dependencyGraphs: make(map[string]*dependency.ServiceDependencyGraph),
	}
}

// NewDefaultProcessor creates a new Processor with default real dependencies.
func NewDefaultProcessor(force bool) *Processor {
	repo := NewRepositoryAdapter(repository.NewRepository())
	systemdMgr := NewSystemdAdapter()
	fsMgr := NewFileSystemAdapter()
	logger := log.GetLogger()

	return NewProcessor(repo, systemdMgr, fsMgr, logger, force)
}

// ProcessProjects processes Docker Compose projects and converts them to Podman systemd units.
func (p *Processor) ProcessProjects(projects []*types.Project, cleanup bool) error {
	return p.processProjects(projects, cleanup)
}

// WithExistingProcessedUnits sets existing processed units for tracking across multiple calls.
func (p *Processor) WithExistingProcessedUnits(existingUnits map[string]bool) *Processor {
	if existingUnits != nil {
		p.processedUnits = existingUnits
	}
	return p
}

// GetProcessedUnits returns the map of processed units.
func (p *Processor) GetProcessedUnits() map[string]bool {
	return p.processedUnits
}
