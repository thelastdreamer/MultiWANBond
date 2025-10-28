#!/bin/bash
# MultiWANBond Linux/macOS Installer
# One-click installation script

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Detect OS
OS="unknown"
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
fi

# Detect Linux distribution
DISTRO="unknown"
if [ "$OS" = "linux" ]; then
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        DISTRO=$ID
    fi
fi

echo ""
echo -e "${CYAN}================================================================${NC}"
echo -e "${CYAN}       MultiWANBond Installer for $OS${NC}"
echo -e "${CYAN}================================================================${NC}"
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo -e "${YELLOW}⚠ Please do not run this installer as root/sudo${NC}"
    echo -e "${YELLOW}  The installer will ask for sudo when needed${NC}"
    echo ""
    exit 1
fi

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install package based on distro
install_package() {
    local package=$1

    if [ "$OS" = "macos" ]; then
        if ! command_exists brew; then
            echo -e "${YELLOW}Installing Homebrew...${NC}"
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        fi
        brew install "$package"
    elif [ "$OS" = "linux" ]; then
        case "$DISTRO" in
            ubuntu|debian)
                sudo apt update
                sudo apt install -y "$package"
                ;;
            fedora)
                sudo dnf install -y "$package"
                ;;
            centos|rhel)
                sudo yum install -y "$package"
                ;;
            arch)
                sudo pacman -S --noconfirm "$package"
                ;;
            *)
                echo -e "${RED}Unsupported distribution: $DISTRO${NC}"
                echo -e "${YELLOW}Please install $package manually${NC}"
                return 1
                ;;
        esac
    fi
}

# 1. Check Go installation
echo -e "${CYAN}[1/7] Checking Go installation...${NC}"
GO_INSTALLED=false
GO_VERSION=""

if command_exists go; then
    GO_VERSION=$(go version 2>/dev/null | grep -oP 'go\d+\.\d+' || echo "")

    if [[ $GO_VERSION =~ go([0-9]+)\.([0-9]+) ]]; then
        MAJOR=${BASH_REMATCH[1]}
        MINOR=${BASH_REMATCH[2]}

        if [ "$MAJOR" -ge 1 ] && [ "$MINOR" -ge 21 ]; then
            echo -e "${GREEN}  ✓ Go $MAJOR.$MINOR is installed${NC}"
            GO_INSTALLED=true
        else
            echo -e "${YELLOW}  ⚠ Go $MAJOR.$MINOR is installed but version 1.21+ is required${NC}"
        fi
    fi
else
    echo -e "${YELLOW}  ⚠ Go is not installed${NC}"
fi

if [ "$GO_INSTALLED" = false ]; then
    echo ""
    echo -e "${CYAN}Go 1.21 or later is required to run MultiWANBond.${NC}"
    read -p "Would you like to install Go now? (Y/n): " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
        echo -e "${CYAN}Installing Go...${NC}"

        if [ "$OS" = "macos" ]; then
            install_package go
        elif [ "$OS" = "linux" ]; then
            # Download and install Go manually for better version control
            GO_VERSION="1.22.0"
            GO_ARCH="amd64"

            if [[ $(uname -m) == "aarch64" ]]; then
                GO_ARCH="arm64"
            fi

            GO_URL="https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"

            echo -e "${CYAN}Downloading Go ${GO_VERSION}...${NC}"
            wget -q --show-progress "$GO_URL" -O /tmp/go.tar.gz

            echo -e "${CYAN}Installing Go...${NC}"
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf /tmp/go.tar.gz
            rm /tmp/go.tar.gz

            # Add to PATH
            if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
                echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
            fi
            if ! grep -q '/usr/local/go/bin' ~/.profile; then
                echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
            fi

            export PATH=$PATH:/usr/local/go/bin

            echo -e "${GREEN}  ✓ Go installed successfully${NC}"
        fi
    else
        echo -e "${RED}Go is required. Exiting installer.${NC}"
        exit 1
    fi
fi

# 2. Check Git installation
echo -e "${CYAN}[2/7] Checking Git installation...${NC}"
GIT_INSTALLED=false

if command_exists git; then
    echo -e "${GREEN}  ✓ Git is installed${NC}"
    GIT_INSTALLED=true
else
    echo -e "${YELLOW}  ⚠ Git is not installed${NC}"
fi

if [ "$GIT_INSTALLED" = false ]; then
    echo ""
    echo -e "${CYAN}Git is recommended for easy updates.${NC}"
    read -p "Would you like to install Git now? (Y/n): " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
        install_package git
        GIT_INSTALLED=true
    fi
fi

# 3. Set up Go environment
echo -e "${CYAN}[3/7] Setting up Go environment...${NC}"

GO_PATH="$HOME/go"
GO_MOD_CACHE="$GO_PATH/pkg/mod"

export GOPATH=$GO_PATH
export GOMODCACHE=$GO_MOD_CACHE
export GO111MODULE=on

# Add to shell rc files
for rc in ~/.bashrc ~/.zshrc ~/.profile; do
    if [ -f "$rc" ]; then
        if ! grep -q "GOPATH=" "$rc"; then
            echo "export GOPATH=$GO_PATH" >> "$rc"
            echo "export PATH=\$PATH:\$GOPATH/bin" >> "$rc"
        fi
    fi
