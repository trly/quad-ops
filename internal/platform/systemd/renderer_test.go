package systemd

import (
	"context"
	"fmt"
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

func TestRenderer_RenderInitContainer(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "web-init-0",
		Description: "Init container 0 for service web",
		Container: service.Container{
			Image:         "busybox:latest",
			Command:       []string{"sh", "-c", "echo 'init'"},
			RestartPolicy: service.RestartPolicyNo,
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.Artifacts, 1)
	assert.Equal(t, "web-init-0.container", result.Artifacts[0].Path)
	assert.NotEmpty(t, result.Artifacts[0].Hash)

	content := string(result.Artifacts[0].Content)
	assert.Contains(t, content, "[Unit]")
	assert.Contains(t, content, "Description=Init container 0 for service web")
	assert.Contains(t, content, "[Container]")
	assert.Contains(t, content, "Image=busybox:latest")
	assert.Contains(t, content, "Exec=sh -c echo 'init'")
	assert.Contains(t, content, "[Service]")
	assert.Contains(t, content, "Type=oneshot")
	assert.Contains(t, content, "RemainAfterExit=yes")
	assert.Contains(t, content, "Restart=no")
	assert.Contains(t, content, "[Install]")
	assert.Contains(t, content, "WantedBy=default.target")
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

	var volumeArtifact, containerArtifact *string
	for _, a := range result.Artifacts {
		switch a.Path {
		case "data.volume":
			content := string(a.Content)
			volumeArtifact = &content
		case "app.container":
			content := string(a.Content)
			containerArtifact = &content
		}
	}

	require.NotNil(t, volumeArtifact)
	assert.Contains(t, *volumeArtifact, "[Unit]")
	assert.Contains(t, *volumeArtifact, "[Volume]")
	assert.Contains(t, *volumeArtifact, "VolumeName=data")
	assert.Contains(t, *volumeArtifact, "Options=device=tmpfs")
	assert.Contains(t, *volumeArtifact, "Options=type=tmpfs")
	assert.Contains(t, *volumeArtifact, "Label=env=test")
	assert.Contains(t, *volumeArtifact, "[Install]")
	assert.Contains(t, *volumeArtifact, "WantedBy=default.target")

	require.NotNil(t, containerArtifact)
	assert.Contains(t, *containerArtifact, "After=data.volume")
	assert.Contains(t, *containerArtifact, "Requires=data.volume")
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
			// Container explicitly declares it uses the backend network
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"backend"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	assert.Len(t, result.Artifacts, 2)

	var networkArtifact, containerArtifact *string
	for _, a := range result.Artifacts {
		switch a.Path {
		case "backend.network":
			content := string(a.Content)
			networkArtifact = &content
		case "app.container":
			content := string(a.Content)
			containerArtifact = &content
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
	assert.Contains(t, *networkArtifact, "[Install]")
	assert.Contains(t, *networkArtifact, "WantedBy=default.target")

	require.NotNil(t, containerArtifact)
	assert.Contains(t, *containerArtifact, "After=backend.network")
	assert.Contains(t, *containerArtifact, "Requires=backend.network")
	assert.Contains(t, *containerArtifact, "Network=backend.network")
}

func TestRenderer_MultipleNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	// Test case: service with multiple networks (like immich example)
	spec := service.Spec{
		Name:        "immich-server",
		Description: "Immich server with multiple networks",
		Networks: []service.Network{
			{
				Name:     "default",
				Driver:   "bridge",
				External: false,
			},
			{
				Name:     "infrastructure-proxy",
				Driver:   "bridge",
				External: false,
			},
		},
		Container: service.Container{
			Image: "immich-server:latest",
			// Container explicitly uses both networks
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"default", "infrastructure-proxy"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	assert.Len(t, result.Artifacts, 3) // 2 network units + 1 container unit

	var containerArtifact *string
	networkPaths := []string{}

	for _, a := range result.Artifacts {
		switch {
		case strings.HasSuffix(a.Path, ".container"):
			content := string(a.Content)
			containerArtifact = &content
		case strings.HasSuffix(a.Path, ".network"):
			networkPaths = append(networkPaths, a.Path)
		}
	}

	require.NotNil(t, containerArtifact)

	// Verify container depends on both networks
	assert.Contains(t, *containerArtifact, "After=default.network")
	assert.Contains(t, *containerArtifact, "After=infrastructure-proxy.network")
	assert.Contains(t, *containerArtifact, "Requires=default.network")
	assert.Contains(t, *containerArtifact, "Requires=infrastructure-proxy.network")

	// Verify container joins both networks
	assert.Contains(t, *containerArtifact, "Network=default.network")
	assert.Contains(t, *containerArtifact, "Network=infrastructure-proxy.network")

	// Verify both network units were created
	assert.Len(t, networkPaths, 2)
	assert.Contains(t, networkPaths, "default.network")
	assert.Contains(t, networkPaths, "infrastructure-proxy.network")
}

func TestRenderer_ServiceDependencyWithUnitTypeSuffixes(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	tests := []struct {
		name        string
		depName     string
		expectedDep string
		notExpected string
	}{
		{
			name:        "dependency with .network suffix",
			depName:     "devops-infrastructure-proxy.network",
			expectedDep: "After=devops-infrastructure-proxy.network",
			notExpected: ".network.service",
		},
		{
			name:        "dependency with .volume suffix",
			depName:     "data.volume",
			expectedDep: "After=data.volume",
			notExpected: ".volume.service",
		},
		{
			name:        "dependency with .pod suffix",
			depName:     "my-pod.pod",
			expectedDep: "After=my-pod.pod",
			notExpected: ".pod.service",
		},
		{
			name:        "dependency with .service suffix",
			depName:     "db.service",
			expectedDep: "After=db.service",
			notExpected: ".service.service",
		},
		{
			name:        "normal service dependency (no suffix)",
			depName:     "db",
			expectedDep: "After=db.service",
			notExpected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := service.Spec{
				Name:        "web",
				Description: "Test service",
				Container: service.Container{
					Image: "test:latest",
				},
				DependsOn: []string{tt.depName},
			}

			ctx := context.Background()
			result, err := r.Render(ctx, []service.Spec{spec})
			require.NoError(t, err)

			content := string(result.Artifacts[0].Content)
			assert.Contains(t, content, tt.expectedDep, "should contain expected dependency directive")

			if tt.notExpected != "" {
				assert.NotContains(t, content, tt.notExpected, "should not contain malformed suffix")
			}
		})
	}
}

func TestRenderer_NetworkWithNetworkSuffix(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	// Test case: external network name that contains .network suffix
	// This reproduces the issue where names like "devops-infrastructure-proxy.network"
	// are used as network names (e.g., from external Docker Compose projects)
	spec := service.Spec{
		Name:        "web",
		Description: "Web service using external network with .network suffix",
		Networks: []service.Network{
			{
				Name:     "devops-infrastructure-proxy.network",
				Driver:   "bridge",
				External: true,
			},
		},
		Container: service.Container{
			Image: "nginx:latest",
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"devops-infrastructure-proxy.network"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	assert.Len(t, result.Artifacts, 1) // Only container unit (external network should not create a unit)

	var containerArtifact *string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			content := string(a.Content)
			containerArtifact = &content
		}
	}

	require.NotNil(t, containerArtifact)

	// The dependency should be After=devops-infrastructure-proxy.network (not .network.service)
	assert.Contains(t, *containerArtifact, "After=devops-infrastructure-proxy.network")
	assert.Contains(t, *containerArtifact, "Requires=devops-infrastructure-proxy.network")

	// The Network directive should also use the correct name
	assert.Contains(t, *containerArtifact, "Network=devops-infrastructure-proxy.network")

	// Should NOT contain the malformed .network.service suffix
	assert.NotContains(t, *containerArtifact, ".network.service")
}

func TestRenderer_ExternalNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app-external",
		Description: "App using external network",
		Networks: []service.Network{
			{
				Name:     "local-net",
				Driver:   "bridge",
				External: false,
			},
			{
				Name:     "external-net",
				Driver:   "bridge",
				External: true, // External network should not create unit file or add to container
			},
		},
		Container: service.Container{
			Image: "myapp:latest",
			// Container explicitly uses the local network
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"local-net"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	// Should only have 1 network unit (external is skipped) + 1 container unit
	assert.Len(t, result.Artifacts, 2)

	var containerArtifact *string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			content := string(a.Content)
			containerArtifact = &content
		}
		// Verify no external network unit was created
		assert.NotEqual(t, "external-net.network", a.Path)
	}

	require.NotNil(t, containerArtifact)

	// Container should depend on and join only the local network
	assert.Contains(t, *containerArtifact, "After=local-net.network")
	assert.Contains(t, *containerArtifact, "Requires=local-net.network")
	assert.Contains(t, *containerArtifact, "Network=local-net.network")

	// Should NOT have external network directives
	assert.NotContains(t, *containerArtifact, "After=external-net.network")
	assert.NotContains(t, *containerArtifact, "Network=external-net.network")
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

	var buildArtifact, containerArtifact *string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".build") {
			content := string(a.Content)
			buildArtifact = &content
		} else if strings.HasSuffix(a.Path, ".container") {
			content := string(a.Content)
			containerArtifact = &content
		}
	}

	require.NotNil(t, buildArtifact)
	assert.Contains(t, *buildArtifact, "[Unit]")
	assert.Contains(t, *buildArtifact, "WorkingDirectory=./app")
	assert.Contains(t, *buildArtifact, "[Build]")
	assert.Contains(t, *buildArtifact, "ImageTag=myapp:latest")
	assert.Contains(t, *buildArtifact, "ImageTag=myapp:v1.0")
	assert.Contains(t, *buildArtifact, "File=Dockerfile.prod")
	assert.Contains(t, *buildArtifact, "Target=production")
	assert.Contains(t, *buildArtifact, "Pull=always")
	assert.Contains(t, *buildArtifact, "SetWorkingDirectory=/build")
	assert.Contains(t, *buildArtifact, "Environment=VERSION=1.0")
	assert.Contains(t, *buildArtifact, "Label=version=1.0")

	require.NotNil(t, containerArtifact)
	assert.Contains(t, *containerArtifact, "After=builder-build.service")
	assert.Contains(t, *containerArtifact, "Requires=builder-build.service")
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

