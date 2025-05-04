---
title: "Services"
---

# Services

Services in Docker Compose are the core components that define your containers. Quad-Ops converts these service definitions to Podman container units that are managed by systemd.

## Supported Properties

- `image`: Container image (fully qualified names with registry prefix recommended)
- `ports`: Port mappings
- `volumes`: Volume mounts
- `networks`: Network connections
- `environment`: Environment variables
- `env_file`: Environment variable files
- `command`: Command to run
- `entrypoint`: Container entrypoint
- `user`: User to run as
- `working_dir`: Working directory
- `init`: Enable init process (`init: true/false`)
- `read_only`: Read-only container filesystem
- `depends_on`: Container startup dependencies
- `hostname`: Custom hostname
- `secrets`: File-based secrets
- `labels`: Container labels
- `restart`: Restart policy
- `cap_add`: Add container capabilities
- `cap_drop`: Drop container capabilities

## Example

```yaml
services:
  webapp:
    image: docker.io/library/wordpress:latest
    ports:
      - "8080:80"
    volumes:
      - wp-content:/var/www/html/wp-content
    environment:
      - WORDPRESS_DB_HOST=db
      - WORDPRESS_DB_NAME=wordpress
      - WORDPRESS_DB_USER=wp_user
      - WORDPRESS_DB_PASSWORD=db_password
    depends_on:
      - db
    restart: always
    networks:
      - app-network
    init: true
    labels:
      - "com.example.description=WordPress web application"
    hostname: webapp
  
  db:
    image: docker.io/library/mariadb:latest
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
    cap_add:
      - SYS_NICE
```

## Conversion to Podman Container Units

When Quad-Ops processes a service definition from a Docker Compose file, it creates a corresponding Podman container unit with the following mapping:

| Docker Compose Property | Podman Container Property |
|-------------------------|---------------------------|
| `image` | `Image` |
| `ports` | `PublishPort` |
| `volumes` | `Volume` |
| `networks` | `Network` |
| `environment` | `Environment` |
| `env_file` | `EnvironmentFile` |
| `command` | `Exec` |
| `entrypoint` | `Entrypoint` |
| `user` | `User` |
| `working_dir` | `WorkingDir` |
| `init` | `RunInit` |
| `read_only` | `ReadOnly` |
| `depends_on` | Converted to systemd unit dependencies |
| `hostname` | `HostName` |
| `labels` | `Label` |
| `restart` | Converted to systemd unit restart policy |
| `cap_add` | `AddCapability` |
| `cap_drop` | `DropCapability` |

## Important Notes

1. **Service Dependencies**: The `depends_on` property is converted to systemd unit dependencies using `After` and `Requires` directives.

2. **Container Naming**: By default, container hostnames match their service names without the systemd- prefix that Podman normally adds.

3. **Image Names**: Always use fully qualified image names with registry prefix (docker.io/, quay.io/, etc.) to avoid resolution issues.

4. **Restart Policies**: Docker Compose restart policies are converted to equivalent systemd restart configurations.

5. **Volumes**: Named volumes require the `.volume` suffix in Volume directives (e.g., `Volume=data.volume:/data`).

6. **Networks**: When using networks with aliases, the service name is automatically used as a network alias for simple service discovery.

7. **Service-Specific Environment Files**: Quad-Ops automatically detects and uses service-specific environment files present in the compose directory.

## Service-Specific Environment Files

Quad-Ops can automatically detect and use service-specific environment files to help manage environment variables, especially those with special characters that might cause issues with shell interpretation.

For a service named `service1`, Quad-Ops looks for the following files in the Docker Compose directory:

- `.env.service1` - Hidden env file with service name suffix
- `service1.env` - Service name with .env extension
- `env/service1.env` - In env subdirectory
- `envs/service1.env` - In envs subdirectory

These files are automatically added to the Quadlet unit with the `EnvironmentFile` directive, which allows Podman to properly handle environment variables with special characters like asterisks, spaces, quotes, and more.

### Example

For a service named `webapp` in a Docker Compose file, you could create a file named `webapp.env` in the same directory:

```env
# webapp.env
APP_DOMAINS=*.example.com
APP_NAME="My Web Application"
ALLOWED_HOSTS=host1,host2,host3
```

This is especially useful for environment variables that contain special characters that might otherwise be interpreted by the shell, such as asterisks (`*`), quotes (`"'`), spaces, or other special characters.

With this approach, you don't need to escape or specially quote these values in your Docker Compose file, as they will be properly handled by Podman through the environment file.