package systemd

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getValue is a helper to get a key value from the Container section.
func getValue(unit Unit, key string) string {
	section := unit.File.Section("Container")
	if section == nil {
		return ""
	}
	return section.Key(key).String()
}

// getValues is a helper to get all values (including shadows) for a key from the Container section.
func getValues(unit Unit, key string) []string {
	section := unit.File.Section("Container")
	if section == nil {
		return []string{}
	}
	k := section.Key(key)
	if k == nil {
		return []string{}
	}
	return k.ValueWithShadows()
}

// getServiceValue is a helper to get a key value from the Service section.
func getServiceValue(unit Unit, key string) string {
	section := unit.File.Section("Service")
	if section == nil {
		return ""
	}
	return section.Key(key).String()
}

// getUnitValues is a helper to get all values (including shadows) for a key from the Unit section.
func getUnitValues(unit Unit, key string) []string {
	section := unit.File.Section("Unit")
	if section == nil {
		return []string{}
	}
	k := section.Key(key)
	if k == nil {
		return []string{}
	}
	return k.ValueWithShadows()
}

// TestBuildContainer_BasicContainer tests that a simple container creates the correct unit structure.
func TestBuildContainer_BasicContainer(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "testproject-myservice.container", unit.Name)
	assert.NotNil(t, unit.File)
	assert.Equal(t, "alpine:latest", getValue(unit, "Image"))
}

// TestBuildContainer_MissingImage tests that Image is required.
func TestBuildContainer_MissingImage(t *testing.T) {
	svc := &types.ServiceConfig{}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Empty(t, getValue(unit, "Image"))
}

// TestBuildContainer_ContainerName tests that explicit container name is mapped.
func TestBuildContainer_ContainerName(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:         "alpine:latest",
		ContainerName: "my-container",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "my-container", getValue(unit, "ContainerName"))
}

// TestBuildContainer_WithEntrypoint tests that entrypoint is mapped.
func TestBuildContainer_WithEntrypoint(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:      "alpine:latest",
		Entrypoint: types.ShellCommand{"/bin/sh", "-c"},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	entrypoint := getValue(unit, "Entrypoint")
	assert.NotEmpty(t, entrypoint)
	assert.Contains(t, entrypoint, "/bin/sh")
}

// TestBuildContainer_WithCommand tests that command is mapped to Exec.
func TestBuildContainer_WithCommand(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:   "alpine:latest",
		Command: types.ShellCommand{"sleep", "infinity"},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	exec := getValue(unit, "Exec")
	assert.NotEmpty(t, exec)
	assert.Contains(t, exec, "sleep")
}

// TestBuildContainer_WithWorkingDir tests that working directory is mapped.
func TestBuildContainer_WithWorkingDir(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:      "alpine:latest",
		WorkingDir: "/app",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "/app", getValue(unit, "WorkingDir"))
}

// TestBuildContainer_WithUser tests that user is mapped.
func TestBuildContainer_WithUser(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		User:  "nobody",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "nobody", getValue(unit, "User"))
}

// TestBuildContainer_WithGroupAdd tests that group_add is mapped.
func TestBuildContainer_WithGroupAdd(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:    "alpine:latest",
		GroupAdd: []string{"sudo", "wheel"},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "Group")
	assert.Len(t, vals, 2)
	assert.Equal(t, "sudo", vals[0])
	assert.Equal(t, "wheel", vals[1])
}

// TestBuildContainer_WithHostname tests that hostname is mapped.
func TestBuildContainer_WithHostname(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:    "alpine:latest",
		Hostname: "myhost",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "myhost", getValue(unit, "HostName"))
}

// TestBuildContainer_WithDomainName tests that domain name is mapped.
func TestBuildContainer_WithDomainName(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:      "alpine:latest",
		DomainName: "example.com",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	// Note: Podman uses HostName for both hostname and domain
	assert.Contains(t, getValue(unit, "HostName"), "example.com")
}

