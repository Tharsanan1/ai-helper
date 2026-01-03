package worktree

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/git"
	"github.com/tharsanan1/ai-helper/internal/launcher"
	"github.com/tharsanan1/ai-helper/internal/util"
	"github.com/tharsanan1/ai-helper/internal/worktree"
)

var (
	// Flags for switch command
	switchClaudeMode   string
	switchClaudeArgs   []string
	switchOpenCode     bool
	switchGemini       bool
	switchDroid        bool
	switchTerminalName string
)

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:     "switch <name>",
	Aliases: []string{"sw", "go"},
	Short:   "Switch to an existing worktree and launch Claude",
	Long: `Switch to an existing git worktree and launch Claude Code CLI.

This command finds the worktree by name and launches Claude in its directory.

Examples:
  # Switch to a worktree
  ctl worktree switch feature-auth

  # Switch with custom Claude mode
  ctl worktree switch feature-auth --claude-mode chat

  # Switch and launch OpenCode instead of Claude
  ctl worktree switch feature-auth --opencode

  # Switch and launch Gemini CLI instead of Claude
  ctl worktree switch feature-auth --gemini

  # Switch and launch Droid CLI instead of Claude
  ctl worktree switch feature-auth --droid`,
	Args: cobra.ExactArgs(1),
	RunE: runSwitch,
}

// RegisterSwitchFlags registers flags for the switch command
func RegisterSwitchFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&switchClaudeMode, "claude-mode", "", "Claude mode: chat, agent (default from config)")
	cmd.Flags().StringSliceVar(&switchClaudeArgs, "claude-args", []string{}, "Additional arguments to pass to Claude CLI")
	cmd.Flags().BoolVar(&switchOpenCode, "opencode", false, "Launch OpenCode instead of Claude")
	cmd.Flags().BoolVar(&switchGemini, "gemini", false, "Launch Gemini CLI instead of Claude")
	cmd.Flags().BoolVar(&switchDroid, "droid", false, "Launch Droid CLI instead of Claude")
	cmd.Flags().StringVar(&switchTerminalName, "terminal-name", "", "Terminal window name (default: worktree name)")
}

func init() {
	RegisterSwitchFlags(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
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

	// Get worktree path
	worktreePath, err := wtManager.GetPath(name)
	if err != nil {
		return fmt.Errorf("failed to find worktree: %w", err)
	}

	// Print what we're doing
	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Switching to worktree %q at %s...\n", name, worktreePath)
	}

	if util.GlobalContext.IsDryRun() {
		fmt.Println("Dry run: would switch to worktree", name)
		return nil
	}

	// Launch tool
	if switchOpenCode {
		if err := launchOpenCodeTool(worktreePath, name); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch OpenCode: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch OpenCode: %v\n", err)
			}
			return nil
		}
	} else if switchGemini {
		if err := launchGeminiToolForSwitch(worktreePath, name); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Gemini: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Gemini: %v\n", err)
			}
			return nil
		}
	} else if switchDroid {
		if err := launchDroidToolForSwitch(worktreePath, name); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Droid: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Droid: %v\n", err)
			}
			return nil
		}
	} else {
		// Launch Claude
		claudeLauncher := launcher.NewClaudeLauncher(cfg.Claude.CLIPath)

		if !claudeLauncher.IsAvailable() {
			return fmt.Errorf("Claude CLI not found. Please install it or specify the path in config")
		}

		// Prepare launch options
		mode := switchClaudeMode
		if mode == "" {
			mode = cfg.Claude.DefaultMode
		}

		args := append(cfg.Claude.ExtraArgs, switchClaudeArgs...)

		opts := launcher.LaunchOptions{
			WorkDir:      worktreePath,
			Args:         args,
			Mode:         mode,
			Interactive:  launcher.IsTTY(),
			TerminalName: getSwitchTerminalName(name),
		}

		if util.GlobalContext.IsColorEnabled() {
			color.Cyan("Launching Claude Code CLI in %s...\n", worktreePath)
		} else {
			fmt.Printf("Launching Claude Code CLI in %s...\n", worktreePath)
		}

		// Launch Claude
		ctx := context.Background()
		if err := claudeLauncher.Launch(ctx, opts); err != nil {
			return fmt.Errorf("failed to launch Claude: %w", err)
		}
	}

	return nil
}

func getSwitchTerminalName(name string) string {
	if switchTerminalName != "" {
		return switchTerminalName
	}
	return name
}

func launchGeminiToolForSwitch(worktreePath string, terminalName string) error {
	geminiLauncher := launcher.NewGeminiLauncher("")

	if !geminiLauncher.IsAvailable() {
		return fmt.Errorf("Gemini CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: getSwitchTerminalName(terminalName),
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching Gemini in %s...\n", worktreePath)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Launching Gemini...\n")
	} else {
		fmt.Println("Launching Gemini...")
	}

	ctx := context.Background()
	if err := geminiLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func launchDroidToolForSwitch(worktreePath string, terminalName string) error {
	droidLauncher := launcher.NewDroidLauncher("")

	if !droidLauncher.IsAvailable() {
		return fmt.Errorf("Droid CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: getSwitchTerminalName(terminalName),
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching Droid in %s...\n", worktreePath)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Launching Droid...\n")
	} else {
		fmt.Println("Launching Droid...")
	}

	ctx := context.Background()
	if err := droidLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func RunSwitch(cmd *cobra.Command, args []string) error {
	return runSwitch(cmd, args)
}
