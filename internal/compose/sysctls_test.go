package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecConverter_Sysctls(t *testing.T) {
	tests := []struct {
		name     string
		sysctls  map[string]string
		expected map[string]string
	}{
		{
			name: "single sysctl",
			sysctls: map[string]string{
				"net.ipv4.ip_forward": "1",
			},
			expected: map[string]string{
				"net.ipv4.ip_forward": "1",
			},
		},
		{
			name: "multiple sysctls",
			sysctls: map[string]string{
				"net.ipv4.ip_forward": "1",
				"net.core.somaxconn":  "1024",
			},
			expected: map[string]string{
				"net.ipv4.ip_forward": "1",
				"net.core.somaxconn":  "1024",
			},
		},
		{
			name: "kernel parameters",
			sysctls: map[string]string{
				"kernel.shmmax":                "68719476736",
				"kernel.shmall":                "4294967296",
				"net.ipv4.tcp_keepalive_time":  "600",
				"net.ipv4.tcp_keepalive_intvl": "60",
				"net.ipv4.conf.all.rp_filter":  "2",
			},
			expected: map[string]string{
				"kernel.shmmax":                "68719476736",
				"kernel.shmall":                "4294967296",
				"net.ipv4.tcp_keepalive_time":  "600",
				"net.ipv4.tcp_keepalive_intvl": "60",
				"net.ipv4.conf.all.rp_filter":  "2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &types.Project{
				Name: "test",
				Services: types.Services{
					"app": {
						Name:    "app",
						Image:   "nginx:alpine",
						Sysctls: tt.sysctls,
					},
				},
			}

			converter := NewSpecConverter("/tmp")
			specs, err := converter.ConvertProject(project)
			require.NoError(t, err)
			require.Len(t, specs, 1)

			// Verify sysctls are properly converted
			assert.Equal(t, tt.expected, specs[0].Container.Sysctls,
				"Sysctls should be converted correctly from Docker Compose")
		})
	}
}

func TestSpecConverter_NoSysctls(t *testing.T) {
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"app": {
				Name:  "app",
				Image: "nginx:alpine",
			},
		},
	}

	converter := NewSpecConverter("/tmp")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	// Verify sysctls is nil when not specified
	assert.Nil(t, specs[0].Container.Sysctls,
		"Sysctls should be nil when not specified in Docker Compose")
}

func TestSpecConverter_EmptySysctls(t *testing.T) {
	project := &types.Project{
		Name: "test",
		Services: types.Services{
			"app": {
				Name:    "app",
				Image:   "nginx:alpine",
				Sysctls: map[string]string{},
			},
		},
	}

	converter := NewSpecConverter("/tmp")
	specs, err := converter.ConvertProject(project)
	require.NoError(t, err)
	require.Len(t, specs, 1)

	// Verify empty sysctls map is preserved
	assert.NotNil(t, specs[0].Container.Sysctls,
		"Empty sysctls map should be preserved")
	assert.Empty(t, specs[0].Container.Sysctls,
		"Sysctls map should be empty")
}
