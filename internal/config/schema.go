package config

// Config represents the complete configuration for ctl
type Config struct {
	Worktree     WorktreeConfig     `mapstructure:"worktree" yaml:"worktree"`
	Claude       ClaudeConfig       `mapstructure:"claude" yaml:"claude"`
	Global       GlobalConfig       `mapstructure:"global" yaml:"global"`
	CopilotSetup CopilotSetupConfig `mapstructure:"copilot_setup" yaml:"copilot_setup"`
}

// WorktreeConfig contains worktree-related configuration
type WorktreeConfig struct {
	// BaseLocation is the base directory for worktrees (relative to repo parent or absolute)
	BaseLocation string `mapstructure:"base_location" yaml:"base_location"`

	// AutoCleanup determines if worktree should be automatically cleaned up when removed
	AutoCleanup bool `mapstructure:"auto_cleanup" yaml:"auto_cleanup"`

	// DefaultSourceBranch is the default branch to create new worktrees from
	DefaultSourceBranch string `mapstructure:"default_source_branch" yaml:"default_source_branch"`
}

// ClaudeConfig contains Claude CLI integration configuration
type ClaudeConfig struct {
	// DefaultMode is the default mode when launching Claude (agent or chat)
	DefaultMode string `mapstructure:"default_mode" yaml:"default_mode"`

	// AutoLaunch determines if Claude should be automatically launched after creating worktree
	AutoLaunch bool `mapstructure:"auto_launch" yaml:"auto_launch"`

	// ExtraArgs are additional arguments to pass to Claude CLI
	ExtraArgs []string `mapstructure:"extra_args" yaml:"extra_args"`

	// CLIPath is the path to Claude CLI (auto-detected if not specified)
	CLIPath string `mapstructure:"cli_path" yaml:"cli_path"`

	// MinimaxAPIKey is the API key for using Minimax APIs with Claude
	MinimaxAPIKey string `mapstructure:"minimax_api_key" yaml:"minimax_api_key"`

	// SystemPrompt is the system prompt to use when launching Claude
	SystemPrompt string `mapstructure:"system_prompt" yaml:"system_prompt"`

	// SystemPromptMode determines how the system prompt is applied: "replace" or "append"
	SystemPromptMode string `mapstructure:"system_prompt_mode" yaml:"system_prompt_mode"`

	// UseSystemPrompt enables/disables the system prompt feature
	UseSystemPrompt bool `mapstructure:"use_system_prompt" yaml:"use_system_prompt"`

	// MinimaxVerbose enables verbose mode when using Minimax APIs
	MinimaxVerbose bool `mapstructure:"minimax_verbose" yaml:"minimax_verbose"`

	// GLMAPIKey is the API key for using GLM APIs with Claude
	GLMAPIKey string `mapstructure:"glm_api_key" yaml:"glm_api_key"`

	// GLMModel is the model name to use with GLM APIs
	GLMModel string `mapstructure:"glm_model" yaml:"glm_model"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	// Verbosity level (0=quiet, 1=normal, 2=verbose)
	Verbosity int `mapstructure:"verbosity" yaml:"verbosity"`

	// Color determines if output should be colorized
	Color bool `mapstructure:"color" yaml:"color"`

	// Editor for opening files (fallback to $EDITOR)
	Editor string `mapstructure:"editor" yaml:"editor"`

	// DefaultCLI is the default CLI to use when launching (claude, gemini, copilot, droid, opencode)
	DefaultCLI string `mapstructure:"default_cli" yaml:"default_cli"`
}

// CopilotSetupConfig contains configuration for the setup-copilot command
type CopilotSetupConfig struct {
	// InstructionsMdPath is the source path for copilot-instructions.md
	InstructionsMdPath string `mapstructure:"instructions_md_path" yaml:"instructions_md_path"`

	// WorkflowYmlPath is the source path for copilot-setup-steps.yml
	WorkflowYmlPath string `mapstructure:"workflow_yml_path" yaml:"workflow_yml_path"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Worktree: WorktreeConfig{
			BaseLocation:        "../.worktrees",
			AutoCleanup:         true,
			DefaultSourceBranch: "", // Empty means use current branch
		},
		Claude: ClaudeConfig{
			DefaultMode:      "agent",
			AutoLaunch:       true,
			ExtraArgs:        []string{},
			CLIPath:          "", // Auto-detect
			SystemPrompt:     "",
			SystemPromptMode: "replace",
			UseSystemPrompt:  false,
			MinimaxVerbose:   false,
			GLMAPIKey:        "",
			GLMModel:         "glm-4.7",
		},
		Global: GlobalConfig{
			Verbosity:  1,
			Color:      true,
			Editor:     "",
			DefaultCLI: "claude",
		},
		CopilotSetup: CopilotSetupConfig{
			InstructionsMdPath: "",
			WorkflowYmlPath:    "",
		},
	}
}
