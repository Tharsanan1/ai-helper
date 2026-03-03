package wso2patch

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/config"
	"github.com/tharsanan1/ai-helper/internal/util"
	wtvalidator "github.com/tharsanan1/ai-helper/internal/worktree"
)

var (
	createVersion string
)

type repoPlan struct {
	repoName   string
	repoPath   string
	branch     string
	targetPath string
	remoteRef  string
}

type createdWorktree struct {
	repoPath string
	path     string
}

var createCmd = &cobra.Command{
	Use:     "create <worktree-name>",
	Aliases: []string{"c"},
	Short:   "Create WSO2 patch worktrees across configured repositories",
	Long: `Create worktrees for all configured WSO2 patch repositories using the branch
mapped to the provided product version.

All required upstream branches are validated before any worktree directories are created.`,
	Example: `  aihelper wso2-patch create patch-4201 --version 4.2.0
  aihelper wso2-patch c patch-4201 --version 4.2.0`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVar(&createVersion, "version", "", "Product version to resolve branch mapping (required)")
	_ = createCmd.MarkFlagRequired("version")
}

func runCreate(cmd *cobra.Command, args []string) error {
	worktreeName := strings.TrimSpace(args[0])
	if err := wtvalidator.ValidateWorktreeName(worktreeName); err != nil {
		return err
	}

	version := strings.TrimSpace(createVersion)
	if version == "" {
		return fmt.Errorf("--version is required")
	}

	cfgManager, err := util.GlobalContext.GetConfigManager()
	if err != nil {
		return fmt.Errorf("failed to get config manager: %w", err)
	}

	cfg, err := cfgManager.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	baseLocation, err := expandPath(cfg.WSO2Patch.BaseLocation)
	if err != nil {
		return fmt.Errorf("failed to resolve wso2-patch.base_location: %w", err)
	}
	if baseLocation == "" {
		return fmt.Errorf("wso2-patch.base_location must be configured")
	}

	if len(cfg.WSO2Patch.Repos) == 0 {
		return fmt.Errorf("wso2-patch.repos is empty; configure at least one repository")
	}

	targetRoot := filepath.Join(baseLocation, worktreeName)
	if _, err := os.Stat(targetRoot); err == nil {
		return fmt.Errorf("target root already exists: %s", targetRoot)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect target root %s: %w", targetRoot, err)
	}

	plans := make([]repoPlan, 0, len(cfg.WSO2Patch.Repos))

	for _, repoCfg := range cfg.WSO2Patch.Repos {
		repoPath, err := expandPath(repoCfg.Path)
		if err != nil {
			return fmt.Errorf("failed to resolve repo path %q: %w", repoCfg.Path, err)
		}
		if repoPath == "" {
			return fmt.Errorf("wso2-patch repo path cannot be empty")
		}

		repoName := strings.TrimSpace(repoCfg.Name)
		if repoName == "" {
			repoName = filepath.Base(repoPath)
		}

		branch, err := resolveBranch(repoCfg, version)
		if err != nil {
			return fmt.Errorf("repo %q: %w", repoName, err)
		}
		branch = strings.TrimSpace(branch)
		if err := validateBranchName(repoPath, branch); err != nil {
			return fmt.Errorf("repo %q: invalid branch %q: %w", repoName, branch, err)
		}

		targetPath := filepath.Join(targetRoot, repoName)
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("target already exists: %s", targetPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect target %s: %w", targetPath, err)
		}

		printProgress("Validating %s (branch: upstream/%s)", repoName, branch)
		if err := validateRepo(repoPath); err != nil {
			return fmt.Errorf("repo %q: %w", repoName, err)
		}

		if err := verifyUpstreamBranch(repoPath, branch); err != nil {
			return fmt.Errorf("repo %q: %w", repoName, err)
		}
		printDone("Validated %s", repoName)

		plans = append(plans, repoPlan{
			repoName:   repoName,
			repoPath:   repoPath,
			branch:     branch,
			targetPath: targetPath,
			remoteRef:  "upstream/" + branch,
		})
	}

	if util.GlobalContext.IsDryRun() {
		fmt.Printf("Dry run: would create WSO2 patch worktrees at %s\n", targetRoot)
		for _, plan := range plans {
			fmt.Printf("  - %s (%s) from %s\n", plan.targetPath, plan.repoName, plan.remoteRef)
		}
		return nil
	}

	for _, plan := range plans {
		printProgress("Fetching %s from %s", plan.repoName, plan.remoteRef)
		if err := fetchUpstreamBranch(plan.repoPath, plan.branch); err != nil {
			return fmt.Errorf("repo %q: failed to fetch upstream branch %q: %w", plan.repoName, plan.branch, err)
		}
		if _, err := runGit(plan.repoPath, "rev-parse", "--verify", plan.remoteRef); err != nil {
			return fmt.Errorf("repo %q: missing remote tracking ref %q after fetch: %w", plan.repoName, plan.remoteRef, err)
		}
		printDone("Fetched %s", plan.repoName)
	}

	printProgress("Creating target root %s", targetRoot)
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		return fmt.Errorf("failed to create target root %s: %w", targetRoot, err)
	}
	printDone("Created target root")

	created := make([]createdWorktree, 0, len(plans))
	for _, plan := range plans {
		printProgress("Creating worktree for %s", plan.repoName)
		if err := os.MkdirAll(filepath.Dir(plan.targetPath), 0755); err != nil {
			rollbackCreatedWorktrees(created)
			_ = removeDirIfEmpty(targetRoot)
			return fmt.Errorf("failed to create directory for %s: %w", plan.targetPath, err)
		}

		if _, err := runGit(plan.repoPath, "worktree", "add", "--detach", plan.targetPath, plan.remoteRef); err != nil {
			rollbackCreatedWorktrees(created)
			_ = removeDirIfEmpty(targetRoot)
			return fmt.Errorf("repo %q: failed to create worktree at %s from %s: %w", plan.repoName, plan.targetPath, plan.remoteRef, err)
		}

		created = append(created, createdWorktree{repoPath: plan.repoPath, path: plan.targetPath})
		printDone("Created %s worktree", plan.repoName)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Created WSO2 patch worktrees at: %s\n", targetRoot)
	} else {
		fmt.Printf("Created WSO2 patch worktrees at: %s\n", targetRoot)
	}

	for _, plan := range plans {
		fmt.Printf("  - %s (%s from %s)\n", plan.targetPath, plan.repoName, plan.remoteRef)
	}

	printProgress("Switching to %s", targetRoot)
	return execShellInDir(targetRoot)
}

