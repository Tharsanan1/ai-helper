package cli

import (
	"github.com/spf13/cobra"
	cliConfig "github.com/tharsanan1/ai-helper/internal/cli/config"
	"github.com/tharsanan1/ai-helper/internal/cli/extract"
	"github.com/tharsanan1/ai-helper/internal/cli/launch"
	"github.com/tharsanan1/ai-helper/internal/cli/pr"
	"github.com/tharsanan1/ai-helper/internal/cli/setupcopilot"
	"github.com/tharsanan1/ai-helper/internal/cli/syncmain"
	"github.com/tharsanan1/ai-helper/internal/cli/worktree"

	"github.com/tharsanan1/ai-helper/internal/util"
)

var (
	// Global flags
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "aihelper",
	Short: "AI-powered Git worktree manager with Claude Code integration",
	Long: `aihelper is a command-line tool that helps you manage git worktrees
and seamlessly launch development tools like Claude Code CLI.

It provides an intuitive interface for creating, listing, and managing
git worktrees, with built-in support for automatically launching AI tools
in the worktree context.`,
	Version: "0.1.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/aihelper/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&util.GlobalContext.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&util.GlobalContext.DryRun, "dry-run", false, "dry run mode (don't execute actions)")
	rootCmd.PersistentFlags().BoolVar(&util.GlobalContext.NoColor, "no-color", false, "disable colored output")

	// Add command groups
	rootCmd.AddCommand(worktree.WorktreeCmd)
	rootCmd.AddCommand(cliConfig.ConfigCmd)
	rootCmd.AddCommand(launch.LaunchCmd)
	rootCmd.AddCommand(pr.PrCmd)
	rootCmd.AddCommand(extract.ExtractCmd)
	rootCmd.AddCommand(setupcopilot.SetupCopilotCmd)
	rootCmd.AddCommand(syncmain.SyncMainCmd)

	// Add single-letter shortcuts
	rootCmd.AddCommand(worktree.CreateShortcutCmd)
	rootCmd.AddCommand(worktree.SwitchShortcutCmd)
	rootCmd.AddCommand(worktree.ListShortcutCmd)
	rootCmd.AddCommand(worktree.RemoveShortcutCmd)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	// Config manager will be initialized when needed in commands
}
