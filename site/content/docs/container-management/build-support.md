---
title: "Build Support"
weight: 40
---

# Build Support

Quad-Ops converts Docker Compose build configurations to Podman Quadlet build units, enabling containerized builds managed by systemd.

## Build Configuration Overview

Docker Compose `build` sections are converted to Podman Quadlet `.build` units that:
- **Build container images** from Dockerfiles
- **Manage build dependencies** through systemd
- **Handle build arguments** and environment variables
- **Support multi-stage builds** and build targets
- **Integrate with container units** for automatic image usage

## Basic Build Configuration

### Simple Build

**Docker Compose:**
```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:80"
```

**Generated build unit:** `myproject-app-build.build`
```ini
[Unit]
Description=Build myproject-app

[Build]
ImageTag=localhost/myproject-app:latest
SetWorkingDirectory=/path/to/context

[Service]
Type=oneshot

[Install]
WantedBy=default.target
```

**Generated container unit:** `myproject-app.container`
```ini
[Unit]
Description=myproject-app container
After=myproject-app-build.service

[Container]
Image=localhost/myproject-app:latest
PublishPort=8080:80
NetworkAlias=app

[Install]
WantedBy=default.target
```

## Advanced Build Configuration

### Complete Build Setup

**Docker Compose:**
```yaml
version: '3.8'
services:
  api:
    build:
      context: ./backend
      dockerfile: Dockerfile.prod
      target: production
      args:
        - NODE_ENV=production
        - VERSION=1.0.0
        - BUILD_DATE=${BUILD_DATE}
      labels:
        - "app=api"
        - "version=1.0.0"
      network: host
      pull: true
      secrets:
        - source: api_key
          target: /run/secrets/api_key
    environment:
      - NODE_ENV=production
```

**Generated build unit:** `myproject-api-build.build`
```ini
[Unit]
Description=Build myproject-api

[Build]
ImageTag=localhost/myproject-api:latest
SetWorkingDirectory=./backend
File=Dockerfile.prod
Target=production
BuildArg=NODE_ENV=production
BuildArg=VERSION=1.0.0
BuildArg=BUILD_DATE=2024-01-15T10:30:00Z
Label=app=api
Label=version=1.0.0
Pull=true
Network=host
Secret=api_key,target=/run/secrets/api_key

[Service]
Type=oneshot

[Install]
WantedBy=default.target
```

## Build Arguments

### Static Build Arguments

```yaml
services:
  app:
    build:
      context: .
      args:
        NODE_ENV: production
        VERSION: "1.0.0"
        DEBIAN_FRONTEND: noninteractive
```

### Dynamic Build Arguments

```yaml
services:
  app:
    build:
      context: .
      args:
        - NODE_ENV=${NODE_ENV}
        - VERSION=${GIT_TAG}
        - BUILD_DATE=${BUILD_DATE}
```

**Environment file (`.env`):**
```env
NODE_ENV=production
GIT_TAG=v1.2.3
BUILD_DATE=2024-01-15T10:30:00Z
```

### Build Arguments in Dockerfile

**Dockerfile:**
```dockerfile
FROM node:18-alpine
ARG NODE_ENV=development
ARG VERSION=latest
ARG BUILD_DATE

ENV NODE_ENV=${NODE_ENV}
LABEL version=${VERSION}
LABEL build_date=${BUILD_DATE}

COPY package.json package-lock.json ./
RUN npm ci --only=production

COPY . .
EXPOSE 3000
CMD ["node", "server.js"]
```

## Multi-Stage Builds

### Development and Production Targets

**Dockerfile:**
```dockerfile
# Build stage
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Development stage
FROM node:18-alpine AS development
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
EXPOSE 3000
CMD ["npm", "run", "dev"]

# Production stage
FROM nginx:alpine AS production
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

**Docker Compose (Development):**
```yaml
services:
  app-dev:
    build:
      context: .
      target: development
    volumes:
      - .:/app
      - /app/node_modules
```

**Docker Compose (Production):**
```yaml
services:
  app:
    build:
      context: .
      target: production
    ports:
      - "80:80"
```

## Build Secrets

### Secret Mounting

**Docker Compose:**
```yaml
services:
  app:
    build:
      context: .
      secrets:
        - source: github_token
          target: /run/secrets/github_token
        - source: npm_token
          target: /run/secrets/npm_token

secrets:
  github_token:
    file: ./secrets/github_token.txt
  npm_token:
    file: ./secrets/npm_token.txt
```

**Dockerfile:**
```dockerfile
FROM node:18-alpine

# Use secret during build
RUN --mount=type=secret,id=npm_token \
    echo "//registry.npmjs.org/:_authToken=$(cat /run/secrets/npm_token)" > ~/.npmrc && \
    npm ci --only=production && \
    rm ~/.npmrc

COPY . .
CMD ["node", "server.js"]
```

### SSH Authentication

**Docker Compose:**
```yaml
services:
  app:
    build:
      context: .
      ssh:
        - default=/home/user/.ssh/id_rsa
```

**Dockerfile:**
```dockerfile
FROM alpine/git AS source
RUN --mount=type=ssh \
    git clone git@github.com:private/repo.git /src

