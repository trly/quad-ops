package systemd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/trly/quad-ops/internal/log"
	"github.com/trly/quad-ops/internal/platform"
	"github.com/trly/quad-ops/internal/service"
	"github.com/trly/quad-ops/internal/sorting"
)

// Renderer implements platform.Renderer for systemd/Quadlet.
type Renderer struct {
	logger log.Logger
}

// NewRenderer creates a new systemd renderer.
func NewRenderer(logger log.Logger) *Renderer {
	return &Renderer{
		logger: logger,
	}
}

// Name returns the platform name.
func (r *Renderer) Name() string {
	return "systemd"
}

// Render converts service specs to systemd Quadlet unit files.
func (r *Renderer) Render(_ context.Context, specs []service.Spec) (*platform.RenderResult, error) {
	result := &platform.RenderResult{
		Artifacts:      make([]platform.Artifact, 0),
		ServiceChanges: make(map[string]platform.ChangeStatus),
	}

	for _, spec := range specs {
		artifacts, err := r.renderService(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to render service %s: %w", spec.Name, err)
		}

		artifactPaths := make([]string, 0, len(artifacts))
		contentHashes := make([]string, 0, len(artifacts))

		for _, artifact := range artifacts {
			artifactPaths = append(artifactPaths, artifact.Path)
			contentHashes = append(contentHashes, artifact.Hash)
		}

		combinedHash := r.combineHashes(contentHashes)

		result.Artifacts = append(result.Artifacts, artifacts...)
		result.ServiceChanges[spec.Name] = platform.ChangeStatus{
			Changed:       false,
			ArtifactPaths: artifactPaths,
			ContentHash:   combinedHash,
		}
	}

	return result, nil
}

// renderService renders a single service spec into one or more artifacts.
func (r *Renderer) renderService(spec service.Spec) ([]platform.Artifact, error) {
	artifacts := make([]platform.Artifact, 0)

	// Render volumes first
	for _, vol := range spec.Volumes {
		if !vol.External {
			content := r.renderVolume(vol)
			hash := r.computeHash(content)
			artifacts = append(artifacts, platform.Artifact{
				Path:    fmt.Sprintf("%s.volume", vol.Name),
				Content: []byte(content),
				Mode:    0644,
				Hash:    hash,
			})
		}
	}

	// Render networks
	for _, net := range spec.Networks {
		if !net.External {
			content := r.renderNetwork(net)
			hash := r.computeHash(content)
			artifacts = append(artifacts, platform.Artifact{
				Path:    fmt.Sprintf("%s.network", net.Name),
				Content: []byte(content),
				Mode:    0644,
				Hash:    hash,
			})
		}
	}

	// Render build unit if needed
	if spec.Container.Build != nil {
		content := r.renderBuild(spec.Name, spec.Description, *spec.Container.Build, spec.DependsOn)
		hash := r.computeHash(content)
		artifacts = append(artifacts, platform.Artifact{
			Path:    fmt.Sprintf("%s-build.build", spec.Name),
			Content: []byte(content),
			Mode:    0644,
			Hash:    hash,
		})
	}

	// Render container unit
	content := r.renderContainer(spec)
	hash := r.computeHash(content)
	artifacts = append(artifacts, platform.Artifact{
		Path:    fmt.Sprintf("%s.container", spec.Name),
		Content: []byte(content),
		Mode:    0644,
		Hash:    hash,
	})

	return artifacts, nil
}

