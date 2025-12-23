---
title: "Systemd Service"
weight: 20
---

# Systemd Service

For production use, Quad-Ops runs as a systemd oneshot service paired with a timer to periodically sync your Git repositories and start your container infrastructure.

## Installation Options

| Option | Description | Use Case |
|--------|-------------|----------|
| **System-wide** | Runs as root, manages system-wide containers | Production servers |
| **Template (per-user)** | Runs as a specific user, manages rootless containers | Development environments, shared servers |

## System-Wide Service

```bash
# Download the service and timer files from GitHub
sudo curl -L -o /etc/systemd/system/quad-ops.service \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops.service
sudo curl -L -o /etc/systemd/system/quad-ops.timer \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops.timer

# Enable and start the timer
sudo systemctl daemon-reload
sudo systemctl enable --now quad-ops.timer

# Run an immediate sync
sudo systemctl start quad-ops.service

# View logs
sudo journalctl -u quad-ops -f
```

## Template Service (for specific users)

System administrators can run quad-ops for specific users using the template service:

```bash
# Download the template service and timer files from GitHub
sudo curl -L -o /etc/systemd/system/quad-ops@.service \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops@.service
sudo curl -L -o /etc/systemd/system/quad-ops@.timer \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops@.timer

# Enable and start the timer for a specific user
sudo systemctl daemon-reload
sudo systemctl enable --now quad-ops@username.timer

# Run an immediate sync for a specific user
sudo systemctl start quad-ops@username.service

# View logs for specific user
sudo journalctl -u quad-ops@username -f
```

## Service Customization

Use systemd override files to customize service behavior without modifying the original unit files.

**Note:** When overriding `ExecStart`, you must first clear it with an empty `ExecStart=` line, then set the new value.

### Custom Config Path

```bash
# For system service
sudo systemctl edit quad-ops

# Add this content to the override file:
[Service]
ExecStart=
ExecStart=/usr/local/bin/quad-ops sync --config /path/to/custom/config.yaml
ExecStart=/usr/local/bin/quad-ops up --config /path/to/custom/config.yaml
```

### Enable Verbose Logging

```bash
# For system service
sudo systemctl edit quad-ops

# Add this content to the override file:
[Service]
ExecStart=
ExecStart=/usr/local/bin/quad-ops sync --verbose
ExecStart=/usr/local/bin/quad-ops up --verbose
```

### Custom Sync Interval

Override the timer frequency by editing the timer unit:

```bash
sudo systemctl edit quad-ops.timer

# Add this content to the override file:
[Timer]
OnUnitActiveSec=
OnUnitActiveSec=10min
```

### Apply Override Changes

After creating override files, reload systemd and restart the timer:

```bash
sudo systemctl daemon-reload
sudo systemctl restart quad-ops.timer
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
