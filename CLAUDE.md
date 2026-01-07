# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ai-helper** (also known as **ctl**) is a Go-based command-line tool for managing Git worktrees with seamless integration for launching development tools like Claude Code CLI. It simplifies creating isolated worktrees for feature development and automatically launches AI coding assistants in the worktree context.

### Key Features
- Git worktree management (create, list, remove, switch)
- Multi-AI tool integration (Claude, Gemini, Copilot, Droid, OpenCode)
- Docker sandbox support for safe AI execution
- Minimax API integration for custom system prompts
- Layered configuration system (global, repository, CLI flags)
- Support for launching tools in new terminal windows

## Technology Stack

- **Language**: Go 1.24+
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra) v1.10.2
- **Configuration**: [Viper](https://github.com/spf13/viper) v1.21.0
- **Git Operations**: Direct git CLI commands via os/exec
- **Docker**: Containerized sandbox environment

## Architecture

The codebase follows a clean layered architecture:

### Core Layers

1. **CLI Layer** (`internal/cli/`)
   - Commands are organized into subdirectories (`config/`, `worktree/`)
   - Uses Cobra's command structure with persistent flags
   - Shortcut commands (c, sw, ls, r) for common operations

2. **Configuration Layer** (`internal/config/`)
   - Schema defines configuration structure (`config/schema.go`)
   - Loader handles file I/O and environment variables
   - Default values provided for all options
   - Support for worktree, Claude, and global settings

3. **Git Operations Layer** (`internal/git/`)
   - Wraps git CLI commands using os/exec
   - Handles worktree lifecycle operations
   - Branch management and validation

4. **Launcher Layer** (`internal/launcher/`)
   - Interface-based design for tool launchers (`launcher.go`)
   - Individual implementations per tool:
     - `claude.go` - Claude Code CLI with Minimax integration
     - `gemini.go` - Google Gemini CLI
     - `copilot.go` - GitHub Copilot CLI
     - `droid.go`, `opencode.go` - Other AI tools
     - `shell.go` - Generic shell command support

5. **Business Logic Layer** (`internal/worktree/`)
   - Worktree manager handles creation, listing, removal
   - Validator ensures names and paths are valid
   - Integration with git client and config

6. **Error Handling** (`pkg/errors/`)
   - Custom error types for worktrees, branches, and tools
   - Structured error messages for better UX

### Docker Sandbox

The `docker/Dockerfile` provides a containerized environment:
- Ubuntu 24.04 base with Node.js 20.x
- Pre-installed AI CLIs (Claude, Copilot, Gemini)
- Non-root user for security
- Volume mounts for SSH, gitconfig, and aihelper config
- Used via `docker run` with pre-configured command examples in README

### Minimax Integration

Recent commits added Minimax API support:
- System prompt configuration in `ClaudeConfig`
- API key support for Minimax
- Configurable system prompt mode (replace/append)
- Verbose mode for debugging

## Development Workflow

### Common Commands

```bash
# Build the binary
make build
go build -o ctl ./cmd/aihelper

# Install system-wide (requires sudo)
make install

# Install to ~/bin (no sudo)
make install-user

# Run tests
make test
go test -v ./...

# Run with coverage
make test-coverage

# Development (build and run)
make run

# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Build release binaries for multiple platforms
make release
```

### Project Structure

```
ai-helper/
├── cmd/aihelper/              # Application entry point
├── internal/
│   ├── cli/                   # CLI commands
│   │   ├── config/           # config subcommands
│   │   └── worktree/         # worktree subcommands
│   ├── config/               # Configuration management
│   ├── git/                  # Git operations
│   ├── launcher/             # Tool launcher implementations
│   ├── util/                 # Shared utilities
│   └── worktree/             # Worktree business logic
└── pkg/errors/               # Custom error types
```

### Adding New Features

1. **New Command**: Add to `internal/cli/` with appropriate subdirectory
2. **New Tool Launcher**: Implement `ToolLauncher` interface in `internal/launcher/`
3. **Configuration**: Extend `Config` struct in `internal/config/schema.go`
4. **Git Operations**: Add methods to `internal/git/client.go`

## Configuration

### Configuration Layers (Priority: CLI flags > Repo config > Global config > Defaults)

**Global config**: `~/.config/aihelper/config.yaml`
**Repository config**: `.aihelper.yaml` (in repo root)

### Key Configuration Options

```yaml
worktree:
  base_location: "../.worktrees"      # Worktree directory
  auto_cleanup: true                   # Auto-delete branch on removal
  default_source_branch: "main"        # Default source branch

claude:
  default_mode: "agent"               # Default launch mode
  auto_launch: true                   # Auto-launch on worktree creation
  extra_args: []                      # Additional CLI args
  cli_path: ""                        # Path to Claude CLI
  minimax_api_key: ""                 # Minimax API key
  system_prompt: ""                   # Custom system prompt
  system_prompt_mode: "replace"       # How to apply prompt
  use_system_prompt: false            # Enable/disable prompt
  minimax_verbose: false              # Minimax debug mode

global:
  verbosity: 1                        # 0=quiet, 1=normal, 2=verbose
  color: true                         # Colored output
  editor: ""                          # Editor fallback to $EDITOR
  default_cli: "claude"               # Default AI tool
```

## Testing

- Uses Go's built-in testing framework
- Test files: `*_test.go` pattern
- Run all tests: `go test ./...`
- Run with coverage: `go test -cover ./...`
- Coverage report: `coverage.html` (generated by `make test-coverage`)

## Command Reference

### Primary Commands

- `aihelper worktree create <name>` - Create worktree and optionally launch AI tool
- `aihelper worktree list` - List all worktrees (alias: `ls`)
- `aihelper worktree remove <name>` - Remove worktree (alias: `rm`)
- `aihelper worktree switch <name>` - Switch to worktree and launch AI (alias: `sw`)
- `aihelper config list` - Show configuration
- `aihelper config get <key>` - Get config value
- `aihelper config set <key> <value>` - Set config value

### AI Tool Flags

- `--claude`, `--gemini`, `--copilot`, `--droid`, `--opencode` - Choose AI tool
- `--no-claude` - Don't launch any AI tool
- `--new-terminal` - Launch in new terminal window
- `--sandbox` - Run in Docker sandbox

## Build and Release

The Makefile provides comprehensive build targets:
- `make build` - Single-platform build
- `make release` - Multi-platform release builds (Linux AMD64/ARM64, macOS AMD64/ARM64, Windows AMD64)
- Release binaries output to `dist/` directory

Version information embedded via Makefile:
- `VERSION` from Makefile
- `GIT_COMMIT` from git
- `BUILD_DATE` from current time

## Installation

Three installation methods:

1. **Quick install** (from GitHub):
   ```bash
   curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash
   ```

2. **From source**:
   ```bash
   git clone https://github.com/tharsanan1/ai-helper.git
   cd ai-helper
   make install  # or make install-user
   ```

3. **Manual build**:
   ```bash
   go build -o ctl ./cmd/aihelper
   sudo mv ctl /usr/local/bin/
   ```

## Key Implementation Details

### Worktree Location Strategy

Default location: `../.worktrees/<repo-name>/<worktree-name>`

- Relative to repository parent directory
- Keeps worktrees outside main repo
- Organized by repository name
- Supports multiple projects

### Docker Sandbox Usage

Example commands from README for running in sandbox:

```bash
# Claude
docker run --rm -it \
  --network host \
  -v $(pwd):/workspace:rw \
  -v ~/.aihelper-config:/home/developer:rw \
  -v ~/.ssh:/home/developer/.ssh:ro \
  -v ~/.gitconfig:/home/developer/.gitconfig:ro \
  -w /workspace \
  aihelper-sandbox:latest \
  claude --dangerously-skip-permissions

# Copilot
docker run --rm -it \
  --network host \
  -v $(pwd):/workspace:rw \
  -v ~/.aihelper-config:/home/developer:rw \
  -v ~/.ssh:/home/developer/.ssh:ro \
  -v ~/.gitconfig:/home/developer/.gitconfig:ro \
  -w /workspace \
  aihelper-sandbox:latest \
  copilot --allow-all-tools --allow-all-paths

# Gemini
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

## Important Files to Know

- `cmd/aihelper/main.go` - Application entry point
- `internal/cli/root.go` - Root command and global flags
- `internal/config/schema.go` - Configuration structure and defaults
- `internal/git/client.go` - Git command wrapper
- `internal/launcher/launcher.go` - Tool launcher interface
- `internal/worktree/manager.go` - Worktree business logic
- `Makefile` - Build and development commands
- `README.md` - User documentation and examples
- `docker/Dockerfile` - Sandbox environment definition

## Dependencies

Main dependencies (from `go.mod`):
- `github.com/spf13/cobra` v1.10.2 - CLI framework
- `github.com/spf13/viper` v1.21.0 - Configuration
- `github.com/fatih/color` v1.18.0 - Colored terminal output
- `golang.org/x/term` v0.38.0 - Terminal handling
- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing

All dependencies are vendored in `go.sum`.

## Bash Aliases

The project recommends helpful aliases for faster workflows (documented in README):

```bash
# Worktree shortcuts
alias wc='aihelper c'                    # Create
alias ws='aihelper sw'                   # Switch
alias wl='aihelper ls'                   # List
alias wr='aihelper r'                    # Remove

# AI tool specific
alias wca='aihelper c'                   # Claude (default)
alias wco='aihelper c --opencode'        # OpenCode
alias wcg='aihelper c --gemini'          # Gemini
alias wcd='aihelper c --droid'           # Droid
alias wcp='aihelper c --copilot'         # Copilot
alias wcn='aihelper c --no-claude'       # No launcher
```
