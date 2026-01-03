package worktree

import (
	"github.com/spf13/cobra"
)

// WorktreeCmd represents the worktree command group
var WorktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage git worktrees",
	Long: `Manage git worktrees with integrated tool launching.

The worktree command group provides subcommands for creating, listing,
removing, and switching between git worktrees.`,
}

func init() {
	// Add subcommands
	WorktreeCmd.AddCommand(createCmd)
	WorktreeCmd.AddCommand(listCmd)
	WorktreeCmd.AddCommand(removeCmd)
	WorktreeCmd.AddCommand(switchCmd)
}