// TestBuildContainer_WithPullPolicy tests that pull policy is mapped.
func TestBuildContainer_WithPullPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		expected string
	}{
		{"always", "always", "always"},
		{"never", "never", "never"},
		{"missing", "missing", "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &types.ServiceConfig{
				Image:      "alpine:latest",
				PullPolicy: tt.policy,
			}
			unit := BuildContainer("testproject", "myservice", svc, nil, nil)

			assert.Equal(t, tt.expected, getValue(unit, "Pull"))
		})
	}
}

// TestBuildContainer_WithLabels tests that labels are mapped with dot-notation.
func TestBuildContainer_WithLabels(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Labels: types.Labels{
			"app":     "myapp",
			"version": "1.0",
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "myapp", getValue(unit, "Label.app"))
	assert.Equal(t, "1.0", getValue(unit, "Label.version"))
}

// TestBuildContainer_WithEnvironment tests that environment variables are mapped.
func TestBuildContainer_WithEnvironment(t *testing.T) {
	fooVal := "bar"
	bazVal := "qux"
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Environment: types.MappingWithEquals{
			"FOO": &fooVal,
			"BAZ": &bazVal,
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	envVals := getValues(unit, "Environment")
	assert.Len(t, envVals, 2)
	assert.Contains(t, envVals, "FOO=bar")
	assert.Contains(t, envVals, "BAZ=qux")
}

// TestBuildContainer_WithDNS tests that DNS servers are mapped.
func TestBuildContainer_WithDNS(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		DNS: types.StringList{
			"8.8.8.8",
			"1.1.1.1",
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "DNS")
	assert.Len(t, vals, 2)
	assert.Equal(t, "8.8.8.8", vals[0])
	assert.Equal(t, "1.1.1.1", vals[1])
}

// TestBuildContainer_WithDNSSearch tests that DNS search domains are mapped.
func TestBuildContainer_WithDNSSearch(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		DNSSearch: types.StringList{
			"example.com",
			"local",
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "DNSSearch")
	assert.Len(t, vals, 2)
	assert.Equal(t, "example.com", vals[0])
	assert.Equal(t, "local", vals[1])
}

// TestBuildContainer_WithDNSOpts tests that DNS options are mapped.
func TestBuildContainer_WithDNSOpts(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		DNSOpts: []string{
			"ndots:1",
			"timeout:2",
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "DNSOption")
	assert.Len(t, vals, 2)
	assert.Equal(t, "ndots:1", vals[0])
	assert.Equal(t, "timeout:2", vals[1])
}

// TestBuildContainer_WithExtraHosts tests that extra hosts are mapped.
func TestBuildContainer_WithExtraHosts(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		ExtraHosts: types.HostsList{
			"api.example.com": []string{"192.168.1.100"},
			"db.example.com":  []string{"192.168.1.101"},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "AddHost")
	assert.Len(t, vals, 2)
	assert.NotEmpty(t, vals[0])
	assert.NotEmpty(t, vals[1])
}

// TestBuildContainer_WithPorts tests that ports are mapped.
func TestBuildContainer_WithPorts(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "nginx:latest",
		Ports: []types.ServicePortConfig{
			{
				Published: "8080",
				Target:    80,
				Protocol:  "tcp",
			},
			{
				HostIP:    "127.0.0.1",
				Published: "3306",
				Target:    3306,
				Protocol:  "tcp",
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	ports := getValues(unit, "PublishPort")
	assert.Len(t, ports, 2)
	assert.Contains(t, ports[0], "8080")
	assert.Contains(t, ports[0], "80")

	assert.Contains(t, ports[1], "3306")
	assert.Contains(t, ports[1], "127.0.0.1")
}

// TestBuildContainer_WithVolumes tests that named volumes reference Quadlet .volume units
// and bind mounts are passed through unchanged.
func TestBuildContainer_WithVolumes(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "nginx:latest",
		Volumes: []types.ServiceVolumeConfig{
			{
				Type:   "volume",
				Source: "data",
				Target: "/data",
			},
			{
				Type:   "bind",
				Source: "/host/path",
				Target: "/container/path",
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "Volume")
	assert.Len(t, vals, 2)
	assert.Contains(t, vals, "testproject-data.volume:/data:rw")
	assert.Contains(t, vals, "/host/path:/container/path:rw")
}

// TestBuildContainer_WithExternalVolume tests that external volumes use the Podman volume name.
func TestBuildContainer_WithExternalVolume(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "nginx:latest",
		Volumes: []types.ServiceVolumeConfig{
			{
				Type:   "volume",
				Source: "shared",
				Target: "/shared",
			},
		},
	}
	projectVolumes := types.Volumes{
		"shared": types.VolumeConfig{External: true, Name: "my-shared-vol"},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, projectVolumes)

	vals := getValues(unit, "Volume")
	assert.Len(t, vals, 1)
	assert.Contains(t, vals, "my-shared-vol:/shared:rw")
}

// TestBuildContainer_WithTmpfs tests that tmpfs mounts are mapped.
func TestBuildContainer_WithTmpfs(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Tmpfs: []string{"/tmp", "/run"},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "Tmpfs")
	assert.Len(t, vals, 2)
	assert.Equal(t, "/tmp", vals[0])
	assert.Equal(t, "/run", vals[1])
}

// TestBuildContainer_WithDevices tests that devices are mapped.
func TestBuildContainer_WithDevices(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Devices: []types.DeviceMapping{
			{
				Source:      "/dev/sda",
				Target:      "/dev/sda",
				Permissions: "rwm",
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "AddDevice")
	assert.Len(t, vals, 1)
	assert.NotEmpty(t, vals[0])
}

// TestBuildContainer_WithCapabilities tests that capabilities are mapped.
func TestBuildContainer_WithCapabilities(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:   "alpine:latest",
		CapAdd:  []string{"NET_ADMIN", "SYS_ADMIN"},
		CapDrop: []string{"NET_RAW"},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	addCaps := getValues(unit, "AddCapability")
	assert.Len(t, addCaps, 2)
	assert.Equal(t, "NET_ADMIN", addCaps[0])
	assert.Equal(t, "SYS_ADMIN", addCaps[1])

	dropCaps := getValues(unit, "DropCapability")
	assert.Len(t, dropCaps, 1)
	assert.Equal(t, "NET_RAW", dropCaps[0])
}

// TestBuildContainer_WithPrivileged tests that privileged mode is mapped via PodmanArgs.
func TestBuildContainer_WithPrivileged(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:      "alpine:latest",
		Privileged: true,
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "PodmanArgs")
	assert.Contains(t, vals, "--privileged")
}

// TestBuildContainer_WithSecurityOpt tests that security options are mapped to Quadlet keys.
func TestBuildContainer_WithSecurityOpt(t *testing.T) {
	tests := []struct {
		name     string
		opt      string
		key      string
		expected string
	}{
		{"label=disable", "label=disable", "SecurityLabelDisable", "true"},
		{"label:disable", "label:disable", "SecurityLabelDisable", "true"},
		{"label=nested", "label=nested", "SecurityLabelNested", "true"},
		{"label=type", "label=type:spc_t", "SecurityLabelType", "spc_t"},
		{"label=level", "label=level:s0:c1,c2", "SecurityLabelLevel", "s0:c1,c2"},
		{"label=filetype", "label=filetype:usr_t", "SecurityLabelFileType", "usr_t"},
		{"no-new-privileges", "no-new-privileges", "NoNewPrivileges", "true"},
		{"no-new-privileges:true", "no-new-privileges:true", "NoNewPrivileges", "true"},
		{"seccomp", "seccomp=/tmp/profile.json", "SeccompProfile", "/tmp/profile.json"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &types.ServiceConfig{
				Image:       "alpine:latest",
				SecurityOpt: []string{tc.opt},
			}
			unit := BuildContainer("testproject", "myservice", svc, nil, nil)

			assert.Equal(t, tc.expected, getValue(unit, tc.key))
		})
	}
}

// TestBuildContainer_WithSecurityOptMask tests that mask/unmask are mapped as shadows.
func TestBuildContainer_WithSecurityOptMask(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		SecurityOpt: []string{
			"mask=/proc/kcore:/proc/keys",
			"unmask=ALL",
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	masks := getValues(unit, "Mask")
	assert.Len(t, masks, 1)
	assert.Equal(t, "/proc/kcore:/proc/keys", masks[0])

	unmasks := getValues(unit, "Unmask")
	assert.Len(t, unmasks, 1)
	assert.Equal(t, "ALL", unmasks[0])
}

// TestBuildContainer_WithIpc tests that IPC mode is mapped.
func TestBuildContainer_WithIpc(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Ipc:   "host",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "host", getValue(unit, "Ipc"))
}

// TestBuildContainer_WithPid tests that PID mode is mapped.
func TestBuildContainer_WithPid(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Pid:   "host",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "host", getValue(unit, "Pid"))
}

