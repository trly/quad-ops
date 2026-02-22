---
title: "Systemd Timer"
weight: 20
---

# Systemd Timer

Quad-Ops ships systemd unit files in [`build/package/`](https://github.com/trly/quad-ops/tree/main/build/package) that run it as a oneshot service triggered by a timer. On each timer tick the service runs `quad-ops sync`, which pulls the latest Git configuration, generates Quadlet units, pre-pulls container images, and starts services.

## Unit Files

### `quad-ops.service` — System-Wide Service

```ini
[Unit]
Description=Quad-Ops Container Manager
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/quad-ops sync
```

Runs as root. The `oneshot` type means systemd waits for the command to finish. The service requires network connectivity via the `network-online.target` dependency.

### `quad-ops.timer` — System-Wide Timer

```ini
[Unit]
Description=Quad-Ops Sync Timer

[Timer]
OnBootSec=1min
OnUnitActiveSec=5min
Persistent=true

[Install]
WantedBy=timers.target
```

| Directive | Description |
|-----------|-------------|
| `OnBootSec=1min` | First run 1 minute after boot |
| `OnUnitActiveSec=5min` | Subsequent runs every 5 minutes |
| `Persistent=true` | If a scheduled run was missed (e.g. system was off), run immediately on next boot |

### `quad-ops@.service` — Template Service (Per-User)

```ini
[Unit]
Description=Quad-Ops Container Manager for %i
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
User=%i
ExecStart=/usr/local/bin/quad-ops sync
```

A systemd template unit — the `%i` specifier is replaced by the instance name (the part after `@`). When enabled as `quad-ops@alice.service`, the service runs as user `alice` and manages that user's rootless containers.

### `quad-ops@.timer` — Template Timer (Per-User)

```ini
[Unit]
Description=Quad-Ops Sync Timer for %i

[Timer]
OnBootSec=1min
OnUnitActiveSec=5min
Persistent=true

[Install]
WantedBy=timers.target
```

Same schedule as the system-wide timer but associated with a specific user's service instance.

## Installation

### System-Wide

```bash
sudo curl -L -o /etc/systemd/system/quad-ops.service \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops.service
sudo curl -L -o /etc/systemd/system/quad-ops.timer \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops.timer

sudo systemctl daemon-reload
sudo systemctl enable --now quad-ops.timer
```

### Per-User (Template)

```bash
sudo curl -L -o /etc/systemd/system/quad-ops@.service \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops@.service
sudo curl -L -o /etc/systemd/system/quad-ops@.timer \
  https://raw.githubusercontent.com/trly/quad-ops/main/build/package/quad-ops@.timer

sudo systemctl daemon-reload
sudo systemctl enable --now quad-ops@username.timer
```

Replace `username` with the target user account.

## Managing the Timer

```bash
# Check timer status
sudo systemctl status quad-ops.timer

# List all active timers
systemctl list-timers --all | grep quad-ops

# Trigger an immediate run without waiting for the timer
sudo systemctl start quad-ops.service

# View service logs
sudo journalctl -u quad-ops -f

# For a template instance
sudo systemctl status quad-ops@username.timer
sudo systemctl start quad-ops@username.service
sudo journalctl -u quad-ops@username -f
```

## Customization

Use systemd drop-in overrides to change behavior without editing the shipped unit files.

### Custom Config Path

```bash
sudo systemctl edit quad-ops

# Add to the override file:
[Service]
ExecStart=
ExecStart=/usr/local/bin/quad-ops sync --config /path/to/config.yaml
```

{{< hint warning >}}
When overriding `ExecStart` in a `oneshot` service, you must first clear it with an empty `ExecStart=` line, then provide the new command.
{{< /hint >}}

### Verbose Logging

```bash
sudo systemctl edit quad-ops

[Service]
ExecStart=
ExecStart=/usr/local/bin/quad-ops sync --verbose
```

### Custom Timer Interval

```bash
sudo systemctl edit quad-ops.timer

[Timer]
OnBootSec=
OnBootSec=2min
OnUnitActiveSec=
OnUnitActiveSec=10min
```

After any override change, reload and restart:

```bash
sudo systemctl daemon-reload
sudo systemctl restart quad-ops.timer
```