// renderContainer renders a container service spec to a .container unit file.
func (r *Renderer) renderContainer(spec service.Spec) string {
	var builder strings.Builder

	builder.WriteString("[Unit]\n")
	if spec.Description != "" {
		builder.WriteString(formatKeyValue("Description", spec.Description))
	}

	if len(spec.DependsOn) > 0 {
		deps := make([]string, len(spec.DependsOn))
		copy(deps, spec.DependsOn)
		sort.Strings(deps)
		for _, dep := range deps {
			builder.WriteString(fmt.Sprintf("After=%s.service\n", dep))
			builder.WriteString(fmt.Sprintf("Requires=%s.service\n", dep))
		}
	}

	// Add dependencies for volumes
	if len(spec.Volumes) > 0 {
		for _, vol := range spec.Volumes {
			if !vol.External {
				builder.WriteString(fmt.Sprintf("After=%s.volume\n", vol.Name))
				builder.WriteString(fmt.Sprintf("Requires=%s.volume\n", vol.Name))
			}
		}
	}

	// Add dependencies for networks
	if len(spec.Networks) > 0 {
		for _, net := range spec.Networks {
			if !net.External {
				builder.WriteString(fmt.Sprintf("After=%s.network\n", net.Name))
				builder.WriteString(fmt.Sprintf("Requires=%s.network\n", net.Name))
			}
		}
	}

	// Add dependencies for build
	if spec.Container.Build != nil {
		builder.WriteString(fmt.Sprintf("After=%s-build.service\n", spec.Name))
		builder.WriteString(fmt.Sprintf("Requires=%s-build.service\n", spec.Name))
	}

	builder.WriteString("\n[Container]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	if spec.Container.Image != "" {
		builder.WriteString(formatKeyValue("Image", spec.Container.Image))
	}

	if spec.Container.ContainerName != "" {
		builder.WriteString(formatKeyValue("ContainerName", spec.Container.ContainerName))
	}

	if spec.Container.Hostname != "" {
		builder.WriteString(formatKeyValue("HostName", spec.Container.Hostname))
	}

	r.addEnvironment(&builder, spec.Container)
	r.addPorts(&builder, spec.Container)
	r.addMounts(&builder, spec.Container)
	r.addNetworks(&builder, spec.Container, spec)
	r.addExecution(&builder, spec.Container)
	r.addHealthcheck(&builder, spec.Container)
	r.addResources(&builder, spec.Container)
	r.addSecurity(&builder, spec.Container)
	r.addLogging(&builder, spec.Container)
	r.addSecrets(&builder, spec.Container)
	r.addAdvanced(&builder, spec.Container)

	builder.WriteString("\n[Service]\n")

	// Configure init containers as oneshot services
	if strings.Contains(spec.Name, "-init-") {
		builder.WriteString(formatKeyValue("Type", "oneshot"))
		builder.WriteString(formatKeyValue("RemainAfterExit", "yes"))
	}

	restart := r.mapRestartPolicy(spec.Container.RestartPolicy)
	builder.WriteString(formatKeyValue("Restart", restart))

	builder.WriteString("\n[Install]\n")
	builder.WriteString("WantedBy=default.target\n")

	return builder.String()
}

// addEnvironment adds environment variables and files.
func (r *Renderer) addEnvironment(builder *strings.Builder, c service.Container) {
	envKeys := sorting.GetSortedMapKeys(c.Env)
	for _, k := range envKeys {
		fmt.Fprintf(builder, "Environment=%s=%s\n", k, c.Env[k])
	}

	if len(c.EnvFiles) > 0 {
		sorted := make([]string, len(c.EnvFiles))
		copy(sorted, c.EnvFiles)
		sort.Strings(sorted)
		for _, f := range sorted {
			builder.WriteString(formatKeyValue("EnvironmentFile", f))
		}
	}
}

// addPorts adds port mappings.
func (r *Renderer) addPorts(builder *strings.Builder, c service.Container) {
	if len(c.Ports) == 0 {
		return
	}

	ports := make([]string, 0, len(c.Ports))
	for _, p := range c.Ports {
		portStr := ""
		if p.Host != "" {
			portStr = fmt.Sprintf("%s:%d:%d", p.Host, p.HostPort, p.Container)
		} else if p.HostPort > 0 {
			portStr = fmt.Sprintf("%d:%d", p.HostPort, p.Container)
		} else {
			portStr = fmt.Sprintf("%d", p.Container)
		}
		if p.Protocol != "" && p.Protocol != "tcp" {
			portStr += "/" + p.Protocol
		}
		ports = append(ports, portStr)
	}

	sort.Strings(ports)
	for _, p := range ports {
		builder.WriteString(formatKeyValue("PublishPort", p))
	}
}