// TestBuildContainer_WithNetworkMode tests that network mode is mapped.
func TestBuildContainer_WithNetworkMode(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:       "alpine:latest",
		NetworkMode: "host",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "host", getValue(unit, "Network"))
}

// TestBuildContainer_WithMultipleNetworks tests that multiple networks use bridge mode.
func TestBuildContainer_WithMultipleNetworks(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Networks: map[string]*types.ServiceNetworkConfig{
			"default": {},
			"proxy":   {},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	// Should set bridge mode for multiple networks
	assert.Equal(t, "bridge", getValue(unit, "Network"))
	// Should include all networks as shadow values
	networks := getValues(unit, "Network")
	assert.Contains(t, networks, "testproject-default.network")
	assert.Contains(t, networks, "testproject-proxy.network")
	assert.Len(t, networks, 3) // bridge mode + 2 networks
}

// TestBuildContainer_WithExternalNetwork tests that external networks use the Podman network name.
func TestBuildContainer_WithExternalNetwork(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Networks: map[string]*types.ServiceNetworkConfig{
			"default":  {},
			"external": {},
		},
	}
	projectNetworks := types.Networks{
		"default":  types.NetworkConfig{},
		"external": types.NetworkConfig{External: true, Name: "my-external-net"},
	}
	unit := BuildContainer("testproject", "myservice", svc, projectNetworks, nil)

	networks := getValues(unit, "Network")
	assert.Contains(t, networks, "testproject-default.network")
	assert.Contains(t, networks, "my-external-net")
	assert.Len(t, networks, 3) // bridge + 2 networks
}