done

mkdir -p "$GO_PATH"
mkdir -p "$GO_MOD_CACHE"

echo -e "${GREEN}  ✓ Go environment configured${NC}"

# 4. Download/Update MultiWANBond
echo -e "${CYAN}[4/7] Installing MultiWANBond...${NC}"

INSTALL_DIR="$HOME/.local/share/multiwanbond"
CONFIG_DIR="$HOME/.config/multiwanbond"

if [ -d "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}  ⚠ MultiWANBond is already installed at $INSTALL_DIR${NC}"
    read -p "Would you like to update it? (Y/n): " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
        echo -e "${CYAN}  Updating MultiWANBond...${NC}"
        cd "$INSTALL_DIR"
        if [ "$GIT_INSTALLED" = true ]; then
            git pull
        else
            echo -e "${YELLOW}  Git not installed, skipping update${NC}"
        fi
    fi
else
    echo -e "${CYAN}  Downloading MultiWANBond...${NC}"

    if [ "$GIT_INSTALLED" = true ]; then
        git clone https://github.com/thelastdreamer/MultiWANBond.git "$INSTALL_DIR"
    else
        echo -e "${CYAN}  Downloading ZIP (Git not available)...${NC}"
        mkdir -p "$INSTALL_DIR"
        wget -q --show-progress https://github.com/thelastdreamer/MultiWANBond/archive/refs/heads/main.zip -O /tmp/multiwanbond.zip
        unzip -q /tmp/multiwanbond.zip -d /tmp/
        mv /tmp/MultiWANBond-main/* "$INSTALL_DIR/"
        rm -rf /tmp/multiwanbond.zip /tmp/MultiWANBond-main
    fi

    echo -e "${GREEN}  ✓ MultiWANBond downloaded to $INSTALL_DIR${NC}"
fi

# 5. Download and verify dependencies
echo -e "${CYAN}[5/7] Downloading dependencies...${NC}"
cd "$INSTALL_DIR"

if go mod tidy; then
    echo -e "${GREEN}  ✓ Dependencies downloaded and verified${NC}"
else
    echo -e "${RED}  ✗ Failed to download dependencies${NC}"
    exit 1
fi

# 6. Build MultiWANBond
echo -e "${CYAN}[6/7] Building MultiWANBond...${NC}"

export CGO_ENABLED=0
if go build -ldflags "-s -w" -o "$INSTALL_DIR/multiwanbond" ./cmd/server/main.go; then
    echo -e "${GREEN}  ✓ MultiWANBond built successfully${NC}"
else
    echo -e "${RED}  ✗ Build failed${NC}"
    exit 1
fi

# Make executable
chmod +x "$INSTALL_DIR/multiwanbond"

# Add to PATH
BIN_DIR="$HOME/.local/bin"
mkdir -p "$BIN_DIR"

if [ ! -L "$BIN_DIR/multiwanbond" ]; then
    ln -s "$INSTALL_DIR/multiwanbond" "$BIN_DIR/multiwanbond"
fi

# Add to PATH in rc files
for rc in ~/.bashrc ~/.zshrc ~/.profile; do
    if [ -f "$rc" ]; then
        if ! grep -q "$BIN_DIR" "$rc"; then
            echo "export PATH=\$PATH:$BIN_DIR" >> "$rc"
        fi
    fi
done

export PATH=$PATH:$BIN_DIR

# 7. Create config directory and run setup wizard
echo -e "${CYAN}[7/7] Running setup wizard...${NC}"

mkdir -p "$CONFIG_DIR"

echo ""
echo -e "${GREEN}================================================================${NC}"
echo -e "${GREEN}       Installation Complete!${NC}"
echo -e "${GREEN}================================================================${NC}"
echo ""
echo -e "${GREEN}MultiWANBond has been installed to: $INSTALL_DIR${NC}"
echo -e "${GREEN}Configuration will be stored in: $CONFIG_DIR${NC}"
echo ""

# Run setup wizard
echo -e "${CYAN}Starting setup wizard...${NC}"
echo ""

"$INSTALL_DIR/multiwanbond" setup --config "$CONFIG_DIR/config.json"

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}================================================================${NC}"
    echo ""
    echo -e "${GREEN}Setup complete! MultiWANBond is ready to use.${NC}"
    echo ""
    echo -e "${CYAN}To start MultiWANBond:${NC}"
    echo "  multiwanbond --config $CONFIG_DIR/config.json"
    echo ""
    echo -e "${CYAN}To manage configuration:${NC}"
    echo "  multiwanbond config --help"
    echo ""
    echo -e "${CYAN}To add/remove WAN interfaces later:${NC}"
    echo "  multiwanbond wan add"
    echo "  multiwanbond wan remove"
    echo "  multiwanbond wan list"
    echo ""
    echo -e "${CYAN}Note: You may need to restart your shell or run:${NC}"
    echo "  source ~/.bashrc  # or ~/.zshrc"
    echo ""
    echo -e "${GREEN}================================================================${NC}"
else
    echo -e "${YELLOW}Setup wizard was cancelled or encountered an error.${NC}"
    echo -e "${CYAN}You can run it again with:${NC}"
    echo "  multiwanbond setup"
fi

echo ""
