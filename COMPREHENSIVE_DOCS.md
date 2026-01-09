# AI Helper - Comprehensive User Guide

## Overview

AI Helper (`aihelper`) is a command-line tool for managing Git worktrees with seamless integration for launching AI development tools like Claude Code CLI, Gemini, Copilot, Droid, and OpenCode. It simplifies the workflow of creating isolated worktrees for feature development and automatically launches AI tools in the worktree context.

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Command Reference](#command-reference)
4. [Configuration](#configuration)
5. [Advanced Features](#advanced-features)
6. [Examples](#examples)
7. [Troubleshooting](#troubleshooting)

---

## Installation

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash
```

### From Source

```bash
git clone https://github.com/tharsanan1/ai-helper.git
cd ai-helper
go build -o aihelper ./cmd/aihelper
sudo mv aihelper /usr/local/bin/
```

### Requirements

- **Go 1.24+** - for building from source
- **Git 2.5+** - for worktree support
- **Docker** (optional) - for sandbox feature
- **AI CLIs** (optional) - Claude Code CLI, Gemini CLI, Copilot CLI, Droid, or OpenCode

---

## Shell Completion

AI Helper provides rich shell completion for Bash, Zsh, Fish, and PowerShell. This feature is highly recommended as it improves productivity by:

- Automatically completing commands and flags
- Providing descriptions for flags (Zsh)
- Dynamically completing values (e.g., Claude modes)

### Zsh Setup (Recommended)

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

For other shells, see the [Installation Guide](INSTALL.md#3-shell-completion-recommended).

---

## Quick Start

### Create a Worktree

```bash
# Create and launch Claude
aihelper worktree create feature-auth

# Create with custom branch name
aihelper worktree create feature-auth -b auth/login

# Create without launching AI tool
aihelper worktree create experiment --no-claude
```

### List Worktrees

```bash
aihelper worktree list
```

### Switch to Worktree

```bash
aihelper worktree switch feature-auth
```

### Remove Worktree

```bash
# Remove worktree only
aihelper worktree remove feature-auth

# Remove and delete branch
aihelper worktree remove feature-auth --delete-branch
```

---

## Command Reference

## Global Flags

These flags work with all commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | | Specify config file path (default: `~/.config/aihelper/config.yaml`) |
| `--verbose` | `-v` | Enable verbose output |
| `--dry-run` | | Show what would happen without executing |
| `--no-color` | | Disable colored output |
| `--version` | | Display version information |

---

## Worktree Commands

### `worktree create <name>`

Create a new git worktree and optionally launch an AI tool.

**Aliases:** `c`, `new`

**Flags:**

| Flag | Description | Example |
|------|-------------|---------|
| `-b, --branch <name>` | Branch name (default: same as worktree name) | `-b auth/login` |
| `-l, --location <path>` | Base directory for worktrees | `-l ~/worktrees` |
| `-f, --from <branch>` | Source branch (default: current branch) | `-f main` |
| `-e, --existing-branch` | Use existing branch instead of creating new | `--existing-branch` |
| `--no-claude` | Create worktree without launching AI tool | `--no-claude` |
| `--claude` | Explicitly launch Claude | `--claude` |
| `--claude-mode <mode>` | Claude mode: `chat` or `agent` | `--claude-mode chat` |
| `--claude-args <args>` | Additional Claude CLI arguments | `--claude-args --dangerously-skip-permissions` |
| `--gemini` | Launch Gemini CLI instead of Claude | `--gemini` |
| `--copilot` | Launch Copilot CLI instead of Claude | `--copilot` |
| `--droid` | Launch Droid CLI instead of Claude | `--droid` |
| `--opencode` | Launch OpenCode instead of Claude | `--opencode` |
| `--minimax` | Launch Claude with Minimax APIs | `--minimax` |
| `--system-prompt <text>` | System prompt to use | `--system-prompt "You are an expert developer"` |
| `--append-system-prompt` | Append system prompt instead of replacing | `--append-system-prompt` |
| `--terminal-name <name>` | Terminal window name | `--terminal-name "Feature Auth"` |
| `--new-terminal` | Launch in a new terminal window | `--new-terminal` |
| `--sandbox` | Launch in Docker sandbox | `--sandbox` |

**Examples:**

```bash
# Basic usage
aihelper worktree create feature-payment

# Create with custom branch
aihelper worktree create feature-payment -b payment/stripe

# Create from main branch
aihelper worktree create bugfix-123 -f main

# Create and launch in new terminal
aihelper worktree create feature-x --new-terminal

# Create with Gemini
aihelper worktree create feature-x --gemini

# Create with custom system prompt
aihelper worktree create feature-x --system-prompt "You are a code reviewer"

# Create with Minimax APIs
aihelper worktree create feature-x --minimax

# Create in sandbox
aihelper worktree create feature-x --sandbox
```

---

### `worktree switch <name>`

Switch to an existing worktree and launch an AI tool.

**Aliases:** `sw`, `go`

**Flags:**

| Flag | Description | Example |
|------|-------------|---------|
| `--claude` | Explicitly launch Claude | `--claude` |
| `--claude-mode <mode>` | Claude mode: `chat` or `agent` | `--claude-mode chat` |
| `--claude-args <args>` | Additional Claude CLI arguments | `--claude-args --dangerously-skip-permissions` |
| `--gemini` | Launch Gemini CLI instead of Claude | `--gemini` |
| `--copilot` | Launch Copilot CLI instead of Claude | `--copilot` |
| `--droid` | Launch Droid CLI instead of Claude | `--droid` |
| `--opencode` | Launch OpenCode instead of Claude | `--opencode` |
| `--minimax` | Launch Claude with Minimax APIs | `--minimax` |
| `--system-prompt <text>` | System prompt to use | `--system-prompt "You are an expert developer"` |
| `--append-system-prompt` | Append system prompt instead of replacing | `--append-system-prompt` |
| `--terminal-name <name>` | Terminal window name | `--terminal-name "Feature Auth"` |
| `--sandbox` | Launch in Docker sandbox | `--sandbox` |

**Examples:**

```bash
# Switch to worktree
aihelper worktree switch feature-auth

# Switch with custom mode
aihelper worktree switch feature-auth --claude-mode chat

# Switch with Gemini
aihelper worktree switch feature-auth --gemini

# Switch with custom system prompt
aihelper worktree switch feature-auth --system-prompt "You are a code reviewer"

# Switch with Minimax APIs
aihelper worktree switch feature-auth --minimax

# Switch in sandbox
aihelper worktree switch feature-auth --sandbox
```

---

### `worktree list`

List all worktrees with their branches, paths, and commits.

**Aliases:** `ls`

**Examples:**

```bash
aihelper worktree list
```

**Output Format:**
```
NAME                 BRANCH                         COMMIT     PATH
--------------------------------------------------------------------------------
feature-auth         auth/user-login                a1b2c3d    /Users/you/Documents/.worktrees/ai-helper/feature-auth
experiment           experiment                     e4f5g6h    /Users/you/Documents/.worktrees/ai-helper/experiment
```

---

### `worktree remove <name> [...]`

Remove one or more worktrees and optionally delete their branches.

**Aliases:** `rm`, `delete`

**Flags:**

| Flag | Description |
|------|-------------|
| `-d, --delete-branch` | Also delete the associated git branch |
| `--force` | Force remove even with modified or untracked files |

**Examples:**

```bash
# Remove single worktree
aihelper worktree remove feature-auth

# Remove multiple worktrees
aihelper worktree remove feature-auth feature-billing

# Remove and delete branches
aihelper worktree remove feature-auth --delete-branch

# Force remove (with untracked files)
aihelper worktree remove feature-auth --force
```

---

## Config Commands

### `config list`

List all configuration values.

**Aliases:** `ls`, `show`

**Examples:**

```bash
aihelper config list
```

---

### `config get <key>`

Get a specific configuration value.

**Examples:**

```bash
# Get worktree base location
aihelper config get worktree.base_location

# Get auto launch setting
aihelper config get claude.auto_launch

# Get verbosity level
aihelper config get global.verbosity

# Get default CLI
aihelper config get global.default_cli

# Get Minimax API key
aihelper config get claude.minimax_api_key

# Get system prompt
aihelper config get claude.system_prompt
```

---

### `config set <key> <value>`

Set a configuration value in the global config.

**Flags:**

| Flag | Description |
|------|-------------|
| `-g, --global` | Set in global config (default) |

**Examples:**

```bash
# Set worktree base location
aihelper config set worktree.base_location ~/worktrees

# Disable auto launch
aihelper config set claude.auto_launch false

# Set verbosity level
aihelper config set global.verbosity 2

# Set default CLI to Gemini
aihelper config set global.default_cli gemini

# Set Minimax API key
aihelper config set claude.minimax_api_key your-api-key-here

# Set system prompt
aihelper config set claude.system_prompt "You are an expert developer"

# Enable system prompt
aihelper config set claude.use_system_prompt true

# Set system prompt mode (replace or append)
aihelper config set claude.system_prompt_mode append

# Enable Minimax verbose mode
aihelper config set claude.minimax_verbose true

# Set default source branch
aihelper config set worktree.default_source_branch main

# Enable auto cleanup
aihelper config set worktree.auto_cleanup true
```

---

## Configuration

AI Helper uses a layered configuration system with the following precedence:

**Priority Order:** Flags > Repository Config > Global Config > Defaults

### Configuration Files

1. **Global config:** `~/.config/aihelper/config.yaml`
2. **Repository config:** `.aihelper.yaml` (in repository root)
3. **Command-line flags:** Override all config settings

### Configuration Schema

#### Worktree Settings (`worktree`)

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `base_location` | string | `"../.worktrees"` | Base directory for worktrees (relative to repo parent or absolute) |
| `auto_cleanup` | bool | `true` | Automatically delete branch when removing worktree |
| `default_source_branch` | string | `""` | Default source branch for new worktrees (empty = current branch) |

#### Claude Settings (`claude`)

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `default_mode` | string | `"agent"` | Default mode: `agent` or `chat` |
| `auto_launch` | bool | `true` | Automatically launch Claude after creating worktree |
| `extra_args` | []string | `[]` | Additional arguments to pass to Claude CLI |
| `cli_path` | string | `""` | Path to Claude CLI (auto-detected if empty) |
| `minimax_api_key` | string | `""` | API key for Minimax APIs |
| `system_prompt` | string | `""` | System prompt to use when launching Claude |
| `system_prompt_mode` | string | `"replace"` | How to apply system prompt: `replace` or `append` |
| `use_system_prompt` | bool | `false` | Enable/disable system prompt feature |
| `minimax_verbose` | bool | `false` | Enable verbose mode for Minimax APIs |

#### Global Settings (`global`)

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `verbosity` | int | `1` | Verbosity level: 0=quiet, 1=normal, 2=verbose |
| `color` | bool | `true` | Enable colored output |
| `editor` | string | `""` | Editor for opening files (falls back to $EDITOR) |
| `default_cli` | string | `"claude"` | Default CLI to launch: `claude`, `gemini`, `copilot`, `droid`, or `opencode` |

### Example Configuration

```yaml
worktree:
  base_location: "../.worktrees"
  auto_cleanup: true
  default_source_branch: "main"

claude:
  default_mode: "agent"
  auto_launch: true
  extra_args: []
  cli_path: ""
  minimax_api_key: "your-api-key-here"
  system_prompt: "You are an expert software engineer"
  system_prompt_mode: "replace"
  use_system_prompt: false
  minimax_verbose: false

global:
  verbosity: 1
  color: true
  editor: ""
  default_cli: "claude"
```

---

## Advanced Features

### System Prompts

You can set a custom system prompt to guide the AI tool's behavior:

**Via Command Line:**
```bash
# Replace default system prompt
aihelper worktree create feature-x --system-prompt "You are a senior backend engineer"

# Append to default system prompt
aihelper worktree create feature-x --system-prompt "Focus on security best practices" --append-system-prompt
```

**Via Configuration:**
```bash
# Set default system prompt
aihelper config set claude.system_prompt "You are a code reviewer"

# Enable using system prompt
aihelper config set claude.use_system_prompt true

# Set mode to append instead of replace
aihelper config set claude.system_prompt_mode append
```

### Minimax API Integration

Use Minimax APIs for enhanced AI capabilities:

1. **Get API Key:** Obtain from Minimax platform
2. **Configure:**
   ```bash
   aihelper config set claude.minimax_api_key your-api-key-here
   ```
3. **Use:**
   ```bash
   aihelper worktree create feature-x --minimax
   aihelper worktree switch feature-x --minimax
   ```
4. **Enable verbose mode:**
   ```bash
   aihelper config set claude.minimax_verbose true
   ```

### Docker Sandbox

Run AI tools in isolated Docker containers:

**Prerequisites:**
- Docker installed and running
- `aihelper-sandbox:latest` image built (see Dockerfile in repo)

**Usage:**
```bash
# Create worktree in sandbox
aihelper worktree create feature-x --sandbox

# Switch to worktree in sandbox
aihelper worktree switch feature-x --sandbox
```

**Sandbox Features:**
- Isolated environment
- Mounted SSH keys (read-only)
- Mounted git config
- Network access enabled
- Working directory: `/workspace`

### Multiple AI Tool Support

AI Helper supports launching different AI tools:

**Supported Tools:**
- **Claude** (default) - Anthropic's Claude Code CLI
- **Gemini** - Google's Gemini CLI
- **Copilot** - GitHub Copilot CLI
- **Droid** - Droid CLI
- **OpenCode** - OpenCode CLI

**Selection Methods:**

1. **Command-line flags** (override config):
   ```bash
   aihelper worktree create feature-x --gemini
   aihelper worktree create feature-x --copilot
   aihelper worktree create feature-x --opencode
   aihelper worktree create feature-x --droid
   ```

2. **Configuration default** (applies to all commands):
   ```bash
   aihelper config set global.default_cli gemini
   aihelper config set global.default_cli copilot
   aihelper config set global.default_cli opencode
   ```

### New Terminal Window

Launch AI tools in a new terminal window:

```bash
# Create worktree and open in new terminal
aihelper worktree create feature-x --new-terminal

# Custom terminal name
aihelper worktree create feature-x --new-terminal --terminal-name "Feature X Development"
```

### Worktree Location Strategy

**Default Location:**
```
../.worktrees/<repo-name>/<worktree-name>
```

**Example:**
- Repository: `/Users/you/projects/my-app`
- Worktree: `/Users/you/projects/.worktrees/my-app/feature-auth`

**Custom Locations:**

1. **Global config:**
   ```bash
   aihelper config set worktree.base_location ~/worktrees
   ```

2. **Per-command:**
   ```bash
   aihelper worktree create feature-x -l ~/custom/worktrees
   ```

3. **Absolute path:**
   ```bash
   aihelper worktree create feature-x -l /tmp/worktrees
   ```

---

## Examples

### Complete Workflow

```bash
# 1. Create a feature worktree with Claude
aihelper worktree create feature-authentication

# 2. Develop your feature...
# (worktree is at ../.worktrees/your-repo/feature-authentication)

# 3. Switch back to main to create another worktree
cd /path/to/main/repo
aihelper worktree create feature-billing

# 4. List all worktrees
aihelper worktree list

# 5. Switch to any worktree
aihelper worktree switch feature-authentication

# 6. Remove completed worktrees
aihelper worktree remove feature-authentication --delete-branch
```

### Using Different AI Tools

```bash
# Create with Claude (default)
aihelper worktree create feature-x

# Create with Gemini
aihelper worktree create feature-x --gemini

# Create with Copilot
aihelper worktree create feature-x --copilot

# Create with OpenCode
aihelper worktree create feature-x --opencode

# Create with Droid
aihelper worktree create feature-x --droid

# Set default CLI globally
aihelper config set global.default_cli gemini
# Now all future commands use Gemini by default
```

### Advanced Examples

```bash
# Create from specific branch with custom system prompt
aihelper worktree create hotfix-security -f main \
  --system-prompt "You are a security expert" \
  --terminal-name "Security Hotfix"

# Create with extra Claude arguments
aihelper worktree create feature-x \
  --claude-args "--dangerously-skip-permissions" \
  --claude-mode chat

# Create multiple worktrees at once
aihelper worktree create feature-ui
aihelper worktree create feature-api
aihelper worktree create feature-db

# Use with Minimax APIs
aihelper config set claude.minimax_api_key your-key
aihelper worktree create feature-x --minimax --minimax-verbose

# Create in sandbox environment
aihelper worktree create experimental-feature --sandbox --no-claude
```

---

## Bash Aliases

Add these to your `~/.bashrc` or `~/.zshrc` for faster workflow:

```bash
# Worktree shortcuts
alias wc='aihelper c'
alias ws='aihelper sw'
alias wl='aihelper ls'
alias wr='aihelper r'

# Config shortcuts
alias cfg='aihelper config'
alias cfgl='aihelper config list'
alias cfgs='aihelper config set'
alias cfgg='aihelper config get'

# AI tool shortcuts
alias wcc='aihelper c --claude'
alias wcg='aihelper c --gemini'
alias wcp='aihelper c --copilot'
alias wco='aihelper c --opencode'
alias wcd='aihelper c --droid'
alias wcn='aihelper c --no-claude'

# Switch shortcuts
alias wsg='aihelper sw --gemini'
alias wsp='aihelper sw --copilot'
alias wso='aihelper sw --opencode'
```

---

## Troubleshooting

### Common Issues

#### 1. Claude CLI Not Found

**Error:** `Failed to find Claude CLI`

**Solutions:**
```bash
# Verify Claude is installed
which claude

# Specify path in config
aihelper config set claude.cli_path /path/to/claude

# Check verbose output
aihelper -v worktree create feature-x
```

#### 2. Worktree Already Exists

**Error:** `worktree already exists`

**Solutions:**
```bash
# List existing worktrees
aihelper worktree list

# Switch to existing worktree
aihelper worktree switch feature-x

# Remove existing worktree
aihelper worktree remove feature-x
```

#### 3. Permission Denied

**Error:** `permission denied` when creating worktrees

**Solutions:**
```bash
# Check base location permissions
ls -la ~/.config/aihelper/

# Use different location
aihelper worktree create feature-x -l ~/worktrees

# Set in config
aihelper config set worktree.base_location ~/worktrees
```

#### 4. Not in Git Repository

**Error:** `not in a git repository`

**Solutions:**
```bash
# Navigate to git repository
cd /path/to/your/repo

# Verify you're in a git repo
git status
```

#### 5. Docker Sandbox Issues

**Error:** `command not found: docker` or sandbox errors

**Solutions:**
```bash
# Install Docker
# See https://docs.docker.com/get-docker/

# Build sandbox image
cd /path/to/ai-helper
docker build -t aihelper-sandbox:latest ./docker

# Verify Docker is running
docker ps
```

#### 6. Minimax API Issues

**Error:** `Minimax API key not configured` or API errors

**Solutions:**
```bash
# Set API key
aihelper config set claude.minimax_api_key your-key-here

# Enable verbose mode for debugging
aihelper config set claude.minimax_verbose true

# Verify key is set
aihelper config get claude.minimax_api_key
```

#### 7. Branch Already Exists

**Error:** `branch already exists`

**Solutions:**
```bash
# Use existing branch flag
aihelper worktree create feature-x --existing-branch

# Or use different branch name
aihelper worktree create feature-x -b feature-x-new
```

### Debug Mode

Enable verbose output for detailed information:

```bash
# Global verbose
aihelper -v worktree create feature-x

# Or set verbosity in config
aihelper config set global.verbosity 2
```

### Dry Run Mode

Preview actions without executing:

```bash
# Dry run create
aihelper --dry-run worktree create feature-x

# Dry run remove
aihelper --dry-run worktree remove feature-x
```

### Check Configuration

```bash
# View all config
aihelper config list

# Get specific value
aihelper config get worktree.base_location
aihelper config get claude.auto_launch
aihelper config get global.default_cli
```

### Rebuild and Test

```bash
# Clean build
make clean && make build

# Test installation
make install-user
aihelper --version
```

---

## Shortcut Commands

AI Helper provides single-letter shortcuts for common commands:

| Command | Aliases | Description |
|---------|---------|-------------|
| `aihelper c <name>` | `aihelper new` | Create worktree |
| `aihelper sw <name>` | `aihelper go` | Switch to worktree |
| `aihelper ls` | | List worktrees |
| `aihelper r <name>` | `aihelper rm`, `aihelper delete` | Remove worktree |

**Usage:**
```bash
aihelper c feature-x      # Create
aihelper sw feature-x     # Switch
aihelper ls              # List
aihelper r feature-x     # Remove
```

---

## Best Practices

### 1. Use Descriptive Names
```bash
# Good
aihelper worktree create feature-user-authentication
aihelper worktree create bugfix-memory-leak
aihelper worktree create hotfix-security-patch

# Less descriptive
aihelper worktree create feature1
aihelper worktree create bug1
```

### 2. Set Defaults via Configuration
```bash
# Set your preferred defaults
aihelper config set global.default_cli gemini
aihelper config set worktree.default_source_branch main
aihelper config set claude.auto_launch true
```

### 3. Clean Up Regularly
```bash
# Remove completed worktrees
aihelper worktree remove completed-feature --delete-branch

# List before cleanup
aihelper worktree list
```

### 4. Use Terminal Names for Organization
```bash
# Custom terminal names help when you have multiple terminals open
aihelper worktree create feature-x --terminal-name "Feature X Dev"
aihelper worktree create feature-y --terminal-name "Feature Y Dev"
```

### 5. Version Control Configuration
```bash
# Global settings in ~/.config/aihelper/config.yaml
# Repo-specific settings in .aihelper.yaml

# Commit .aihelper.yaml to share team defaults
echo "worktree:
  default_source_branch: main
  auto_cleanup: true
claude:
  auto_launch: true
  default_mode: agent" > .aihelper.yaml
git add .aihelper.yaml
git commit -m "Add AI Helper config defaults"
```

---

## Integration Examples

### VS Code Integration

Open worktrees directly in VS Code:

```bash
# After creating/switching, open in VS Code
aihelper worktree switch feature-x
code .
```

### Terminal Multiplexer (tmux)

Create named sessions for worktrees:

```bash
# Create worktree and attach to tmux session
aihelper worktree create feature-x --new-terminal
tmux new -s feature-x

# Switch and attach
aihelper worktree switch feature-x
tmux attach -t feature-x
```

---

## API Reference

### Configuration Keys

#### Worktree Configuration Keys

- `worktree.base_location` - Base directory for worktrees
- `worktree.auto_cleanup` - Auto cleanup branches
- `worktree.default_source_branch` - Default source branch

#### Claude Configuration Keys

- `claude.default_mode` - Default Claude mode (agent/chat)
- `claude.auto_launch` - Auto launch Claude
- `claude.extra_args` - Extra CLI arguments
- `claude.cli_path` - Claude CLI path
- `claude.minimax_api_key` - Minimax API key
- `claude.system_prompt` - System prompt text
- `claude.system_prompt_mode` - System prompt mode (replace/append)
- `claude.use_system_prompt` - Enable system prompt
- `claude.minimax_verbose` - Minimax verbose mode

#### Global Configuration Keys

- `global.verbosity` - Verbosity level (0-2)
- `global.color` - Enable colors
- `global.editor` - Default editor
- `global.default_cli` - Default CLI tool

---

## Support and Resources

- **GitHub Repository:** https://github.com/tharsanan1/ai-helper
- **Issues:** https://github.com/tharsanan1/ai-helper/issues
- **Discussions:** https://github.com/tharsanan1/ai-helper/discussions

---

## License

MIT License - see LICENSE file for details

---

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) - CLI framework
- Configuration powered by [Viper](https://github.com/spf13/viper) - Configuration management
- Inspired by modern development workflows and the need for better worktree management
