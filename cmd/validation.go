package cmd

import (
	"context"
	"fmt"
	"sort"

	"github.com/trly/quad-ops/internal/compose"
	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/service"
)

// systemdUnitNameForService constructs a systemd unit name from project and service.
func systemdUnitNameForService(project, svc string) string {
	return compose.Prefix(project, svc) + ".service"
}

// launchdLabelForService constructs a launchd label from project and service.
func launchdLabelForService(project, svc string) string {
	return fmt.Sprintf("com.quad-ops.%s.%s", project, svc)
}

// validateExternalDependencies checks that all external service units exist.
// Batch-aware: checks in-memory specs first before calling Lifecycle.Exists().
// Platform-aware: uses correct unit name/label format for systemd/launchd.
func validateExternalDependencies(
	ctx context.Context,
	specs []service.Spec,
	lifecycle LifecycleInterface,
	logger log.Logger,
	platform string,
) error {
	// Build map of services being deployed in this batch (by full name)
	batchServices := make(map[string]bool)
	for _, spec := range specs {
		batchServices[spec.Name] = true
	}

	var missingRequired []string
	var missingOptional []string

	for i := range specs {
		spec := &specs[i] // Get pointer to modify ExistsInRuntime flag

		for j := range spec.ExternalDependencies {
			extDep := &spec.ExternalDependencies[j] // Get pointer to modify flag

			// Construct expected external service name
			externalServiceName := compose.Prefix(extDep.Project, extDep.Service)

			// Check if external dep is satisfied by current batch
			if batchServices[externalServiceName] {
				extDep.ExistsInRuntime = true
				continue
			}

			// Not in batch, check runtime via platform-specific name
			var checkName string
			switch platform {
			case "systemd":
				checkName = systemdUnitNameForService(extDep.Project, extDep.Service)
			case "launchd":
				checkName = launchdLabelForService(extDep.Project, extDep.Service)
			default:
				return fmt.Errorf("unsupported platform: %s", platform)
			}

			exists, err := lifecycle.Exists(ctx, checkName)
			if err != nil {
				return fmt.Errorf("failed to check external service %s: %w", externalServiceName, err)
			}

			extDep.ExistsInRuntime = exists

			if !exists {
				if extDep.Optional {
					missingOptional = append(missingOptional, externalServiceName)
				} else {
					missingRequired = append(missingRequired, externalServiceName)
				}
			}
		}
	}

	if len(missingOptional) > 0 {
		sort.Strings(missingOptional)
		logger.Warn("Optional external dependencies not found", "services", missingOptional)
	}

	if len(missingRequired) > 0 {
		sort.Strings(missingRequired)
		return fmt.Errorf("required external services not found: %v (ensure dependency projects are deployed first)", missingRequired)
	}

	return nil
}

// validateExternalResources checks that external networks and volumes exist.
// Batch-aware: checks in-memory specs first before calling podman.
func validateExternalResources(
	ctx context.Context,
	specs []service.Spec,
	runner execx.Runner,
) error {
	var missingNetworks []string
	var missingVolumes []string

	// Build map of networks being created in this batch (non-external)
	batchNetworks := make(map[string]bool)
	for _, spec := range specs {
		for _, net := range spec.Networks {
			if !net.External {
				batchNetworks[net.Name] = true
			}
		}
	}

	// Build map of volumes being created in this batch (non-external)
	batchVolumes := make(map[string]bool)
	for _, spec := range specs {
		for _, vol := range spec.Volumes {
			if !vol.External {
				batchVolumes[vol.Name] = true
			}
		}
	}

	// Collect unique external networks
	externalNets := make(map[string]bool)
	for _, spec := range specs {
		for _, net := range spec.Networks {
			if net.External {
				externalNets[net.Name] = true
			}
		}
	}

	// Check networks: first check batch, then podman
	for netName := range externalNets {
		// Skip if being created in this batch
		if batchNetworks[netName] {
			continue
		}

		// Not in batch, check podman runtime
		_, err := runner.CombinedOutput(ctx, "podman", "network", "inspect", netName)
		if err != nil {
			missingNetworks = append(missingNetworks, netName)
		}
	}

	// Collect unique external volumes
	externalVols := make(map[string]bool)
	for _, spec := range specs {
		for _, vol := range spec.Volumes {
			if vol.External {
				externalVols[vol.Name] = true
			}
		}
	}

	// Check volumes: first check batch, then podman
	for volName := range externalVols {
		// Skip if being created in this batch
		if batchVolumes[volName] {
			continue
		}

		// Not in batch, check podman runtime
		_, err := runner.CombinedOutput(ctx, "podman", "volume", "inspect", volName)
		if err != nil {
			missingVolumes = append(missingVolumes, volName)
		}
	}

	if len(missingNetworks) > 0 {
		sort.Strings(missingNetworks)
		return fmt.Errorf("external networks not found: %v (create with 'podman network create' or check compose config)", missingNetworks)
	}

	if len(missingVolumes) > 0 {
		sort.Strings(missingVolumes)
		return fmt.Errorf("external volumes not found: %v (create with 'podman volume create' or check compose config)", missingVolumes)
	}

	return nil
}
