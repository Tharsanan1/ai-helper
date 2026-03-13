# AI Helper - Git Worktree Manager with Claude Code Integration

`aihelper` is a command-line tool for managing git worktrees with seamless integration for launching development tools like Claude Code CLI. It simplifies the workflow of creating isolated worktrees for feature development and automatically launches Claude in the worktree context.

## 🚀 Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash
```

<details>
<summary>What happens during installation?</summary>

The installer will:
- ✅ Check for Go and Git
- ✅ Ask where to install (system-wide or user-only)
- ✅ Build and install the `aihelper` binary
- ✅ Optionally set up shell completion (including rich Zsh support!)
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
- **Rich Shell Completion**: Tab completion with descriptions for Zsh, plus Bash/Fish/PowerShell support

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
go build -o aihelper ./cmd/aihelper

# Move to a directory in your PATH
sudo mv aihelper /usr/local/bin/

# Verify installation
aihelper --version
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
aihelper worktree create feature-auth

# Create with a custom branch name
aihelper worktree create feature-auth -b auth/user-login

# Create from a specific source branch
aihelper worktree create hotfix -f main -b hotfix/security-patch

# Create without launching Claude
aihelper worktree create experiment --no-claude
```

### List Worktrees

```bash
aihelper worktree list
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
aihelper worktree remove feature-auth

# Remove worktree and delete the branch
aihelper worktree remove feature-auth --delete-branch
```

### Switch to a Worktree

Launch Claude in an existing worktree:

```bash
aihelper worktree switch feature-auth
```

### Launch in Current Directory

Launch an AI tool in the current directory without creating a worktree:

```bash
# Launch Claude (default)
aihelper launch

# Launch with Kimi APIs
aihelper l --kimi

# Launch Gemini
aihelper run --gemini
```

### Peer Test Workflows

Prepare a versioned peer test workspace from a configured product pack:

```bash
aihelper peertest \
  --product-version 4.4.0 \
  --peertest-issue https://github.com/wso2-enterprise/wso2-apim-internal/issues/15426 \
  --username you@example.com \
  --password '<secret>'
```

Run an already prepared peer test pack:

```bash
aihelper peertest \
  --run \
  --product-version 4.4.0 \
  --peertest-issue https://github.com/wso2-enterprise/wso2-apim-internal/issues/15426
```

Run the version-specific browser smoke test:

```bash
aihelper peertest \
  --smoketest \
  --product-version 4.4.0 \
  --headless
```

By default, smoke test inputs such as tenant domain, admin credentials, tenant user credentials, and screenshot delay are loaded from `peertest.products."<version>".smoketest` in the config file. Flags override those values when needed.

The `4.4.0` smoke test currently covers:
- tenant creation
- tenant user creation
- API creation and publish in Publisher
- Dev Portal subscription
- production key generation
- API Console GET execution

## Configuration

### Configuration Files

`aihelper` uses a layered configuration system:

1. **Global config**: `~/.config/aihelper/config.yaml`
2. **Repository config**: `.aihelper.yaml` (in repository root)
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

  # Default CLI to launch (claude, gemini, copilot, droid, opencode)
  default_cli: "claude"

peertest:
  products:
    "4.4.0":
      pack_path: "~/Documents/wso2/apim/4.4.0/wso2am-4.4.0.13.zip"
      workspace_root: "~/Documents/wso2/apim/4.4.0/peertests"
      working_dir: "bin"
      steps:
        - "./wso2update_darwin -u {{username}} -p {{password}}"
        - "./wso2update_darwin"
        - "export WSO2_UPDATES_UPDATE_LEVEL_STATE=TESTING"
        - "./wso2update_darwin"
        - 'grep "Applied " ../updates/logs/wso2update-{{today}}.log'
      run_working_dir: "bin"
      run_steps:
        - "sh api-manager.sh"
      smoketest:
        base_url: "https://localhost:9443"
        admin_user: "admin"
        admin_password: "admin"
        tenant_domain: "peertest9.com"
        tenant_admin_user: "peer9"
        tenant_admin_password: "peer1"
        tenant_admin_email: "peer@peertest9.com"
        tenant_admin_first_name: "peer"
        tenant_admin_last_name: "admin"
        tenant_user: "peertestuser9"
        tenant_user_password: "peer1"
        api_endpoint: "https://httpbin.org/anything"
        api_name_prefix: "PeerTestAPI"
        api_version: "1.0.0"
        screenshot_dir: "/Users/tharsanan/Documents/tmp/peertest-smoketest-shots-9"
        screenshot_delay_ms: 1000
        slow_mo: 250
```

### Manage Configuration

```bash
# View all configuration
aihelper config list

# Get a specific value
aihelper config get worktree.base_location

# Set a value
aihelper config set claude.auto_launch false
aihelper config set worktree.base_location /custom/path
```

### Switching Between AI CLIs

By default, `aihelper` launches Claude. You can use other AI tools or configure a default:

**Command-line flags** (override defaults):
```bash
# Launch Gemini
aihelper worktree create feature-x --gemini

# Launch Copilot
aihelper worktree create feature-x --copilot

