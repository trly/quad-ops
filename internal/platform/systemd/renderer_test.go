package systemd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestRenderer_Name(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	assert.Equal(t, "systemd", r.Name())
}

func TestRenderer_RenderContainer(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "web",
		Description: "Web server",
		Container: service.Container{
			Image:         "nginx:latest",
			ContainerName: "my-web",
			Hostname:      "web.local",
			Env: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
			Ports: []service.Port{
				{HostPort: 8080, Container: 80, Protocol: "tcp"},
			},
			RestartPolicy: service.RestartPolicyAlways,
		},
		DependsOn: []string{"db"},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.Artifacts, 1)
	assert.Equal(t, "web.container", result.Artifacts[0].Path)
	assert.NotEmpty(t, result.Artifacts[0].Hash)

	content := string(result.Artifacts[0].Content)
	assert.Contains(t, content, "[Unit]")
	assert.Contains(t, content, "Description=Web server")
	assert.Contains(t, content, "After=db.service")
	assert.Contains(t, content, "Requires=db.service")
	assert.Contains(t, content, "[Container]")
	assert.Contains(t, content, "Image=nginx:latest")
	assert.Contains(t, content, "ContainerName=my-web")
	assert.Contains(t, content, "HostName=web.local")
	assert.Contains(t, content, "Environment=BAZ=qux")
	assert.Contains(t, content, "Environment=FOO=bar")
	assert.Contains(t, content, "PublishPort=8080:80")
	assert.Contains(t, content, "[Service]")
	assert.Contains(t, content, "Restart=always")
	assert.Contains(t, content, "[Install]")
	assert.Contains(t, content, "WantedBy=default.target")

	assert.Contains(t, result.ServiceChanges, "web")
	assert.False(t, result.ServiceChanges["web"].Changed)
	assert.Equal(t, []string{"web.container"}, result.ServiceChanges["web"].ArtifactPaths)
}

