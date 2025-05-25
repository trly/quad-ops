#!/bin/bash

set -e

# Default values
REPO="trly/quad-ops"
INSTALL_PATH=""
USER_INSTALL=false
BINARY_NAME="quad-ops"
VERSION_OVERRIDE=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Install quad-ops from GitHub releases

OPTIONS:
    -u, --user          Install to \$HOME/.local/bin (user install)
    --install-path PATH Install to specific path (overrides --user)
    --version VERSION   Install specific version (e.g., v1.2.3)
    -h, --help          Show this help message

Default install location: /opt/quad-ops/bin
Default behavior: Install latest version
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--user)
            USER_INSTALL=true
            shift
            ;;
        --install-path)
            INSTALL_PATH="$2"
            shift 2
            ;;
        --version)
            VERSION_OVERRIDE="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Determine install path
if [[ -n "$INSTALL_PATH" ]]; then
    FINAL_INSTALL_PATH="$INSTALL_PATH"
elif [[ "$USER_INSTALL" == true ]]; then
    FINAL_INSTALL_PATH="$HOME/.local/bin"
else
    FINAL_INSTALL_PATH="/opt/quad-ops/bin"
fi

print_info "Install path: $FINAL_INSTALL_PATH"

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

print_info "Detected architecture: $ARCH"

# Check dependencies
for cmd in curl tar sha256sum; do
    if ! command -v $cmd >/dev/null 2>&1; then
        print_error "Required command '$cmd' not found"
        exit 1
    fi
done

# Get version to install
if [[ -n "$VERSION_OVERRIDE" ]]; then
    VERSION="$VERSION_OVERRIDE"
    print_info "Installing specified version: $VERSION"
