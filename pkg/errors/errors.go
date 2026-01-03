package errors

import (
	"errors"
	"fmt"
)

var (
	// ErrNotInGitRepo is returned when the current directory is not in a git repository
	ErrNotInGitRepo = errors.New("not in a git repository")

	// ErrWorktreeExists is returned when trying to create a worktree that already exists
	ErrWorktreeExists = errors.New("worktree already exists")

	// ErrWorktreeNotFound is returned when the specified worktree cannot be found
	ErrWorktreeNotFound = errors.New("worktree not found")

	// ErrBranchNotFound is returned when the specified branch cannot be found
	ErrBranchNotFound = errors.New("branch not found")

	// ErrBranchExists is returned when trying to create a branch that already exists
	ErrBranchExists = errors.New("branch already exists")

	// ErrToolNotAvailable is returned when a required tool is not available
	ErrToolNotAvailable = errors.New("tool not available")

	// ErrInvalidWorktreeName is returned when the worktree name is invalid
	ErrInvalidWorktreeName = errors.New("invalid worktree name")

	// ErrInvalidBranchName is returned when the branch name is invalid
	ErrInvalidBranchName = errors.New("invalid branch name")

	// ErrInvalidPath is returned when a path is invalid or unsafe
	ErrInvalidPath = errors.New("invalid path")

	// ErrPermissionDenied is returned when there are permission issues
	ErrPermissionDenied = errors.New("permission denied")
)

// WorktreeError wraps an error with additional context about the worktree
type WorktreeError struct {
	Worktree string
	Err      error
}

func (e *WorktreeError) Error() string {
	return fmt.Sprintf("worktree %q: %v", e.Worktree, e.Err)
}

func (e *WorktreeError) Unwrap() error {
	return e.Err
}

// NewWorktreeError creates a new WorktreeError
func NewWorktreeError(worktree string, err error) *WorktreeError {
	return &WorktreeError{
		Worktree: worktree,
		Err:      err,
	}
}

// BranchError wraps an error with additional context about the branch
type BranchError struct {
	Branch string
	Err    error
}

func (e *BranchError) Error() string {
	return fmt.Sprintf("branch %q: %v", e.Branch, e.Err)
}

func (e *BranchError) Unwrap() error {
	return e.Err
}

// NewBranchError creates a new BranchError
func NewBranchError(branch string, err error) *BranchError {
	return &BranchError{
		Branch: branch,
		Err:    err,
	}
}

// ToolError wraps an error with additional context about a tool
type ToolError struct {
	Tool string
	Err  error
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("tool %q: %v", e.Tool, e.Err)
}

func (e *ToolError) Unwrap() error {
	return e.Err
}

// NewToolError creates a new ToolError
func NewToolError(tool string, err error) *ToolError {
	return &ToolError{
		Tool: tool,
		Err:  err,
	}
}
