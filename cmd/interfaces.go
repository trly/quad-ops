package cmd

import (
	"context"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/service"
)

// SystemValidator provides system validation capabilities for commands.
type SystemValidator interface {
	SystemRequirements() error
}

// GitSyncerInterface wraps repository.GitSyncer for testing.
type GitSyncerInterface interface {
	SyncAll(ctx context.Context, repos []config.Repository) ([]repository.SyncResult, error)
	SyncRepo(ctx context.Context, repo config.Repository) repository.SyncResult
}

// ComposeProcessorInterface processes Docker Compose projects to service specs.
type ComposeProcessorInterface interface {
	Process(ctx context.Context, project *types.Project) ([]service.Spec, error)
}

// RendererInterface wraps platform.Renderer for testing.
type RendererInterface interface {
	Name() string
	Render(ctx context.Context, specs []service.Spec) (*platform.RenderResult, error)
}

// ArtifactStoreInterface wraps repository.ArtifactStore for testing.
type ArtifactStoreInterface interface {
	Write(ctx context.Context, artifacts []platform.Artifact) ([]string, error)
	List(ctx context.Context) ([]platform.Artifact, error)
	Delete(ctx context.Context, paths []string) error
}

// LifecycleInterface wraps platform.Lifecycle for testing.
type LifecycleInterface interface {
	Name() string
	Reload(ctx context.Context) error
	Start(ctx context.Context, name string) error
	Stop(ctx context.Context, name string) error
	Restart(ctx context.Context, name string) error
	Status(ctx context.Context, name string) (*platform.ServiceStatus, error)
	StartMany(ctx context.Context, names []string) map[string]error
	StopMany(ctx context.Context, names []string) map[string]error
	RestartMany(ctx context.Context, names []string) map[string]error
	Exists(ctx context.Context, name string) (bool, error)
}
