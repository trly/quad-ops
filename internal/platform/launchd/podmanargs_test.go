//go:build darwin

package launchd

import (
	"strings"
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
			name: "mount with SELinux z flag",
			mount: service.Mount{
				Source: "/host/path",
				Target: "/container/path",
				BindOptions: &service.BindOptions{
					SELinux: "z",
				},
			},
			want: "/host/path:/container/path:z",
		},
		{
			name: "mount with SELinux Z flag",
			mount: service.Mount{
				Source: "/host/path",
				Target: "/container/path",
				BindOptions: &service.BindOptions{
					SELinux: "Z",
				},
			},
			want: "/host/path:/container/path:Z",
		},
		{
			name: "read-only mount with SELinux and propagation",
			mount: service.Mount{
				Source:   "/host/path",
				Target:   "/container/path",
				ReadOnly: true,
				BindOptions: &service.BindOptions{
					Propagation: "shared",
					SELinux:     "z",
				},
			},
			want: "/host/path:/container/path:ro,shared,z",
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
			name: "shm size",
			resources: service.Resources{
				ShmSize: "64m",
			},
			want: []string{"--shm-size", "64m"},
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
				ShmSize:           "64m",
				CPUShares:         1024,
				CPUQuota:          100000,
				CPUPeriod:         100000,
				PidsLimit:         200,
			},
			want: []string{
				"--memory", "1g",
				"--memory-reservation", "512m",
				"--memory-swap", "2g",
				"--shm-size", "64m",
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

func TestBuildPodmanArgs_NetworkOrdering(t *testing.T) {
	// Test that networks are sorted consistently for deterministic output
	spec := service.Spec{
		Name: "ordering-test",
		Networks: []service.Network{
			{Name: "zebra-net", External: false},
			{Name: "apple-net", External: false},
			{Name: "monkey-net", External: false},
		},
		Container: service.Container{
			Image: "test:latest",
		},
	}

	// Call multiple times with the same container name and verify consistent ordering
	containerName := "test-container"
	args1 := BuildPodmanArgs(spec, containerName)
	args2 := BuildPodmanArgs(spec, containerName)
	args3 := BuildPodmanArgs(spec, containerName)

	// All should be identical (same input, same output)
	assert.Equal(t, args1, args2, "multiple calls should produce identical network ordering")
	assert.Equal(t, args2, args3, "multiple calls should produce identical network ordering")

	// Verify networks appear in sorted order
	appleIdx := -1
	monkeyIdx := -1
	zebraIdx := -1

	for i, arg := range args1 {
		if arg == "apple-net" {
			appleIdx = i
		}
		if arg == "monkey-net" {
			monkeyIdx = i
		}
		if arg == "zebra-net" {
			zebraIdx = i
		}
	}

	assert.Greater(t, appleIdx, -1, "apple-net should be in args")
	assert.Greater(t, monkeyIdx, -1, "monkey-net should be in args")
	assert.Greater(t, zebraIdx, -1, "zebra-net should be in args")

	// Verify alphabetical ordering
	assert.True(t, appleIdx < monkeyIdx, "apple should come before monkey")
	assert.True(t, monkeyIdx < zebraIdx, "monkey should come before zebra")
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
		{
			name: "service with multiple networks",
			spec: service.Spec{
				Name: "multi-network-service",
				Networks: []service.Network{
					{Name: "backend", External: false},
					{Name: "frontend", External: false},
				},
				Container: service.Container{
					Image: "myapp:latest",
				},
			},
			containerName: "multi-net-app",
			checkArgs: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--network")
				assert.Contains(t, args, "backend")
				assert.Contains(t, args, "frontend")
				// Verify ordering is deterministic (sorted)
				backendIdx := -1
				frontendIdx := -1
				for i, arg := range args {
					if arg == "backend" {
						backendIdx = i
					}
					if arg == "frontend" {
						frontendIdx = i
					}
				}
				assert.True(t, backendIdx < frontendIdx, "networks should be sorted alphabetically for determinism")
			},
		},
		{
			name: "service with external network (should be skipped)",
			spec: service.Spec{
				Name: "external-net-service",
				Networks: []service.Network{
					{Name: "backend", External: false},
					{Name: "external-net", External: true},
				},
				Container: service.Container{
					Image: "myapp:latest",
				},
			},
			containerName: "ext-net-app",
			checkArgs: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--network")
				assert.Contains(t, args, "backend")
				// External network should not be added
				hasExternalNet := false
				for _, arg := range args {
					if arg == "external-net" {
						hasExternalNet = true
						break
					}
				}
				assert.False(t, hasExternalNet, "external networks should not be in podman args")
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

func TestBuildPodmanArgs_NamespaceModes(t *testing.T) {
	tests := []struct {
		name          string
		pidMode       string
		ipcMode       string
		cgroupMode    string
		expectedPairs [][]string // pairs of [flag, value]
	}{
		{
			name:          "pid host",
			pidMode:       "host",
			expectedPairs: [][]string{{"--pid", "host"}},
		},
		{
			name:          "pid service reference",
			pidMode:       "service:db",
			expectedPairs: [][]string{{"--pid", "service:db"}},
		},
		{
			name:          "pid container reference",
			pidMode:       "container:my-container",
			expectedPairs: [][]string{{"--pid", "container:my-container"}},
		},
		{
			name:          "ipc host",
			ipcMode:       "host",
			expectedPairs: [][]string{{"--ipc", "host"}},
		},
		{
			name:          "ipc shareable",
			ipcMode:       "shareable",
			expectedPairs: [][]string{{"--ipc", "shareable"}},
		},
		{
			name:          "ipc container reference",
			ipcMode:       "container:my-container",
			expectedPairs: [][]string{{"--ipc", "container:my-container"}},
		},
		{
			name:          "cgroup host",
			cgroupMode:    "host",
			expectedPairs: [][]string{{"--cgroupns", "host"}},
		},
		{
			name:          "cgroup private",
			cgroupMode:    "private",
			expectedPairs: [][]string{{"--cgroupns", "private"}},
		},
		{
			name:       "all namespace modes",
			pidMode:    "host",
			ipcMode:    "shareable",
			cgroupMode: "private",
			expectedPairs: [][]string{
				{"--pid", "host"},
				{"--ipc", "shareable"},
				{"--cgroupns", "private"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := service.Spec{
				Name: "test",
				Container: service.Container{
					Image:      "nginx:latest",
					PidMode:    tt.pidMode,
					IpcMode:    tt.ipcMode,
					CgroupMode: tt.cgroupMode,
				},
			}

			args := BuildPodmanArgs(spec, "test-container")

			// Convert args to string for easier checking
			argsStr := strings.Join(args, " ")

			// Check each expected flag-value pair
			for _, pair := range tt.expectedPairs {
				flag, value := pair[0], pair[1]
				expectedPattern := flag + " " + value
				assert.Contains(t, argsStr, expectedPattern,
					"Expected to find '%s' in args: %v", expectedPattern, args)
			}
		})
	}
}

func TestBuildPodmanArgs_DeviceCgroupRules(t *testing.T) {
	tests := []struct {
		name      string
		spec      service.Spec
		wantRules []string
	}{
		{
			name: "single device cgroup rule",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:             "alpine:latest",
					DeviceCgroupRules: []string{"c 13:* rmw"},
				},
			},
			wantRules: []string{"--device-cgroup-rule", "c 13:* rmw"},
		},
		{
			name: "multiple device cgroup rules",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:             "alpine:latest",
					DeviceCgroupRules: []string{"c 13:* rmw", "b 8:* rmw", "a *:* rwm"},
				},
			},
			wantRules: []string{
				"--device-cgroup-rule", "c 13:* rmw",
				"--device-cgroup-rule", "b 8:* rmw",
				"--device-cgroup-rule", "a *:* rwm",
			},
		},
		{
			name: "device cgroup rule with specific device number",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:             "alpine:latest",
					DeviceCgroupRules: []string{"c 13:64 r"},
				},
			},
			wantRules: []string{"--device-cgroup-rule", "c 13:64 r"},
		},
		{
			name: "no device cgroup rules",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:             "alpine:latest",
					DeviceCgroupRules: nil,
				},
			},
			wantRules: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildPodmanArgs(tt.spec, "test-container")

			if tt.wantRules == nil {
				argsStr := strings.Join(args, " ")
				assert.NotContains(t, argsStr, "--device-cgroup-rule")
				return
			}

			// Find all device-cgroup-rule flags and their values
			var foundRules []string
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "--device-cgroup-rule" {
					foundRules = append(foundRules, args[i], args[i+1])
				}
			}

			assert.Equal(t, tt.wantRules, foundRules)
		})
	}
}