func resolveBranch(repoCfg config.WSO2PatchRepoConfig, version string) (string, error) {
	if branch, ok := repoCfg.VersionBranchMap[version]; ok && strings.TrimSpace(branch) != "" {
		return strings.TrimSpace(branch), nil
	}

	template := strings.TrimSpace(repoCfg.BranchTemplate)
	if template != "" {
		return strings.ReplaceAll(template, "<version>", version), nil
	}

	return "", fmt.Errorf("no branch mapping found for version %q", version)
}

func validateRepo(repoPath string) error {
	info, err := os.Stat(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("repository path does not exist: %s", repoPath)
		}
		return fmt.Errorf("failed to inspect repository path %s: %w", repoPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repository path is not a directory: %s", repoPath)
	}

	if _, err := runGit(repoPath, "rev-parse", "--is-inside-work-tree"); err != nil {
		return fmt.Errorf("not a git repository: %s (%w)", repoPath, err)
	}

	if _, err := runGit(repoPath, "remote", "get-url", "upstream"); err != nil {
		return fmt.Errorf("upstream remote not found in %s (%w)", repoPath, err)
	}

	return nil
}

func verifyUpstreamBranch(repoPath, branch string) error {
	_, err := runGit(repoPath, "ls-remote", "--heads", "--exit-code", "upstream", branch)
	if err != nil {
		return fmt.Errorf("upstream branch %q not found", branch)
	}

	return nil
}

func fetchUpstreamBranch(repoPath, branch string) error {
	_, err := runGit(repoPath, "fetch", "upstream", branch)
	return err
}

func validateBranchName(repoPath, branch string) error {
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("branch name is empty")
	}
	_, err := runGit(repoPath, "check-ref-format", "--branch", branch)
	return err
}

func rollbackCreatedWorktrees(created []createdWorktree) {
	for i := len(created) - 1; i >= 0; i-- {
		entry := created[i]
		_, _ = runGit(entry.repoPath, "worktree", "remove", "--force", entry.path)
		_ = os.RemoveAll(entry.path)
	}
}

func runGit(repoPath string, args ...string) (string, error) {
	gitArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", gitArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), errMsg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func expandPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		if trimmed == "~" {
			return home, nil
		}
		trimmed = filepath.Join(home, strings.TrimPrefix(trimmed, "~/"))
	}

	return filepath.Clean(trimmed), nil
}

func removeDirIfEmpty(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) == 0 {
		return os.Remove(path)
	}
	return nil
}

func execShellInDir(dir string) error {
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		for _, candidate := range []string{"/bin/zsh", "/bin/bash", "/bin/sh"} {
			if _, err := os.Stat(candidate); err == nil {
				shell = candidate
				break
			}
		}
	}
	if shell == "" {
		return fmt.Errorf("could not determine shell")
	}

	args := []string{filepath.Base(shell)}
	return syscall.Exec(shell, args, os.Environ())
}

func printProgress(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if util.GlobalContext.IsColorEnabled() {
		color.Cyan("→ %s\n", msg)
		return
	}
	fmt.Printf("-> %s\n", msg)
}

func printDone(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ %s\n", msg)
		return
	}
	fmt.Printf("✓ %s\n", msg)
}
