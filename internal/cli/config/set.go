package config

import (
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	internalConfig "github.com/tharsanan1/ai-helper/internal/config"
)

var (
	// Flags for set command
	setGlobal bool
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set the value of a configuration key.

By default, this updates the global configuration (~/.config/aihelper/config.yaml).
Use --global flag to explicitly specify global config.

Examples:
  aihelper config set worktree.base_location /custom/path
  aihelper config set claude.auto_launch false
  aihelper config set global.verbosity 2
  aihelper config set global.default_cli gemini`,
	Args: cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return getConfigKeys(), cobra.ShellCompDirectiveNoFileComp
		}
		// For the second argument (value), we could provide suggestions based on the key
		if len(args) == 1 {
			key := args[0]
			switch key {
			case "claude.default_mode":
				return []string{"agent", "chat"}, cobra.ShellCompDirectiveNoFileComp
			case "claude.system_prompt_mode":
				return []string{"replace", "append"}, cobra.ShellCompDirectiveNoFileComp
			case "global.default_cli":
				return []string{"claude", "gemini", "copilot", "droid", "opencode"}, cobra.ShellCompDirectiveNoFileComp
			case "worktree.auto_cleanup", "claude.auto_launch", "claude.use_system_prompt", "claude.minimax_verbose", "global.color":
				return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
			}
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: runSet,
}

func getConfigKeys() []string {
	return []string{
		"worktree.base_location",
		"worktree.auto_cleanup",
		"worktree.default_source_branch",
		"claude.default_mode",
		"claude.auto_launch",
		"claude.cli_path",
		"claude.minimax_api_key",
		"claude.system_prompt",
		"claude.system_prompt_mode",
		"claude.use_system_prompt",
		"claude.minimax_verbose",
		"claude.glm_api_key",
		"claude.glm_model",
		"claude.kimi_api_key",
		"claude.kimi_base_url",
		"global.verbosity",
		"global.color",
		"global.editor",
		"global.default_cli",
		"copilot_setup.instructions_md_path",
		"copilot_setup.workflow_yml_path",
	}
}

func init() {
	setCmd.Flags().BoolVarP(&setGlobal, "global", "g", true, "Set in global config (default)")
}

func runSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Load current config
	cfg, err := internalConfig.LoadGlobalConfig()
	if err != nil {
		// If config doesn't exist, create default
		cfg = internalConfig.DefaultConfig()
	}

	// Parse and set the value based on the key
	if err := setConfigValue(cfg, key, value); err != nil {
		return err
	}

	// Save the config
	if err := internalConfig.SaveGlobalConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Print success message
	// Note: we use color by default since set command modifies config
	color.Green("✓ Set %s = %s\n", key, value)

	return nil
}

// setConfigValue sets a value in the config based on the key path
func setConfigValue(cfg *internalConfig.Config, key, value string) error {
	switch key {
	// Worktree config
	case "worktree.base_location":
		cfg.Worktree.BaseLocation = value
	case "worktree.auto_cleanup":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
		cfg.Worktree.AutoCleanup = val
	case "worktree.default_source_branch":
		cfg.Worktree.DefaultSourceBranch = value

	// Claude config
	case "claude.default_mode":
		if value != "agent" && value != "chat" {
			return fmt.Errorf("invalid mode: %s (must be 'agent' or 'chat')", value)
		}
		cfg.Claude.DefaultMode = value
	case "claude.auto_launch":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
		cfg.Claude.AutoLaunch = val
	case "claude.cli_path":
		cfg.Claude.CLIPath = value
	case "claude.minimax_api_key":
		cfg.Claude.MinimaxAPIKey = value
	case "claude.system_prompt":
		cfg.Claude.SystemPrompt = value
	case "claude.system_prompt_mode":
		if value != "replace" && value != "append" {
			return fmt.Errorf("invalid system prompt mode: %s (must be 'replace' or 'append')", value)
		}
		cfg.Claude.SystemPromptMode = value
	case "claude.use_system_prompt":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
		cfg.Claude.UseSystemPrompt = val
	case "claude.minimax_verbose":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
		cfg.Claude.MinimaxVerbose = val
	case "claude.glm_api_key":
		cfg.Claude.GLMAPIKey = value
	case "claude.glm_model":
		cfg.Claude.GLMModel = value
	case "claude.kimi_api_key":
		cfg.Claude.KimiAPIKey = value
	case "claude.kimi_base_url":
		cfg.Claude.KimiBaseURL = value
	case "claude.glm_base_url":
		cfg.Claude.GLMBaseURL = value

	// Global config
	case "global.verbosity":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer value for %s: %s", key, value)
		}
		cfg.Global.Verbosity = val
	case "global.color":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %s", key, value)
		}
		cfg.Global.Color = val
	case "global.editor":
		cfg.Global.Editor = value
	case "global.default_cli":
		validCLIs := map[string]bool{"claude": true, "gemini": true, "copilot": true, "droid": true, "opencode": true}
		if !validCLIs[value] {
			return fmt.Errorf("invalid CLI: %s (must be one of: claude, gemini, copilot, droid, opencode)", value)
		}
		cfg.Global.DefaultCLI = value

	// Copilot setup config
	case "copilot_setup.instructions_md_path":
		cfg.CopilotSetup.InstructionsMdPath = value
	case "copilot_setup.workflow_yml_path":
		cfg.CopilotSetup.WorkflowYmlPath = value

	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}
