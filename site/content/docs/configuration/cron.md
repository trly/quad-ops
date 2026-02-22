---
title: "Cron"
weight: 25
---

# Cron

As an alternative to [systemd timers](systemd-service), Quad-Ops can be scheduled with cron.

## System-Wide (root)

```bash
sudo crontab -e
```

```cron
# Sync repositories and start services every 5 minutes
*/5 * * * * /usr/local/bin/quad-ops sync
```

## User Mode (rootless)

```bash
crontab -e
```

```cron
# Sync repositories and start services every 5 minutes
*/5 * * * * /usr/local/bin/quad-ops sync
```

Quad-Ops automatically detects user mode and uses the appropriate default paths (`~/.local/share/quad-ops`, `~/.config/containers/systemd`).

## Custom Config Path

```cron
*/5 * * * * /usr/local/bin/quad-ops sync --config /path/to/config.yaml
```

## Logging Output

Redirect output to a log file for troubleshooting:

```cron
*/5 * * * * /usr/local/bin/quad-ops sync >> /var/log/quad-ops.log 2>&1
```
