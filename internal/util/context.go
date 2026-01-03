package util

import (
	"fmt"

	"github.com/tharsanan1/ai-helper/internal/config"
)

// CLIContext holds the global CLI context
type CLIContext struct {
	ConfigManager *config.Manager
	Verbose       bool
	DryRun        bool
	NoColor       bool
}

// GetConfigManager returns the config manager, initializing it if needed
func (c *CLIContext) GetConfigManager() (*config.Manager, error) {
	if c.ConfigManager != nil {
		return c.ConfigManager, nil
	}

	// Create config manager
	cfgManager := config.NewManager("")

	// Load config from files
	if err := cfgManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Override with flags if provided
	if c.Verbose {
		cfgManager.Set("global.verbosity", 2)
	}
	if c.NoColor {
		cfgManager.Set("global.color", false)
	}

	c.ConfigManager = cfgManager
	return cfgManager, nil
}

// IsVerbose returns whether verbose mode is enabled
func (c *CLIContext) IsVerbose() bool {
	return c.Verbose
}

// IsDryRun returns whether dry-run mode is enabled
func (c *CLIContext) IsDryRun() bool {
	return c.DryRun
}

// IsColorEnabled returns whether colored output is enabled
func (c *CLIContext) IsColorEnabled() bool {
	if c.NoColor {
		return false
	}

	if c.ConfigManager != nil {
		return c.ConfigManager.GetBool("global.color")
	}

	return true
}

// GlobalContext is the global CLI context instance
var GlobalContext = &CLIContext{}
