package config

import (
	"github.com/spf13/cobra"
)

// ConfigCmd represents the config command group
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `Manage aihelper configuration settings.

The config command group provides subcommands for viewing and modifying
configuration values.`,
}

func init() {
	// Add subcommands
	ConfigCmd.AddCommand(getCmd)
	ConfigCmd.AddCommand(setCmd)
	ConfigCmd.AddCommand(listCmd)
}
