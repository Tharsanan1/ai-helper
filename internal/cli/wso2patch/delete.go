package wso2patch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tharsanan1/ai-helper/internal/util"
	wtvalidator "github.com/tharsanan1/ai-helper/internal/worktree"
)

type deletePlan struct {
	repoName string
	repoPath string
	path     string
}

var deleteCmd = &cobra.Command{
	Use:     "delete <worktree-name>",
	Aliases: []string{"d", "rm", "remove"},
	Short:   "Delete WSO2 patch worktrees for a patch name",
	Long: `Delete all configured repository worktrees for the given WSO2 patch worktree name.

This removes worktrees from each configured repository using git worktree remove --force.`,
	Example: `  aihelper wso2-patch delete my-fix-421
  aihelper wso2-patch d my-fix-421`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	worktreeName := strings.TrimSpace(args[0])
	if err := wtvalidator.ValidateWorktreeName(worktreeName); err != nil {
		return err
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
		return fmt.Errorf("wso2-patch.repos is empty; nothing to delete")
	}

	targetRoot := filepath.Join(baseLocation, worktreeName)
	info, err := os.Stat(targetRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target root not found: %s", targetRoot)
		}
		return fmt.Errorf("failed to inspect target root %s: %w", targetRoot, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target root is not a directory: %s", targetRoot)
	}

	plans := make([]deletePlan, 0, len(cfg.WSO2Patch.Repos))
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

		worktreePath := filepath.Join(targetRoot, repoName)
		if _, err := os.Stat(worktreePath); err == nil {
			plans = append(plans, deletePlan{repoName: repoName, repoPath: repoPath, path: worktreePath})
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect %s: %w", worktreePath, err)
		}
	}

	if len(plans) == 0 {
		return fmt.Errorf("no configured repo worktrees found under %s", targetRoot)
	}

	if util.GlobalContext.IsDryRun() {
		fmt.Printf("Dry run: would delete WSO2 patch worktrees from %s\n", targetRoot)
		for _, plan := range plans {
			fmt.Printf("  - %s (%s)\n", plan.path, plan.repoName)
		}
		return nil
	}

	for _, plan := range plans {
		printProgress("Removing worktree for %s", plan.repoName)
		if _, err := runGit(plan.repoPath, "worktree", "remove", "--force", plan.path); err != nil {
			return fmt.Errorf("repo %q: failed to remove worktree %s: %w", plan.repoName, plan.path, err)
		}
		_ = os.RemoveAll(plan.path)
		printDone("Removed %s worktree", plan.repoName)
	}

	if err := removeDirIfEmpty(targetRoot); err != nil {
		return fmt.Errorf("failed to clean target root %s: %w", targetRoot, err)
	}

	if util.GlobalContext.IsColorEnabled() {
		color.Green("✓ Deleted WSO2 patch worktrees for: %s\n", worktreeName)
	} else {
		fmt.Printf("Deleted WSO2 patch worktrees for: %s\n", worktreeName)
	}

	return nil
}
