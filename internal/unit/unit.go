// Package unit provides quadlet unit generation and management functionality
package unit

import (
	"github.com/trly/quad-ops/internal/unit/dependency"
	"github.com/trly/quad-ops/internal/unit/model"
	"github.com/trly/quad-ops/internal/unit/processor"
	"github.com/trly/quad-ops/internal/unit/quadlet"
	"github.com/trly/quad-ops/internal/unit/repository"
)

// Container represents the configuration for a container unit.
type Container = model.Container

// Volume represents the configuration for a volume unit.
type Volume = model.Volume

// Network represents the configuration for a network unit.
type Network = model.Network

// Secret represents a secret configuration for a container.
type Secret = model.Secret

// SystemdConfig represents the configuration for a systemd unit.
type SystemdConfig = model.SystemdConfig

// QuadletUnit represents the configuration for a Quadlet unit with various types.
type QuadletUnit = model.QuadletUnitConfig

// Unit represents a database record for a unit.
type Unit = model.Unit

// SystemdUnit defines the interface for systemd unit operations.
type SystemdUnit = model.SystemdUnit

// BaseSystemdUnit provides common systemd unit operations.
type BaseSystemdUnit = model.BaseSystemdUnit

// ServiceDependency represents the dependencies of a service in both directions.
type ServiceDependency = dependency.ServiceDependency

// Repository defines the interface for unit data access operations.
type Repository = repository.Repository

// NewContainer creates a new Container with the given name.
var NewContainer = model.NewContainer

// NewVolume creates a new Volume with the given name.
var NewVolume = model.NewVolume

// NewNetwork creates a new Network with the given name.
var NewNetwork = model.NewNetwork

// BuildServiceDependencyTree builds a bidirectional dependency tree for all services in a project.
var BuildServiceDependencyTree = dependency.BuildServiceDependencyTree

// ApplyDependencyRelationships applies dependencies to a quadlet unit based on the dependency tree.
var ApplyDependencyRelationships = dependency.ApplyDependencyRelationships

// ProcessComposeProjects processes Docker Compose projects and converts them to Podman systemd units.
var ProcessComposeProjects = processor.ProcessComposeProjects

// CleanupOrphanedUnits cleans up units that are no longer in use or belong to removed repositories.
var CleanupOrphanedUnits = processor.CleanupOrphanedUnits

// NewUnitRepository creates a new SQL-based unit repository.
var NewUnitRepository = repository.NewUnitRepository

// GenerateQuadletUnit generates a quadlet unit file content from a unit configuration.
var GenerateQuadletUnit = quadlet.GenerateQuadletUnit
