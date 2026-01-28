package worktree

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/gh"
	"github.com/tharsanan1/ai-helper/internal/git"
	"github.com/tharsanan1/ai-helper/internal/launcher"
	"github.com/tharsanan1/ai-helper/internal/util"
	"github.com/tharsanan1/ai-helper/internal/worktree"
)

var (
	// Flags for create command
	createBranch             string
	createLocation           string
	createFrom               string
	createIssue              int
	createPR                 int
	createNoClaude           bool
	createClaudeMode         string
	createClaudeArgs         []string
	createExistingBranch     bool
	createOpenCode           bool
	createGemini             bool
	createDroid              bool
	createCopilot            bool
	createClaude             bool
	createMinimax            bool
	createGLM                bool
	createKimi               bool
	createSystemPrompt       string
	createAppendSystemPrompt bool
	createTerminalName       string
	createNewTerminal        bool
	createSandbox            bool
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:     "create [name]",
	Aliases: []string{"c", "new"},
	Short:   "Create a new worktree and optionally launch Claude",
	Long: `Create a new git worktree with the specified name and optionally launch Claude Code CLI.

If --issue is provided, the [name] argument is optional and will be automatically generated from the issue.

Examples:
  # Create a worktree from a GitHub issue (name auto-generated)
  aihelper worktree create --issue 123

  # Create a worktree with a specific name and launch Claude
  aihelper worktree create feature-auth`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCreate,
}

