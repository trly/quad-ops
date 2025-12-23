---
title: "Installation"
weight: 10
---

# Getting Started with Quad-Ops (Manual Installation)

This guide provides step-by-step manual installation instructions for users who prefer not to use the automated installer script.

> **Quick Start Available:** For faster installation, see our [Quick Start](../quick-start/) guide using the automated installer script.

## Prerequisites

- [Podman](https://podman.io/docs/installation) 4.0+
- [Git](https://git-scm.com/downloads)
- systemd-based Linux distribution

## Manual Installation

### Option 1: Install Prebuilt Binary (Recommended)

```bash
# Download latest release (update version as needed)
VERSION=$(curl -s https://api.github.com/repos/trly/quad-ops/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
wget "https://github.com/trly/quad-ops/releases/download/${VERSION}/quad-ops_${VERSION#v}_linux_amd64.tar.gz"

# Verify checksum (optional but recommended)
wget "https://github.com/trly/quad-ops/releases/download/${VERSION}/quad-ops_${VERSION#v}_checksums.txt"
sha256sum -c "quad-ops_${VERSION#v}_checksums.txt" --ignore-missing

# Extract the binary
tar -xzf "quad-ops_${VERSION#v}_linux_amd64.tar.gz"

# Install binary to standard location
sudo mv quad-ops /usr/local/bin/
sudo chmod +x /usr/local/bin/quad-ops
sudo chown root:root /usr/local/bin/quad-ops

# Verify installation
quad-ops version
```

### Option 2: Install from Source

```bash
# Clone the repository
git clone https://github.com/trly/quad-ops.git
cd quad-ops

# Build the binary
go build -o quad-ops ./cmd/quad-ops

# Install to system directory
sudo mv quad-ops /usr/local/bin/
sudo chmod +x /usr/local/bin/quad-ops
```

### Install Configuration Files

```bash
# Create configuration directory
sudo mkdir -p /etc/quad-ops

# Download and install example configuration
wget https://raw.githubusercontent.com/trly/quad-ops/main/configs/config.yaml.example
sudo mv config.yaml.example /etc/quad-ops/

# Copy to active configuration and customize
sudo cp /etc/quad-ops/config.yaml.example /etc/quad-ops/config.yaml
```

## Basic Configuration

### Creating Your First Project

Create a Git repository with a Docker Compose file:

```bash
# Create a new directory and initialize Git
mkdir -p ~/my-quad-ops-project
cd ~/my-quad-ops-project
git init

# Create a simple Docker Compose file
cat > docker-compose.yaml << 'EOF'
services:
  web:
    image: docker.io/library/nginx:latest
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html

volumes:
  html:
EOF

# Create the necessary directories for bind mounts
mkdir -p html
echo "<h1>Hello from Quad-Ops!</h1>" > html/index.html

# Commit to Git
git add .
git commit -m "Initial commit with Nginx Docker Compose"
```

Edit your configuration file at `/etc/quad-ops/config.yaml`:

```yaml
repositories:
  - name: my-project
    url: "file:///home/yourusername/my-quad-ops-project"
```

## Running Your First Sync

```bash
# Sync repositories and write systemd unit files
sudo quad-ops sync
```

This will:
1. Clone the configured repository
2. Find the Docker Compose file
3. Convert it to Podman Quadlet units
4. Write the units to the quadlet directory

## Verifying Your Setup

### Check Running Containers

```bash
# Use podman to verify containers are running
sudo podman ps
```

## Managing

Quad-Ops creates systemd units that you can manage with standard systemd commands:

```bash
# Restart a container
sudo systemctl restart my-project-web

# Stop a container
sudo systemctl stop my-project-web

# Start a container
sudo systemctl start my-project-web

# View logs
sudo journalctl -u my-project-web
```

## Docker Compose Tips for Quad-Ops

### Best Practices

1. **Always use fully qualified image names**
   ✅ `image: docker.io/library/nginx:latest`
   ❌ `image: nginx`

2. **Create bind mount directories before syncing**
   Podman doesn't auto-create directories like Docker does.

3. **Use `depends_on` for proper startup order**
   ```yaml
   services:
     webapp:
       depends_on:
         - db
   ```

4. **Specify custom networks**
   ```yaml
   services:
     webapp:
       networks:
         - backend
   networks:
     backend:
   ```

## Success! What's Next?

Congratulations! You now have Quad-Ops manually installed and running. Here are some next steps:

1. **Create your own Docker Compose files** with your applications
2. **Push your configurations to Git** repositories for proper GitOps workflow
3. **Explore more advanced features** like secrets management and network configuration

For more detailed information, check out these guides:

- [Quick Start](../quick-start/) - Automated installation for future deployments
- [Repository Configuration](../configuration/repository-configuration/) - Repository and compose file options
- [Command Reference](../command-reference/) - Available commands and options