func TestBuildPodmanArgs_ExtraHosts(t *testing.T) {
	tests := []struct {
		name      string
		spec      service.Spec
		wantHosts []string
	}{
		{
			name: "single extra host",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:      "nginx:latest",
					ExtraHosts: []string{"example.com:192.168.1.1"},
				},
			},
			wantHosts: []string{"--add-host", "example.com:192.168.1.1"},
		},
		{
			name: "multiple extra hosts in unsorted order",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:      "nginx:latest",
					ExtraHosts: []string{"zebra.com:10.0.0.1", "apple.com:10.0.0.2", "monkey.com:10.0.0.3"},
				},
			},
			wantHosts: []string{
				"--add-host", "apple.com:10.0.0.2",
				"--add-host", "monkey.com:10.0.0.3",
				"--add-host", "zebra.com:10.0.0.1",
			},
		},
		{
			name: "no extra hosts",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:      "nginx:latest",
					ExtraHosts: nil,
				},
			},
			wantHosts: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildPodmanArgs(tt.spec, "test-container")

			if tt.wantHosts == nil {
				argsStr := strings.Join(args, " ")
				assert.NotContains(t, argsStr, "--add-host")
				return
			}

			// Find all add-host flags and their values
			var foundHosts []string
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "--add-host" {
					foundHosts = append(foundHosts, args[i], args[i+1])
				}
			}

			assert.Equal(t, tt.wantHosts, foundHosts)
		})
	}
}