// addMounts adds volume and bind mounts.
func (r *Renderer) addMounts(builder *strings.Builder, c service.Container) {
	if len(c.Mounts) == 0 {
		return
	}

	mounts := make([]string, 0, len(c.Mounts))
	for _, m := range c.Mounts {
		source := m.Source
		// Append .volume suffix for named volumes to enable automatic Quadlet dependencies
		if m.Type == service.MountTypeVolume {
			source = source + ".volume"
		}
		mountStr := fmt.Sprintf("%s:%s", source, m.Target)
		if m.ReadOnly {
			mountStr += ":ro"
		}
		mounts = append(mounts, mountStr)
	}

	sort.Strings(mounts)
	for _, m := range mounts {
		builder.WriteString(formatKeyValue("Volume", m))
	}

	if len(c.Tmpfs) > 0 {
		sorted := make([]string, len(c.Tmpfs))
		copy(sorted, c.Tmpfs)
		sort.Strings(sorted)
		for _, t := range sorted {
			builder.WriteString(formatKeyValue("Tmpfs", t))
		}
	}
}

// addNetworks adds network configuration.
func (r *Renderer) addNetworks(builder *strings.Builder, c service.Container, spec service.Spec) {
	if c.Network.Mode != "" && c.Network.Mode != "bridge" {
		builder.WriteString(formatKeyValue("Network", c.Network.Mode))
	}

	if len(c.Network.Aliases) > 0 {
		sorted := make([]string, len(c.Network.Aliases))
		copy(sorted, c.Network.Aliases)
		sort.Strings(sorted)
		for _, alias := range sorted {
			builder.WriteString(formatKeyValue("NetworkAlias", alias))
		}
	}

	// Add Network directives for service-specific networks with .network suffix
	// This enables service-to-service DNS resolution and automatic Quadlet dependencies
	if len(c.Network.ServiceNetworks) > 0 {
		sorted := make([]string, len(c.Network.ServiceNetworks))
		copy(sorted, c.Network.ServiceNetworks)
		sort.Strings(sorted)
		for _, net := range sorted {
			builder.WriteString(formatKeyValue("Network", net+".network"))
		}
	} else {
		// Fallback: Add Network directives for project-level networks with .network suffix
		// Sort networks for deterministic ordering
		networks := make([]string, 0, len(spec.Networks))
		for _, net := range spec.Networks {
			if !net.External {
				networks = append(networks, net.Name+".network")
			}
		}
		sort.Strings(networks)
		for _, net := range networks {
			builder.WriteString(formatKeyValue("Network", net))
		}
	}
}

// addExecution adds execution configuration.
func (r *Renderer) addExecution(builder *strings.Builder, c service.Container) {
	if len(c.Entrypoint) > 0 {
		builder.WriteString("Entrypoint=" + strings.Join(c.Entrypoint, " ") + "\n")
	}

	if len(c.Command) > 0 {
		builder.WriteString("Exec=" + strings.Join(c.Command, " ") + "\n")
	}

	if c.User != "" {
		builder.WriteString(formatKeyValue("User", c.User))
	}

	if c.Group != "" {
		builder.WriteString(formatKeyValue("Group", c.Group))
	}

	if c.WorkingDir != "" {
		builder.WriteString(formatKeyValue("WorkingDir", c.WorkingDir))
	}

	if c.Init {
		builder.WriteString(formatKeyValue("RunInit", "yes"))
	}

	if c.ReadOnly {
		builder.WriteString(formatKeyValue("ReadOnly", "yes"))
	}
}

