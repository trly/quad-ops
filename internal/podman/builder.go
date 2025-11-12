// Package podman provides shared utilities for building podman command arguments.
package podman

import (
	"fmt"
	"sort"
	"strings"

	"github.com/trly/quad-ops/internal/service"
)

// BuildAllRunArgs converts a service.Spec into complete podman run command arguments.
// Used by launchd to generate the full plist command.
func BuildAllRunArgs(spec service.Spec, containerName string) []string {
	args := []string{"run", "--rm", "--name", containerName}

	args = appendBasicContainerArgs(args, spec.Container)
	args = appendEnvironmentArgs(args, spec.Container)
	args = appendPortArgs(args, spec.Container.Ports)
	args = appendMountArgs(args, spec.Container)
	args = appendNetworkArgs(args, &spec.Container.Network, spec.Networks)
	args = appendExtraHostsAndDNSArgs(args, spec.Container)
	args = appendLabelArgs(args, spec.Container.Labels)
	args = appendResourceArgs(args, spec.Container.Resources)
	args = appendSecurityArgs(args, spec.Container.Security)
	args = appendLimitsArgs(args, spec.Container)
	args = appendNamespaceArgs(args, spec.Container)
	args = appendDeviceArgs(args, spec.Container)
	args = appendSecretArgs(args, spec.Container)
	args = appendHealthcheckArgs(args, spec.Container.Healthcheck)
	args = appendStopConfiguration(args, spec.Container)
	args = append(args, spec.Container.PodmanArgs...)
	args = appendImageAndCommand(args, spec.Container)

	return args
}

// BuildQuadletPodmanArgs returns only args that systemd Quadlet cannot express natively.
// Used by systemd renderer to populate PodmanArgs= directives.
// Excludes: Image, ContainerName, HostName, Environment, Volume, Network, PublishPort, Memory, Health, Ulimit, Sysctl, Security, etc.
func BuildQuadletPodmanArgs(spec service.Spec) []string {
	args := make([]string, 0, 8) // Pre-allocate for typical case

	c := spec.Container

	// Security options are now handled by native Quadlet directives in addSecurity()
	// - Privileged -> SecurityLabelDisable
	// - CapAdd -> AddCapability
	// - CapDrop -> DropCapability
	// - SecurityOpt -> SecurityLabelType/SecurityLabelLevel
	// - GroupAdd -> PodmanArgs (handled in addSecurity)

	// Resource constraints that need PodmanArgs (Quadlet doesn't support all)
	// Note: Using = format for Quadlet PodmanArgs directive compatibility
	if c.Resources.MemoryReservation != "" {
		args = append(args, fmt.Sprintf("--memory-reservation=%s", c.Resources.MemoryReservation))
	}
	if c.Resources.MemorySwap != "" {
		args = append(args, fmt.Sprintf("--memory-swap=%s", c.Resources.MemorySwap))
	}
	if c.Resources.CPUShares > 0 {
		args = append(args, fmt.Sprintf("--cpu-shares=%d", c.Resources.CPUShares))
	}
	if c.Resources.CPUQuota > 0 {
		args = append(args, fmt.Sprintf("--cpu-quota=%d", c.Resources.CPUQuota))
	}
	if c.Resources.CPUPeriod > 0 {
		args = append(args, fmt.Sprintf("--cpu-period=%d", c.Resources.CPUPeriod))
	}

	// Custom PodmanArgs passthrough
	args = append(args, c.PodmanArgs...)

	return args
}

// appendBasicContainerArgs appends basic container configuration arguments.
func appendBasicContainerArgs(args []string, c service.Container) []string {
	if c.WorkingDir != "" {
		args = append(args, "-w", c.WorkingDir)
	}
	if c.User != "" {
		userArg := c.User
		if c.Group != "" {
			userArg = fmt.Sprintf("%s:%s", c.User, c.Group)
		}
		args = append(args, "-u", userArg)
	}
	if c.Hostname != "" {
		args = append(args, "--hostname", c.Hostname)
	}
	if c.ReadOnly {
		args = append(args, "--read-only")
	}
	if c.Init {
		args = append(args, "--init")
	}
	return args
}

