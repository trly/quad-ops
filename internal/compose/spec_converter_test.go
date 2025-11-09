package compose

import (
	"testing"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trly/quad-ops/internal/service"
)

func TestSpecConverter_ConvertProject(t *testing.T) {
	tests := []struct {
		name    string
		project *types.Project
		want    int // number of specs expected
		wantErr bool
	}{
		{
			name: "basic service",
			project: &types.Project{
				Name:       "test-project",
				WorkingDir: "/test",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
				},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "multiple services",
			project: &types.Project{
				Name:       "multi",
				WorkingDir: "/test",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
					"db": {
						Name:  "db",
						Image: "postgres:15",
					},
				},
			},
			want:    2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewSpecConverter(tt.project.WorkingDir)
			specs, err := converter.ConvertProject(tt.project)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, specs, tt.want)

			// Validate all specs
			for _, spec := range specs {
				assert.NoError(t, spec.Validate())
			}
		})
	}
}

func TestSpecConverter_ConvertService(t *testing.T) {
	tests := []struct {
		name           string
		serviceName    string
		composeService types.ServiceConfig
		project        *types.Project
		validate       func(t *testing.T, specs []service.Spec)
		wantErr        bool
	}{
		{
			name:        "basic container",
			serviceName: "web",
			composeService: types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				assert.Equal(t, "test-web", spec.Name)
				assert.Equal(t, "nginx:latest", spec.Container.Image)
			},
		},
		{
			name:        "service with environment",
			serviceName: "app",
			composeService: types.ServiceConfig{
				Name:  "app",
				Image: "app:1.0",
				Environment: types.MappingWithEquals{
					"DEBUG":   strPtr("true"),
					"PORT":    strPtr("8080"),
					"EMPTY":   nil,
					"API_KEY": strPtr("secret123"),
				},
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				assert.Len(t, spec.Container.Env, 4)
				assert.Equal(t, "true", spec.Container.Env["DEBUG"])
				assert.Equal(t, "8080", spec.Container.Env["PORT"])
				assert.Equal(t, "", spec.Container.Env["EMPTY"])
				assert.Equal(t, "secret123", spec.Container.Env["API_KEY"])
			},
		},
		{
			name:        "service with ports",
			serviceName: "web",
			composeService: types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
				Ports: []types.ServicePortConfig{
					{
						Published: "8080",
						Target:    80,
						Protocol:  "tcp",
					},
					{
						Published: "8443",
						Target:    443,
						Protocol:  "tcp",
						HostIP:    "0.0.0.0",
					},
				},
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				assert.Len(t, spec.Container.Ports, 2)
				assert.Equal(t, uint16(8080), spec.Container.Ports[0].HostPort)
				assert.Equal(t, uint16(80), spec.Container.Ports[0].Container)
				assert.Equal(t, "tcp", spec.Container.Ports[0].Protocol)
				assert.Equal(t, "0.0.0.0", spec.Container.Ports[1].Host)
			},
		},
		{
			name:        "service with volumes",
			serviceName: "app",
			composeService: types.ServiceConfig{
				Name:  "app",
				Image: "app:1.0",
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "bind",
						Source: "/host/path",
						Target: "/container/path",
					},
					{
						Type:   "volume",
						Source: "data-vol",
						Target: "/data",
					},
					{
						Type:     "bind",
						Source:   "/readonly",
						Target:   "/ro",
						ReadOnly: true,
					},
				},
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				assert.Len(t, spec.Container.Mounts, 3)
				assert.Equal(t, service.MountTypeBind, spec.Container.Mounts[0].Type)
				assert.Equal(t, "/host/path", spec.Container.Mounts[0].Source)
				assert.Equal(t, service.MountTypeVolume, spec.Container.Mounts[1].Type)
				assert.True(t, spec.Container.Mounts[2].ReadOnly)
			},
		},
		{
			name:        "service with healthcheck",
			serviceName: "web",
			composeService: types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
				HealthCheck: &types.HealthCheckConfig{
					Test:          []string{"CMD", "curl", "-f", "http://localhost"},
					Interval:      durationPtr(30 * time.Second),
					Timeout:       durationPtr(10 * time.Second),
					Retries:       uint64Ptr(3),
					StartPeriod:   durationPtr(40 * time.Second),
					StartInterval: durationPtr(5 * time.Second),
				},
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				require.NotNil(t, spec.Container.Healthcheck)
				assert.Equal(t, []string{"CMD", "curl", "-f", "http://localhost"}, spec.Container.Healthcheck.Test)
				assert.Equal(t, 30*time.Second, spec.Container.Healthcheck.Interval)
				assert.Equal(t, 10*time.Second, spec.Container.Healthcheck.Timeout)
				assert.Equal(t, 3, spec.Container.Healthcheck.Retries)
				assert.Equal(t, 40*time.Second, spec.Container.Healthcheck.StartPeriod)
				assert.Equal(t, 5*time.Second, spec.Container.Healthcheck.StartInterval)
			},
		},
		{
			name:        "service with restart policy",
			serviceName: "app",
			composeService: types.ServiceConfig{
				Name:    "app",
				Image:   "app:1.0",
				Restart: "unless-stopped",
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				assert.Equal(t, service.RestartPolicyUnlessStopped, spec.Container.RestartPolicy)
			},
		},
		{
			name:        "service with build config",
			serviceName: "app",
			composeService: types.ServiceConfig{
				Name: "app",
				Build: &types.BuildConfig{
					Context:    "./app",
					Dockerfile: "Dockerfile.prod",
					Target:     "production",
					Args: types.MappingWithEquals{
						"VERSION": strPtr("1.0.0"),
						"PYTHON":  strPtr("3.11"),
					},
					Tags: []string{"myapp:latest", "myapp:v1.0.0"},
				},
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				require.NotNil(t, spec.Container.Build)
				assert.Contains(t, spec.Container.Build.Context, "/test/app")
				assert.Equal(t, "Dockerfile.prod", spec.Container.Build.Dockerfile)
				assert.Equal(t, "production", spec.Container.Build.Target)
				assert.Equal(t, "1.0.0", spec.Container.Build.Args["VERSION"])
				assert.Equal(t, []string{"myapp:latest", "myapp:v1.0.0"}, spec.Container.Build.Tags)
			},
		},
		{
			name:        "service with dependencies",
			serviceName: "web",
			composeService: types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
				DependsOn: types.DependsOnConfig{
					"db":    {Condition: "service_healthy"},
					"redis": {Condition: "service_started"},
				},
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				assert.Len(t, spec.DependsOn, 2)
				// Dependencies should be sorted
				assert.Contains(t, spec.DependsOn, "test-db")
				assert.Contains(t, spec.DependsOn, "test-redis")
			},
		},
		{
			name:        "service with resources",
			serviceName: "app",
			composeService: types.ServiceConfig{
				Name:  "app",
				Image: "app:1.0",
				Deploy: &types.DeployConfig{
					Resources: types.Resources{
						Limits: &types.Resource{
							MemoryBytes: 512 * 1024 * 1024,   // 512MB
							NanoCPUs:    types.NanoCPUs(0.5), // 50% of one CPU
							Pids:        100,
						},
						Reservations: &types.Resource{
							MemoryBytes: 256 * 1024 * 1024, // 256MB
						},
					},
				},
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				assert.Equal(t, "512m", spec.Container.Resources.Memory)
				assert.Equal(t, "256m", spec.Container.Resources.MemoryReservation)
				assert.Equal(t, int64(50000), spec.Container.Resources.CPUQuota)
				assert.Equal(t, int64(100000), spec.Container.Resources.CPUPeriod)
				assert.Equal(t, int64(100), spec.Container.Resources.PidsLimit)
			},
		},
		{
			name:        "service with security settings",
			serviceName: "app",
			composeService: types.ServiceConfig{
				Name:       "app",
				Image:      "app:1.0",
				Privileged: true,
				CapAdd:     []string{"NET_ADMIN", "SYS_TIME"},
				CapDrop:    []string{"ALL"},
				SecurityOpt: []string{
					"seccomp=unconfined",
					"apparmor=docker-default",
					"label=type:container_runtime_t",
				},
				ReadOnly: true,
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				assert.True(t, spec.Container.Security.Privileged)
				assert.Equal(t, []string{"NET_ADMIN", "SYS_TIME"}, spec.Container.Security.CapAdd)
				assert.Equal(t, []string{"ALL"}, spec.Container.Security.CapDrop)
				assert.Equal(t, "unconfined", spec.Container.Security.SeccompProfile)
				assert.Equal(t, "docker-default", spec.Container.Security.AppArmorProfile)
				assert.Equal(t, "container_runtime_t", spec.Container.Security.SELinuxType)
				assert.True(t, spec.Container.Security.ReadonlyRootfs)
			},
		},
		{
			name:        "service with user and working directory",
			serviceName: "app",
			composeService: types.ServiceConfig{
				Name:       "app",
				Image:      "app:1.0",
				User:       "1000:1000",
				WorkingDir: "/app",
				Hostname:   "app-host",
			},
			project: &types.Project{
				Name:       "test",
				WorkingDir: "/test",
			},
			validate: func(t *testing.T, specs []service.Spec) {
				require.Len(t, specs, 1)
				spec := specs[0]
				assert.Equal(t, "1000", spec.Container.User)
				assert.Equal(t, "1000", spec.Container.Group)
				assert.Equal(t, "/app", spec.Container.WorkingDir)
				assert.Equal(t, "app-host", spec.Container.Hostname)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewSpecConverter(tt.project.WorkingDir)
			specs, err := converter.convertService(tt.serviceName, tt.composeService, tt.project)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			for _, spec := range specs {
				assert.NoError(t, spec.Validate())
			}

			if tt.validate != nil {
				tt.validate(t, specs)
			}
		})
	}
}

