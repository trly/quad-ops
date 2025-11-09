//go:build darwin

package launchd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/trly/quad-ops/internal/service"
)

// BuildPodmanArgs converts a service.Spec into podman run command arguments.
func BuildPodmanArgs(spec service.Spec, containerName string) []string {
	args := []string{"run"}

	// Always use --rm to avoid name collisions on restart
	args = append(args, "--rm")

	// Container name
	args = append(args, "--name", containerName)

	// Detach is not used - launchd manages the process lifecycle
	// Do NOT use -d/--detach

	// Working directory
	if spec.Container.WorkingDir != "" {
		args = append(args, "-w", spec.Container.WorkingDir)
	}

	// User and group
	if spec.Container.User != "" {
		userArg := spec.Container.User
		if spec.Container.Group != "" {
			userArg = fmt.Sprintf("%s:%s", spec.Container.User, spec.Container.Group)
		}
		args = append(args, "-u", userArg)
	}

	// Hostname
	if spec.Container.Hostname != "" {
		args = append(args, "--hostname", spec.Container.Hostname)
	}

	// Read-only root filesystem
	if spec.Container.ReadOnly {
		args = append(args, "--read-only")
	}

	// Init
	if spec.Container.Init {
		args = append(args, "--init")
	}

	// Environment files
	for _, envFile := range spec.Container.EnvFiles {
		args = append(args, "--env-file", envFile)
	}

	// Environment variables
	for k, v := range spec.Container.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Port mappings
	for _, port := range spec.Container.Ports {
		portArg := buildPortArg(port)
		args = append(args, "-p", portArg)
	}

	// Mounts
	for _, mount := range spec.Container.Mounts {
		volumeArg := buildVolumeArg(mount)
		args = append(args, "-v", volumeArg)
	}

	// Tmpfs mounts
	for _, tmpfs := range spec.Container.Tmpfs {
		args = append(args, "--tmpfs", tmpfs)
	}

	// Network mode (primary network, typically bridge)
	if spec.Container.Network.Mode != "" {
		args = append(args, "--network", spec.Container.Network.Mode)
	}

	// Service-level networks (additional networks the service joins for DNS resolution)
	// Sort networks for deterministic ordering
	networks := spec.Container.Network.ServiceNetworks
	if len(networks) == 0 {
		// Fallback to project-level networks for backward compatibility
		networks = make([]string, 0, len(spec.Networks))
		for _, net := range spec.Networks {
			if !net.External {
				networks = append(networks, net.Name)
			}
		}
		sort.Strings(networks)
	}
	for _, net := range networks {
		args = append(args, "--network", net)
	}

	// Container labels
	for k, v := range spec.Container.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	// Resources
	args = appendResourceArgs(args, spec.Container.Resources)

	// Security
	args = appendSecurityArgs(args, spec.Container.Security)

	// Ulimits
	for _, ulimit := range spec.Container.Ulimits {
		ulimitArg := fmt.Sprintf("%s=%d:%d", ulimit.Name, ulimit.Soft, ulimit.Hard)
		args = append(args, "--ulimit", ulimitArg)
	}

	// Sysctls
	for k, v := range spec.Container.Sysctls {
		args = append(args, "--sysctl", fmt.Sprintf("%s=%s", k, v))
	}

	// User namespace
	if spec.Container.UserNS != "" {
		args = append(args, "--userns", spec.Container.UserNS)
	}

	// Namespace modes
	if spec.Container.PidMode != "" {
		args = append(args, "--pid", spec.Container.PidMode)
	}
	if spec.Container.IpcMode != "" {
		args = append(args, "--ipc", spec.Container.IpcMode)
	}
	if spec.Container.CgroupMode != "" {
		args = append(args, "--cgroupns", spec.Container.CgroupMode)
	}

	// PIDs limit
	if spec.Container.PidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", spec.Container.PidsLimit))
	}

	// Secrets (Podman-specific feature)
	for _, secret := range spec.Container.Secrets {
		secretArg := buildSecretArg(secret)
		args = append(args, "--secret", secretArg)
	}

	// Environment secrets (Podman-specific feature)
	for secretName, envVarName := range spec.Container.EnvSecrets {
		secretArg := fmt.Sprintf("%s,type=env,target=%s", secretName, envVarName)
		args = append(args, "--secret", secretArg)
	}

	// Healthcheck
	args = appendHealthcheckArgs(args, spec.Container.Healthcheck)

	// Additional Podman arguments
	args = append(args, spec.Container.PodmanArgs...)

	// Image
	args = append(args, spec.Container.Image)

	// Entrypoint override
	if len(spec.Container.Entrypoint) > 0 {
		args = append(args, "--entrypoint", spec.Container.Entrypoint[0])
		if len(spec.Container.Entrypoint) > 1 {
			args = append(args, spec.Container.Entrypoint[1:]...)
		}
	}

	// Command
	args = append(args, spec.Container.Command...)

	// Additional arguments
	args = append(args, spec.Container.Args...)

	return args
}

