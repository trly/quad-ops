package unit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/log"
)

// Build represents the configuration for a build unit.
type Build struct {
	BaseUnit                              // Embed the base struct
	ImageTag            []string          // Specifies the name which is assigned to the resulting image
	File                string            // Path to Containerfile/Dockerfile
	SetWorkingDirectory string            // Sets the working directory for the build context
	Label               []string          // Add metadata to the resulting image
	Annotation          []string          // Add OCI annotations to the resulting image
	Env                 map[string]string // Environment variables for the build process
	Secret              []string          // Pass secret information for build
	Network             []string          // Set network mode for RUN instructions
	Pull                string            // Pull policy for base images (always, missing, never, newer)
	Volume              []string          // Mount volumes for build process
	Target              string            // Set the target build stage to build
	PodmanArgs          []string          // Additional arguments to pass to podman build
}

// NewBuild creates a new Build with the given name.
func NewBuild(name string) *Build {
	return &Build{
		BaseUnit: BaseUnit{
			Name:     name,
			UnitType: "build",
		},
	}
}

// FromComposeBuild converts a Docker Compose build configuration to a Podman Quadlet build configuration.
func (b *Build) FromComposeBuild(buildConfig types.BuildConfig, service types.ServiceConfig, projectName string) *Build {
	// Basic fields
	b.setBasicBuildFields(buildConfig, service)

	// Process environment variables
	b.processEnvironment(buildConfig)

	// Process networks
	b.processNetworks(buildConfig, projectName)

	// Process volumes
	b.processVolumes(buildConfig, projectName)

	// Process advanced build configuration
	b.processAdvancedConfig(buildConfig)

	// Sort all fields for deterministic output
	sortBuild(b)

	return b
}

// setBasicBuildFields sets simple fields directly from the build config.
func (b *Build) setBasicBuildFields(buildConfig types.BuildConfig, service types.ServiceConfig) {
	// Generate the ImageTag from the service name and project name
	if service.Image != "" {
		b.ImageTag = append(b.ImageTag, service.Image)
	} else {
		// If no image is specified, create a default image tag based on service name
		defaultTag := fmt.Sprintf("localhost/%s:latest", service.Name)
		b.ImageTag = append(b.ImageTag, defaultTag)
	}

	// Handle Dockerfile/Containerfile path
	if buildConfig.Dockerfile != "" {
		b.File = buildConfig.Dockerfile
	} else {
		// Default to Dockerfile in the context directory
		b.File = "Dockerfile"
	}

	// Handle build context
	if buildConfig.Context != "" {
		// Check if it's a Git URL or regular path
		if strings.HasPrefix(buildConfig.Context, "http://") || strings.HasPrefix(buildConfig.Context, "https://") {
			// For Git URLs, set the working directory to the context URL
			b.SetWorkingDirectory = buildConfig.Context
		} else {
			// For regular paths, use the absolute path to the repository directory
			// We need to pass the project's WorkingDir, which we don't have access to here
			// We'll set this to "repo" as a marker and update it in compose_processor.go
			b.SetWorkingDirectory = "repo"
		}
	} else {
		// Default to current directory
		b.SetWorkingDirectory = "repo"
	}

	// Process labels
	if buildConfig.Labels != nil {
		b.Label = append(b.Label, buildConfig.Labels.AsList()...)
	}

	// Process target stage - only set if specified
	if buildConfig.Target != "" {
		b.Target = buildConfig.Target
	}

	// Process pull policy
	switch buildConfig.Pull {
	case true:
		b.Pull = "always"
	case false:
		// Even when set to false, allow pulling missing images to prevent build failures
		b.Pull = "missing"
	default:
		// Default to missing if not specified
		b.Pull = "missing"
	}
}

// processEnvironment handles environment variables for the build process.
func (b *Build) processEnvironment(buildConfig types.BuildConfig) {
	// Process build args as environment variables
	if len(buildConfig.Args) > 0 {
		if b.Env == nil {
			b.Env = make(map[string]string)
		}

		// Convert build args to environment variables
		for k, v := range buildConfig.Args {
			if v != nil {
				b.Env[k] = *v
			}
		}
	}
}

