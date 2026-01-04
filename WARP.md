# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Commands

### Prerequisites
- Go 1.24+ (see `go.mod`).
- Git 2.5+ (for `git worktree` support).
- Optional CLIs for tool integration: Claude Code CLI (`claude`), Droid (`droid`), Gemini (`gemini`), OpenCode (`opencode`).

### Build and run
- Build the main binary (named `aihelper` in this repo):
  - `make build` – builds `./aihelper` from `./cmd/aihelper`.
  - `go build -o aihelper ./cmd/aihelper`
- Run without installing:
  - `make dev` – `go run ./cmd/aihelper`.
  - `./aihelper --help` – top-level CLI help after `make build`.

### Install / uninstall
- Install to system PATH (typically `/usr/local/bin`, requires sudo):
  - `make install`
- Install for the current user only (to `~/bin`):
  - `make install-user`
- Uninstall:
  - `make uninstall` – removes from `${PREFIX}/bin` (default `/usr/local/bin`).
  - `make uninstall-user` – removes `~/bin/aihelper`.

### Tests
- Run the full test suite:
  - `make test` – runs `go test -v ./...`.
  - `go test ./...`
- Run tests with coverage:
  - `make test-coverage` – writes `coverage.out` and `coverage.html`.
- Run a single package or test (standard Go usage):
  - `go test ./internal/worktree -run TestName` – run tests matching `TestName` in `internal/worktree`.

### Formatting, linting, and checks
- Format code:
  - `make fmt` – runs `go fmt ./...`.
- Static analysis:
  - `make vet` – runs `go vet ./...`.
- Lint (if `golangci-lint` is installed):
  - `make lint` – runs `golangci-lint run ./...`.
- Combined check (recommended before releases/PRs):
  - `make check` – runs `fmt`, `vet`, and `test`.

### Dependencies and releases
- Module dependencies:
  - `make deps` – `go mod download` and `go mod tidy`.
- Cross-platform release binaries:
  - `make release` – builds `aihelper` for multiple OS/architectures into `dist/`.
- Makefile help:
  - `make help` – lists all available targets.

Note: Documentation and examples in `README.md`, `INSTALL.md`, `QUICKREF.md`, and `UPDATING.md` may still refer to the CLI as `ctl`, but the current binary and root command name in this codebase is `aihelper`.

## High-level architecture

This is a Go CLI tool that manages Git worktrees and launches AI/dev CLIs (Claude, Droid, Gemini, OpenCode) in the context of those worktrees.

### Entry point and CLI command tree
- `cmd/aihelper/main.go`
  - Thin entry that calls `internal/cli.Execute()`.
- `internal/cli/root.go`
  - Defines the root Cobra command `aihelper`.
  - Global flags:
    - `--config` – override config file path (defaults to `$HOME/.config/aihelper/config.yaml`).
    - `--verbose` / `-v` – increase verbosity and propagate to the config manager.
    - `--dry-run` – show actions without executing them.
    - `--no-color` – disable colored output.
  - Registers command groups:
    - `worktree` – main worktree management commands.
    - `config` – configuration inspection and mutation.
  - Also exposes shortcut commands at the root level that delegate to worktree commands:
    - `c` / `new` – create worktree.
    - `sw` / `go` – switch to worktree.
    - `ls` – list worktrees.
    - `r` / `rm` / `del` – remove worktrees.

