# Quad-Ops

![Build](https://github.com/trly/quad-ops/actions/workflows/build.yml/badge.svg) ![Docs](https://github.com/trly/quad-ops/actions/workflows/docs.yaml/badge.svg)

Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.
It automatically generates systemd unit files from Docker Compose files and handles unit reloading.

For full documentation, visit our [GitHub Pages](https://trly.github.io/quad-ops/).

## Configuration

### Repository Settings

```yaml
repositories:
  - name: quad-ops-compose  # Repository name (required)
    url: "https://github.com/example/repo.git"  # Git repository URL (required)
    ref: "main"  # Git reference to checkout: branch, tag, or commit hash (optional)
    composeDir: "compose"  # Subdirectory where Docker Compose files are located (optional)
    cleanup: "delete"  # Cleanup policy: "delete" or "keep" (default: "keep")
```

#### Cleanup Policy

- `keep` (default): Units from this repository remain deployed even when removed from Docker Compose files
- `delete`: Units that no longer exist in the repository Docker Compose files will be stopped and removed

## Development

### Install from Source
```bash
# clone the repository
git clone https://github.com/trly/quad-ops.git

# build the binary
go build -o quad-ops main.go

# move to system directory
sudo mv quad-ops /usr/local/bin/

# copy the default config file
sudo cp config.yaml /etc/quad-ops/config.yaml

# install the systemd service file (optional)
sudo cp buildd/quad-ops.service /etc/systemd/system/quad-ops.service

# reload systemd daemon
sudo systemctl daemon-reload

# enable and start the service
sudo systemctl enable quad-ops

# start the service
sudo systemctl start quad-ops
```