// RegisterCreateFlags registers flags for the create command
func RegisterCreateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&createBranch, "branch", "b", "", "Branch name (default: same as worktree name)")
	cmd.Flags().StringVarP(&createLocation, "location", "l", "", "Base directory for worktrees (default from config)")
	cmd.Flags().StringVarP(&createFrom, "from", "f", "", "Source branch to create from (default: current branch)")
	cmd.Flags().IntVarP(&createIssue, "issue", "i", 0, "Create worktree from GitHub issue number")
	cmd.Flags().IntVarP(&createPR, "pr", "p", 0, "Create worktree from GitHub pull request number")
	cmd.Flags().BoolVar(&createNoClaude, "no-claude", false, "Create worktree without launching Claude")
	cmd.Flags().StringVar(&createClaudeMode, "claude-mode", "", "Claude mode: chat, agent (default from config)")
	cmd.Flags().StringSliceVar(&createClaudeArgs, "claude-args", []string{}, "Additional arguments to pass to Claude CLI")
	cmd.Flags().BoolVarP(&createExistingBranch, "existing-branch", "e", false, "Use existing branch instead of creating new")
	cmd.Flags().BoolVar(&createOpenCode, "opencode", false, "Launch OpenCode instead of Claude")
	cmd.Flags().BoolVar(&createGemini, "gemini", false, "Launch Gemini CLI instead of Claude")
	cmd.Flags().BoolVar(&createDroid, "droid", false, "Launch Droid CLI instead of Claude")
	cmd.Flags().BoolVar(&createCopilot, "copilot", false, "Launch Copilot CLI instead of Claude")
	cmd.Flags().BoolVar(&createClaude, "claude", false, "Launch Claude (explicitly, useful for overriding defaults)")
	cmd.Flags().BoolVar(&createMinimax, "minimax", false, "Launch Claude with Minimax APIs (requires minimax_api_key in config)")
	cmd.Flags().BoolVar(&createGLM, "glm", false, "Launch Claude with GLM APIs (requires glm_api_key in config)")
	cmd.Flags().BoolVar(&createKimi, "kimi", false, "Launch Claude with Kimi APIs (requires kimi_api_key in config)")
	cmd.Flags().StringVar(&createSystemPrompt, "system-prompt", "", "System prompt to use when launching Claude (overrides config)")
	cmd.Flags().BoolVar(&createAppendSystemPrompt, "append-system-prompt", false, "Append system prompt instead of replacing")
	cmd.Flags().StringVar(&createTerminalName, "terminal-name", "", "Terminal window name (default: worktree name)")
	cmd.Flags().BoolVar(&createNewTerminal, "new-terminal", false, "Launch agent in a new terminal window")
	cmd.Flags().BoolVar(&createSandbox, "sandbox", false, "Launch agent in a docker sandbox")

	// Register flag completion
	_ = cmd.RegisterFlagCompletionFunc("claude-mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"chat", "agent"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func init() {
	RegisterCreateFlags(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" && createIssue == 0 && createPR == 0 {
		return fmt.Errorf("accepts 1 arg(s), received 0 (or provide --issue or --pr to auto-generate name)")
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

	// Create git client
	gitClient, err := git.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize git client: %w", err)
	}

	// Handle PR flag
	if createPR > 0 {
		ghClient := gh.NewClient()
		if !ghClient.IsAvailable() {
			return fmt.Errorf("gh cli not found. Please install it to use --pr flag")
		}

		var pr *gh.PR
		var prErr error
		var remoteName string

		// 1. Try upstream
		upstreamURL, err := gitClient.GetRemoteURL("upstream")
		if err == nil && upstreamURL != "" {
			s := util.NewSpinner(fmt.Sprintf("Fetching PR #%d from upstream (%s)...", createPR, upstreamURL))
			s.Start()
			pr, prErr = ghClient.GetPR(upstreamURL, createPR)
			s.Stop()
			if prErr == nil {
				remoteName = "upstream"
			} else if util.GlobalContext.IsVerbose() {
				fmt.Printf("PR not found in upstream: %v\n", prErr)
			}
		}

		// 2. Try origin (if upstream failed or didn't exist)
		if pr == nil {
			originURL, err := gitClient.GetRemoteURL("origin")
			if err != nil {
				return fmt.Errorf("failed to get origin remote: %w", err)
			}

			s := util.NewSpinner(fmt.Sprintf("Fetching PR #%d from origin (%s)...", createPR, originURL))
			s.Start()
			pr, err = ghClient.GetPR(originURL, createPR)
			s.Stop()
			if err != nil {
				return fmt.Errorf("PR #%d not found in origin or upstream: %w", createPR, err)
			}
			remoteName = "origin"
		}

		// PR found!
		if util.GlobalContext.IsColorEnabled() {
			color.Green("✓ Found PR: %s\n", pr.Title)
		} else {
			fmt.Printf("Found PR: %s\n", pr.Title)
		}

		// Determine branch name
		// Format: pr-<number>-<sanitized-title>
		if createBranch == "" {
			reg := regexp.MustCompile("[^a-zA-Z0-9]+")
			cleanTitle := reg.ReplaceAllString(strings.ToLower(pr.Title), "-")
			cleanTitle = strings.Trim(cleanTitle, "-")
			// Limit length
			if len(cleanTitle) > 50 {
				cleanTitle = cleanTitle[:50]
				cleanTitle = strings.TrimRight(cleanTitle, "-")
			}
			createBranch = fmt.Sprintf("pr-%d-%s", pr.Number, cleanTitle)
		}

		if name == "" {
			name = createBranch
		}

		// Check if worktree already exists for this branch or path
		wtManager := worktree.NewManager(gitClient, cfg)
		wts, err := wtManager.List()
		if err == nil {
			for _, wt := range wts {
				if wt.Branch == createBranch || strings.HasSuffix(wt.Path, name) {
					if util.GlobalContext.IsColorEnabled() {
						color.Green("✓ Worktree already exists for PR #%d at %s\n", createPR, wt.Path)
					} else {
						fmt.Printf("Worktree already exists for PR #%d at %s\n", createPR, wt.Path)
					}
					// Launch tool directly
					return launchTool(wt.Path, name, cfg, createSystemPrompt, createAppendSystemPrompt)
				}
			}
		}

		// Check if branch exists locally
		exists, err := gitClient.BranchExists(createBranch)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}

		if !exists {
			// Fetch PR to local branch
			// git fetch <remote> pull/<id>/head:<local-branch>
			refSpec := fmt.Sprintf("pull/%d/head:%s", pr.Number, createBranch)
			s := util.NewSpinner(fmt.Sprintf("Fetching PR code to branch %s...", createBranch))
			s.Start()
			err = gitClient.FetchRef(remoteName, refSpec)
			s.Stop()
			if err != nil {
				return fmt.Errorf("failed to fetch PR code: %w", err)
			}
			if util.GlobalContext.IsColorEnabled() {
				color.Green("✓ Fetched PR code to branch: %s\n", createBranch)
			} else {
				fmt.Printf("Fetched PR code to branch: %s\n", createBranch)
			}
		}

		// Ensure we use the existing branch (since we just created/verified it)
		createExistingBranch = true

		// Set system prompt
		prContext := fmt.Sprintf("You are reviewing GitHub PR #%d: %s\n\nDescription:\n%s\n\nBranch: %s", pr.Number, pr.Title, pr.Body, pr.HeadRefName)
		if createSystemPrompt != "" {
			createSystemPrompt = prContext + "\n\nAdditional Instructions:\n" + createSystemPrompt
		} else {
			createSystemPrompt = prContext
		}
	}

	// Handle issue flag
	if createIssue > 0 {
		ghClient := gh.NewClient()
		if !ghClient.IsAvailable() {
			return fmt.Errorf("gh cli not found. Please install it to use --issue flag")
		}

		var issue *gh.Issue
		var issueErr error

		// 1. Try upstream
		upstreamURL, err := gitClient.GetRemoteURL("upstream")
		if err == nil && upstreamURL != "" {
			s := util.NewSpinner(fmt.Sprintf("Fetching issue #%d from upstream (%s)...", createIssue, upstreamURL))
			s.Start()
			issue, issueErr = ghClient.GetIssue(upstreamURL, createIssue)
			s.Stop()
			if issueErr != nil {
				// Upstream exists but issue fetch failed (likely not found)
				if util.GlobalContext.IsVerbose() {
					fmt.Printf("Issue not found in upstream: %v\n", issueErr)
				}

				// Prompt user to check origin
				fmt.Printf("Issue #%d not found in upstream. Fetch from origin? [y/N] ", createIssue)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))

				if response != "y" && response != "yes" {
					return fmt.Errorf("issue not found in upstream and fallback cancelled")
				}
			}
		}

		// 2. Try origin (if upstream failed or didn't exist)
		if issue == nil {
			originURL, err := gitClient.GetRemoteURL("origin")
			if err != nil {
				return fmt.Errorf("failed to get origin remote: %w", err)
			}

			s := util.NewSpinner(fmt.Sprintf("Fetching issue #%d from origin (%s)...", createIssue, originURL))
			s.Start()
			issue, err = ghClient.GetIssue(originURL, createIssue)
			s.Stop()
			if err != nil {
				return fmt.Errorf("issue #%d not found in origin or upstream: %w", createIssue, err)
			}
		}

		// Issue found! Configure worktree options
		if util.GlobalContext.IsColorEnabled() {
			color.Green("✓ Found issue: %s\n", issue.Title)
		} else {
			fmt.Printf("Found issue: %s\n", issue.Title)
		}

		// Generate branch name if not provided
		if createBranch == "" {
			// Try Gemini first
			if _, err := exec.LookPath("gemini"); err == nil {
				s := util.NewSpinner("Generating branch name from issue using Gemini...")
				s.Start()
				prompt := fmt.Sprintf("Generate a concise, standard git branch name for GitHub Issue #%d: '%s'. The format MUST be 'issue/%d/<short-kebab-case-description>'. Return ONLY the branch name, no other text.", issue.Number, issue.Title, issue.Number)
				cmd := exec.Command("gemini", prompt)
				var out bytes.Buffer
				cmd.Stdout = &out
				runErr := cmd.Run()
				s.Stop()
				if runErr == nil {
					generatedName := strings.TrimSpace(out.String())
					// Basic validation: ensure it doesn't contain spaces or newlines and looks like a branch name
					if generatedName != "" && !strings.ContainsAny(generatedName, " \n\t") {
						createBranch = generatedName
						fmt.Printf("Generated branch name: %s\n", createBranch)
					}
				} else if util.GlobalContext.IsVerbose() {
					fmt.Printf("Gemini branch name generation failed: %v\n", runErr)
				}
			}

			// Fallback if Gemini failed or returned empty
			if createBranch == "" {
				// Sanitize title: lowercase, replace non-alphanumeric with hyphens
				reg := regexp.MustCompile("[^a-zA-Z0-9]+")
				cleanTitle := reg.ReplaceAllString(strings.ToLower(issue.Title), "-")
				cleanTitle = strings.Trim(cleanTitle, "-")
				createBranch = fmt.Sprintf("issue-%d-%s", issue.Number, cleanTitle)
				fmt.Printf("Auto-generated branch name: %s\n", createBranch)
			}
		}

		// If name still empty, use a sanitized version of the branch name (removing slashes)
		if name == "" {
			name = strings.ReplaceAll(createBranch, "/", "-")
		}

		// Set system prompt
		issueContext := fmt.Sprintf("You are working on GitHub Issue #%d: %s\n\nDescription:\n%s", issue.Number, issue.Title, issue.Body)
		if createSystemPrompt != "" {
			// Prepend issue context to user-provided system prompt
			createSystemPrompt = issueContext + "\n\nAdditional Instructions:\n" + createSystemPrompt
		} else {
			createSystemPrompt = issueContext
		}
	}

	// Create worktree manager
	wtManager := worktree.NewManager(gitClient, cfg)

	// Prepare create options
	opts := worktree.CreateOptions{
		Name:           name,
		Branch:         createBranch,
		Location:       createLocation,
		From:           createFrom,
		ExistingBranch: createExistingBranch,
	}

	// Print what we're doing if verbose
	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Creating worktree %q...\n", name)
		if opts.Branch != "" {
			fmt.Printf("  Branch: %s\n", opts.Branch)
		}
		if opts.From != "" {
			fmt.Printf("  From: %s\n", opts.From)
		}
	}

	// Create the worktree
	if util.GlobalContext.IsDryRun() {
		fmt.Println("Dry run: would create worktree", name)
		return nil
	}

	worktreePath, err := wtManager.Create(opts)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Print success message
	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Created worktree at: %s\n", worktreePath)
	} else {
		fmt.Printf("Created worktree at: %s\n", worktreePath)
	}

	// Determine which CLI to launch
	return launchTool(worktreePath, name, cfg, createSystemPrompt, createAppendSystemPrompt)
}

