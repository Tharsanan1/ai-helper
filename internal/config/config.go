package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	// GlobalConfigDir is the directory for global configuration
	GlobalConfigDir = ".config/aihelper"

	// GlobalConfigFile is the name of the global configuration file
	GlobalConfigFile = "config.yaml"

	// RepoConfigFile is the name of the repo-level configuration file
	RepoConfigFile = ".aihelper.yaml"
)

// Manager handles configuration loading and management
type Manager struct {
	viper     *viper.Viper
	repoRoot  string
	configSet bool
}

// NewManager creates a new configuration manager
func NewManager(repoRoot string) *Manager {
	v := viper.New()

	// Set defaults
	defaults := DefaultConfig()
	v.SetDefault("worktree.base_location", defaults.Worktree.BaseLocation)
	v.SetDefault("worktree.auto_cleanup", defaults.Worktree.AutoCleanup)
	v.SetDefault("worktree.default_source_branch", defaults.Worktree.DefaultSourceBranch)
	v.SetDefault("claude.default_mode", defaults.Claude.DefaultMode)
	v.SetDefault("claude.auto_launch", defaults.Claude.AutoLaunch)
	v.SetDefault("claude.extra_args", defaults.Claude.ExtraArgs)
	v.SetDefault("claude.cli_path", defaults.Claude.CLIPath)
	v.SetDefault("claude.minimax_api_key", defaults.Claude.MinimaxAPIKey)
	v.SetDefault("global.verbosity", defaults.Global.Verbosity)
	v.SetDefault("global.color", defaults.Global.Color)
	v.SetDefault("global.editor", defaults.Global.Editor)

	return &Manager{
		viper:    v,
		repoRoot: repoRoot,
	}
}

// Load loads the configuration from global and repo config files
// Priority: Flags > Repo config > Global config > Defaults
func (m *Manager) Load() error {
	// Load global config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	globalConfigPath := filepath.Join(homeDir, GlobalConfigDir, GlobalConfigFile)
	if _, err := os.Stat(globalConfigPath); err == nil {
		m.viper.SetConfigFile(globalConfigPath)
		if err := m.viper.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read global config: %w", err)
		}
		m.configSet = true
	}

	// Load repo config (if in a repo)
	if m.repoRoot != "" {
		repoConfigPath := filepath.Join(m.repoRoot, RepoConfigFile)
		if _, err := os.Stat(repoConfigPath); err == nil {
			m.viper.SetConfigFile(repoConfigPath)
			if err := m.viper.MergeInConfig(); err != nil {
				return fmt.Errorf("failed to read repo config: %w", err)
			}
			m.configSet = true
		}
	}

	return nil
}

// Get returns the complete configuration
func (m *Manager) Get() (*Config, error) {
	var cfg Config
	if err := m.viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}

// GetString returns a configuration value as a string
func (m *Manager) GetString(key string) string {
	return m.viper.GetString(key)
}

// GetBool returns a configuration value as a bool
func (m *Manager) GetBool(key string) bool {
	return m.viper.GetBool(key)
}

// GetInt returns a configuration value as an int
func (m *Manager) GetInt(key string) int {
	return m.viper.GetInt(key)
}

// GetStringSlice returns a configuration value as a string slice
func (m *Manager) GetStringSlice(key string) []string {
	return m.viper.GetStringSlice(key)
}

// Set sets a configuration value
func (m *Manager) Set(key string, value interface{}) {
	m.viper.Set(key, value)
}

// GlobalConfigPath returns the path to the global configuration file
func GlobalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, GlobalConfigDir, GlobalConfigFile), nil
}

// RepoConfigPath returns the path to the repo configuration file
func RepoConfigPath(repoRoot string) string {
	return filepath.Join(repoRoot, RepoConfigFile)
}