func TestSpecConverter_ConvertProjectVolumes(t *testing.T) {
	tests := []struct {
		name    string
		project *types.Project
		want    int
	}{
		{
			name: "single volume",
			project: &types.Project{
				Name: "test",
				Volumes: types.Volumes{
					"data": {
						Name:   "data",
						Driver: "local",
					},
				},
			},
			want: 1,
		},
		{
			name: "external volume skipped",
			project: &types.Project{
				Name: "test",
				Volumes: types.Volumes{
					"data": {
						Name:     "data",
						External: true,
					},
				},
			},
			want: 0,
		},
		{
			name: "multiple volumes with options",
			project: &types.Project{
				Name: "test",
				Volumes: types.Volumes{
					"data": {
						Name:   "data",
						Driver: "local",
						DriverOpts: map[string]string{
							"type": "nfs",
						},
						Labels: types.Labels{
							"app": "myapp",
						},
					},
					"cache": {
						Name:   "cache",
						Driver: "local",
					},
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewSpecConverter("/test")
			volumes := converter.convertProjectVolumes(tt.project)
			assert.Len(t, volumes, tt.want)

			// Validate all volumes
			for _, vol := range volumes {
				assert.NoError(t, vol.Validate())
				assert.Contains(t, vol.Name, "test-")
			}
		})
	}
}

func TestSpecConverter_ConvertProjectNetworks(t *testing.T) {
	tests := []struct {
		name    string
		project *types.Project
		want    int
	}{
		{
			name: "single network",
			project: &types.Project{
				Name: "test",
				Networks: types.Networks{
					"frontend": {
						Name:   "frontend",
						Driver: "bridge",
					},
				},
			},
			want: 1,
		},
		{
			name: "external network skipped",
			project: &types.Project{
				Name: "test",
				Networks: types.Networks{
					"frontend": {
						Name:     "frontend",
						External: true,
					},
				},
			},
			want: 0,
		},
		{
			name: "network with IPAM",
			project: &types.Project{
				Name: "test",
				Networks: types.Networks{
					"backend": {
						Name:   "backend",
						Driver: "bridge",
						Ipam: types.IPAMConfig{
							Driver: "default",
							Config: []*types.IPAMPool{
								{
									Subnet:  "172.20.0.0/16",
									Gateway: "172.20.0.1",
								},
							},
						},
					},
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewSpecConverter("/test")
			networks := converter.convertProjectNetworks(tt.project)
			assert.Len(t, networks, tt.want)

			// Validate all networks
			for _, net := range networks {
				assert.NoError(t, net.Validate())
				assert.Contains(t, net.Name, "test-")
			}
		})
	}
}

func TestSpecConverter_ConvertRestartPolicy(t *testing.T) {
	tests := []struct {
		input string
		want  service.RestartPolicy
	}{
		{"no", service.RestartPolicyNo},
		{"always", service.RestartPolicyAlways},
		{"on-failure", service.RestartPolicyOnFailure},
		{"unless-stopped", service.RestartPolicyUnlessStopped},
		{"", service.RestartPolicyNo},
		{"invalid", service.RestartPolicyNo},
	}

	converter := NewSpecConverter("/test")
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := converter.convertRestartPolicy(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSpecConverter_FormatBytes(t *testing.T) {
	tests := []struct {
		input types.UnitBytes
		want  string
	}{
		{512, "512"},
		{1024, "1k"},
		{1024 * 512, "512k"},
		{1024 * 1024, "1m"},
		{1024 * 1024 * 512, "512m"},
		{1024 * 1024 * 1024, "1g"},
		{1024 * 1024 * 1024 * 2, "2g"},
	}

	converter := NewSpecConverter("/test")
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := converter.formatBytes(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSpecConverter_ConvertCPU(t *testing.T) {
	tests := []struct {
		name       string
		input      types.NanoCPUs
		wantQuota  int64
		wantPeriod int64
	}{
		{"0.5 CPUs", types.NanoCPUs(0.5), 50000, 100000},
		{"1.0 CPUs", types.NanoCPUs(1.0), 100000, 100000},
		{"2.0 CPUs", types.NanoCPUs(2.0), 200000, 100000},
		{"0.25 CPUs", types.NanoCPUs(0.25), 25000, 100000},
		{"zero CPUs", types.NanoCPUs(0), 0, 0},
	}

	converter := NewSpecConverter("/test")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quota, period := converter.convertCPU(tt.input)
			assert.Equal(t, tt.wantQuota, quota)
			assert.Equal(t, tt.wantPeriod, period)
		})
	}
}

func TestSpecConverter_NameSanitization(t *testing.T) {
	tests := []struct {
		projectName string
		serviceName string
		wantPrefix  string
	}{
		{"my-project", "web", "my-project-web"},
		{"project_name", "db", "project_name-db"},
		{"Project With Spaces", "api", "Project-With-Spaces-api"},
	}

	for _, tt := range tests {
		t.Run(tt.projectName+"/"+tt.serviceName, func(t *testing.T) {
			converter := NewSpecConverter("/test")
			project := &types.Project{
				Name:       tt.projectName,
				WorkingDir: "/test",
				Services: types.Services{
					tt.serviceName: {
						Name:  tt.serviceName,
						Image: "test:latest",
					},
				},
			}

			specs, err := converter.ConvertProject(project)
			require.NoError(t, err)
			require.Len(t, specs, 1)

			// Name should be sanitized
			assert.Equal(t, service.SanitizeName(tt.wantPrefix), specs[0].Name)
		})
	}
}

func TestSpecConverter_ServiceNetworks(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		composeService  types.ServiceConfig
		projectNetworks types.Networks
		projectName     string
		validate        func(t *testing.T, spec service.Spec)
	}{
		{
			name:        "service with no networks specified",
			serviceName: "web",
			composeService: types.ServiceConfig{
				Name:     "web",
				Image:    "nginx:latest",
				Networks: nil,
			},
			projectNetworks: types.Networks{},
			projectName:     "test",
			validate: func(t *testing.T, spec service.Spec) {
				// Should have project networks only (none in this case)
				assert.Len(t, spec.Networks, 0)
			},
		},
		{
			name:        "service with single service-level network",
			serviceName: "api",
			composeService: types.ServiceConfig{
				Name:  "api",
				Image: "api:1.0",
				Networks: map[string]*types.ServiceNetworkConfig{
					"backend": {
						Aliases: []string{"api-service"},
					},
				},
			},
			projectNetworks: types.Networks{
				"backend": {
					Name:   "backend",
					Driver: "bridge",
				},
			},
			projectName: "myapp",
			validate: func(t *testing.T, spec service.Spec) {
				// Should have the service-level network
				assert.Len(t, spec.Networks, 1)
				assert.Equal(t, service.SanitizeName("myapp-backend"), spec.Networks[0].Name)
				assert.Equal(t, "bridge", spec.Networks[0].Driver)
			},
		},
		{
			name:        "service with multiple service-level networks",
			serviceName: "web",
			composeService: types.ServiceConfig{
				Name:  "web",
				Image: "nginx:latest",
				Networks: map[string]*types.ServiceNetworkConfig{
					"frontend": {},
					"backend":  {},
				},
			},
			projectNetworks: types.Networks{
				"frontend": {
					Name:   "frontend",
					Driver: "bridge",
				},
				"backend": {
					Name:   "backend",
					Driver: "bridge",
				},
			},
			projectName: "test",
			validate: func(t *testing.T, spec service.Spec) {
				// Should have both service-level networks
				assert.Len(t, spec.Networks, 2)
				networkNames := make(map[string]bool)
				for _, net := range spec.Networks {
					networkNames[net.Name] = true
				}
				assert.True(t, networkNames[service.SanitizeName("test-frontend")])
				assert.True(t, networkNames[service.SanitizeName("test-backend")])
			},
		},
		{
			name:        "service with external networks",
			serviceName: "app",
			composeService: types.ServiceConfig{
				Name:  "app",
				Image: "app:1.0",
				Networks: map[string]*types.ServiceNetworkConfig{
					"external-net": {},
				},
			},
			projectNetworks: types.Networks{
				"external-net": {
					Name:     "external-net",
					Driver:   "bridge",
					External: true,
				},
			},
			projectName: "test",
			validate: func(t *testing.T, spec service.Spec) {
				// Should still include external networks
				assert.Len(t, spec.Networks, 1)
				assert.Equal(t, service.SanitizeName("test-external-net"), spec.Networks[0].Name)
				assert.True(t, spec.Networks[0].External)
			},
		},
		{
			name:        "service with external network from different project (not in project.Networks)",
			serviceName: "llm-app",
			composeService: types.ServiceConfig{
				Name:  "llm-app",
				Image: "app:1.0",
				Networks: map[string]*types.ServiceNetworkConfig{
					"infrastructure-proxy": {},
				},
			},
			projectNetworks: types.Networks{},
			projectName:     "llm",
			validate: func(t *testing.T, spec service.Spec) {
				// Should NOT prefix with current project name
				// The external network is from another project and should be used as-is
				assert.Len(t, spec.Networks, 1)
				assert.Equal(t, service.SanitizeName("infrastructure-proxy"), spec.Networks[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewSpecConverter("/test")
			project := &types.Project{
				Name:       tt.projectName,
				WorkingDir: "/test",
				Networks:   tt.projectNetworks,
				Services: types.Services{
					tt.serviceName: tt.composeService,
				},
			}

			specs, err := converter.ConvertProject(project)
			require.NoError(t, err)
			require.Len(t, specs, 1)

			spec := specs[0]
			assert.NoError(t, spec.Validate())

			if tt.validate != nil {
				tt.validate(t, spec)
			}
		})
	}
}

// Helper functions.

func strPtr(s string) *string {
	return &s
}

func durationPtr(d time.Duration) *types.Duration {
	td := types.Duration(d)
	return &td
}

func uint64Ptr(i uint64) *uint64 {
	return &i
}