FROM node:18-alpine
COPY --from=source /src /app
WORKDIR /app
RUN npm ci
CMD ["node", "server.js"]
```

## Build Networks

### Custom Build Network

```yaml
services:
  app:
    build:
      context: .
      network: build-network

networks:
  build-network:
    external: true
```

### Host Network for Build

```yaml
services:
  app:
    build:
      context: .
      network: host  # Use host networking during build
```

## Podman-Specific Build Extensions

### Build Cache Configuration

Use Quad-Ops' [x-podman-buildargs extension](../docker-compose-support/#build-extensions) for additional build arguments:

```yaml
services:
  app:
    build:
      context: .
    x-podman-buildargs:
      BUILDAH_LAYERS: "true"
      BUILDKIT_INLINE_CACHE: "1"
```

### Custom Build Volumes

Use Quad-Ops' [x-podman-volumes extension](../docker-compose-support/#volume-extensions) for build-time volume mounts:

```yaml
services:
  app:
    build:
      context: .
    x-podman-volumes:
      - "build-cache:/cache"
      - "/tmp/build:/tmp:rw"
```

## Build Dependencies and Lifecycle

### Build Order Management

**Multi-service build order:**
```yaml
services:
  base:
    build:
      context: ./base
      dockerfile: Dockerfile.base

  app:
    build:
      context: ./app
    depends_on:
      - base  # Ensures base is built first

  frontend:
    build:
      context: ./frontend
    depends_on:
      - app   # Can reference app's API during build
```

### Build Triggers

Builds are triggered when:
1. **Dockerfile changes** detected
2. **Build context files** modified
3. **Build arguments** change
4. **Manual rebuild** requested

### Build Unit Management

**Check build status:**
```bash
# List build units
sudo quad-ops unit list -t build

# Check build service status
sudo systemctl status myproject-app-build

# View build logs
sudo journalctl -u myproject-app-build
```

**Manual build operations:**
```bash
# Trigger rebuild
sudo systemctl start myproject-app-build

# Restart dependent containers after rebuild
sudo systemctl restart myproject-app
```

## Build Context Optimization

### .dockerignore Usage

**`.dockerignore`:**
```
node_modules/
npm-debug.log
Dockerfile*
.dockerignore
.git
.gitignore
README.md
.env
.nyc_output
coverage/
.nyc_output/
```

### Minimal Build Context

```yaml
services:
  app:
    build:
      context: .
      dockerfile: docker/Dockerfile
      # Only necessary files copied by Dockerfile
```

**Dockerfile with selective copying:**
```dockerfile
FROM node:18-alpine
WORKDIR /app

# Copy package files first for better caching
COPY package*.json ./
RUN npm ci --only=production

# Copy source code
COPY src/ ./src/
COPY public/ ./public/

EXPOSE 3000
CMD ["node", "src/server.js"]
```

### Debug Build Process

**Check generated build unit:**
```bash
# View build unit file
cat /etc/containers/systemd/myproject-app-build.build

# Check build service status
systemctl status myproject-app-build.service
```

**Manual build testing:**
```bash
# Test build manually
cd /path/to/context
podman build -t test-image .

# Test with same args as quadlet
podman build \
  --build-arg NODE_ENV=production \
  --build-arg VERSION=1.0.0 \
  --target production \
  -t test-image .
```

**Build logs analysis:**
```bash
# View build logs
journalctl -u myproject-app-build.service -f

# Check for specific errors
journalctl -u myproject-app-build.service | grep -i error
```

### Performance Optimization

**Layer caching optimization:**
```dockerfile
# ✅ Good - Dependencies cached separately
FROM node:18-alpine
COPY package*.json ./
RUN npm ci
COPY . .

# ❌ Bad - Everything rebuilt on any change
FROM node:18-alpine
COPY . .
RUN npm ci
```

**Multi-stage optimization:**
```dockerfile
# Build stage with all dependencies
FROM node:18 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Runtime stage with minimal footprint
FROM node:18-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package*.json ./
RUN npm ci --only=production
EXPOSE 3000
CMD ["node", "dist/server.js"]
```

## Integration with CI/CD

### Build in Development

```yaml
# Local development with live builds
services:
  app:
    build: .
    volumes:
      - .:/app
      - /app/node_modules
    environment:
      - NODE_ENV=development
```

### Production Builds

```yaml
# Production with optimized builds
services:
  app:
    build:
      context: .
      target: production
      args:
        - NODE_ENV=production
        - VERSION=${GIT_TAG}
    environment:
      - NODE_ENV=production
```

### Automated Build Pipeline

```bash
#!/bin/bash
# build-and-deploy.sh

# Set build variables
export GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)
export BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Trigger quad-ops sync (includes builds)
quad-ops sync --repository myapp

# Verify builds completed
if systemctl is-active myproject-app-build.service; then
    echo "Build completed successfully"
    quad-ops unit list -t container
else
    echo "Build failed"
    journalctl -u myproject-app-build.service --no-pager
    exit 1
fi
```

## Next Steps

- [Docker Compose Support](docker-compose-support) - Complete compose reference
- [Supported Features](../podman-systemd/supported-features) - Feature compatibility
- [Repository Structure](repository-structure) - Understanding build contexts