package config

// Config represents the complete configuration for ctl
type Config struct {
	Worktree     WorktreeConfig     `mapstructure:"worktree" yaml:"worktree"`
	Claude       ClaudeConfig       `mapstructure:"claude" yaml:"claude"`
	Global       GlobalConfig       `mapstructure:"global" yaml:"global"`
	CopilotSetup CopilotSetupConfig `mapstructure:"copilot_setup" yaml:"copilot_setup"`
	WSO2Patch    WSO2PatchConfig    `mapstructure:"wso2-patch" yaml:"wso2-patch"`
	PeerTest     PeerTestConfig     `mapstructure:"peertest" yaml:"peertest"`
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

	// GLMBaseURL is the base URL for GLM APIs
	GLMBaseURL string `mapstructure:"glm_base_url" yaml:"glm_base_url"`

	// KimiAPIKey is the API key for using Kimi APIs with Claude
	KimiAPIKey string `mapstructure:"kimi_api_key" yaml:"kimi_api_key"`

	// KimiBaseURL is the base URL for Kimi APIs (default: https://api.kimi.com/coding/)
	KimiBaseURL string `mapstructure:"kimi_base_url" yaml:"kimi_base_url"`
}

// GetGLMBaseURL returns the GLM base URL with default fallback
func (c *ClaudeConfig) GetGLMBaseURL() string {
	if c.GLMBaseURL != "" {
		return c.GLMBaseURL
	}
	return "https://api.z.ai/api/anthropic"
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

// WSO2PatchConfig contains settings for coordinated WSO2 patch worktree creation
type WSO2PatchConfig struct {
	// BaseLocation is where wso2 patch worktrees are created
	BaseLocation string `mapstructure:"base_location" yaml:"base_location"`

	// Repos contains the repo paths and version-to-branch mapping details
	Repos []WSO2PatchRepoConfig `mapstructure:"repos" yaml:"repos"`
}

// WSO2PatchRepoConfig defines branch resolution and source repo details
type WSO2PatchRepoConfig struct {
	// Name is the output folder name for the repo worktree (defaults to repo folder name if empty)
	Name string `mapstructure:"name" yaml:"name"`

	// Path is the local git repository path
	Path string `mapstructure:"path" yaml:"path"`

	// VersionBranchMap maps product version to upstream branch name
	VersionBranchMap map[string]string `mapstructure:"version_branch_map" yaml:"version_branch_map"`

	// BranchTemplate resolves branch names for versions not in VersionBranchMap
	// Use "<version>" placeholder, e.g. "support-<version>.x-full"
	BranchTemplate string `mapstructure:"branch_template" yaml:"branch_template"`
}

// PeerTestConfig contains versioned product pack and workflow settings.
type PeerTestConfig struct {
	Products map[string]PeerTestProductConfig `mapstructure:"products" yaml:"products"`
}

// PeerTestProductConfig defines how to create and update a peertest workspace.
type PeerTestProductConfig struct {
	// PackPath is the zip file for the product pack.
	PackPath string `mapstructure:"pack_path" yaml:"pack_path"`

	// WorkspaceRoot is where peertest folders are created.
	WorkspaceRoot string `mapstructure:"workspace_root" yaml:"workspace_root"`

	// WorkingDir is the directory inside the extracted product where steps run.
	// If empty, "bin" is used.
	WorkingDir string `mapstructure:"working_dir" yaml:"working_dir"`

	// Steps is the ordered list of shell commands to run inside WorkingDir.
	Steps []string `mapstructure:"steps" yaml:"steps"`

	// RunWorkingDir is the directory inside the extracted product where run steps execute.
	// If empty, "bin" is used.
	RunWorkingDir string `mapstructure:"run_working_dir" yaml:"run_working_dir"`

	// RunSteps is the ordered list of shell commands to start the prepared peer test pack.
	RunSteps []string `mapstructure:"run_steps" yaml:"run_steps"`

	// SmokeTest contains defaults for the version-specific browser smoke test.
	SmokeTest PeerTestSmokeTestConfig `mapstructure:"smoketest" yaml:"smoketest"`
}

// PeerTestSmokeTestConfig defines default smoke test inputs for a product version.
type PeerTestSmokeTestConfig struct {
	BaseURL              string `mapstructure:"base_url" yaml:"base_url"`
	AdminUser            string `mapstructure:"admin_user" yaml:"admin_user"`
	AdminPassword        string `mapstructure:"admin_password" yaml:"admin_password"`
	TenantDomain         string `mapstructure:"tenant_domain" yaml:"tenant_domain"`
	TenantAdminUser      string `mapstructure:"tenant_admin_user" yaml:"tenant_admin_user"`
	TenantAdminPassword  string `mapstructure:"tenant_admin_password" yaml:"tenant_admin_password"`
	TenantAdminEmail     string `mapstructure:"tenant_admin_email" yaml:"tenant_admin_email"`
	TenantAdminFirstName string `mapstructure:"tenant_admin_first_name" yaml:"tenant_admin_first_name"`
	TenantAdminLastName  string `mapstructure:"tenant_admin_last_name" yaml:"tenant_admin_last_name"`
	TenantUser           string `mapstructure:"tenant_user" yaml:"tenant_user"`
	TenantUserPassword   string `mapstructure:"tenant_user_password" yaml:"tenant_user_password"`
	APIEndpoint          string `mapstructure:"api_endpoint" yaml:"api_endpoint"`
	APINamePrefix        string `mapstructure:"api_name_prefix" yaml:"api_name_prefix"`
	APIVersion           string `mapstructure:"api_version" yaml:"api_version"`
	ScreenshotDir        string `mapstructure:"screenshot_dir" yaml:"screenshot_dir"`
	ScreenshotDelayMs    int    `mapstructure:"screenshot_delay_ms" yaml:"screenshot_delay_ms"`
	SlowMo               int    `mapstructure:"slow_mo" yaml:"slow_mo"`
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
			GLMBaseURL:       "https://api.z.ai/api/anthropic",
			KimiAPIKey:       "",
			KimiBaseURL:      "https://api.kimi.com/coding/",
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
		WSO2Patch: WSO2PatchConfig{
			BaseLocation: "~/Documents/worktree/wso2-patch",
			Repos: []WSO2PatchRepoConfig{
				{
					Name: "carbon-apimgt",
					Path: "/Users/tharsanan/Documents/github/forked/carbon-apimgt",
					VersionBranchMap: map[string]string{
						"1.9.0":  "support-1.2.0",
						"1.9.1":  "support-1.2.5",
						"1.10.0": "support-5.0.3",
						"2.0.0":  "support-6.0.4",
						"2.1.0":  "support-6.1.66",
						"2.2.0":  "support-6.2.201",
						"2.5.0":  "support-6.3.95",
						"2.6.0":  "support-6.4.50.x-full",
						"3.0.0":  "support-6.5.349.x-full",
						"3.1.0":  "support-6.6.163.x-full",
						"3.2.0":  "support-6.7.206.x-full",
						"3.2.1":  "support-6.7.210.x-full",
						"4.0.0":  "support-9.0.174.x-full",
						"4.1.0":  "support-9.20.74.x-full",
						"4.2.0":  "support-9.28.116.x-full",
						"4.3.0":  "support-9.29.120.x-full",
						"4.4.0":  "support-9.30.67.x-full",
						"4.5.0":  "support-9.31.86.x-full",
						"4.6.0":  "support-9.32.147.x-full",
					},
				},
				{
					Name:           "product-apim",
					Path:           "/Users/tharsanan/Documents/github/forked/product-apim",
					BranchTemplate: "support-<version>.x-full",
				},
			},
		},
		PeerTest: PeerTestConfig{
			Products: map[string]PeerTestProductConfig{
				"4.4.0": {
					PackPath:      "~/Documents/wso2/apim/4.4.0/wso2am-4.4.0.13.zip",
					WorkspaceRoot: "~/Documents/wso2/apim/4.4.0/peertests",
					WorkingDir:    "bin",
					Steps: []string{
						"./wso2update_darwin -u {{username}} -p {{password}}",
						"./wso2update_darwin",
						"export WSO2_UPDATES_UPDATE_LEVEL_STATE=TESTING",
						"./wso2update_darwin",
						`grep "Applied " ../updates/logs/wso2update-{{today}}.log`,
					},
					RunWorkingDir: "bin",
					RunSteps: []string{
						"sh api-manager.sh",
					},
					SmokeTest: PeerTestSmokeTestConfig{
						BaseURL:              "https://localhost:9443",
						AdminUser:            "admin",
						AdminPassword:        "admin",
						TenantDomain:         "peertest.com",
						TenantAdminUser:      "peer",
						TenantAdminPassword:  "peer1",
						TenantAdminEmail:     "peer@peertest.com",
						TenantAdminFirstName: "peer",
						TenantAdminLastName:  "admin",
						TenantUser:           "peertestuser",
						TenantUserPassword:   "peer1",
						APIEndpoint:          "https://httpbin.org/anything",
						APINamePrefix:        "PeerTestAPI",
						APIVersion:           "1.0.0",
						ScreenshotDir:        "",
						ScreenshotDelayMs:    1000,
						SlowMo:               250,
					},
				},
			},
		},
	}
}
