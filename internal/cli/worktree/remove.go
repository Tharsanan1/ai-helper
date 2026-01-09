package worktree

import (
	"fmt"
	"os"

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
	Use:     "remove <name> [<name>...]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove one or more worktrees",
	Long: `Remove git worktrees and optionally delete the associated branches.

The worktree directories will be removed from the filesystem.
Use the --delete-branch flag to also delete the git branches.
Use the --force flag to remove worktrees with modified or untracked files.

Examples:
  # Remove a single worktree
  aihelper worktree remove feature-auth

  # Remove multiple worktrees
  aihelper worktree remove feature-auth feature-billing

  # Remove worktrees and delete branches
  aihelper worktree remove feature-auth feature-billing --delete-branch

  # Force remove worktrees with untracked files
  aihelper worktree remove feature-auth --force`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: getWorktreeNames,
	RunE:              runRemove,
}

// RegisterRemoveFlags registers flags for the remove command
func RegisterRemoveFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&removeDeleteBranch, "delete-branch", "d", false, "Also delete the associated git branch")
	cmd.Flags().BoolVar(&removeForce, "force", false, "Force remove worktree even with modified or untracked files")
}

func init() {
	RegisterRemoveFlags(removeCmd)
}

func RunRemove(cmd *cobra.Command, args []string) error {
	return runRemove(cmd, args)
}

func runRemove(cmd *cobra.Command, args []string) error {
	cfgManager, err := util.GlobalContext.GetConfigManager()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	cfg, err := cfgManager.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	gitClient, err := git.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize git client: %w", err)
	}

	wtManager := worktree.NewManager(gitClient, cfg)

	deleteBranch := removeDeleteBranch
	if !removeDeleteBranch && cfg.Worktree.AutoCleanup {
		deleteBranch = true
	}

	hasErrors := false

	for _, name := range args {
		opts := worktree.RemoveOptions{
			Name:         name,
			DeleteBranch: deleteBranch,
			Force:        removeForce,
		}

		if util.GlobalContext.IsVerbose() {
			fmt.Printf("Removing worktree %q...\n", name)
			if deleteBranch {
				fmt.Println("  Will also delete associated branch")
			}
		}

		if util.GlobalContext.IsDryRun() {
			fmt.Println("Dry run: would remove worktree", name)
			if deleteBranch {
				fmt.Println("Dry run: would delete associated branch")
			}
			continue
		}

		if err := wtManager.Remove(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing worktree %q: %v\n", name, err)
			hasErrors = true
			continue
		}

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
	}

	if hasErrors {
		return fmt.Errorf("one or more worktrees failed to remove")
	}

	return nil
}
