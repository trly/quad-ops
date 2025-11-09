package systemd

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

// TestRequiresMountsFor_SingleBindMount verifies that a container with a single
// bind mount gets a RequiresMountsFor directive for the host path.
func TestRequiresMountsFor_SingleBindMount(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App with bind mount",
		Container: service.Container{
			Image: "alpine:latest",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeBind,
					Source: "/host/data",
					Target: "/app/data",
				},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Extract [Unit] section
	unitStart := strings.Index(containerContent, "[Unit]")
	containerStart := strings.Index(containerContent, "[Container]")
	require.Greater(t, containerStart, unitStart)
	unitSection := containerContent[unitStart:containerStart]

	// CRITICAL: Bind mount must have RequiresMountsFor directive
	assert.Contains(t, unitSection, "RequiresMountsFor=/host/data",
		"Bind mount must include RequiresMountsFor directive for host path")
}

// TestRequiresMountsFor_MultipleBindMounts verifies that a container with
// multiple bind mounts gets RequiresMountsFor directives for all host paths.
func TestRequiresMountsFor_MultipleBindMounts(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App with multiple bind mounts",
		Container: service.Container{
			Image: "alpine:latest",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeBind,
					Source: "/host/data",
					Target: "/app/data",
				},
				{
					Type:   service.MountTypeBind,
					Source: "/host/config",
					Target: "/app/config",
				},
				{
					Type:   service.MountTypeBind,
					Source: "/mnt/nfs/shared",
					Target: "/app/shared",
				},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Extract [Unit] section
	unitStart := strings.Index(containerContent, "[Unit]")
	containerStart := strings.Index(containerContent, "[Container]")
	require.Greater(t, containerStart, unitStart)
	unitSection := containerContent[unitStart:containerStart]

	// All bind mount paths must have RequiresMountsFor directives
	assert.Contains(t, unitSection, "RequiresMountsFor=/host/config")
	assert.Contains(t, unitSection, "RequiresMountsFor=/host/data")
	assert.Contains(t, unitSection, "RequiresMountsFor=/mnt/nfs/shared")

	// Verify they appear in sorted order (deterministic output)
	configPos := strings.Index(unitSection, "RequiresMountsFor=/host/config")
	dataPos := strings.Index(unitSection, "RequiresMountsFor=/host/data")
	nfsPos := strings.Index(unitSection, "RequiresMountsFor=/mnt/nfs/shared")

	assert.Less(t, configPos, dataPos, "RequiresMountsFor should be in sorted order")
	assert.Less(t, dataPos, nfsPos, "RequiresMountsFor should be in sorted order")
}

// TestRequiresMountsFor_MixedMountTypes verifies that only bind mounts get
// RequiresMountsFor directives, not volume or tmpfs mounts.
func TestRequiresMountsFor_MixedMountTypes(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App with mixed mount types",
		Container: service.Container{
			Image: "alpine:latest",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeBind,
					Source: "/host/data",
					Target: "/app/data",
				},
				{
					Type:   service.MountTypeVolume,
					Source: "my-volume",
					Target: "/app/vol",
				},
				{
					Type:   service.MountTypeTmpfs,
					Target: "/tmp",
				},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Extract [Unit] section
	unitStart := strings.Index(containerContent, "[Unit]")
	containerStart := strings.Index(containerContent, "[Container]")
	require.Greater(t, containerStart, unitStart)
	unitSection := containerContent[unitStart:containerStart]

	// Only bind mount should have RequiresMountsFor
	assert.Contains(t, unitSection, "RequiresMountsFor=/host/data",
		"Bind mount must have RequiresMountsFor directive")

	// Volume and tmpfs mounts should NOT have RequiresMountsFor
	assert.NotContains(t, unitSection, "RequiresMountsFor=my-volume",
		"Volume mounts should not have RequiresMountsFor directive")
	assert.NotContains(t, unitSection, "RequiresMountsFor=/tmp",
		"Tmpfs mounts should not have RequiresMountsFor directive")
}

