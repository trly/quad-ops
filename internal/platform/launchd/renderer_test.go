//go:build darwin

package launchd

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

func TestRenderer_Render(t *testing.T) {
	tests := []struct {
		name     string
		spec     service.Spec
		opts     Options
		validate func(t *testing.T, content string)
	}{
		{
			name: "basic service",
			spec: service.Spec{
				Name:        "test-service",
				Description: "Test service",
				Container: service.Container{
					Image:         "docker.io/library/nginx:latest",
					RestartPolicy: service.RestartPolicyAlways,
				},
			},
			opts: testOptions(),
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "<key>Label</key>")
				assert.Contains(t, content, "<string>dev.trly.quad-ops.test-service</string>")
				assert.Contains(t, content, "<key>ProgramArguments</key>")
				assert.Contains(t, content, "<string>run</string>")
				assert.Contains(t, content, "<string>--rm</string>")
				assert.Contains(t, content, "<string>docker.io/library/nginx:latest</string>")
				assert.Contains(t, content, "<key>RunAtLoad</key>")
				assert.Contains(t, content, "<true/>")
				assert.Contains(t, content, "<key>KeepAlive</key>")
			},
		},
		{
			name: "service with environment variables",
			spec: service.Spec{
				Name: "env-service",
				Container: service.Container{
					Image: "docker.io/library/redis:latest",
					Env: map[string]string{
						"REDIS_PASSWORD": "secret123",
						"REDIS_PORT":     "6379",
					},
					RestartPolicy: service.RestartPolicyAlways,
				},
			},
			opts: testOptions(),
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "<string>-e</string>")
				assert.Contains(t, content, "REDIS_PASSWORD=secret123")
				assert.Contains(t, content, "REDIS_PORT=6379")
			},
		},
		{
			name: "service with ports",
			spec: service.Spec{
				Name: "web-service",
				Container: service.Container{
					Image: "docker.io/library/nginx:latest",
					Ports: []service.Port{
						{HostPort: 8080, Container: 80, Protocol: "tcp"},
						{Host: "127.0.0.1", HostPort: 8443, Container: 443, Protocol: "tcp"},
					},
					RestartPolicy: service.RestartPolicyAlways,
				},
			},
			opts: testOptions(),
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "<string>-p</string>")
				assert.Contains(t, content, "<string>8080:80/tcp</string>")
				assert.Contains(t, content, "<string>127.0.0.1:8443:443/tcp</string>")
			},
		},
		{
			name: "service with volumes",
			spec: service.Spec{
				Name: "data-service",
				Container: service.Container{
					Image: "docker.io/library/postgres:latest",
					Mounts: []service.Mount{
						{
							Source:   "/var/lib/postgres/data",
							Target:   "/var/lib/postgresql/data",
							Type:     service.MountTypeBind,
							ReadOnly: false,
						},
						{
							Source:   "/etc/config",
							Target:   "/app/config",
							Type:     service.MountTypeBind,
							ReadOnly: true,
						},
					},
					RestartPolicy: service.RestartPolicyAlways,
				},
			},
			opts: testOptions(),
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "<string>-v</string>")
				assert.Contains(t, content, "/var/lib/postgres/data:/var/lib/postgresql/data")
				assert.Contains(t, content, "/etc/config:/app/config:ro")
			},
		},
		{
			name: "restart policy - on-failure",
			spec: service.Spec{
				Name: "retry-service",
				Container: service.Container{
					Image:         "docker.io/library/busybox:latest",
					RestartPolicy: service.RestartPolicyOnFailure,
				},
			},
			opts: testOptions(),
			validate: func(t *testing.T, content string) {
				assert.Contains(t, content, "<key>KeepAlive</key>")
				assert.Contains(t, content, "<key>SuccessfulExit</key>")
				assert.Contains(t, content, "<false/>")
			},
		},
		{
			name: "restart policy - no",
			spec: service.Spec{
				Name: "oneshot-service",
				Container: service.Container{
					Image:         "docker.io/library/busybox:latest",
					RestartPolicy: service.RestartPolicyNo,
				},
			},
			opts: testOptions(),
			validate: func(t *testing.T, content string) {
				// KeepAlive should be false
				lines := strings.Split(content, "\n")
				for i, line := range lines {
					if strings.Contains(line, "<key>KeepAlive</key>") {
						// Next line should be <false/>
						if i+1 < len(lines) {
							assert.Contains(t, lines[i+1], "<false/>")
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.NewTestLogger(t)
			renderer, err := NewRenderer(tt.opts, logger)
			require.NoError(t, err)

			result, err := renderer.Render(context.Background(), []service.Spec{tt.spec})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Artifacts, 1)

			artifact := result.Artifacts[0]
			content := string(artifact.Content)

			// Validate it's valid XML plist structure
			assert.Contains(t, content, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
			assert.Contains(t, content, "<!DOCTYPE plist")
			assert.Contains(t, content, "<plist version=\"1.0\">")
			assert.Contains(t, content, "</plist>")

			// Ensure Program key is not present when ProgramArguments is used
			assert.NotContains(t, content, "<key>Program</key>", "Program key should not be present when ProgramArguments is used")
			assert.Contains(t, content, "<key>ProgramArguments</key>", "ProgramArguments key should be present")

			// Run test-specific validations
			tt.validate(t, content)
		})
	}
}

func TestRenderer_Name(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	renderer, err := NewRenderer(testOptions(), logger)
	require.NoError(t, err)

	assert.Equal(t, "launchd", renderer.Name())
}

func TestBuildLabel(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		service   string
		wantLabel string
	}{
		{
			name:      "simple service",
			prefix:    "dev.trly.quad-ops",
			service:   "nginx",
			wantLabel: "dev.trly.quad-ops.nginx",
		},
		{
			name:      "service with dashes and underscores",
			prefix:    "dev.trly.quad-ops",
			service:   "my-app_v2",
			wantLabel: "dev.trly.quad-ops.my-app_v2",
		},
		{
			name:      "service with periods",
			prefix:    "dev.trly.quad-ops",
			service:   "app.v1.test",
			wantLabel: "dev.trly.quad-ops.app.v1.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := testOptions()
			opts.LabelPrefix = tt.prefix

			label := opts.LabelFor(tt.service)
			assert.Equal(t, tt.wantLabel, label)
		})
	}
}

