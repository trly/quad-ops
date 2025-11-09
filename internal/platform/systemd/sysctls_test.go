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

func TestRenderer_Sysctls(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	tests := []struct {
		name     string
		sysctls  map[string]string
		expected []string
	}{
		{
			name: "single sysctl",
			sysctls: map[string]string{
				"net.ipv4.ip_forward": "1",
			},
			expected: []string{
				"Sysctl=net.ipv4.ip_forward=1",
			},
		},
		{
			name: "multiple sysctls",
			sysctls: map[string]string{
				"net.ipv4.ip_forward": "1",
				"net.core.somaxconn":  "1024",
			},
			expected: []string{
				"Sysctl=net.core.somaxconn=1024",
				"Sysctl=net.ipv4.ip_forward=1",
			},
		},
		{
			name: "sysctls with various values",
			sysctls: map[string]string{
				"net.ipv4.ip_forward":          "1",
				"net.ipv4.conf.all.rp_filter":  "2",
				"kernel.shmmax":                "68719476736",
				"net.ipv4.tcp_keepalive_time":  "600",
				"net.ipv4.tcp_keepalive_intvl": "60",
			},
			expected: []string{
				"Sysctl=kernel.shmmax=68719476736",
				"Sysctl=net.ipv4.conf.all.rp_filter=2",
				"Sysctl=net.ipv4.ip_forward=1",
				"Sysctl=net.ipv4.tcp_keepalive_intvl=60",
				"Sysctl=net.ipv4.tcp_keepalive_time=600",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := service.Spec{
				Name:        "sysctl-test",
				Description: "Test container with sysctls",
				Container: service.Container{
					Image:         "nginx:alpine",
					Sysctls:       tt.sysctls,
					RestartPolicy: service.RestartPolicyAlways,
				},
			}

			ctx := context.Background()
			result, err := r.Render(ctx, []service.Spec{spec})
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Len(t, result.Artifacts, 1)
			content := string(result.Artifacts[0].Content)

			// Verify all expected sysctl directives are present
			for _, expectedLine := range tt.expected {
				assert.Contains(t, content, expectedLine,
					"Container unit should contain '%s'", expectedLine)
			}

			// Verify sysctls are sorted alphabetically
			if len(tt.expected) > 1 {
				lines := strings.Split(content, "\n")
				var sysctlLines []string
				for _, line := range lines {
					if strings.HasPrefix(line, "Sysctl=") {
						sysctlLines = append(sysctlLines, line)
					}
				}

				// Verify order matches expected (which is pre-sorted)
				for i, expected := range tt.expected {
					if i < len(sysctlLines) {
						assert.Equal(t, expected, sysctlLines[i],
							"Sysctl directives should be in alphabetical order")
					}
				}
			}
		})
	}
}

func TestRenderer_NoSysctls(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "no-sysctls",
		Description: "Test container without sysctls",
		Container: service.Container{
			Image:         "nginx:alpine",
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	content := string(result.Artifacts[0].Content)
	assert.NotContains(t, content, "Sysctl=",
		"Container unit should not contain Sysctl directive when no sysctls are specified")
}

func TestRenderer_SysctlsFormat(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	spec := service.Spec{
		Name:        "format-test",
		Description: "Test sysctl format",
		Container: service.Container{
			Image: "nginx:alpine",
			Sysctls: map[string]string{
				"net.ipv4.ip_forward": "1",
			},
			RestartPolicy: service.RestartPolicyAlways,
		},
	}

	ctx := context.Background()
	result, err := r.Render(ctx, []service.Spec{spec})
	require.NoError(t, err)
	require.NotNil(t, result)

	content := string(result.Artifacts[0].Content)

	// Verify format is exactly "Sysctl=key=value" (one per line)
	lines := strings.Split(content, "\n")
	sysctlCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "Sysctl=") {
			sysctlCount++
			// Verify format: Sysctl=key=value
			assert.Regexp(t, `^Sysctl=[a-z0-9_.]+=[a-z0-9_.]+$`, line,
				"Sysctl directive should follow format 'Sysctl=key=value'")
		}
	}
	assert.Equal(t, 1, sysctlCount, "Should have exactly one Sysctl directive")
}
