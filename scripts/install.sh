#!/usr/bin/env bash
# Skint Installation Script
# Supports macOS and Linux

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging
info() { echo -e "${BLUE}→${NC} $*"; }
success() { echo -e "${GREEN}✓${NC} $*"; }
warn() { echo -e "${YELLOW}⚠${NC} $*" >&2; }
error() { echo -e "${RED}✗${NC} $*" >&2; }

# Configuration
REPO="sammcj/skint"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-}"

# Detect OS and architecture
detect_platform() {
    local os arch

    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$os" in
        linux) os="linux" ;;
        darwin) os="darwin" ;;
        *) error "Unsupported OS: $os"; exit 1 ;;
    esac

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) error "Unsupported architecture: $arch"; exit 1 ;;
    esac

    echo "${os}_${arch}"
}

# Get latest release version
get_latest_version() {
    local version
    version=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$version" ]; then
        error "Failed to get latest version"
        exit 1
    fi
    echo "$version"
}

# Download and install
download_and_install() {
    local version="$1"
    local platform="$2"
    local bin_dir="$3"

    # Remove 'v' prefix if present
    version="${version#v}"

    local filename="skint_${version}_${platform}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/v${version}/${filename}"

    info "Downloading Skint ${version}..."

    # Create temp directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Download
    if ! curl -fsSL "$url" -o "${tmp_dir}/${filename}"; then
        error "Failed to download from $url"
        exit 1
    fi

    # Extract
    info "Extracting..."
    tar -xzf "${tmp_dir}/${filename}" -C "$tmp_dir"

    # Install
    info "Installing to ${bin_dir}..."
    mkdir -p "$bin_dir"

    if [ -f "${tmp_dir}/skint" ]; then
        cp "${tmp_dir}/skint" "${bin_dir}/skint"
        chmod +x "${bin_dir}/skint"
    else
        error "Could not find skint binary in archive"
        exit 1
    fi

    success "Installed Skint ${version}"
}

# Build from source (fallback)
build_from_source() {
    local bin_dir="$1"

    info "Building from source..."

    # Check for Go
    if ! command -v go &>/dev/null; then
        error "Go is not installed. Please install Go 1.24 or later."
        exit 1
    fi

    # Check Go version
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    if [ "$(printf '%s\n' "1.24" "$go_version" | sort -V | head -n1)" != "1.24" ]; then
        error "Go 1.24 or later is required (found ${go_version})"
        exit 1
    fi

    # Create temp directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Clone repo
    info "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO}.git" "${tmp_dir}/skint"

    # Build
    info "Building..."
    cd "${tmp_dir}/skint/skint"
    go build -o "${bin_dir}/skint" ./cmd/skint

    success "Built and installed Skint"
}

# Get bin directory
get_bin_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Check SKINT_BIN env var
    if [ -n "${SKINT_BIN:-}" ]; then
        echo "$SKINT_BIN"
        return
    fi

    # Default based on OS
    if [ "$(uname -s)" = "Darwin" ]; then
        echo "$HOME/bin"
    else
        echo "$HOME/.local/bin"
    fi
}

# Check PATH
check_path() {
    local bin_dir="$1"

    if [[ ":$PATH:" != *":${bin_dir}:"* ]]; then
        warn "'$bin_dir' is not in your PATH"
        echo
        info "Add it to your shell profile:"

        local shell_rc="$HOME/.bashrc"
        if [ -n "${ZSH_VERSION:-}" ] || [ "${SHELL##*/}" = "zsh" ]; then
            shell_rc="$HOME/.zshrc"
        elif [ "${SHELL##*/}" = "fish" ]; then
            echo "  fish_add_path $bin_dir"
            return
        fi

        echo "  echo 'export PATH=\"$bin_dir:\$PATH\"' >> $shell_rc"
        echo "  source $shell_rc"
    fi
}

# Main installation
main() {
    echo
    echo "  ____ _       _   _"
    echo " / ___| | ___ | |_| |__   ___ _ __"
    echo "| |   | |/ _ \\| __| '_ \\ / _ \\ '__|"
    echo "| |___| | (_) | |_| | | |  __/ |"
    echo " \\____|_|\\___/ \\__|_| |_|\\___|_|"
    echo

    info "Installing Skint..."
    echo

    # Detect platform
    platform=$(detect_platform)
    info "Platform: $platform"

    # Get bin directory
    bin_dir=$(get_bin_dir)
    info "Install directory: $bin_dir"
    echo

    # Get version
    if [ "$VERSION" = "latest" ]; then
        version=$(get_latest_version)
    else
        version="$VERSION"
    fi
    info "Version: $version"

    # Try to download binary, fallback to building from source
    if ! download_and_install "$version" "$platform" "$bin_dir" 2>/dev/null; then
        warn "Binary download failed, building from source..."
        build_from_source "$bin_dir"
    fi

    echo
    success "Skint installation complete!"
    echo

    # Check PATH
    check_path "$bin_dir"
    echo

    # Next steps
    info "Next steps:"
    echo "  skint config      # Configure a provider"
    echo "  skint use native  # Use Claude with native Anthropic"
    echo "  skint --help      # View all commands"
    echo
}

# Run
main "$@"
