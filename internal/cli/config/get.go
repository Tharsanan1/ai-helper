package config

import (
	"fmt"

	"github.com/spf13/cobra"
	internalConfig "github.com/tharsanan1/ai-helper/internal/config"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get the value of a configuration key.

Examples:
  aihelper config get worktree.base_location
  aihelper config get claude.auto_launch
  aihelper config get global.verbosity
  aihelper config get global.default_cli`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Create config manager
	cfgManager := internalConfig.NewManager("")
	if err := cfgManager.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get the value
	value := cfgManager.GetString(key)
	if value == "" {
		// Try as bool
		if cfgManager.GetBool(key) {
			fmt.Println("true")
			return nil
		}

		// Try as int
		if intVal := cfgManager.GetInt(key); intVal != 0 {
			fmt.Println(intVal)
			return nil
		}

		// Value is empty or doesn't exist
		fmt.Printf("(not set)\n")
		return nil
	}

	fmt.Println(value)
	return nil
}
