//go:build !darwin

package launchd

import (
	"context"
	"fmt"
	"runtime"

	"github.com/trly/quad-ops/internal/execx"
	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/service"
)

// Renderer is a stub for non-Darwin platforms.
type Renderer struct{}

// Lifecycle is a stub for non-Darwin platforms.
type Lifecycle struct{}

// Options is a stub for non-Darwin platforms.
type Options struct {
	LabelPrefix string
}

// OptionsFromSettings is a stub for non-Darwin platforms.
func OptionsFromSettings(_, _ string, _ bool) Options {
	return Options{}
}

// NewRenderer is a stub for non-Darwin platforms.
func NewRenderer(_ Options, _ log.Logger) (*Renderer, error) {
	return nil, fmt.Errorf("launchd renderer is only available on darwin, current platform: %s", runtime.GOOS)
}

// NewLifecycle is a stub for non-Darwin platforms.
func NewLifecycle(_ Options, _ execx.Runner, _ log.Logger) (*Lifecycle, error) {
	return nil, fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

// Name returns the platform name.
func (r *Renderer) Name() string {
	return "launchd"
}

// Render is a stub for non-Darwin platforms.
func (r *Renderer) Render(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
	return nil, fmt.Errorf("launchd renderer is only available on darwin, current platform: %s", runtime.GOOS)
}

// Name returns the platform name.
func (l *Lifecycle) Name() string {
	return "launchd"
}

// Reload is a stub for non-Darwin platforms.
func (l *Lifecycle) Reload(_ context.Context) error {
	return fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

// Start is a stub for non-Darwin platforms.
func (l *Lifecycle) Start(_ context.Context, _ string) error {
	return fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

// Stop is a stub for non-Darwin platforms.
func (l *Lifecycle) Stop(_ context.Context, _ string) error {
	return fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

// Restart is a stub for non-Darwin platforms.
func (l *Lifecycle) Restart(_ context.Context, _ string) error {
	return fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

// Status is a stub for non-Darwin platforms.
func (l *Lifecycle) Status(_ context.Context, _ string) (*platform.ServiceStatus, error) {
	return nil, fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

// StartMany is a stub for non-Darwin platforms.
func (l *Lifecycle) StartMany(_ context.Context, _ []string) map[string]error {
	return map[string]error{"_": fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)}
}

// StopMany is a stub for non-Darwin platforms.
func (l *Lifecycle) StopMany(_ context.Context, _ []string) map[string]error {
	return map[string]error{"_": fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)}
}

// RestartMany is a stub for non-Darwin platforms.
func (l *Lifecycle) RestartMany(_ context.Context, _ []string) map[string]error {
	return map[string]error{"_": fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)}
}
