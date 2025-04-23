// Package unit provides quadlet unit generation and management functionality
package unit

import (
	"github.com/trly/quad-ops/internal/unit/dependency"
	"github.com/trly/quad-ops/internal/unit/model"
	"github.com/trly/quad-ops/internal/unit/processor"
	"github.com/trly/quad-ops/internal/unit/quadlet"
	"github.com/trly/quad-ops/internal/unit/repository"
)

// Re-export types
type (
	Container         = model.Container
	Volume            = model.Volume
	Network           = model.Network
	Secret            = model.Secret
	SystemdConfig     = model.SystemdConfig
	QuadletUnit       = model.QuadletUnitConfig
	Unit              = model.Unit
	SystemdUnit       = model.SystemdUnit
	BaseSystemdUnit   = model.BaseSystemdUnit
	ServiceDependency = dependency.ServiceDependency
	Repository        = repository.Repository
)

// Re-export functions
var (
	// Container operations
	NewContainer = model.NewContainer
	
	// Volume operations
	NewVolume = model.NewVolume
	
	// Network operations
	NewNetwork = model.NewNetwork
	
	// Dependency operations
	BuildServiceDependencyTree   = dependency.BuildServiceDependencyTree
	ApplyDependencyRelationships = dependency.ApplyDependencyRelationships
	
	// Processor operations
	ProcessComposeProjects = processor.ProcessComposeProjects
	CleanupOrphanedUnits   = processor.CleanupOrphanedUnits
	
	// Repository operations
	NewUnitRepository = repository.NewUnitRepository
	
	// Quadlet operations
	GenerateQuadletUnit = quadlet.GenerateQuadletUnit
)