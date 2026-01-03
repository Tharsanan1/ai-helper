# Updating ctl

## For Users: How to Update

### Quick Update (Recommended)

Re-run the installation script to get the latest version:

```bash
curl -fsSL https://raw.githubusercontent.com/tharsanan1/ai-helper/main/install.sh | bash
```

### Manual Update

If you cloned the repository:

```bash
cd /path/to/ai-helper
git pull origin main
make install
```

### Using Go Install

```bash
go install github.com/tharsanan1/ai-helper/cmd/ctl@latest
```

### Check Your Current Version

```bash
ctl --version
```

---

## For Maintainers: Release Workflow

### 1. Making Changes

```bash
# Make your changes
vim internal/whatever.go

# Test locally
make build
./ctl --help

# Commit and push
git add .
git commit -m "feat: add new feature"
git push origin main
```

### 2. Creating a Release

#### Update Version Number

Edit `internal/cli/root.go`:

```go
var rootCmd = &cobra.Command{
    Use:   "ctl",
    Short: "...",
    Version: "0.2.0",  // Update this
}
```

#### Create Changelog Entry

Add to `CHANGELOG.md`:

```markdown
## [0.2.0] - 2024-01-15

### Added
- New worktree merge command
- PR creation support

### Fixed
- Branch existence check bug

### Changed
- Improved error messages
```

#### Tag and Push

```bash
# Commit version bump
git add .
git commit -m "chore: bump version to 0.2.0"

# Create tag
git tag -a v0.2.0 -m "Release version 0.2.0"

# Push with tags
git push origin main --tags
```

#### Build Release Binaries

```bash
# Build for all platforms
make release

# Binaries will be in dist/
ls dist/
# ctl-darwin-amd64
# ctl-darwin-arm64
# ctl-linux-amd64
# ctl-linux-arm64
# ctl-windows-amd64.exe
```

#### Create GitHub Release

1. Go to https://github.com/tharsanan1/ai-helper/releases
2. Click "Draft a new release"
3. Choose tag: `v0.2.0`
4. Title: `v0.2.0 - Feature Name`
5. Description: Copy from CHANGELOG.md
6. Upload binaries from `dist/`
7. Publish release

### 3. Automated Releases with GitHub Actions (Optional)

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Build
        run: make release

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: dist/*
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Then releases happen automatically when you push a tag!

---

## Version Numbering (SemVer)

Follow semantic versioning: `MAJOR.MINOR.PATCH`

- **MAJOR** (1.0.0): Breaking changes, incompatible API changes
- **MINOR** (0.1.0): New features, backwards compatible
- **PATCH** (0.0.1): Bug fixes, backwards compatible

Examples:
- `v0.1.0` → `v0.1.1`: Bug fix
- `v0.1.1` → `v0.2.0`: New feature (worktree merge)
- `v0.2.0` → `v1.0.0`: Breaking change (command structure changed)

---

## Communicating Updates

### Announce in README

Add a "What's New" section:

```markdown
## 🎉 What's New

**v0.2.0** (2024-01-15)
- Added `ctl worktree merge` command
- Improved error messages
- [Full changelog](CHANGELOG.md)
```

### GitHub Discussions/Issues

Create a discussion post for major releases to notify users.

---

## Hotfix Workflow

For urgent bug fixes:

```bash
# 1. Create hotfix branch
git checkout -b hotfix/critical-bug

# 2. Fix the bug
vim internal/whatever.go

# 3. Test
make build
./ctl test-command

# 4. Commit and merge
git commit -am "fix: critical bug in worktree creation"
git checkout main
git merge hotfix/critical-bug

# 5. Bump patch version and release
# 0.1.0 → 0.1.1
git tag v0.1.1
git push origin main --tags
```

---

## Deprecation Workflow

When removing features:

1. **Mark as deprecated** (1 version before removal):
   ```go
   // Deprecated: Use NewCommand instead. Will be removed in v2.0.0
   func OldCommand() { ... }
   ```

2. **Add warning message**:
   ```go
   fmt.Fprintf(os.Stderr, "Warning: This command is deprecated and will be removed in v2.0.0\n")
   ```

3. **Update documentation**

4. **Remove in next major version**

---

## User Migration Guide

When you make breaking changes, create a migration guide:

### Example: `MIGRATION.md`

```markdown
# Migration Guide

## Upgrading from v0.x to v1.0

### Breaking Changes

**Command structure changed:**

Old:
```bash
ctl create-worktree feature-x
```

New:
```bash
ctl worktree create feature-x
```

**Config location changed:**

Old: `~/.ctl/config.yaml`
New: `~/.config/ctl/config.yaml`

To migrate:
```bash
mv ~/.ctl/config.yaml ~/.config/ctl/config.yaml
```
```

---

## Testing Releases

Before tagging:

```bash
# Test build
make clean && make build

# Test installation
make install-user

# Test commands
ctl --version
ctl worktree create test --no-claude
ctl worktree list
ctl worktree remove test

# Test on different platforms (if possible)
GOOS=linux make build
GOOS=windows make build
```

---

## Rollback Procedure

If a release has critical bugs:

```bash
# 1. Delete the bad tag locally and remotely
git tag -d v0.2.0
git push origin :refs/tags/v0.2.0

# 2. Delete GitHub release

# 3. Fix the issue

# 4. Re-release with patch version
git tag v0.2.1
git push origin main --tags
```

---

## Questions?

- Open an issue: https://github.com/tharsanan1/ai-helper/issues
- Start a discussion: https://github.com/tharsanan1/ai-helper/discussions