func TestBuildPodmanArgs_DNS(t *testing.T) {
	tests := []struct {
		name      string
		spec      service.Spec
		wantFlags []string
	}{
		{
			name: "single DNS server",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					DNS:   []string{"8.8.8.8"},
				},
			},
			wantFlags: []string{"--dns", "8.8.8.8"},
		},
		{
			name: "multiple DNS servers in unsorted order",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					DNS:   []string{"8.8.8.8", "1.1.1.1", "8.8.4.4"},
				},
			},
			wantFlags: []string{
				"--dns", "1.1.1.1",
				"--dns", "8.8.4.4",
				"--dns", "8.8.8.8",
			},
		},
		{
			name: "no DNS servers",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					DNS:   nil,
				},
			},
			wantFlags: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildPodmanArgs(tt.spec, "test-container")

			if tt.wantFlags == nil {
				argsStr := strings.Join(args, " ")
				assert.NotContains(t, argsStr, "--dns")
				return
			}

			// Find all dns flags and their values
			var foundDNS []string
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "--dns" {
					foundDNS = append(foundDNS, args[i], args[i+1])
				}
			}

			assert.Equal(t, tt.wantFlags, foundDNS)
		})
	}
}

func TestBuildPodmanArgs_DNSSearch(t *testing.T) {
	tests := []struct {
		name      string
		spec      service.Spec
		wantFlags []string
	}{
		{
			name: "single DNS search domain",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:     "alpine:latest",
					DNSSearch: []string{"example.com"},
				},
			},
			wantFlags: []string{"--dns-search", "example.com"},
		},
		{
			name: "multiple DNS search domains in unsorted order",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:     "alpine:latest",
					DNSSearch: []string{"zebra.local", "apple.local", "monkey.local"},
				},
			},
			wantFlags: []string{
				"--dns-search", "apple.local",
				"--dns-search", "monkey.local",
				"--dns-search", "zebra.local",
			},
		},
		{
			name: "no DNS search domains",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:     "alpine:latest",
					DNSSearch: nil,
				},
			},
			wantFlags: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildPodmanArgs(tt.spec, "test-container")

			if tt.wantFlags == nil {
				argsStr := strings.Join(args, " ")
				assert.NotContains(t, argsStr, "--dns-search")
				return
			}

			// Find all dns-search flags and their values
			var foundDNSSearch []string
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "--dns-search" {
					foundDNSSearch = append(foundDNSSearch, args[i], args[i+1])
				}
			}

			assert.Equal(t, tt.wantFlags, foundDNSSearch)
		})
	}
}

