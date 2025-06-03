---
title: "Container Management"
weight: 30
bookCollapseSection: true
---

# Container Management

This section covers how Quad-Ops processes Git repositories, converts Docker Compose files, and manages container lifecycles through Podman Quadlet and systemd.

## Overview

Quad-Ops provides a GitOps approach to container management by:

1. **Monitoring Git repositories** for Docker Compose file changes
2. **Converting Docker Compose** configurations to Podman Quadlet units
3. **Managing systemd services** for container lifecycle operations
4. **Handling dependencies** between services, volumes, and networks

## Key Concepts

### GitOps Workflow
Changes to containers are driven by Git commits, not manual commands. This ensures:
- **Version control** of all infrastructure changes
- **Rollback capability** through Git history
- **Audit trail** of who changed what and when
- **Automated deployment** of approved changes

### Podman Quadlet Integration
Quad-Ops leverages Podman's Quadlet feature to create systemd-native container management:
- **Systemd units** for containers, volumes, and networks
- **Dependency management** through systemd's After/Requires directives
- **Service restart** and failure handling via systemd
- **Logging integration** with journald

### Repository Processing
Each configured repository is processed independently:
- **Git synchronization** pulls latest changes
- **File discovery** locates Docker Compose files
- **Conversion process** generates Quadlet units
- **Deployment** loads and starts systemd services

## Section Contents

### [Repository Structure](repository-structure)
Understanding how Quad-Ops reads and processes files from Git repositories.

### [Docker Compose Support](docker-compose-support)  
Comprehensive guide to supported Docker Compose features and conversion details.

### [Environment Files](environment-files)
How environment files are discovered, processed, and used in container configuration.

### [Build Support](build-support)
Docker Compose build configurations and Podman Quadlet build unit conversion.

### [Init Containers](init-containers)
Using init containers for service initialization, similar to Kubernetes init containers.