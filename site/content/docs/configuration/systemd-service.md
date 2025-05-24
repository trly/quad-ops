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
# Create the service file
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

# Enable and start the service
sudo systemctl daemon-reload
sudo systemctl enable quad-ops
sudo systemctl start quad-ops

# View logs
sudo journalctl -u quad-ops -f
```

## User Mode Service

```bash
# Create config directory and service file
mkdir -p ~/.config/quad-ops
mkdir -p ~/.config/systemd/user/

# Create service file
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

# Enable lingering (required for service to run after logout)
loginctl enable-linger $(whoami)

# Enable and start the service
systemctl --user daemon-reload
systemctl --user enable quad-ops
systemctl --user start quad-ops

# View logs
journalctl --user -u quad-ops -f
```

## Common Configuration Options

### Custom Config Path

```ini
[Service]
Environment="QUAD_OPS_CONFIG=/path/to/custom/config.yaml"
ExecStart=/usr/local/bin/quad-ops sync
```

### Custom Sync Interval

```ini
[Service]
ExecStart=/usr/local/bin/quad-ops sync --interval 2m
```

### Enable Verbose Logging

```ini
[Service]
Environment="QUAD_OPS_VERBOSE=true"
ExecStart=/usr/local/bin/quad-ops sync
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