// processNetworks handles network settings for RUN instructions.
func (b *Build) processNetworks(buildConfig types.BuildConfig, projectName string) {
	// Process network mode for RUN instructions
	if buildConfig.Network != "" {
		// Check if it's a special network name
		if buildConfig.Network == "host" || buildConfig.Network == "none" {
			b.Network = append(b.Network, buildConfig.Network)
		} else {
			// This is a project-defined network - format for Podman Quadlet with .network suffix
			networkRef := fmt.Sprintf("%s-%s.network", projectName, buildConfig.Network)
			b.Network = append(b.Network, networkRef)
		}
	}
}

// processVolumes handles volume mounts for the build process.
func (b *Build) processVolumes(buildConfig types.BuildConfig, projectName string) {
	// If explicit volumes are defined for the build
	if ext, ok := buildConfig.Extensions["x-podman-volumes"]; ok {
		if volumes, ok := ext.([]interface{}); ok {
			for _, vol := range volumes {
				if volStr, ok := vol.(string); ok {
					// Check if it's a volume reference
					parts := strings.Split(volStr, ":")
					if len(parts) >= 2 && !strings.HasPrefix(parts[0], "/") {
						// Convert named volumes to Podman Quadlet format
						b.Volume = append(b.Volume, fmt.Sprintf("%s-%s.volume:%s", projectName, parts[0], parts[1]))
					} else {
						// Regular bind mount - use as-is
						b.Volume = append(b.Volume, volStr)
					}
				}
			}
		}
	}
}

// processAdvancedConfig processes advanced build configuration options.
func (b *Build) processAdvancedConfig(buildConfig types.BuildConfig) {
	// Process secrets
	if len(buildConfig.Secrets) > 0 {
		for _, secret := range buildConfig.Secrets {
			// Format the secret as expected by podman build --secret
			secretStr := secret.Source
			if secret.Target != "" && secret.Target != secret.Source {
				secretStr = fmt.Sprintf("%s,target=%s", secretStr, secret.Target)
			}
			b.Secret = append(b.Secret, secretStr)
		}
	}

	// Process SSH authentication if specified
	if len(buildConfig.SSH) > 0 {
		// SSH authentication not directly supported by Quadlet build units
		// Add through PodmanArgs
		for _, ssh := range buildConfig.SSH {
			b.PodmanArgs = append(b.PodmanArgs, fmt.Sprintf("--ssh=%s", ssh))
		}
		log.GetLogger().Warn("SSH authentication for builds is not directly supported by Podman Quadlet. Using PodmanArgs directive instead.")
	}

	// Process cache configuration if specified
	if len(buildConfig.CacheFrom) > 0 {
		// Cache configuration not directly supported by Quadlet build units
		// Add through PodmanArgs
		for _, cache := range buildConfig.CacheFrom {
			b.PodmanArgs = append(b.PodmanArgs, fmt.Sprintf("--cache-from=%s", cache))
		}
		log.GetLogger().Warn("Build cache configuration is not directly supported by Podman Quadlet. Using PodmanArgs directive instead.")
	}

	// Process additional build extensions if any
	if ext, ok := buildConfig.Extensions["x-podman-buildargs"]; ok {
		if args, ok := ext.([]interface{}); ok {
			for _, arg := range args {
				if argStr, ok := arg.(string); ok {
					b.PodmanArgs = append(b.PodmanArgs, argStr)
				}
			}
		}
	}
}

// sortBuild ensures all slices in a build config are sorted deterministically in-place.
func sortBuild(build *Build) {
	// Sort all slices for deterministic output
	if len(build.ImageTag) > 0 {
		sort.Strings(build.ImageTag)
	}

	if len(build.Label) > 0 {
		sort.Strings(build.Label)
	}

	if len(build.Annotation) > 0 {
		sort.Strings(build.Annotation)
	}

	if len(build.Secret) > 0 {
		sort.Strings(build.Secret)
	}

	if len(build.Network) > 0 {
		sort.Strings(build.Network)
	}

	if len(build.Volume) > 0 {
		sort.Strings(build.Volume)
	}

	if len(build.PodmanArgs) > 0 {
		sort.Strings(build.PodmanArgs)
	}

	// Sort environment variables keys
	// This will be done when generating the unit file content
}
