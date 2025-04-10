#!/bin/sh
# Outrig installation script
# This script downloads and installs the latest version of Outrig for Linux
# Usage: curl -sf https://raw.githubusercontent.com/outrigdev/outrig/main/assets/install.sh | sh

set -e

# Print a message with a prefix
info() {
    echo "INFO: $1"
}

# Print error message and exit
# Print error message and exit
error() {
    echo "Error: $1" >&2
    exit 1
}

# Detect architecture
detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)
            ARCH="x86_64"
            ;;
        amd64)
            ARCH="x86_64"
            ;;
        arm64)
            ARCH="arm64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac
    echo "$ARCH"
}

# Detect OS
detect_os() {
    OS=$(uname -s)
    case "$OS" in
        Linux)
            OS="Linux"
            ;;
        *)
            error "This installation script is only for Linux. For other platforms, please see https://github.com/outrigdev/outrig#installation"
            ;;
    esac
    echo "$OS"
}

# Check if command exists
has_command() {
    command -v "$1" >/dev/null 2>&1
}

# Check if running as root
check_root() {
    if [ "$(id -u)" = "0" ]; then
        info "Warning: You are running this script as root, which is not necessary."
        info "This script will install Outrig to the current user's ~/.local/bin directory."
        info "Press Ctrl+C to cancel or wait 5 seconds to continue..."
        sleep 5
    fi
}

# Main installation function
install_outrig() {
    # Check if running as root
    check_root
    info "Installing Outrig..."
    
    # Detect OS and architecture
    OS=$(detect_os)
    ARCH=$(detect_arch)
    
    info "Detected OS: $OS, Architecture: $ARCH"
    
    # Check for required commands
    has_command curl || error "curl is required for installation"
    has_command tar || error "tar is required for installation"
    
    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT
    
    # Get the latest release URL
    RELEASE_URL="https://github.com/outrigdev/outrig/releases/latest/download/outrig_${OS}_${ARCH}.tar.gz"
    
    info "Downloading Outrig from $RELEASE_URL..."
    
    # Download and extract
    if ! curl -L --progress-bar "$RELEASE_URL" | tar xz -C "$TMP_DIR"; then
        error "Failed to download or extract Outrig. Please check your internet connection and try again."
    fi
    
    # Find the outrig binary in the extracted directory
    OUTRIG_BIN=$(find "$TMP_DIR" -name "outrig" -type f)
    
    if [ -z "$OUTRIG_BIN" ]; then
        error "Could not find outrig binary in the downloaded archive"
    fi
    
    # Install to ~/.local/bin
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
    info "Installing to $INSTALL_DIR..."
    cp "$OUTRIG_BIN" "$INSTALL_DIR/outrig"
    chmod 755 "$INSTALL_DIR/outrig"
    
    # Check if ~/.local/bin is in PATH using POSIX-compatible pattern matching
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *) info "Note: $INSTALL_DIR is not in your PATH. You may need to add it to use outrig." ;;
    esac
    
    info "Outrig has been installed successfully!"
    info "Run 'outrig server' to start the Outrig server"
    
    # No additional verification needed as we already checked PATH above
}

# Run the installation
install_outrig