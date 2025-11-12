package systemd

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/testutil"
)

// Golden test suite to lock in current renderer behavior before refactoring.
// These tests capture byte-for-byte output equivalence using fixture files.
//
// To regenerate golden files: go test -run TestRenderer_GoldenTests -update

var updateGolden = flag.Bool("update", false, "Update golden test fixtures")

func TestRenderer_GoldenTests(t *testing.T) {
	logger := testutil.NewTestLogger(t)
	r := NewRenderer(logger)

	cases := []struct {
		name          string
		specs         []service.Spec
		artifactCount int // Expected number of artifacts
	}{
		{
			name:          "basic_container",
			artifactCount: 1,
			specs: []service.Spec{
				{
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
				},
			},
		},
		{
			name:          "container_with_volumes",
			artifactCount: 2,
			specs: []service.Spec{
				{
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
				},
			},
		},
		{
			name:          "container_with_network",
			artifactCount: 2,
			specs: []service.Spec{
				{
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
						Network: service.NetworkMode{
							Mode:            "bridge",
							ServiceNetworks: []string{"backend"},
						},
					},
				},
			},
		},
		{
			name:          "container_with_bind_mounts",
			artifactCount: 1,
			specs: []service.Spec{
				{
					Name:        "app",
					Description: "App with bind mounts",
					Container: service.Container{
						Image: "alpine:latest",
						Mounts: []service.Mount{
							{
								Type:   service.MountTypeBind,
								Source: "/host/data",
								Target: "/app/data",
							},
							{
								Type:   service.MountTypeBind,
								Source: "/host/config",
								Target: "/app/config",
							},
						},
					},
				},
			},
		},
		{
			name:          "container_with_resources",
			artifactCount: 1,
			specs: []service.Spec{
				{
					Name: "app",
					Container: service.Container{
						Image: "alpine:latest",
						Resources: service.Resources{
							Memory:            "512m",
							MemoryReservation: "256m",
							MemorySwap:        "1g",
							ShmSize:           "64m",
						},
					},
				},
			},
		},
		{
			name:          "external_network",
			artifactCount: 2,
			specs: []service.Spec{
				{
					Name: "app",
					Networks: []service.Network{
						{
							Name:     "infrastructure_proxy",
							Driver:   "bridge",
							External: true,
						},
					},
					Container: service.Container{
						Image: "alpine:latest",
						Network: service.NetworkMode{
							Mode:            "bridge",
							ServiceNetworks: []string{"infrastructure_proxy"},
						},
					},
				},
			},
		},
		{
			name:          "multiple_networks",
			artifactCount: 3,
			specs: []service.Spec{
				{
					Name:        "immich-server",
					Description: "Immich server with multiple networks",
					Networks: []service.Network{
						{
							Name:     "default",
							Driver:   "bridge",
							External: false,
						},
						{
							Name:     "infrastructure_proxy",
							Driver:   "bridge",
							External: true,
						},
					},
					Container: service.Container{
						Image: "immich-server:latest",
						Network: service.NetworkMode{
							Mode: "bridge",
							ServiceNetworks: []string{
								"default",
								"infrastructure_proxy",
							},
						},
					},
				},
			},
		},
		{
			name:          "container_with_security",
			artifactCount: 1,
			specs: []service.Spec{
				{
					Name: "app",
					Container: service.Container{
						Image: "alpine:latest",
						Security: service.Security{
							Privileged: true,
							CapAdd:     []string{"NET_ADMIN", "SYS_TIME"},
							CapDrop:    []string{"MKNOD"},
							SecurityOpt: []string{
								"label=type:container_runtime_t",
								"no-new-privileges",
							},
						},
					},
				},
			},
		},
		{
			name:          "container_with_healthcheck",
			artifactCount: 1,
			specs: []service.Spec{
				{
					Name: "app",
					Container: service.Container{
						Image: "nginx:latest",
						Healthcheck: &service.Healthcheck{
							Test:        []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
							Interval:    30 * time.Second,
							Timeout:     10 * time.Second,
							Retries:     3,
							StartPeriod: 40 * time.Second,
						},
					},
				},
			},
		},
		{
			name:          "container_with_build",
			artifactCount: 2,
			specs: []service.Spec{
				{
					Name: "app",
					Container: service.Container{
						Image: "myapp:latest",
						Build: &service.Build{
							Context:    "/path/to/build",
							Dockerfile: "Dockerfile.prod",
							Target:     "production",
							Args: map[string]string{
								"VERSION": "1.0.0",
								"ENV":     "production",
							},
							Tags: []string{
								"myapp:latest",
								"myapp:1.0.0",
							},
							Pull: true,
						},
					},
				},
			},
		},
		{
			name:          "init_container",
			artifactCount: 1,
			specs: []service.Spec{
				{
					Name:        "web-init-0",
					Description: "Init container 0 for service web",
					Container: service.Container{
						Image:         "busybox:latest",
						Command:       []string{"sh", "-c", "echo 'init'"},
						RestartPolicy: service.RestartPolicyNo,
					},
				},
			},
		},
		{
			name:          "quadlet_volume_extensions",
			artifactCount: 2,
			specs: []service.Spec{
				{
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
				},
			},
		},
		{
			name:          "quadlet_network_extensions",
			artifactCount: 2,
			specs: []service.Spec{
				{
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
				},
			},
		},
		{
			name:          "tmpfs_with_options",
			artifactCount: 1,
			specs: []service.Spec{
				{
					Name:        "test",
					Description: "Test tmpfs options",
					Container: service.Container{
						Image: "nginx:alpine",
						Mounts: []service.Mount{
							{
								Target:   "/tmp/data",
								Type:     service.MountTypeTmpfs,
								ReadOnly: false,
								TmpfsOptions: &service.TmpfsOptions{
									Size: "64m",
									Mode: 1777,
									UID:  1000,
									GID:  1000,
								},
							},
						},
						RestartPolicy: service.RestartPolicyAlways,
					},
				},
			},
		},
		{
			name:          "sysctls",
			artifactCount: 1,
			specs: []service.Spec{
				{
					Name:        "sysctl-test",
					Description: "Test container with sysctls",
					Container: service.Container{
						Image: "nginx:alpine",
						Sysctls: map[string]string{
							"net.ipv4.ip_forward":          "1",
							"net.ipv4.conf.all.rp_filter":  "2",
							"kernel.shmmax":                "68719476736",
							"net.ipv4.tcp_keepalive_time":  "600",
							"net.ipv4.tcp_keepalive_intvl": "60",
						},
						RestartPolicy: service.RestartPolicyAlways,
					},
				},
			},
		},
		{
			name:          "network_online_with_ports",
			artifactCount: 2,
			specs: []service.Spec{
				{
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
				},
			},
		},
		{
			name:          "no_network_online_without_network",
			artifactCount: 1,
			specs: []service.Spec{
				{
					Name:        "worker",
					Description: "Background worker without network needs",
					Container: service.Container{
						Image: "worker:1.0",
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			result, err := r.Render(ctx, tc.specs)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Verify artifact count matches expectation
			require.Len(t, result.Artifacts, tc.artifactCount,
				"Expected %d artifacts, got %d. This may indicate missing or extra unit files.",
				tc.artifactCount, len(result.Artifacts))

			// Verify each artifact against golden file
			for _, artifact := range result.Artifacts {
				goldenPath := filepath.Join("testdata", tc.name, artifact.Path+".golden")

				if *updateGolden {
					// Create testdata directory if needed
					dir := filepath.Dir(goldenPath)
					err := os.MkdirAll(dir, 0755)
					require.NoError(t, err)

					// Write golden file
					err = os.WriteFile(goldenPath, artifact.Content, 0644)
					require.NoError(t, err)
					t.Logf("Updated golden file: %s", goldenPath)
				} else {
					// Read golden file and compare byte-for-byte
					expected, err := os.ReadFile(goldenPath)
					require.NoErrorf(t, err, "Failed to read golden file: %s", goldenPath)

					if diff := cmp.Diff(string(expected), string(artifact.Content)); diff != "" {
						t.Errorf("Artifact %s content mismatch (-want +got):\n%s\nRun with -update to regenerate golden files.",
							artifact.Path, diff)
					}
				}
			}
		})
	}
}
