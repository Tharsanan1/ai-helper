# Installation Guide for ctl

## Quick Install (Recommended)

### Option 1: Install to /usr/local/bin (macOS/Linux)

```bash
# Clone the repository
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper

# Build the binary
go build -o ctl ./cmd/ctl

# Install to system PATH (requires sudo)
sudo mv ctl /usr/local/bin/

# Verify installation
ctl --version
```

### Option 2: Install to ~/bin (No sudo required)

```bash
# Clone the repository
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper

# Build the binary
go build -o ctl ./cmd/ctl

# Create ~/bin if it doesn't exist
mkdir -p ~/bin

# Move binary to ~/bin
mv ctl ~/bin/

# Add ~/bin to PATH if not already there
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc  # or ~/.bashrc
source ~/.zshrc  # or source ~/.bashrc

# Verify installation
ctl --version
```

### Option 3: Install via Go (if you have Go installed)

```bash
# Install directly from source
go install github.com/tharsanan1/ai-helper/cmd/ctl@latest

# Verify installation (assumes $GOPATH/bin is in PATH)
ctl --version
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
go build -o ctl ./cmd/ctl

# 4. Test the binary
./ctl --help

# 5. Install to your preferred location
# Choose one of the following:

# System-wide (requires sudo)
sudo mv ctl /usr/local/bin/

# User-only
mv ctl ~/bin/  # or any directory in your PATH
```

### Platform-Specific Installation

#### macOS

```bash
# Using Homebrew (if you create a formula - see below)
brew tap tharsanan1/ctl
brew install ctl

# Manual installation
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper
go build -o ctl ./cmd/ctl
sudo mv ctl /usr/local/bin/
```

#### Linux

```bash
# Debian/Ubuntu
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper
go build -o ctl ./cmd/ctl
sudo mv ctl /usr/local/bin/

# Arch (if you create a PKGBUILD)
yay -S ctl-git
```

#### Windows

```powershell
# Using PowerShell
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper
go build -o ctl.exe ./cmd/ctl

# Move to a directory in PATH
Move-Item ctl.exe C:\Windows\System32\

# Or create a custom bin directory
mkdir $HOME\bin
Move-Item ctl.exe $HOME\bin\
$env:PATH += ";$HOME\bin"
```

## Verify Installation

After installation, verify everything works:

```bash
# Check version
ctl --version

# View help
ctl --help

# Test in a git repository
cd /path/to/your/git/repo
ctl worktree list
```

## Post-Installation Setup

### 1. Initialize Configuration (Optional)

```bash
# View default configuration
ctl config list

# Customize settings
ctl config set worktree.base_location ~/worktrees
ctl config set claude.auto_launch true
```

### 2. Configure Claude Integration (Optional)

If Claude CLI is not in your PATH:

```bash
# Find Claude installation
which claude

# Set Claude path in config
ctl config set claude.cli_path /path/to/claude
```

### 3. Shell Completion (Optional)

```bash
# Bash
ctl completion bash > /etc/bash_completion.d/ctl

# Zsh
ctl completion zsh > "${fpath[1]}/_ctl"

# Fish
ctl completion fish > ~/.config/fish/completions/ctl.fish

# PowerShell
ctl completion powershell > ctl.ps1
```

## Updating

### Manual Update

```bash
cd /path/to/ai-helper
git pull origin main
go build -o ctl ./cmd/ctl
sudo mv ctl /usr/local/bin/  # or your install location
```

### Via Go Install

```bash
go install github.com/tharsanan1/ai-helper/cmd/ctl@latest
```

## Uninstallation

```bash
# Remove the binary
sudo rm /usr/local/bin/ctl  # or rm ~/bin/ctl

# Remove configuration (optional)
rm -rf ~/.config/aihelper

# Remove shell completions (optional)
rm /etc/bash_completion.d/ctl  # or relevant completion file
```

## Troubleshooting

### "ctl: command not found"

**Solution:** The binary is not in your PATH.

```bash
# Check where ctl is installed
which ctl

# If not found, add the directory to PATH
export PATH="/path/to/ctl/directory:$PATH"

# Make it permanent
echo 'export PATH="/path/to/ctl/directory:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### "permission denied" when running ctl

**Solution:** Make the binary executable.

```bash
chmod +x /path/to/ctl
```

### Build fails with "go: cannot find main module"

**Solution:** Ensure you're in the project directory and go.mod exists.

```bash
cd /path/to/ai-helper
ls go.mod  # Should exist
go build -o ctl ./cmd/ctl
```

## Creating Distribution Packages

### Homebrew Formula (macOS/Linux)

Create a Homebrew tap:

```ruby
# Formula/ctl.rb
class Ctl < Formula
  desc "Git worktree manager with Claude Code integration"
  homepage "https://github.com/tharsanan1/ai-helper"
  url "https://github.com/tharsanan1/ai-helper/archive/v0.1.0.tar.gz"
  sha256 "YOUR_SHA256_HERE"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"ctl"), "./cmd/ctl"
  end

  test do
    system "#{bin}/ctl", "--version"
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
project_name: ctl
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
    main: ./cmd/ctl
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
2. Configure your settings: `ctl config list`
3. Try creating a worktree: `ctl worktree create test-feature --no-claude`
4. Explore all commands: `ctl --help`

For more help, see the [main README](README.md) or open an issue on GitHub.
