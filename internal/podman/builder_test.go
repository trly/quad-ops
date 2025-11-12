package podman

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/service"
)

func TestBuildAllRunArgs(t *testing.T) {
	tests := []struct {
		name          string
		spec          service.Spec
		containerName string
		want          []string
	}{
		{
			name: "basic container",
			spec: service.Spec{
				Name: "basic",
				Container: service.Container{
					Image: "nginx:latest",
				},
			},
			containerName: "basic-container",
			want: []string{
				"run",
				"--rm",
				"--name", "basic-container",
				"nginx:latest",
			},
		},
		{
			name: "container with ports",
			spec: service.Spec{
				Name: "web",
				Container: service.Container{
					Image: "nginx:latest",
					Ports: []service.Port{
						{HostPort: 8080, Container: 80, Protocol: "tcp"},
					},
				},
			},
			containerName: "web-container",
			want: []string{
				"run",
				"--rm",
				"--name", "web-container",
				"-p", "8080:80/tcp",
				"nginx:latest",
			},
		},
		{
			name: "container with environment",
			spec: service.Spec{
				Name: "app",
				Container: service.Container{
					Image: "node:18",
					Env: map[string]string{
						"NODE_ENV": "production",
						"PORT":     "3000",
					},
				},
			},
			containerName: "app-container",
			want: []string{
				"run",
				"--rm",
				"--name", "app-container",
				"-e", "NODE_ENV=production", // Sorted alphabetically
				"-e", "PORT=3000",
				"node:18",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAllRunArgs(tt.spec, tt.containerName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildQuadletPodmanArgs(t *testing.T) {
	tests := []struct {
		name string
		spec service.Spec
		want []string
	}{
		{
			name: "basic container - no extra args",
			spec: service.Spec{
				Name: "basic",
				Container: service.Container{
					Image: "nginx:latest",
				},
			},
			want: []string{},
		},
		{
			name: "container with memory reservation",
			spec: service.Spec{
				Name: "app",
				Container: service.Container{
					Image: "node:18",
					Resources: service.Resources{
						MemoryReservation: "512m",
					},
				},
			},
			want: []string{
				"--memory-reservation=512m",
			},
		},
		{
			name: "container with CPU shares",
			spec: service.Spec{
				Name: "app",
				Container: service.Container{
					Image: "nginx:latest",
					Resources: service.Resources{
						CPUShares: 512,
					},
				},
			},
			want: []string{
				"--cpu-shares=512",
			},
		},
		{
			name: "container with security options - now handled by native Quadlet directives",
			spec: service.Spec{
				Name: "privileged",
				Container: service.Container{
					Image: "alpine:latest",
					Security: service.Security{
						Privileged: true,
						CapAdd:     []string{"NET_ADMIN", "SYS_TIME"},
						CapDrop:    []string{"MKNOD"},
					},
				},
			},
			want: []string{}, // Security options now use native Quadlet directives, not PodmanArgs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildQuadletPodmanArgs(tt.spec)
			assert.Equal(t, tt.want, got)
		})
	}
}