# Launch Droid
aihelper worktree create feature-x --droid

# Launch OpenCode
aihelper worktree create feature-x --opencode

# Explicitly launch Claude (useful for overriding config defaults)
aihelper worktree create feature-x --claude
```

**Configuration-based default** (applies to both `create` and `switch` commands):
```bash
# Set default CLI to Gemini
aihelper config set global.default_cli gemini

# Set default CLI to Copilot
aihelper config set global.default_cli copilot

# Reset to Claude (default)
aihelper config set global.default_cli claude
```

Supported values for `global.default_cli`: `claude`, `gemini`, `copilot`, `droid`, `opencode`

## Bash Aliases

To speed up your workflow, add these aliases to your `~/.bashrc` or `~/.zshrc`:

### Basic Aliases

```bash
# Worktree shortcuts - use single letter commands
alias ctl='aihelper'                      # If you prefer 'ctl' as the main command
alias wc='aihelper c'                     # Create worktree
alias ws='aihelper sw'                    # Switch worktree
alias wl='aihelper ls'                    # List worktrees
alias wr='aihelper r'                     # Remove worktree
alias wlch='aihelper l'                   # Launch in current dir

# Config shortcuts
alias cfg='aihelper config'               # Config command
alias cfgl='aihelper config list'         # List config
alias cfgs='aihelper config set'          # Set config
alias cfgg='aihelper config get'          # Get config
```

### Advanced Aliases with Pre-configured Options

```bash
# Create with Claude in agent mode (default launcher)
alias wca='aihelper c'                    # Default creates with Claude

# Create with specific AI tools
alias wcc='aihelper c'                    # Claude (default)
alias wco='aihelper c --opencode'         # OpenCode
alias wcg='aihelper c --gemini'           # Gemini
alias wcd='aihelper c --droid'            # Droid
alias wcp='aihelper c --copilot'          # Copilot

# Create without launching any AI tool
alias wcn='aihelper c --no-claude'        # No launcher

# Create and open in new terminal
alias wcnt='aihelper c --new-terminal'    # New terminal window

# Switch with specific AI tools
alias wso='aihelper sw --opencode'        # Switch + OpenCode
alias wsg='aihelper sw --gemini'          # Switch + Gemini
alias wsd='aihelper sw --droid'           # Switch + Droid
alias wsp='aihelper sw --copilot'         # Switch + Copilot

# Create from a specific branch
alias wcm='aihelper c -f main'            # Create from main branch
alias wcd='aihelper c -f develop'         # Create from develop branch

# Delete worktree with branch cleanup
alias wrd='aihelper r --delete-branch'    # Remove + delete branch
```

### Usage Examples

```bash
# Create a feature branch and launch Claude
wc feature-auth

# Create from main branch with custom branch name
wc feature-payment -b payment/stripe -f main

# Switch to existing worktree
ws feature-auth

# List all worktrees
wl

# Remove worktree and delete the branch
wrd feature-auth

# Create with OpenCode instead of Claude
wco feature-mobile

# View configuration
cfgl

# Set a config value
cfgs claude.auto_launch false
```

### One-liner Installation

Add these to your shell config file:

```bash
# For ~/.bashrc or ~/.zshrc
cat >> ~/.bashrc << 'EOF'
# aihelper aliases
alias wc='aihelper c'
alias ws='aihelper sw'
alias wl='aihelper ls'
alias wr='aihelper r'
alias wca='aihelper c'
alias wco='aihelper c --opencode'
alias wcg='aihelper c --gemini'
alias wcd='aihelper c --droid'
alias wcp='aihelper c --copilot'
alias wcn='aihelper c --no-claude'
alias wso='aihelper sw --opencode'
alias wsg='aihelper sw --gemini'
alias wsd='aihelper sw --droid'
alias wsp='aihelper sw --copilot'
alias cfg='aihelper config'
alias cfgl='aihelper config list'
alias cfgs='aihelper config set'
alias cfgg='aihelper config get'
EOF
source ~/.bashrc
```

Then reload your shell:

```bash
source ~/.bashrc
```

### Command Reference

### Global Flags

- `--verbose, -v`: Enable verbose output
- `--dry-run`: Show what would happen without executing
- `--no-color`: Disable colored output
- `--config <path>`: Specify config file location

### Worktree Commands

#### `aihelper worktree create <name> [flags]`

Create a new git worktree.

**Flags:**
- `-b, --branch <name>`: Branch name (default: same as worktree name)
- `-l, --location <path>`: Base directory for worktrees
- `-f, --from <branch>`: Source branch to create from (default: current branch)
- `--no-claude`: Don't launch Claude after creation
- `--claude`: Explicitly launch Claude (useful for overriding `default_cli` config)
- `--claude-mode <mode>`: Claude mode (agent|chat)
- `--claude-args <args>`: Additional Claude CLI arguments
- `--gemini`: Launch Gemini CLI instead of Claude
- `--copilot`: Launch Copilot CLI instead of Claude
- `--droid`: Launch Droid CLI instead of Claude
- `--opencode`: Launch OpenCode instead of Claude
- `-e, --existing-branch`: Use existing branch instead of creating new

**Examples:**
```bash
# Basic usage
aihelper worktree create feature-payment

