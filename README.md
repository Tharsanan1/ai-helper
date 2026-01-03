# ctl - Git Worktree Manager with Claude Code Integration

`ctl` is a command-line tool for managing git worktrees with seamless integration for launching development tools like Claude Code CLI. It simplifies the workflow of creating isolated worktrees for feature development and automatically launches Claude in the worktree context.

## 🚀 Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash
```

<details>
<summary>What happens during installation?</summary>

The installer will:
- ✅ Check for Go and Git
- ✅ Ask where to install (system-wide or user-only)
- ✅ Build and install the `ctl` binary
- ✅ Optionally set up shell completion
- ✅ Verify the installation works

No sudo required if you choose user-only installation!
</details>

**More options:** [Alternative installation methods](#installation) | [Detailed guide](INSTALL.md)

## Features

- **Easy Worktree Management**: Create, list, remove, and switch between git worktrees with simple commands
- **Claude Code Integration**: Automatically launch Claude Code CLI in worktree directories
- **Flexible Configuration**: Global and per-repository configuration support
- **Smart Defaults**: Sensible defaults with extensive customization options
- **Clean CLI Interface**: Intuitive command structure following modern CLI conventions

## Installation

### Quick Install (One-Liner)

```bash
curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash
```

Or download and run the install script:

```bash
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper
./install.sh
```

### Using Make

```bash
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper

# Install system-wide (requires sudo)
make install

# Or install to ~/bin (no sudo)
make install-user
```

### From Source

```bash
# Clone the repository
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper

# Build the binary
go build -o ctl ./cmd/ctl

# Move to a directory in your PATH
sudo mv ctl /usr/local/bin/

# Verify installation
ctl --version
```

### Requirements

- Go 1.24 or later
- Git 2.5 or later (for worktree support)
- Claude Code CLI (optional, for Claude integration)

📖 **Detailed installation instructions**: See [INSTALL.md](INSTALL.md) for platform-specific guides, troubleshooting, and more options.

## Quick Start

### Create a Worktree

Create a new worktree and launch Claude:

```bash
# Create a worktree named 'feature-auth' with a branch of the same name
ctl worktree create feature-auth

# Create with a custom branch name
ctl worktree create feature-auth -b auth/user-login

# Create from a specific source branch
ctl worktree create hotfix -f main -b hotfix/security-patch

# Create without launching Claude
ctl worktree create experiment --no-claude
```

### List Worktrees

```bash
ctl worktree list
```

Output:
```
NAME                 BRANCH                         COMMIT     PATH
--------------------------------------------------------------------------------
feature-auth         auth/user-login                a1b2c3d    /Users/you/Documents/.worktrees/ai-helper/feature-auth
experiment           experiment                     e4f5g6h    /Users/you/Documents/.worktrees/ai-helper/experiment
```

### Remove a Worktree

```bash
# Remove worktree only
ctl worktree remove feature-auth

# Remove worktree and delete the branch
ctl worktree remove feature-auth --delete-branch
```

### Switch to a Worktree

Launch Claude in an existing worktree:

```bash
ctl worktree switch feature-auth
```

## Configuration

### Configuration Files

`ctl` uses a layered configuration system:

1. **Global config**: `~/.config/ctl/config.yaml`
2. **Repository config**: `.ctl.yaml` (in repository root)
3. **Command-line flags**: Override config file settings

Priority: Flags > Repository config > Global config > Defaults

### Configuration Options

```yaml
worktree:
  # Base directory for worktrees (relative to repo parent or absolute)
  base_location: "../.worktrees"

  # Automatically delete branch when removing worktree
  auto_cleanup: true

  # Default source branch for new worktrees
  default_source_branch: "main"

claude:
  # Default mode when launching Claude (agent or chat)
  default_mode: "agent"

  # Automatically launch Claude after creating worktree
  auto_launch: true

  # Additional arguments to pass to Claude CLI
  extra_args: []

  # Path to Claude CLI (auto-detected if not specified)
  cli_path: ""

global:
  # Verbosity level (0=quiet, 1=normal, 2=verbose)
  verbosity: 1

  # Colorize output
  color: true

  # Editor for opening files
  editor: ""
```

### Manage Configuration

```bash
# View all configuration
ctl config list

# Get a specific value
ctl config get worktree.base_location

# Set a value
ctl config set claude.auto_launch false
ctl config set worktree.base_location /custom/path
```

## Command Reference

### Global Flags

- `--verbose, -v`: Enable verbose output
- `--dry-run`: Show what would happen without executing
- `--no-color`: Disable colored output
- `--config <path>`: Specify config file location

### Worktree Commands

#### `ctl worktree create <name> [flags]`

Create a new git worktree.

**Flags:**
- `-b, --branch <name>`: Branch name (default: same as worktree name)
- `-l, --location <path>`: Base directory for worktrees
- `-f, --from <branch>`: Source branch to create from (default: current branch)
- `--no-claude`: Don't launch Claude after creation
- `--claude-mode <mode>`: Claude mode (agent|chat)
- `--claude-args <args>`: Additional Claude CLI arguments
- `-e, --existing-branch`: Use existing branch instead of creating new

**Examples:**
```bash
# Basic usage
ctl worktree create feature-payment

