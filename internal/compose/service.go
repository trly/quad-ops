package compose

import (
	"fmt"
	"os"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/dependency"
	"github.com/trly/quad-ops/internal/sorting"
	"github.com/trly/quad-ops/internal/unit"
)

// processServices processes all container services from a Docker Compose project.
func (p *Processor) processServices(project *types.Project, dependencyGraph *dependency.ServiceDependencyGraph) error {
	for serviceName, service := range project.Services {
		p.logger.Debug("Processing service", "service", serviceName)

		prefixedName := Prefix(project.Name, serviceName)

		// Process build if present
		if err := p.processBuildIfPresent(&service, serviceName, project, dependencyGraph); err != nil {
			return err
		}

		// Create and configure container
		container := p.createContainerFromService(service, prefixedName, serviceName, project)

		// Process init containers first
		initContainers, err := unit.ParseInitContainers(service)
		if err != nil {
			p.logger.Error("Failed to parse init containers", "service", serviceName, "error", err)
			return err
		}

		var initUnitNames []string
		for i, initContainer := range initContainers {
			initName := fmt.Sprintf("%s-%s-init-%d", project.Name, serviceName, i)
			initContainerUnit := unit.CreateInitContainerUnit(initContainer, initName, container)

			// Create init quadlet unit
			initQuadletUnit := createInitQuadletUnit(initName, initContainerUnit)

			// Apply dependency relationships to init container (same as main service)
			if err := unit.ApplyDependencyRelationships(&initQuadletUnit, serviceName, dependencyGraph, project.Name); err != nil {
				p.logger.Error("Failed to apply dependency relationships to init container", "init", initName, "error", err)
			}

			// Process the init unit
			if err := p.processUnit(&initQuadletUnit); err != nil {
				return err
			}

			initUnitNames = append(initUnitNames, initName+".service")
		}

		// Create main container quadlet unit
		quadletUnit := createQuadletUnit(prefixedName, container)

		// Add init container dependencies to main container
		for _, initUnitName := range initUnitNames {
			quadletUnit.Systemd.After = append(quadletUnit.Systemd.After, initUnitName)
			quadletUnit.Systemd.Requires = append(quadletUnit.Systemd.Requires, initUnitName)
		}

		// Apply dependencies and process
		if err := p.finishProcessingService(&quadletUnit, serviceName, dependencyGraph, project.Name); err != nil {
			return err
		}
	}
	return nil
}

// processBuildIfPresent handles build configuration for a service.
func (p *Processor) processBuildIfPresent(service *types.ServiceConfig, serviceName string, project *types.Project, dependencyGraph *dependency.ServiceDependencyGraph) error {
	if service.Build == nil {
		return nil
	}

	p.logger.Debug("Processing build for service", "service", serviceName)

	buildUnitName := fmt.Sprintf("%s-%s-build", project.Name, serviceName)
	build := unit.NewBuild(buildUnitName)
	build = build.FromComposeBuild(*service.Build, *service, project.Name)

	// Configure build context
	if build.SetWorkingDirectory == "repo" {
		build.SetWorkingDirectory = project.WorkingDir
		p.logger.Debug("Setting build context to project working directory",
			"service", serviceName, "context", build.SetWorkingDirectory)
	}

	// Handle production target
	if err := p.handleProductionTarget(build, serviceName, project.WorkingDir); err != nil {
		p.logger.Debug("Error checking Dockerfile for production target", "error", err)
	}

	buildQuadletUnit := unit.QuadletUnit{
		Name:  buildUnitName,
		Type:  "build",
		Build: *build,
		Systemd: unit.SystemdConfig{
			RemainAfterExit: true,
		},
	}

	// Process the build unit
	if err := p.processUnit(&buildQuadletUnit); err != nil {
		p.logger.Error("Failed to process build unit", "error", err)
	}

	// Update service image and dependencies
	service.Image = fmt.Sprintf("%s.build", buildUnitName)
	return p.addBuildDependency(dependencyGraph, serviceName)
}

