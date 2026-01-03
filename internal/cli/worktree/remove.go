package worktree

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/git"
	"github.com/tharsanan1/ai-helper/internal/util"
	"github.com/tharsanan1/ai-helper/internal/worktree"
)

var (
	// Flags for remove command
	removeDeleteBranch bool
	removeForce        bool
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a worktree",
	Long: `Remove a git worktree and optionally delete the associated branch.

The worktree directory will be removed from the filesystem.
Use the --delete-branch flag to also delete the git branch.
Use the --force flag to remove worktrees with modified or untracked files.

Examples:
  # Remove worktree only
  ctl worktree remove feature-auth

  # Remove worktree and delete branch
  ctl worktree remove feature-auth --delete-branch

  # Force remove worktree with untracked files
  ctl worktree remove feature-auth --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeDeleteBranch, "delete-branch", "d", false, "Also delete the associated git branch")
	removeCmd.Flags().BoolVar(&removeForce, "force", false, "Force remove worktree even with modified or untracked files")
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Get config manager
	cfgManager, err := util.GlobalContext.GetConfigManager()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	cfg, err := cfgManager.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create git client
	gitClient, err := git.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize git client: %w", err)
	}

	// Create worktree manager
	wtManager := worktree.NewManager(gitClient, cfg)

	// Prepare remove options
	deleteBranch := removeDeleteBranch
	if !removeDeleteBranch && cfg.Worktree.AutoCleanup {
		deleteBranch = true
	}

	opts := worktree.RemoveOptions{
		Name:         name,
		DeleteBranch: deleteBranch,
		Force:        removeForce,
	}

	// Print what we're doing if verbose
	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Removing worktree %q...\n", name)
		if deleteBranch {
			fmt.Println("  Will also delete associated branch")
		}
	}

	// Remove the worktree
	if util.GlobalContext.IsDryRun() {
		fmt.Println("Dry run: would remove worktree", name)
		if deleteBranch {
			fmt.Println("Dry run: would delete associated branch")
		}
		return nil
	}

	if err := wtManager.Remove(opts); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Print success message
	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Removed worktree: %s\n", name)
		if deleteBranch {
			color.Green("✓ Deleted associated branch\n")
		}
	} else {
		fmt.Printf("Removed worktree: %s\n", name)
		if deleteBranch {
			fmt.Println("Deleted associated branch")
		}
	}

	return nil
}