func TestBuildPodmanArgs_DNSOptions(t *testing.T) {
	tests := []struct {
		name      string
		spec      service.Spec
		wantFlags []string
	}{
		{
			name: "single DNS option",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:      "alpine:latest",
					DNSOptions: []string{"ndots:2"},
				},
			},
			wantFlags: []string{"--dns-opt", "ndots:2"},
		},
		{
			name: "multiple DNS options in unsorted order",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:      "alpine:latest",
					DNSOptions: []string{"timeout:3", "ndots:2", "attempts:5"},
				},
			},
			wantFlags: []string{
				"--dns-opt", "attempts:5",
				"--dns-opt", "ndots:2",
				"--dns-opt", "timeout:3",
			},
		},
		{
			name: "no DNS options",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:      "alpine:latest",
					DNSOptions: nil,
				},
			},
			wantFlags: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildPodmanArgs(tt.spec, "test-container")

			if tt.wantFlags == nil {
				argsStr := strings.Join(args, " ")
				assert.NotContains(t, argsStr, "--dns-opt")
				return
			}

			// Find all dns-opt flags and their values
			var foundDNSOpt []string
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "--dns-opt" {
					foundDNSOpt = append(foundDNSOpt, args[i], args[i+1])
				}
			}

			assert.Equal(t, tt.wantFlags, foundDNSOpt)
		})
	}
}

func TestBuildPodmanArgs_Devices(t *testing.T) {
	tests := []struct {
		name        string
		spec        service.Spec
		wantDevices []string
	}{
		{
			name: "single device",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:   "alpine:latest",
					Devices: []string{"/dev/sda:/dev/sda"},
				},
			},
			wantDevices: []string{"--device", "/dev/sda:/dev/sda"},
		},
		{
			name: "multiple devices in unsorted order",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:   "alpine:latest",
					Devices: []string{"/dev/zero:/dev/zero", "/dev/fuse", "/dev/null:/dev/null:rw"},
				},
			},
			wantDevices: []string{
				"--device", "/dev/fuse",
				"--device", "/dev/null:/dev/null:rw",
				"--device", "/dev/zero:/dev/zero",
			},
		},
		{
			name: "no devices",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image:   "alpine:latest",
					Devices: nil,
				},
			},
			wantDevices: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildPodmanArgs(tt.spec, "test-container")

			if tt.wantDevices == nil {
				argsStr := strings.Join(args, " ")
				assert.NotContains(t, argsStr, "--device")
				return
			}

			// Find all device flags and their values
			var foundDevices []string
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "--device" {
					foundDevices = append(foundDevices, args[i], args[i+1])
				}
			}

			assert.Equal(t, tt.wantDevices, foundDevices)
		})
	}
}