func TestRenderer_ServiceNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	renderer, err := NewRenderer(testOptions(), logger)
	require.NoError(t, err)

	// Test: Service with single service-level network
	spec := service.Spec{
		Name:        "web-app",
		Description: "Web app with single network",
		Networks: []service.Network{
			{
				Name:     "backend",
				Driver:   "bridge",
				External: false,
			},
		},
		Container: service.Container{
			Image:         "docker.io/library/nginx:latest",
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	result, err := renderer.Render(context.Background(), []service.Spec{spec})
	require.NoError(t, err)
	require.Len(t, result.Artifacts, 1)

	content := string(result.Artifacts[0].Content)

	// Verify --network flag is present for service-level network
	assert.Contains(t, content, "<string>--network</string>")
	assert.Contains(t, content, "<string>backend</string>")
}

func TestRenderer_MultipleServiceNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	renderer, err := NewRenderer(testOptions(), logger)
	require.NoError(t, err)

	// Test: Service with multiple service-level networks (like immich example)
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
			Image:         "docker.io/immich:latest",
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	result, err := renderer.Render(context.Background(), []service.Spec{spec})
	require.NoError(t, err)
	require.Len(t, result.Artifacts, 1)

	content := string(result.Artifacts[0].Content)

	// Verify --network flags are present for all service-level networks
	assert.Contains(t, content, "<string>--network</string>")

	// Both networks should be in the ProgramArguments
	assert.Contains(t, content, "<string>default</string>")
	assert.Contains(t, content, "<string>infrastructure-proxy</string>")

	// Verify network ordering is consistent (networks should be in sorted order)
	defaultIdx := strings.Index(content, "<string>default</string>")
	proxyIdx := strings.Index(content, "<string>infrastructure-proxy</string>")
	assert.True(t, defaultIdx < proxyIdx, "Networks should be in sorted order for determinism")
}

