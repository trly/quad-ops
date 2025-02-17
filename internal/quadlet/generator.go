package quadlet

import (
	"fmt"
	"log"
	"strings"
)

func GenerateQuadletUnit(unit QuadletUnit, verbose bool) string {
	if verbose {
		log.Printf("Generating Quadlet unit for %s of type %s", unit.Name, unit.Type)
	}

	// [Unit] section
	content := "[Unit]\n"
	if unit.Systemd.Description != "" {
		content += fmt.Sprintf("Description=%s\n", unit.Systemd.Description)
	}
	if unit.Systemd.Documentation != "" {
		content += fmt.Sprintf("Documentation=%s\n", unit.Systemd.Documentation)
	}
	if len(unit.Systemd.After) > 0 {
		content += fmt.Sprintf("After=%s\n", strings.Join(unit.Systemd.After, " "))
	}
	if len(unit.Systemd.Before) > 0 {
		content += fmt.Sprintf("Before=%s\n", strings.Join(unit.Systemd.Before, " "))
	}
	if len(unit.Systemd.Requires) > 0 {
		content += fmt.Sprintf("Requires=%s\n", strings.Join(unit.Systemd.Requires, " "))
	}
	if len(unit.Systemd.Wants) > 0 {
		content += fmt.Sprintf("Wants=%s\n", strings.Join(unit.Systemd.Wants, " "))
	}
	if len(unit.Systemd.Conflicts) > 0 {
		content += fmt.Sprintf("Conflicts=%s\n", strings.Join(unit.Systemd.Conflicts, " "))
	}

	// Type-specific sections
	switch unit.Type {
	case "container":
		content += "\n[Container]\n"
		if unit.Container.Image != "" {
			content += fmt.Sprintf("Image=%s\n", unit.Container.Image)
		}
		for _, label := range unit.Container.Label {
			content += fmt.Sprintf("Label=%s\n", label)
		}
		for _, port := range unit.Container.PublishPort {
			content += fmt.Sprintf("PublishPort=%s\n", port)
		}
		for k, v := range unit.Container.Environment {
			content += fmt.Sprintf("Environment=%s=%s\n", k, v)
		}
		if unit.Container.EnvironmentFile != "" {
			content += fmt.Sprintf("EnvironmentFile=%s\n", unit.Container.EnvironmentFile)
		}
		for _, vol := range unit.Container.Volume {
			content += fmt.Sprintf("Volume=%s\n", vol)
		}
		for _, net := range unit.Container.Network {
			content += fmt.Sprintf("Network=%s\n", net)
		}
		if len(unit.Container.Command) > 0 {
			content += fmt.Sprintf("Command=%s\n", strings.Join(unit.Container.Command, " "))
		}
		if len(unit.Container.Entrypoint) > 0 {
			content += fmt.Sprintf("Entrypoint=%s\n", strings.Join(unit.Container.Entrypoint, " "))
		}
		if unit.Container.User != "" {
			content += fmt.Sprintf("User=%s\n", unit.Container.User)
		}
		if unit.Container.Group != "" {
			content += fmt.Sprintf("Group=%s\n", unit.Container.Group)
		}
		if unit.Container.WorkingDir != "" {
			content += fmt.Sprintf("WorkingDir=%s\n", unit.Container.WorkingDir)
		}
		if len(unit.Container.PodmanArgs) > 0 {
			content += fmt.Sprintf("PodmanArgs=%s\n", strings.Join(unit.Container.PodmanArgs, " "))
		}
		if unit.Container.RunInit {
			content += "RunInit=yes\n"
		}
		if unit.Container.Notify {
			content += "Notify=yes\n"
		}
		if unit.Container.Privileged {
			content += "Privileged=yes\n"
		}
		if unit.Container.ReadOnly {
			content += "ReadOnly=yes\n"
		}
		for _, label := range unit.Container.SecurityLabel {
			content += fmt.Sprintf("SecurityLabel=%s\n", label)
		}
		if unit.Container.HostName != "" {
			content += fmt.Sprintf("HostName=%s\n", unit.Container.HostName)
		}
		for _, secret := range unit.Container.Secrets {
			secretOpts := []string{secret.Name}

			if secret.Type != "" {
				secretOpts = append(secretOpts, fmt.Sprintf("type=%s", secret.Type))
			}
			if secret.Target != "" {
				secretOpts = append(secretOpts, fmt.Sprintf("target=%s", secret.Target))
			}
			if secret.UID != 0 {
				secretOpts = append(secretOpts, fmt.Sprintf("uid=%d", secret.UID))
			}
			if secret.GID != 0 {
				secretOpts = append(secretOpts, fmt.Sprintf("gid=%d", secret.GID))
			}
			if secret.Mode != "" {
				secretOpts = append(secretOpts, fmt.Sprintf("mode=%s", secret.Mode))
			}

			content += fmt.Sprintf("Secret=%s\n", strings.Join(secretOpts, ","))
		}

	case "volume":
		content += "\n[Volume]\n"
		for _, label := range unit.Volume.Label {
			content += fmt.Sprintf("Label=%s\n", label)
		}
		if unit.Volume.Device != "" {
			content += fmt.Sprintf("Device=%s\n", unit.Volume.Device)
		}
		for _, opt := range unit.Volume.Options {
			content += fmt.Sprintf("Options=%s\n", opt)
		}
		if unit.Volume.UID != 0 {
			content += fmt.Sprintf("UID=%d\n", unit.Volume.UID)
		}
		if unit.Volume.GID != 0 {
			content += fmt.Sprintf("GID=%d\n", unit.Volume.GID)
		}
		if unit.Volume.Mode != "" {
			content += fmt.Sprintf("Mode=%s\n", unit.Volume.Mode)
		}
		if unit.Volume.Chown {
			content += "Chown=yes\n"
		}
		if unit.Volume.Selinux {
			content += "SELinux=yes\n"
		}
		if unit.Volume.Copy {
			content += "Copy=yes\n"
		}
		if unit.Volume.Group != "" {
			content += fmt.Sprintf("Group=%s\n", unit.Volume.Group)
		}
		if unit.Volume.Size != "" {
			content += fmt.Sprintf("Size=%s\n", unit.Volume.Size)
		}
		if unit.Volume.Capacity != "" {
			content += fmt.Sprintf("Capacity=%s\n", unit.Volume.Capacity)
		}
		if unit.Volume.Type != "" {
			content += fmt.Sprintf("Type=%s\n", unit.Volume.Type)
		}

	case "network":
		content += "\n[Network]\n"
		for _, label := range unit.Network.Label {
			content += fmt.Sprintf("Label=%s\n", label)
		}
		if unit.Network.Driver != "" {
			content += fmt.Sprintf("Driver=%s\n", unit.Network.Driver)
		}
		if unit.Network.Gateway != "" {
			content += fmt.Sprintf("Gateway=%s\n", unit.Network.Gateway)
		}
		if unit.Network.IPRange != "" {
			content += fmt.Sprintf("IPRange=%s\n", unit.Network.IPRange)
		}
		if unit.Network.Subnet != "" {
			content += fmt.Sprintf("Subnet=%s\n", unit.Network.Subnet)
		}
		if unit.Network.IPv6 {
			content += "IPv6=yes\n"
		}
		if unit.Network.Internal {
			content += "Internal=yes\n"
		}
		if unit.Network.DNSEnabled {
			content += "DNSEnabled=yes\n"
		}
		for _, opt := range unit.Network.Options {
			content += fmt.Sprintf("Options=%s\n", opt)
		}

	case "image":
		content += "\n[Image]\n"
		if unit.Image.Image != "" {
			content += fmt.Sprintf("Image=%s\n", unit.Image.Image)
		}
		if len(unit.Image.PodmanArgs) > 0 {
			content += fmt.Sprintf("PodmanArgs=%s\n", strings.Join(unit.Image.PodmanArgs, " "))
		}
	}

	// [Service] section
	content += "\n[Service]\n"
	if unit.Systemd.Type != "" {
		content += fmt.Sprintf("Type=%s\n", unit.Systemd.Type)
	}
	if unit.Systemd.RestartPolicy != "" {
		content += fmt.Sprintf("Restart=%s\n", unit.Systemd.RestartPolicy)
	}
	if unit.Systemd.TimeoutStartSec != 0 {
		content += fmt.Sprintf("TimeoutStartSec=%d\n", unit.Systemd.TimeoutStartSec)
	}
	if unit.Systemd.RemainAfterExit {
		content += "RemainAfterExit=yes\n"
	}

	return content
}