// addHealthcheck adds healthcheck configuration.
func (r *Renderer) addHealthcheck(builder *strings.Builder, c service.Container) {
	if c.Healthcheck == nil {
		return
	}

	hc := c.Healthcheck
	if len(hc.Test) > 0 {
		if len(hc.Test) == 2 && (hc.Test[0] == "CMD" || hc.Test[0] == "CMD-SHELL") {
			fmt.Fprintf(builder, "HealthCmd=%s %s\n", hc.Test[0], hc.Test[1])
		} else {
			builder.WriteString("HealthCmd=" + strings.Join(hc.Test, " ") + "\n")
		}
	}

	if hc.Interval > 0 {
		builder.WriteString(formatKeyValue("HealthInterval", formatDuration(hc.Interval)))
	}

	if hc.Timeout > 0 {
		builder.WriteString(formatKeyValue("HealthTimeout", formatDuration(hc.Timeout)))
	}

	if hc.Retries > 0 {
		fmt.Fprintf(builder, "HealthRetries=%d\n", hc.Retries)
	}

	if hc.StartPeriod > 0 {
		builder.WriteString(formatKeyValue("HealthStartPeriod", formatDuration(hc.StartPeriod)))
	}

	if hc.StartInterval > 0 {
		builder.WriteString(formatKeyValue("HealthStartupInterval", formatDuration(hc.StartInterval)))
	}
}

// addResources adds resource constraints.
func (r *Renderer) addResources(builder *strings.Builder, c service.Container) {
	if c.Resources.PidsLimit > 0 {
		fmt.Fprintf(builder, "PidsLimit=%d\n", c.Resources.PidsLimit)
	}

	if c.PidsLimit > 0 {
		fmt.Fprintf(builder, "PidsLimit=%d\n", c.PidsLimit)
	}

	if len(c.Ulimits) > 0 {
		ulimits := make([]string, 0, len(c.Ulimits))
		for _, u := range c.Ulimits {
			if u.Soft == u.Hard {
				ulimits = append(ulimits, fmt.Sprintf("%s=%d", u.Name, u.Soft))
			} else {
				ulimits = append(ulimits, fmt.Sprintf("%s=%d:%d", u.Name, u.Soft, u.Hard))
			}
		}
		sort.Strings(ulimits)
		for _, u := range ulimits {
			builder.WriteString(formatKeyValue("Ulimit", u))
		}
	}

	if len(c.Sysctls) > 0 {
		keys := sorting.GetSortedMapKeys(c.Sysctls)
		for _, k := range keys {
			fmt.Fprintf(builder, "Sysctl=%s=%s\n", k, c.Sysctls[k])
		}
	}
}

// addSecurity adds security configuration.
func (r *Renderer) addSecurity(builder *strings.Builder, c service.Container) {
	if c.Security.Privileged {
		builder.WriteString(formatKeyValue("PodmanArgs", "--privileged"))
	}

	for _, cap := range c.Security.CapAdd {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cap-add=%s", cap)))
	}

	for _, cap := range c.Security.CapDrop {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cap-drop=%s", cap)))
	}

	for _, opt := range c.Security.SecurityOpt {
		builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--security-opt=%s", opt)))
	}

	if c.UserNS != "" {
		builder.WriteString(formatKeyValue("UserNS", c.UserNS))
	}
}

// addLogging adds logging configuration.
func (r *Renderer) addLogging(builder *strings.Builder, c service.Container) {
	if c.Logging.Driver != "" {
		builder.WriteString(formatKeyValue("LogDriver", c.Logging.Driver))
	}

	if len(c.Logging.Options) > 0 {
		keys := sorting.GetSortedMapKeys(c.Logging.Options)
		for _, k := range keys {
			fmt.Fprintf(builder, "LogOpt=%s=%s\n", k, c.Logging.Options[k])
		}
	}
}

// addSecrets adds secrets configuration.
func (r *Renderer) addSecrets(builder *strings.Builder, c service.Container) {
	if len(c.Secrets) == 0 && len(c.EnvSecrets) == 0 {
		return
	}

	secrets := make([]service.Secret, len(c.Secrets))
	copy(secrets, c.Secrets)
	sort.Slice(secrets, func(i, j int) bool {
		return secrets[i].Source < secrets[j].Source
	})

	for _, s := range secrets {
		secretStr := s.Source
		if s.Target != "" {
			secretStr += ",target=" + s.Target
		}
		if s.Type != "" {
			secretStr += ",type=" + s.Type
		}
		if s.UID != "" {
			secretStr += ",uid=" + s.UID
		}
		if s.GID != "" {
			secretStr += ",gid=" + s.GID
		}
		if s.Mode != "" {
			secretStr += ",mode=" + s.Mode
		}
		builder.WriteString(formatKeyValue("Secret", secretStr))
	}

	// Add environment secrets
	envSecretKeys := sorting.GetSortedMapKeys(c.EnvSecrets)
	for _, secretName := range envSecretKeys {
		envVarName := c.EnvSecrets[secretName]
		secretStr := fmt.Sprintf("%s,type=env,target=%s", secretName, envVarName)
		builder.WriteString(formatKeyValue("Secret", secretStr))
	}
}

