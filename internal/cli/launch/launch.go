package launch

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/launcher"
	"github.com/tharsanan1/ai-helper/internal/util"
)

var (
	// Flags for launch command
	launchClaudeMode         string
	launchClaudeArgs         []string
	launchOpenCode           bool
	launchGemini             bool
	launchDroid              bool
	launchCopilot            bool
	launchClaude             bool
	launchMinimax            bool
	launchGLM                bool
	launchKimi               bool
	launchSystemPrompt       string
	launchAppendSystemPrompt bool
	launchTerminalName       string
	launchNewTerminal        bool
	launchSandbox            bool
)

// LaunchCmd represents the launch command
var LaunchCmd = &cobra.Command{
	Use:     "launch",
	Aliases: []string{"l", "run"},
	Short:   "Launch an AI tool in the current directory",
	Long: `Launch an AI tool (Claude, Gemini, etc.) in the current directory
with the required environment variables and configuration.

Examples:
  # Launch Claude (default)
  aihelper launch

  # Launch Claude with Kimi APIs
  aihelper launch --kimi

  # Launch Gemini CLI
  aihelper launch --gemini

  # Launch in a new terminal window
  aihelper launch --new-terminal`,
	RunE: runLaunch,
}

// RegisterLaunchFlags registers flags for the launch command
func RegisterLaunchFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&launchClaudeMode, "claude-mode", "", "Claude mode: chat, agent (default from config)")
	cmd.Flags().StringSliceVar(&launchClaudeArgs, "claude-args", []string{}, "Additional arguments to pass to Claude CLI")
	cmd.Flags().BoolVar(&launchOpenCode, "opencode", false, "Launch OpenCode instead of Claude")
	cmd.Flags().BoolVar(&launchGemini, "gemini", false, "Launch Gemini CLI instead of Claude")
	cmd.Flags().BoolVar(&launchDroid, "droid", false, "Launch Droid CLI instead of Claude")
	cmd.Flags().BoolVar(&launchCopilot, "copilot", false, "Launch Copilot CLI instead of Claude")
	cmd.Flags().BoolVar(&launchClaude, "claude", false, "Launch Claude (explicitly, useful for overriding defaults)")
	cmd.Flags().BoolVar(&launchMinimax, "minimax", false, "Launch Claude with Minimax APIs (requires minimax_api_key in config)")
	cmd.Flags().BoolVar(&launchGLM, "glm", false, "Launch Claude with GLM APIs (requires glm_api_key in config)")
	cmd.Flags().BoolVar(&launchKimi, "kimi", false, "Launch Claude with Kimi APIs (requires kimi_api_key in config)")
	cmd.Flags().StringVar(&launchSystemPrompt, "system-prompt", "", "System prompt to use when launching Claude (overrides config)")
	cmd.Flags().BoolVar(&launchAppendSystemPrompt, "append-system-prompt", false, "Append system prompt instead of replacing")
	cmd.Flags().StringVar(&launchTerminalName, "terminal-name", "", "Terminal window name")
	cmd.Flags().BoolVar(&launchNewTerminal, "new-terminal", false, "Launch agent in a new terminal window")
	cmd.Flags().BoolVar(&launchSandbox, "sandbox", false, "Launch agent in a docker sandbox")

	// Register flag completion
	_ = cmd.RegisterFlagCompletionFunc("claude-mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"chat", "agent"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func init() {
	RegisterLaunchFlags(LaunchCmd)
}

func runLaunch(cmd *cobra.Command, args []string) error {
	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get config manager
	cfgManager, err := util.GlobalContext.GetConfigManager()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	cfg, err := cfgManager.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine which CLI to launch
	useOpenCode := launchOpenCode
	useGemini := launchGemini
	useDroid := launchDroid
	useCopilot := launchCopilot
	useClaude := launchClaude
	useMinimax := launchMinimax
	useGLM := launchGLM
	useKimi := launchKimi

	// If no explicit flag is set, determine based on default CLI
	if !useOpenCode && !useGemini && !useDroid && !useCopilot && !useClaude && !useMinimax && !useGLM && !useKimi {
		defaultCLI := cfg.Global.DefaultCLI
		if defaultCLI == "" {
			defaultCLI = "claude"
		}

		switch defaultCLI {
		case "gemini":
			useGemini = true
		case "copilot":
			useCopilot = true
		case "droid":
			useDroid = true
		case "opencode":
			useOpenCode = true
		default: // claude
			useClaude = true
		}
	}

	if useOpenCode {
		return launchOpenCodeTool(workDir, launchTerminalName, launchSandbox)
	} else if useGemini {
		return launchGeminiTool(workDir, launchTerminalName)
	} else if useDroid {
		return launchDroidTool(workDir, launchTerminalName)
	} else if useCopilot {
		return launchCopilotTool(workDir, launchTerminalName)
	} else if useClaude || useMinimax || useGLM || useKimi {
		return launchClaudeTool(workDir, cfg, useMinimax, useGLM, useKimi)
	}

	return nil
}

