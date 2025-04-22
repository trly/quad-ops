---
title: "Getting Started"
weight: 5
---

# Getting Started with Quad-Ops

This guide walks you through setting up Quad-Ops and syncing your first Docker Compose file in under 10 minutes.

## Prerequisites

- [Podman](https://podman.io/docs/installation) 4.0+
- [Git](https://git-scm.com/downloads)
- systemd-based Linux distribution

## Installation

### Option 1: Install Prebuilt Binary (Recommended)

```bash
# Download latest release (update version as needed)
wget https://github.com/trly/quad-ops/releases/latest/download/quad-ops_linux_amd64.tar.gz

# Extract the binary
tar -xzf quad-ops_linux_amd64.tar.gz

# Move to system directory
sudo mv quad-ops /usr/local/bin/
sudo chmod +x /usr/local/bin/quad-ops

# Verify installation
quad-ops --version
```

### Option 2: Install from Source

```bash
# Clone the repository
git clone https://github.com/trly/quad-ops.git
cd quad-ops

# Build the binary
go build -o quad-ops cmd/quad-ops/main.go

# Move to system directory
sudo mv quad-ops /usr/local/bin/
```

## Basic Configuration

### Creating Your First Project

```bash
# Create directories
sudo mkdir -p /etc/quad-ops /etc/containers/systemd
sudo chmod 755 /etc/containers/systemd
```

Create a basic config file at `/etc/quad-ops/config.yaml`:

```yaml
# Global settings
syncInterval: 5m 

# Sample repository using quad-ops examples
repositories:
  - name: quad-ops
    url: "https://github.com/trly/quad-ops.git"
    ref: "main"
    composeDir: "examples"
```

## Running Your First Sync

```bash
# Perform the initial synchronization
sudo quad-ops sync
```

This will:
1. Clone the quad-ops repository
2. Find the Docker Compose file in the examples directory
3. Convert it to Podman Quadlet units
4. Load the units into systemd
5. Start the containers

## Verifying Your Setup

### Check Quad-Ops Units

```bash
# List all units managed by quad-ops
sudo quad-ops unit list -t all
```

You should see output similar to:

```
ID  Name                                Type       Unit State  SHA1                                      Cleanup Policy  Created At
1   quad-ops-multi-service-db           container  active      c79f25a54e5aca33d8bdf7e4b4776969959aa4b4  keep            2025-04-21 22:45:15 +0000 UTC
2   quad-ops-multi-service-webapp       container  active      106a63b255e897348957b4b2cee17a6e9e4d0e00  keep            2025-04-21 22:45:15 +0000 UTC
3   quad-ops-multi-service-db-data      volume     active      05763d60c00d6ef3f4f8a026083877eb6545c48b  keep            2025-04-21 22:45:15 +0000 UTC
4   quad-ops-multi-service-wp-content   volume     active      05763d60c00d6ef3f4f8a026083877eb6545c48b  keep            2025-04-21 22:45:15 +0000 UTC
5   quad-ops-multi-service-app-network  network    active      479a643178b4bb4d2fdd8d6193c749e34c35ce83  keep            2025-04-21 22:45:15 +0000 UTC
```

### Check Running Containers

```bash
# Use podman to verify containers are running
sudo podman ps
```

You should see WordPress and MariaDB containers running:

```
CONTAINER ID  IMAGE                               COMMAND               CREATED      STATUS      PORTS                 NAMES
a31ba0448047  docker.io/library/mariadb:latest    mariadbd              3 hours ago  Up 3 hours  3306/tcp              quad-ops-multi-service-db
731cd5df42ff  docker.io/library/wordpress:latest  apache2-foregroun...  3 hours ago  Up 3 hours  0.0.0.0:8080->80/tcp  quad-ops-multi-service-webapp
```

### Accessing Your Application

Open your browser and navigate to `http://localhost:8080` to see the WordPress site from the example.

## Managing Services

Quad-Ops creates systemd units that you can manage with standard systemd commands:

```bash
# Restart a container
sudo systemctl restart quad-ops-multi-service-webapp

# Stop a container
sudo systemctl stop quad-ops-multi-service-webapp

# Start a container
sudo systemctl start quad-ops-multi-service-webapp

# View logs
sudo journalctl -u quad-ops-multi-service-webapp
```

## Using Your Own Docker Compose Files

### 1. Creating Your First Project Repository

Create a Git repository with a Docker Compose file:

```bash
# Create a new directory and initialize Git
mkdir -p ~/my-quad-ops-project
cd ~/my-quad-ops-project
git init

# Create a simple Docker Compose file
cat > docker-compose.yaml << 'EOF'
version: '3.8'

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

### 2. Update Quad-Ops Configuration

Add your new repository to `/etc/quad-ops/config.yaml`:

```yaml
# Global settings
syncInterval: 5m

repositories:
  - name: quad-ops
    url: "https://github.com/trly/quad-ops.git"
    ref: "main"
    composeDir: "examples"
  
  - name: my-project
    url: "file:///home/yourusername/my-quad-ops-project"
    cleanup: "delete"  # Remove units when they're deleted from Git
```

### 3. Sync Your Project

```bash
sudo quad-ops sync
```

### 4. Verify Deployment

```bash
sudo quad-ops unit list -t container
sudo podman ps
```

## Setting Up for Production

For continuous operation, you should set up Quad-Ops as a systemd service to ensure it runs permanently and automatically monitors your repositories.

Options include:
- **System-wide service**: For managing containers as root
- **User service**: For rootless container management

See the [Systemd Service](../configuration/systemd-service/) guide for detailed instructions on setting up either option.

## Troubleshooting

### Common Issues and Solutions

| Issue | Solutions |
|-------|----------|
| **Container won't start** | • Check logs: `journalctl -u servicename`<br>• Verify bind mount directories exist<br>• Ensure image names are fully qualified |
| **Permission denied** | • Verify `/etc/containers/systemd` permissions<br>• Check SELinux contexts if applicable |
| **Networking issues** | • Check network units: `systemctl status reponame-projectname-networkname.network`<br>• Verify container name resolution with `podman exec container ping servicename` |
| **Sync failing** | • Check Git access: `journalctl -u quad-ops \| grep "git"`<br>• Verify repository URLs and credentials |

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

Congratulations! You now have Quad-Ops running and managing your containers. Here are some next steps:

1. **Create your own Docker Compose files** with your applications
2. **Push your configurations to Git** repositories for proper GitOps workflow
3. **Explore more advanced features** like secrets management and network configuration

For more detailed information, check out these guides after you're comfortable with the basics:

- [Docker Compose Support](../docker-compose/)
- [Systemd Service Configuration](../configuration/systemd-service/)
- [Dependency Management](../dependency-management/)
