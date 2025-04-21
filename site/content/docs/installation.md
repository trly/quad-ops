---
title: "Installation"
weight: 1
bookFlatSection: false
bookToc: true
bookHidden: false
bookCollapseSection: false
bookComments: false
bookSearchExclude: false
---

# Installation

This guide helps you set up and configure Quad-Ops to manage your Podman containers through Git repositories.

## Prerequisites

- Podman 4.0+ installed
- Git installed
- systemd-based Linux distribution
- Quadlet feature enabled in Podman

## Installation

### Binary Installation

Download the latest release from the [GitHub Releases page](https://github.com/trly/quad-ops/releases):

```bash
# Download the latest release (replace VERSION with the actual version)
wget https://github.com/trly/quad-ops/releases/download/vVERSION/quad-ops_Linux_x86_64.tar.gz

# Extract the archive
tar -xzf quad-ops_Linux_x86_64.tar.gz

# Move to system directory
sudo mv quad-ops /usr/local/bin/

# Make executable
sudo chmod +x /usr/local/bin/quad-ops
```

### Install from Source

```bash
# Clone the repository
git clone https://github.com/trly/quad-ops.git

# Change to the repo directory
cd quad-ops

# Build the binary
go build -o quad-ops cmd/quad-ops/main.go

# Move to system directory
sudo mv quad-ops /usr/local/bin/
```

## Configuration

Create a configuration directory and file:

```bash
# Create configuration directory
sudo mkdir -p /etc/quad-ops

# Create a basic configuration file
sudo touch /etc/quad-ops/config.yaml
```

Edit `/etc/quad-ops/config.yaml` to define your repositories:

```yaml
repositories:
  - name: example-apps  # Repository name (required)
    url: "https://github.com/example/apps.git"  # Git repository URL (required)
    ref: "main"  # Git reference to checkout (optional, defaults to default branch)
    composeDir: "compose"  # Directory containing Docker Compose files (optional)
    cleanup: "delete"  # Cleanup policy: "delete" or "keep" (optional, defaults to "keep")
```

### Cleanup Policy Options

- `keep` (default): Units from this repository remain deployed even when the compose file is removed
- `delete`: Units that no longer exist in the repository Docker Compose files will be stopped and removed

## Running as a systemd Service

Create a systemd service file:

```bash
sudo tee /etc/systemd/system/quad-ops.service > /dev/null << 'EOF'
[Unit]
Description=Quad-Ops - GitOps for Podman Quadlet
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/quad-ops sync
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
EOF
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable quad-ops
sudo systemctl start quad-ops
```

## Running in User Mode (Rootless)

For rootless operation, configure Quad-Ops to run in your user's systemd session:

```bash
# Create user config directory
mkdir -p ~/.config/quad-ops

# Create user config file
cp /etc/quad-ops/config.yaml ~/.config/quad-ops/config.yaml
```

Create a user systemd service:

```bash
mkdir -p ~/.config/systemd/user/

cat > ~/.config/systemd/user/quad-ops.service << 'EOF'
[Unit]
Description=Quad-Ops - GitOps for Podman Quadlet (User Mode)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/quad-ops sync
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=default.target
EOF
```

Enable and start the user service:

```bash
systemctl --user daemon-reload
systemctl --user enable quad-ops
systemctl --user start quad-ops
```

## Verifying Installation

Check the status of the service:

```bash
# For system-wide installation
sudo systemctl status quad-ops

# For user installation
systemctl --user status quad-ops
```

View logs:

```bash
# For system-wide installation
sudo journalctl -u quad-ops

# For user installation
journalctl --user -u quad-ops
```