func launchOpenCodeTool(workDir string, terminalName string, sandbox bool) error {
	opencodeLauncher := launcher.NewOpenCodeLauncher("")
	if !opencodeLauncher.IsAvailable() {
		return fmt.Errorf("OpenCode CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      workDir,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
		NewTerminal:  launchNewTerminal,
		Sandbox:      sandbox,
	}

	color.Cyan("Launching OpenCode in %s...\n", workDir)
	return opencodeLauncher.Launch(context.Background(), opts)
}

func launchGeminiTool(workDir string, terminalName string) error {
	geminiLauncher := launcher.NewGeminiLauncher("")
	if !geminiLauncher.IsAvailable() {
		return fmt.Errorf("Gemini CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      workDir,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
		NewTerminal:  launchNewTerminal,
		Sandbox:      launchSandbox,
	}

	color.Cyan("Launching Gemini in %s...\n", workDir)
	return geminiLauncher.Launch(context.Background(), opts)
}

func launchDroidTool(workDir string, terminalName string) error {
	droidLauncher := launcher.NewDroidLauncher("")
	if !droidLauncher.IsAvailable() {
		return fmt.Errorf("Droid CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      workDir,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
		NewTerminal:  launchNewTerminal,
		Sandbox:      launchSandbox,
	}

	color.Cyan("Launching Droid in %s...\n", workDir)
	return droidLauncher.Launch(context.Background(), opts)
}

func launchCopilotTool(workDir string, terminalName string) error {
	copilotLauncher := launcher.NewCopilotLauncher("")
	if !copilotLauncher.IsAvailable() {
		return fmt.Errorf("Copilot CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      workDir,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
		NewTerminal:  launchNewTerminal,
		Sandbox:      launchSandbox,
	}

	color.Cyan("Launching Copilot in %s...\n", workDir)
	return copilotLauncher.Launch(context.Background(), opts)
}

func launchClaudeTool(workDir string, cfg *config.Config, useMinimax bool, useGLM bool, useKimi bool) error {
	claudeLauncher := launcher.NewClaudeLauncher(cfg.Claude.CLIPath)
	if !claudeLauncher.IsAvailable() {
		return fmt.Errorf("Claude CLI not found. Please install it or specify the path in config")
	}

	mode := launchClaudeMode
	if mode == "" {
		mode = cfg.Claude.DefaultMode
	}

	args := append(cfg.Claude.ExtraArgs, launchClaudeArgs...)
	env := make(map[string]string)

	if useMinimax {
		if cfg.Claude.MinimaxAPIKey == "" {
			return fmt.Errorf("minimax_api_key not set in config. Use 'aihelper config set claude.minimax_api_key <your-key>'")
		}
		env["ANTHROPIC_BASE_URL"] = "https://api.minimax.io/anthropic"
		env["ANTHROPIC_API_KEY"] = cfg.Claude.MinimaxAPIKey
	} else if useGLM {
		if cfg.Claude.GLMAPIKey == "" {
			return fmt.Errorf("glm_api_key not set in config. Use 'aihelper config set claude.glm_api_key <your-key>'")
		}
		glmModel := cfg.Claude.GLMModel
		if glmModel == "" {
			glmModel = "glm-4.7"
		}
		env["ANTHROPIC_BASE_URL"] = cfg.Claude.GetGLMBaseURL()
		env["ANTHROPIC_AUTH_TOKEN"] = cfg.Claude.GLMAPIKey
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = glmModel
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = glmModel
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = glmModel
	} else if useKimi {
		if cfg.Claude.KimiAPIKey == "" {
			return fmt.Errorf("kimi_api_key not set in config. Use 'aihelper config set claude.kimi_api_key <your-key>'")
		}
		kimiBaseURL := cfg.Claude.KimiBaseURL
		if kimiBaseURL == "" {
			kimiBaseURL = "https://api.kimi.com/coding/"
		}
		env["ANTHROPIC_BASE_URL"] = kimiBaseURL
		env["ANTHROPIC_API_KEY"] = cfg.Claude.KimiAPIKey
	}

	effectiveSystemPrompt := launchSystemPrompt
	if effectiveSystemPrompt == "" && cfg.Claude.UseSystemPrompt {
		effectiveSystemPrompt = cfg.Claude.SystemPrompt
	}

	if (useMinimax || useKimi) && effectiveSystemPrompt != "" {
		if launchAppendSystemPrompt || cfg.Claude.SystemPromptMode == "append" {
			args = append(args, "--append-system-prompt", effectiveSystemPrompt)
		} else {
			args = append(args, "--system-prompt", effectiveSystemPrompt)
		}
	}

	if useMinimax && cfg.Claude.MinimaxVerbose {
		args = append(args, "--verbose")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      workDir,
		Args:         args,
		Mode:         mode,
		Env:          env,
		Interactive:  launcher.IsTTY(),
		TerminalName: launchTerminalName,
		NewTerminal:  launchNewTerminal,
		Sandbox:      launchSandbox,
	}

	if useMinimax {
		color.Cyan("Launching Claude Code CLI with Minimax APIs in %s...\n", workDir)
	} else if useGLM {
		color.Cyan("Launching Claude Code CLI with GLM APIs in %s...\n", workDir)
	} else if useKimi {
		color.Cyan("Launching Claude Code CLI with Kimi APIs in %s...\n", workDir)
	} else {
		color.Cyan("Launching Claude Code CLI in %s...\n", workDir)
	}

	return claudeLauncher.Launch(context.Background(), opts)
}