func TestRenderer_ExternalNetworksSkipped(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	renderer, err := NewRenderer(testOptions(), logger)
	require.NoError(t, err)

	// Test: Service with mix of internal and external networks
	spec := service.Spec{
		Name:        "app-external",
		Description: "App using mixed networks",
		Networks: []service.Network{
			{
				Name:     "local-net",
				Driver:   "bridge",
				External: false,
			},
			{
				Name:     "external-net",
				Driver:   "bridge",
				External: true, // Should be skipped
			},
		},
		Container: service.Container{
			Image:         "docker.io/library/myapp:latest",
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	result, err := renderer.Render(context.Background(), []service.Spec{spec})
	require.NoError(t, err)
	require.Len(t, result.Artifacts, 1)

	content := string(result.Artifacts[0].Content)

	// Verify local network is present
	assert.Contains(t, content, "<string>local-net</string>")

	// Verify external network is NOT added to the plist
	assert.NotContains(t, content, "<string>external-net</string>")
}

func TestRenderer_NoNetworks(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	renderer, err := NewRenderer(testOptions(), logger)
	require.NoError(t, err)

	// Test: Service with no service-level networks
	spec := service.Spec{
		Name:        "simple-service",
		Description: "Simple service without networks",
		Container: service.Container{
			Image:         "docker.io/library/nginx:latest",
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	result, err := renderer.Render(context.Background(), []service.Spec{spec})
	require.NoError(t, err)
	require.Len(t, result.Artifacts, 1)

	content := string(result.Artifacts[0].Content)

	// Should still have valid plist structure
	assert.Contains(t, content, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, content, "<plist version=\"1.0\">")
	assert.Contains(t, content, "</plist>")
}

func TestRenderer_NetworkOrderingConsistency(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	renderer, err := NewRenderer(testOptions(), logger)
	require.NoError(t, err)

	// Test: Multiple renders of the same spec should have consistent network ordering
	spec := service.Spec{
		Name:        "consistent-app",
		Description: "App to test network ordering consistency",
		Networks: []service.Network{
			{
				Name:     "zebra",
				Driver:   "bridge",
				External: false,
			},
			{
				Name:     "apple",
				Driver:   "bridge",
				External: false,
			},
			{
				Name:     "monkey",
				Driver:   "bridge",
				External: false,
			},
		},
		Container: service.Container{
			Image:         "docker.io/library/alpine:latest",
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	// Render multiple times and verify consistent ordering
	var results [3]string
	for i := 0; i < 3; i++ {
		result, err := renderer.Render(context.Background(), []service.Spec{spec})
		require.NoError(t, err)
		require.Len(t, result.Artifacts, 1)
		results[i] = string(result.Artifacts[0].Content)
	}

	// All three renders should be identical
	assert.Equal(t, results[0], results[1], "First and second render should be identical")
	assert.Equal(t, results[1], results[2], "Second and third render should be identical")

	// Verify networks appear in alphabetical order in the content
	appleIdx := strings.Index(results[0], "<string>apple</string>")
	monkeyIdx := strings.Index(results[0], "<string>monkey</string>")
	zebraIdx := strings.Index(results[0], "<string>zebra</string>")

	assert.True(t, appleIdx < monkeyIdx, "apple should come before monkey")
	assert.True(t, monkeyIdx < zebraIdx, "monkey should come before zebra")
}

func TestRenderer_ServiceDependencies(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	renderer, err := NewRenderer(testOptions(), logger)
	require.NoError(t, err)

	// Test: Service with dependencies
	spec := service.Spec{
		Name:        "web-app",
		Description: "Web app that depends on database and cache",
		DependsOn:   []string{"db", "redis"},
		Container: service.Container{
			Image:         "docker.io/library/nginx:latest",
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	result, err := renderer.Render(context.Background(), []service.Spec{spec})
	require.NoError(t, err)
	require.Len(t, result.Artifacts, 1)

	content := string(result.Artifacts[0].Content)

	// Verify DependsOn array is present in plist
	assert.Contains(t, content, "<key>DependsOn</key>")
	assert.Contains(t, content, "<array>")

	// Verify dependencies are converted to launchd labels
	// Dependencies should be prefixed with the label prefix
	assert.Contains(t, content, "<string>dev.trly.quad-ops.db</string>")
	assert.Contains(t, content, "<string>dev.trly.quad-ops.redis</string>")
}

func TestRenderer_NoDependencies(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	renderer, err := NewRenderer(testOptions(), logger)
	require.NoError(t, err)

	// Test: Service without dependencies should not have DependsOn key
	spec := service.Spec{
		Name:        "standalone-app",
		Description: "Standalone app with no dependencies",
		Container: service.Container{
			Image:         "docker.io/library/nginx:latest",
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	result, err := renderer.Render(context.Background(), []service.Spec{spec})
	require.NoError(t, err)
	require.Len(t, result.Artifacts, 1)

	content := string(result.Artifacts[0].Content)

	// Verify DependsOn array is not present
	assert.NotContains(t, content, "<key>DependsOn</key>")
}
