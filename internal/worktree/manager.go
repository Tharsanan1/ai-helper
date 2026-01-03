package worktree

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/git"
	pkgerrors "github.com/tharsanan1/ai-helper/pkg/errors"
)

// Manager handles worktree operations
type Manager struct {
	gitClient *git.Client
	config    *config.Config
}

// CreateOptions contains options for creating a worktree
type CreateOptions struct {
	// Name is the name of the worktree
	Name string

	// Branch is the branch name (defaults to Name if not provided)
	Branch string

	// Location is the base directory for the worktree (defaults to config value)
	Location string

	// From is the source branch to create from (defaults to current branch)
	From string

	// ExistingBranch indicates whether to use an existing branch
	ExistingBranch bool
}

// RemoveOptions contains options for removing a worktree
type RemoveOptions struct {
	// Name is the name of the worktree
	Name string

	// DeleteBranch indicates whether to delete the associated branch
	DeleteBranch bool

	// Force indicates whether to force remove the worktree
	Force bool
}

// WorktreeInfo contains information about a worktree
type WorktreeInfo struct {
	Name   string
	Path   string
	Branch string
	Commit string
}

// NewManager creates a new worktree manager
func NewManager(gitClient *git.Client, cfg *config.Config) *Manager {
	return &Manager{
		gitClient: gitClient,
		config:    cfg,
	}
}

// Create creates a new worktree
func (m *Manager) Create(opts CreateOptions) (string, error) {
	// Validate worktree name
	if err := ValidateWorktreeName(opts.Name); err != nil {
		return "", err
	}

	// Determine branch name
	branchName := opts.Branch
	if branchName == "" {
		branchName = SanitizeBranchName(opts.Name)
	}

	// Validate branch name
	if err := ValidateBranchName(branchName); err != nil {
		return "", err
	}

	// Determine source branch
	fromBranch := opts.From
	if fromBranch == "" {
		// Prefer current branch, fall back to config default
		var err error
		fromBranch, err = m.gitClient.GetCurrentBranch()
		if err != nil {
			// If we can't get current branch, try config default
			if m.config.Worktree.DefaultSourceBranch != "" {
				fromBranch = m.config.Worktree.DefaultSourceBranch
			} else {
				return "", fmt.Errorf("failed to determine source branch: %w", err)
			}
		}
	}

	// Determine worktree location
	location := opts.Location
	if location == "" {
		location = m.config.Worktree.BaseLocation
	}

	// Compute full worktree path
	worktreePath, err := m.computeWorktreePath(location, opts.Name)
	if err != nil {
		return "", err
	}

	// Validate path
	if err := ValidatePath(worktreePath); err != nil {
		return "", err
	}

	// Check if worktree path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return "", pkgerrors.NewWorktreeError(opts.Name, pkgerrors.ErrWorktreeExists)
	}

	// Check if branch exists
	branchExists, err := m.gitClient.BranchExists(branchName)
	if err != nil {
		return "", fmt.Errorf("failed to check branch existence: %w", err)
	}

	if branchExists && !opts.ExistingBranch {
		return "", fmt.Errorf("branch %q already exists. Use --existing-branch flag to use it, or choose a different branch name", branchName)
	}

	if !branchExists && opts.ExistingBranch {
		return "", pkgerrors.NewBranchError(branchName, pkgerrors.ErrBranchNotFound)
	}

	// Create branch if it doesn't exist
	if !branchExists {
		if err := m.gitClient.CreateBranch(branchName, fromBranch); err != nil {
			return "", fmt.Errorf("failed to create branch: %w", err)
		}
	}

	// Create worktree
	if err := m.gitClient.CreateWorktree(worktreePath, branchName); err != nil {
		// Cleanup: remove branch if we created it
		if !branchExists {
			_ = m.gitClient.DeleteBranch(branchName)
		}
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}

	return worktreePath, nil
}

// List returns a list of all worktrees
func (m *Manager) List() ([]WorktreeInfo, error) {
	gitWorktrees, err := m.gitClient.ListWorktrees()
	if err != nil {
		return nil, err
	}

	worktrees := make([]WorktreeInfo, 0, len(gitWorktrees))
	for _, gw := range gitWorktrees {
		// Extract worktree name from path
		name := filepath.Base(gw.Path)

		worktrees = append(worktrees, WorktreeInfo{
			Name:   name,
			Path:   gw.Path,
			Branch: gw.Branch,
			Commit: gw.Commit,
		})
	}

	return worktrees, nil
}

// Remove removes a worktree
func (m *Manager) Remove(opts RemoveOptions) error {
	// Validate worktree name
	if err := ValidateWorktreeName(opts.Name); err != nil {
		return err
	}

	// Find the worktree
	worktrees, err := m.List()
	if err != nil {
		return err
	}

	var targetWorktree *WorktreeInfo
	for i := range worktrees {
		if worktrees[i].Name == opts.Name {
			targetWorktree = &worktrees[i]
			break
		}
	}

	if targetWorktree == nil {
		return pkgerrors.NewWorktreeError(opts.Name, pkgerrors.ErrWorktreeNotFound)
	}

	// Remove the worktree
	if err := m.gitClient.RemoveWorktree(targetWorktree.Path, opts.Force); err != nil {
		return err
	}

	// Delete branch if requested
	if opts.DeleteBranch && targetWorktree.Branch != "" {
		if err := m.gitClient.DeleteBranch(targetWorktree.Branch); err != nil {
			// Don't fail if branch deletion fails, just warn
			fmt.Fprintf(os.Stderr, "Warning: failed to delete branch %q: %v\n", targetWorktree.Branch, err)
		}
	}

	return nil
}

// GetPath returns the path for a worktree
func (m *Manager) GetPath(name string) (string, error) {
	worktrees, err := m.List()
	if err != nil {
		return "", err
	}

	for _, wt := range worktrees {
		if wt.Name == name {
			return wt.Path, nil
		}
	}

	return "", pkgerrors.NewWorktreeError(name, pkgerrors.ErrWorktreeNotFound)
}

// computeWorktreePath computes the full path for a worktree
func (m *Manager) computeWorktreePath(location, name string) (string, error) {
	var basePath string

	// Check if location is absolute
	if filepath.IsAbs(location) {
		basePath = location
	} else {
		// Relative to repo parent
		repoRoot := m.gitClient.GetRepositoryRoot()
		repoParent := filepath.Dir(repoRoot)
		basePath = filepath.Join(repoParent, location)
	}

	// Include repo name in the path
	repoName := filepath.Base(m.gitClient.GetRepositoryRoot())
	fullPath := filepath.Join(basePath, repoName, name)

	// Ensure parent directory exists
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree parent directory: %w", err)
	}

	return fullPath, nil
}
