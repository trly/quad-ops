# Split Database and App Example

This example demonstrates how to split a multi-service application into separate Docker Compose files using quad-ops, with one service accessing another via an external network.

## Overview

This directory contains two Docker Compose files:

1. `database/docker-compose.yml` - Defines a MariaDB database service and creates a network called `db-network`
2. `wordpress/docker-compose.yml` - Defines a WordPress webapp that connects to the database using an external network reference

## How it works

1. quad-ops processes `database/docker-compose.yml` and creates:
   - A container unit for the MariaDB database
   - A network unit for `db-network`
   - A volume unit for `db-data`

2. quad-ops processes `wordpress/docker-compose.yml` and:
   - Creates a container unit for the WordPress application
   - Creates a volume unit for `wp-content`
   - References the database's network as external with the `systemd-` prefix (no network unit created)
   - Connects to the database network to allow communication between services

## Key Features

- **Service Separation**: Database and webapp are defined and managed separately
- **External Network**: The webapp references the database's network as external, specifying the name with the `systemd-` prefix Podman adds
- **DNS Resolution**: The webapp connects to the database using the DNS name `quad-ops-database-db`

## Configuration in quad-ops

```yaml
repositories:
  - name: external-resources
    url: https://github.com/example/services.git
    ref: main
    composeDir: examples/external-resources
```

## Benefits of This Approach

1. **Independent Lifecycle**: The database can be updated or restarted without affecting the webapp configuration
2. **Security Isolation**: Keeps database configuration separate from application configuration
3. **Simplified Management**: Each service can be maintained by different teams or repositories
4. **Flexible Deployment**: The database could be moved to a different host without changing the webapp configuration

## Accessing the Application

After deploying with quad-ops, the WordPress application will be available at http://localhost:8080