// TestBuildContainer_WithExternalNetworkNoName tests external network without explicit name.
func TestBuildContainer_WithExternalNetworkNoName(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Networks: map[string]*types.ServiceNetworkConfig{
			"shared": {},
		},
	}
	projectNetworks := types.Networks{
		"shared": types.NetworkConfig{External: true},
	}
	unit := BuildContainer("testproject", "myservice", svc, projectNetworks, nil)

	networks := getValues(unit, "Network")
	assert.Contains(t, networks, "shared")
}

// TestBuildContainer_WithReadOnly tests that read-only filesystem is mapped.
func TestBuildContainer_WithReadOnly(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:    "alpine:latest",
		ReadOnly: true,
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "true", getValue(unit, "ReadOnly"))
}

// TestBuildContainer_WithMemory tests that memory limits are mapped.
func TestBuildContainer_WithMemory(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:          "alpine:latest",
		MemLimit:       536870912,  // 512MB
		MemSwapLimit:   1073741824, // 1GB
		MemReservation: 268435456,  // 256MB
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "536870912", getValue(unit, "Memory"))
	assert.Equal(t, "1073741824", getValue(unit, "MemorySwap"))
	assert.Equal(t, "268435456", getValue(unit, "MemoryReservation"))
}

// TestBuildContainer_WithCPU tests that CPU limits are mapped.
func TestBuildContainer_WithCPU(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:     "alpine:latest",
		CPUS:      0.5,
		CPUShares: 1024,
		CPUSet:    "0,1",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "0.5", getValue(unit, "Cpus"))
	assert.Equal(t, "1024", getValue(unit, "CpuWeight"))
	assert.Equal(t, "0,1", getValue(unit, "CpuSet"))
}

// TestBuildContainer_WithOomKillDisable tests that OOM kill disable is mapped.
func TestBuildContainer_WithOomKillDisable(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:          "alpine:latest",
		OomKillDisable: true,
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "-999", getValue(unit, "OomScoreAdj"))
}

