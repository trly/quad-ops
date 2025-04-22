---
title: "Systemd Service Configuration"
weight: 20
---

# Systemd Service Configuration

For production use, Quad-Ops should run as a systemd service to continuously monitor your Git repositories and update your container infrastructure. This guide covers both system-wide and user-mode (rootless) configurations.

## System-Wide Service (Root)

The system-wide service runs as root and manages containers for the entire system.

### Create the Service File

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

### Enable and Start the Service

```bash
sudo systemctl daemon-reload
sudo systemctl enable quad-ops
sudo systemctl start quad-ops
```

### View Service Status and Logs

```bash
# Check service status
sudo systemctl status quad-ops

# View logs
sudo journalctl -u quad-ops

# Follow logs in real-time
sudo journalctl -u quad-ops -f
```

## User Mode Service (Rootless)

Running in user mode allows deploying containers without root privileges, using your user's systemd session.

### Configuration for User Mode

```bash
# Create user config directory
mkdir -p ~/.config/quad-ops

# Create user config file (if not already created)
cp /etc/quad-ops/config.yaml ~/.config/quad-ops/config.yaml
# Or create a new one
tee ~/.config/quad-ops/config.yaml > /dev/null << 'EOF'
repositories:
  - name: my-apps
    url: "https://github.com/yourusername/my-apps.git"
    ref: "main"
EOF
```

### Create User Service File

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

### Enable and Start User Service

```bash
systemctl --user daemon-reload
systemctl --user enable quad-ops
systemctl --user start quad-ops
```

### View User Service Status and Logs

```bash
# Check service status
systemctl --user status quad-ops

# View logs
journalctl --user -u quad-ops

# Follow logs in real-time
journalctl --user -u quad-ops -f
```

## Advanced Service Configuration

### Setting Environment Variables

You can configure environment variables for the Quad-Ops service:

```bash
sudo tee /etc/systemd/system/quad-ops.service > /dev/null << 'EOF'
[Unit]
Description=Quad-Ops - GitOps for Podman Quadlet
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment="QUAD_OPS_CONFIG=/path/to/custom/config.yaml"
Environment="QUAD_OPS_VERBOSE=true"
ExecStart=/usr/local/bin/quad-ops sync
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
EOF
```

### Running with Increased Sync Frequency

For more frequent updates:

```bash
sudo tee /etc/systemd/system/quad-ops.service > /dev/null << 'EOF'
[Unit]
Description=Quad-Ops - GitOps for Podman Quadlet
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/quad-ops sync --interval 1m
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
EOF
```

## Troubleshooting

### Service Starts But No Units Are Created

Check the logs for Git repository access issues:

```bash
journalctl -u quad-ops | grep -i "git"
```

Common issues:
- SSH key not available for private repositories
- Repository URL incorrect
- Network connectivity problems

### Permission Denied Errors

For system-wide installation, ensure Quad-Ops has permission to access the quadlet directory:

```bash
# Check permissions
ls -la /etc/containers/systemd

# Fix permissions if needed
sudo chmod 755 /etc/containers/systemd
```

For user-mode:

```bash
# Create user quadlet directory
mkdir -p ~/.config/containers/systemd
chmod 755 ~/.config/containers/systemd
```

### Service Fails with Timeout

If the service times out during startup:

```bash
# Increase timeout
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
TimeoutStartSec=120

[Install]
WantedBy=multi-user.target
EOF
```

## LingerEnabled for User Services

To ensure user services continue running after logout:

```bash
# Enable lingering for your user
loginctl enable-linger $(whoami)

# Check lingering status
loginctl show-user $(whoami)
```