else
    print_info "Getting latest release information..."
    RELEASE_INFO=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
    if [[ $? -ne 0 ]]; then
        print_error "Failed to get release information"
        exit 1
    fi

    VERSION=$(echo "$RELEASE_INFO" | grep '"tag_name":' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    if [[ -z "$VERSION" ]]; then
        print_error "Failed to parse version from release info"
        exit 1
    fi

    print_info "Latest version: $VERSION"
fi

# Construct download URLs
BINARY_FILE="${BINARY_NAME}_${VERSION#v}_linux_${ARCH}.tar.gz"
CHECKSUM_FILE="${BINARY_NAME}_${VERSION#v}_checksums.txt"
BINARY_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY_FILE"
CHECKSUM_URL="https://github.com/$REPO/releases/download/$VERSION/$CHECKSUM_FILE"

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

print_info "Downloading files to temporary directory: $TEMP_DIR"

# Download binary
print_info "Downloading $BINARY_FILE..."
if ! curl -L -o "$TEMP_DIR/$BINARY_FILE" "$BINARY_URL"; then
    print_error "Failed to download binary"
    exit 1
fi

# Download checksums
print_info "Downloading checksums..."
if ! curl -L -o "$TEMP_DIR/$CHECKSUM_FILE" "$CHECKSUM_URL"; then
    print_error "Failed to download checksums"
    exit 1
fi

# Verify checksum
print_info "Verifying checksum..."
cd "$TEMP_DIR"
EXPECTED_CHECKSUM=$(grep "$BINARY_FILE" "$CHECKSUM_FILE" | awk '{print $1}')
if [[ -z "$EXPECTED_CHECKSUM" ]]; then
    print_error "Could not find checksum for $BINARY_FILE"
    exit 1
fi

ACTUAL_CHECKSUM=$(sha256sum "$BINARY_FILE" | awk '{print $1}')
if [[ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]]; then
    print_error "Checksum verification failed!"
    print_error "Expected: $EXPECTED_CHECKSUM"
    print_error "Actual: $ACTUAL_CHECKSUM"
    exit 1
fi

print_info "Checksum verification passed"

# Extract binary
print_info "Extracting binary..."
if ! tar -xzf "$BINARY_FILE"; then
    print_error "Failed to extract binary"
    exit 1
fi

# Find the extracted binary
EXTRACTED_BINARY=$(find . -name "$BINARY_NAME" -type f | head -1)
if [[ -z "$EXTRACTED_BINARY" ]]; then
    print_error "Could not find extracted binary"
    exit 1
fi

# Create install directory if it doesn't exist
if [[ ! -d "$FINAL_INSTALL_PATH" ]]; then
    print_info "Creating install directory: $FINAL_INSTALL_PATH"
    if [[ "$FINAL_INSTALL_PATH" == "/opt/"* ]]; then
        # System install requires sudo
        sudo mkdir -p "$FINAL_INSTALL_PATH"
    else
        mkdir -p "$FINAL_INSTALL_PATH"
    fi
fi

# Install binary
FINAL_BINARY_PATH="$FINAL_INSTALL_PATH/$BINARY_NAME"
print_info "Installing binary to: $FINAL_BINARY_PATH"

if [[ "$FINAL_INSTALL_PATH" == "/opt/"* ]]; then
    # System install requires sudo
    sudo cp "$EXTRACTED_BINARY" "$FINAL_BINARY_PATH"
    sudo chmod +x "$FINAL_BINARY_PATH"
    sudo chown root:root "$FINAL_BINARY_PATH"
else
    cp "$EXTRACTED_BINARY" "$FINAL_BINARY_PATH"
    chmod +x "$FINAL_BINARY_PATH"
fi

print_info "Installation completed successfully!"

# Install systemd service
install_systemd_service() {
    local service_name service_url service_path
    
    if [[ "$USER_INSTALL" == true ]]; then
        service_name="quad-ops.service"
        service_url="https://raw.githubusercontent.com/$REPO/$VERSION/build/package/$service_name"
        service_path="$HOME/.config/systemd/user/quad-ops.service"
        
        print_info "Installing user systemd service..."
        mkdir -p "$(dirname "$service_path")"
        
        if curl -L -o "$service_path" "$service_url" 2>/dev/null; then
            systemctl --user daemon-reload 2>/dev/null || true
            print_info "User systemd service installed at: $service_path"
            print_info "To enable and start: systemctl --user enable --now quad-ops"
        else
            print_warn "Failed to download user systemd service file"
        fi
    else
        print_info "Installing system systemd services..."
        
        # Install regular service
        service_name="quad-ops.service"
        service_url="https://raw.githubusercontent.com/$REPO/$VERSION/build/package/$service_name"
        service_path="/etc/systemd/system/$service_name"
        
        if curl -L -o "$TEMP_DIR/$service_name" "$service_url" 2>/dev/null; then
            sudo cp "$TEMP_DIR/$service_name" "$service_path"
            print_info "Regular service installed at: $service_path"
        else
            print_warn "Failed to download regular systemd service file"
        fi
        
        # Install template service for user mode
        template_name="quad-ops@.service"
        template_url="https://raw.githubusercontent.com/$REPO/$VERSION/build/package/$template_name"
        template_path="/etc/systemd/user/$template_name"
        
        sudo mkdir -p "/etc/systemd/user"
        if curl -L -o "$TEMP_DIR/$template_name" "$template_url" 2>/dev/null; then
            sudo cp "$TEMP_DIR/$template_name" "$template_path"
            print_info "User template service installed at: $template_path"
        else
            print_warn "Failed to download template systemd service file"
        fi
        
        sudo systemctl daemon-reload 2>/dev/null || true
        print_info "To enable and start: sudo systemctl enable --now quad-ops"
        print_info "For user mode: systemctl --user enable --now quad-ops@"
    fi
}

# Install profile.d script for system-wide PATH
install_profile_script() {
    if [[ "$USER_INSTALL" != true && "$FINAL_INSTALL_PATH" == "/opt/quad-ops/bin" ]]; then
        print_info "Installing system-wide PATH script..."
        
        cat > "$TEMP_DIR/quad-ops.sh" << 'EOF'
export PATH="/opt/quad-ops/bin:$PATH"
EOF
        
        sudo cp "$TEMP_DIR/quad-ops.sh" "/etc/profile.d/quad-ops.sh"
        sudo chmod 644 "/etc/profile.d/quad-ops.sh"
        sudo chown root:root "/etc/profile.d/quad-ops.sh"
        
        print_info "PATH script installed at: /etc/profile.d/quad-ops.sh"
        print_info "Log out and back in for the PATH change to take effect system-wide"
    fi
}

# Install example configuration file
install_example_config() {
    local config_url config_path
    
    config_url="https://raw.githubusercontent.com/$REPO/$VERSION/configs/config.yaml.example"
    
    if [[ "$USER_INSTALL" == true ]]; then
        config_path="$HOME/.config/quad-ops/config.yaml.example"
        
        print_info "Installing user example configuration..."
        mkdir -p "$(dirname "$config_path")"
        
        if curl -L -o "$config_path" "$config_url" 2>/dev/null; then
            print_info "Example config installed at: $config_path"
            print_info "Copy to config.yaml and customize: cp '$config_path' '$HOME/.config/quad-ops/config.yaml'"
        else
            print_warn "Failed to download example configuration file"
        fi
    else
        config_path="/etc/opt/quad-ops/config.yaml.example"
        
        print_info "Installing system example configuration..."
        sudo mkdir -p "$(dirname "$config_path")"
        
        if curl -L -o "$TEMP_DIR/config.yaml.example" "$config_url" 2>/dev/null; then
            sudo cp "$TEMP_DIR/config.yaml.example" "$config_path"
            sudo chown root:root "$config_path"
            sudo chmod 644 "$config_path"
            print_info "Example config installed at: $config_path"
            print_info "Copy to config.yaml and customize: sudo cp '$config_path' '/etc/opt/quad-ops/config.yaml'"
        else
            print_warn "Failed to download example configuration file"
        fi
    fi
}

# Install systemd service if systemctl is available
if command -v systemctl >/dev/null 2>&1; then
    install_systemd_service
fi

# Install profile.d script for system installs
install_profile_script

# Install example configuration file
install_example_config

# Check if install path is in PATH
if [[ ":$PATH:" != *":$FINAL_INSTALL_PATH:"* ]]; then
    print_warn "Warning: $FINAL_INSTALL_PATH is not in your PATH"
    if [[ "$USER_INSTALL" == true ]]; then
        print_warn "Add this line to your shell profile (.bashrc, .zshrc, etc.):"
        print_warn "export PATH=\"\$PATH:\$HOME/.local/bin\""
    elif [[ "$FINAL_INSTALL_PATH" == "/opt/quad-ops/bin" ]]; then
        print_warn "Add this line to your shell profile (.bashrc, .zshrc, etc.):"
        print_warn "export PATH=\"\$PATH:/opt/quad-ops/bin\""
    fi
fi

# Test installation
if command -v "$BINARY_NAME" >/dev/null 2>&1 || [[ -x "$FINAL_BINARY_PATH" ]]; then
    print_info "Installation verified. Run '$BINARY_NAME --help' to get started."
else
    print_warn "Binary installed but not found in PATH. Use full path: $FINAL_BINARY_PATH"
fi