### Worktree domain (Git + business logic)
- `internal/worktree/`
  - `manager.go`
    - `Manager` orchestrates all worktree operations and sits between CLI commands and Git:
      - `Create(opts CreateOptions) (string, error)`
        - Validates worktree and branch names.
        - Derives the source branch (current branch by default, with fallback to config `worktree.default_source_branch`).
        - Resolves the base location (CLI flag `--location` → config `worktree.base_location`, default `"../.worktrees"`).
        - Computes a full worktree path including repo name: `<repo-parent>/<base_location>/<repo-name>/<worktree-name>`.
        - Ensures parent directories exist and that the target path does not already exist.
        - Creates a new branch (unless `--existing-branch` was used and the branch already exists) and then a Git worktree via `internal/git.Client`.
      - `List() ([]WorktreeInfo, error)`
        - Wraps `git worktree list --porcelain` via `internal/git` and exposes a simplified `WorktreeInfo` struct with name, path, branch, and commit.
      - `Remove(opts RemoveOptions) error`
        - Locates worktrees by name, removes them via `git worktree remove`, and optionally deletes the associated branch.
        - Honors `worktree.auto_cleanup` config when deciding whether to delete the branch by default.
      - `GetPath(name string)`
        - Convenience for finding the filesystem path of a named worktree.
    - Encapsulates the policy that worktrees live outside the main repo under a shared `.worktrees` directory organized by repository.
  - `validator.go`
    - Centralized validation for:
      - Worktree names – restricted to alphanumeric, `-`, `_`, with some reserved names rejected.
      - Branch names – simplified Git-like validation with checks against unsafe patterns (`..`, leading/trailing `/`, `//`).
      - Paths – resolves to absolute paths and rejects anything containing `..` to avoid traversal.
    - `SanitizeBranchName` converts a worktree name into a conventional branch name (lowercase, `_` → `-`).

### Git abstraction
- `internal/git/client.go`
  - Wraps the Git CLI using `os/exec` instead of using a Go Git library.
  - Responsibilities:
    - Discover and cache the repository root (`git rev-parse --show-toplevel`); errors with `pkg/errors.ErrNotInGitRepo` if not inside a repository.
    - Query the current branch.
    - Check if branches exist (`git show-ref --verify --quiet refs/heads/<branch>`).
    - Create branches and worktrees, handling “already exists” cases by returning typed errors (`ErrBranchExists`, `ErrWorktreeExists`).
    - List worktrees via porcelain output and parse them into structured `WorktreeInfo` values.
    - Remove worktrees (optionally forced) and delete branches, mapping common failure modes to typed errors.
  - This package is the single place that knows about raw Git commands; higher layers operate on the abstractions it exposes.

### Configuration system
- `internal/config/`
  - `schema.go`
    - Defines the typed configuration model:
      - `WorktreeConfig` – `base_location`, `auto_cleanup`, `default_source_branch`.
      - `ClaudeConfig` – `default_mode` (`agent`/`chat`), `auto_launch`, `extra_args`, `cli_path`.
      - `GlobalConfig` – `verbosity`, `color`, `editor`, `default_cli` (which AI CLI to launch: `claude`, `gemini`, `copilot`, `droid`, `opencode`).
    - `DefaultConfig()` centralizes all default values used across the app.
  - `config.go`
    - `Manager` wraps a dedicated `viper.Viper` instance and provides typed getters and mutation.
    - Load precedence: defaults → global config (`$HOME/.config/aihelper/config.yaml`) → repo config (`.aihelper.yaml` in repo root) → in-memory overrides (e.g., verbose/no-color flags via `util.GlobalContext`).
    - Helper functions return canonical global and repo config paths.
  - `loader.go`
    - File-level helpers for reading/writing YAML configs (global and repo) using the typed `Config` struct.
    - `InitGlobalConfig` can bootstrap a default config file at the global path.

- `internal/util/context.go`
  - `CLIContext` (exposed as singleton `GlobalContext`) tracks per-invocation state:
    - Holds a lazily-initialized `*config.Manager`.
    - Remembers `Verbose`, `DryRun`, and `NoColor` flags.
    - Applies CLI flag overrides to the underlying config (e.g., forcing `global.verbosity` or `global.color`).
  - CLI code should obtain config via `GlobalContext.GetConfigManager()` when running inside a repository.