// TestBuildContainer_WithOomScoreAdj tests that OOM score adjustment is mapped.
func TestBuildContainer_WithOomScoreAdj(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:       "alpine:latest",
		OomScoreAdj: 100,
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "100", getValue(unit, "OomScoreAdj"))
}

// TestBuildContainer_WithPidsLimit tests that PID limit is mapped.
func TestBuildContainer_WithPidsLimit(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:     "alpine:latest",
		PidsLimit: 1024,
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "1024", getValue(unit, "PidsLimit"))
}

// TestBuildContainer_WithShmSize tests that shared memory size is mapped.
func TestBuildContainer_WithShmSize(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:   "alpine:latest",
		ShmSize: 67108864, // 64MB
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "67108864", getValue(unit, "ShmSize"))
}

// TestBuildContainer_WithSysctls tests that sysctls are mapped.
func TestBuildContainer_WithSysctls(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Sysctls: map[string]string{
			"net.ipv4.ip_forward": "1",
			"kernel.shmmax":       "68719476736",
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "1", getValue(unit, "Sysctl.net.ipv4.ip_forward"))
	assert.Equal(t, "68719476736", getValue(unit, "Sysctl.kernel.shmmax"))
}

// TestBuildContainer_WithUlimits tests that ulimits are mapped.
func TestBuildContainer_WithUlimits(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Ulimits: map[string]*types.UlimitsConfig{
			"nofile": {
				Soft: 1024,
				Hard: 2048,
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "1024:2048", getValue(unit, "Ulimit.nofile"))
}

// TestBuildContainer_WithTty tests that TTY is mapped.
func TestBuildContainer_WithTty(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Tty:   true,
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "true", getValue(unit, "Tty"))
}

// TestBuildContainer_WithStdinOpen tests that stdin open is mapped.
func TestBuildContainer_WithStdinOpen(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:     "alpine:latest",
		StdinOpen: true,
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "true", getValue(unit, "Interactive"))
}

// TestBuildContainer_WithStopSignal tests that stop signal is mapped.
func TestBuildContainer_WithStopSignal(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:      "alpine:latest",
		StopSignal: "SIGTERM",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "SIGTERM", getValue(unit, "StopSignal"))
}

// TestBuildContainer_WithStopGracePeriod tests that stop grace period is mapped.
func TestBuildContainer_WithStopGracePeriod(t *testing.T) {
	dur := types.Duration(30000000000) // 30 seconds
	svc := &types.ServiceConfig{
		Image:           "alpine:latest",
		StopGracePeriod: &dur,
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "30", getValue(unit, "StopTimeout"))
}

// TestBuildContainer_WithRestart tests that restart policy is mapped to [Service] section.
func TestBuildContainer_WithRestart(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:   "alpine:latest",
		Restart: "unless-stopped",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	// Restart should be in [Service] section, not [Container]
	assert.Equal(t, "", getValue(unit, "Restart"))
	// unless-stopped maps to "always" in systemd
	assert.Equal(t, "always", getServiceValue(unit, "Restart"))
}

// TestBuildContainer_WithRestartPolicies tests various restart policy mappings.
func TestBuildContainer_WithRestartPolicies(t *testing.T) {
	tests := []struct {
		compose  string
		expected string
	}{
		{"no", "no"},
		{"always", "always"},
		{"on-failure", "on-failure"},
		{"unless-stopped", "always"},
	}
	for _, tt := range tests {
		t.Run(tt.compose, func(t *testing.T) {
			svc := &types.ServiceConfig{
				Image:   "alpine:latest",
				Restart: tt.compose,
			}
			unit := BuildContainer("testproject", "myservice", svc, nil, nil)
			assert.Equal(t, tt.expected, getServiceValue(unit, "Restart"))
		})
	}
}

// TestBuildContainer_WithInit tests that init process is mapped.
func TestBuildContainer_WithInit(t *testing.T) {
	init := true
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Init:  &init,
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "true", getValue(unit, "RunInit"))
}

// TestBuildContainer_WithLogDriver tests that log driver is mapped.
func TestBuildContainer_WithLogDriver(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:     "alpine:latest",
		LogDriver: "json-file",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "json-file", getValue(unit, "LogDriver"))
}