// TestRequiresMountsFor_NoBindMounts verifies that containers without bind
// mounts don't get RequiresMountsFor directives.
func TestRequiresMountsFor_NoBindMounts(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App without bind mounts",
		Container: service.Container{
			Image: "alpine:latest",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeVolume,
					Source: "data",
					Target: "/app/data",
				},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Should NOT have RequiresMountsFor directive
	assert.NotContains(t, containerContent, "RequiresMountsFor=",
		"Containers without bind mounts should not have RequiresMountsFor directives")
}

// TestRequiresMountsFor_ReadOnlyBindMount verifies that read-only bind mounts
// still get RequiresMountsFor directives.
func TestRequiresMountsFor_ReadOnlyBindMount(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App with read-only bind mount",
		Container: service.Container{
			Image: "alpine:latest",
			Mounts: []service.Mount{
				{
					Type:     service.MountTypeBind,
					Source:   "/host/config",
					Target:   "/app/config",
					ReadOnly: true,
				},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Extract [Unit] section
	unitStart := strings.Index(containerContent, "[Unit]")
	containerStart := strings.Index(containerContent, "[Container]")
	require.Greater(t, containerStart, unitStart)
	unitSection := containerContent[unitStart:containerStart]

	// Read-only bind mount still needs RequiresMountsFor
	assert.Contains(t, unitSection, "RequiresMountsFor=/host/config",
		"Read-only bind mount must have RequiresMountsFor directive")
}

// TestRequiresMountsFor_Ordering verifies that RequiresMountsFor directives
// appear in the [Unit] section in sorted order for deterministic output.
func TestRequiresMountsFor_Ordering(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "web",
		Description: "Web server with bind mounts and dependencies",
		Container: service.Container{
			Image: "nginx:latest",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeBind,
					Source: "/var/www",
					Target: "/usr/share/nginx/html",
				},
				{
					Type:   service.MountTypeBind,
					Source: "/etc/nginx/conf",
					Target: "/etc/nginx",
				},
			},
		},
		DependsOn: []string{"db"},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Extract [Unit] section
	unitStart := strings.Index(containerContent, "[Unit]")
	containerStart := strings.Index(containerContent, "[Container]")
	require.Greater(t, containerStart, unitStart)
	unitSection := containerContent[unitStart:containerStart]

	// Both RequiresMountsFor directives should be present
	assert.Contains(t, unitSection, "RequiresMountsFor=/etc/nginx/conf")
	assert.Contains(t, unitSection, "RequiresMountsFor=/var/www")

	// Verify they appear in sorted order
	confPos := strings.Index(unitSection, "RequiresMountsFor=/etc/nginx/conf")
	wwwPos := strings.Index(unitSection, "RequiresMountsFor=/var/www")
	assert.Less(t, confPos, wwwPos, "RequiresMountsFor should be in sorted order")
}

// TestRequiresMountsFor_EmptySourcePath verifies that bind mounts with empty
// source paths don't add RequiresMountsFor directives.
func TestRequiresMountsFor_EmptySourcePath(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App with bind mount without source",
		Container: service.Container{
			Image: "alpine:latest",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeBind,
					Source: "", // Empty source
					Target: "/app/data",
				},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Should NOT have RequiresMountsFor directive for empty source
	assert.NotContains(t, containerContent, "RequiresMountsFor=",
		"Bind mounts with empty source should not have RequiresMountsFor directives")
}

// TestRequiresMountsFor_RelativePathBindMount verifies that bind mounts with
// relative paths get RequiresMountsFor directives.
func TestRequiresMountsFor_RelativePathBindMount(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App with relative path bind mount",
		Container: service.Container{
			Image: "alpine:latest",
			Mounts: []service.Mount{
				{
					Type:   service.MountTypeBind,
					Source: "./data",
					Target: "/app/data",
				},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Extract [Unit] section
	unitStart := strings.Index(containerContent, "[Unit]")
	containerStart := strings.Index(containerContent, "[Container]")
	require.Greater(t, containerStart, unitStart)
	unitSection := containerContent[unitStart:containerStart]

	// Relative path bind mount should have RequiresMountsFor
	assert.Contains(t, unitSection, "RequiresMountsFor=./data",
		"Relative path bind mount must have RequiresMountsFor directive")
}