// handleProductionTarget checks and handles production build target.
func (p *Processor) handleProductionTarget(build *unit.Build, serviceName, workingDir string) error {
	if build.Target != "production" {
		return nil
	}

	// Use the more robust path validation that handles filepath.Clean internally
	validDockerfilePath, err := sorting.ValidatePathWithinBase("Dockerfile", workingDir)
	if err != nil {
		return fmt.Errorf("invalid dockerfile path for service %s: %w", serviceName, err)
	}

	dockerfilePath := validDockerfilePath

	if _, err := os.Stat(dockerfilePath); err != nil {
		return err
	}

	content, err := os.ReadFile(dockerfilePath) //nolint:gosec
	if err != nil {
		return err
	}

	if !strings.Contains(string(content), "FROM ") || !strings.Contains(string(content), " as production") {
		build.Target = ""
		p.logger.Debug("Removing target='production' as it doesn't exist in Dockerfile", "service", serviceName)
	}
	return nil
}

// addBuildDependency adds a build dependency to the dependency graph.
func (p *Processor) addBuildDependency(dependencyGraph *dependency.ServiceDependencyGraph, serviceName string) error {
	buildName := fmt.Sprintf("%s-build", serviceName)
	if err := dependencyGraph.AddService(buildName); err != nil {
		p.logger.Debug("Build service already exists in dependency graph", "service", buildName)
	}
	if err := dependencyGraph.AddDependency(serviceName, buildName); err != nil {
		p.logger.Error("Failed to add build dependency", "service", serviceName, "dependency", buildName, "error", err)
		return err
	}
	return nil
}

// createContainerFromService creates a container unit from a compose service.
func (p *Processor) createContainerFromService(service types.ServiceConfig, prefixedName, serviceName string, project *types.Project) *unit.Container {
	container := unit.NewContainer(prefixedName)
	container = container.FromComposeService(service, project)

	// Add environment files
	p.addEnvironmentFiles(container, serviceName, project.WorkingDir)

	// Configure container naming
	p.configureContainerNaming(container, prefixedName, serviceName)

	return container
}

// addEnvironmentFiles adds environment files to the container.
func (p *Processor) addEnvironmentFiles(container *unit.Container, serviceName, workingDir string) {
	envFiles := FindEnvFiles(serviceName, workingDir)
	container.EnvironmentFile = append(container.EnvironmentFile, envFiles...)
}

// configureContainerNaming configures container naming and aliases.
func (p *Processor) configureContainerNaming(container *unit.Container, prefixedName string, serviceName string) {
	// Use quad-ops preferred naming (no systemd- prefix)
	container.ContainerName = prefixedName

	// Add service name as NetworkAlias for portability
	container.NetworkAlias = append(container.NetworkAlias, serviceName)

	// Add custom hostname as NetworkAlias if different from service name
	if container.HostName != "" && container.HostName != serviceName {
		container.NetworkAlias = append(container.NetworkAlias, container.HostName)
	}
}

// createQuadletUnit creates a quadlet unit from a container.
func createQuadletUnit(prefixedName string, container *unit.Container) unit.QuadletUnit {
	systemdConfig := unit.SystemdConfig{}

	if container.RestartPolicy != "" {
		systemdConfig.RestartPolicy = container.RestartPolicy
	}

	return unit.QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *container,
		Systemd:   systemdConfig,
	}
}

// createInitQuadletUnit creates a quadlet unit for an init container.
func createInitQuadletUnit(prefixedName string, container *unit.Container) unit.QuadletUnit {
	systemdConfig := unit.SystemdConfig{
		Type:            "oneshot",
		RemainAfterExit: true,
	}

	return unit.QuadletUnit{
		Name:      prefixedName,
		Type:      "container",
		Container: *container,
		Systemd:   systemdConfig,
	}
}

// finishProcessingService applies dependencies and processes the service unit.
func (p *Processor) finishProcessingService(quadletUnit *unit.QuadletUnit, serviceName string, dependencyGraph *dependency.ServiceDependencyGraph, projectName string) error {
	// Apply dependency relationships
	if err := unit.ApplyDependencyRelationships(quadletUnit, serviceName, dependencyGraph, projectName); err != nil {
		p.logger.Error("Failed to apply dependency relationships", "service", serviceName, "error", err)
	}

	// Process the quadlet unit
	if err := p.processUnit(quadletUnit); err != nil {
		p.logger.Error("Failed to process unit", "error", err)
		return err
	}
	return nil
}