// TestBuildContainer_WithLogOpts tests that log options are mapped.
func TestBuildContainer_WithLogOpts(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		LogOpt: map[string]string{
			"max-size": "10m",
			"max-file": "3",
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "10m", getValue(unit, "LogOpt.max-size"))
	assert.Equal(t, "3", getValue(unit, "LogOpt.max-file"))
}

// TestBuildContainer_ExtensionGlobalArgs tests x-quad-ops-podman-args extension.
func TestBuildContainer_ExtensionGlobalArgs(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-podman-args": []interface{}{
				"--log-level=debug",
				"--log-driver=journald",
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "PodmanArgs")
	assert.Len(t, vals, 2)
	assert.Equal(t, "--log-level=debug", vals[0])
	assert.Equal(t, "--log-driver=journald", vals[1])
}

// TestBuildContainer_ExtensionContainerArgs tests x-quad-ops-container-args extension.
func TestBuildContainer_ExtensionContainerArgs(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-container-args": []interface{}{
				"--gpus=all",
				"--device-write-bps=/dev/sda:10mb",
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "PodmanArgs")
	assert.Len(t, vals, 2)
	assert.Equal(t, "--gpus=all", vals[0])
	assert.Equal(t, "--device-write-bps=/dev/sda:10mb", vals[1])
}

// TestBuildContainer_AllFieldsTogether tests a container with multiple features.
func TestBuildContainer_AllFieldsTogether(t *testing.T) {
	dur := types.Duration(30000000000) // 30 seconds
	retries := uint64(3)
	debugVal := "true"

	svc := &types.ServiceConfig{
		Image:         "myimage:latest",
		ContainerName: "my-container",
		Hostname:      "myhost",
		User:          "appuser",
		WorkingDir:    "/app",
		Labels: types.Labels{
			"app": "myapp",
		},
		Environment: types.MappingWithEquals{
			"DEBUG": &debugVal,
		},
		Ports: []types.ServicePortConfig{
			{
				Published: "8080",
				Target:    8080,
				Protocol:  "tcp",
			},
		},
		Volumes: []types.ServiceVolumeConfig{
			{
				Type:   "volume",
				Source: "data",
				Target: "/data",
			},
		},
		CapAdd:          []string{"NET_ADMIN"},
		Privileged:      false,
		MemLimit:        536870912,
		CPUS:            0.5,
		Restart:         "unless-stopped",
		StopSignal:      "SIGTERM",
		StopGracePeriod: &dur,
		HealthCheck: &types.HealthCheckConfig{
			Test:     types.HealthCheckTest{"CMD-SHELL", "curl localhost:8080"},
			Interval: &dur,
			Timeout:  &dur,
			Retries:  &retries,
		},
		Extensions: map[string]interface{}{
			"x-quad-ops-container-args": []interface{}{
				"--custom-flag",
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "myimage:latest", getValue(unit, "Image"))
	assert.Equal(t, "my-container", getValue(unit, "ContainerName"))
	assert.Equal(t, "myhost", getValue(unit, "HostName"))
	assert.Equal(t, "appuser", getValue(unit, "User"))
	assert.Equal(t, "/app", getValue(unit, "WorkingDir"))
	assert.Equal(t, "myapp", getValue(unit, "Label.app"))
	assert.Contains(t, getValues(unit, "Environment"), "DEBUG=true")
	assert.Contains(t, getValues(unit, "PublishPort")[0], "8080")
	assert.Contains(t, getValues(unit, "Volume")[0], "testproject-data.volume")
	assert.Equal(t, "NET_ADMIN", getValues(unit, "AddCapability")[0])
	assert.Equal(t, "536870912", getValue(unit, "Memory"))
	assert.Equal(t, "0.5", getValue(unit, "Cpus"))
	assert.Equal(t, "always", getServiceValue(unit, "Restart"))
	assert.Equal(t, "SIGTERM", getValue(unit, "StopSignal"))
	assert.Contains(t, getValue(unit, "HealthCmd"), "curl")
	assert.Equal(t, "--custom-flag", getValues(unit, "PodmanArgs")[0])
}

// TestBuildContainer_SectionStructure tests that the unit always has a Container section.
func TestBuildContainer_SectionStructure(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	require.NotNil(t, unit.File)
	section := unit.File.Section("Container")
	require.NotNil(t, section)
}

// TestBuildContainer_NameDerivation tests that the unit name is derived from the project and service name.
func TestBuildContainer_NameDerivation(t *testing.T) {
	tests := []struct {
		project      string
		service      string
		expectedUnit string
	}{
		{"myproject", "web", "myproject-web.container"},
		{"myproject", "db-primary", "myproject-db-primary.container"},
		{"myproject", "cache_service", "myproject-cache_service.container"},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			svc := &types.ServiceConfig{
				Image: "alpine:latest",
			}
			unit := BuildContainer(tt.project, tt.service, svc, nil, nil)
			assert.Equal(t, tt.expectedUnit, unit.Name)
		})
	}
}

