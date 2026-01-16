package syncmain

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var (
	// Flags for sync-main command
	upstreamName string
	branchName   string
	noPush       bool
)

// SyncMainCmd represents the sync-main command
var SyncMainCmd = &cobra.Command{
	Use:   "sync-main",
	Short: "Sync local main branch with upstream",
	Long: `Sync the local main branch with the upstream remote.

This command will:
1. Verify this is a git repository
2. Verify the upstream remote exists
3. Fetch the latest from upstream
4. Update local main branch to match upstream/main
5. Force push to origin (unless --no-push is specified)

This is useful for keeping a fork's main branch in sync with the upstream repository.

Example:
  aihelper sync-main
  aihelper sync-main --upstream=upstream --branch=main
  aihelper sync-main --no-push`,
	RunE: runSyncMain,
}

func init() {
	SyncMainCmd.Flags().StringVar(&upstreamName, "upstream", "upstream", "Name of the upstream remote")
	SyncMainCmd.Flags().StringVar(&branchName, "branch", "main", "Name of the branch to sync")
	SyncMainCmd.Flags().BoolVar(&noPush, "no-push", false, "Don't push to origin after syncing")
}

func runSyncMain(cmd *cobra.Command, args []string) error {
	// Step 1: Check if this is a git repository
	if err := checkGitRepo(); err != nil {
		return err
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Println("Verified: This is a git repository")
	}

	// Step 2: Check if upstream remote exists
	if err := checkRemoteExists(upstreamName); err != nil {
		return err
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Verified: Remote '%s' exists\n", upstreamName)
	}

	// Step 3: Fetch from upstream
	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Fetching from %s...\n", upstreamName)
	} else {
		fmt.Printf("Fetching from %s...\n", upstreamName)
	}

	if !util.GlobalContext.IsDryRun() {
		if err := gitFetch(upstreamName); err != nil {
			return err
		}
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Fetched from %s\n", upstreamName)
	} else {
		fmt.Printf("✓ Fetched from %s\n", upstreamName)
	}

	// Step 4: Check if upstream branch exists
	upstreamBranch := fmt.Sprintf("%s/%s", upstreamName, branchName)
	if err := checkBranchExists(upstreamBranch); err != nil {
		return fmt.Errorf("upstream branch '%s' does not exist: %w", upstreamBranch, err)
	}

	// Step 5: Update local branch to match upstream
	// Using git branch -f to force update the local branch
	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Updating local '%s' to match '%s'...\n", branchName, upstreamBranch)
	} else {
		fmt.Printf("Updating local '%s' to match '%s'...\n", branchName, upstreamBranch)
	}

	if !util.GlobalContext.IsDryRun() {
		if err := updateLocalBranch(branchName, upstreamBranch); err != nil {
			return err
		}
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Updated local '%s' branch\n", branchName)
	} else {
		fmt.Printf("✓ Updated local '%s' branch\n", branchName)
	}

	// Step 6: Force push to origin (unless --no-push)
	if !noPush {
		// Check if origin remote exists
		if err := checkRemoteExists("origin"); err != nil {
			return fmt.Errorf("origin remote not found: %w", err)
		}

		if util.GlobalContext.IsColorEnabled() {
			color.Cyan("Force pushing '%s' to origin...\n", branchName)
		} else {
			fmt.Printf("Force pushing '%s' to origin...\n", branchName)
		}

		if !util.GlobalContext.IsDryRun() {
			if err := gitForcePush("origin", branchName); err != nil {
				return err
			}
		}

		if util.GlobalContext.IsColorEnabled() {
			color.Green("✓ Force pushed '%s' to origin\n", branchName)
		} else {
			fmt.Printf("✓ Force pushed '%s' to origin\n", branchName)
		}
	} else {
		fmt.Println("Skipping push to origin (--no-push flag specified)")
	}

	// Final success message
	if util.GlobalContext.IsColorEnabled() {
		color.Green("\n✓ Successfully synced '%s' with %s/%s\n", branchName, upstreamName, branchName)
	} else {
		fmt.Printf("\n✓ Successfully synced '%s' with %s/%s\n", branchName, upstreamName, branchName)
	}

	return nil
}

// checkGitRepo verifies the current directory is a git repository
func checkGitRepo() error {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("not a git repository: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// checkRemoteExists verifies a remote exists
func checkRemoteExists(remoteName string) error {
	cmd := exec.Command("git", "remote", "get-url", remoteName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("remote '%s' not found: %s", remoteName, strings.TrimSpace(string(output)))
	}
	return nil
}

// checkBranchExists verifies a branch (local or remote) exists
func checkBranchExists(branchName string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("branch '%s' does not exist", branchName)
	}
	return nil
}

// gitFetch fetches from a remote
func gitFetch(remoteName string) error {
	cmd := exec.Command("git", "fetch", remoteName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %s - %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// updateLocalBranch updates a local branch to point to a specific ref
func updateLocalBranch(localBranch, targetRef string) error {
	// Use git branch -f to force update the local branch
	// This works even if we're currently on a different branch
	cmd := exec.Command("git", "branch", "-f", localBranch, targetRef)
	if output, err := cmd.CombinedOutput(); err != nil {
		// If we're currently on this branch, we need a different approach
		if strings.Contains(string(output), "cannot force update the current branch") {
			// Switch to detached HEAD, update branch, then we're done
			// Or use git reset if we're on the branch
			resetCmd := exec.Command("git", "checkout", localBranch)
			if _, err := resetCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to checkout %s: %w", localBranch, err)
			}
			resetCmd = exec.Command("git", "reset", "--hard", targetRef)
			if output, err := resetCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("git reset failed: %s - %w", strings.TrimSpace(string(output)), err)
			}
			return nil
		}
		return fmt.Errorf("git branch -f failed: %s - %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// gitForcePush force pushes a branch to a remote
func gitForcePush(remoteName, branchName string) error {
	cmd := exec.Command("git", "push", "--force", remoteName, branchName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push --force failed: %s - %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}
