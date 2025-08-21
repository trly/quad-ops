package compose

import (
	"fmt"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/systemd"
)

// processProjects is the main orchestration method that processes all projects.
func (p *Processor) processProjects(projects []*types.Project, cleanup bool) error {
	// Build dependency graphs for all projects first
	for _, project := range projects {
		dependencyGraph, err := dependency.BuildServiceDependencyGraph(project)
		if err != nil {
			return fmt.Errorf("failed to build dependency graph for project %s: %w", project.Name, err)
		}
		p.dependencyGraphs[project.Name] = dependencyGraph

		p.logger.Debug("Processing compose project", "project", project.Name, "services", len(project.Services), "networks", len(project.Networks), "volumes", len(project.Volumes))
	}

	// Process each project
	for _, project := range projects {
		if err := p.processProject(project); err != nil {
			p.logger.Error("Failed to process project", "project", project.Name, "error", err)
			return err
		}
	}

	// Clean up any orphaned units BEFORE restarting changed units to avoid dependency conflicts
	if cleanup {
		if err := p.cleanupOrphans(); err != nil {
			p.logger.Error("Failed to clean up orphaned units", "error", err)
		}
		// Wait for systemd to fully process unit removals before proceeding with restarts
		time.Sleep(1 * time.Second)
	}

	// Restart changed units if any
	if len(p.changedUnits) > 0 {
		if err := p.restartChangedUnits(); err != nil {
			p.logger.Error("Failed to restart changed units", "error", err)
		}
	}

	return nil
}

// processProject processes a single Docker Compose project.
func (p *Processor) processProject(project *types.Project) error {
	dependencyGraph := p.dependencyGraphs[project.Name]

	// Process services (containers)
	if err := p.processServices(project, dependencyGraph); err != nil {
		p.logger.Error("Failed to process services", "project", project.Name, "error", err)
		return err
	}

	// Process volumes
	if err := p.processVolumes(project); err != nil {
		p.logger.Error("Failed to process volumes", "project", project.Name, "error", err)
		return err
	}

	// Process networks
	if err := p.processNetworks(project); err != nil {
		p.logger.Error("Failed to process networks", "project", project.Name, "error", err)
		return err
	}

	return nil
}

// restartChangedUnits handles restarting units that have changed.
func (p *Processor) restartChangedUnits() error {
	// Convert QuadletUnit slice to systemd.UnitChange slice
	systemdUnits := make([]systemd.UnitChange, len(p.changedUnits))
	for i, unit := range p.changedUnits {
		systemdUnits[i] = systemd.UnitChange{
			Name: unit.Name,
			Type: unit.Type,
		}
	}

	return p.systemd.RestartChangedUnits(systemdUnits, p.dependencyGraphs)
}