// TestBuildContainer_ExtensionInvalidTypes tests that non-string items in extensions are skipped.
func TestBuildContainer_ExtensionInvalidTypes(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-container-args": []interface{}{
				"--arg1",
				123, // non-string, should be skipped
				"--arg3",
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "PodmanArgs")
	assert.Len(t, vals, 2)
	assert.Equal(t, "--arg1", vals[0])
	assert.Equal(t, "--arg3", vals[1])
}

// TestBuildContainer_SecurityOpts tests that security options are mapped.
func TestBuildContainer_SecurityOpts(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		SecurityOpt: []string{
			"apparmor=unconfined",
			"seccomp=/etc/seccomp.json",
			"no-new-privileges",
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "unconfined", getValue(unit, "AppArmor"))
	assert.Equal(t, "/etc/seccomp.json", getValue(unit, "SeccompProfile"))
	assert.Equal(t, "true", getValue(unit, "NoNewPrivileges"))
}

// TestBuildContainer_HealthCheck tests health check mapping.
func TestBuildContainer_HealthCheck(t *testing.T) {
	interval := types.Duration(10000000000) // 10 seconds
	timeout := types.Duration(5000000000)   // 5 seconds
	retries := uint64(3)
	startPeriod := types.Duration(0)

	svc := &types.ServiceConfig{
		Image: "nginx:latest",
		HealthCheck: &types.HealthCheckConfig{
			Test:        types.HealthCheckTest{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
			Interval:    &interval,
			Timeout:     &timeout,
			Retries:     &retries,
			StartPeriod: &startPeriod,
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Contains(t, getValue(unit, "HealthCmd"), "curl")
	assert.Contains(t, getValue(unit, "HealthInterval"), "10")
	assert.Contains(t, getValue(unit, "HealthTimeout"), "5")
	assert.Equal(t, "3", getValue(unit, "HealthRetries"))
}

// TestBuildContainer_WithAnnotations tests annotation support via extension.
func TestBuildContainer_WithAnnotations(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-annotations": map[string]interface{}{
				"io.podman.annotations.app":     "myapp",
				"io.podman.annotations.version": "1.0",
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "Annotation")
	assert.Len(t, vals, 2)
	assert.Contains(t, vals, "io.podman.annotations.app=myapp")
	assert.Contains(t, vals, "io.podman.annotations.version=1.0")
}

// TestBuildContainer_WithMounts tests advanced mount support via extension.
func TestBuildContainer_WithMounts(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "alpine:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-mounts": []interface{}{
				"type=tmpfs,destination=/run,mode=1777",
				"type=bind,source=/host,destination=/container,options=bind,ro",
			},
		},
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	vals := getValues(unit, "Mount")
	assert.Len(t, vals, 2)
	assert.Equal(t, "type=tmpfs,destination=/run,mode=1777", vals[0])
	assert.Equal(t, "type=bind,source=/host,destination=/container,options=bind,ro", vals[1])
}

// TestBuildContainer_EmptyOptionals tests that empty optional fields are not added.
func TestBuildContainer_EmptyOptionals(t *testing.T) {
	svc := &types.ServiceConfig{
		Image:         "alpine:latest",
		ContainerName: "",
		Hostname:      "",
		DomainName:    "",
		WorkingDir:    "",
		User:          "",
		LogDriver:     "",
		StopSignal:    "",
		NetworkMode:   "",
		PullPolicy:    "",
	}
	unit := BuildContainer("testproject", "myservice", svc, nil, nil)

	assert.Equal(t, "testproject-myservice", getValue(unit, "ContainerName"))
	assert.Empty(t, getValue(unit, "WorkingDir"))
	assert.Empty(t, getValue(unit, "User"))
	assert.Empty(t, getValue(unit, "LogDriver"))
	assert.Empty(t, getValue(unit, "StopSignal"))
	assert.Empty(t, getValue(unit, "Pull"))
}

// TestBuildContainer_WithEnvSecrets tests that x-quad-ops-env-secrets maps to Secret directives.
func TestBuildContainer_WithEnvSecrets(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "myapp:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-env-secrets": map[string]string{
				"db_password_secret": "DATABASE_PASSWORD",
				"api_key_secret":     "API_KEY",
				"jwt_secret":         "JWT_SECRET",
			},
		},
	}
	unit := BuildContainer("testproject", "api", svc, nil, nil)

	secrets := getValues(unit, "Secret")
	assert.Len(t, secrets, 3)
	assert.Contains(t, secrets, "db_password_secret,type=env,target=DATABASE_PASSWORD")
	assert.Contains(t, secrets, "api_key_secret,type=env,target=API_KEY")
	assert.Contains(t, secrets, "jwt_secret,type=env,target=JWT_SECRET")
}

