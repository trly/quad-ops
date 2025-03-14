# Quad-Ops

![Build](https://github.com/trly/quad-ops/actions/workflows/build-and-release.yml/badge.svg) ![Docs](https://github.com/trly/quad-ops/actions/workflows/docs.yaml/badge.svg)

Quad-Ops manages Quadlet container units by synchronizing them from Git repositories.
It automatically generates systemd unit files from YAML manifests and handles unit reloading.

For full documentation, visit our [GitHub Pages](https://trly.github.io/quad-ops/).

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
