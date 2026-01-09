package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	pkgerrors "github.com/tharsanan1/ai-helper/pkg/errors"
)

// Client wraps git operations using os/exec
type Client struct {
	repoRoot string
}

// WorktreeInfo contains information about a git worktree
type WorktreeInfo struct {
	Path   string
	Branch string
	Commit string
}

// NewClient creates a new git client
// It verifies that the current directory is in a git repository
func NewClient() (*Client, error) {
	root, err := getRepositoryRoot()
	if err != nil {
		return nil, err
	}

	return &Client{
		repoRoot: root,
	}, nil
}

// getRepositoryRoot returns the root directory of the git repository
func getRepositoryRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 && strings.Contains(stderr.String(), "not a git repository") {
			return "", pkgerrors.ErrNotInGitRepo
		}
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetRepositoryRoot returns the repository root directory
func (c *Client) GetRepositoryRoot() string {
	return c.repoRoot
}

// GetCurrentBranch returns the name of the current branch
func (c *Client) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %s", stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// BranchExists checks if a branch exists
func (c *Client) BranchExists(branch string) (bool, error) {
	// Check if the branch exists by verifying refs/heads/<branch>
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)

	// show-ref returns exit code 0 if ref exists, 1 if not found
	// We don't need to capture output for --quiet
	err := cmd.Run()
	if err != nil {
		// Exit code 1 means branch not found (expected)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		// Any other error is unexpected
		return false, fmt.Errorf("failed to check branch existence: %w", err)
	}

	return true, nil
}

// RefExists checks if a reference exists (e.g., local branch, remote branch, tag)
func (c *Client) RefExists(ref string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	if err := cmd.Run(); err != nil {
		return false, nil
	}
	return true, nil
}

// CreateBranch creates a new branch from the specified source branch
func (c *Client) CreateBranch(name, from string) error {
	cmd := exec.Command("git", "branch", name, from)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "already exists") {
			return pkgerrors.NewBranchError(name, pkgerrors.ErrBranchExists)
		}
		return fmt.Errorf("failed to create branch: %s", stderr.String())
	}

	return nil
}

// CreateWorktree creates a new worktree at the specified path for the given branch
func (c *Client) CreateWorktree(path, branch string) error {
	cmd := exec.Command("git", "worktree", "add", path, branch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "already exists") {
			return pkgerrors.NewWorktreeError(path, pkgerrors.ErrWorktreeExists)
		}
		return fmt.Errorf("failed to create worktree: %s", stderr.String())
	}

	return nil
}

// ListWorktrees returns a list of all worktrees
func (c *Client) ListWorktrees() ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %s", stderr.String())
	}

	return parseWorktreeList(stdout.String()), nil
}

// RemoveWorktree removes a worktree at the specified path
func (c *Client) RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "not a working tree") {
			return pkgerrors.NewWorktreeError(path, pkgerrors.ErrWorktreeNotFound)
		}
		return fmt.Errorf("failed to remove worktree: %s", stderr.String())
	}

	return nil
}

// DeleteBranch deletes a branch
func (c *Client) DeleteBranch(branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "not found") {
			return pkgerrors.NewBranchError(branch, pkgerrors.ErrBranchNotFound)
		}
		return fmt.Errorf("failed to delete branch: %s", stderr.String())
	}

	return nil
}

// HasUnpushedCommits checks if the current branch has commits not pushed to origin
func (c *Client) HasUnpushedCommits(branch string) (bool, error) {
	// Check if remote branch exists
	err := exec.Command("git", "ls-remote", "--exit-code", "--heads", "origin", branch).Run()
	if err != nil {
		// Remote branch doesn't exist, so all commits are unpushed (if any)
		return true, nil
	}

	// Check for commits ahead of origin
	cmd := exec.Command("git", "rev-list", "--count", "origin/"+branch+"..HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to check unpushed commits: %w", err)
	}

	count := strings.TrimSpace(stdout.String())
	return count != "0", nil
}

// Push pushes the specified branch to origin
func (c *Client) Push(branch string) error {
	cmd := exec.Command("git", "push", "-u", "origin", branch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch: %s", stderr.String())
	}
	return nil
}

// HasRemote checks if a remote with the given name exists
func (c *Client) HasRemote(name string) (bool, error) {
	cmd := exec.Command("git", "remote", "get-url", name)
	if err := cmd.Run(); err != nil {
		return false, nil
	}
	return true, nil
}

// GetRemoteURL returns the URL of the remote with the given name
func (c *Client) GetRemoteURL(name string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", name)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("remote %s not found", name)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// FetchRef fetches a specific reference from a remote
func (c *Client) FetchRef(remote, refSpec string) error {
	cmd := exec.Command("git", "fetch", remote, refSpec)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch ref %s from %s: %s", refSpec, remote, stderr.String())
	}
	return nil
}

// GetCommitLogs returns the commit logs between base and head
func (c *Client) GetCommitLogs(base, head string) (string, error) {
	cmd := exec.Command("git", "log", "--no-merges", "--pretty=format:%h %s", fmt.Sprintf("%s..%s", base, head))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get commit logs: %w", err)
	}
	return stdout.String(), nil
}

// parseWorktreeList parses the porcelain output of git worktree list
func parseWorktreeList(output string) []WorktreeInfo {
	var worktrees []WorktreeInfo
	var current WorktreeInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "worktree":
			current.Path = value
		case "branch":
			// Branch is in format "refs/heads/branch-name"
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "HEAD":
			current.Commit = value
		}
	}

	// Add the last worktree if exists
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees
}