// appendEnvironmentArgs appends environment-related arguments.
func appendEnvironmentArgs(args []string, c service.Container) []string {
	for _, envFile := range c.EnvFiles {
		args = append(args, "--env-file", envFile)
	}
	// Sort env keys for deterministic output
	if len(c.Env) > 0 {
		envKeys := make([]string, 0, len(c.Env))
		for k := range c.Env {
			envKeys = append(envKeys, k)
		}
		sort.Strings(envKeys)
		for _, k := range envKeys {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, c.Env[k]))
		}
	}
	return args
}

// appendPortArgs appends port mapping arguments.
func appendPortArgs(args []string, ports []service.Port) []string {
	for _, port := range ports {
		args = append(args, "-p", buildPortArg(port))
	}
	return args
}

// appendMountArgs appends mount and tmpfs arguments.
func appendMountArgs(args []string, c service.Container) []string {
	for _, mount := range c.Mounts {
		if mount.Type == service.MountTypeTmpfs {
			args = append(args, "--tmpfs", buildTmpfsArg(mount))
			continue
		}
		args = append(args, "-v", buildVolumeArg(mount))
	}
	for _, tmpfs := range c.Tmpfs {
		args = append(args, "--tmpfs", tmpfs)
	}
	return args
}

// appendNetworkArgs appends network configuration arguments.
func appendNetworkArgs(args []string, network *service.NetworkMode, projectNetworks []service.Network) []string {
	if network.Mode != "" {
		args = append(args, "--network", network.Mode)
	}

	networks := network.ServiceNetworks
	if len(networks) == 0 {
		networks = make([]string, 0, len(projectNetworks))
		for _, net := range projectNetworks {
			if !net.External {
				networks = append(networks, net.Name)
			}
		}
		sort.Strings(networks)
	}
	for _, net := range networks {
		args = append(args, "--network", net)
	}
	return args
}

// appendLabelArgs appends container label arguments.
func appendLabelArgs(args []string, labels map[string]string) []string {
	if len(labels) == 0 {
		return args
	}
	// Sort label keys for deterministic output
	labelKeys := make([]string, 0, len(labels))
	for k := range labels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)
	for _, k := range labelKeys {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, labels[k]))
	}
	return args
}

// appendLimitsArgs appends ulimit and sysctl arguments.
func appendLimitsArgs(args []string, c service.Container) []string {
	for _, ulimit := range c.Ulimits {
		args = append(args, "--ulimit", fmt.Sprintf("%s=%d:%d", ulimit.Name, ulimit.Soft, ulimit.Hard))
	}
	// Sort sysctl keys for deterministic output
	if len(c.Sysctls) > 0 {
		sysctlKeys := make([]string, 0, len(c.Sysctls))
		for k := range c.Sysctls {
			sysctlKeys = append(sysctlKeys, k)
		}
		sort.Strings(sysctlKeys)
		for _, k := range sysctlKeys {
			args = append(args, "--sysctl", fmt.Sprintf("%s=%s", k, c.Sysctls[k]))
		}
	}
	return args
}

// appendNamespaceArgs appends namespace mode arguments.
func appendNamespaceArgs(args []string, c service.Container) []string {
	if c.UserNS != "" {
		args = append(args, "--userns", c.UserNS)
	}
	if c.PidMode != "" {
		args = append(args, "--pid", c.PidMode)
	}
	if c.IpcMode != "" {
		args = append(args, "--ipc", c.IpcMode)
	}
	if c.CgroupMode != "" {
		args = append(args, "--cgroupns", c.CgroupMode)
	}
	if c.PidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", c.PidsLimit))
	}
	return args
}

// appendDeviceArgs appends device and device cgroup rule arguments.
func appendDeviceArgs(args []string, c service.Container) []string {
	// Sort devices for deterministic output
	if len(c.Devices) > 0 {
		devices := make([]string, len(c.Devices))
		copy(devices, c.Devices)
		sort.Strings(devices)
		for _, device := range devices {
			args = append(args, "--device", device)
		}
	}
	for _, rule := range c.DeviceCgroupRules {
		args = append(args, "--device-cgroup-rule", rule)
	}
	return args
}

