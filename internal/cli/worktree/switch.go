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
	switchCopilot      bool
	switchClaude       bool
	switchMinimax      bool
	switchGLM          bool
	switchSystemPrompt string
	switchAppendSystemPrompt bool
	switchTerminalName string
	switchSandbox      bool
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
  aihelper worktree switch feature-auth

  # Switch with custom Claude mode
  aihelper worktree switch feature-auth --claude-mode chat

  # Switch and launch OpenCode instead of Claude
  aihelper worktree switch feature-auth --opencode

  # Switch and launch Gemini CLI instead of Claude
  aihelper worktree switch feature-auth --gemini

  # Switch and launch Droid CLI instead of Claude
  aihelper worktree switch feature-auth --droid

  # Switch and launch Copilot CLI instead of Claude
  aihelper worktree switch feature-auth --copilot

  # Switch and launch Claude (explicitly)
  aihelper worktree switch feature-auth --claude

  # Switch and launch Claude with Minimax APIs
  aihelper worktree switch feature-auth --minimax

  # Switch with custom system prompt
  aihelper worktree switch feature-auth --system-prompt "You are a senior engineer"

  # Switch and append system prompt
  aihelper worktree switch feature-auth --system-prompt "Focus on testing" --append-system-prompt

  # Switch with custom terminal name
  aihelper worktree switch feature-auth --terminal-name "Feature Auth"

  # Switch in Docker sandbox
  aihelper worktree switch feature-auth --sandbox

  # Switch with custom Claude arguments
  	aihelper worktree switch feature-auth --claude-args "--dangerously-skip-permissions"`,
  	Args:              cobra.ExactArgs(1),
  	ValidArgsFunction: getWorktreeNames,
  	RunE:              runSwitch,
  }
  
// RegisterSwitchFlags registers flags for the switch command
func RegisterSwitchFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&switchClaudeMode, "claude-mode", "", "Claude mode: chat, agent (default from config)")
	cmd.Flags().StringSliceVar(&switchClaudeArgs, "claude-args", []string{}, "Additional arguments to pass to Claude CLI")
	cmd.Flags().BoolVar(&switchOpenCode, "opencode", false, "Launch OpenCode instead of Claude")
	cmd.Flags().BoolVar(&switchGemini, "gemini", false, "Launch Gemini CLI instead of Claude")
	cmd.Flags().BoolVar(&switchDroid, "droid", false, "Launch Droid CLI instead of Claude")
	cmd.Flags().BoolVar(&switchCopilot, "copilot", false, "Launch Copilot CLI instead of Claude")
	cmd.Flags().BoolVar(&switchClaude, "claude", false, "Launch Claude (explicitly, useful for overriding defaults)")
	cmd.Flags().BoolVar(&switchMinimax, "minimax", false, "Launch Claude with Minimax APIs (requires minimax_api_key in config)")
	cmd.Flags().BoolVar(&switchGLM, "glm", false, "Launch Claude with GLM APIs (requires glm_api_key in config)")
	cmd.Flags().StringVar(&switchSystemPrompt, "system-prompt", "", "System prompt to use when launching Claude (overrides config)")
	cmd.Flags().BoolVar(&switchAppendSystemPrompt, "append-system-prompt", false, "Append system prompt instead of replacing")
	cmd.Flags().StringVar(&switchTerminalName, "terminal-name", "", "Terminal window name (default: worktree name)")
	cmd.Flags().BoolVar(&switchSandbox, "sandbox", false, "Launch agent in a docker sandbox")

	// Register flag completion
	_ = cmd.RegisterFlagCompletionFunc("claude-mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"chat", "agent"}, cobra.ShellCompDirectiveNoFileComp
	})
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

	// Launch tool based on flags or default CLI
	launchOpenCode := switchOpenCode
	launchGemini := switchGemini
	launchDroid := switchDroid
	launchCopilot := switchCopilot
	launchClaude := switchClaude
	launchMinimax := switchMinimax
	launchGLM := switchGLM

	// If no explicit flag is set, determine based on default CLI
	if !launchOpenCode && !launchGemini && !launchDroid && !launchCopilot && !launchClaude && !launchMinimax && !launchGLM {
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
			launchClaude = true
		}
	}

	if launchOpenCode {
		if err := launchOpenCodeTool(worktreePath, name, switchSandbox); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch OpenCode: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch OpenCode: %v\n", err)
			}
			return nil
		}
	} else if launchGemini {
		if err := launchGeminiToolForSwitch(worktreePath, name); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Gemini: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Gemini: %v\n", err)
			}
			return nil
		}
	} else if launchDroid {
		if err := launchDroidToolForSwitch(worktreePath, name); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Droid: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Droid: %v\n", err)
			}
			return nil
		}
	} else if launchCopilot {
		if err := launchCopilotToolForSwitch(worktreePath, name); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Copilot: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Copilot: %v\n", err)
			}
			return nil
		}
	} else if launchClaude || launchMinimax || launchGLM {
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

		// Prepare environment variables
		env := make(map[string]string)
		if launchMinimax {
			// Set Minimax environment variables
			if cfg.Claude.MinimaxAPIKey == "" {
				return fmt.Errorf("minimax_api_key not set in config. Use 'aihelper config set claude.minimax_api_key <your-key>'")
			}
			env["ANTHROPIC_BASE_URL"] = "https://api.minimax.io/anthropic"
			env["ANTHROPIC_API_KEY"] = cfg.Claude.MinimaxAPIKey
		} else if launchGLM {
			// Set GLM environment variables
			if cfg.Claude.GLMAPIKey == "" {
				return fmt.Errorf("glm_api_key not set in config. Use 'aihelper config set claude.glm_api_key <your-key>'")
			}
			glmModel := cfg.Claude.GLMModel
			if glmModel == "" {
				glmModel = "glm-4.7"
			}
			env["ANTHROPIC_AUTH_TOKEN"] = cfg.Claude.GLMAPIKey
			env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = glmModel
			env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = glmModel
			env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = glmModel
		}

		// Handle system prompt
		effectiveSystemPrompt := switchSystemPrompt
		if effectiveSystemPrompt == "" && cfg.Claude.UseSystemPrompt {
			effectiveSystemPrompt = cfg.Claude.SystemPrompt
		}

		if launchMinimax && effectiveSystemPrompt != "" {
			if switchAppendSystemPrompt || cfg.Claude.SystemPromptMode == "append" {
				args = append(args, "--append-system-prompt", effectiveSystemPrompt)
			} else {
				args = append(args, "--system-prompt", effectiveSystemPrompt)
			}
		}

		// Handle minimax verbose mode
		if launchMinimax && cfg.Claude.MinimaxVerbose {
			args = append(args, "--verbose")
		}

		opts := launcher.LaunchOptions{
			WorkDir:      worktreePath,
			Args:         args,
			Mode:         mode,
			Env:          env,
			Interactive:  launcher.IsTTY(),
			TerminalName: getSwitchTerminalName(name),
			Sandbox:      switchSandbox,
		}

		if util.GlobalContext.IsColorEnabled() {
			if launchMinimax {
				color.Cyan("Launching Claude Code CLI with Minimax APIs in %s...\n", worktreePath)
			} else if launchGLM {
				color.Cyan("Launching Claude Code CLI with GLM APIs in %s...\n", worktreePath)
			} else {
				color.Cyan("Launching Claude Code CLI in %s...\n", worktreePath)
			}
		} else {
			if launchMinimax {
				fmt.Printf("Launching Claude Code CLI with Minimax APIs in %s...\n", worktreePath)
			} else if launchGLM {
				fmt.Printf("Launching Claude Code CLI with GLM APIs in %s...\n", worktreePath)
			} else {
				fmt.Printf("Launching Claude Code CLI in %s...\n", worktreePath)
			}
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
		Sandbox:      switchSandbox,
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
		Sandbox:      switchSandbox,
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

func launchCopilotToolForSwitch(worktreePath string, terminalName string) error {
	copilotLauncher := launcher.NewCopilotLauncher("")

	if !copilotLauncher.IsAvailable() {
		return fmt.Errorf("Copilot CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: getSwitchTerminalName(terminalName),
		Sandbox:      switchSandbox,
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

func RunSwitch(cmd *cobra.Command, args []string) error {
	return runSwitch(cmd, args)
}
