#!/bin/bash
#
# Installation script for aihelper - AI-Powered Git Worktree Manager
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash
#   or
#   ./install.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default installation directory
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
USE_SUDO="true"

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

echo -e "${BLUE}aihelper Installation Script${NC}"
echo "========================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed.${NC}"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}✓${NC} Found Go version: $GO_VERSION"

# Check if git is installed
if ! command -v git &> /dev/null; then
    echo -e "${RED}Error: Git is not installed.${NC}"
    echo "Please install Git first."
    exit 1
fi

GIT_VERSION=$(git --version | awk '{print $3}')
echo -e "${GREEN}✓${NC} Found Git version: $GIT_VERSION"

# Determine installation directory
echo ""
echo "Installation directory options:"
echo "  1) /usr/local/bin (requires sudo, system-wide)"
echo "  2) ~/bin (no sudo, user-only)"
echo "  3) Custom directory"
echo ""
read -p "Choose installation directory [1]: " INSTALL_CHOICE

case $INSTALL_CHOICE in
    2)
        INSTALL_DIR="$HOME/bin"
        USE_SUDO="false"
        mkdir -p "$INSTALL_DIR"
        ;;
    3)
        read -p "Enter custom directory path: " CUSTOM_DIR
        INSTALL_DIR="$CUSTOM_DIR"
        USE_SUDO="false"
        mkdir -p "$INSTALL_DIR"
        ;;
    *)
        INSTALL_DIR="/usr/local/bin"
        USE_SUDO="true"
        ;;
esac

echo -e "Installing to: ${YELLOW}$INSTALL_DIR${NC}"

# Check if installation directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH${NC}"
    if [ "$USE_SUDO" = "false" ]; then
        echo "Add this line to your ~/.bashrc or ~/.zshrc:"
        echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
fi

# Clone or use existing repository
REPO_URL="https://github.com/tharsanan1/ai-helper.git"
TEMP_DIR=$(mktemp -d)
BUILD_DIR="$TEMP_DIR/ai-helper"

echo ""
echo "Downloading source code..."

if [ -d ".git" ] && git remote -v | grep -q "ai-helper"; then
    # We're already in the repo
    echo -e "${GREEN}✓${NC} Using local repository"
    BUILD_DIR="$(pwd)"
else
    # Clone the repository
    echo "Cloning from $REPO_URL..."
    git clone --depth 1 "$REPO_URL" "$BUILD_DIR" || {
        echo -e "${RED}Error: Failed to clone repository${NC}"
        exit 1
    }
    echo -e "${GREEN}✓${NC} Repository cloned"
fi

# Build the binary
echo ""
echo "Building aihelper..."
cd "$BUILD_DIR"

go build -o aihelper -ldflags="-s -w" ./cmd/aihelper || {
    echo -e "${RED}Error: Build failed${NC}"
    exit 1
}

echo -e "${GREEN}✓${NC} Build successful"

# Install the binary
echo ""
echo "Installing aihelper to $INSTALL_DIR..."

if [ "$USE_SUDO" = "true" ]; then
    sudo mv aihelper "$INSTALL_DIR/" || {
        echo -e "${RED}Error: Installation failed${NC}"
        echo "Try running with sudo or choose a different installation directory"
        exit 1
    }
    sudo chmod +x "$INSTALL_DIR/aihelper"
else
    mv aihelper "$INSTALL_DIR/" || {
        echo -e "${RED}Error: Installation failed${NC}"
        exit 1
    }
    chmod +x "$INSTALL_DIR/aihelper"
fi

echo -e "${GREEN}✓${NC} Installation complete"

# Cleanup
if [ "$BUILD_DIR" != "$(pwd)" ]; then
    rm -rf "$TEMP_DIR"
fi

# Verify installation
echo ""
echo "Verifying installation..."

if command -v aihelper &> /dev/null; then
    AIHELPER_VERSION=$(aihelper --version 2>&1 | head -n1)
    echo -e "${GREEN}✓${NC} aihelper is installed: $AIHELPER_VERSION"
else
    echo -e "${YELLOW}Warning: aihelper command not found in PATH${NC}"
    echo "You may need to:"
    echo "  1. Add $INSTALL_DIR to your PATH"
    echo "  2. Restart your terminal"
fi

# Check for Claude CLI
echo ""
if command -v claude &> /dev/null; then
    CLAUDE_PATH=$(which claude)
    echo -e "${GREEN}✓${NC} Claude CLI found at: $CLAUDE_PATH"
else
    echo -e "${YELLOW}Note: Claude CLI not found${NC}"
    echo "Claude Code integration is optional. You can:"
    echo "  - Use aihelper without Claude (--no-claude flag)"
    echo "  - Install Claude CLI later and configure it with:"
    echo "    aihelper config set claude.cli_path /path/to/claude"
fi

# Post-installation instructions
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Installation successful!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Next steps:"
echo "  1. Run 'aihelper --help' to see available commands"
echo "  2. Run 'aihelper config list' to view configuration"
echo "  3. Try creating a worktree:"
echo "     cd /path/to/git/repo"
echo "     aihelper worktree create my-feature --no-claude"
echo ""
echo "For more information, visit:"
echo "  https://github.com/tharsanan1/ai-helper"
echo ""

# Offer to set up shell completion
echo -e "Would you like to set up shell completion? (y/n)"
read -p "> " SETUP_COMPLETION

if [[ "$SETUP_COMPLETION" =~ ^[Yy]$ ]]; then
    SHELL_NAME=$(basename "$SHELL")

    case "$SHELL_NAME" in
        bash)
            COMPLETION_DIR="/etc/bash_completion.d"
            if [ -d "$COMPLETION_DIR" ]; then
                sudo aihelper completion bash > "$COMPLETION_DIR/aihelper" 2>/dev/null || true
                echo -e "${GREEN}✓${NC} Bash completion installed"
                echo "Run 'source ~/.bashrc' or restart your terminal"
            else
                echo "Bash completion directory not found. Skipping."
            fi
            ;;
        zsh)
            COMPLETION_FILE="$HOME/.zsh/completions/_aihelper"
            mkdir -p "$(dirname "$COMPLETION_FILE")"
            aihelper completion zsh > "$COMPLETION_FILE" 2>/dev/null || true
            echo -e "${GREEN}✓${NC} Zsh completion installed"
            echo "Add this to your ~/.zshrc if not already there:"
            echo "  fpath=(~/.zsh/completions \$fpath)"
            echo "  autoload -Uz compinit && compinit"
            ;;
        fish)
            COMPLETION_DIR="$HOME/.config/fish/completions"
            mkdir -p "$COMPLETION_DIR"
            aihelper completion fish > "$COMPLETION_DIR/aihelper.fish" 2>/dev/null || true
            echo -e "${GREEN}✓${NC} Fish completion installed"
            ;;
        *)
            echo "Shell completion not available for $SHELL_NAME"
            ;;
    esac
fi

echo ""
echo -e "${BLUE}Happy coding! 🚀${NC}"
