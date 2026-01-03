package worktree

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	pkgerrors "github.com/tharsanan/ai-helper/pkg/errors"
)

var (
	// Valid worktree name pattern: alphanumeric, hyphens, underscores
	worktreeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9_]*$`)

	// Valid branch name pattern (simplified - git is more permissive)
	branchNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9_/]*$`)
)

// ValidateWorktreeName validates a worktree name
func ValidateWorktreeName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", pkgerrors.ErrInvalidWorktreeName)
	}

	if len(name) > 100 {
		return fmt.Errorf("%w: name too long (max 100 characters)", pkgerrors.ErrInvalidWorktreeName)
	}

	if !worktreeNamePattern.MatchString(name) {
		return fmt.Errorf("%w: name must start with alphanumeric and contain only letters, numbers, hyphens, and underscores", pkgerrors.ErrInvalidWorktreeName)
	}

	// Prevent some reserved/problematic names
	reserved := []string{".", "..", "HEAD", "main", "master"}
	for _, r := range reserved {
		if name == r {
			return fmt.Errorf("%w: %q is a reserved name", pkgerrors.ErrInvalidWorktreeName, name)
		}
	}

	return nil
}

// ValidateBranchName validates a branch name
func ValidateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", pkgerrors.ErrInvalidBranchName)
	}

	if len(name) > 255 {
		return fmt.Errorf("%w: name too long (max 255 characters)", pkgerrors.ErrInvalidBranchName)
	}

	if !branchNamePattern.MatchString(name) {
		return fmt.Errorf("%w: name must start with alphanumeric and contain only letters, numbers, hyphens, underscores, and slashes", pkgerrors.ErrInvalidBranchName)
	}

	// Check for problematic patterns
	if strings.Contains(name, "..") {
		return fmt.Errorf("%w: name cannot contain '..'", pkgerrors.ErrInvalidBranchName)
	}

	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("%w: name cannot start or end with '/'", pkgerrors.ErrInvalidBranchName)
	}

	if strings.Contains(name, "//") {
		return fmt.Errorf("%w: name cannot contain consecutive slashes", pkgerrors.ErrInvalidBranchName)
	}

	return nil
}

// ValidatePath validates a path and ensures it's safe
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: path cannot be empty", pkgerrors.ErrInvalidPath)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: failed to resolve absolute path: %v", pkgerrors.ErrInvalidPath, err)
	}

	// Check for path traversal attempts
	if strings.Contains(absPath, "..") {
		return fmt.Errorf("%w: path contains directory traversal", pkgerrors.ErrInvalidPath)
	}

	return nil
}

// SanitizeBranchName converts a worktree name to a valid branch name
// This is used when the user doesn't specify a branch name
func SanitizeBranchName(name string) string {
	// Replace underscores with hyphens for convention
	name = strings.ReplaceAll(name, "_", "-")

	// Ensure it's lowercase (common convention)
	name = strings.ToLower(name)

	return name
}