// addAdvanced adds advanced Podman arguments.
func (r *Renderer) addAdvanced(builder *strings.Builder, c service.Container) {
	if len(c.PodmanArgs) > 0 {
		sorted := make([]string, len(c.PodmanArgs))
		copy(sorted, c.PodmanArgs)
		sort.Strings(sorted)
		for _, arg := range sorted {
			builder.WriteString(formatKeyValue("PodmanArgs", arg))
		}
	}

	if len(c.Labels) > 0 {
		keys := sorting.GetSortedMapKeys(c.Labels)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Label", fmt.Sprintf("%s=%s", k, c.Labels[k])))
		}
	}
}

// renderVolume renders a volume spec to a .volume unit file.
func (r *Renderer) renderVolume(vol service.Volume) string {
	var builder strings.Builder

	builder.WriteString("[Unit]\n")
	builder.WriteString(formatKeyValue("Description", fmt.Sprintf("Volume %s", vol.Name)))

	builder.WriteString("\n[Volume]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	if vol.Name != "" {
		builder.WriteString(formatKeyValue("VolumeName", vol.Name))
	}

	if vol.Driver != "" && vol.Driver != "local" {
		builder.WriteString(formatKeyValue("Driver", vol.Driver))
	}

	if len(vol.Options) > 0 {
		keys := sorting.GetSortedMapKeys(vol.Options)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Options", fmt.Sprintf("%s=%s", k, vol.Options[k])))
		}
	}

	if len(vol.Labels) > 0 {
		keys := sorting.GetSortedMapKeys(vol.Labels)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Label", fmt.Sprintf("%s=%s", k, vol.Labels[k])))
		}
	}

	// Add Quadlet-specific extensions if present
	if vol.Quadlet != nil {
		if len(vol.Quadlet.ContainersConfModule) > 0 {
			sorted := make([]string, len(vol.Quadlet.ContainersConfModule))
			copy(sorted, vol.Quadlet.ContainersConfModule)
			sort.Strings(sorted)
			for _, module := range sorted {
				builder.WriteString(formatKeyValue("ContainersConfModule", module))
			}
		}

		if len(vol.Quadlet.GlobalArgs) > 0 {
			sorted := make([]string, len(vol.Quadlet.GlobalArgs))
			copy(sorted, vol.Quadlet.GlobalArgs)
			sort.Strings(sorted)
			for _, arg := range sorted {
				builder.WriteString(formatKeyValue("GlobalArgs", arg))
			}
		}

		if len(vol.Quadlet.PodmanArgs) > 0 {
			sorted := make([]string, len(vol.Quadlet.PodmanArgs))
			copy(sorted, vol.Quadlet.PodmanArgs)
			sort.Strings(sorted)
			for _, arg := range sorted {
				builder.WriteString(formatKeyValue("PodmanArgs", arg))
			}
		}
	}

	builder.WriteString("\n[Install]\n")
	builder.WriteString("WantedBy=default.target\n")

	return builder.String()
}