func launchTool(worktreePath string, name string, cfg *config.Config, systemPrompt string, appendSystemPrompt bool) error {
	// Priority: explicit flags > default CLI > claude (if auto-launch enabled)
	launchOpenCode := createOpenCode
	launchGemini := createGemini
	launchDroid := createDroid
	launchCopilot := createCopilot
	launchClaude := createClaude
	launchMinimax := createMinimax
	launchGLM := createGLM
	launchKimi := createKimi

	// Only launch if an explicit flag is set (no auto-launch behavior)
	if !createNoClaude && !launchOpenCode && !launchGemini && !launchDroid && !launchCopilot && !launchClaude && !launchMinimax && !launchGLM && !launchKimi {
		// No explicit tool specified - exec into the worktree directory
		return execShellInDir(worktreePath)
	}

	if launchOpenCode {
		if err := launchOpenCodeTool(worktreePath, name, createSandbox); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch OpenCode: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch OpenCode: %v\n", err)
			}
			return nil
		}
	} else if launchGemini {
		if err := launchGeminiTool(worktreePath, getTerminalName(name)); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Gemini: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Gemini: %v\n", err)
			}
			return nil
		}
	} else if launchDroid {
		if err := launchDroidTool(worktreePath, getTerminalName(name)); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Droid: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Droid: %v\n", err)
			}
			return nil
		}
	} else if launchCopilot {
		if err := launchCopilotTool(worktreePath, getTerminalName(name)); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Copilot: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Copilot: %v\n", err)
			}
			return nil
		}
	} else if launchClaude || launchMinimax || launchGLM || launchKimi {
		if err := launchClaudeTool(worktreePath, name, cfg, launchMinimax, launchGLM, launchKimi, systemPrompt, appendSystemPrompt); err != nil {
			if util.GlobalContext.IsColorEnabled() {
				color.Yellow("Warning: failed to launch Claude: %v\n", err)
			} else {
				fmt.Printf("Warning: failed to launch Claude: %v\n", err)
			}
			return nil
		}
	}

	return nil
}