// appendSecretArgs appends secret arguments.
func appendSecretArgs(args []string, c service.Container) []string {
	for _, secret := range c.Secrets {
		args = append(args, "--secret", buildSecretArg(secret))
	}
	// Sort env secrets keys for deterministic output
	if len(c.EnvSecrets) > 0 {
		secretKeys := make([]string, 0, len(c.EnvSecrets))
		for k := range c.EnvSecrets {
			secretKeys = append(secretKeys, k)
		}
		sort.Strings(secretKeys)
		for _, secretName := range secretKeys {
			args = append(args, "--secret", fmt.Sprintf("%s,type=env,target=%s", secretName, c.EnvSecrets[secretName]))
		}
	}
	return args
}

// appendImageAndCommand appends the image, entrypoint, command, and args.
func appendImageAndCommand(args []string, c service.Container) []string {
	args = append(args, c.Image)
	if len(c.Entrypoint) > 0 {
		args = append(args, "--entrypoint", c.Entrypoint[0])
		if len(c.Entrypoint) > 1 {
			args = append(args, c.Entrypoint[1:]...)
		}
	}
	args = append(args, c.Command...)
	args = append(args, c.Args...)
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

// buildTmpfsArg builds a tmpfs mount argument with options.
func buildTmpfsArg(mount service.Mount) string {
	tmpfsStr := mount.Target
	var options []string

	if mount.TmpfsOptions == nil {
		return tmpfsStr
	}

	if mount.TmpfsOptions.Size != "" {
		options = append(options, "size="+mount.TmpfsOptions.Size)
	}
	if mount.TmpfsOptions.Mode != 0 {
		// Mode is rendered as decimal for cross-platform compatibility
		options = append(options, fmt.Sprintf("mode=%d", mount.TmpfsOptions.Mode))
	}

	// Only include UID/GID if non-zero (matches systemd behavior)
	if mount.TmpfsOptions.UID != 0 {
		options = append(options, fmt.Sprintf("uid=%d", mount.TmpfsOptions.UID))
	}
	if mount.TmpfsOptions.GID != 0 {
		options = append(options, fmt.Sprintf("gid=%d", mount.TmpfsOptions.GID))
	}

	if len(options) > 0 {
		tmpfsStr += ":" + strings.Join(options, ",")
	}

	return tmpfsStr
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

// appendStopConfiguration appends stop signal and timeout arguments.
func appendStopConfiguration(args []string, c service.Container) []string {
	if c.StopSignal != "" {
		args = append(args, "--stop-signal", c.StopSignal)
	}

	if c.StopGracePeriod > 0 {
		seconds := int(c.StopGracePeriod.Seconds())
		args = append(args, "--stop-timeout", fmt.Sprintf("%d", seconds))
	}

	return args
}

// appendExtraHostsAndDNSArgs appends extra hosts and DNS-related arguments.
func appendExtraHostsAndDNSArgs(args []string, c service.Container) []string {
	// Sort extra hosts for deterministic output
	if len(c.ExtraHosts) > 0 {
		hosts := make([]string, len(c.ExtraHosts))
		copy(hosts, c.ExtraHosts)
		sort.Strings(hosts)
		for _, host := range hosts {
			args = append(args, "--add-host", host)
		}
	}

	// Sort DNS servers for deterministic output
	if len(c.DNS) > 0 {
		dns := make([]string, len(c.DNS))
		copy(dns, c.DNS)
		sort.Strings(dns)
		for _, server := range dns {
			args = append(args, "--dns", server)
		}
	}

	// Sort DNS search domains for deterministic output
	if len(c.DNSSearch) > 0 {
		search := make([]string, len(c.DNSSearch))
		copy(search, c.DNSSearch)
		sort.Strings(search)
		for _, domain := range search {
			args = append(args, "--dns-search", domain)
		}
	}

	// Sort DNS options for deterministic output
	if len(c.DNSOptions) > 0 {
		opts := make([]string, len(c.DNSOptions))
		copy(opts, c.DNSOptions)
		sort.Strings(opts)
		for _, opt := range opts {
			args = append(args, "--dns-opt", opt)
		}
	}

	return args
}