// renderNetwork renders a network spec to a .network unit file.
func (r *Renderer) renderNetwork(net service.Network) string {
	var builder strings.Builder

	builder.WriteString("[Unit]\n")
	builder.WriteString(formatKeyValue("Description", fmt.Sprintf("Network %s", net.Name)))

	builder.WriteString("\n[Network]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	if net.Name != "" {
		builder.WriteString(formatKeyValue("NetworkName", net.Name))
	}

	if net.Driver != "" && net.Driver != "bridge" {
		builder.WriteString(formatKeyValue("Driver", net.Driver))
	}

	if net.IPAM != nil && len(net.IPAM.Config) > 0 {
		config := net.IPAM.Config[0]
		if config.Subnet != "" {
			builder.WriteString(formatKeyValue("Subnet", config.Subnet))
		}
		if config.Gateway != "" {
			builder.WriteString(formatKeyValue("Gateway", config.Gateway))
		}
		if config.IPRange != "" {
			builder.WriteString(formatKeyValue("IPRange", config.IPRange))
		}
	}

	if net.IPv6 {
		builder.WriteString(formatKeyValue("IPv6", "yes"))
	}

	if net.Internal {
		builder.WriteString(formatKeyValue("Internal", "yes"))
	}

	if len(net.Options) > 0 {
		keys := sorting.GetSortedMapKeys(net.Options)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Options", fmt.Sprintf("%s=%s", k, net.Options[k])))
		}
	}

	if len(net.Labels) > 0 {
		keys := sorting.GetSortedMapKeys(net.Labels)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Label", fmt.Sprintf("%s=%s", k, net.Labels[k])))
		}
	}

	// Add Quadlet-specific extensions if present
	if net.Quadlet != nil {
		if net.Quadlet.DisableDNS {
			builder.WriteString(formatKeyValue("DisableDNS", "yes"))
		}

		if len(net.Quadlet.DNS) > 0 {
			sorted := make([]string, len(net.Quadlet.DNS))
			copy(sorted, net.Quadlet.DNS)
			sort.Strings(sorted)
			for _, dns := range sorted {
				builder.WriteString(formatKeyValue("DNS", dns))
			}
		}

		if len(net.Quadlet.Options) > 0 {
			keys := sorting.GetSortedMapKeys(net.Quadlet.Options)
			for _, k := range keys {
				builder.WriteString(formatKeyValue("Options", fmt.Sprintf("%s=%s", k, net.Quadlet.Options[k])))
			}
		}

		if len(net.Quadlet.ContainersConfModule) > 0 {
			sorted := make([]string, len(net.Quadlet.ContainersConfModule))
			copy(sorted, net.Quadlet.ContainersConfModule)
			sort.Strings(sorted)
			for _, module := range sorted {
				builder.WriteString(formatKeyValue("ContainersConfModule", module))
			}
		}

		if len(net.Quadlet.GlobalArgs) > 0 {
			sorted := make([]string, len(net.Quadlet.GlobalArgs))
			copy(sorted, net.Quadlet.GlobalArgs)
			sort.Strings(sorted)
			for _, arg := range sorted {
				builder.WriteString(formatKeyValue("GlobalArgs", arg))
			}
		}

		if len(net.Quadlet.PodmanArgs) > 0 {
			sorted := make([]string, len(net.Quadlet.PodmanArgs))
			copy(sorted, net.Quadlet.PodmanArgs)
			sort.Strings(sorted)
			for _, arg := range sorted {
				builder.WriteString(formatKeyValue("PodmanArgs", arg))
			}
		}
	}

	builder.WriteString("\n[Install]\n")
	builder.WriteString("WantedBy=default.target\n")

	return builder.String()
}

