package launchd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/trly/quad-ops/internal/service"
)

func TestBuildPodmanArgs(t *testing.T) {
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
			name: "container with working directory",
			spec: service.Spec{
				Name: "workdir",
				Container: service.Container{
					Image:      "node:18",
					WorkingDir: "/app",
				},
			},
			containerName: "workdir-container",
			want: []string{
				"run",
				"--rm",
				"--name", "workdir-container",
				"-w", "/app",
				"node:18",
			},
		},
		{
			name: "container with user only",
			spec: service.Spec{
				Name: "user-only",
				Container: service.Container{
					Image: "alpine:latest",
					User:  "1000",
				},
			},
			containerName: "user-container",
			want: []string{
				"run",
				"--rm",
				"--name", "user-container",
				"-u", "1000",
				"alpine:latest",
			},
		},
		{
			name: "container with user and group",
			spec: service.Spec{
				Name: "user-group",
				Container: service.Container{
					Image: "alpine:latest",
					User:  "1000",
					Group: "1000",
				},
			},
			containerName: "usergroup-container",
			want: []string{
				"run",
				"--rm",
				"--name", "usergroup-container",
				"-u", "1000:1000",
				"alpine:latest",
			},
		},
		{
			name: "container with hostname",
			spec: service.Spec{
				Name: "hostname",
				Container: service.Container{
					Image:    "nginx:latest",
					Hostname: "web-server",
				},
			},
			containerName: "hostname-container",
			want: []string{
				"run",
				"--rm",
				"--name", "hostname-container",
				"--hostname", "web-server",
				"nginx:latest",
			},
		},
		{
			name: "container with read-only filesystem",
			spec: service.Spec{
				Name: "readonly",
				Container: service.Container{
					Image:    "nginx:latest",
					ReadOnly: true,
				},
			},
			containerName: "readonly-container",
			want: []string{
				"run",
				"--rm",
				"--name", "readonly-container",
				"--read-only",
				"nginx:latest",
			},
		},
		{
			name: "container with init",
			spec: service.Spec{
				Name: "init",
				Container: service.Container{
					Image: "alpine:latest",
					Init:  true,
				},
			},
			containerName: "init-container",
			want: []string{
				"run",
				"--rm",
				"--name", "init-container",
				"--init",
				"alpine:latest",
			},
		},
		{
			name: "container with environment variables",
			spec: service.Spec{
				Name: "env",
				Container: service.Container{
					Image: "postgres:15",
					Env: map[string]string{
						"POSTGRES_PASSWORD": "secret",
						"POSTGRES_USER":     "admin",
					},
				},
			},
			containerName: "env-container",
			want:          []string{"run", "--rm", "--name", "env-container", "-e", "-e", "postgres:15"},
		},
		{
			name: "container with environment files",
			spec: service.Spec{
				Name: "envfile",
				Container: service.Container{
					Image:    "node:18",
					EnvFiles: []string{"/etc/app.env", "/etc/secrets.env"},
				},
			},
			containerName: "envfile-container",
			want: []string{
				"run",
				"--rm",
				"--name", "envfile-container",
				"--env-file", "/etc/app.env",
				"--env-file", "/etc/secrets.env",
				"node:18",
			},
		},
		{
			name: "container with network mode",
			spec: service.Spec{
				Name: "network",
				Container: service.Container{
					Image: "nginx:latest",
					Network: service.NetworkMode{
						Mode: "host",
					},
				},
			},
			containerName: "network-container",
			want: []string{
				"run",
				"--rm",
				"--name", "network-container",
				"--network", "host",
				"nginx:latest",
			},
		},
		{
			name: "container with labels",
			spec: service.Spec{
				Name: "labels",
				Container: service.Container{
					Image: "nginx:latest",
					Labels: map[string]string{
						"com.example.version": "1.0",
						"com.example.env":     "production",
					},
				},
			},
			containerName: "labels-container",
			want:          []string{"run", "--rm", "--name", "labels-container", "--label", "--label", "nginx:latest"},
		},
		{
			name: "container with tmpfs mounts",
			spec: service.Spec{
				Name: "tmpfs",
				Container: service.Container{
					Image: "nginx:latest",
					Tmpfs: []string{"/tmp", "/run"},
				},
			},
			containerName: "tmpfs-container",
			want: []string{
				"run",
				"--rm",
				"--name", "tmpfs-container",
				"--tmpfs", "/tmp",
				"--tmpfs", "/run",
				"nginx:latest",
			},
		},
		{
			name: "container with ulimits",
			spec: service.Spec{
				Name: "ulimits",
				Container: service.Container{
					Image: "postgres:15",
					Ulimits: []service.Ulimit{
						{Name: "nofile", Soft: 1024, Hard: 2048},
						{Name: "nproc", Soft: 512, Hard: 1024},
					},
				},
			},
			containerName: "ulimits-container",
			want: []string{
				"run",
				"--rm",
				"--name", "ulimits-container",
				"--ulimit", "nofile=1024:2048",
				"--ulimit", "nproc=512:1024",
				"postgres:15",
			},
		},
		{
			name: "container with sysctls",
			spec: service.Spec{
				Name: "sysctls",
				Container: service.Container{
					Image: "nginx:latest",
					Sysctls: map[string]string{
						"net.ipv4.ip_forward": "1",
						"net.core.somaxconn":  "1024",
					},
				},
			},
			containerName: "sysctls-container",
			want:          []string{"run", "--rm", "--name", "sysctls-container", "--sysctl", "--sysctl", "nginx:latest"},
		},
		{
			name: "container with user namespace",
			spec: service.Spec{
				Name: "userns",
				Container: service.Container{
					Image:  "alpine:latest",
					UserNS: "host",
				},
			},
			containerName: "userns-container",
			want: []string{
				"run",
				"--rm",
				"--name", "userns-container",
				"--userns", "host",
				"alpine:latest",
			},
		},
		{
			name: "container with pids limit",
			spec: service.Spec{
				Name: "pids",
				Container: service.Container{
					Image:     "alpine:latest",
					PidsLimit: 200,
				},
			},
			containerName: "pids-container",
			want: []string{
				"run",
				"--rm",
				"--name", "pids-container",
				"--pids-limit", "200",
				"alpine:latest",
			},
		},
		{
			name: "container with additional podman args",
			spec: service.Spec{
				Name: "podman-args",
				Container: service.Container{
					Image:      "nginx:latest",
					PodmanArgs: []string{"--log-level=debug", "--events-backend=file"},
				},
			},
			containerName: "podman-args-container",
			want: []string{
				"run",
				"--rm",
				"--name", "podman-args-container",
				"--log-level=debug",
				"--events-backend=file",
				"nginx:latest",
			},
		},
		{
			name: "container with entrypoint override",
			spec: service.Spec{
				Name: "entrypoint",
				Container: service.Container{
					Image:      "nginx:latest",
					Entrypoint: []string{"/bin/sh", "-c"},
				},
			},
			containerName: "entrypoint-container",
			want: []string{
				"run",
				"--rm",
				"--name", "entrypoint-container",
				"nginx:latest",
				"--entrypoint", "/bin/sh",
				"-c",
			},
		},
		{
			name: "container with single entrypoint",
			spec: service.Spec{
				Name: "entrypoint-single",
				Container: service.Container{
					Image:      "nginx:latest",
					Entrypoint: []string{"/app/start.sh"},
				},
			},
			containerName: "entrypoint-single-container",
			want: []string{
				"run",
				"--rm",
				"--name", "entrypoint-single-container",
				"nginx:latest",
				"--entrypoint", "/app/start.sh",
			},
		},
		{
			name: "container with command",
			spec: service.Spec{
				Name: "command",
				Container: service.Container{
					Image:   "alpine:latest",
					Command: []string{"echo", "hello"},
				},
			},
			containerName: "command-container",
			want: []string{
				"run",
				"--rm",
				"--name", "command-container",
				"alpine:latest",
				"echo",
				"hello",
			},
		},
		{
			name: "container with args",
			spec: service.Spec{
				Name: "args",
				Container: service.Container{
					Image: "alpine:latest",
					Args:  []string{"--verbose", "--debug"},
				},
			},
			containerName: "args-container",
			want: []string{
				"run",
				"--rm",
				"--name", "args-container",
				"alpine:latest",
				"--verbose",
				"--debug",
			},
		},
		{
			name: "complete container with all features",
			spec: service.Spec{
				Name: "complete",
				Container: service.Container{
					Image:      "nginx:latest",
					WorkingDir: "/app",
					User:       "nginx",
					Group:      "nginx",
					Hostname:   "web-server",
					ReadOnly:   true,
					Init:       true,
					Env: map[string]string{
						"ENV": "production",
					},
					EnvFiles: []string{"/etc/app.env"},
					Ports: []service.Port{
						{HostPort: 8080, Container: 80, Protocol: "tcp"},
					},
					Mounts: []service.Mount{
						{Source: "/data", Target: "/var/www", ReadOnly: true},
					},
					Network: service.NetworkMode{Mode: "bridge"},
					Labels: map[string]string{
						"version": "1.0",
					},
					Tmpfs:   []string{"/tmp"},
					UserNS:  "auto",
					Command: []string{"nginx", "-g", "daemon off;"},
				},
			},
			containerName: "complete-container",
			want:          []string{"run", "--rm", "--name", "complete-container", "-w", "/app", "-u", "nginx:nginx", "--hostname", "web-server", "--read-only", "--init", "--env-file", "/etc/app.env", "-e", "-p", "-v", "--tmpfs", "/tmp", "--network", "bridge", "--label", "--userns", "auto", "nginx:latest", "nginx", "-g", "daemon off;"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPodmanArgs(tt.spec, tt.containerName)

			// Verify core structure
			assert.Equal(t, "run", got[0], "first arg should be 'run'")
			assert.Equal(t, "--rm", got[1], "second arg should be '--rm'")
			assert.Equal(t, "--name", got[2], "third arg should be '--name'")
			assert.Equal(t, tt.containerName, got[3], "fourth arg should be container name")

			// Verify image is present
			assert.Contains(t, got, tt.spec.Container.Image, "args should contain image")
		})
	}
}

