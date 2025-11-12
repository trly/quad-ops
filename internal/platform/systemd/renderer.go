package systemd

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/service"
)

// Renderer implements platform.Renderer for systemd/Quadlet.
type Renderer struct {
	logger log.Logger
}

// NewRenderer creates a new systemd renderer.
func NewRenderer(logger log.Logger) *Renderer {
	return &Renderer{
		logger: logger,
	}
}

// Name returns the platform name.
func (r *Renderer) Name() string {
	return "systemd"
}

// Render converts service specs to systemd Quadlet unit files.
func (r *Renderer) Render(_ context.Context, specs []service.Spec) (*platform.RenderResult, error) {
	result := &platform.RenderResult{
		Artifacts:      make([]platform.Artifact, 0),
		ServiceChanges: make(map[string]platform.ChangeStatus),
	}

	for _, spec := range specs {
		artifacts, err := r.renderService(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to render service %s: %w", spec.Name, err)
		}

		artifactPaths := make([]string, 0, len(artifacts))
		contentHashes := make([]string, 0, len(artifacts))

		for _, artifact := range artifacts {
			artifactPaths = append(artifactPaths, artifact.Path)
			contentHashes = append(contentHashes, artifact.Hash)
		}

		combinedHash := r.combineHashes(contentHashes)

		result.Artifacts = append(result.Artifacts, artifacts...)
		result.ServiceChanges[spec.Name] = platform.ChangeStatus{
			Changed:       false,
			ArtifactPaths: artifactPaths,
			ContentHash:   combinedHash,
		}
	}

	return result, nil
}

// renderService renders a single service spec into one or more artifacts.
func (r *Renderer) renderService(spec service.Spec) ([]platform.Artifact, error) {
	artifacts := make([]platform.Artifact, 0)

	// Render volumes first
	for _, vol := range spec.Volumes {
		if !vol.External {
			content := renderVolume(vol)
			hash := r.computeHash(content)
			artifacts = append(artifacts, platform.Artifact{
				Path:    vol.Name + UnitSuffixVolume,
				Content: []byte(content),
				Mode:    0644,
				Hash:    hash,
			})
		}
	}

	// Render networks (both project-local and external)
	for _, net := range spec.Networks {
		content := renderNetwork(net)
		hash := r.computeHash(content)
		artifacts = append(artifacts, platform.Artifact{
			Path:    net.Name + UnitSuffixNetwork,
			Content: []byte(content),
			Mode:    0644,
			Hash:    hash,
		})
	}

	// Render build unit if needed
	if spec.Container.Build != nil {
		content := renderBuild(spec.Name, spec.Description, *spec.Container.Build, spec.DependsOn)
		hash := r.computeHash(content)
		artifacts = append(artifacts, platform.Artifact{
			Path:    spec.Name + "-build" + UnitSuffixBuild,
			Content: []byte(content),
			Mode:    0644,
			Hash:    hash,
		})
	}

	// Render container unit
	content := renderContainer(spec)
	hash := r.computeHash(content)
	artifacts = append(artifacts, platform.Artifact{
		Path:    spec.Name + UnitSuffixContainer,
		Content: []byte(content),
		Mode:    0644,
		Hash:    hash,
	})

	return artifacts, nil
}

// computeHash computes SHA256 hash of content.
func (r *Renderer) computeHash(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// combineHashes combines multiple hashes into a single hash.
func (r *Renderer) combineHashes(hashes []string) string {
	h := sha256.New()
	for _, hash := range hashes {
		h.Write([]byte(hash))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
