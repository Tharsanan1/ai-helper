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
	RunE: runSet,
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

	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}
