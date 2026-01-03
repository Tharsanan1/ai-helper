# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of ctl - Git Worktree Manager with Claude Code Integration

## [0.1.0] - 2024-01-03

### Added
- `ctl worktree create` - Create git worktrees with Claude integration
- `ctl worktree list` - List all worktrees
- `ctl worktree remove` - Remove worktrees
- `ctl worktree switch` - Switch to existing worktree and launch Claude
- `ctl config set/get/list` - Configuration management
- Layered configuration system (global + repo-level)
- Automatic Claude Code CLI integration
- Interactive installation script
- Makefile for easy building and installation
- Comprehensive documentation (README, INSTALL, UPDATING)

### Features
- Noun-verb command structure (`ctl worktree create`)
- Smart branch handling (uses current branch by default)
- Configurable worktree locations
- Verbose and dry-run modes
- Colored output with `--no-color` option
- Input validation and error handling

### Documentation
- Complete README with quick start guide
- Detailed INSTALL.md with platform-specific instructions
- UPDATING.md for maintainers and users
- Installation script with interactive prompts

[Unreleased]: https://github.com/tharsanan1/ai-helper/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/tharsanan1/ai-helper/releases/tag/v0.1.0
