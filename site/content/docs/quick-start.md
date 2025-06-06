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

### System-Wide Installation (Recommended)

Install quad-ops system-wide with root privileges:

```bash
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash
```

This installs:
- Binary: `/usr/local/bin/quad-ops`
- Config: `/etc/quad-ops/config.yaml.example`
- Services: `/etc/systemd/system/quad-ops.service` and `/etc/systemd/system/quad-ops@.service`

### User Installation

Install quad-ops for the current user only (rootless containers):

```bash
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --user
```

This installs:
- Binary: `$HOME/.local/bin/quad-ops`
- Config: `$HOME/.config/quad-ops/config.yaml.example`
- Service: `$HOME/.config/systemd/user/quad-ops.service`

## Installation Options

### Specific Version

```bash
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --version v1.2.3
```

### Custom Install Path

```bash
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --install-path /usr/local/bin
```

### Help and Options

```bash
curl -fsSL https://raw.githubusercontent.com/trly/quad-ops/main/install.sh | bash -s -- --help
```

## Post-Installation Setup

### 1. Add to PATH (if needed)

The installer will warn you if the install location isn't in your PATH:

**User install:**
```bash
echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.bashrc
source ~/.bashrc
```

Note: `/usr/local/bin` is already in PATH by default for system installations.

### 2. Configure Quad-Ops

Copy and customize the example configuration:

**System install:**
```bash
sudo cp /etc/quad-ops/config.yaml.example /etc/quad-ops/config.yaml
sudo nano /etc/quad-ops/config.yaml
```

**User install:**
```bash
cp $HOME/.config/quad-ops/config.yaml.example $HOME/.config/quad-ops/config.yaml
nano $HOME/.config/quad-ops/config.yaml
```

### 3. Your First Sync

### Test with Example Repository

Edit your config file to include the example repository:

```yaml
# Global settings
syncInterval: 5m

# Example repository
repositories:
  - name: quad-ops-examples
    url: "https://github.com/trly/quad-ops.git"
    ref: "main"
    composeDir: "examples/multi-service"
```

### Run the Sync

**System mode:**
```bash
sudo quad-ops sync
```

**User mode:**
```bash
quad-ops --user sync
```

### Verify Installation

```bash
# List managed units
quad-ops unit list

# Check running containers
podman ps
```

## Enable Automatic Syncing

### System Service

```bash
sudo systemctl enable --now quad-ops
```

### User Service

```bash
systemctl --user enable --now quad-ops
```

### Template Service (for specific users)

System administrators can run quad-ops for specific users:

```bash
sudo systemctl enable --now quad-ops@username
```

## Next Steps

🎉 **Congratulations!** Quad-Ops is now installed and running.

- **Visit your application:** If using the example, check `http://localhost:8080`
- **Create your own projects:** See [Container Management](../container-management/) for information on setting up a new repository to deploy from

## Troubleshooting

### Permission Denied

If you get permission errors, ensure you have the necessary privileges:
- System install: requires `sudo` for installation
- User install: runs without `sudo` but containers run rootless

### Path Issues

If `quad-ops` command isn't found:
1. Check the installer output for PATH warnings
2. Add the install directory to your PATH
3. Restart your shell or run `source ~/.bashrc`

### Service Issues

If systemd services fail to start:
```bash
# Check service status
systemctl status quad-ops

# View logs
journalctl -u quad-ops
```

## Alternative Installation

For users who prefer manual installation or need more control, see the [Installation](../installation/) guide for step-by-step manual instructions.
