package podman

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/service"
)

func TestBuildPodmanArgs_Sysctls(t *testing.T) {
	tests := []struct {
		name          string
		spec          service.Spec
		containerName string
		wantContains  []string
	}{
		{
			name: "single sysctl",
			spec: service.Spec{
				Name: "single-sysctl",
				Container: service.Container{
					Image: "nginx:alpine",
					Sysctls: map[string]string{
						"net.ipv4.ip_forward": "1",
					},
				},
			},
			containerName: "single-sysctl-container",
			wantContains: []string{
				"--sysctl",
				"net.ipv4.ip_forward=1",
			},
		},
		{
			name: "multiple sysctls",
			spec: service.Spec{
				Name: "multi-sysctl",
				Container: service.Container{
					Image: "nginx:alpine",
					Sysctls: map[string]string{
						"net.ipv4.ip_forward": "1",
						"net.core.somaxconn":  "1024",
					},
				},
			},
			containerName: "multi-sysctl-container",
			wantContains: []string{
				"--sysctl",
				"net.ipv4.ip_forward=1",
				"--sysctl",
				"net.core.somaxconn=1024",
			},
		},
		{
			name: "sysctls with various kernel parameters",
			spec: service.Spec{
				Name: "kernel-params",
				Container: service.Container{
					Image: "postgres:15",
					Sysctls: map[string]string{
						"kernel.shmmax":                "68719476736",
						"kernel.shmall":                "4294967296",
						"net.ipv4.tcp_keepalive_time":  "600",
						"net.ipv4.tcp_keepalive_intvl": "60",
					},
				},
			},
			containerName: "kernel-params-container",
			wantContains: []string{
				"--sysctl",
				"kernel.shmmax=68719476736",
				"--sysctl",
				"kernel.shmall=4294967296",
				"--sysctl",
				"net.ipv4.tcp_keepalive_time=600",
				"--sysctl",
				"net.ipv4.tcp_keepalive_intvl=60",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAllRunArgs(tt.spec, tt.containerName)

			// Verify all expected sysctl arguments are present
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want,
					"Podman args should contain '%s'", want)
			}

			// Verify --sysctl flags come in pairs (flag, value)
			sysctlIndices := findAllIndices(got, "--sysctl")
			for _, idx := range sysctlIndices {
				// Ensure there's a value after --sysctl
				assert.Less(t, idx+1, len(got),
					"--sysctl flag should be followed by a value")

				// Verify the value format (key=value)
				if idx+1 < len(got) {
					value := got[idx+1]
					assert.Regexp(t, `^[a-z0-9_.]+=[a-z0-9_.]+$`, value,
						"Sysctl value should follow format 'key=value', got: %s", value)
				}
			}
		})
	}
}

func TestBuildPodmanArgs_NoSysctls(t *testing.T) {
	spec := service.Spec{
		Name: "no-sysctls",
		Container: service.Container{
			Image: "nginx:alpine",
		},
	}

	got := BuildAllRunArgs(spec, "no-sysctls-container")

	// Verify --sysctl is not present
	assert.NotContains(t, got, "--sysctl",
		"Podman args should not contain --sysctl when no sysctls are specified")
}

// Helper function to find all indices of a string in a slice.
func findAllIndices(slice []string, target string) []int {
	var indices []int
	for i, s := range slice {
		if s == target {
			indices = append(indices, i)
		}
	}
	return indices
}
