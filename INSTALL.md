# Installation Guide for aihelper

## Quick Install (Recommended)

### Option 1: Install to /usr/local/bin (macOS/Linux)

```bash
# Clone the repository
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper

# Build the binary
go build -o aihelper ./cmd/aihelper

# Install to system PATH (requires sudo)
sudo mv aihelper /usr/local/bin/

# Verify installation
aihelper --version
```

### Option 2: Install to ~/bin (No sudo required)

```bash
# Clone the repository
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper

# Build the binary
go build -o aihelper ./cmd/aihelper

# Create ~/bin if it doesn't exist
mkdir -p ~/bin

# Move binary to ~/bin
mv aihelper ~/bin/

# Add ~/bin to PATH if not already there
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc  # or ~/.bashrc
source ~/.zshrc  # or source ~/.bashrc

# Verify installation
aihelper --version
```

### Option 3: Install via Go (if you have Go installed)

```bash
# Install directly from source
go install github.com/tharsanan1/ai-helper/cmd/aihelper@latest

# Verify installation (assumes $GOPATH/bin is in PATH)
aihelper --version
```

## Detailed Installation Steps

### Prerequisites

1. **Git** (version 2.5+)
   ```bash
   git --version
   ```

2. **Go** (version 1.24+) - Required for building from source
   ```bash
   go version
   ```

3. **Claude Code CLI** (optional - for Claude integration)
   ```bash
   which claude
   ```

### Build from Source

```bash
# 1. Clone the repository
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper

# 2. Download dependencies
go mod download

# 3. Build the binary
go build -o aihelper ./cmd/aihelper

# 4. Test the binary
./aihelper --help

# 5. Install to your preferred location
# Choose one of the following:

# System-wide (requires sudo)
sudo mv aihelper /usr/local/bin/

# User-only
mv aihelper ~/bin/  # or any directory in your PATH
```

### Platform-Specific Installation

#### macOS

```bash
# Using Homebrew (if you create a formula - see below)
brew tap tharsanan1/aihelper
brew install aihelper

# Manual installation
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper
go build -o aihelper ./cmd/aihelper
sudo mv aihelper /usr/local/bin/
```

#### Linux

```bash
# Debian/Ubuntu
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper
go build -o aihelper ./cmd/aihelper
sudo mv aihelper /usr/local/bin/

# Arch (if you create a PKGBUILD)
yay -S aihelper-git
```

#### Windows

```powershell
# Using PowerShell
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper
go build -o aihelper.exe ./cmd/aihelper

# Move to a directory in PATH
Move-Item aihelper.exe C:\Windows\System32\

# Or create a custom bin directory
mkdir $HOME\bin
Move-Item aihelper.exe $HOME\bin\
$env:PATH += ";$HOME\bin"
```

## Verify Installation

After installation, verify everything works:

```bash
# Check version
aihelper --version

# View help
aihelper --help

# Test in a git repository
cd /path/to/your/git/repo
aihelper worktree list
```

## Post-Installation Setup

### 1. Initialize Configuration (Optional)

```bash
# View default configuration
aihelper config list

# Customize settings
aihelper config set worktree.base_location ~/worktrees
aihelper config set claude.auto_launch true
```

### 2. Configure Claude Integration (Optional)

If Claude CLI is not in your PATH:

```bash
# Find Claude installation
which claude

# Set Claude path in config
aihelper config set claude.cli_path /path/to/claude
```

### 3. Shell Completion (Recommended)

AI Helper supports shell completion for Bash, Zsh, Fish, and PowerShell. This provides tab completion for commands, flags, and even dynamic values (like Claude modes).

**Note for Zsh Users:** AI Helper includes rich Zsh completion support, including descriptions for flags and commands!

#### Zsh (Recommended Setup)

To enable rich Zsh completion with descriptions:

```bash
# 1. Create a completion directory
mkdir -p ~/.zsh/completions

# 2. Generate completion script
aihelper completion zsh > ~/.zsh/completions/_aihelper

# 3. Add to your ~/.zshrc (if not already present):
echo 'fpath=(~/.zsh/completions $fpath)' >> ~/.zshrc
echo 'autoload -U compinit; compinit' >> ~/.zshrc

# 4. Reload shell
source ~/.zshrc
```

#### Bash

```bash
# Linux
aihelper completion bash > /etc/bash_completion.d/aihelper

# macOS (Homebrew)
aihelper completion bash > $(brew --prefix)/etc/bash_completion.d/aihelper
```

#### Fish

```bash
aihelper completion fish > ~/.config/fish/completions/aihelper.fish
```

#### PowerShell

```powershell
aihelper completion powershell > aihelper.ps1
```

## Updating

### Manual Update

```bash
cd /path/to/ai-helper
git pull origin main
go build -o aihelper ./cmd/aihelper
sudo mv aihelper /usr/local/bin/  # or your install location
```

### Via Go Install

```bash
go install github.com/tharsanan1/ai-helper/cmd/aihelper@latest
```

## Uninstallation

```bash
# Remove the binary
sudo rm /usr/local/bin/aihelper  # or rm ~/bin/aihelper

# Remove configuration (optional)
rm -rf ~/.config/aihelper

# Remove shell completions (optional)
rm ~/.zsh/completions/_aihelper  # or relevant completion file
```

## Troubleshooting

### "aihelper: command not found"

**Solution:** The binary is not in your PATH.

```bash
# Check where aihelper is installed
which aihelper

# If not found, add the directory to PATH
export PATH="/path/to/aihelper/directory:$PATH"

# Make it permanent
echo 'export PATH="/path/to/aihelper/directory:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### "permission denied" when running aihelper

**Solution:** Make the binary executable.

```bash
chmod +x /path/to/aihelper
```

### Build fails with "go: cannot find main module"

**Solution:** Ensure you're in the project directory and go.mod exists.

```bash
cd /path/to/ai-helper
ls go.mod  # Should exist
go build -o aihelper ./cmd/aihelper
```

## Creating Distribution Packages

### Homebrew Formula (macOS/Linux)

Create a Homebrew tap:

```ruby
# Formula/aihelper.rb
class Aihelper < Formula
  desc "Git worktree manager with Claude Code integration"
  homepage "https://github.com/tharsanan1/ai-helper"
  url "https://github.com/tharsanan1/ai-helper/archive/v0.1.0.tar.gz"
  sha256 "YOUR_SHA256_HERE"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"aihelper"), "./cmd/aihelper"
  end

  test do
    system "#{bin}/aihelper", "--version"
  end
end
```

### GitHub Releases with Binaries

Use GoReleaser to create release binaries:

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Create .goreleaser.yml
cat > .goreleaser.yml <<EOF
project_name: aihelper
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    main: ./cmd/aihelper
archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: 'checksums.txt'
release:
  github:
    owner: tharsanan
    name: ai-helper
EOF

# Create release
git tag v0.1.0
goreleaser release
```

Then users can download pre-built binaries from GitHub releases!

## Next Steps

After installation:

1. Read the [Quick Start Guide](README.md#quick-start)
2. Configure your settings: `aihelper config list`
3. Try creating a worktree: `aihelper worktree create test-feature --no-claude`
4. Explore all commands: `aihelper --help`

For more help, see the [main README](README.md) or open an issue on GitHub.