### Tool launchers (AI and editor CLIs)
- `internal/launcher/`
  - `launcher.go`
    - Defines `ToolLauncher` interface and `LaunchOptions` (workdir, args, mode, env, interactivity, terminal name).
    - All concrete launchers implement this interface, which keeps CLI code decoupled from specific tools.
  - `claude.go`, `droid.go`, `gemini.go`, `opencode.go`
    - Concrete implementations for each supported external CLI.
    - Responsibilities:
      - Discover the executable: prefer `exec.LookPath`, then fall back to common absolute locations.
      - Respect `LaunchOptions` (working directory, args, terminal title, interactivity).
      - Wire up stdin/stdout/stderr for interactive use.
      - Set the terminal title on supported terminals via `SetTerminalTitle`.
      - Handle OS signals (`SIGINT`, `SIGTERM`) and propagate them to the child process; also handle context cancellation.
    - Each launcher returns typed `ToolError` values (from `pkg/errors`) when the backing tool is not available.
  - `shell.go`
    - Utility helpers for checking TTY support, discovering the user’s shell, and setting the terminal title.

- Integration with CLI commands:
  - `internal/cli/worktree/create.go`
    - After creating a worktree, conditionally launches a tool based on flags and config:
      - Default path: launch Claude Code CLI if `claude.auto_launch` is true and `--no-claude`, `--opencode`, `--gemini`, and `--droid` are not set.
      - Alternative tools: `--opencode`, `--gemini`, or `--droid` choose OpenCode / Gemini / Droid instead of Claude.
      - Respects `--claude-mode`, `--claude-args`, and config defaults when building `LaunchOptions`.
      - Honors `--terminal-name` for naming the terminal window; falls back to the worktree name.
  - `internal/cli/worktree/switch.go`
    - Mirrors the same tool-selection logic when switching to an existing worktree, including alternative tool flags and terminal naming.

### CLI command packages
- `internal/cli/worktree/`
  - Provides the `worktree` command group and its subcommands:
    - `create` – provisions a branch and worktree, then launches a tool.
    - `list` – prints a table of worktrees (name, branch, short commit hash, path) with optional colorized output.
    - `remove` – removes one or more worktrees and optionally their branches; supports `--delete-branch` and `--force`.
    - `switch` – locates a named worktree and launches the chosen tool in its directory.
  - This package is the main consumer of `internal/worktree`, `internal/git`, `internal/config`, `internal/launcher`, and `internal/util`.

- `internal/cli/config/`
  - `config` command group for viewing and mutating configuration without touching YAML files directly:
    - `list` – dumps the effective configuration in YAML and shows the global config path.
    - `get` – prints a single configuration value, trying string, bool, and int forms.
    - `set` – updates specific keys in the global config, with type-aware parsing and validation (e.g., boolean and integer fields, allowed Claude modes).

### Error handling
- `pkg/errors/`
  - Centralized error types for domain-specific conditions:
    - Base sentinel errors for Git, worktrees, branches, tools, invalid names/paths, and permission issues.
    - Wrapper types (`WorktreeError`, `BranchError`, `ToolError`) that attach context (e.g., which worktree, branch, or tool) while supporting `errors.Is`/`errors.Unwrap`.
  - Higher-level code (CLI, worktree manager, Git client, launchers) constructs and returns these errors so that user-facing commands can emit clear, contextual messages.

## Repository documentation

Key docs that expand on behavior and workflows:
- `README.md`
  - User-facing overview, quick install instructions (including the `install.sh` one-liner), feature list, configuration reference, and examples of `worktree` and `config` commands.
  - Describes the default worktree layout under a `.worktrees` directory organized by repository name.
- `INSTALL.md`
  - Detailed installation options across platforms (manual builds, `go install`, Windows instructions, and shell completion setup).
- `QUICKREF.md`
  - Compact command reference for both users and maintainers, including common `aihelper`/`ctl` workflows and Makefile targets.
- `UPDATING.md`
  - Update instructions for users and a release workflow for maintainers (version bumping, changelog updates, tagging, building release binaries, and optional GitHub Actions automation).

When in doubt about expected end-user behavior or release process, prefer these documents as the source of truth and use this WARP.md for commands and architecture context specific to working inside the codebase.