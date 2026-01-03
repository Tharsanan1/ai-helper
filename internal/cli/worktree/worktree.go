package worktree

import (
	"github.com/spf13/cobra"
)

var (
	WorktreeCmd       *cobra.Command
	CreateShortcutCmd *cobra.Command
	SwitchShortcutCmd *cobra.Command
	ListShortcutCmd   *cobra.Command
	RemoveShortcutCmd *cobra.Command
)

func init() {
	// Initialize WorktreeCmd
	WorktreeCmd = &cobra.Command{
		Use:     "worktree",
		Aliases: []string{"wt", "tree"},
		Short:   "Manage git worktrees",
		Long: `Manage git worktrees with integrated tool launching.

The worktree command group provides subcommands for creating, listing,
removing, and switching between git worktrees.`,
	}

	// Add subcommands to worktree
	WorktreeCmd.AddCommand(createCmd)
	WorktreeCmd.AddCommand(listCmd)
	WorktreeCmd.AddCommand(removeCmd)
	WorktreeCmd.AddCommand(switchCmd)

	// Create shortcuts that delegate to worktree subcommands
	CreateShortcutCmd = &cobra.Command{
		Use:     "c <name>",
		Aliases: []string{"new"},
		Short:   "Create a new worktree",
		Example: `  aihelper c feature-auth
  aihelper c feature-auth -b auth/login`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			createCmd.SetArgs(args)
			return createCmd.Execute()
		},
	}

	SwitchShortcutCmd = &cobra.Command{
		Use:     "sw <name>",
		Aliases: []string{"go"},
		Short:   "Switch to an existing worktree",
		Example: `  aihelper sw feature-auth
  aihelper sw feature-auth --claude-mode chat`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switchCmd.SetArgs(args)
			return switchCmd.Execute()
		},
	}

	ListShortcutCmd = &cobra.Command{
		Use:   "ls",
		Short: "List all worktrees",
		Example: `  aihelper ls
  aihelper ls --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			listCmd.SetArgs(args)
			return listCmd.Execute()
		},
	}

	RemoveShortcutCmd = &cobra.Command{
		Use:     "rm <name> [<name>...]",
		Aliases: []string{"del"},
		Short:   "Remove worktrees",
		Example: `  aihelper rm feature-auth
  aihelper rm feature-auth --delete-branch`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			removeCmd.SetArgs(args)
			return removeCmd.Execute()
		},
	}
}