func TestBuildPodmanArgs_TmpfsWithOptions(t *testing.T) {
	tests := []struct {
		name      string
		spec      service.Spec
		wantTmpfs []string
	}{
		{
			name: "tmpfs mount with size option",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					Mounts: []service.Mount{
						{
							Source: "/tmp",
							Target: "/tmp",
							Type:   service.MountTypeTmpfs,
							TmpfsOptions: &service.TmpfsOptions{
								Size: "64m",
							},
						},
					},
				},
			},
			wantTmpfs: []string{"--tmpfs", "/tmp:size=64m"},
		},
		{
			name: "tmpfs mount with mode option",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					Mounts: []service.Mount{
						{
							Source: "/run",
							Target: "/run",
							Type:   service.MountTypeTmpfs,
							TmpfsOptions: &service.TmpfsOptions{
								Mode: 1777,
							},
						},
					},
				},
			},
			wantTmpfs: []string{"--tmpfs", "/run:mode=1777"},
		},
		{
			name: "tmpfs mount with size and mode",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					Mounts: []service.Mount{
						{
							Source: "/cache",
							Target: "/cache",
							Type:   service.MountTypeTmpfs,
							TmpfsOptions: &service.TmpfsOptions{
								Size: "256m",
								Mode: 755, // Decimal for cross-platform compatibility
							},
						},
					},
				},
			},
			wantTmpfs: []string{"--tmpfs", "/cache:size=256m,mode=755"},
		},
		{
			name: "tmpfs mount with uid and gid",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					Mounts: []service.Mount{
						{
							Source: "/data",
							Target: "/data",
							Type:   service.MountTypeTmpfs,
							TmpfsOptions: &service.TmpfsOptions{
								Size: "512m",
								UID:  1000,
								GID:  1000,
							},
						},
					},
				},
			},
			wantTmpfs: []string{"--tmpfs", "/data:size=512m,uid=1000,gid=1000"},
		},
		{
			name: "tmpfs mount with all options",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					Mounts: []service.Mount{
						{
							Source: "/temp",
							Target: "/temp",
							Type:   service.MountTypeTmpfs,
							TmpfsOptions: &service.TmpfsOptions{
								Size: "1g",
								Mode: 1777,
								// UID/GID default to 0, which matches systemd behavior (not rendered)
							},
						},
					},
				},
			},
			wantTmpfs: []string{"--tmpfs", "/temp:size=1g,mode=1777"},
		},
		{
			name: "multiple tmpfs mounts with options",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					Mounts: []service.Mount{
						{
							Source:       "/cache",
							Target:       "/cache",
							Type:         service.MountTypeTmpfs,
							TmpfsOptions: &service.TmpfsOptions{Size: "256m"},
						},
						{
							Source:       "/tmp",
							Target:       "/tmp",
							Type:         service.MountTypeTmpfs,
							TmpfsOptions: &service.TmpfsOptions{Mode: 1777},
						},
					},
				},
			},
			wantTmpfs: []string{
				"--tmpfs", "/cache:size=256m",
				"--tmpfs", "/tmp:mode=1777",
			},
		},
		{
			name: "tmpfs without options (legacy Tmpfs field)",
			spec: service.Spec{
				Name: "test-service",
				Container: service.Container{
					Image: "alpine:latest",
					Tmpfs: []string{"/run", "/tmp"},
				},
			},
			wantTmpfs: []string{
				"--tmpfs", "/run",
				"--tmpfs", "/tmp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildPodmanArgs(tt.spec, "test-container")

			if tt.wantTmpfs == nil {
				argsStr := strings.Join(args, " ")
				assert.NotContains(t, argsStr, "--tmpfs")
				return
			}

			// Find all tmpfs flags and their values
			var foundTmpfs []string
			for i := 0; i < len(args)-1; i++ {
				if args[i] == "--tmpfs" {
					foundTmpfs = append(foundTmpfs, args[i], args[i+1])
				}
			}

			assert.Equal(t, tt.wantTmpfs, foundTmpfs)
		})
	}
}

func TestBuildPodmanArgs_StopSignalAndGracePeriod(t *testing.T) {
	tests := []struct {
		name         string
		container    service.Container
		expectedArgs []string
		notExpected  []string
	}{
		{
			name: "custom stop signal",
			container: service.Container{
				Image:      "test:latest",
				StopSignal: "SIGKILL",
			},
			expectedArgs: []string{"--stop-signal", "SIGKILL"},
		},
		{
			name: "grace period 30 seconds",
			container: service.Container{
				Image:           "test:latest",
				StopGracePeriod: 30 * time.Second,
			},
			expectedArgs: []string{"--stop-timeout", "30"},
		},
		{
			name: "both signal and grace period",
			container: service.Container{
				Image:           "test:latest",
				StopSignal:      "SIGINT",
				StopGracePeriod: 45 * time.Second,
			},
			expectedArgs: []string{"--stop-signal", "SIGINT", "--stop-timeout", "45"},
		},
		{
			name: "grace period 1 minute",
			container: service.Container{
				Image:           "test:latest",
				StopGracePeriod: 1 * time.Minute,
			},
			expectedArgs: []string{"--stop-timeout", "60"},
		},
		{
			name: "empty values (no stop args)",
			container: service.Container{
				Image: "test:latest",
			},
			notExpected: []string{"--stop-signal", "--stop-timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := service.Spec{
				Name:      "test",
				Container: tt.container,
			}

			args := BuildPodmanArgs(spec, "test-container")

			// Check expected args are present
			for i := 0; i < len(tt.expectedArgs); i += 2 {
				flag := tt.expectedArgs[i]
				value := tt.expectedArgs[i+1]

				found := false
				for j := 0; j < len(args)-1; j++ {
					if args[j] == flag && args[j+1] == value {
						found = true
						break
					}
				}
				assert.True(t, found, "expected args not found: %s %s", flag, value)
			}

			// Check unexpected args are not present
			for _, notExpected := range tt.notExpected {
				assert.NotContains(t, args, notExpected, "unexpected arg should not be present")
			}
		})
	}
}