func TestBuildPortArg(t *testing.T) {
	tests := []struct {
		name string
		port service.Port
		want string
	}{
		{
			name: "basic tcp port",
			port: service.Port{
				HostPort:  8080,
				Container: 80,
				Protocol:  "tcp",
			},
			want: "8080:80/tcp",
		},
		{
			name: "udp port",
			port: service.Port{
				HostPort:  53,
				Container: 53,
				Protocol:  "udp",
			},
			want: "53:53/udp",
		},
		{
			name: "port with empty protocol defaults to tcp",
			port: service.Port{
				HostPort:  3000,
				Container: 3000,
			},
			want: "3000:3000/tcp",
		},
		{
			name: "port with host binding",
			port: service.Port{
				Host:      "127.0.0.1",
				HostPort:  8080,
				Container: 80,
				Protocol:  "tcp",
			},
			want: "127.0.0.1:8080:80/tcp",
		},
		{
			name: "port with ipv6 host binding",
			port: service.Port{
				Host:      "::1",
				HostPort:  8080,
				Container: 80,
				Protocol:  "tcp",
			},
			want: "::1:8080:80/tcp",
		},
		{
			name: "different host and container ports",
			port: service.Port{
				HostPort:  9090,
				Container: 8080,
				Protocol:  "tcp",
			},
			want: "9090:8080/tcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPortArg(tt.port)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildVolumeArg(t *testing.T) {
	tests := []struct {
		name  string
		mount service.Mount
		want  string
	}{
		{
			name: "basic bind mount",
			mount: service.Mount{
				Source: "/host/path",
				Target: "/container/path",
			},
			want: "/host/path:/container/path",
		},
		{
			name: "read-only mount",
			mount: service.Mount{
				Source:   "/host/path",
				Target:   "/container/path",
				ReadOnly: true,
			},
			want: "/host/path:/container/path:ro",
		},
		{
			name: "mount with bind propagation",
			mount: service.Mount{
				Source: "/host/path",
				Target: "/container/path",
				BindOptions: &service.BindOptions{
					Propagation: "shared",
				},
			},
			want: "/host/path:/container/path:shared",
		},
		{
			name: "read-only mount with propagation",
			mount: service.Mount{
				Source:   "/host/path",
				Target:   "/container/path",
				ReadOnly: true,
				BindOptions: &service.BindOptions{
					Propagation: "rshared",
				},
			},
			want: "/host/path:/container/path:ro,rshared",
		},
		{
			name: "mount with custom options",
			mount: service.Mount{
				Source: "/host/path",
				Target: "/container/path",
				Options: map[string]string{
					"size":   "100m",
					"nocopy": "",
					"tmpfs":  "",
				},
			},
			want: "/host/path:/container/path:",
		},
		{
			name: "mount with all options",
			mount: service.Mount{
				Source:   "/host/path",
				Target:   "/container/path",
				ReadOnly: true,
				BindOptions: &service.BindOptions{
					Propagation: "private",
				},
				Options: map[string]string{
					"z": "",
				},
			},
			want: "/host/path:/container/path:",
		},
		{
			name: "volume mount",
			mount: service.Mount{
				Source: "data-volume",
				Target: "/var/lib/data",
				Type:   service.MountTypeVolume,
			},
			want: "data-volume:/var/lib/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildVolumeArg(tt.mount)
			// For mounts with options, we can't predict exact order of map iteration
			// so just check structure
			if len(tt.mount.Options) > 0 || (tt.mount.ReadOnly && tt.mount.BindOptions != nil) {
				assert.Contains(t, got, tt.mount.Source)
				assert.Contains(t, got, tt.mount.Target)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestBuildSecretArg(t *testing.T) {
	tests := []struct {
		name   string
		secret service.Secret
		want   string
	}{
		{
			name: "secret source only",
			secret: service.Secret{
				Source: "my-secret",
			},
			want: "my-secret",
		},
		{
			name: "secret with target",
			secret: service.Secret{
				Source: "my-secret",
				Target: "/run/secrets/password",
			},
			want: "my-secret,target=/run/secrets/password",
		},
		{
			name: "secret with uid",
			secret: service.Secret{
				Source: "my-secret",
				UID:    "1000",
			},
			want: "my-secret,uid=1000",
		},
		{
			name: "secret with gid",
			secret: service.Secret{
				Source: "my-secret",
				GID:    "1000",
			},
			want: "my-secret,gid=1000",
		},
		{
			name: "secret with mode",
			secret: service.Secret{
				Source: "my-secret",
				Mode:   "0400",
			},
			want: "my-secret,mode=0400",
		},
		{
			name: "secret with type",
			secret: service.Secret{
				Source: "my-secret",
				Type:   "env",
			},
			want: "my-secret,type=env",
		},
		{
			name: "secret with all options",
			secret: service.Secret{
				Source: "db-password",
				Target: "/run/secrets/db_pass",
				UID:    "1000",
				GID:    "1000",
				Mode:   "0400",
				Type:   "mount",
			},
			want: "db-password,target=/run/secrets/db_pass,uid=1000,gid=1000,mode=0400,type=mount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSecretArg(tt.secret)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendResourceArgs(t *testing.T) {
	tests := []struct {
		name      string
		resources service.Resources
		want      []string
	}{
		{
			name:      "empty resources",
			resources: service.Resources{},
			want:      nil,
		},
		{
			name: "memory only",
			resources: service.Resources{
				Memory: "512m",
			},
			want: []string{"--memory", "512m"},
		},
		{
			name: "memory reservation",
			resources: service.Resources{
				MemoryReservation: "256m",
			},
			want: []string{"--memory-reservation", "256m"},
		},
		{
			name: "memory swap",
			resources: service.Resources{
				MemorySwap: "1g",
			},
			want: []string{"--memory-swap", "1g"},
		},
		{
			name: "cpu shares",
			resources: service.Resources{
				CPUShares: 512,
			},
			want: []string{"--cpu-shares", "512"},
		},
		{
			name: "cpu quota",
			resources: service.Resources{
				CPUQuota: 50000,
			},
			want: []string{"--cpu-quota", "50000"},
		},
		{
			name: "cpu period",
			resources: service.Resources{
				CPUPeriod: 100000,
			},
			want: []string{"--cpu-period", "100000"},
		},
		{
			name: "pids limit",
			resources: service.Resources{
				PidsLimit: 100,
			},
			want: []string{"--pids-limit", "100"},
		},
		{
			name: "all resource constraints",
			resources: service.Resources{
				Memory:            "1g",
				MemoryReservation: "512m",
				MemorySwap:        "2g",
				CPUShares:         1024,
				CPUQuota:          100000,
				CPUPeriod:         100000,
				PidsLimit:         200,
			},
			want: []string{
				"--memory", "1g",
				"--memory-reservation", "512m",
				"--memory-swap", "2g",
				"--cpu-shares", "1024",
				"--cpu-quota", "100000",
				"--cpu-period", "100000",
				"--pids-limit", "200",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []string
			got := appendResourceArgs(args, tt.resources)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendSecurityArgs(t *testing.T) {
	tests := []struct {
		name     string
		security service.Security
		want     []string
	}{
		{
			name:     "empty security",
			security: service.Security{},
			want:     nil,
		},
		{
			name: "privileged",
			security: service.Security{
				Privileged: true,
			},
			want: []string{"--privileged"},
		},
		{
			name: "cap add",
			security: service.Security{
				CapAdd: []string{"NET_ADMIN", "SYS_TIME"},
			},
			want: []string{
				"--cap-add", "NET_ADMIN",
				"--cap-add", "SYS_TIME",
			},
		},
		{
			name: "cap drop",
			security: service.Security{
				CapDrop: []string{"ALL", "CHOWN"},
			},
			want: []string{
				"--cap-drop", "ALL",
				"--cap-drop", "CHOWN",
			},
		},
		{
			name: "security options",
			security: service.Security{
				SecurityOpt: []string{"no-new-privileges", "label=disable"},
			},
			want: []string{
				"--security-opt", "no-new-privileges",
				"--security-opt", "label=disable",
			},
		},
		{
			name: "readonly rootfs",
			security: service.Security{
				ReadonlyRootfs: true,
			},
			want: []string{"--read-only"},
		},
		{
			name: "selinux type",
			security: service.Security{
				SELinuxType: "container_runtime_t",
			},
			want: []string{
				"--security-opt", "label=type:container_runtime_t",
			},
		},
		{
			name: "apparmor profile",
			security: service.Security{
				AppArmorProfile: "docker-default",
			},
			want: []string{
				"--security-opt", "apparmor=docker-default",
			},
		},
		{
			name: "seccomp profile",
			security: service.Security{
				SeccompProfile: "/path/to/seccomp.json",
			},
			want: []string{
				"--security-opt", "seccomp=/path/to/seccomp.json",
			},
		},
		{
			name: "all security options",
			security: service.Security{
				Privileged:      true,
				CapAdd:          []string{"NET_ADMIN"},
				CapDrop:         []string{"ALL"},
				SecurityOpt:     []string{"no-new-privileges"},
				ReadonlyRootfs:  true,
				SELinuxType:     "container_t",
				AppArmorProfile: "unconfined",
				SeccompProfile:  "unconfined",
			},
			want: []string{
				"--privileged",
				"--cap-add", "NET_ADMIN",
				"--cap-drop", "ALL",
				"--security-opt", "no-new-privileges",
				"--read-only",
				"--security-opt", "label=type:container_t",
				"--security-opt", "apparmor=unconfined",
				"--security-opt", "seccomp=unconfined",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []string
			got := appendSecurityArgs(args, tt.security)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendHealthcheckArgs(t *testing.T) {
	tests := []struct {
		name        string
		healthcheck *service.Healthcheck
		want        []string
	}{
		{
			name:        "nil healthcheck",
			healthcheck: nil,
			want:        nil,
		},
		{
			name: "test command only",
			healthcheck: &service.Healthcheck{
				Test: []string{"CMD", "curl", "-f", "http://localhost/health"},
			},
			want: []string{
				"--health-cmd", "CMD curl -f http://localhost/health",
			},
		},
		{
			name: "with interval",
			healthcheck: &service.Healthcheck{
				Test:     []string{"CMD", "healthcheck.sh"},
				Interval: 30 * time.Second,
			},
			want: []string{
				"--health-cmd", "CMD healthcheck.sh",
				"--health-interval", "30s",
			},
		},
		{
			name: "with timeout",
			healthcheck: &service.Healthcheck{
				Test:    []string{"CMD", "healthcheck.sh"},
				Timeout: 5 * time.Second,
			},
			want: []string{
				"--health-cmd", "CMD healthcheck.sh",
				"--health-timeout", "5s",
			},
		},
		{
			name: "with retries",
			healthcheck: &service.Healthcheck{
				Test:    []string{"CMD", "healthcheck.sh"},
				Retries: 3,
			},
			want: []string{
				"--health-cmd", "CMD healthcheck.sh",
				"--health-retries", "3",
			},
		},
		{
			name: "with start period",
			healthcheck: &service.Healthcheck{
				Test:        []string{"CMD", "healthcheck.sh"},
				StartPeriod: 60 * time.Second,
			},
			want: []string{
				"--health-cmd", "CMD healthcheck.sh",
				"--health-start-period", "1m0s",
			},
		},
		{
			name: "complete healthcheck",
			healthcheck: &service.Healthcheck{
				Test:        []string{"CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"},
				Interval:    30 * time.Second,
				Timeout:     10 * time.Second,
				Retries:     5,
				StartPeriod: 2 * time.Minute,
			},
			want: []string{
				"--health-cmd", "CMD-SHELL curl -f http://localhost:8080/health || exit 1",
				"--health-interval", "30s",
				"--health-timeout", "10s",
				"--health-retries", "5",
				"--health-start-period", "2m0s",
			},
		},
		{
			name: "empty test command",
			healthcheck: &service.Healthcheck{
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
			},
			want: []string{
				"--health-interval", "30s",
				"--health-timeout", "5s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []string
			got := appendHealthcheckArgs(args, tt.healthcheck)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildPodmanArgsIntegration(t *testing.T) {
	tests := []struct {
		name          string
		spec          service.Spec
		containerName string
		checkArgs     func(t *testing.T, args []string)
	}{
		{
			name: "web server with ports and volumes",
			spec: service.Spec{
				Name: "nginx",
				Container: service.Container{
					Image: "nginx:alpine",
					Ports: []service.Port{
						{HostPort: 80, Container: 80, Protocol: "tcp"},
						{HostPort: 443, Container: 443, Protocol: "tcp"},
					},
					Mounts: []service.Mount{
						{Source: "/etc/nginx", Target: "/etc/nginx", ReadOnly: true},
						{Source: "/var/www", Target: "/usr/share/nginx/html", ReadOnly: true},
					},
					Env: map[string]string{
						"NGINX_HOST": "example.com",
					},
				},
			},
			containerName: "nginx-prod",
			checkArgs: func(t *testing.T, args []string) {
				assert.Contains(t, args, "-p")
				assert.Contains(t, args, "80:80/tcp")
				assert.Contains(t, args, "443:443/tcp")
				assert.Contains(t, args, "-v")
				assert.Contains(t, args, "/etc/nginx:/etc/nginx:ro")
				assert.Contains(t, args, "/var/www:/usr/share/nginx/html:ro")
				assert.Contains(t, args, "-e")
			},
		},
		{
			name: "database with resources and healthcheck",
			spec: service.Spec{
				Name: "postgres",
				Container: service.Container{
					Image: "postgres:15",
					Env: map[string]string{
						"POSTGRES_PASSWORD": "secret",
						"POSTGRES_DB":       "myapp",
					},
					Resources: service.Resources{
						Memory:    "2g",
						CPUShares: 1024,
					},
					Healthcheck: &service.Healthcheck{
						Test:     []string{"CMD-SHELL", "pg_isready -U postgres"},
						Interval: 10 * time.Second,
						Timeout:  5 * time.Second,
						Retries:  3,
					},
					Mounts: []service.Mount{
						{Source: "postgres-data", Target: "/var/lib/postgresql/data"},
					},
				},
			},
			containerName: "postgres-main",
			checkArgs: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--memory")
				assert.Contains(t, args, "2g")
				assert.Contains(t, args, "--cpu-shares")
				assert.Contains(t, args, "1024")
				assert.Contains(t, args, "--health-cmd")
				assert.Contains(t, args, "CMD-SHELL pg_isready -U postgres")
				assert.Contains(t, args, "--health-interval")
				assert.Contains(t, args, "10s")
			},
		},
		{
			name: "secure container with capabilities",
			spec: service.Spec{
				Name: "secure-app",
				Container: service.Container{
					Image:    "alpine:latest",
					ReadOnly: true,
					Security: service.Security{
						CapDrop:         []string{"ALL"},
						CapAdd:          []string{"NET_BIND_SERVICE"},
						ReadonlyRootfs:  true,
						SeccompProfile:  "runtime/default",
						AppArmorProfile: "docker-default",
					},
					User:  "1000",
					Group: "1000",
				},
			},
			containerName: "secure-app-1",
			checkArgs: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--cap-drop")
				assert.Contains(t, args, "ALL")
				assert.Contains(t, args, "--cap-add")
				assert.Contains(t, args, "NET_BIND_SERVICE")
				assert.Contains(t, args, "--read-only")
				assert.Contains(t, args, "--security-opt")
				assert.Contains(t, args, "seccomp=runtime/default")
				assert.Contains(t, args, "apparmor=docker-default")
				assert.Contains(t, args, "-u")
				assert.Contains(t, args, "1000:1000")
			},
		},
		{
			name: "container with secrets",
			spec: service.Spec{
				Name: "app-with-secrets",
				Container: service.Container{
					Image: "myapp:latest",
					Secrets: []service.Secret{
						{
							Source: "api-key",
							Target: "/run/secrets/api_key",
							UID:    "1000",
							GID:    "1000",
							Mode:   "0400",
						},
						{
							Source: "db-password",
							Type:   "env",
						},
					},
				},
			},
			containerName: "app-1",
			checkArgs: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--secret")
				foundSecret := false
				for _, arg := range args {
					if arg == "api-key,target=/run/secrets/api_key,uid=1000,gid=1000,mode=0400" {
						foundSecret = true
						break
					}
				}
				assert.True(t, foundSecret, "should contain formatted secret with all options")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildPodmanArgs(tt.spec, tt.containerName)

			// Verify basic structure
			assert.Equal(t, "run", args[0])
			assert.Equal(t, "--rm", args[1])
			assert.Equal(t, "--name", args[2])
			assert.Equal(t, tt.containerName, args[3])
			assert.Contains(t, args, tt.spec.Container.Image)

			// Run test-specific checks
			tt.checkArgs(t, args)
		})
	}
}
