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

// TestNetworkOnlineTarget_ContainerWithNetworks verifies that containers
// with networks get network-online.target dependencies.
func TestNetworkOnlineTarget_ContainerWithNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "web",
		Description: "Web server with networks",
		Networks: []service.Network{
			{
				Name:   "backend",
				Driver: "bridge",
			},
		},
		Container: service.Container{
			Image: "nginx:latest",
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"backend"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the container artifact
	var containerContent string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			containerContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, containerContent, "Container artifact not found")

	// CRITICAL: Container with networks must wait for network to be online
	assert.Contains(t, containerContent, "After=network-online.target",
		"Containers with networks must include After=network-online.target")
	assert.Contains(t, containerContent, "Wants=network-online.target",
		"Containers with networks must include Wants=network-online.target")
}

// TestNetworkOnlineTarget_ContainerWithPublishedPorts verifies that containers
// with published ports get network-online.target dependencies.
func TestNetworkOnlineTarget_ContainerWithPublishedPorts(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "web",
		Description: "Web server with published ports",
		Container: service.Container{
			Image: "nginx:latest",
			Ports: []service.Port{
				{HostPort: 8080, Container: 80, Protocol: "tcp"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// CRITICAL: Container with published ports must wait for network to be online
	assert.Contains(t, containerContent, "After=network-online.target",
		"Containers with published ports must include After=network-online.target")
	assert.Contains(t, containerContent, "Wants=network-online.target",
		"Containers with published ports must include Wants=network-online.target")
}

// TestNetworkOnlineTarget_ContainerWithMultipleNetworks verifies that containers
// with multiple networks still get a single network-online.target dependency.
func TestNetworkOnlineTarget_ContainerWithMultipleNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App with multiple networks",
		Networks: []service.Network{
			{Name: "frontend", Driver: "bridge"},
			{Name: "backend", Driver: "bridge"},
		},
		Container: service.Container{
			Image: "app:1.0",
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"frontend", "backend"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the container artifact
	var containerContent string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			containerContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, containerContent)

	// Should have network-online.target dependency
	assert.Contains(t, containerContent, "After=network-online.target")
	assert.Contains(t, containerContent, "Wants=network-online.target")

	// Verify it appears only once
	afterCount := strings.Count(containerContent, "After=network-online.target")
	wantsCount := strings.Count(containerContent, "Wants=network-online.target")
	assert.Equal(t, 1, afterCount, "Should have exactly one After=network-online.target")
	assert.Equal(t, 1, wantsCount, "Should have exactly one Wants=network-online.target")
}

// TestNetworkOnlineTarget_ContainerWithNetworksAndPorts verifies that containers
// with both networks and published ports get network-online.target dependency.
func TestNetworkOnlineTarget_ContainerWithNetworksAndPorts(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "web",
		Description: "Web server with networks and ports",
		Networks: []service.Network{
			{Name: "frontend", Driver: "bridge"},
		},
		Container: service.Container{
			Image: "nginx:latest",
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"frontend"},
			},
			Ports: []service.Port{
				{HostPort: 8080, Container: 80, Protocol: "tcp"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the container artifact
	var containerContent string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			containerContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, containerContent)

	// Should have network-online.target dependency
	assert.Contains(t, containerContent, "After=network-online.target")
	assert.Contains(t, containerContent, "Wants=network-online.target")
}

// TestNetworkOnlineTarget_ContainerWithoutNetworksOrPorts verifies that containers
// without networks or published ports do NOT get network-online.target dependency.
func TestNetworkOnlineTarget_ContainerWithoutNetworksOrPorts(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "worker",
		Description: "Background worker without network needs",
		Container: service.Container{
			Image: "worker:1.0",
			// No networks, no ports
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Should NOT have network-online.target dependency
	assert.NotContains(t, containerContent, "network-online.target",
		"Containers without networks or ports should not depend on network-online.target")
}

// TestNetworkOnlineTarget_ContainerWithHostNetwork verifies that containers
// using host network mode get network-online.target dependency.
func TestNetworkOnlineTarget_ContainerWithHostNetwork(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App using host network",
		Container: service.Container{
			Image: "app:1.0",
			Network: service.NetworkMode{
				Mode: "host",
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	containerContent := string(result.Artifacts[0].Content)

	// Host network mode still needs network to be online
	assert.Contains(t, containerContent, "After=network-online.target")
	assert.Contains(t, containerContent, "Wants=network-online.target")
}

// TestNetworkOnlineTarget_ContainerWithExternalNetworks verifies that containers
// using external networks get network-online.target dependency.
func TestNetworkOnlineTarget_ContainerWithExternalNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "app",
		Description: "App using external network",
		Networks: []service.Network{
			{
				Name:     "infrastructure-proxy",
				Driver:   "bridge",
				External: true,
			},
		},
		Container: service.Container{
			Image: "app:1.0",
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"infrastructure-proxy"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the container artifact
	var containerContent string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			containerContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, containerContent)

	// External networks still require network to be online
	assert.Contains(t, containerContent, "After=network-online.target")
	assert.Contains(t, containerContent, "Wants=network-online.target")
}

// TestNetworkOnlineTarget_InitContainer verifies that init containers
// with networks get network-online.target dependency.
func TestNetworkOnlineTarget_InitContainer(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "web-init-0",
		Description: "Init container with network",
		Networks: []service.Network{
			{Name: "backend", Driver: "bridge"},
		},
		Container: service.Container{
			Image: "busybox:latest",
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"backend"},
			},
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the container artifact
	var containerContent string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			containerContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, containerContent)

	// Init containers with networks also need network-online.target
	assert.Contains(t, containerContent, "After=network-online.target")
	assert.Contains(t, containerContent, "Wants=network-online.target")
}

// TestNetworkOnlineTarget_Ordering verifies that network-online.target appears
// before other After directives in the [Unit] section for clarity.
func TestNetworkOnlineTarget_Ordering(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "web",
		Description: "Web server",
		Networks: []service.Network{
			{Name: "backend", Driver: "bridge"},
		},
		Container: service.Container{
			Image: "nginx:latest",
			Network: service.NetworkMode{
				Mode:            "bridge",
				ServiceNetworks: []string{"backend"},
			},
		},
		DependsOn: []string{"db"},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Find the container artifact
	var containerContent string
	for _, a := range result.Artifacts {
		if strings.HasSuffix(a.Path, ".container") {
			containerContent = string(a.Content)
			break
		}
	}

	require.NotEmpty(t, containerContent)

	// Extract [Unit] section
	unitStart := strings.Index(containerContent, "[Unit]")
	containerStart := strings.Index(containerContent, "[Container]")
	require.Greater(t, containerStart, unitStart)

	unitSection := containerContent[unitStart:containerStart]

	// network-online.target should appear in [Unit] section
	assert.Contains(t, unitSection, "After=network-online.target")
	assert.Contains(t, unitSection, "Wants=network-online.target")

	// Both should appear before service dependencies
	networkOnlinePos := strings.Index(unitSection, "network-online.target")
	dbServicePos := strings.Index(unitSection, "db.service")
	if dbServicePos > 0 {
		assert.Less(t, networkOnlinePos, dbServicePos,
			"network-online.target should appear before service dependencies for clarity")
	}
}
