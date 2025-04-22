---
title: Single Service Example
---

# Single Service Example

This example demonstrates a minimal Docker Compose file with a single service, ideal for simple deployments or testing Quad-Ops.

## Docker Compose File

```yaml
version: '3.8'

services:
  webapp:
    image: docker.io/nginx:latest
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html:ro
    restart: always
    environment:
      - NGINX_HOST=example.com
```

## Generated Systemd Units

When Quad-Ops processes this Docker Compose file, it will generate the following systemd unit:

### Container Unit (myrepo-webapp.container)

```ini
[Unit]
Description=Podman container for myrepo-webapp

[Container]
Image=docker.io/nginx:latest
Volume=./html:/usr/share/nginx/html:ro
PublishPort=8080:80
Environment=NGINX_HOST=example.com

[Service]
Restart=always
TimeoutStartSec=60
TimeoutStopSec=60

[Install]
WantedBy=multi-user.target
```

## Key Points

1. **Image Name**: Note the fully qualified image name (`docker.io/nginx:latest`)
2. **Volume Path**: The bind mount path `./html` must exist before starting the container
3. **Restart Policy**: `restart: always` is translated to systemd's `Restart=always`

## Usage

1. Create the HTML directory before starting:

```bash
# Create the directory for the volume mount
mkdir -p /path/to/repo/html

# Add a test file
echo '<h1>Quad-Ops Test</h1>' > /path/to/repo/html/index.html
```

2. After syncing with Quad-Ops, manage the service:

```bash
# Start the service
systemctl start myrepo-webapp.service

# Check status
systemctl status myrepo-webapp.service

# View logs
journalctl -u myrepo-webapp.service
```

3. Access the service:

```bash
curl http://localhost:8080
```

## Common Issues

**Issue**: Container fails to start with permission errors

**Solution**: Check that the bind mount directory exists and has appropriate permissions:

```bash
chmod -R 755 /path/to/repo/html
```

**Issue**: Cannot access the service on port 8080

**Solution**: Check firewall settings and ensure the port is allowed:

```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```