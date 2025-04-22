---
title: "Web + Database"
---

# Web + Database Example

This example demonstrates a common pattern: a web application with a database backend, showing how Quad-Ops handles service dependencies and networks.

## Docker Compose File

```yaml
version: '3.8'

services:
  webapp:
    image: docker.io/wordpress:latest
    ports:
      - "8080:80"
    volumes:
      - wp-content:/var/www/html/wp-content
    environment:
      - WORDPRESS_DB_HOST=myapp-db
      - WORDPRESS_DB_NAME=wordpress
      - WORDPRESS_DB_USER=wp_user
      - WORDPRESS_DB_PASSWORD=db_password
    depends_on:
      - db
    restart: always
    networks:
      - app-network

  db:
    image: docker.io/mariadb:10.6
    volumes:
      - db-data:/var/lib/mysql
    environment:
      - MYSQL_DATABASE=wordpress
      - MYSQL_USER=wp_user
      - MYSQL_PASSWORD=db_password
      - MYSQL_ROOT_PASSWORD=root_password
    restart: always
    networks:
      - app-network

volumes:
  wp-content:
  db-data:

networks:
  app-network:
    driver: bridge
```

## Generated Systemd Units

When Quad-Ops processes this Docker Compose file, it generates the following systemd units:

### Container Units

**myapp-webapp.container**:
```ini
[Unit]
Description=Podman container for myapp-webapp
Requires=myapp-db.service
After=myapp-db.service

[Container]
Image=docker.io/wordpress:latest
Volume=wp-content.volume:/var/www/html/wp-content
PublishPort=8080:80
Environment=WORDPRESS_DB_HOST=myapp-db
Environment=WORDPRESS_DB_NAME=wordpress
Environment=WORDPRESS_DB_USER=wp_user
Environment=WORDPRESS_DB_PASSWORD=db_password
Network=app-network.network

[Service]
Restart=always
TimeoutStartSec=60
TimeoutStopSec=60

[Install]
WantedBy=multi-user.target
```

**myapp-db.container**:
```ini
[Unit]
Description=Podman container for myapp-db

[Container]
Image=docker.io/mariadb:10.6
Volume=db-data.volume:/var/lib/mysql
Environment=MYSQL_DATABASE=wordpress
Environment=MYSQL_USER=wp_user
Environment=MYSQL_PASSWORD=db_password
Environment=MYSQL_ROOT_PASSWORD=root_password
Network=app-network.network

[Service]
Restart=always
TimeoutStartSec=60
TimeoutStopSec=60

[Install]
WantedBy=multi-user.target
```

### Volume Units

**wp-content.volume**:
```ini
[Unit]
Description=Podman volume for myapp-wp-content

[Volume]
Driver=local

[Install]
WantedBy=multi-user.target
```

**db-data.volume**:
```ini
[Unit]
Description=Podman volume for myapp-db-data

[Volume]
Driver=local

[Install]
WantedBy=multi-user.target
```

### Network Unit

**app-network.network**:
```ini
[Unit]
Description=Podman network for myapp-app-network

[Network]
Driver=bridge

[Install]
WantedBy=multi-user.target
```

## Key Points

1. **Container Dependencies**: Note how `depends_on` is translated to `Requires=myapp-db.service` and `After=myapp-db.service`

2. **DNS Resolution**: The database host is referenced as `myapp-db` with the default quad-ops settings (`usePodmanDefaultNames: false`)

3. **Named Volumes**: Volumes are created as separate systemd units with the `.volume` suffix

4. **Network Creation**: The custom network is created as a separate systemd unit with the `.network` suffix

## Usage

After syncing with Quad-Ops, start the services:

```bash
# Start all units for the project (will automatically start dependencies in correct order)
systemctl start myapp-webapp.service

# Check status of all units
systemctl list-units --all '*myapp*'
```

Access WordPress at `http://localhost:8080` and complete the installation process.

## Common Issues

**Issue**: WordPress can't connect to the database

**Solution**: Check the DNS hostname format. With default quad-ops settings (`usePodmanDefaultNames: false`), container names follow this format:

```
<project>-<service>
```

So in this example, the database host should be `myapp-db`. If you've set `usePodmanDefaultNames: true`, then use `systemd-myapp-db` instead.

**Issue**: Data persistence problems after restart

**Solution**: Verify the volume units are created and started:

```bash
# Check volume status
systemctl status wp-content.volume db-data.volume

# Verify the volumes exist in Podman
podman volume ls
```