# Custom branch name
ctl worktree create feature-payment -b payment/stripe-integration

# From specific branch
ctl worktree create bugfix-123 -f develop

# Use existing branch
ctl worktree create feature-x -b existing-feature --existing-branch
```

#### `ctl worktree list`

List all worktrees with their branches and paths.

**Aliases:** `ls`

#### `ctl worktree remove <name> [flags]`

Remove a worktree.

**Flags:**
- `-d, --delete-branch`: Also delete the associated git branch

**Aliases:** `rm`, `delete`

**Examples:**
```bash
# Remove worktree only
ctl worktree remove feature-payment

# Remove worktree and delete branch
ctl worktree remove feature-payment -d
```

#### `ctl worktree switch <name> [flags]`

Switch to an existing worktree and launch Claude.

**Flags:**
- `--claude-mode <mode>`: Claude mode (agent|chat)
- `--claude-args <args>`: Additional Claude CLI arguments

### Config Commands

#### `ctl config list`

Display all configuration values.

**Aliases:** `ls`, `show`

#### `ctl config get <key>`

Get a specific configuration value.

**Examples:**
```bash
ctl config get worktree.base_location
ctl config get claude.auto_launch
```

#### `ctl config set <key> <value>`

Set a configuration value in global config.

**Flags:**
- `-g, --global`: Set in global config (default)

**Examples:**
```bash
ctl config set worktree.base_location /custom/worktrees
ctl config set claude.auto_launch false
ctl config set global.verbosity 2
```

## Worktree Location Strategy

By default, worktrees are created at:

```
../.worktrees/<repo-name>/<worktree-name>
```

For example, if your repository is at `/Users/you/projects/my-app`:
- Main repo: `/Users/you/projects/my-app`
- Worktree location: `/Users/you/projects/.worktrees/my-app/feature-auth`

This keeps worktrees:
- Outside the main repository (cleaner, no .gitignore issues)
- Organized by repository (supports multiple projects)
- Easy to find and manage

You can customize this location globally or per-command:

```bash
# Global config
ctl config set worktree.base_location /custom/path

# Per-command
ctl worktree create feature-x -l /custom/path
```

## Development

### Project Structure

```
ai-helper/
├── cmd/ctl/                  # Application entry point
├── internal/
│   ├── cli/                  # CLI commands
│   │   ├── config/          # Config commands
│   │   └── worktree/        # Worktree commands
│   ├── config/              # Configuration management
│   ├── git/                 # Git operations
│   ├── launcher/            # Tool launcher interface
│   ├── worktree/            # Worktree business logic
│   └── util/                # Shared utilities
└── pkg/
    └── errors/              # Custom error types
```

### Building

```bash
# Build for current platform
go build -o ctl ./cmd/ctl

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o ctl-linux-amd64 ./cmd/ctl
GOOS=darwin GOARCH=arm64 go build -o ctl-darwin-arm64 ./cmd/ctl
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Verbose output
go test -v ./...
```

## Troubleshooting

### Claude CLI Not Found

If `ctl` can't find the Claude CLI:

1. Ensure Claude is installed and in your PATH:
   ```bash
   which claude
   ```

2. Specify the path in config:
   ```bash
   ctl config set claude.cli_path /path/to/claude
   ```

### Worktree Already Exists

If you get an error that a worktree already exists:

```bash
# List existing worktrees
ctl worktree list

# Remove the existing worktree
ctl worktree remove <name>

# Or switch to it
ctl worktree switch <name>
```

### Permission Denied

If you get permission errors when creating worktrees:

1. Check the base location has write permissions:
   ```bash
   ls -la ../.worktrees
   ```

2. Use a different location:
   ```bash
   ctl worktree create <name> -l ~/worktrees
   ```

## Future Enhancements

The architecture supports easy addition of:

- **New tool integrations**: VS Code, Cursor, Zed, etc.
- **Project templates**: Initialize worktrees with project scaffolding
- **Hook system**: Run custom scripts on worktree lifecycle events
- **Remote worktree support**: Manage worktrees on remote servers

## Updating

To update to the latest version:

```bash
# Quick update
curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash

# Or manually
cd /path/to/ai-helper
git pull origin main
make install
```

Check your version: `ctl --version`

📖 **Complete update guide**: See [UPDATING.md](UPDATING.md)

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

### For Maintainers

See [UPDATING.md](UPDATING.md) for the release workflow.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Configuration powered by [Viper](https://github.com/spf13/viper)
- Inspired by modern development workflows and the need for better worktree management