// buildPortArg builds a port mapping argument.
func buildPortArg(port service.Port) string {
	protocol := port.Protocol
	if protocol == "" {
		protocol = "tcp"
	}

	if port.Host != "" {
		return fmt.Sprintf("%s:%d:%d/%s", port.Host, port.HostPort, port.Container, protocol)
	}
	return fmt.Sprintf("%d:%d/%s", port.HostPort, port.Container, protocol)
}

// buildVolumeArg builds a volume mount argument.
func buildVolumeArg(mount service.Mount) string {
	var parts []string
	parts = append(parts, mount.Source, mount.Target)

	var opts []string
	if mount.ReadOnly {
		opts = append(opts, "ro")
	}

	// Add bind options
	if mount.BindOptions != nil {
		if mount.BindOptions.Propagation != "" {
			opts = append(opts, mount.BindOptions.Propagation)
		}
		if mount.BindOptions.SELinux != "" {
			opts = append(opts, mount.BindOptions.SELinux)
		}
	}

	// Add custom options
	for k, v := range mount.Options {
		if v == "" {
			opts = append(opts, k)
		} else {
			opts = append(opts, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if len(opts) > 0 {
		parts = append(parts, strings.Join(opts, ","))
	}

	return strings.Join(parts, ":")
}

// buildSecretArg builds a secret argument (Podman-specific).
func buildSecretArg(secret service.Secret) string {
	arg := secret.Source

	var opts []string
	if secret.Target != "" {
		opts = append(opts, fmt.Sprintf("target=%s", secret.Target))
	}
	if secret.UID != "" {
		opts = append(opts, fmt.Sprintf("uid=%s", secret.UID))
	}
	if secret.GID != "" {
		opts = append(opts, fmt.Sprintf("gid=%s", secret.GID))
	}
	if secret.Mode != "" {
		opts = append(opts, fmt.Sprintf("mode=%s", secret.Mode))
	}
	if secret.Type != "" {
		opts = append(opts, fmt.Sprintf("type=%s", secret.Type))
	}

	if len(opts) > 0 {
		arg = fmt.Sprintf("%s,%s", arg, strings.Join(opts, ","))
	}

	return arg
}

// appendResourceArgs appends resource constraint arguments.
func appendResourceArgs(args []string, res service.Resources) []string {
	if res.Memory != "" {
		args = append(args, "--memory", res.Memory)
	}
	if res.MemoryReservation != "" {
		args = append(args, "--memory-reservation", res.MemoryReservation)
	}
	if res.MemorySwap != "" {
		args = append(args, "--memory-swap", res.MemorySwap)
	}
	if res.ShmSize != "" {
		args = append(args, "--shm-size", res.ShmSize)
	}
	if res.CPUShares > 0 {
		args = append(args, "--cpu-shares", fmt.Sprintf("%d", res.CPUShares))
	}
	if res.CPUQuota > 0 {
		args = append(args, "--cpu-quota", fmt.Sprintf("%d", res.CPUQuota))
	}
	if res.CPUPeriod > 0 {
		args = append(args, "--cpu-period", fmt.Sprintf("%d", res.CPUPeriod))
	}
	if res.PidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", res.PidsLimit))
	}
	return args
}

// appendSecurityArgs appends security-related arguments.
func appendSecurityArgs(args []string, sec service.Security) []string {
	if sec.Privileged {
		args = append(args, "--privileged")
	}
	for _, cap := range sec.CapAdd {
		args = append(args, "--cap-add", cap)
	}
	for _, cap := range sec.CapDrop {
		args = append(args, "--cap-drop", cap)
	}
	for _, opt := range sec.SecurityOpt {
		args = append(args, "--security-opt", opt)
	}
	if sec.ReadonlyRootfs {
		args = append(args, "--read-only")
	}
	if sec.SELinuxType != "" {
		args = append(args, "--security-opt", fmt.Sprintf("label=type:%s", sec.SELinuxType))
	}
	if sec.AppArmorProfile != "" {
		args = append(args, "--security-opt", fmt.Sprintf("apparmor=%s", sec.AppArmorProfile))
	}
	if sec.SeccompProfile != "" {
		args = append(args, "--security-opt", fmt.Sprintf("seccomp=%s", sec.SeccompProfile))
	}
	for _, group := range sec.GroupAdd {
		args = append(args, "--group-add", group)
	}
	return args
}

// appendHealthcheckArgs appends healthcheck arguments.
func appendHealthcheckArgs(args []string, hc *service.Healthcheck) []string {
	if hc == nil {
		return args
	}

	if len(hc.Test) > 0 {
		testCmd := strings.Join(hc.Test, " ")
		args = append(args, "--health-cmd", testCmd)
	}
	if hc.Interval > 0 {
		args = append(args, "--health-interval", hc.Interval.String())
	}
	if hc.Timeout > 0 {
		args = append(args, "--health-timeout", hc.Timeout.String())
	}
	if hc.Retries > 0 {
		args = append(args, "--health-retries", fmt.Sprintf("%d", hc.Retries))
	}
	if hc.StartPeriod > 0 {
		args = append(args, "--health-start-period", hc.StartPeriod.String())
	}

	return args
}
