package worktree

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/git"
	"github.com/tharsanan1/ai-helper/internal/launcher"
	"github.com/tharsanan1/ai-helper/internal/util"
	"github.com/tharsanan1/ai-helper/internal/worktree"
)

var (
	// Flags for create command
	createBranch         string
	createLocation       string
	createFrom           string
	createNoClaude       bool
	createClaudeMode     string
	createClaudeArgs     []string
	createExistingBranch bool
	createOpenCode       bool
	createGemini         bool
	createTerminalName   string
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:     "create <name>",
	Aliases: []string{"c", "new"},
	Short:   "Create a new worktree and optionally launch Claude",
	Long: `Create a new git worktree with the specified name and optionally launch Claude Code CLI.

The create command will:
1. Create a new branch (or use an existing one)
2. Create a git worktree at the configured location
3. Launch Claude Code CLI in the worktree directory (unless --no-claude is specified)

Examples:
  # Create a worktree and launch Claude
  ctl worktree create feature-auth

  # Create with custom branch name
  ctl worktree create feature-auth -b auth/login

  # Create from specific source branch
  ctl worktree create hotfix -f main -b hotfix/security

  # Create without launching Claude
  ctl worktree create experiment --no-claude

  # Use an existing branch
  ctl worktree create feature-x -b existing-branch --existing-branch

  # Create worktree and launch OpenCode instead of Claude
  ctl worktree create feature-x --opencode

  # Create worktree and launch Gemini CLI instead of Claude
  ctl worktree create feature-x --gemini`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVarP(&createBranch, "branch", "b", "", "Branch name (default: same as worktree name)")
	createCmd.Flags().StringVarP(&createLocation, "location", "l", "", "Base directory for worktrees (default from config)")
	createCmd.Flags().StringVarP(&createFrom, "from", "f", "", "Source branch to create from (default: current branch)")
	createCmd.Flags().BoolVar(&createNoClaude, "no-claude", false, "Create worktree without launching Claude")
	createCmd.Flags().StringVar(&createClaudeMode, "claude-mode", "", "Claude mode: chat, agent (default from config)")
	createCmd.Flags().StringSliceVar(&createClaudeArgs, "claude-args", []string{}, "Additional arguments to pass to Claude CLI")
	createCmd.Flags().BoolVarP(&createExistingBranch, "existing-branch", "e", false, "Use existing branch instead of creating new")
	createCmd.Flags().BoolVar(&createOpenCode, "opencode", false, "Launch OpenCode instead of Claude")
	createCmd.Flags().BoolVar(&createGemini, "gemini", false, "Launch Gemini CLI instead of Claude")
	createCmd.Flags().StringVar(&createTerminalName, "terminal-name", "", "Terminal window name (default: worktree name)")
}

func runCreate(cmd *cobra.Command, args []string) error {
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

	// Prepare create options
	opts := worktree.CreateOptions{
		Name:           name,
		Branch:         createBranch,
		Location:       createLocation,
		From:           createFrom,
		ExistingBranch: createExistingBranch,
	}

	// Print what we're doing if verbose
	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Creating worktree %q...\n", name)
		if opts.Branch != "" {
			fmt.Printf("  Branch: %s\n", opts.Branch)
		}
		if opts.From != "" {
			fmt.Printf("  From: %s\n", opts.From)
		}
	}

	// Create the worktree
	if util.GlobalContext.IsDryRun() {
		fmt.Println("Dry run: would create worktree", name)
		return nil
	}

	worktreePath, err := wtManager.Create(opts)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Print success message
	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Created worktree at: %s\n", worktreePath)
	} else {
		fmt.Printf("Created worktree at: %s\n", worktreePath)
	}

	// Launch tool if enabled
	launchClaude := !createNoClaude && cfg.Claude.AutoLaunch && !createOpenCode && !createGemini
	launchOpenCode := createOpenCode
	launchGemini := createGemini

	if launchOpenCode {
		if err := launchOpenCodeTool(worktreePath, name); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch OpenCode: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch OpenCode: %v\n", err)
			}
			return nil
		}
	} else if launchGemini {
		if err := launchGeminiTool(worktreePath, getTerminalName(name)); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Gemini: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Gemini: %v\n", err)
			}
			return nil
		}
	} else if launchClaude {
		if err := launchClaudeTool(worktreePath, name, cfg); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Claude: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Claude: %v\n", err)
			}
			return nil
		}
	}

	return nil
}

func launchOpenCodeTool(worktreePath string, terminalName string) error {
	opencodeLauncher := launcher.NewOpenCodeLauncher("")

	if !opencodeLauncher.IsAvailable() {
		return fmt.Errorf("OpenCode CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: getTerminalName(terminalName),
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching OpenCode in %s...\n", worktreePath)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Launching OpenCode...\n")
	} else {
		fmt.Println("Launching OpenCode...")
	}

	ctx := context.Background()
	if err := opencodeLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func launchGeminiTool(worktreePath string, terminalName string) error {
	geminiLauncher := launcher.NewGeminiLauncher("")

	if !geminiLauncher.IsAvailable() {
		return fmt.Errorf("Gemini CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
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

func launchClaudeTool(worktreePath string, terminalName string, cfg *config.Config) error {
	// Create Claude launcher
	claudeLauncher := launcher.NewClaudeLauncher(cfg.Claude.CLIPath)

	// Check if Claude is available
	if !claudeLauncher.IsAvailable() {
		return fmt.Errorf("Claude CLI not found. Please install it or specify the path in config")
	}

	// Prepare launch options
	mode := createClaudeMode
	if mode == "" {
		mode = cfg.Claude.DefaultMode
	}

	args := append(cfg.Claude.ExtraArgs, createClaudeArgs...)

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Args:         args,
		Mode:         mode,
		Interactive:  launcher.IsTTY(),
		TerminalName: getTerminalName(terminalName),
	}

	// Print what we're doing
	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching Claude in %s...\n", worktreePath)
		if mode != "" {
			fmt.Printf("  Mode: %s\n", mode)
		}
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Launching Claude Code CLI...\n")
	} else {
		fmt.Println("Launching Claude Code CLI...")
	}

	// Launch Claude
	ctx := context.Background()
	if err := claudeLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func getTerminalName(name string) string {
	if createTerminalName != "" {
		return createTerminalName
	}
	return name
}

func RunCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, args)
}