func TestRenderer_RenderVolume(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App with volume",
		Volumes: []service.Volume{
			{
				Name:   "data",
				Driver: "local",
				Options: map[string]string{
					"type":   "tmpfs",
					"device": "tmpfs",
				},
				Labels: map[string]string{
					"env": "test",
				},
			},
		},
		Container: service.Container{
			Image: "alpine:latest",
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	assert.Len(t, result.Artifacts, 2)

	var volumeArtifact *string
	for _, a := range result.Artifacts {
		if a.Path == "data.volume" {
			content := string(a.Content)
			volumeArtifact = &content
			break
		}
	}

	require.NotNil(t, volumeArtifact)
	assert.Contains(t, *volumeArtifact, "[Unit]")
	assert.Contains(t, *volumeArtifact, "[Volume]")
	assert.Contains(t, *volumeArtifact, "VolumeName=data")
	assert.Contains(t, *volumeArtifact, "Options=device=tmpfs")
	assert.Contains(t, *volumeArtifact, "Options=type=tmpfs")
	assert.Contains(t, *volumeArtifact, "Label=env=test")
}

func TestRenderer_RenderNetwork(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App with network",
		Networks: []service.Network{
			{
				Name:     "backend",
				Driver:   "bridge",
				Internal: true,
				IPv6:     true,
				IPAM: &service.IPAM{
					Config: []service.IPAMConfig{
						{
							Subnet:  "172.20.0.0/16",
							Gateway: "172.20.0.1",
						},
					},
				},
			},
		},
		Container: service.Container{
			Image: "alpine:latest",
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	assert.Len(t, result.Artifacts, 2)

	var networkArtifact *string
	for _, a := range result.Artifacts {
		if a.Path == "backend.network" {
			content := string(a.Content)
			networkArtifact = &content
			break
		}
	}

	require.NotNil(t, networkArtifact)
	assert.Contains(t, *networkArtifact, "[Unit]")
	assert.Contains(t, *networkArtifact, "[Network]")
	assert.Contains(t, *networkArtifact, "NetworkName=backend")
	assert.Contains(t, *networkArtifact, "Subnet=172.20.0.0/16")
	assert.Contains(t, *networkArtifact, "Gateway=172.20.0.1")
	assert.Contains(t, *networkArtifact, "IPv6=yes")
	assert.Contains(t, *networkArtifact, "Internal=yes")
}

func TestRenderer_RenderBuild(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "builder",
		Description: "Build service",
		Container: service.Container{
			Image: "myapp:latest",
			Build: &service.Build{
				Context:             "./app",
				Dockerfile:          "Dockerfile.prod",
				Target:              "production",
				Pull:                true,
				SetWorkingDirectory: "/build",
				Tags:                []string{"myapp:latest", "myapp:v1.0"},
				Args: map[string]string{
					"VERSION": "1.0",
				},
				Labels: map[string]string{
					"version": "1.0",
				},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	var buildArtifact *string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".build") {
			content := string(a.Content)
			buildArtifact = &content
			break
		}
	}

	require.NotNil(t, buildArtifact)
	assert.Contains(t, *buildArtifact, "[Unit]")
	assert.Contains(t, *buildArtifact, "[Build]")
	assert.Contains(t, *buildArtifact, "ImageTag=myapp:latest")
	assert.Contains(t, *buildArtifact, "ImageTag=myapp:v1.0")
	assert.Contains(t, *buildArtifact, "File=Dockerfile.prod")
	assert.Contains(t, *buildArtifact, "Target=production")
	assert.Contains(t, *buildArtifact, "Pull=always")
	assert.Contains(t, *buildArtifact, "SetWorkingDirectory=/build")
	assert.Contains(t, *buildArtifact, "Environment=VERSION=1.0")
	assert.Contains(t, *buildArtifact, "Label=version=1.0")
}

func TestRenderer_RenderHealthcheck(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "app",
		Container: service.Container{
			Image: "nginx:latest",
			Healthcheck: &service.Healthcheck{
				Test:          []string{"CMD", "curl -f http://localhost/health"},
				Interval:      30 * time.Second,
				Timeout:       10 * time.Second,
				Retries:       3,
				StartPeriod:   60 * time.Second,
				StartInterval: 5 * time.Second,
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	content := string(result.Artifacts[0].Content)
	assert.Contains(t, content, "HealthCmd=CMD curl -f http://localhost/health")
	assert.Contains(t, content, "HealthInterval=30s")
	assert.Contains(t, content, "HealthTimeout=10s")
	assert.Contains(t, content, "HealthRetries=3")
	assert.Contains(t, content, "HealthStartPeriod=1m")
	assert.Contains(t, content, "HealthStartupInterval=5s")
}

func TestRenderer_RenderSecurity(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "secure",
		Container: service.Container{
			Image: "alpine:latest",
			Security: service.Security{
				Privileged:  true,
				CapAdd:      []string{"NET_ADMIN", "SYS_TIME"},
				CapDrop:     []string{"ALL"},
				SecurityOpt: []string{"label=disable"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	content := string(result.Artifacts[0].Content)
	assert.Contains(t, content, "PodmanArgs=--privileged")
	assert.Contains(t, content, "PodmanArgs=--cap-add=NET_ADMIN")
	assert.Contains(t, content, "PodmanArgs=--cap-add=SYS_TIME")
	assert.Contains(t, content, "PodmanArgs=--cap-drop=ALL")
	assert.Contains(t, content, "PodmanArgs=--security-opt=label=disable")
}

func TestRenderer_MultipleServices(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	specs := []service.Spec{
		{
			Name: "web",
			Container: service.Container{
				Image: "nginx:latest",
			},
		},
		{
			Name: "db",
			Container: service.Container{
				Image: "postgres:latest",
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, specs)
	require.NoError(t, err)

	assert.Len(t, result.Artifacts, 2)
	assert.Len(t, result.ServiceChanges, 2)
	assert.Contains(t, result.ServiceChanges, "web")
	assert.Contains(t, result.ServiceChanges, "db")
}

func TestRenderer_ExternalResources(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "app",
		Volumes: []service.Volume{
			{Name: "data", External: false},
			{Name: "cache", External: true},
		},
		Networks: []service.Network{
			{Name: "internal", External: false},
			{Name: "public", External: true},
		},
		Container: service.Container{
			Image: "alpine:latest",
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	paths := make([]string, 0, len(result.Artifacts))
	for _, a := range result.Artifacts {
		paths = append(paths, a.Path)
	}

	assert.Contains(t, paths, "data.volume")
	assert.NotContains(t, paths, "cache.volume")
	assert.Contains(t, paths, "internal.network")
	assert.NotContains(t, paths, "public.network")
}

func TestRenderer_HashConsistency(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "test",
		Container: service.Container{
			Image: "alpine:latest",
		},
	}

	ctx := context.Background()
	result1, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	result2, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	assert.Equal(t, result1.Artifacts[0].Hash, result2.Artifacts[0].Hash)
	assert.Equal(t, result1.ServiceChanges["test"].ContentHash, result2.ServiceChanges["test"].ContentHash)
}