func launchOpenCodeTool(worktreePath string, terminalName string, sandbox bool) error {
	opencodeLauncher := launcher.NewOpenCodeLauncher("")

	if !opencodeLauncher.IsAvailable() {
		return fmt.Errorf("OpenCode CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: getTerminalName(terminalName),
		NewTerminal:  createNewTerminal,
		Sandbox:      sandbox,
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching OpenCode in %s...\n", worktreePath)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Launching OpenCode...\n")
	} else {
		fmt.Println("Launching OpenCode...")
	}

	ctx := context.Background()
	if err := opencodeLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func launchGeminiTool(worktreePath string, terminalName string) error {
	geminiLauncher := launcher.NewGeminiLauncher("")

	if !geminiLauncher.IsAvailable() {
		return fmt.Errorf("Gemini CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
		NewTerminal:  createNewTerminal,
		Sandbox:      createSandbox,
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching Gemini in %s...\n", worktreePath)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Launching Gemini...\n")
	} else {
		fmt.Println("Launching Gemini...")
	}

	ctx := context.Background()
	if err := geminiLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func launchDroidTool(worktreePath string, terminalName string) error {
	droidLauncher := launcher.NewDroidLauncher("")

	if !droidLauncher.IsAvailable() {
		return fmt.Errorf("Droid CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
		NewTerminal:  createNewTerminal,
		Sandbox:      createSandbox,
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching Droid in %s...\n", worktreePath)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Launching Droid...\n")
	} else {
		fmt.Println("Launching Droid...")
	}

	ctx := context.Background()
	if err := droidLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func launchCopilotTool(worktreePath string, terminalName string) error {
	copilotLauncher := launcher.NewCopilotLauncher("")

	if !copilotLauncher.IsAvailable() {
		return fmt.Errorf("Copilot CLI not found. Please install it")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Interactive:  launcher.IsTTY(),
		TerminalName: terminalName,
		NewTerminal:  createNewTerminal,
		Sandbox:      createSandbox,
	}

	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching Copilot in %s...\n", worktreePath)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("Launching Copilot...\n")
	} else {
		fmt.Println("Launching Copilot...")
	}

	ctx := context.Background()
	if err := copilotLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func launchClaudeTool(worktreePath string, terminalName string, cfg *config.Config, useMinimax bool, useGLM bool, useKimi bool, systemPrompt string, appendSystemPrompt bool) error {
	// Create Claude launcher
	claudeLauncher := launcher.NewClaudeLauncher(cfg.Claude.CLIPath)

	// Check if Claude is available
	if !claudeLauncher.IsAvailable() {
		return fmt.Errorf("Claude CLI not found. Please install it or specify the path in config")
	}

	// Prepare launch options
	mode := createClaudeMode
	if mode == "" {
		mode = cfg.Claude.DefaultMode
	}

	args := append(cfg.Claude.ExtraArgs, createClaudeArgs...)

	// Prepare environment variables
	env := make(map[string]string)
	if useMinimax {
		// Set Minimax environment variables
		if cfg.Claude.MinimaxAPIKey == "" {
			return fmt.Errorf("minimax_api_key not set in config. Use 'aihelper config set claude.minimax_api_key <your-key>'")
		}
		env["ANTHROPIC_BASE_URL"] = "https://api.minimax.io/anthropic"
		env["ANTHROPIC_API_KEY"] = cfg.Claude.MinimaxAPIKey
	} else if useGLM {
		// Set GLM environment variables
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
		// Set Kimi environment variables
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

	// Handle system prompt
	effectiveSystemPrompt := systemPrompt
	if effectiveSystemPrompt == "" && cfg.Claude.UseSystemPrompt {
		effectiveSystemPrompt = cfg.Claude.SystemPrompt
	}

	if (useMinimax || useKimi) && effectiveSystemPrompt != "" {
		if appendSystemPrompt || cfg.Claude.SystemPromptMode == "append" {
			args = append(args, "--append-system-prompt", effectiveSystemPrompt)
		} else {
			args = append(args, "--system-prompt", effectiveSystemPrompt)
		}
	}

	// Handle minimax verbose mode
	if useMinimax && cfg.Claude.MinimaxVerbose {
		args = append(args, "--verbose")
	}

	opts := launcher.LaunchOptions{
		WorkDir:      worktreePath,
		Args:         args,
		Mode:         mode,
		Env:          env,
		Interactive:  launcher.IsTTY(),
		TerminalName: getTerminalName(terminalName),
		NewTerminal:  createNewTerminal,
		Sandbox:      createSandbox,
	}

	// Print what we're doing
	if util.GlobalContext.IsVerbose() {
		fmt.Printf("Launching Claude in %s...\n", worktreePath)
		if mode != "" {
			fmt.Printf("  Mode: %s\n", mode)
		}
	}

	if util.GlobalContext.IsColorEnabled() {
		if useMinimax {
			color.Cyan("Launching Claude Code CLI with Minimax APIs...\n")
		} else if useGLM {
			color.Cyan("Launching Claude Code CLI with GLM APIs...\n")
		} else {
			color.Cyan("Launching Claude Code CLI...\n")
		}
	} else {
		if useMinimax {
			fmt.Println("Launching Claude Code CLI with Minimax APIs...")
		} else if useGLM {
			fmt.Println("Launching Claude Code CLI with GLM APIs...")
		} else {
			fmt.Println("Launching Claude Code CLI...")
		}
	}

	// Launch Claude
	ctx := context.Background()
	if err := claudeLauncher.Launch(ctx, opts); err != nil {
		return err
	}

	return nil
}

func getTerminalName(name string) string {
	if createTerminalName != "" {
		return createTerminalName
	}
	return name
}

func execShellInDir(dir string) error {
	// Change to the worktree directory
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	// Get the user's shell from SHELL environment variable
	shell := os.Getenv("SHELL")
	if shell == "" {
		// Fallback to common shells
		for _, s := range []string{"/bin/zsh", "/bin/bash", "/bin/sh"} {
			if _, err := os.Stat(s); err == nil {
				shell = s
				break
			}
		}
	}
	if shell == "" {
		return fmt.Errorf("could not determine shell")
	}

	// Exec the shell (replaces current process)
	// argv[0] is typically the shell name, argv[1] starts actual args
	args := []string{filepath.Base(shell)}

	return syscall.Exec(shell, args, os.Environ())
}

func RunCreate(cmd *cobra.Command, args []string) error {
	return runCreate(cmd, args)
}
