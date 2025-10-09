package launchd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/service"
)

// Renderer implements platform.Renderer for macOS launchd.
type Renderer struct {
	opts   Options
	logger log.Logger
}

// NewRenderer creates a new launchd renderer.
func NewRenderer(opts Options, logger log.Logger) (*Renderer, error) {
	// Validate and normalize options
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	return &Renderer{
		opts:   opts,
		logger: logger,
	}, nil
}

// Name returns the platform name.
func (r *Renderer) Name() string {
	return "launchd"
}

// Render converts service specs to launchd plist artifacts.
func (r *Renderer) Render(_ context.Context, specs []service.Spec) (*platform.RenderResult, error) {
	result := &platform.RenderResult{
		Artifacts:      []platform.Artifact{},
		ServiceChanges: make(map[string]platform.ChangeStatus),
	}

	for _, spec := range specs {
		artifacts, err := r.renderService(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to render service %s: %w", spec.Name, err)
		}

		// Track artifacts for this service
		var paths []string
		var combinedHash string

		for _, artifact := range artifacts {
			result.Artifacts = append(result.Artifacts, artifact)
			paths = append(paths, artifact.Path)

			// Combine hashes for change detection
			if combinedHash == "" {
				combinedHash = artifact.Hash
			} else {
				h := sha256.New()
				h.Write([]byte(combinedHash))
				h.Write([]byte(artifact.Hash))
				combinedHash = fmt.Sprintf("%x", h.Sum(nil))
			}
		}

		result.ServiceChanges[spec.Name] = platform.ChangeStatus{
			Changed:       false, // Let ArtifactStore determine changes via content hash
			ArtifactPaths: paths,
			ContentHash:   combinedHash,
		}
	}

	return result, nil
}

// renderService renders a single service to plist artifact(s).
func (r *Renderer) renderService(spec service.Spec) ([]platform.Artifact, error) {
	// Generate label and container name
	label := r.buildLabel(spec.Name)
	containerName := label

	// Build podman command arguments
	podmanArgs := BuildPodmanArgs(spec, containerName)

	// Determine restart policy mapping
	keepAlive := r.mapRestartPolicy(spec.Container.RestartPolicy)

	// Build plist
	plist := &Plist{
		Label:               label,
		Program:             r.opts.PodmanPath,
		ProgramArguments:    append([]string{r.opts.PodmanPath}, podmanArgs...),
		RunAtLoad:           true,
		KeepAlive:           keepAlive,
		ThrottleInterval:    30,
		AbandonProcessGroup: false,
		ProcessType:         "Background",
		StandardOutPath:     filepath.Join(r.opts.LogsDir, fmt.Sprintf("%s.out.log", label)),
		StandardErrorPath:   filepath.Join(r.opts.LogsDir, fmt.Sprintf("%s.err.log", label)),
	}

	// Add working directory if specified
	if spec.Container.WorkingDir != "" {
		plist.WorkingDirectory = spec.Container.WorkingDir
	}

	// Add user/group for system domain
	if r.opts.Domain == DomainSystem {
		if spec.Container.User != "" {
			plist.UserName = spec.Container.User
		}
		if spec.Container.Group != "" {
			plist.GroupName = spec.Container.Group
		}
	}

	// Encode plist
	content, err := EncodePlist(plist)
	if err != nil {
		return nil, fmt.Errorf("failed to encode plist: %w", err)
	}

	// Calculate content hash
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	// Build artifact path
	artifactPath := fmt.Sprintf("%s.plist", label)

	artifact := platform.Artifact{
		Path:    artifactPath,
		Content: content,
		Mode:    0644,
		Hash:    hash,
	}

	r.logger.Debug("Rendered launchd plist",
		"service", spec.Name,
		"label", label,
		"path", artifactPath,
	)

	return []platform.Artifact{artifact}, nil
}

// buildLabel creates a launchd label from service name.
func (r *Renderer) buildLabel(serviceName string) string {
	return SanitizeLabel(fmt.Sprintf("%s.%s", r.opts.LabelPrefix, serviceName))
}

// mapRestartPolicy maps service.RestartPolicy to launchd KeepAlive.
func (r *Renderer) mapRestartPolicy(policy service.RestartPolicy) interface{} {
	switch policy {
	case service.RestartPolicyNo:
		return false
	case service.RestartPolicyAlways, service.RestartPolicyUnlessStopped:
		return true
	case service.RestartPolicyOnFailure:
		// Restart only on failure (non-zero exit)
		return map[string]bool{
			"SuccessfulExit": false,
		}
	default:
		// Default to always restart
		return true
	}
}
