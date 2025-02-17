# Quad-Ops

Quad-Ops is a service that manages Quadlet container units by synchronizing them from a Git repository. It automatically generates systemd unit files from YAML manifests.

## Features

- Git repository synchronization with support for branches, tags and commits
- YAML manifest to Quadlet unit conversion
- Runs as a systemd service
- Configuration via YAML

## Installation

1. Build the binary:

```bash
go build -o quad-ops cmd/quadlet/main.go
```

2. Create required directories:
```bash
mkdir -p /etc/quad-ops
```

3. Copy the binary, configuration file, and systemd service file:
```bash
cp quad-ops /usr/local/bin/
cp configs/config.yaml /etc/quad-ops/
cp build/package/quad-ops.service /etc/systemd/system/
```

4. Enable and start the service:
```bash
systemctl enable quad-ops
systemctl start quad-ops
```

## Configuration

The configuration file is located at `/etc/quad-ops/config.
```yaml
git:
  repo_url: "https://github.com/your-org/your-repo.git"
  target: "main"  # Branch, tag or commit hash

paths:
  manifests_dir: "./manifests" 
  quadlet_dir: "/etc/containers/systemd"
```