// TestBuildContainer_WithEnvSecretsEmpty tests that empty env secrets don't create directives.
func TestBuildContainer_WithEnvSecretsEmpty(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "myapp:latest",
		Extensions: map[string]interface{}{
			"x-quad-ops-env-secrets": map[string]string{},
		},
	}
	unit := BuildContainer("testproject", "api", svc, nil, nil)

	secrets := getValues(unit, "Secret")
	assert.Empty(t, secrets)
}

// TestBuildContainer_NoEnvSecrets tests that services without env secrets don't create Secret directives.
func TestBuildContainer_NoEnvSecrets(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "myapp:latest",
	}
	unit := BuildContainer("testproject", "api", svc, nil, nil)

	secrets := getValues(unit, "Secret")
	assert.Empty(t, secrets)
}

// TestBuildContainer_InternalDependencies tests that internal dependencies generate Requires/After.
func TestBuildContainer_InternalDependencies(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "myapp:latest",
		Extensions: map[string]any{
			"x-quad-ops-dependencies": map[string]string{
				"db":    "service_started",
				"cache": "service_started",
			},
		},
	}
	unit := BuildContainer("myproject", "web", svc, nil, nil)

	requires := getUnitValues(unit, "Requires")
	after := getUnitValues(unit, "After")

	assert.Len(t, requires, 2)
	assert.Contains(t, requires, "myproject-db.service")
	assert.Contains(t, requires, "myproject-cache.service")

	assert.Len(t, after, 2)
	assert.Contains(t, after, "myproject-db.service")
	assert.Contains(t, after, "myproject-cache.service")
}

// TestBuildContainer_NoDependencies tests that no Unit section is created without dependencies.
func TestBuildContainer_NoDependencies(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "myapp:latest",
	}
	unit := BuildContainer("myproject", "api", svc, nil, nil)

	requires := getUnitValues(unit, "Requires")
	after := getUnitValues(unit, "After")

	assert.Empty(t, requires)
	assert.Empty(t, after)
}

// TestBuildContainer_EmptyDependencies tests that empty deps don't create Unit section.
func TestBuildContainer_EmptyDependencies(t *testing.T) {
	svc := &types.ServiceConfig{
		Image: "myapp:latest",
		Extensions: map[string]any{
			"x-quad-ops-dependencies": map[string]string{},
		},
	}
	unit := BuildContainer("myproject", "api", svc, nil, nil)

	requires := getUnitValues(unit, "Requires")
	after := getUnitValues(unit, "After")

	assert.Empty(t, requires)
	assert.Empty(t, after)
}
