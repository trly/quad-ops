---
title: "Web + Database + Reverse Proxy"
---

# Complete Stack with Reverse Proxy

This example demonstrates a more complex setup with a web application, database, and Traefik reverse proxy for handling multiple services with SSL termination.

## Docker Compose File

```yaml
version: '3.8'

services:
  traefik:
    image: docker.io/traefik:v2.9
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"
    volumes:
      - /var/run/podman/podman.sock:/var/run/docker.sock:ro
      - ./traefik/config:/etc/traefik
      - ./traefik/certificates:/certificates
    networks:
      - proxy
    restart: always
    command:
      - --api.insecure=true
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --providers.file.directory=/etc/traefik
      - --entrypoints.web.address=:80
      - --entrypoints.websecure.address=:443
      - --certificatesresolvers.myresolver.acme.email=user@example.com
      - --certificatesresolvers.myresolver.acme.storage=/certificates/acme.json
      - --certificatesresolvers.myresolver.acme.tlschallenge=true

  webapp:
    image: docker.io/wordpress:latest
    volumes:
      - wp-content:/var/www/html/wp-content
    environment:
      - WORDPRESS_DB_HOST=myapp-db
      - WORDPRESS_DB_NAME=wordpress
      - WORDPRESS_DB_USER=wp_user
      - WORDPRESS_DB_PASSWORD_FILE=/run/secrets/db_password
    depends_on:
      - db
    networks:
      - proxy
      - app-network
    restart: always
    secrets:
      - source: db_password
        target: /run/secrets/db_password
        mode: 0400
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.webapp.rule=Host(`wordpress.example.com`)"
      - "traefik.http.routers.webapp.entrypoints=websecure"
      - "traefik.http.routers.webapp.tls.certresolver=myresolver"
      - "traefik.http.services.webapp.loadbalancer.server.port=80"

  db:
    image: docker.io/mariadb:10.6
    volumes:
      - db-data:/var/lib/mysql
    environment:
      - MYSQL_DATABASE=wordpress
      - MYSQL_USER=wp_user
      - MYSQL_PASSWORD_FILE=/run/secrets/db_password
      - MYSQL_ROOT_PASSWORD_FILE=/run/secrets/db_root_password
    restart: always
    networks:
      - app-network
    secrets:
      - source: db_password
        target: /run/secrets/db_password
        mode: 0400
      - source: db_root_password
        target: /run/secrets/db_root_password
        mode: 0400

volumes:
  wp-content:
  db-data:

networks:
  proxy:
    driver: bridge
  app-network:
    driver: bridge
    internal: true

secrets:
  db_password:
    file: ./secrets/db_password.txt
  db_root_password:
    file: ./secrets/db_root_password.txt
```

## Key Points

1. **Traefik Configuration**: Traefik is used as a reverse proxy with SSL termination
   - Note the Podman socket is mounted at `/var/run/podman/podman.sock`
   - We mount configuration and certificate directories

2. **Multiple Networks**:
   - `proxy`: Public-facing network for Traefik and web applications
   - `app-network`: Internal network (note `internal: true`) for backend services

3. **Secrets Management**:
   - Secrets are stored as files and mounted into containers
   - Environment variables reference secrets with `_FILE` suffix

4. **Labels for Traefik**:
   - The webapp container uses labels to configure Traefik routing

## Prerequisites

Before running this example:

1. Create necessary directories:

```bash
mkdir -p traefik/config traefik/certificates secrets
```

2. Create basic Traefik configuration:

```bash
cat > traefik/config/traefik.yml << 'EOF'
http:
  middlewares:
    secure-headers:
      headers:
        sslRedirect: true
        forceSTSHeader: true
        stsIncludeSubdomains: true
        stsPreload: true
        stsSeconds: 31536000
EOF
```

3. Create secret files:

```bash
echo "secure_password" > secrets/db_password.txt
echo "secure_root_password" > secrets/db_root_password.txt
chmod 600 secrets/db_password.txt secrets/db_root_password.txt
```

## Podman-Specific Considerations

1. **Socket Path**: The Podman socket path differs from Docker:
   - Docker: `/var/run/docker.sock`
   - Podman: `/var/run/podman/podman.sock`

2. **Traefik Provider**: While we still use the `providers.docker` setting, Traefik will work with Podman

3. **Internal Networks**: With Quadlet, internal networks work as expected, isolating backend services

4. **DNS Resolution**: Remember that with default quad-ops settings, container DNS names use the format `myapp-servicename`

## Usage

After syncing with Quad-Ops:

```bash
# Start the entire stack
systemctl start myapp-traefik.service

# Check status
systemctl status myapp-traefik.service myapp-webapp.service myapp-db.service
```

Access the applications:
- Traefik Dashboard: `http://localhost:8080`
- WordPress: `https://wordpress.example.com` (After adding DNS entry or /etc/hosts entry)

## Troubleshooting

**Issue**: Traefik can't access the Podman socket

**Solution**: Check socket permissions and SELinux context:

```bash
# For rootless Podman
chmod 777 $XDG_RUNTIME_DIR/podman/podman.sock

# For system Podman
chmod 777 /var/run/podman/podman.sock
```

**Issue**: Traefik can see the containers but routing doesn't work

**Solution**: Verify that containers are on the same network and check the label configuration