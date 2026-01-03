# Quick Reference

## User Workflows

### Installation
```bash
# First time install
curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash

# Update to latest
curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash
```

### Daily Usage
```bash
# Create worktree
ctl worktree create feature-name

# List worktrees
ctl worktree list

# Switch to worktree
ctl worktree switch feature-name

# Remove worktree
ctl worktree remove feature-name -d
```

### Configuration
```bash
# View config
ctl config list

# Set config
ctl config set worktree.base_location ~/worktrees
ctl config set claude.auto_launch false

# Get config
ctl config get claude.auto_launch
```

---

## Maintainer Workflows

### Making Changes
```bash
# 1. Make changes
vim internal/whatever.go

# 2. Test
make build
./ctl --help

# 3. Commit and push
git add .
git commit -m "feat: add new feature"
git push origin main
```

### Creating Releases

#### Version Bump
```bash
# Edit version
vim internal/cli/root.go  # Change Version: "0.1.0" -> "0.2.0"

# Update changelog
vim CHANGELOG.md  # Add release notes

# Commit
git add .
git commit -m "chore: bump version to 0.2.0"
```

#### Tag and Release
```bash
# Create tag
git tag -a v0.2.0 -m "Release v0.2.0"

# Push with tags
git push origin main --tags

# Build binaries
make release

# Upload to GitHub Releases
# Go to https://github.com/tharsanan1/ai-helper/releases
# Create new release and upload files from dist/
```

### Quick Commands
```bash
# Build
make build

# Install locally
make install-user

# Clean
make clean

# Test
make test

# Check code
make check

# Build for all platforms
make release
```

---

## Common Scenarios

### User: "How do I update?"
```bash
curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash
```

### Maintainer: "Hot fix needed!"
```bash
# 1. Fix bug
vim internal/whatever.go

# 2. Test
make build && ./ctl test-command

# 3. Bump patch version
vim internal/cli/root.go  # 0.1.0 -> 0.1.1

# 4. Release
git add .
git commit -m "fix: critical bug"
git tag v0.1.1
git push origin main --tags
```

### User: "Where are my worktrees?"
```bash
# Default location
ls ../.worktrees/$(basename $(git rev-parse --show-toplevel))

# Or use
ctl worktree list
```

### Maintainer: "Need to test before release"
```bash
# Build
make build

# Test commands
./ctl --version
./ctl worktree create test --no-claude
./ctl worktree list
./ctl worktree remove test -d

# Test install
make install-user
ctl --version
```

---

## File Locations

```
User files:
  ~/.config/ctl/config.yaml    # Global config
  .ctl.yaml                    # Repo config
  ../.worktrees/               # Worktrees

Project files:
  cmd/ctl/main.go             # Entry point
  internal/cli/               # Commands
  internal/worktree/          # Business logic
  go.mod                      # Dependencies
  Makefile                    # Build commands
```

---

## Help Commands

```bash
ctl --help                          # General help
ctl worktree --help                 # Worktree help
ctl worktree create --help          # Create help
ctl config --help                   # Config help

make help                           # Makefile help
```

---

## Troubleshooting

```bash
# Check version
ctl --version

# Verbose mode
ctl -v worktree create test

# Dry run
ctl --dry-run worktree create test

# Check config
ctl config list

# Rebuild
make clean && make build
```

---

## Links

- **Full Documentation**: [README.md](README.md)
- **Installation Guide**: [INSTALL.md](INSTALL.md)
- **Update Guide**: [UPDATING.md](UPDATING.md)
- **Changelog**: [CHANGELOG.md](CHANGELOG.md)
- **GitHub**: https://github.com/tharsanan1/ai-helper
