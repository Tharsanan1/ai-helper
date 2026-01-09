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
	createDroid          bool
	createCopilot        bool
	createClaude         bool
	createMinimax        bool
	createSystemPrompt   string
	createAppendSystemPrompt bool
	createTerminalName   string
	createNewTerminal    bool
	createSandbox        bool
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
  aihelper worktree create feature-auth

  # Create and launch agent in a new terminal window
  aihelper worktree create feature-auth --copilot --new-terminal

  # Create with custom branch name
  aihelper worktree create feature-auth -b auth/login

  # Create from specific source branch
  aihelper worktree create hotfix -f main -b hotfix/security

  # Create without launching Claude
  aihelper worktree create experiment --no-claude

  # Use an existing branch
  aihelper worktree create feature-x -b existing-branch --existing-branch

  # Create worktree and launch OpenCode instead of Claude
  aihelper worktree create feature-x --opencode

  # Create worktree and launch Gemini CLI instead of Claude
  aihelper worktree create feature-x --gemini

  # Create worktree and launch Droid CLI instead of Claude
  aihelper worktree create feature-x --droid

  # Create worktree and launch Copilot CLI instead of Claude
  aihelper worktree create feature-x --copilot

  # Create worktree and launch Claude (explicitly)
  aihelper worktree create feature-x --claude

  # Create worktree and launch Claude with Minimax APIs
  aihelper worktree create feature-x --minimax

  # Create worktree with custom system prompt
  aihelper worktree create feature-x --system-prompt "You are a senior engineer"

  # Create worktree and append system prompt
  aihelper worktree create feature-x --system-prompt "Focus on testing" --append-system-prompt

  # Create worktree in a new terminal with custom name
  aihelper worktree create feature-x --new-terminal --terminal-name "Feature X"

  # Create worktree in Docker sandbox
  aihelper worktree create feature-x --sandbox

  # Create with custom Claude mode and arguments
  aihelper worktree create feature-x --claude-mode chat --claude-args "--dangerously-skip-permissions"

  # Create at custom location
  aihelper worktree create feature-x -l ~/custom/worktrees`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

// RegisterCreateFlags registers flags for the create command
func RegisterCreateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&createBranch, "branch", "b", "", "Branch name (default: same as worktree name)")
	cmd.Flags().StringVarP(&createLocation, "location", "l", "", "Base directory for worktrees (default from config)")
	cmd.Flags().StringVarP(&createFrom, "from", "f", "", "Source branch to create from (default: current branch)")
	cmd.Flags().BoolVar(&createNoClaude, "no-claude", false, "Create worktree without launching Claude")
	cmd.Flags().StringVar(&createClaudeMode, "claude-mode", "", "Claude mode: chat, agent (default from config)")
	cmd.Flags().StringSliceVar(&createClaudeArgs, "claude-args", []string{}, "Additional arguments to pass to Claude CLI")
	cmd.Flags().BoolVarP(&createExistingBranch, "existing-branch", "e", false, "Use existing branch instead of creating new")
	cmd.Flags().BoolVar(&createOpenCode, "opencode", false, "Launch OpenCode instead of Claude")
	cmd.Flags().BoolVar(&createGemini, "gemini", false, "Launch Gemini CLI instead of Claude")
	cmd.Flags().BoolVar(&createDroid, "droid", false, "Launch Droid CLI instead of Claude")
	cmd.Flags().BoolVar(&createCopilot, "copilot", false, "Launch Copilot CLI instead of Claude")
	cmd.Flags().BoolVar(&createClaude, "claude", false, "Launch Claude (explicitly, useful for overriding defaults)")
	cmd.Flags().BoolVar(&createMinimax, "minimax", false, "Launch Claude with Minimax APIs (requires minimax_api_key in config)")
	cmd.Flags().StringVar(&createSystemPrompt, "system-prompt", "", "System prompt to use when launching Claude (overrides config)")
	cmd.Flags().BoolVar(&createAppendSystemPrompt, "append-system-prompt", false, "Append system prompt instead of replacing")
	cmd.Flags().StringVar(&createTerminalName, "terminal-name", "", "Terminal window name (default: worktree name)")
	cmd.Flags().BoolVar(&createNewTerminal, "new-terminal", false, "Launch agent in a new terminal window")
	cmd.Flags().BoolVar(&createSandbox, "sandbox", false, "Launch agent in a docker sandbox")

	// Register flag completion
	_ = cmd.RegisterFlagCompletionFunc("claude-mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"chat", "agent"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func init() {
	RegisterCreateFlags(createCmd)
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

	// Determine which CLI to launch
	// Priority: explicit flags > default CLI > claude (if auto-launch enabled)
	launchOpenCode := createOpenCode
	launchGemini := createGemini
	launchDroid := createDroid
	launchCopilot := createCopilot
	launchClaude := createClaude
	launchMinimax := createMinimax

	// If no explicit flag is set and not --no-claude, determine based on default CLI
	if !createNoClaude && !launchOpenCode && !launchGemini && !launchDroid && !launchCopilot && !launchClaude && !launchMinimax {
		defaultCLI := cfg.Global.DefaultCLI
		if defaultCLI == "" {
			defaultCLI = "claude"
		}

		switch defaultCLI {
		case "gemini":
			launchGemini = true
		case "copilot":
			launchCopilot = true
		case "droid":
			launchDroid = true
		case "opencode":
			launchOpenCode = true
		default: // claude
			launchClaude = cfg.Claude.AutoLaunch
		}
	}

	if launchOpenCode {
		if err := launchOpenCodeTool(worktreePath, name, createSandbox); err != nil {
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
	} else if launchDroid {
		if err := launchDroidTool(worktreePath, getTerminalName(name)); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Droid: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Droid: %v\n", err)
			}
			return nil
		}
	} else if launchCopilot {
		if err := launchCopilotTool(worktreePath, getTerminalName(name)); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Copilot: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Copilot: %v\n", err)
			}
			return nil
		}
	} else if launchClaude || launchMinimax {
		if err := launchClaudeTool(worktreePath, name, cfg, launchMinimax, createSystemPrompt, createAppendSystemPrompt); err != nil {
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

func launchOpenCodeTool(worktreePath string, terminalName string, sandbox bool) error {
	opencodeLauncher := launcher.NewOpenCodeLauncher("")

	if !opencodeLauncher.IsAvailable() {
		return fmt.Errorf("OpenCode CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: getTerminalName(terminalName),
		NewTerminal:  createNewTerminal,
		Sandbox:      sandbox,
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
		NewTerminal:  createNewTerminal,
		Sandbox:      createSandbox,
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

func launchDroidTool(worktreePath string, terminalName string) error {
	droidLauncher := launcher.NewDroidLauncher("")

	if !droidLauncher.IsAvailable() {
		return fmt.Errorf("Droid CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
		NewTerminal:  createNewTerminal,
		Sandbox:      createSandbox,
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

func launchCopilotTool(worktreePath string, terminalName string) error {
	copilotLauncher := launcher.NewCopilotLauncher("")

	if !copilotLauncher.IsAvailable() {
		return fmt.Errorf("Copilot CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
		NewTerminal:  createNewTerminal,
		Sandbox:      createSandbox,
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching Copilot in %s...\n", worktreePath)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Launching Copilot...\n")
	} else {
		fmt.Println("Launching Copilot...")
	}

	ctx := context.Background()
	if err := copilotLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func launchClaudeTool(worktreePath string, terminalName string, cfg *config.Config, useMinimax bool, systemPrompt string, appendSystemPrompt bool) error {
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

	// Prepare environment variables
	env := make(map[string]string)
	if useMinimax {
		// Set Minimax environment variables
		if cfg.Claude.MinimaxAPIKey == "" {
			return fmt.Errorf("minimax_api_key not set in config. Use 'aihelper config set claude.minimax_api_key <your-key>'")
		}
		env["ANTHROPIC_BASE_URL"] = "https://api.minimax.io/anthropic"
		env["ANTHROPIC_API_KEY"] = cfg.Claude.MinimaxAPIKey
	}

	// Handle system prompt
	effectiveSystemPrompt := systemPrompt
	if effectiveSystemPrompt == "" && cfg.Claude.UseSystemPrompt {
		effectiveSystemPrompt = cfg.Claude.SystemPrompt
	}

	if useMinimax && effectiveSystemPrompt != "" {
		if appendSystemPrompt || cfg.Claude.SystemPromptMode == "append" {
			args = append(args, "--append-system-prompt", effectiveSystemPrompt)
		} else {
			args = append(args, "--system-prompt", effectiveSystemPrompt)
		}
	}

	// Handle minimax verbose mode
	if useMinimax && cfg.Claude.MinimaxVerbose {
		args = append(args, "--verbose")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Args:         args,
		Mode:         mode,
		Env:          env,
		Interactive:  launcher.IsTTY(),
		TerminalName: getTerminalName(terminalName),
		NewTerminal:  createNewTerminal,
		Sandbox:      createSandbox,
	}

	// Print what we're doing
	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching Claude in %s...\n", worktreePath)
		if mode != "" {
			fmt.Printf("  Mode: %s\n", mode)
		}
	}

	if util.GlobalContext.IsColorEnabled() {
		if useMinimax {
			color.Cyan("Launching Claude Code CLI with Minimax APIs...\n")
		} else {
			color.Cyan("Launching Claude Code CLI...\n")
		}
	} else {
		if useMinimax {
			fmt.Println("Launching Claude Code CLI with Minimax APIs...")
		} else {
			fmt.Println("Launching Claude Code CLI...")
		}
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
