package worktree

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/git"
	"github.com/tharsanan1/ai-helper/internal/util"
	"github.com/tharsanan1/ai-helper/internal/worktree"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all worktrees",
	Long: `List all git worktrees with their branches and paths.

Displays a table showing:
- Worktree name
- Branch
- Path
- Latest commit (short hash)`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Get list of worktrees
	worktrees, err := wtManager.List()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found.")
		return nil
	}

	// Display worktrees
	fmt.Printf("%-20s %-30s %-10s %s\n", "NAME", "BRANCH", "COMMIT", "PATH")
	fmt.Println(strings.Repeat("-", 80))

	for _, wt := range worktrees {
		commit := wt.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}

		branch := wt.Branch
		if branch == "" {
			branch = "(detached)"
		}

		// Color the output if enabled
		if util.GlobalContext.IsColorEnabled() {
			fmt.Printf("%-20s %-30s %-10s %s\n",
				color.CyanString(wt.Name),
				color.GreenString(branch),
				color.YellowString(commit),
				wt.Path)
		} else {
			fmt.Printf("%-20s %-30s %-10s %s\n", wt.Name, branch, commit, wt.Path)
		}
	}

	return nil
}

func RunList(cmd *cobra.Command, args []string) error {
	return runList(cmd, args)
}