func TestRenderer_QuadletVolumeExtensions(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "app",
		Volumes: []service.Volume{
			{
				Name:   "data",
				Driver: "local",
				Quadlet: &service.QuadletVolume{
					ContainersConfModule: []string{"/etc/containers/storage.conf"},
					GlobalArgs:           []string{"--log-level=debug"},
					PodmanArgs:           []string{"--opt=type=tmpfs"},
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

	var volumeContent string
	for _, a := range result.Artifacts {
		if a.Path == "data.volume" {
			volumeContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, volumeContent)
	assert.Contains(t, volumeContent, "ContainersConfModule=/etc/containers/storage.conf")
	assert.Contains(t, volumeContent, "GlobalArgs=--log-level=debug")
	assert.Contains(t, volumeContent, "PodmanArgs=--opt=type=tmpfs")
}

func TestRenderer_QuadletNetworkExtensions(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "app",
		Networks: []service.Network{
			{
				Name:   "backend",
				Driver: "bridge",
				Quadlet: &service.QuadletNetwork{
					DisableDNS:           true,
					DNS:                  []string{"8.8.8.8", "8.8.4.4"},
					ContainersConfModule: []string{"/etc/containers/network.conf"},
					PodmanArgs:           []string{"--dns-search=example.com"},
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

	var networkContent string
	for _, a := range result.Artifacts {
		if a.Path == "backend.network" {
			networkContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, networkContent)
	assert.Contains(t, networkContent, "DisableDNS=yes")
	assert.Contains(t, networkContent, "DNS=8.8.4.4")
	assert.Contains(t, networkContent, "DNS=8.8.8.8")
	assert.Contains(t, networkContent, "ContainersConfModule=/etc/containers/network.conf")
	assert.Contains(t, networkContent, "PodmanArgs=--dns-search=example.com")
}

// TestRenderer_ContainerNetworksDependOnlyOnUsedNetworks verifies that containers only
// declare dependencies on networks they actually use (via ServiceNetworks), not on all
// project-level networks. This prevents cross-project network reference errors.
func TestRenderer_ContainerNetworksDependOnlyOnUsedNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	// Simulate a container in one project (media) that doesn't explicitly use
	// any networks (like a case where ServiceNetworks is empty)
	spec := service.Spec{
		Name:        "media-immich-server",
		Description: "Immich server",
		// Project has multiple networks defined
		Networks: []service.Network{
			{Name: "media_default", Driver: "bridge", External: false},
			{Name: "media_proxy", Driver: "bridge", External: false},
		},
		Container: service.Container{
			Image: "immich-server:latest",
			// Container has NO ServiceNetworks (empty)
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{}, // Empty!
			},
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	// Should have 2 network units + 1 container unit
	assert.Len(t, result.Artifacts, 3)

	// Find the container artifact
	var containerContent string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			containerContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, containerContent)

	// The container should NOT have Network= directives since ServiceNetworks is empty
	// It will use the default network implicitly
	assert.NotContains(t, containerContent, "Network=media_default.network")
	assert.NotContains(t, containerContent, "Network=media_proxy.network")

	// The container should NOT depend on any networks since ServiceNetworks is empty
	assert.NotContains(t, containerContent, "After=media_default.network")
	assert.NotContains(t, containerContent, "After=media_proxy.network")
	assert.NotContains(t, containerContent, "Requires=media_default.network")
	assert.NotContains(t, containerContent, "Requires=media_proxy.network")
}

// TestRenderer_ContainerNetworksDependOnExplicitNetworks verifies that containers
// declare dependencies only on networks they explicitly declare via ServiceNetworks.
func TestRenderer_ContainerNetworksDependOnExplicitNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	// Container explicitly declares which networks it uses
	spec := service.Spec{
		Name:        "media-immich-server",
		Description: "Immich server with explicit networks",
		// Project has multiple networks
		Networks: []service.Network{
			{Name: "media_default", Driver: "bridge", External: false},
			{Name: "media_cache", Driver: "bridge", External: false},
			{Name: "infrastructure_proxy", Driver: "bridge", External: false},
		},
		Container: service.Container{
			Image: "immich-server:latest",
			// Container explicitly uses only media_default and infrastructure_proxy
			Network: service.NetworkMode{
				Mode: "bridge",
				ServiceNetworks: []string{
					"media_default",
					"infrastructure_proxy",
				},
			},
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	// Should have 3 network units + 1 container unit
	assert.Len(t, result.Artifacts, 4)

	// Find the container artifact
	var containerContent string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			containerContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, containerContent)

	// The container should have Network= directives only for its explicit networks
	assert.Contains(t, containerContent, "Network=media_default.network")
	assert.Contains(t, containerContent, "Network=infrastructure_proxy.network")
	// But NOT for media_cache which it doesn't use
	assert.NotContains(t, containerContent, "Network=media_cache.network")

	// The container should depend only on its explicit networks
	assert.Contains(t, containerContent, "After=media_default.network")
	assert.Contains(t, containerContent, "After=infrastructure_proxy.network")
	assert.Contains(t, containerContent, "Requires=media_default.network")
	assert.Contains(t, containerContent, "Requires=infrastructure_proxy.network")
	// But NOT for media_cache
	assert.NotContains(t, containerContent, "After=media_cache.network")
	assert.NotContains(t, containerContent, "Requires=media_cache.network")
}

func TestRenderer_RenderMemoryConstraints(t *testing.T) {
	tests := []struct {
		name            string
		resources       service.Resources
		expectMemory    string // substring to check for, empty means not present
		expectMemSwap   string
		expectMemReserv string
	}{
		{
			name:         "memory only",
			resources:    service.Resources{Memory: "512m"},
			expectMemory: "Memory=512m",
		},
		{
			name:            "memory and reservation",
			resources:       service.Resources{Memory: "1g", MemoryReservation: "512m"},
			expectMemory:    "Memory=1g",
			expectMemReserv: "MemoryReservation=512m",
		},
		{
			name:          "memory and swap",
			resources:     service.Resources{Memory: "1g", MemorySwap: "2g"},
			expectMemory:  "Memory=1g",
			expectMemSwap: "MemorySwap=2g",
		},
		{
			name:            "all memory constraints",
			resources:       service.Resources{Memory: "1g", MemoryReservation: "512m", MemorySwap: "2g"},
			expectMemory:    "Memory=1g",
			expectMemReserv: "MemoryReservation=512m",
			expectMemSwap:   "MemorySwap=2g",
		},
		{
			name:      "empty values not rendered",
			resources: service.Resources{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.NewTestLogger(t)
			r := NewRenderer(logger)

			spec := service.Spec{
				Name: "app",
				Container: service.Container{
					Image:     "alpine:latest",
					Resources: tt.resources,
				},
			}

			ctx := context.Background()
			result, err := r.Render(ctx, []service.Spec{spec})
			require.NoError(t, err)

			content := string(result.Artifacts[0].Content)

			if tt.expectMemory != "" {
				assert.Contains(t, content, tt.expectMemory, "expected Memory directive")
			} else if tt.resources.Memory != "" {
				assert.NotContains(t, content, "Memory=", "Memory should not be rendered when empty")
			}

			if tt.expectMemReserv != "" {
				assert.Contains(t, content, tt.expectMemReserv, "expected MemoryReservation directive")
			}

			if tt.expectMemSwap != "" {
				assert.Contains(t, content, tt.expectMemSwap, "expected MemorySwap directive")
			}
		})
	}
}

func TestRenderer_RenderCPUConstraints(t *testing.T) {
	tests := []struct {
		name          string
		resources     service.Resources
		expectCPUArgs []string // substrings to check for in PodmanArgs
	}{
		{
			name:          "cpu shares only",
			resources:     service.Resources{CPUShares: 1024},
			expectCPUArgs: []string{"PodmanArgs=--cpu-shares 1024"},
		},
		{
			name:          "cpu quota only",
			resources:     service.Resources{CPUQuota: 100000},
			expectCPUArgs: []string{"PodmanArgs=--cpu-quota 100000"},
		},
		{
			name:          "cpu period only",
			resources:     service.Resources{CPUPeriod: 100000},
			expectCPUArgs: []string{"PodmanArgs=--cpu-period 100000"},
		},
		{
			name: "all cpu constraints",
			resources: service.Resources{
				CPUShares: 1024,
				CPUQuota:  100000,
				CPUPeriod: 100000,
			},
			expectCPUArgs: []string{
				"PodmanArgs=--cpu-period 100000",
				"PodmanArgs=--cpu-quota 100000",
				"PodmanArgs=--cpu-shares 1024",
			},
		},
		{
			name:          "zero cpu constraints not rendered",
			resources:     service.Resources{},
			expectCPUArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.NewTestLogger(t)
			r := NewRenderer(logger)

			spec := service.Spec{
				Name: "app",
				Container: service.Container{
					Image:     "alpine:latest",
					Resources: tt.resources,
				},
			}

			ctx := context.Background()
			result, err := r.Render(ctx, []service.Spec{spec})
			require.NoError(t, err)

			content := string(result.Artifacts[0].Content)

			for _, expectArg := range tt.expectCPUArgs {
				assert.Contains(t, content, expectArg, "expected CPU constraint PodmanArgs")
			}
		})
	}
}

func TestRenderer_RenderMixedConstraints(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "app",
		Container: service.Container{
			Image: "alpine:latest",
			Resources: service.Resources{
				Memory:            "2g",
				MemoryReservation: "1g",
				MemorySwap:        "4g",
				CPUShares:         2048,
				CPUQuota:          200000,
				CPUPeriod:         100000,
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	content := string(result.Artifacts[0].Content)

	// Memory constraints
	assert.Contains(t, content, "Memory=2g")
	assert.Contains(t, content, "MemoryReservation=1g")
	assert.Contains(t, content, "MemorySwap=4g")

	// CPU constraints via PodmanArgs
	assert.Contains(t, content, "PodmanArgs=--cpu-period 100000")
	assert.Contains(t, content, "PodmanArgs=--cpu-quota 200000")
	assert.Contains(t, content, "PodmanArgs=--cpu-shares 2048")
}

func TestRenderer_ZeroValuesNotRendered(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "app",
		Container: service.Container{
			Image: "alpine:latest",
			Resources: service.Resources{
				Memory:            "", // empty string
				MemoryReservation: "", // empty string
				MemorySwap:        "", // empty string
				CPUShares:         0,  // zero
				CPUQuota:          0,  // zero
				CPUPeriod:         0,  // zero
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	content := string(result.Artifacts[0].Content)

	// Ensure no memory directives are present
	assert.NotContains(t, content, "Memory=")
	assert.NotContains(t, content, "MemoryReservation=")
	assert.NotContains(t, content, "MemorySwap=")

	// Ensure no CPU directives are present
	assert.NotContains(t, content, "--cpu-shares")
	assert.NotContains(t, content, "--cpu-quota")
	assert.NotContains(t, content, "--cpu-period")
}

func TestRenderer_ResourcesWithPidsLimitAndUlimits(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "app",
		Container: service.Container{
			Image: "alpine:latest",
			Resources: service.Resources{
				Memory:    "1g",
				CPUShares: 1024,
				PidsLimit: 512,
			},
			Ulimits: []service.Ulimit{
				{Name: "nofile", Soft: 1024, Hard: 2048},
				{Name: "nproc", Soft: 256, Hard: 256},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	content := string(result.Artifacts[0].Content)

	// Memory constraint
	assert.Contains(t, content, "Memory=1g")

	// CPU constraint via PodmanArgs
	assert.Contains(t, content, "PodmanArgs=--cpu-shares 1024")

	// PidsLimit
	assert.Contains(t, content, "PidsLimit=512")

	// Ulimits
	assert.Contains(t, content, "Ulimit=nofile=1024:2048")
	assert.Contains(t, content, "Ulimit=nproc=256")
}

func TestRenderer_NoDuplicatePidsLimit(t *testing.T) {
	tests := []struct {
		name           string
		resourcesLimit int64 // Container.Resources.PidsLimit
		containerLimit int64 // Container.PidsLimit
		expected       int64 // Expected rendered value
		shouldBeOnce   bool  // Should appear only once in output
	}{
		{
			name:           "only Resources.PidsLimit",
			resourcesLimit: 512,
			containerLimit: 0,
			expected:       512,
			shouldBeOnce:   true,
		},
		{
			name:           "Resources.PidsLimit takes precedence",
			resourcesLimit: 512,
			containerLimit: 1024,
			expected:       512,
			shouldBeOnce:   true,
		},
		{
			name:           "no PidsLimit when both zero",
			resourcesLimit: 0,
			containerLimit: 0,
			expected:       0,
			shouldBeOnce:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.NewTestLogger(t)
			r := NewRenderer(logger)

			spec := service.Spec{
				Name: "app",
				Container: service.Container{
					Image: "alpine:latest",
					Resources: service.Resources{
						PidsLimit: tt.resourcesLimit,
					},
					PidsLimit: tt.containerLimit,
				},
			}

			ctx := context.Background()
			result, err := r.Render(ctx, []service.Spec{spec})
			require.NoError(t, err)

			content := string(result.Artifacts[0].Content)

			if tt.expected > 0 {
				expectedStr := fmt.Sprintf("PidsLimit=%d", tt.expected)
				assert.Contains(t, content, expectedStr)

				// Count occurrences of PidsLimit directive
				count := strings.Count(content, "PidsLimit=")
				if tt.shouldBeOnce {
					assert.Equal(t, 1, count, "PidsLimit should appear exactly once")
				}
			} else {
				assert.NotContains(t, content, "PidsLimit=")
			}
		})
	}
}

func TestRenderer_TimeoutStartSecDefault(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "web",
		Description: "Web server",
		Container: service.Container{
			Image:         "nginx:latest",
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	content := string(result.Artifacts[0].Content)

	// Verify TimeoutStartSec is set to 900 seconds (15 minutes)
	assert.Contains(t, content, "TimeoutStartSec=900")

	// Verify it's in the [Service] section
	serviceIdx := strings.Index(content, "[Service]")
	installIdx := strings.Index(content, "[Install]")
	assert.Greater(t, serviceIdx, -1)
	assert.Greater(t, installIdx, -1)

	timeoutIdx := strings.Index(content, "TimeoutStartSec=900")
	assert.Greater(t, timeoutIdx, serviceIdx, "TimeoutStartSec should be in [Service] section")
	assert.Less(t, timeoutIdx, installIdx, "TimeoutStartSec should be before [Install] section")
}

func TestRenderer_RenderExtraHosts(t *testing.T) {
	tests := []struct {
		name           string
		extraHosts     []string
		expectContains []string
	}{
		{
			name:       "single extra host",
			extraHosts: []string{"database:192.168.1.10"},
			expectContains: []string{
				"AddHost=database:192.168.1.10",
			},
		},
		{
			name: "multiple extra hosts",
			extraHosts: []string{
				"api:10.0.0.5",
				"cache:192.168.1.20",
				"database:192.168.1.10",
				"localhost:127.0.0.1",
			},
			expectContains: []string{
				"AddHost=api:10.0.0.5",
				"AddHost=cache:192.168.1.20",
				"AddHost=database:192.168.1.10",
				"AddHost=localhost:127.0.0.1",
			},
		},
		{
			name: "ipv6 extra hosts",
			extraHosts: []string{
				"ipv6host2:2001:db8::1",
				"ipv6host:::1",
			},
			expectContains: []string{
				"AddHost=ipv6host2:2001:db8::1",
				"AddHost=ipv6host:::1",
			},
		},
		{
			name: "hostname with multiple IPs",
			extraHosts: []string{
				"multihost:192.168.1.10",
				"multihost:192.168.1.11",
				"multihost:192.168.1.12",
			},
			expectContains: []string{
				"AddHost=multihost:192.168.1.10",
				"AddHost=multihost:192.168.1.11",
				"AddHost=multihost:192.168.1.12",
			},
		},
		{
			name: "special characters in hostname",
			extraHosts: []string{
				"host-with-dash:192.168.1.10",
				"host.with.dots:192.168.1.20",
			},
			expectContains: []string{
				"AddHost=host-with-dash:192.168.1.10",
				"AddHost=host.with.dots:192.168.1.20",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.NewTestLogger(t)
			r := NewRenderer(logger)

			spec := service.Spec{
				Name: "web",
				Container: service.Container{
					Image:      "nginx:latest",
					ExtraHosts: tt.extraHosts,
				},
			}

			ctx := context.Background()
			result, err := r.Render(ctx, []service.Spec{spec})
			require.NoError(t, err)

			content := string(result.Artifacts[0].Content)

			// Verify all expected AddHost directives are present
			for _, expected := range tt.expectContains {
				assert.Contains(t, content, expected)
			}

			// Verify AddHost directives are in [Container] section
			containerIdx := strings.Index(content, "[Container]")
			serviceIdx := strings.Index(content, "[Service]")
			assert.Greater(t, containerIdx, -1)
			assert.Greater(t, serviceIdx, -1)

			for _, expected := range tt.expectContains {
				idx := strings.Index(content, expected)
				assert.Greater(t, idx, containerIdx, "%s should be in [Container] section", expected)
				assert.Less(t, idx, serviceIdx, "%s should be before [Service] section", expected)
			}
		})
	}
}

func TestRenderer_NoExtraHosts(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "web",
		Container: service.Container{
			Image:      "nginx:latest",
			ExtraHosts: nil,
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	content := string(result.Artifacts[0].Content)

	// Verify no AddHost directives are present
	assert.NotContains(t, content, "AddHost=")
}

func TestRenderer_EmptyExtraHosts(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name: "web",
		Container: service.Container{
			Image:      "nginx:latest",
			ExtraHosts: []string{},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)

	content := string(result.Artifacts[0].Content)

	// Verify no AddHost directives are present
	assert.NotContains(t, content, "AddHost=")
}