// renderBuild renders a build spec to a .build unit file.
func (r *Renderer) renderBuild(name, description string, build service.Build, dependsOn []string) string {
	var builder strings.Builder

	builder.WriteString("[Unit]\n")
	if description != "" {
		builder.WriteString(formatKeyValue("Description", fmt.Sprintf("Build %s", description)))
	} else {
		builder.WriteString(formatKeyValue("Description", fmt.Sprintf("Build %s", name)))
	}

	if build.Context != "" {
		builder.WriteString(formatKeyValue("WorkingDirectory", build.Context))
	}

	if len(dependsOn) > 0 {
		deps := make([]string, len(dependsOn))
		copy(deps, dependsOn)
		sort.Strings(deps)
		for _, dep := range deps {
			builder.WriteString(fmt.Sprintf("After=%s.service\n", dep))
			builder.WriteString(fmt.Sprintf("Requires=%s.service\n", dep))
		}
	}

	builder.WriteString("\n[Build]\n")
	builder.WriteString(formatKeyValue("Label", "managed-by=quad-ops"))

	if len(build.Tags) > 0 {
		sorted := make([]string, len(build.Tags))
		copy(sorted, build.Tags)
		sort.Strings(sorted)
		for _, tag := range sorted {
			builder.WriteString(formatKeyValue("ImageTag", tag))
		}
	}

	if build.Dockerfile != "" {
		builder.WriteString(formatKeyValue("File", build.Dockerfile))
	}

	if build.SetWorkingDirectory != "" {
		builder.WriteString(formatKeyValue("SetWorkingDirectory", build.SetWorkingDirectory))
	}

	if build.Target != "" {
		builder.WriteString(formatKeyValue("Target", build.Target))
	}

	if build.Pull {
		builder.WriteString(formatKeyValue("Pull", "always"))
	}

	if len(build.Args) > 0 {
		keys := sorting.GetSortedMapKeys(build.Args)
		for _, k := range keys {
			fmt.Fprintf(&builder, "Environment=%s=%s\n", k, build.Args[k])
		}
	}

	if len(build.Labels) > 0 {
		keys := sorting.GetSortedMapKeys(build.Labels)
		for _, k := range keys {
			builder.WriteString(formatKeyValue("Label", fmt.Sprintf("%s=%s", k, build.Labels[k])))
		}
	}

	if len(build.Annotations) > 0 {
		sorted := make([]string, len(build.Annotations))
		copy(sorted, build.Annotations)
		sort.Strings(sorted)
		for _, a := range sorted {
			builder.WriteString(formatKeyValue("Annotation", a))
		}
	}

	if len(build.Networks) > 0 {
		sorted := make([]string, len(build.Networks))
		copy(sorted, build.Networks)
		sort.Strings(sorted)
		for _, n := range sorted {
			builder.WriteString(formatKeyValue("Network", n))
		}
	}

	if len(build.Volumes) > 0 {
		sorted := make([]string, len(build.Volumes))
		copy(sorted, build.Volumes)
		sort.Strings(sorted)
		for _, v := range sorted {
			builder.WriteString(formatKeyValue("Volume", v))
		}
	}

	if len(build.Secrets) > 0 {
		sorted := make([]string, len(build.Secrets))
		copy(sorted, build.Secrets)
		sort.Strings(sorted)
		for _, s := range sorted {
			builder.WriteString(formatKeyValue("Secret", s))
		}
	}

	if len(build.CacheFrom) > 0 {
		sorted := make([]string, len(build.CacheFrom))
		copy(sorted, build.CacheFrom)
		sort.Strings(sorted)
		for _, cache := range sorted {
			builder.WriteString(formatKeyValue("PodmanArgs", fmt.Sprintf("--cache-from=%s", cache)))
		}
	}

	if len(build.PodmanArgs) > 0 {
		sorted := make([]string, len(build.PodmanArgs))
		copy(sorted, build.PodmanArgs)
		sort.Strings(sorted)
		for _, arg := range sorted {
			builder.WriteString(formatKeyValue("PodmanArgs", arg))
		}
	}

	builder.WriteString("\n[Install]\n")
	builder.WriteString("WantedBy=default.target\n")

	return builder.String()
}

// mapRestartPolicy maps service.RestartPolicy to systemd restart value.
func (r *Renderer) mapRestartPolicy(policy service.RestartPolicy) string {
	switch policy {
	case service.RestartPolicyAlways:
		return "always"
	case service.RestartPolicyOnFailure:
		return "on-failure"
	case service.RestartPolicyUnlessStopped:
		return "always"
	case service.RestartPolicyNo:
		return "no"
	default:
		return "no"
	}
}

// formatKeyValue formats a key-value pair for unit files.
func formatKeyValue(key, value string) string {
	return fmt.Sprintf("%s=%s\n", key, value)
}

// formatDuration formats a time.Duration for systemd (e.g., "30s", "1m").
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

// computeHash computes SHA256 hash of content.
func (r *Renderer) computeHash(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// combineHashes combines multiple hashes into a single hash.
func (r *Renderer) combineHashes(hashes []string) string {
	h := sha256.New()
	for _, hash := range hashes {
		h.Write([]byte(hash))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
