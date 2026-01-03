package config

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	internalConfig "github.com/tharsanan/ai-helper/internal/config"
	"gopkg.in/yaml.v3"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "show"},
	Short:   "List all configuration values",
	Long: `List all configuration values in YAML format.

This shows the complete configuration with all settings.`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	// Create config manager
	cfgManager := internalConfig.NewManager("")
	if err := cfgManager.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg, err := cfgManager.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Print with syntax highlighting if color is enabled
	useColor := cfg.Global.Color
	if useColor {
		// Simple syntax highlighting
		lines := string(data)
		fmt.Println(color.CyanString("Current configuration:"))
		fmt.Println(lines)
	} else {
		fmt.Println("Current configuration:")
		fmt.Println(string(data))
	}

	// Print config file locations
	fmt.Println()
	if useColor {
		color.Yellow("Config file locations:")
	} else {
		fmt.Println("Config file locations:")
	}

	globalPath, _ := internalConfig.GlobalConfigPath()
	fmt.Printf("  Global: %s\n", globalPath)

	return nil
}
