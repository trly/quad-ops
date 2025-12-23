---
title: "Quick Start"
weight: 5
---

# Quick Start with Quad-Ops

Get Quad-Ops running in under 5 minutes using our automated installer script.

## Prerequisites

- [Podman](https://podman.io/docs/installation) 4.0+
- [Git](https://git-scm.com/downloads)
- systemd-based Linux distribution
- `curl`, `tar`, `sha256sum` (usually pre-installed)

## One-Line Installation

Install quad-ops system-wide with root privileges:

```bash
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash
```

**Installation includes:**
- Binary: `/usr/local/bin/quad-ops`
- Config: `/etc/quad-ops/config.yaml.example`

### Installation Options

#### Specific Version

```bash
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --version v1.2.3
```

#### Custom Install Path

```bash
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --install-path /usr/local/bin
```

#### Help and Options

```bash
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --help
```

## Post-Installation Setup

### 1. Configure Quad-Ops

Copy and customize the example configuration:

```bash
sudo cp /etc/quad-ops/config.yaml.example /etc/quad-ops/config.yaml
sudo nano /etc/quad-ops/config.yaml
```

### 2. Validate Your Configuration

Before deploying, validate your Docker Compose files to catch any configuration issues:

```bash
# Validate local compose files
quad-ops validate /path/to/your/compose/files

# Or validate a single compose file
quad-ops validate docker-compose.yml
```

This step ensures:
- Your compose files are valid
- Project and service names follow [naming conventions](../configuration/repository-configuration/#naming-conventions)
- Quad-ops extensions are compatible
- Security requirements are met

### 3. Your First Sync

Edit your config file to include the example repository:

```yaml
repositories:
  - name: quad-ops-examples
    url: "https://github.com/trly/quad-ops.git"
    ref: "main"
    composeDir: "examples/multi-service"
```

### Sync repositories

```bash
# Sync repositories and write systemd unit files
sudo quad-ops sync
```

### Verify Installation

```bash
# Check running containers
podman ps
```

## Next Steps

🎉 **Congratulations!** Quad-Ops is now installed and running.

- **Visit your application:** If using the example, check `http://localhost:8080`
- **Create your own projects:** See [Repository Configuration](../configuration/repository-configuration/) for information on setting up a new repository to deploy from

## Troubleshooting

### Permission Denied

If you get permission errors, ensure you have the necessary privileges:
- System install requires `sudo` for installation and running commands

### Path Issues

If `quad-ops` command isn't found:
1. Check the installer output for PATH warnings
2. Add the install directory to your PATH
3. Restart your shell or run `source ~/.bashrc`

### Service Issues

If containers fail to start, check systemd status for the generated units:
```bash
# Check service status
systemctl status <unit-name>

# View logs
journalctl -u <unit-name>
```

## Alternative Installation

For users who prefer manual installation or need more control, see the [Installation](../installation/) guide for step-by-step manual instructions.
