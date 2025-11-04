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
				assert.Contains(t, content, "<string>com.github.trly.test-service</string>")
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
			prefix:    "com.github.trly",
			service:   "nginx",
			wantLabel: "com.github.trly.nginx",
		},
		{
			name:      "service with special chars",
			prefix:    "com.github.trly",
			service:   "my-app_v2",
			wantLabel: "com.github.trly.my-app_v2",
		},
		{
			name:      "service with invalid chars",
			prefix:    "com.github.trly",
			service:   "app@v1#test",
			wantLabel: "com.github.trly.app-v1-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := testOptions()
			opts.LabelPrefix = tt.prefix

			logger := testutil.NewTestLogger(t)
			renderer, err := NewRenderer(opts, logger)
			require.NoError(t, err)

			label := renderer.buildLabel(tt.service)
			assert.Equal(t, tt.wantLabel, label)
		})
	}
}
