package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
)

func TestExtensions_Integration_BuildArgs(t *testing.T) {
	baseVal := "ubuntu"
	versionVal := "1.0"

	// Create project with build args and x-podman-buildargs extension
	project := &types.Project{
		Name:       "test-project",
		WorkingDir: ".",
		Services: map[string]types.ServiceConfig{
			"app": {
				Name: "app",
				Build: &types.BuildConfig{
					Context:    ".",
					Dockerfile: "Dockerfile",
					Args: types.MappingWithEquals{
						"BASE":    &baseVal,
						"VERSION": &versionVal,
					},
				},
				Extensions: map[string]interface{}{
					"x-podman-buildargs": map[string]interface{}{
						"BUILDKIT_INLINE_CACHE": "1",
						"BUILDPLATFORM":         "linux/amd64",
						"VERSION":               "2.0",
					},
				},
			},
		},
	}

	// Convert to service specs
	sc := NewSpecConverter(".")
	specs, err := sc.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	// Verify build args were merged
	app := specs[0]
	require.NotNil(t, app.Container.Build)
	require.NotNil(t, app.Container.Build.Args)

	// Compose args should be present
	assert.Equal(t, "ubuntu", app.Container.Build.Args["BASE"])

	// Podman args should be present and override compose args
	assert.Equal(t, "2.0", app.Container.Build.Args["VERSION"])
	assert.Equal(t, "1", app.Container.Build.Args["BUILDKIT_INLINE_CACHE"])
	assert.Equal(t, "linux/amd64", app.Container.Build.Args["BUILDPLATFORM"])
}

func TestExtensions_Integration_Volumes(t *testing.T) {
	// Create project with compose volumes and x-podman-volumes extension
	project := &types.Project{
		Name:       "test-project",
		WorkingDir: ".",
		Services: map[string]types.ServiceConfig{
			"app": {
				Name:  "app",
				Image: "nginx:latest",
				Volumes: []types.ServiceVolumeConfig{
					{
						Source: "./html",
						Target: "/usr/share/nginx/html",
						Type:   "bind",
					},
				},
				Extensions: map[string]interface{}{
					"x-podman-volumes": []interface{}{
						"cache:/tmp/cache:O",
						"logs:/logs:U",
						"/data:/data:ro",
					},
				},
			},
		},
	}

	// Convert to service specs
	sc := NewSpecConverter(".")
	specs, err := sc.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	// Verify mounts include both compose volumes and podman volumes
	app := specs[0]
	require.NotNil(t, app.Container.Mounts)

	// Should have at least 4 mounts (1 from compose, 3 from x-podman-volumes)
	assert.GreaterOrEqual(t, len(app.Container.Mounts), 4)

	// Verify specific mounts
	var cacheMount, logsMount, dataMount, htmlMount *service.Mount
	for i := range app.Container.Mounts {
		mount := &app.Container.Mounts[i]
		switch mount.Target {
		case "/tmp/cache":
			cacheMount = mount
		case "/logs":
			logsMount = mount
		case "/data":
			dataMount = mount
		case "/usr/share/nginx/html":
			htmlMount = mount
		}
	}

	// Check cache mount (volume)
	require.NotNil(t, cacheMount, "cache mount not found")
	assert.Equal(t, "test-project-cache", cacheMount.Source)
	assert.Equal(t, "/tmp/cache", cacheMount.Target)

	// Check logs mount (volume)
	require.NotNil(t, logsMount, "logs mount not found")
	assert.Equal(t, "test-project-logs", logsMount.Source)
	assert.Equal(t, "/logs", logsMount.Target)

	// Check data mount (bind)
	require.NotNil(t, dataMount, "data mount not found")
	assert.Equal(t, "/data", dataMount.Source)
	assert.Equal(t, "/data", dataMount.Target)
	assert.True(t, dataMount.ReadOnly)

	// Check html mount (bind from compose)
	require.NotNil(t, htmlMount, "html mount not found")
	assert.Equal(t, htmlMount.Target, "/usr/share/nginx/html")
}