# Custom branch name
aihelper worktree create feature-payment -b payment/stripe-integration

# From specific branch
aihelper worktree create bugfix-123 -f develop

# Use existing branch
aihelper worktree create feature-x -b existing-feature --existing-branch
```

#### `aihelper worktree list`

List all worktrees with their branches and paths.

**Aliases:** `ls`

#### `aihelper worktree remove <name> [flags]`

Remove a worktree.

**Flags:**
- `-d, --delete-branch`: Also delete the associated git branch

**Aliases:** `rm`, `delete`

**Examples:**
```bash
# Remove worktree only
aihelper worktree remove feature-payment

# Remove worktree and delete branch
aihelper worktree remove feature-payment -d
```

#### `aihelper worktree switch <name> [flags]`

Switch to an existing worktree and launch Claude (or other AI CLI based on config).

**Flags:**
- `--claude`: Explicitly launch Claude (useful for overriding `default_cli` config)
- `--claude-mode <mode>`: Claude mode (agent|chat)
- `--claude-args <args>`: Additional Claude CLI arguments
- `--gemini`: Launch Gemini CLI instead of Claude
- `--copilot`: Launch Copilot CLI instead of Claude
- `--droid`: Launch Droid CLI instead of Claude
- `--opencode`: Launch OpenCode instead of Claude

#### `aihelper launch [flags]`

Launch an AI tool in the current directory.

**Flags:**
- `--claude`: Explicitly launch Claude
- `--claude-mode <mode>`: Claude mode (agent|chat)
- `--claude-args <args>`: Additional Claude CLI arguments
- `--minimax`: Launch Claude with Minimax APIs
- `--glm`: Launch Claude with GLM APIs
- `--kimi`: Launch Claude with Kimi APIs
- `--gemini`: Launch Gemini CLI
- `--copilot`: Launch Copilot CLI
- `--droid`: Launch Droid CLI
- `--opencode`: Launch OpenCode
- `--new-terminal`: Launch in a new terminal window
- `--sandbox`: Launch in a Docker sandbox

**Aliases:** `l`, `run`

### Config Commands

#### `aihelper config list`

Display all configuration values.

**Aliases:** `ls`, `show`

#### `aihelper config get <key>`

Get a specific configuration value.

**Examples:**
```bash
aihelper config get worktree.base_location
aihelper config get claude.auto_launch
```

#### `aihelper config set <key> <value>`

Set a configuration value in global config.

**Flags:**
- `-g, --global`: Set in global config (default)

**Examples:**
```bash
aihelper config set worktree.base_location /custom/worktrees
aihelper config set claude.auto_launch false
aihelper config set global.verbosity 2
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
aihelper config set worktree.base_location /custom/path

# Per-command
aihelper worktree create feature-x -l /custom/path
```

## Development

### Project Structure

```
ai-helper/
├── cmd/aihelper/             # Application entry point
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
go build -o aihelper ./cmd/aihelper

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o aihelper-linux-amd64 ./cmd/aihelper
GOOS=darwin GOARCH=arm64 go build -o aihelper-darwin-arm64 ./cmd/aihelper
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

If `aihelper` can't find the Claude CLI:

1. Ensure Claude is installed and in your PATH:
   ```bash
   which claude
   ```

2. Specify the path in config:
   ```bash
   aihelper config set claude.cli_path /path/to/claude
   ```

### Worktree Already Exists

If you get an error that a worktree already exists:

```bash
# List existing worktrees
aihelper worktree list

# Remove the existing worktree
aihelper worktree remove <name>

# Or switch to it
aihelper worktree switch <name>
```

### Permission Denied

If you get permission errors when creating worktrees:

1. Check the base location has write permissions:
   ```bash
   ls -la ../.worktrees
   ```

2. Use a different location:
   ```bash
   aihelper worktree create <name> -l ~/worktrees
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

```
docker run --rm -it \
       --network host \
       -v $(pwd):/workspace:rw \
       -v ~/.aihelper-config:/home/developer:rw \
       -v ~/.ssh:/home/developer/.ssh:ro \
       -v ~/.gitconfig:/home/developer/.gitconfig:ro \
       -w /workspace \
       aihelper-sandbox:latest \
       claude --dangerously-skip-permissions
```

```
docker run --rm -it \
       --network host \
       -v $(pwd):/workspace:rw \
       -v ~/.aihelper-config:/home/developer:rw \
       -v ~/.ssh:/home/developer/.ssh:ro \
       -v ~/.gitconfig:/home/developer/.gitconfig:ro \
       -w /workspace \
       aihelper-sandbox:latest \
       copilot --allow-all-tools --allow-all-paths
```

```
docker run --rm -it \
       --network host \
       -v $(pwd):/workspace:rw \
       -v ~/.aihelper-config:/home/developer:rw \
       -v ~/.ssh:/home/developer/.ssh:ro \
       -v ~/.gitconfig:/home/developer/.gitconfig:ro \
       -w /workspace \
       aihelper-sandbox:latest \
       gemini --yolo
```
