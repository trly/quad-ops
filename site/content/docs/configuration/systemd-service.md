---
title: "Systemd Service"
weight: 20
---

# Systemd Service

For production use, Quad-Ops should run as a systemd service to continuously monitor your Git repositories and update your container infrastructure.

## Installation Options

| Option | Description | Use Case |
|--------|-------------|----------|
| **System-wide** | Runs as root, manages system-wide containers | Production servers |
| **User mode** | Runs as a regular user, manages rootless containers | Development environments, shared servers |

## System-Wide Service

```bash
# Download the service file from GitHub
sudo curl -L -o /etc/systemd/system/quad-ops.service \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops.service

# Enable and start the service
sudo systemctl daemon-reload
sudo systemctl enable quad-ops
sudo systemctl start quad-ops

# View logs
sudo journalctl -u quad-ops -f
```

## User Mode Service

```bash
# Create config directories
mkdir -p ~/.config/quad-ops
mkdir -p ~/.config/systemd/user/

# Download the user service file from GitHub
curl -L -o ~/.config/systemd/user/quad-ops.service \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops-user.service

# Enable lingering (required for service to run after logout)
loginctl enable-linger $(whoami)

# Enable and start the service
systemctl --user daemon-reload
systemctl --user enable quad-ops
systemctl --user start quad-ops

# View logs
journalctl --user -u quad-ops -f
```

## Template Service (for specific users)

System administrators can run quad-ops for specific users using the template service:

```bash
# Download the template service file from GitHub
sudo curl -L -o /etc/systemd/system/quad-ops@.service \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops@.service

# Enable and start for specific users
sudo systemctl daemon-reload
sudo systemctl enable quad-ops@username
sudo systemctl start quad-ops@username

# View logs for specific user
sudo journalctl -u quad-ops@username -f
```

## Service Customization

Use systemd override files to customize service behavior without modifying the original service files.

**Note:** When overriding `ExecStart`, you must first clear it with an empty `ExecStart=` line, then set the new value.

### Custom Config Path

```bash
# For system service
sudo systemctl edit quad-ops

# Add this content to the override file:
[Service]
ExecStart=
ExecStart=/usr/local/bin/quad-ops sync --daemon --config /path/to/custom/config.yaml
```

```bash
# For user service
systemctl --user edit quad-ops

# Add this content to the override file:
[Service]
ExecStart=
ExecStart=%h/.local/bin/quad-ops sync --daemon --config %h/.config/quad-ops/custom-config.yaml
```

### Custom Sync Interval

```bash
# For system service
sudo systemctl edit quad-ops

# Add this content to the override file:
[Service]
ExecStart=
ExecStart=/usr/local/bin/quad-ops sync --daemon --sync-interval 2m
```

### Enable Verbose Logging

```bash
# For system service
sudo systemctl edit quad-ops

# Add this content to the override file:
[Service]
ExecStart=
ExecStart=/usr/local/bin/quad-ops sync --daemon --verbose
```

### Apply Override Changes

After creating override files, reload systemd and restart the service:

```bash
# For system service
sudo systemctl daemon-reload
sudo systemctl restart quad-ops

# For user service
systemctl --user daemon-reload
systemctl --user restart quad-ops
```

### Required Directories

```bash
# For system-wide service
sudo mkdir -p /etc/containers/systemd
sudo chmod 755 /etc/containers/systemd

# For user-mode service
mkdir -p ~/.config/containers/systemd
chmod 755 ~/.config/containers/systemd
```