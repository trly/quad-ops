package cmd

import (
	"context"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/trly/quad-ops/internal/config"
	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/fs"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/repository"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

// MockValidator implements SystemValidator for testing.
type MockValidator struct {
	SystemRequirementsFunc func() error
}

func (m *MockValidator) SystemRequirements() error {
	if m.SystemRequirementsFunc != nil {
		return m.SystemRequirementsFunc()
	}
	return nil
}

// MockRenderer implements RendererInterface for testing.
type MockRenderer struct {
	NameFunc   func() string
	RenderFunc func(context.Context, []service.Spec) (*platform.RenderResult, error)
}

func (m *MockRenderer) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock"
}

func (m *MockRenderer) Render(ctx context.Context, specs []service.Spec) (*platform.RenderResult, error) {
	if m.RenderFunc != nil {
		return m.RenderFunc(ctx, specs)
	}
	return &platform.RenderResult{
		Artifacts:      []platform.Artifact{},
		ServiceChanges: map[string]platform.ChangeStatus{},
	}, nil
}

// MockLifecycle implements LifecycleInterface for testing.
type MockLifecycle struct {
	NameFunc        func() string
	ReloadFunc      func(context.Context) error
	StartFunc       func(context.Context, string) error
	StopFunc        func(context.Context, string) error
	RestartFunc     func(context.Context, string) error
	StatusFunc      func(context.Context, string) (*platform.ServiceStatus, error)
	StartManyFunc   func(context.Context, []string) map[string]error
	StopManyFunc    func(context.Context, []string) map[string]error
	RestartManyFunc func(context.Context, []string) map[string]error
}

func (m *MockLifecycle) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock"
}

func (m *MockLifecycle) Reload(_ context.Context) error {
	if m.ReloadFunc != nil {
		return m.ReloadFunc(context.Background())
	}
	return nil
}

func (m *MockLifecycle) Start(_ context.Context, name string) error {
	if m.StartFunc != nil {
		return m.StartFunc(context.Background(), name)
	}
	return nil
}

func (m *MockLifecycle) Stop(_ context.Context, name string) error {
	if m.StopFunc != nil {
		return m.StopFunc(context.Background(), name)
	}
	return nil
}

func (m *MockLifecycle) Restart(_ context.Context, name string) error {
	if m.RestartFunc != nil {
		return m.RestartFunc(context.Background(), name)
	}
	return nil
}

func (m *MockLifecycle) Status(_ context.Context, name string) (*platform.ServiceStatus, error) {
	if m.StatusFunc != nil {
		return m.StatusFunc(context.Background(), name)
	}
	return &platform.ServiceStatus{Name: name}, nil
}

func (m *MockLifecycle) StartMany(_ context.Context, names []string) map[string]error {
	if m.StartManyFunc != nil {
		return m.StartManyFunc(context.Background(), names)
	}
	return make(map[string]error)
}

func (m *MockLifecycle) StopMany(_ context.Context, names []string) map[string]error {
	if m.StopManyFunc != nil {
		return m.StopManyFunc(context.Background(), names)
	}
	return make(map[string]error)
}

func (m *MockLifecycle) RestartMany(_ context.Context, names []string) map[string]error {
	if m.RestartManyFunc != nil {
		return m.RestartManyFunc(context.Background(), names)
	}
	return make(map[string]error)
}

// MockComposeProcessor implements ComposeProcessorInterface for testing.
type MockComposeProcessor struct {
	ProcessFunc func(context.Context, *types.Project) ([]service.Spec, error)
}

func (m *MockComposeProcessor) Process(ctx context.Context, project *types.Project) ([]service.Spec, error) {
	if m.ProcessFunc != nil {
		return m.ProcessFunc(ctx, project)
	}
	return []service.Spec{}, nil
}

// MockArtifactStore implements repository.ArtifactStore for testing.
type MockArtifactStore struct {
	WriteFunc  func(context.Context, []platform.Artifact) ([]string, error)
	ListFunc   func(context.Context) ([]platform.Artifact, error)
	DeleteFunc func(context.Context, []string) error
}

func (m *MockArtifactStore) Write(ctx context.Context, artifacts []platform.Artifact) ([]string, error) {
	if m.WriteFunc != nil {
		return m.WriteFunc(ctx, artifacts)
	}
	return []string{}, nil
}

func (m *MockArtifactStore) List(ctx context.Context) ([]platform.Artifact, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return []platform.Artifact{}, nil
}

func (m *MockArtifactStore) Delete(ctx context.Context, paths []string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, paths)
	}
	return nil
}

// MockGitSyncer implements GitSyncerInterface for testing.
type MockGitSyncer struct {
	SyncAllFunc  func(context.Context, []config.Repository) ([]repository.SyncResult, error)
	SyncRepoFunc func(context.Context, config.Repository) repository.SyncResult
}

func (m *MockGitSyncer) SyncAll(ctx context.Context, repos []config.Repository) ([]repository.SyncResult, error) {
	if m.SyncAllFunc != nil {
		return m.SyncAllFunc(ctx, repos)
	}
	return []repository.SyncResult{}, nil
}

func (m *MockGitSyncer) SyncRepo(ctx context.Context, repo config.Repository) repository.SyncResult {
	if m.SyncRepoFunc != nil {
		return m.SyncRepoFunc(ctx, repo)
	}
	return repository.SyncResult{Repository: repo, Success: true}
}

// AppBuilder provides a fluent interface for building test Apps.
type AppBuilder struct {
	logger           log.Logger
	config           *config.Settings
	validator        SystemValidator
	renderer         RendererInterface
	lifecycle        LifecycleInterface
	artifactStore    repository.ArtifactStore
	composeProcessor ComposeProcessorInterface
	os               string
}

// NewAppBuilder creates a new AppBuilder with sensible defaults.
func NewAppBuilder(t *testing.T) *AppBuilder {
	return &AppBuilder{
		logger:    testutil.NewTestLogger(t),
		config:    &config.Settings{Verbose: false},
		validator: &MockValidator{},
	}
}

func (b *AppBuilder) WithValidator(v SystemValidator) *AppBuilder {
	b.validator = v
	return b
}

func (b *AppBuilder) WithConfig(c *config.Settings) *AppBuilder {
	b.config = c
	return b
}

func (b *AppBuilder) WithVerbose(verbose bool) *AppBuilder {
	b.config.Verbose = verbose
	return b
}

func (b *AppBuilder) WithRenderer(r RendererInterface) *AppBuilder {
	b.renderer = r
	return b
}

func (b *AppBuilder) WithLifecycle(l LifecycleInterface) *AppBuilder {
	b.lifecycle = l
	return b
}

func (b *AppBuilder) WithOS(os string) *AppBuilder {
	b.os = os
	return b
}

func (b *AppBuilder) WithArtifactStore(a repository.ArtifactStore) *AppBuilder {
	b.artifactStore = a
	return b
}

func (b *AppBuilder) WithComposeProcessor(cp ComposeProcessorInterface) *AppBuilder {
	b.composeProcessor = cp
	return b
}

func (b *AppBuilder) Build(t *testing.T) *App {
	return &App{
		Logger:           b.logger,
		Config:           b.config,
		ConfigProvider:   testutil.NewMockConfig(t),
		Runner:           &execx.RealRunner{},
		FSService:        &fs.Service{},
		Validator:        b.validator,
		renderer:         b.renderer,
		lifecycle:        b.lifecycle,
		ArtifactStore:    b.artifactStore,
		ComposeProcessor: b.composeProcessor,
		os:               b.os,
	}
}
