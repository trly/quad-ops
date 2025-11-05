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

type Renderer struct{}

type Lifecycle struct{}

type Options struct{}

func OptionsFromSettings(_, _ string, _ bool) Options {
	return Options{}
}

func NewRenderer(_ Options, _ log.Logger) (*Renderer, error) {
	return nil, fmt.Errorf("launchd renderer is only available on darwin, current platform: %s", runtime.GOOS)
}

func NewLifecycle(_ Options, _ execx.Runner, _ log.Logger) (*Lifecycle, error) {
	return nil, fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

func (r *Renderer) Name() string {
	return "launchd"
}

func (r *Renderer) Render(_ context.Context, _ []service.Spec) (*platform.RenderResult, error) {
	return nil, fmt.Errorf("launchd renderer is only available on darwin, current platform: %s", runtime.GOOS)
}

func (l *Lifecycle) Name() string {
	return "launchd"
}

func (l *Lifecycle) Reload(_ context.Context) error {
	return fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

func (l *Lifecycle) Start(_ context.Context, _ string) error {
	return fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

func (l *Lifecycle) Stop(_ context.Context, _ string) error {
	return fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

func (l *Lifecycle) Restart(_ context.Context, _ string) error {
	return fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

func (l *Lifecycle) Status(_ context.Context, _ string) (*platform.ServiceStatus, error) {
	return nil, fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)
}

func (l *Lifecycle) StartMany(_ context.Context, _ []string) map[string]error {
	return map[string]error{"_": fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)}
}

func (l *Lifecycle) StopMany(_ context.Context, _ []string) map[string]error {
	return map[string]error{"_": fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)}
}

func (l *Lifecycle) RestartMany(_ context.Context, _ []string) map[string]error {
	return map[string]error{"_": fmt.Errorf("launchd lifecycle is only available on darwin, current platform: %s", runtime.GOOS)}
}
